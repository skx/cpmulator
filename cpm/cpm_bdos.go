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

// BdosSysCallExit implements the Exit syscall
func BdosSysCallExit(cpm *CPM) error {
	cpm.CPU.HALT = true
	return ErrBoot
}

// TODO - rename
func fcbToHost(cpm *CPM, fcb fcb.FCB) (string, error) {

	// Get the actual name
	name := strings.ToUpper(fcb.GetFileName())

	// Get the path to which we should search.
	// This is the currently selected driver
	// TODO:
	//  Should we only use this if the fcb-drive is zero?
	//
	path := cpm.drives[string(cpm.currentDrive+'A')]

	//
	// Ok we have a filename, but we probably have an upper-case
	// filename.
	//
	// Run a glob, and if there's an existing file with the same
	// name then replace with the mixed/lower cased version.
	//
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	for _, n := range files {
		if strings.ToUpper(n.Name()) == name {
			name = n.Name()
		}
	}

	// ensure the path is qualified - with our updated name
	name = filepath.Join(path, name)

	// Remapped file
	x := filepath.Base(name)
	x = filepath.Join(string(cpm.currentDrive+'A'), x)

	return name, nil
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

	c := cpm.Memory.Get(addr)
	for c != '$' {
		cpm.output.PutCharacter(c)
		addr++
		c = cpm.Memory.Get(addr)
	}

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
	var ret uint8 = 0

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
	cpm.CPU.States.HL.SetU16(uint16(ret))
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

// BdosSysCallFileOpen opens the filename that matches the pattern on the FCB supplied in DE.
//
// TODO: We don't handle virtual files here.
func BdosSysCallFileOpen(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create an FCB object
	f := fcb.FromBytes(xxx)

	// Lookup the object in our cache
	//
	// If the file was already open then we do nothing.
	// and report that as a success.
	ent, ok := cpm.files[f.GetCacheKey()]
	if ok {

		// seek to the start of the file
		ent.handle.Seek(0, io.SeekStart)

		// set the record-count
		f.SetRecordCount(ent.handle)
		f.S2 = 0x00

		// Update the FCB in RAM
		data := f.AsBytes()
		cpm.Memory.SetRange(ptr, data...)

		// return success
		cpm.CPU.States.HL.SetU16(0x0000)
		cpm.CPU.States.AF.Hi = 0x00
		cpm.CPU.States.BC.Hi = 0x00
	}

	// Find out where we should open
	path, err := fcbToHost(cpm, f)

	// Error opening?  Then return that to the caller.
	if err != nil {
		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// Now we can open the file and see what we have
	handle, err := os.Open(path)

	// again if there is an error let the caller known
	if err != nil {

		slog.Debug("failed to open file",
			slog.String("file", path),
			slog.String("error", err.Error()))

		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// Create a cache entry
	cache := FileCache{
		name:   f.GetFileName(),
		host:   path,
		handle: handle,
	}

	// store it
	cpm.files[f.GetCacheKey()] = cache

	// set the record-count of the file.
	f.SetRecordCount(ent.handle)
	f.S2 = 0x00

	// Update the FCB in RAM
	data := f.AsBytes()
	cpm.Memory.SetRange(ptr, data...)

	// return success
	cpm.CPU.States.HL.SetU16(0x0000)
	cpm.CPU.States.AF.Hi = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	return nil
}

// BdosSysCallFileClose closes the filename that matches the pattern on the FCB supplied in DE.
//
// To handle SUBMIT we need to also do more than close an existing file handle, and remove
// it from our cache.  It seems that we can also be required to _truncate_ a file - for the
// moment this code has been removed.
//
// TODO: Handle truncation on closure.
func BdosSysCallFileClose(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create an FCB object
	f := fcb.FromBytes(xxx)

	// Lookup the object in our cache
	ent, ok := cpm.files[f.GetCacheKey()]

	// Not found in the cache?  Then the file
	// was not open and we return an error
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// Sync the file.
	err := ent.handle.Sync()
	if err != nil {
		slog.Debug("failed to sync file",
			slog.String("file", ent.name),
			slog.String("error", err.Error()))
	}

	// Close the file
	err = ent.handle.Close()
	if err != nil {
		slog.Debug("failed to close file",
			slog.String("file", ent.name),
			slog.String("error", err.Error()))
	}

	// Remove from the cache
	delete(cpm.files, f.GetCacheKey())

	// Return success
	cpm.CPU.States.HL.SetU16(0x0000)
	cpm.CPU.States.AF.Hi = 0x00
	cpm.CPU.States.BC.Hi = 0x00
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

	// Look in the correct location.
	dir := cpm.drives[string(cpm.currentDrive+'A')]

	// Find files in the FCB.
	res, err := fcbPtr.GetMatches(dir)
	if err != nil {
		slog.Debug("fcbPtr.GetMatches returned error",
			slog.String("path", dir),
			slog.String("error", err.Error()))

		// Error
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

		}
	}

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

		}
	}

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

	// Show what we're going to delete
	slog.Debug("SysCallDeleteFile",
		slog.String("pattern", fcbPtr.GetFileName()))

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Remap to the place we're supposed to use.
	path := cpm.drives[string(drive)]

	// Find files in the FCB.
	res, err := fcbPtr.GetMatches(path)
	if err != nil {
		slog.Debug("SysCallDeleteFile - fcbPtr.GetMatches returned error",
			slog.String("path", path),
			slog.String("error", err.Error()))

		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// For each result, if any
	for _, entry := range res {

		// Host path
		path := entry.Host

		slog.Debug("SysCallDeleteFile: deleting file",
			slog.String("path", path))

		err = os.Remove(path)
		if err != nil {

			slog.Debug("SysCallDeleteFile: failed to delete file",
				slog.String("path", path),
				slog.String("error", err.Error()))

			cpm.CPU.States.HL.SetU16(0x00FF)
			return nil
		}
	}

	// Return values:
	cpm.CPU.States.HL.SetU16(0x0000)
	return err
}

// BdosSysCallFileRead reads a record from the file named in the FCB given in DE
func BdosSysCallFileRead(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create an FCB object
	f := fcb.FromBytes(xxx)

	// Lookup the object in our cache
	ent, ok := cpm.files[f.GetCacheKey()]

	// Not found in the cache?  Then the file
	// was not open and we return an error.
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// OK now we need to read a record of data,
	// so we create a space to read it into
	data := make([]uint8, 128)

	// Fill the data-area with Ctrl-Z
	var i uint8 = 0
	for i < 128 {
		data[i] = 0x1A // ctrl-Z
		i++
	}

	// Get the offset from which we should read.
	offset := f.GetSequentialOffset()
	length, eerr := f.GetFileSize(ent.handle)

	if eerr != nil {
		slog.Debug("failed to get file size",
			slog.String("name", f.GetFileName()),
			slog.String("error", eerr.Error()))

		cpm.CPU.States.HL.SetU16(0x0001)
		cpm.CPU.States.AF.Hi = 0x01
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// reading beyond the end of the file is going to fail.
	if offset > length {
		slog.Debug("reading beyond the end of the file",
			slog.Int64("size", length),
			slog.Int64("offset", offset))

		cpm.CPU.States.HL.SetU16(0x0001)
		cpm.CPU.States.AF.Hi = 0x01
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	slog.Debug("reading file",
		slog.String("name", ent.name),
		slog.Int64("size", length),
		slog.Int64("offset", offset))

	// Seek to that offset.
	_, err := ent.handle.Seek(offset, io.SeekStart)
	if err != nil {
		slog.Debug("failed to seek",
			slog.Int64("size", length),
			slog.Int64("offset", offset),
			slog.String("error", err.Error()))

		cpm.CPU.States.HL.SetU16(0x0001)
		cpm.CPU.States.AF.Hi = 0x01
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	var n int
	n, err = ent.handle.Read(data)
	if err != nil && err != io.EOF {
		slog.Debug("failed to read",
			slog.Int64("size", length),
			slog.Int64("offset", offset),
			slog.String("error", err.Error()))
		cpm.CPU.States.HL.SetU16(0x0001)
		cpm.CPU.States.AF.Hi = 0x01
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	if n > 0 {

		// Update the offset to the next record
		f.UpdateSequentialOffset(offset + 128)

		// Update the FCB in RAM, so that
		// record-change takes effect.
		d := f.AsBytes()
		cpm.Memory.SetRange(ptr, d...)

		// Update the DMA area, with the read data
		cpm.Memory.SetRange(cpm.dma, data...)

		cpm.CPU.States.HL.SetU16(0x0000)
		cpm.CPU.States.AF.Hi = 0x00
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	cpm.CPU.States.HL.SetU16(0x0001)
	cpm.CPU.States.AF.Hi = 0x01
	cpm.CPU.States.BC.Hi = 0x00
	return nil
}

// BdosSysCallFileWrite writes a record to the file named in the FCB given in DE
func BdosSysCallFileWrite(cpm *CPM) error {

	panic("BdosSysCallWrite")

	// TODO
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallMakeFile creates the file named in the FCB given in DE
func BdosSysCallMakeFile(cpm *CPM) error {

	panic("BdosSysCallMakeFile")

	// TODO
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

	// Get the name
	dstName := dstPtr.GetFileName()

	// ensure the name is qualified
	dstName = filepath.Join(path, dstName)

	slog.Debug("Renaming file",
		slog.String("src", fileName),
		slog.String("dst", dstName))

	err := os.Rename(fileName, dstName)
	if err != nil {
		slog.Debug("Renaming file failed",
			slog.String("error", err.Error()))
		cpm.CPU.States.HL.SetU16(0x00FF)
		return nil
	}

	// Return values:
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

	panic("BdosSysCallReadRand")

	// TODO
	cpm.CPU.States.HL.SetU16(0x0000)
	return nil
}

// BdosSysCallWriteRand writes a random block from DMA area to the FCB pointed to by DE.
func BdosSysCallWriteRand(cpm *CPM) error {

	panic("BdosSysCallWriteRand")

	// TODO
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

	// Create an FCB object
	f := fcb.FromBytes(xxx)

	// Lookup the object in our cache
	ent, ok := cpm.files[f.GetCacheKey()]

	// Not found in the cache?  Then the file
	// was not open and we return an error
	if !ok {
		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// Get the size
	fileSize, err := f.GetFileSize(ent.handle)
	if err != nil {
		slog.Debug("failed to get file size",
			slog.String("error", err.Error()))
		cpm.CPU.States.HL.SetU16(0x00FF)
		cpm.CPU.States.AF.Hi = 0xFF
		cpm.CPU.States.BC.Hi = 0x00
		return nil
	}

	// Round up.
	for fileSize%128 != 0 {
		fileSize += 1
	}

	// Update the random IO offset
	f.SetRandomOffset(uint16(fileSize / 128))

	v := f.GetRandomOffset()
	if v != uint16(fileSize/128) {
		panic("mismatch")
	}

	data := f.AsBytes()
	cpm.Memory.SetRange(ptr, data...)

	// return success
	cpm.CPU.States.HL.SetU16(0x0000)
	cpm.CPU.States.AF.Hi = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	return nil
}

// BdosSysCallRandRecord Sets the random record count bytes of the FCB to the number
// of the last record read/written by the sequential I/O calls.
func BdosSysCallRandRecord(cpm *CPM) error {

	panic("BdosSysCallRandRecord")

	// TODO
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
