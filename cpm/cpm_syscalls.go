// This file contains the implementations for the CP/M calls we emulate.
//
// NOTE: They are added to the syscalls map in cpm.go
//

package cpm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/skx/cpmulator/fcb"
	"golang.org/x/term"
)

// blkSize is the size of block-based I/O operations
const blkSize = 128

// maxRC is the maximum read count
const maxRC = 128

// dma holds the default DMA address
const dma = 0x80

const MaxS2 = 15

// SysCallExit implements the Exit syscall
func SysCallExit(cpm *CPM) error {
	return ErrExit
}

// SysCallReadChar reads a single character from the console.
func SysCallReadChar(cpm *CPM) error {
	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("error making raw terminal %s", err)
	}

	// read only a single byte
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return fmt.Errorf("error reading a byte from stdin %s", err)
	}

	// restore the state of the terminal to avoid mixing RAW/Cooked
	err = term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		return fmt.Errorf("error restoring terminal state %s", err)
	}

	// Return the character
	cpm.CPU.States.AF.Hi = b[0]

	return nil
}

// SysCallWriteChar writes the single character in the A register to STDOUT.
func SysCallWriteChar(cpm *CPM) error {
	fmt.Printf("%c", (cpm.CPU.States.DE.Lo))
	return nil
}

func SysCallRawIO(cpm *CPM) error {
	if cpm.CPU.States.DE.Lo != 0xff {
		fmt.Printf("%c", cpm.CPU.States.DE.Lo)
	} else {
		return SysCallReadChar(cpm)
	}

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
	return nil
}

// SysCallReadString reads a string from the console, into the buffer pointed to by DE.
func SysCallReadString(cpm *CPM) error {
	addr := cpm.CPU.States.DE.U16()

	text, err := cpm.Reader.ReadString('\n')
	if err != nil {
		return (fmt.Errorf("error reading from STDIN:%s", err))
	}

	// remove trailing newline
	text = strings.TrimSuffix(text, "\n")

	// addr[0] is the size of the input buffer
	// addr[1] should be the size of input read, set it:
	cpm.CPU.Memory.Set(addr+1, uint8(len(text)))

	// addr[2+] should be the text
	i := 0
	for i < len(text) {
		cpm.CPU.Memory.Set(uint16(addr+2+uint16(i)), text[i])
		i++
	}

	return nil
}

