// This file implements the BDOS function-calls.
//
// These are documented online:
//
// * https://www.seasip.info/Cpm/bdos.html

package cpm

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/fcb"
)

// blkSize is the size of block-based I/O operations
const blkSize = 128

// maxRC is the maximum read count
const maxRC = 128

// data2String is a simple helper that is designed to dump a small
// array of data to a string.
//
// It is used to write FCB values and I/O records to logs.
func data2String(data []uint8) string {

	// Ensure we're only dumping a single record
	if len(data) > 128 {
		panic("too big")
	}

	// copy into a record just to deal with short
	// reads or writes.
	t := make([]uint8, 128)
	for n, e := range data {
		t[n] = e
	}

	// now output
	res := ""
	hex := ""
	asc := ""
	for n, e := range t {

		// Sixteen bytes at a time.
		if n%16 == 0 && len(hex) > 0 {
			res += hex
			res += " | "
			res += asc
			res += "\n"

			hex = ""
			asc = ""
		}

		hex += fmt.Sprintf("%02X ", e)
		if e > 32 && e < 128 {
			asc += string(e)
		} else {
			asc += " "
		}
	}
	return res
}

// BdosSysCallExit implements the Exit syscall
func BdosSysCallExit(cpm *CPM) error {
	cpm.CPU.HALT = true
	return ErrBoot
}

// BdosSysCallReadChar reads a single character from the console.
func BdosSysCallReadChar(cpm *CPM) error {

	// Block for input
	c, err := cpm.input.BlockForCharacterWithEcho()
	if err != nil {
		return fmt.Errorf("error in call to BlockForCharacter: %s", err)
	}

	// Return values:
	cpm.CPU.States.HL.SetU16(uint16(c))
	return nil
}

