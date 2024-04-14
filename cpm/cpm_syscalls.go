// This file contains the implementations for the CP/M calls we emulate.
//
// NOTE: They are added to the syscalls map in cpm.go
//

package cpm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/skx/cpmulator/fcb"
	"golang.org/x/term"
)

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
	term.Restore(int(os.Stdin.Fd()), oldState)

	// Return the character
	cpm.CPU.States.AF.Hi = b[0]

	return nil
}

// SysCallWriteChar writes the single character in the A register to STDOUT.
func SysCallWriteChar(cpm *CPM) error {
	fmt.Printf("%c", (cpm.CPU.States.DE.Lo))
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

// SysCallDriveSet updates the current drive number
func SysCallDriveSet(cpm *CPM) error {
	// The drive number passed to this routine is 0 for A:, 1 for B: up to 15 for P:.
	cpm.currentDrive = (cpm.CPU.States.AF.Hi & 0x0F)

	// Success means we return 0x00 in A
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

// SysCallFindFirst finds the first filename, on disk, that matches the glob in the FCB supplied in DE.
func SysCallFindFirst(cpm *CPM) error {
	// The pointer to the FCB
	ptr := cpm.CPU.States.DE.U16()
	// Get the bytes which make up the FCB entry.
	xxx := cpm.Memory.GetRange(ptr, 36)

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
	cpm.Memory.PutRange(0x80, data...)

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
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	return nil
}

// SysCallDriveGet returns the number of the active drive.
func SysCallDriveGet(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = cpm.currentDrive

	return nil
}

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

// SysCallUnimplemented is a placeholder for functions we don't implement
func SysCallUnimplemented(cpm *CPM) error {
	return nil
}