// SysCallDriveAllReset resets the drives
func SysCallDriveAllReset(cpm *CPM) error {
	if cpm.fileIsOpen {
		cpm.fileIsOpen = false
		cpm.file.Close()
	}

	cpm.currentDrive = 1
	cpm.userNumber = 0

	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallDriveSet updates the current drive number
func SysCallDriveSet(cpm *CPM) error {
	// The drive number passed to this routine is 0 for A:, 1 for B: up to 15 for P:.
	cpm.currentDrive = (cpm.CPU.States.AF.Hi & 0x0F)

	// Success means we return 0x00 in A
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallFileOpen opens the filename that matches the pattern on the FCB supplied in DE
func SysCallFileOpen(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

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

	// Now we open..
	var err error
	cpm.file, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	// OK so we've opened a file, and we've cached the
	// handle in our CPM struct.  Record it
	cpm.fileIsOpen = true

	// No error, so we can proceed with the rest of the
	// steps.
	fcbPtr.S1 = 0x00
	fcbPtr.S2 |= 0x80 // not modified
	fcbPtr.RC = 0x00

	// Get file size, in bytes
	fi, err := cpm.file.Stat()
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

	// Update the FCB in memory.
	cpm.Memory.PutRange(ptr, fcbPtr.AsBytes()...)

	// Return success
	cpm.CPU.States.AF.Hi = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	return nil
}

// SysCallFileClose closes the filename that matches the pattern on the FCB supplied in DE
func SysCallFileClose(cpm *CPM) error {

	// Close the handle, if we have one
	if cpm.fileIsOpen {
		cpm.fileIsOpen = false
		cpm.file.Close()
	}

	// Record success
	cpm.CPU.States.AF.Hi = 0x00
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	return nil
}

// SysCallFindFirst finds the first filename, on disk, that matches the glob in the FCB supplied in DE.
func SysCallFindFirst(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

	// Previous results are now invalidated
	cpm.findFirstResults = []string{}
	cpm.findOffset = 0

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	pattern := ""
	name := fcbPtr.GetName()
	ext := fcbPtr.GetType()

	for _, c := range name {
		if c == '?' {
			pattern += "*"
			break
		}
		if c == ' ' {
			continue
		}
		pattern += string(c)
	}
	if ext != "" && ext != "   " {
		pattern += "."
	}

	for _, c := range ext {
		if c == '?' {
			pattern += "*"
			break
		}
		if c == ' ' {
			continue
		}
		pattern += string(c)
	}

	// Run the glob.
	matches, err := filepath.Glob(pattern)
	if err != nil {
		// error in pattern?
		fmt.Printf("glob error %s\n", err)
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// No matches on the glob-search
	if len(matches) == 0 {
		// Return 0xFF for failure
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Here we save the results in our cache,
	// dropping the first
	cpm.findFirstResults = matches[1:]
	cpm.findOffset = 0

	// Create a new FCB and store it in the DMA entry
	x := fcb.FromString(matches[0])
	data := x.AsBytes()
	cpm.Memory.PutRange(dma, data...)

	// Return 0x00 to point to the first entry in the DMA area.
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
	x := fcb.FromString(res)
	data := x.AsBytes()
	cpm.Memory.PutRange(0x80, data...)

	// Return 0x00 to point to the first entry in the DMA area.
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallDeleteFile deletes the filename specified by the FCB in DE.
func SysCallDeleteFile(cpm *CPM) error {

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

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

	// Delete the named file
	err := os.Remove(fileName)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// SysCallWrite writes a record to the file named in the FCB given in DE
func SysCallWrite(cpm *CPM) error {

	// Don't have a file open?  That's a bug
	if !cpm.fileIsOpen {
		cpm.Logger.Error("attempting to write to a file that isn't open")
		cpm.CPU.States.AF.Hi = 0xff
		return nil
	}

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	// Get the data range from the DMA area
	data := cpm.Memory.GetRange(dma, 128)

	// offset
	BlkS2 := 4096
	BlkEx := 128
	offset := int(int(fcbPtr.S2)&MaxS2)*BlkS2*blkSize +
		int(fcbPtr.Ex)*BlkEx*blkSize +
		int(fcbPtr.Cr)*blkSize

	_, err := cpm.file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot seek to position %d: %s", offset, err)
	}

	// Write to the open file
	_, err = cpm.file.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file %s", err)
	}

	MaxCR := 128
	MaxEX := 31

	fcbPtr.S2 &= 0x7F // reset unmodified flag
	fcbPtr.Cr++
	if int(fcbPtr.Cr) > MaxCR {
		fcbPtr.Cr = 1
		fcbPtr.Ex++
	}
	if int(fcbPtr.Ex) > MaxEX {
		fcbPtr.Ex = 0
		fcbPtr.S2++
	}
	fcbPtr.RC++

	// Update the FCB in memory
	cpm.Memory.PutRange(ptr, fcbPtr.AsBytes()...)

	// All done
	cpm.CPU.States.AF.Hi = 0x00
	return nil

}

// SysCallMakeFile creates the file named in the FCB given in DE
func SysCallMakeFile(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

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

	// Create the file
	var err error
	cpm.file, err = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	cpm.fileIsOpen = true
	return nil
}

// SysCallDriveGet returns the number of the active drive.
func SysCallDriveGet(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = cpm.currentDrive

	return nil
}

// SysCallGetDriveDPB returns the address of the DPB, which is faked.
func SysCallGetDriveDPB(cpm *CPM) error {
	cpm.CPU.States.HL.Hi = 0xCD
	cpm.CPU.States.HL.Lo = 0xCD
	return nil
}

func SysCallUserNumber(cpm *CPM) error {

	// We're either setting or getting
	//
	// If the value is 0xFF we return it, otherwise we set
	if cpm.CPU.States.DE.Lo != 0xFF {

		// Set the number - masked, because valid values are 0-15
		cpm.userNumber = (cpm.CPU.States.DE.Lo & 0x0F)
	}

	// Return the current number, which might have changed
	cpm.CPU.States.AF.Hi = cpm.userNumber
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
	sysRead := func(offset int64) int {
		_, err := cpm.file.Seek(offset, io.SeekStart)
		if err != nil {
			fmt.Printf("cannot seek to position %d: %s", offset, err)
			return 1
		}

		for i := range data {
			data[i] = 0x1a
		}

		_, err = cpm.file.Read(data)
		if err != nil {
			fmt.Printf("failed to read offset %d: %s", offset, err)
			return 1
		}

		cpm.Memory.PutRange(0x80, data[:]...)
		return 0
	}

	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()

	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	if !cpm.fileIsOpen {
		return fmt.Errorf("ReadRand called against a non-open file")
	}

	// Get the record to read
	record := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)

	if record > 65535 {
		cpm.CPU.States.AF.Hi = 0x06
		//06	seek Past Physical end of disk
		return nil
	}

	fpos := int64(record) * blkSize

	res := sysRead(fpos)

	// Update the FCB in memory
	cpm.Memory.PutRange(ptr, fcbPtr.AsBytes()...)
	cpm.CPU.States.AF.Hi = uint8(res)
	return nil
}