// BdosSysCallWriteChar writes the single character in the E register to STDOUT.
func BdosSysCallWriteChar(cpm *CPM) error {

	cpm.output.PutCharacter(cpm.CPU.States.DE.Lo)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallAuxRead reads a single character from the auxiliary input.
//
// Note: Echo is not enabled in this function.
func BdosSysCallAuxRead(cpm *CPM) error {

	// Block for input
	c, err := cpm.input.BlockForCharacterNoEcho()
	if err != nil {
		return fmt.Errorf("error in call to BlockForCharacterNoEcho: %s", err)
	}

	// Return values:
	cpm.CPU.States.HL.SetU16(uint16(c))
	return nil
}

// BdosSysCallAuxWrite writes the single character in the C register
// auxiliary / punch output.
func BdosSysCallAuxWrite(cpm *CPM) error {

	// The character we're going to write
	c := cpm.CPU.States.BC.Lo
	cpm.output.PutCharacter(c)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallPrinterWrite should send a single character to the printer,
// we fake that by writing to a file instead.
func BdosSysCallPrinterWrite(cpm *CPM) error {

	// write the character to our printer-file
	err := cpm.prnC(cpm.CPU.States.DE.Lo)

	cpm.CPU.States.HL.SetU16(0x0000)
	return err
}

// BdosSysCallRawIO handles both simple character output, and input.
//
// Note that we have to poll and determine if character input is present
// in this function, otherwise games and things don't work well without it.
//
// Blocking in the handler for 0xFF will make ZORK X work, but not other things
// this is the single hardest function to work with.  Meh.
func BdosSysCallRawIO(cpm *CPM) error {

	cpm.CPU.States.HL.SetU16(0x0000)

	switch cpm.CPU.States.DE.Lo {
	case 0xFF:
		// Return a character without echoing if one is waiting; zero if none is available.
		if cpm.input.PendingInput() {
			out, err := cpm.input.BlockForCharacterNoEcho()
			if err != nil {
				return err
			}
			cpm.CPU.States.HL.SetU16(uint16(out))
		}
		return nil
	case 0xFE:

		// Return console input status. Zero if no character is waiting, nonzero otherwise.
		if cpm.input.PendingInput() {
			cpm.CPU.States.HL.SetU16(0x00FF)
		}
		return nil
	case 0xFD:
		// Wait until a character is ready, return it without echoing.
		out, err := cpm.input.BlockForCharacterNoEcho()
		if err != nil {
			return err
		}
		cpm.CPU.States.HL.SetU16(uint16(out))
		return nil
	default:
		// Anything else is to output a character.
		cpm.output.PutCharacter(cpm.CPU.States.DE.Lo)
	}
	return nil
}

// BdosSysCallGetIOByte gets the IOByte, which is used to describe which devices
// are used for I/O.  No CP/M utilities use it, except for STAT and PIP.
//
// The IOByte lives at 0x0003 in RAM, so it is often accessed directly when it is used.
func BdosSysCallGetIOByte(cpm *CPM) error {

	// Get the value
	c := cpm.Memory.Get(0x0003)

	// return it
	cpm.CPU.States.HL.SetU16(uint16(c))
	return nil
}

// BdosSysCallSetIOByte sets the IOByte, which is used to describe which devices
// are used for I/O.  No CP/M utilities use it, except for STAT and PIP.
//
// The IOByte lives at 0x0003 in RAM, so it is often accessed directly when it is used.
func BdosSysCallSetIOByte(cpm *CPM) error {

	// Set the value
	cpm.Memory.Set(0x003, cpm.CPU.States.DE.Lo)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallWriteString writes the $-terminated string pointed to by DE to STDOUT
func BdosSysCallWriteString(cpm *CPM) error {
	addr := cpm.CPU.States.DE.U16()

	str := ""

	c := cpm.Memory.Get(addr)
	for c != '$' {
		// save the string we write
		str += string(c)

		cpm.output.PutCharacter(c)
		addr++
		c = cpm.Memory.Get(addr)
	}

	// Log the message we wrote, and its length.
	cpm.log = slog.With(
		slog.String("output", str),
		slog.String("length", fmt.Sprintf("%d", len(str))))

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallReadString reads a string from the console, into the buffer pointed to by DE.
func BdosSysCallReadString(cpm *CPM) error {

	// DE points to the buffer
	addr := cpm.CPU.States.DE.U16()

	// If DE is 0x0000 then the DMA area is used instead.
	if addr == 0 {
		addr = cpm.dma
	}

	// First byte is the max len
	max := cpm.Memory.Get(addr)

	// read the input
	text, err := cpm.input.ReadLine(max)

	if err != nil {

		// Ctrl-C pressed during input.
		if err == consolein.ErrInterrupted {

			// Reboot the system
			return ErrBoot
		}

		// We used the command-execution method
		// and this resulted in output to send to
		// the console/user.
		if err == consolein.ErrShowOutput {

			cpm.output.WriteString(text)

			// Now we're going to re-run.
			return BdosSysCallReadString(cpm)
		}

		return err
	}

	// Log the input the console received, and its length.
	cpm.log = slog.With(
		slog.String("input", text),
		slog.String("length", fmt.Sprintf("%d", len(text))))

	// addr[0] is the size of the input buffer
	// addr[1] should be the size of input read, set it:
	cpm.Memory.Set(addr+1, uint8(len(text)))

	// addr[2+] should be the text
	i := 0
	for i < len(text) {
		cpm.Memory.Set(uint16(addr+2+uint16(i)), text[i])
		i++
	}

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallConsoleStatus tests if we have pending console (character) input.
func BdosSysCallConsoleStatus(cpm *CPM) error {

	// Default to assuming nothing is pending
	cpm.CPU.States.HL.SetU16(0x0000)

	if cpm.input.PendingInput() {
		cpm.CPU.States.HL.SetU16(0x00FF)
	}
	return nil
}

// BdosSysCallBDOSVersion returns version details
func BdosSysCallBDOSVersion(cpm *CPM) error {

	// HL = 0x0022 -CP/M 2.2
	cpm.CPU.States.HL.SetU16(0x0022)
	return nil
}

// BdosSysCallDriveAllReset resets the drives.
//
// If there is a file named "$..." then we need to return 0xFF in A,
// which will be read by the CCP - as created by SUBMIT.COM
func BdosSysCallDriveAllReset(cpm *CPM) error {

	// Reset disk - but leave the user-number alone
	cpm.currentDrive = 0

	// Update RAM
	cpm.Memory.Set(0x0004, (cpm.userNumber<<4 | cpm.currentDrive))

	// Default return value
	var ret uint16 = 0

	// drive will default to our current drive, if the FCB drive field is 0
	drive := string(cpm.currentDrive + 'A')

	// Remap to the place we're supposed to use.
	path := cpm.drives[drive]

	// Look for a file with $ in its name
	files, err := os.ReadDir(path)
	if err == nil {
		for _, n := range files {
			if strings.Contains(n.Name(), "$") {
				ret = 0xFF
			}
		}
	}

	// Reset our DMA address to the default
	cpm.dma = 0x80

	// Return values:
	cpm.CPU.States.HL.SetU16(ret)
	return nil
}

// BdosSysCallDriveSet updates the current drive number.
func BdosSysCallDriveSet(cpm *CPM) error {

	// The drive number passed to this routine is 0 for A:, 1 for B:
	// up to 15 for P:.
	drv := cpm.CPU.States.AF.Hi

	// P: is the maximum
	if drv > 15 {
		drv = 15
	}

	// set the drive
	cpm.currentDrive = drv

	// Update RAM
	cpm.Memory.Set(0x0004, (cpm.userNumber<<4 | cpm.currentDrive))

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)

	return nil
}

// BdosSysCallFileOpen opens the filename that matches the pattern on the FCB supplied in DE
func BdosSysCallFileOpen(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Reset the offset
	fcbPtr.Ex = 0
	fcbPtr.S1 = 0
	fcbPtr.S2 = 0
	fcbPtr.RC = 0
	fcbPtr.Cr = 0

	// Get the actual name
	fileName := fcbPtr.GetFileName()

	// No filename?  That's an error
	if fileName == "" {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Remap to the place we're supposed to use.
	path := cpm.drives[string(drive)]

	//
	// Ok we have a filename, but we probably have an upper-case
	// filename.
	//
	// Run a glob, and if there's an existing file with the same
	// name then replace with the mixed/lower cased version.
	//
	files, err2 := os.ReadDir(path)
	if err2 == nil {
		for _, n := range files {
			if strings.ToUpper(n.Name()) == fileName {
				fileName = n.Name()
			}
		}
	}

	// Ensure the filename is qualified
	fileName = filepath.Join(path, fileName)

	// Remapped file
	x := filepath.Base(fileName)
	x = filepath.Join(string(cpm.currentDrive+'A'), x)

	// Can we open this file from our embedded filesystem?
	virt, er := cpm.static.ReadFile(x)
	if er == nil {

		// Yes we can!
		// Save the file handle in our cache.
		cpm.files[fcbPtr.GetCacheKey()] = FileCache{name: fileName, handle: nil}

		// Get file size, in blocks
		fLen := uint8(len(virt) / blkSize)

		// Set record-count
		fcbPtr.RC = maxRC
		if fLen < maxRC {
			fcbPtr.RC = fLen
		}

		// Update the FCB in memory.
		cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

		// Return success
		cpm.CPU.States.HL.SetU16(0x0000)
		return nil
	}

	// Now we open from the filesystem
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {

		// We might fail to open a file because it doesn't exist.
		if os.IsNotExist(err) {

			cpm.CPU.States.HL.SetU16(0x00FF)
			return nil
		}

		// Report the failure, but keep going.
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Save the file handle in our cache.
	cpm.files[fcbPtr.GetCacheKey()] = FileCache{name: fileName, handle: file}

	// Get file size, in bytes
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file size of %s: %s", fileName, err)
	}

	// Get file size, in bytes
	fileSize := fi.Size()

	// Get file size, in blocks
	fLen := uint8(fileSize / blkSize)

	// Set record-count
	fcbPtr.RC = maxRC
	if fLen < maxRC {
		fcbPtr.RC = fLen
	}

	// If the size is bigger than a multiple we deal with that.
	if fileSize > int64(int64(fLen)*int64(blkSize)) {
		fcbPtr.RC += 1
	}

	// Update the FCB in memory.
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// Return success
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFileClose closes the filename that matches the pattern on the FCB supplied in DE.
//
// To handle SUBMIT we need to also do more than close an existing file handle, and remove
// it from our cache.  It seems that we can also be required to _truncate_ a file. Because
// I'm unsure exactly how much this is in-use I'm going to only implement it for
// files with "$" in their name.
func BdosSysCallFileClose(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the file handle from our cache.
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// delete the entry from the cache - regardless
	// of success/failure.
	delete(cpm.files, fcbPtr.GetCacheKey())

	// Close of a virtual file.
	if obj.handle == nil {
		// Record success
		cpm.CPU.States.HL.SetU16(0x0000)
		return nil
	}

	// Is this a file created by submit?
	if strings.HasSuffix(obj.name, "$$$.SUB") {
		// Get the file size, in records
		hostSize, _ := obj.handle.Seek(0, 2)
		hostExtent := int((hostSize) / 16384)

		seqEXT := int(fcbPtr.Ex)*32 + int(0x3F&fcbPtr.S2)
		seqCR := func(n int64) int {
			return int(((n) % 16384) / 128)
		}

		if hostExtent == seqEXT {
			if int(fcbPtr.RC) < seqCR(hostSize) {
				hostSize = int64(16384*seqEXT + int(128*int(fcbPtr.RC)))
				err := obj.handle.Truncate(hostSize)
				if err != nil {
					return fmt.Errorf("error truncating file %s: %s", obj.name, err)
				}
			}
		}
	}
	// close the handle
	err := obj.handle.Close()
	if err != nil {
		return fmt.Errorf("failed to close file %04X:%s", ptr, err)
	}

	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// Record success
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFindFirst finds the first filename, on disk, that matches the glob in the FCB supplied in DE.
func BdosSysCallFindFirst(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Previous results are now invalidated
	cpm.findFirstResults = []fcb.Find{}

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Look in the correct location.
	dir := cpm.drives[string(cpm.currentDrive+'A')]

	// Find files in the FCB.
	res, err := fcbPtr.GetMatches(dir)
	if err != nil {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Add on any virtual files, by merging the drive.
	_ = fs.WalkDir(cpm.static, string(cpm.currentDrive+'A'),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			// Does the entry match the glob?
			if fcbPtr.DoesMatch(filepath.Base(path)) {

				// If so append
				res = append(res, fcb.Find{
					Host: path,
					Name: filepath.Base(path)})
			}

			return nil
		})

	// No matches?  Return an error
	if len(res) < 1 {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Sort the list, since we've added the embedded files
	// onto the end and that will look weird.
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	cpm.log = cpm.log.With(
		slog.Group("glob",
			slog.String("pattern", fcbPtr.GetFileName()),
			slog.Int("matches", len(res))))

	// Build up all the results so we can log those.
	tmpn := []any{}
	for i, e := range res {
		tmpn = append(tmpn, slog.String(fmt.Sprintf("match_%d", i), e.Name))
	}

	// Now make those available for logging.
	cpm.log = cpm.log.With(
		slog.Group("matches", tmpn...))

	// Here we save the results in our cache,
	// dropping the first
	cpm.findFirstResults = res[1:]

	// Create a new FCB and store it in the DMA entry
	x := fcb.FromString(res[0].Name)

	// Get the file-size in records, and add to the FCB
	tmp, err := os.OpenFile(res[0].Host, os.O_RDONLY, 0644)
	if err == nil {
		defer tmp.Close()

		fi, err := tmp.Stat()
		if err == nil {

			fileSize := fi.Size()

			// Get file size, in blocks
			x.RC = uint8(fileSize / blkSize)

			// If the size is bigger than a multiple we deal with that.
			if fileSize > int64(int64(x.RC)*int64(blkSize)) {
				x.RC += 1
			}

		}
	}

	// Log the first result we're returning.
	cpm.log = cpm.log.With(
		slog.Group("returning",
			slog.String("name", x.GetFileName()),
			slog.String("RecordCount", fmt.Sprintf("%d", x.RC))))

	// Update the results
	data := x.AsBytes()
	cpm.Memory.SetRange(cpm.dma, data...)

	// Return 0x00 to point to the first entry in the DMA area.
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFindNext finds the next filename that matches the glob set in the FCB in DE.
func BdosSysCallFindNext(cpm *CPM) error {
	//
	// Assume we've been called with findFirst before
	//
	if len(cpm.findFirstResults) == 0 {
		// Return 0xFF to signal an error
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Get the first item from the list of pending files
	res := cpm.findFirstResults[0]

	// And update our list to remove it.
	cpm.findFirstResults = cpm.findFirstResults[1:]

	// Create a new FCB and store it in the DMA entry
	x := fcb.FromString(res.Name)

	// Get the file-size in records, and add to the FCB
	tmp, err := os.OpenFile(res.Host, os.O_RDONLY, 0644)
	if err == nil {
		defer tmp.Close()

		fi, err := tmp.Stat()
		if err == nil {

			fileSize := fi.Size()

			// Get file size, in blocks
			x.RC = uint8(fileSize / blkSize)

			// If the size is bigger than a multiple we deal with that.
			if fileSize > int64(int64(x.RC)*int64(blkSize)) {
				x.RC += 1
			}

		}
	}

	// Log that we're returning the next result.
	cpm.log = cpm.log.With(
		slog.Group("returning",
			slog.String("name", x.GetFileName()),
			slog.String("RecordCount", fmt.Sprintf("%d", x.RC))))

	data := x.AsBytes()
	cpm.Memory.SetRange(cpm.dma, data...)

	// Return 0x00 to point to the first entry in the DMA area.
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDeleteFile deletes the filename(s) matching the pattern specified by the FCB in DE.
func BdosSysCallDeleteFile(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Remap to the place we're supposed to use.
	path := cpm.drives[string(drive)]

	// Find files that match the FCB-pattern.
	res, err := fcbPtr.GetMatches(path)
	if err != nil {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// For each result, if any
	for _, entry := range res {

		// Host path
		path := entry.Host

		// Ensure we don't have this cached
		x := fcb.FromString(entry.Name)

		// If we have a cached handle ensure we close the file,
		// then delete the entry.
		obj, ok := cpm.files[x.GetCacheKey()]
		if ok {
			obj.handle.Close()
			delete(cpm.files, x.GetCacheKey())
		}

		err = os.Remove(path)
		if err != nil {
			cpm.CPU.States.HL.SetU16(0x00FF)
			return nil
		}
	}

	// Build up all the results so we can log those.
	tmpn := []any{}
	for i, e := range res {
		tmpn = append(tmpn, slog.String(fmt.Sprintf("match_%d", i), e.Name))
	}

	// Now make those available for logging.
	cpm.log = cpm.log.With(
		slog.Group("deleted", tmpn...))

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)
	return err
}

// BdosSysCallRead reads a record from the file named in the FCB given in DE
func BdosSysCallRead(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the file handle in our cache.
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Temporary area to read into
	data := make([]byte, blkSize)

	// Fill the area with data
	for i := range data {
		data[i] = 0x1A
	}

	// Get the next read position
	offset := fcbPtr.GetSequentialOffset()

	// Are we reading from a virtual file?
	if obj.handle == nil {

		// Remap
		p := filepath.Join(string(cpm.currentDrive+'A'), filepath.Base(obj.name))

		// open
		file, err := fs.ReadFile(cpm.static, p)
		if err != nil {
			cpm.CPU.States.HL.SetU16(0x00FF)
			return nil
		}
		i := 0

		// default to being successful
		cpm.CPU.States.HL.SetU16(0x0000)

		// copy each appropriate byte into the data-area
		for i < blkSize {
			if int(offset)+i < len(file) {
				data[i] = file[int(offset)+i]
			} else {
				cpm.CPU.States.HL.SetU16(0x0001)
			}
			i++
		}

		// Copy the data to the DMA area
		cpm.Memory.SetRange(cpm.dma, data...)

		// Update the next read position
		fcbPtr.IncreaseSequentialOffset()

		// Update the FCB in memory
		cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

		// All done
		return nil
	}

	_, err := obj.handle.Seek(int64(offset), io.SeekStart)
	if err != nil {
		cpm.CPU.States.HL.SetU16(0x0001)
		return fmt.Errorf("cannot seek to position %d: %s", offset, err)
	}

	// Read from the file, now we're in the right place
	_, err = obj.handle.Read(data)
	if err != nil && err != io.EOF {
		cpm.CPU.States.HL.SetU16(0x0001)
		return fmt.Errorf("error reading file %s", err)
	}

	// Copy the data to the DMA area
	cpm.Memory.SetRange(cpm.dma, data...)

	// Update the next read position
	fcbPtr.IncreaseSequentialOffset()

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// All done
	cpm.CPU.States.HL.SetU16(0x0000)
	if err == io.EOF {
		cpm.CPU.States.HL.SetU16(0x0001)
	}

	return nil
}

// BdosSysCallWrite writes a record to the file named in the FCB given in DE
func BdosSysCallWrite(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the file handle in our cache.
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// A virtual handle, from our embedded resources.
	if obj.handle == nil {
		return fmt.Errorf("fatal error SysCallWrite against an embedded resource %v", obj)
	}

	// Get the next write position
	offset := fcbPtr.GetSequentialOffset()

	// Get the data range from the DMA area
	data := cpm.Memory.GetRange(cpm.dma, 128)

	// Move to the correct place
	_, err := obj.handle.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot seek to position %d: %s", offset, err)
	}

	// Write to the open file
	_, err = obj.handle.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file %s", err)
	}

	// Update the next write position
	fcbPtr.IncreaseSequentialOffset()

	// Sigh.
	fcbPtr.RC++

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// All done
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallMakeFile creates the file named in the FCB given in DE
func BdosSysCallMakeFile(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Reset the offset
	fcbPtr.Ex = 0
	fcbPtr.S1 = 0
	fcbPtr.S2 = 0
	fcbPtr.RC = 0
	fcbPtr.Cr = 0

	// Get the actual name
	fileName := fcbPtr.GetFileName()

	// Is this already cached?  Then cleanup by
	// closing the file, and removing the cache-key
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if ok {
		obj.handle.Close()
		delete(cpm.files, fcbPtr.GetCacheKey())
	}

	// No filename?  That's an error
	if fileName == "" {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Remap to the place we're supposed to use.
	path := cpm.drives[string(drive)]

	//
	// Ok we have a filename, but we probably have an upper-case
	// filename.
	//
	// Run a glob, and if there's an existing file with the same
	// name then replace with the mixed/lower cased version.
	//
	files, err2 := os.ReadDir(path)
	if err2 == nil {
		for _, n := range files {
			if strings.ToUpper(n.Name()) == fileName {
				fileName = n.Name()
			}
		}
	}

	// Qualify the path
	fileName = filepath.Join(path, fileName)

	// Create the file
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// If the file already exists, truncate and rewind.
	//
	// This ensures that "make file" will always result
	// in either a) an error, or b) an empty file ready
	// for use.
	_ = file.Truncate(0)
	_, _ = file.Seek(0, io.SeekStart)
	_ = file.Sync()

	// Get file size, in bytes
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file size of %s: %s", fileName, err)
	}

	// Get file size, in bytes
	fileSize := fi.Size()

	// Get file size, in blocks
	fLen := uint8(fileSize / blkSize)

	// Set record-count
	fcbPtr.RC = maxRC
	if fLen < maxRC {
		fcbPtr.RC = fLen
	}

	// If the size is bigger than a multiple we deal with that.
	if fileSize > int64(int64(fcbPtr.RC)*int64(blkSize)) {
		fcbPtr.RC += 1
	}

	// Save the file-handle
	cpm.files[fcbPtr.GetCacheKey()] = FileCache{name: fileName, handle: file}

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallRenameFile will handle a rename operation.
// Note that this will not handle cross-directory renames (i.e. file moving).
func BdosSysCallRenameFile(cpm *CPM) error {

	// 1. SRC

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("src",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the actual name
	fileName := fcbPtr.GetFileName()

	// Point to the directory
	path := cpm.drives[string(cpm.currentDrive+'A')]

	//
	// Ok we have a filename, but we probably have an upper-case
	// filename.
	//
	// Run a glob, and if there's an existing file with the same
	// name then replace with the mixed/lower cased version.
	//
	files, err2 := os.ReadDir(path)
	if err2 == nil {
		for _, n := range files {
			if strings.ToUpper(n.Name()) == fileName {
				fileName = n.Name()
			}
		}
	}

	// Ensure the filename is qualified
	fileName = filepath.Join(path, fileName)

	// 2. DEST
	// The pointer to the FCB
	xxx2 := cpm.Memory.GetRange(ptr+16, fcb.SIZE)

	// Create a structure with the contents
	dstPtr := fcb.FromBytes(xxx2)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("dst",
			slog.String("drive", fmt.Sprintf("%02X", dstPtr.Drive)),
			slog.String("name", dstPtr.GetName()),
			slog.String("type", dstPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", dstPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", dstPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", dstPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", dstPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", dstPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", dstPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", dstPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", dstPtr.R2)),
		))

	// Get the name
	dstName := dstPtr.GetFileName()

	// ensure the name is qualified
	dstName = filepath.Join(path, dstName)

	err := os.Rename(fileName, dstName)
	if err != nil {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Return values
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallLoginVec returns the list of logged in drives.
func BdosSysCallLoginVec(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0xFF
	cpm.CPU.States.HL.Lo = 0xFF
	return nil
}

// BdosSysCallDriveGet returns the number of the active drive.
func BdosSysCallDriveGet(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(uint16(cpm.currentDrive))
	return nil
}

// BdosSysCallSetDMA updates the address of the DMA area, which is used for block I/O.
func BdosSysCallSetDMA(cpm *CPM) error {

	// Get the address from BC
	addr := cpm.CPU.States.DE.U16()

	// Update the DMA value.
	cpm.dma = addr

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDriveAlloc will return the address of the allocation bitmap (which blocks are used and
// which are free) in HL.
func BdosSysCallDriveAlloc(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDriveSetRO will mark the current drive as being read-only.
//
// This call is faked.
func BdosSysCallDriveSetRO(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDriveROVec will return a bitfield describing which drives are read-only.
//
// Bit 7 of H corresponds to P: while bit 0 of L corresponds to A:. A bit is set if the corresponding drive is
// set to read-only in software.  As we never set drives to read-only we return 0x0000
func BdosSysCallDriveROVec(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallSetFileAttributes should update the attributes of the given
// file, but it fakes it.
func BdosSysCallSetFileAttributes(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallGetDriveDPB returns the address of the DPB, which is faked.
func BdosSysCallGetDriveDPB(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallUserNumber gets, or sets, the user-number.
func BdosSysCallUserNumber(cpm *CPM) error {

	// We're either setting or getting
	//
	// If the value is 0xFF we return it, otherwise we set
	if cpm.CPU.States.DE.Lo != 0xFF {

		// Set the number - masked, because valid values are 0-15
		cpm.userNumber = (cpm.CPU.States.DE.Lo & 0x0F)

		// Update RAM
		cpm.Memory.Set(0x0004, (cpm.userNumber<<4 | cpm.currentDrive))
	}

	// Return values:
	cpm.CPU.States.HL.SetU16(uint16(cpm.userNumber))
	return nil
}

// BdosSysCallReadRand reads a random block from the FCB pointed to by DE into the DMA area.
func BdosSysCallReadRand(cpm *CPM) error {
	// Temporary area to read into
	data := make([]byte, blkSize)

	// sysRead reads from the given offset
	//
	// Return:
	//  0 : read something successfully
	//  1 : read nothing - error really
	//
	sysRead := func(f *os.File, offset int64) int {

		// Get file size, in bytes
		fi, err := f.Stat()
		if err != nil {
			return 0xFF
		}
		fileSize := fi.Size()

		// If the offset we're reading from is bigger than the file size then
		// pad it up
		if offset > fileSize {
			return 06
		}

		_, err = f.Seek(offset, io.SeekStart)
		if err != nil {
			return 0xFF
		}

		for i := range data {
			data[i] = 0x1A
		}

		_, err = f.Read(data)
		if err != nil {
			if err != io.EOF {
				return 0xFF
			}
		}

		// Log the record we read.
		cpm.log = cpm.log.With(
			slog.Group("record",
				slog.String("offset", fmt.Sprintf("%d", offset)),
				slog.String("size", fmt.Sprintf("%d", fileSize)),
				slog.String("dump", data2String(data))))

		cpm.Memory.SetRange(cpm.dma, data...)
		return 0
	}

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the file handle in our cache.
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// A virtual handle, from our embedded resources.
	if obj.handle == nil {

		// Remap
		p := filepath.Join(string(cpm.currentDrive+'A'), filepath.Base(obj.name))

		// open
		file, err := fs.ReadFile(cpm.static, p)
		if err != nil {
			cpm.CPU.States.HL.SetU16(0x00FF)
			return nil
		}

		// Get the record to read
		record := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)

		// Translate the record to a byte-offset
		offset := int64(record) * blkSize

		// copy each appropriate byte into the data-area
		i := 0
		var res uint8
		for i < blkSize {
			if int(offset)+i < len(file) {
				data[i] = file[int(offset)+i]
			} else {
				res = 0x01
			}
			i++
		}

		// Copy the data to the DMA area
		cpm.Memory.SetRange(cpm.dma, data...)

		cpm.CPU.States.HL.SetU16(uint16(res))
		return nil
	}

	// Get the record to read
	record := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)

	// Translate the record to a byte-offset
	fpos := int64(record) * blkSize

	// Read the data
	res := sysRead(obj.handle, fpos)

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	cpm.CPU.States.HL.SetU16(uint16(res))
	return nil
}

// BdosSysCallWriteRand writes a random block from DMA area to the FCB pointed to by DE.
func BdosSysCallWriteRand(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// Get the file handle in our cache.
	obj, ok := cpm.files[fcbPtr.GetCacheKey()]
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// A virtual handle, from our embedded resources.
	if obj.handle == nil {
		return fmt.Errorf("fatal error SysCallWriteRand against an embedded resource %v", obj)
	}

	// Get the data range from the DMA area
	data := cpm.Memory.GetRange(cpm.dma, 128)

	// Log the record we're writing
	cpm.log = cpm.log.With(
		slog.Group("record",
			slog.String("dump", data2String(data))))

	// Get the record to write
	record := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)

	// Get the file position that translates to
	fpos := int64(record) * blkSize

	// Get file size, in bytes
	fi, err := obj.handle.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file size of: %s", err)
	}
	fileSize := fi.Size()

	// If the offset we're writing to is bigger than the file size then
	// we need to add an appropriate amount of padding.
	padding := fpos - fileSize

	for padding > 0 {
		_, er := obj.handle.Write([]byte{0x00})
		if er != nil {
			return fmt.Errorf("error adding padding: %s", er)
		}
		padding--
	}

	_, err = obj.handle.Seek(fpos, io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot seek to position %d: %s", fpos, err)
	}

	_, err = obj.handle.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to offset %d: %s", fpos, err)
	}

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFileSize updates the Random Record bytes of the given FCB to the
// number of records in the file.
func BdosSysCallFileSize(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	//
	// Seems this doesn't require a file to be open.
	//
	// So we have to go through the dance of getting the filename.
	//

	// Get the actual name
	fileName := fcbPtr.GetFileName()

	// Should we remap drives?
	path := cpm.drives[string(cpm.currentDrive+'A')]

	//
	// Ok we have a filename, but we probably have an upper-case
	// filename.
	//
	// Run a glob, and if there's an existing file with the same
	// name then replace with the mixed/lower cased version.
	//
	files, err2 := os.ReadDir(path)
	if err2 == nil {
		for _, n := range files {
			if strings.ToUpper(n.Name()) == fileName {
				fileName = n.Name()
			}
		}
	}

	// ensure the path is qualified
	fileName = filepath.Join(path, fileName)

	// Remapped file
	x := filepath.Base(fileName)
	x = filepath.Join(string(cpm.currentDrive+'A'), x)

	// fileSize we'll determine
	var fileSize int64

	// Can we open this file from our embedded filesystem?
	virt, er := cpm.static.ReadFile(x)
	if er == nil {

		fileSize = int64(len(virt))
	} else {

		file, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open file for FileSize %s:%s", fileName, err)
		}

		// ensure we close
		defer file.Close()

		// Get file size, in bytes
		fi, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file size of %s: %s", fileName, err)
		}

		fileSize = fi.Size()

	}

	// Now we have the size we need to turn it into the number
	// of records
	records := int(fileSize / 128)

	// Block size is used so round up, if we need to.
	if fileSize > int64(int64(records)*int64(blkSize)) {
		records += 1
	}

	// Cap the size appropriately.
	if records >= 65536 {
		records = 65536
	}

	// Store the value in the three fields
	fcbPtr.R0 = uint8(records & 0xFF)
	fcbPtr.R1 = uint8(records >> 8)
	fcbPtr.R2 = uint8(records >> 16)

	// sanity check because I've messed this up in the past
	n := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)
	if n != records {
		return fmt.Errorf("failed to update because maths is hard %d != %d", n, records)
	}

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallRandRecord Sets the random record count bytes of the FCB to the number
// of the last record read/written by the sequential I/O calls.
func BdosSysCallRandRecord(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Log the FCB
	cpm.log = cpm.log.With(
		slog.Group("fcb",
			slog.String("drive", fmt.Sprintf("%02X", fcbPtr.Drive)),
			slog.String("name", fcbPtr.GetName()),
			slog.String("type", fcbPtr.GetType()),
			slog.String("Ex", fmt.Sprintf("%02X", fcbPtr.Ex)),
			slog.String("S1", fmt.Sprintf("%02X", fcbPtr.S1)),
			slog.String("S2", fmt.Sprintf("%02X", fcbPtr.S2)),
			slog.String("RC", fmt.Sprintf("%02X", fcbPtr.RC)),
			slog.String("CR", fmt.Sprintf("%02X", fcbPtr.Cr)),
			slog.String("R0", fmt.Sprintf("%02X", fcbPtr.R0)),
			slog.String("R1", fmt.Sprintf("%02X", fcbPtr.R1)),
			slog.String("R2", fmt.Sprintf("%02X", fcbPtr.R2)),
		))

	// So the sequential offset is found here
	offset := int(fcbPtr.GetSequentialOffset())

	// Now we set the "random record" which is R0,R1,R2
	fcbPtr.R0 = uint8(offset & 0xFF)
	fcbPtr.R1 = uint8(offset >> 8)
	fcbPtr.R2 = uint8(offset >> 16)

	// sanity check because I've messed this up in the past
	n := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)
	if n != offset {
		return fmt.Errorf("failed to update because maths is hard %d != %d", n, offset)
	}

	// Update the FCB in memory.
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// Return success
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDriveReset allows resetting specific drives, via the bits in DE
// Bit 7 of D corresponds to P: while bit 0 of E corresponds to A:.
// A bit is set if the corresponding drive should be reset.
// Resetting a drive removes its software read-only status.
func BdosSysCallDriveReset(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFileLock implements a NOP version of F_LOCK
func BdosSysCallFileLock(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x00FF)
	return nil
}

// BdosSysCallErrorMode implements a NOP version of F_ERRMODE.
func BdosSysCallErrorMode(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDriveFlush implements a NOP version of DRV_FLUSH
func BdosSysCallDriveFlush(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallFileTimeDate implements a NOP version of F_TIMEDATE
func BdosSysCallFileTimeDate(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallTime implements a NOP version of T_GET.
func BdosSysCallTime(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallDirectScreenFunctions receives a pointer in DE to a parameter block,
// which specifies which function to run.  I've only seen this invoked in
// TurboPascal when choosing the "Execute" or "Run" options.
func BdosSysCallDirectScreenFunctions(cpm *CPM) error {
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallUptime returns the number of "ticks" since the system
// was booted, it is a custom syscall which is implemented by RunCPM
// which we implement for compatibility, notable users include v5
// of BBC BASIC.
func BdosSysCallUptime(cpm *CPM) error {

	// Get elapsed time, since startup
	elapsed := time.Since(cpm.launchTime)

	// In nanoseconds
	timer := elapsed.Nanoseconds()

	// Set it.
	cpm.CPU.States.HL.SetU16(uint16(timer & 0xFFFF))
	cpm.CPU.States.DE.SetU16(uint16((timer >> 16) & 0xFFFF))
	return nil
}
