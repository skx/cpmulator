// This file contains the implementations for the CP/M calls we emulate.
//
// NOTE: They are added to the syscalls map in cpm.go.
//

package cpm

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/skx/cpmulator/fcb"
	cpmio "github.com/skx/cpmulator/io"
)

// blkSize is the size of block-based I/O operations
const blkSize = 128

// maxRC is the maximum read count
const maxRC = 128

// SysCallExit implements the Exit syscall
func SysCallExit(cpm *CPM) error {
	return ErrExit
}

// SysCallReadChar reads a single character from the console.
func SysCallReadChar(cpm *CPM) error {

	// Use our I/O package
	obj := cpmio.New(cpm.Logger)

	// Block for input
	c, err := obj.BlockForCharacterWithEcho()
	if err != nil {
		return fmt.Errorf("error in call to BlockForCharacter: %s", err)
	}

	// Return values:
	// HL = Char, A=Char
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = c
	cpm.CPU.States.AF.Hi = c
	cpm.CPU.States.AF.Lo = 0x00

	return nil
}

// SysCallWriteChar writes the single character in the E register to STDOUT.
func SysCallWriteChar(cpm *CPM) error {

	cpm.outC(cpm.CPU.States.DE.Lo)

	return nil
}

// SysCallAuxRead reads a single character from the auxillary input.
//
// Note: Echo is not enabled in this function.
func SysCallAuxRead(cpm *CPM) error {

	// Use our I/O package
	obj := cpmio.New(cpm.Logger)

	// Block for input
	c, err := obj.BlockForCharacter()
	if err != nil {
		return fmt.Errorf("error in call to BlockForCharacter: %s", err)
	}

	// Return values:
	// HL = Char, A=Char
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = c
	cpm.CPU.States.AF.Hi = c
	cpm.CPU.States.AF.Lo = 0x00

	return nil
}

// SysCallPrinterWrite should send a single character to the printer,
// we fake that by writing to a file instead.
func SysCallPrinterWrite(cpm *CPM) error {

	// If the file doesn't exist, create it.
	f, err := os.OpenFile(cpm.prnPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("SysCallPrinterWrite: Failed to open file 'print.log' %s", err)
	}

	data := make([]byte, 1)
	data[0] = cpm.CPU.States.BC.Lo
	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("SysCallPrinterWrite: Failed to write %s", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("SysCallPrinterWrite: Failed to close %s", err)
	}

	return nil
}

// SysCallAuxWrite writes the single character in the C register
// auxillary / punch output.
func SysCallAuxWrite(cpm *CPM) error {

	// The character we're going to write
	c := cpm.CPU.States.BC.Lo
	cpm.outC(c)
	return nil
}

// SysCallRawIO handles both simple character output, and input.
func SysCallRawIO(cpm *CPM) error {

	// Use our I/O package
	obj := cpmio.New(cpm.Logger)

	switch cpm.CPU.States.DE.Lo {
	case 0xFF, 0xFD:

		out, err := obj.BlockForCharacter()

		// I think this is correct: but it breaks zork
		// TODO: Reassess
		// With this in-place zork runs, mbasic runs, and turbo.com runs
		//		out, err := obj.GetCharOrNull()
		if err != nil {
			return err
		}
		cpm.CPU.States.AF.Hi = out
		return nil
	default:
		cpm.outC(cpm.CPU.States.DE.Lo)
	}
	return nil
}

// SysCallGetIOByte gets the IOByte, which is used to describe which devices
// are used for I/O.  No CP/M utilities use it, except for STAT and PIP.
//
// The IOByte lives at 0x0003 in RAM, so it is often accessed directly when it is used.
func SysCallGetIOByte(cpm *CPM) error {

	// Get the value
	c := cpm.Memory.Get(0x0003)

	// return it
	cpm.CPU.States.AF.Hi = c

	return nil
}

// SysCallSetIOByte sets the IOByte, which is used to describe which devices
// are used for I/O.  No CP/M utilities use it, except for STAT and PIP.
//
// The IOByte lives at 0x0003 in RAM, so it is often accessed directly when it is used.
func SysCallSetIOByte(cpm *CPM) error {

	// Set the value
	cpm.Memory.Set(0x003, cpm.CPU.States.DE.Lo)

	return nil
}

// SysCallWriteString writes the $-terminated string pointed to by DE to STDOUT
func SysCallWriteString(cpm *CPM) error {
	addr := cpm.CPU.States.DE.U16()

	c := cpm.Memory.Get(addr)
	for c != '$' {
		fmt.Printf("%c", c)
		addr++
		c = cpm.Memory.Get(addr)
	}

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallReadString reads a string from the console, into the buffer pointed to by DE.
func SysCallReadString(cpm *CPM) error {

	// DE points to the buffer
	addr := cpm.CPU.States.DE.U16()

	// First byte is the max len
	max := cpm.CPU.Memory.Get(addr)

	obj := cpmio.New(cpm.Logger)
	text, err := obj.ReadLine(max)

	if err != nil {
		return err
	}

	// addr[0] is the size of the input buffer
	// addr[1] should be the size of input read, set it:
	cpm.CPU.Memory.Set(addr+1, uint8(len(text)))

	// addr[2+] should be the text
	i := 0
	for i < len(text) {
		cpm.CPU.Memory.Set(uint16(addr+2+uint16(i)), text[i])
		i++
	}

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallConsoleStatus tests if we have pending console (character) input.
func SysCallConsoleStatus(cpm *CPM) error {

	// Nothing pending
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallBDOSVersion returns version details
func SysCallBDOSVersion(cpm *CPM) error {

	// HL = 0x0022 -CP/M 2.2
	// B = 0x00
	// A = 0x22
	cpm.CPU.States.AF.Hi = 0x22
	cpm.CPU.States.AF.Lo = 0x00
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x22
	cpm.CPU.States.BC.Hi = 0x00

	return nil
}

// SysCallSetDMA updates the address of the DMA area, which is used for block I/O.
func SysCallSetDMA(cpm *CPM) error {

	// Get the address from BC
	addr := cpm.CPU.States.DE.U16()

	// Update the DMA value.
	cpm.dma = addr

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallDriveAllReset resets the drives.
//
// If there is a file named "$..." then we need to return 0xFF in A,
// which will be read by the CCP - as created by SUBMIT.COM
func SysCallDriveAllReset(cpm *CPM) error {

	// Reset disk and user-number
	cpm.currentDrive = 0
	cpm.userNumber = 0

	// Default return value
	var ret uint8 = 0

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'

	// Should we remap drives?
	path := "."
	if cpm.Drives {
		path = string(drive)
	}

	// Look for a file with $ in its name
	files, err := os.ReadDir(path)
	if err == nil {
		for _, n := range files {
			if strings.Contains(n.Name(), "$") {
				ret = 0xff
			}
		}
	}

	// Reset our DMA address to the default
	cpm.dma = 0x80

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = ret
	return nil
}

// SysCallDriveSet updates the current drive number.
func SysCallDriveSet(cpm *CPM) error {

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
	cpm.Memory.Set(0x0004, cpm.currentDrive)

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallFileOpen opens the filename that matches the pattern on the FCB supplied in DE
func SysCallFileOpen(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the parts of the name
	name := fcbPtr.GetName()
	ext := fcbPtr.GetType()

	// Get the actual name
	fileName := name
	if ext != "" && ext != "   " {
		fileName += "."
		fileName += ext
	}

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Should we remap drives?
	path := "."
	if cpm.Drives {
		path = string(drive)
	}

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

	// child logger with more details.
	l := cpm.Logger.With(
		slog.String("function", "SysCallFileOpen"),
		slog.String("name", name),
		slog.String("ext", ext),
		slog.String("drive", string(cpm.currentDrive+'A')),
		slog.String("result", fileName))

	// Should we remap drives?
	if cpm.Drives {
		before := fileName

		fileName = filepath.Join(string(drive), fileName)

		l.Debug("SysCallFileOpen remapped path",
			slog.String("before", before),
			slog.String("after", fileName))
	}

	// Now we open..
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {

		// We might fail to open a file because it doesn't
		// exist.
		if os.IsNotExist(err) {

			l.Debug("failed to open, file does not exist",
				slog.String("path", fileName),
				slog.String("error", err.Error()))

			cpm.CPU.States.AF.Hi = 0xFF
			return nil
		}

		// Ok a different error
		l.Debug("failed to open",
			slog.String("path", fileName),
			slog.String("error", err.Error()))
		return err
	}

	// Save the file handle in our cache.
	cpm.files[ptr] = FileCache{name: fileName, handle: file}

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
	if fLen > maxRC {
		fcbPtr.RC = maxRC
	} else {
		fcbPtr.RC = fLen
	}

	l.Debug("result:OK",
		slog.Int("fcb", int(ptr)),
		slog.Int("handle", int(file.Fd())),
		slog.Int("record_count", int(fcbPtr.RC)),
		slog.Int64("file_size", fileSize))

	// Update the FCB in memory.
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// Return success
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallFileClose closes the filename that matches the pattern on the FCB supplied in DE
func SysCallFileClose(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the file handle from our cache.
	obj, ok := cpm.files[ptr]
	if !ok {
		return fmt.Errorf("tried to close a file that wasn't open")
	}

	err := obj.handle.Close()
	if err != nil {
		return fmt.Errorf("failed to close file %04X:%s", ptr, err)
	}
	delete(cpm.files, ptr)

	// Record success
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallFindFirst finds the first filename, on disk, that matches the glob in the FCB supplied in DE.
func SysCallFindFirst(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Previous results are now invalidated
	cpm.findFirstResults = []fcb.FCBFind{}
	cpm.findOffset = 0

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	dir := "."
	if cpm.Drives {
		dir = string(cpm.currentDrive + 'A')
	}

	// Find files in the FCB.
	res, err := fcbPtr.GetMatches(dir)
	if err != nil {
		cpm.Logger.Debug("fcbPtr.GetMatches returned error",
			slog.String("path", dir),
			slog.String("error", err.Error()))

		cpm.CPU.States.AF.Hi = 0xff
		return nil
	}

	// No matches?  Return an error
	if len(res) < 1 {
		cpm.CPU.States.AF.Hi = 0xff
		return nil
	}

	// Here we save the results in our cache,
	// dropping the first
	cpm.findFirstResults = res[1:]
	cpm.findOffset = 0

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
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallFindNext finds the next filename that matches the glob set in the FCB in DE.
func SysCallFindNext(cpm *CPM) error {
	//
	// Assume we've been called with findFirst before
	//
	if (len(cpm.findFirstResults) == 0) || cpm.findOffset >= len(cpm.findFirstResults) {
		// Return 0xFF to signal an error
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	res := cpm.findFirstResults[cpm.findOffset]
	cpm.findOffset++

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
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallDeleteFile deletes the filename(s) matching the pattern specified by the FCB in DE.
func SysCallDeleteFile(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	path := "."
	if cpm.Drives {
		path = string(drive)
	}

	// Find files in the FCB.
	res, err := fcbPtr.GetMatches(path)
	if err != nil {
		cpm.Logger.Debug("fcbPtr.GetMatches returned error",
			slog.String("path", path),
			slog.String("error", err.Error()))

		cpm.CPU.States.AF.Hi = 0xff
		return nil
	}

	// No matches on the glob-search
	if len(res) == 0 {
		// Return 0xFF for failure
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// For each result
	for _, entry := range res {

		// Host path
		path := entry.Host

		cpm.Logger.Debug("SysCallDeleteFile: deleting file",
			slog.String("path", path))

		err = os.Remove(path)
		if err != nil {

			cpm.Logger.Debug("SysCallDeleteFile: failed to delete file",
				slog.String("path", path),
				slog.String("error", err.Error()))

			cpm.CPU.States.AF.Hi = 0xff
			return nil
		}
	}

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00
	return err
}

// SysCallRead reads a record from the file named in the FCB given in DE
func SysCallRead(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the file handle in our cache.
	obj, ok := cpm.files[ptr]
	if !ok {
		cpm.Logger.Error("SysCallRead: Attempting to read from a file that isn't open")
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Get the next read position
	offset := fcbPtr.GetSequentialOffset()

	_, err := obj.handle.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot seek to position %d: %s", offset, err)
	}

	// Temporary area to read into
	data := make([]byte, blkSize)

	// Fill the area with data
	for i := range data {
		data[i] = 0x1a
	}

	// Read from the file, now we're in the right place
	_, err = obj.handle.Read(data)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading file %s", err)
	}

	// Add logging of the result and details.
	cpm.Logger.Debug("SysCallRead",
		slog.Int("dma", int(cpm.dma)),
		slog.Int("fcb", int(ptr)),
		slog.Int("handle", int(obj.handle.Fd())),
		slog.Int("offset", int(offset)))

	// Copy the data to the DMA area
	cpm.Memory.SetRange(cpm.dma, data...)

	// Update the next read position
	fcbPtr.IncreaseSequentialOffset()

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// All done
	if err == io.EOF {
		cpm.CPU.States.AF.Hi = 0x01
	} else {
		cpm.CPU.States.AF.Hi = 0x00
	}

	return nil
}

// SysCallWrite writes a record to the file named in the FCB given in DE
func SysCallWrite(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the file handle in our cache.
	obj, ok := cpm.files[ptr]
	if !ok {
		cpm.Logger.Error("SysCallWrite: Attempting to write to a file that isn't open")
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Get the next write position
	offset := fcbPtr.GetSequentialOffset()

	// Add logging of the result and details.
	cpm.Logger.Debug("SysCallWrite",
		slog.Int("dma", int(cpm.dma)),
		slog.Int("fcb", int(ptr)),
		slog.Int("handle", int(obj.handle.Fd())),
		slog.Int("offset", int(offset)))

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

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	// All done
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallMakeFile creates the file named in the FCB given in DE
func SysCallMakeFile(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the name
	name := fcbPtr.GetName()
	ext := fcbPtr.GetType()

	fileName := name
	if ext != "" && ext != "   " {
		fileName += "."
		fileName += ext
	}

	// drive will default to our current drive, if the FCB drive field is 0
	drive := cpm.currentDrive + 'A'
	if fcbPtr.Drive != 0 {
		drive = fcbPtr.Drive + 'A' - 1
	}

	// Should we remap drives?
	path := "."
	if cpm.Drives {
		path = string(drive)
	}

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

	// child logger with more details.
	l := cpm.Logger.With(
		slog.String("function", "SysCallMakeFile"),
		slog.String("name", name),
		slog.String("ext", ext),
		slog.String("drive", string(cpm.currentDrive+'A')),
		slog.String("result", fileName))

	// Should we remap drives?
	if cpm.Drives {
		before := fileName

		fileName = filepath.Join(string(drive), fileName)

		l.Debug("SysCallMakeFile remapped path",
			slog.String("before", before),
			slog.String("after", fileName))
	}

	// Create the file
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {

		l.Debug("failed to open",
			slog.String("path", fileName),
			slog.String("error", err.Error()))
		return err
	}

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
	if fLen > maxRC {
		fcbPtr.RC = maxRC
	} else {
		fcbPtr.RC = fLen
	}

	// Save the file-handle
	cpm.files[ptr] = FileCache{name: fileName, handle: file}

	l.Debug("result:OK",
		slog.Int("fcb", int(ptr)),
		slog.Int("handle", int(file.Fd())),
		slog.Int("record_count", int(fcbPtr.RC)),
		slog.Int64("file_size", fileSize))

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)

	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallRenameFile will handle a rename operation.
// Note that this will not handle cross-directory renames (i.e. file moving).
func SysCallRenameFile(cpm *CPM) error {

	// 1. SRC

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the name
	name := fcbPtr.GetName()
	ext := fcbPtr.GetType()

	fileName := name
	if ext != "" && ext != "   " {
		fileName += "."
		fileName += ext
	}

	// Should we remap drives?
	path := "."
	if cpm.Drives {
		path = string(cpm.currentDrive + 'A')
	}

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

	// Should we remap drives?
	if cpm.Drives {
		fileName = filepath.Join(string(cpm.currentDrive+'A'), fileName)
	}

	// 2. DEST
	// The pointer to the FCB
	xxx2 := cpm.Memory.GetRange(ptr+16, fcb.SIZE)

	// Create a structure with the contents
	dstPtr := fcb.FromBytes(xxx2)

	// Get the name
	dName := dstPtr.GetName()
	dExt := dstPtr.GetType()

	dstName := dName
	if dExt != "" && dExt != "   " {
		dstName += "."
		dstName += dExt
	}

	// Should we remap drives?
	if cpm.Drives {
		dstName = filepath.Join(string(cpm.currentDrive+'A'), dstName)
	}

	cpm.Logger.Debug("Renaming file",
		slog.String("src", fileName),
		slog.String("dst", dstName))

	err := os.Rename(fileName, dstName)
	if err != nil {
		cpm.Logger.Debug("Renaming file failed",
			slog.String("error", err.Error()))
		cpm.CPU.States.AF.Hi = 0xFF

		return nil
	}

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallLoginVec returns the list of logged in drives.
func SysCallLoginVec(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0xFF
	cpm.CPU.States.HL.Lo = 0xFF
	return nil
}

// SysCallDriveGet returns the number of the active drive.
func SysCallDriveGet(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = cpm.currentDrive

	return nil
}

// SysCallSetFileAttributes should update the attributes of the given
// file, but it fakes it.
func SysCallSetFileAttributes(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallGetDriveDPB returns the address of the DPB, which is faked.
func SysCallGetDriveDPB(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0xCD
	cpm.CPU.States.HL.Lo = 0xCD
	return nil
}

// SysCallUserNumber gets, or sets, the user-number.
func SysCallUserNumber(cpm *CPM) error {

	// We're either setting or getting
	//
	// If the value is 0xFF we return it, otherwise we set
	if cpm.CPU.States.DE.Lo != 0xFF {

		// Set the number - masked, because valid values are 0-15
		cpm.userNumber = (cpm.CPU.States.DE.Lo & 0x0F)
	}

	// Return values:
	// HL = user, B=0, A=user
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = cpm.userNumber
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = cpm.userNumber
	cpm.CPU.States.AF.Lo = 0x00

	return nil
}

// SysCallReadRand reads a random block from the FCB pointed to by DE into the DMA area.
func SysCallReadRand(cpm *CPM) error {
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
			fmt.Printf("ReadRand:failed to get file size of: %s", err)
		}
		fileSize := fi.Size()

		// If the offset we're reading from is bigger than the file size then
		// pad it up
		if offset > fileSize {
			return 06
		}

		_, err = f.Seek(offset, io.SeekStart)
		if err != nil {
			fmt.Printf("cannot seek to position %d: %s", offset, err)
			return 0xFF
		}

		for i := range data {
			data[i] = 0x1a
		}

		_, err = f.Read(data)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("failed to read offset %d: %s", offset, err)
				return 0xFF
			}
		}

		cpm.Memory.SetRange(cpm.dma, data...)
		return 0
	}

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the file handle in our cache.
	obj, ok := cpm.files[ptr]
	if !ok {
		cpm.Logger.Error("SysCallReadRand: Attempting to read from a file that isn't open")
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Get the record to read
	record := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)

	// Translate the record to a byte-offset
	fpos := int64(record) * blkSize

	// Read the data
	res := sysRead(obj.handle, fpos)

	// Add logging of the result and details.
	cpm.Logger.Debug("SysCallReadRand",
		slog.Int("dma", int(cpm.dma)),
		slog.Int("fcb", int(ptr)),
		slog.Int("handle", int(obj.handle.Fd())),
		slog.Int("record_count", int(fcbPtr.RC)),
		slog.Int("record", record),
		slog.Int64("fpos", fpos),
		slog.Int("result", res))

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)
	cpm.CPU.States.AF.Hi = uint8(res)
	return nil
}

// SysCallWriteRand writes a random block from DMA area to the FCB pointed to by DE.
func SysCallWriteRand(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the file handle in our cache.
	obj, ok := cpm.files[ptr]
	if !ok {
		cpm.Logger.Error("SysCallWriteRand: Attempting to write to a file that isn't open")
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Get the data range from the DMA area
	data := cpm.Memory.GetRange(cpm.dma, 128)

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

	// Add logging of the result and details.
	cpm.Logger.Debug("SysCallWriteRand",
		slog.Int("dma", int(cpm.dma)),
		slog.Int("fcb", int(ptr)),
		slog.Int("padding", int(padding)),
		slog.Int("handle", int(obj.handle.Fd())),
		slog.Int("record_count", int(fcbPtr.RC)),
		slog.Int("record", record),
		slog.Int64("fpos", fpos))

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

	fcbPtr.IncreaseSequentialOffset()

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallFileSize updates the Random Record bytes of the given FCB to the
// number of records in the file.
//
// Returns the result in the A record
func SysCallFileSize(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, fcb.SIZE)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	//
	// Seems this doesn't require a file to be open.
	//
	// So we have to go through the dance of getting the filename.
	//

	// Get the name
	name := fcbPtr.GetName()
	ext := fcbPtr.GetType()

	fileName := name
	if ext != "" && ext != "   " {
		fileName += "."
		fileName += ext
	}

	// Should we remap drives?
	path := "."
	if cpm.Drives {
		path = string(cpm.currentDrive + 'A')
	}

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

	// Should we remap drives?
	if cpm.Drives {
		fileName = filepath.Join(string(cpm.currentDrive+'A'), fileName)
	}

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

	// Now we have the size we need to turn it into the number
	// of records
	records := int(fi.Size() / 128)

	// Store the value in the three fields
	fcbPtr.R0 = uint8(records & 0xff)
	fcbPtr.R1 = uint8(records & 0xff00)
	fcbPtr.R2 = uint8(records & 0xff0000)

	// sanity check because I've messed this up in the past
	n := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)
	if n != records {
		return fmt.Errorf("failed to update because maths is hard")
	}

	// Update the FCB in memory
	cpm.Memory.SetRange(ptr, fcbPtr.AsBytes()...)
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallDriveAlloc will return the address of the allocation bitmap (which blocks are used and
// which are free) in HL.
//
// TODO: Fake me better.  Right now I just return "random memory".
func SysCallDriveAlloc(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	return nil
}

// SysCallDriveSetRO will mark the current drive as being read-only.
//
// This call is faked.
func SysCallDriveSetRO(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallDriveROVec will return a bitfield describing which drives are read-only.
//
// Bit 7 of H corresponds to P: while bit 0 of L corresponds to A:. A bit is set if the corresponding drive is
// set to read-only in software.  As we never set drives to read-only we return 0x0000
func SysCallDriveROVec(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	return nil
}
