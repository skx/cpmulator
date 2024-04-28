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
	obj := cpmio.New()

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

// SysCallWriteChar writes the single character in the E register to STDOUT.
func SysCallWriteChar(cpm *CPM) error {

	// auxIO is set when we see A_READ/A_WRITE
	//
	// Mixing I/O modes is not recommended.
	if cpm.auxIO {
		return nil
	}

	cpm.outC(cpm.CPU.States.DE.Lo)

	return nil
}

// SysCallAuxRead reads a single character from the auxillary input.
//
// NOTE: Documentation implies this is blocking, but it seems like
// tastybasic and mbasic prefer it like this
func SysCallAuxRead(cpm *CPM) error {

	// Now we're using aux I/O
	cpm.auxIO = true

	// Use our I/O package
	obj := cpmio.New()

	// Is something waiting for us?
	p, err := obj.IsPending()
	if err != nil {
		return fmt.Errorf("error calling IsPending:%s", err)
	}

	// If yes, return it
	if p {
		c := obj.GetAvailableChar()

		cpm.CPU.States.HL.Hi = 0x00
		cpm.CPU.States.HL.Lo = c
		cpm.CPU.States.AF.Hi = c
		cpm.CPU.States.AF.Lo = 0x00
		return nil
	}

	// Return nothing
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallAuxWrite writes the single character in the C register auxillary / punch output
func SysCallAuxWrite(cpm *CPM) error {

	// Now we're using aux I/O
	cpm.auxIO = true

	// The character we're going to write
	c := cpm.CPU.States.BC.Lo
	cpm.outC(c)
	return nil
}

// outC attempts to write a single character output, but converting to ANSI from vt.
// This means tracking state and handling multi-byte output properly.
//
// This is all a bit sleazy.
func (cpm *CPM) outC(c uint8) {
	switch cpm.auxStatus {
	case 0:
		switch c {
		case 0x07: /* BEL: flash screen */
			fmt.Printf("\033[?5h\033[?5l")
		case 0x7f: /* DEL: echo BS, space, BS */
			fmt.Printf("\b \b")
		case 0x1a: /* adm3a clear screen */
			fmt.Printf("\033[H\033[2J")
		case 0x0c: /* vt52 clear screen */
			fmt.Printf("\033[H\033[2J")
		case 0x1e: /* adm3a cursor home */
			fmt.Printf("\033[H")
		case 0x1b:
			cpm.auxStatus = 1 /* esc-prefix */
		case 1:
			cpm.auxStatus = 2 /* cursor motion prefix */
		case 2: /* insert line */
			fmt.Printf("\033[L")
		case 3: /* delete line */
			fmt.Printf("\033[M")
		case 0x18, 5: /* clear to eol */
			fmt.Printf("\033[K")
		case 0x12, 0x13:
			// nop
		default:
			fmt.Printf("%c", c)
		}
	case 1: /* we had an esc-prefix */
		switch c {
		case 0x1b:
			fmt.Printf("%c", c)
		case '=', 'Y':
			cpm.auxStatus = 2
		case 'E': /* insert line */
			fmt.Printf("\033[L")
		case 'R': /* delete line */
			fmt.Printf("\033[M")
		case 'B': /* enable attribute */
			cpm.auxStatus = 4
		case 'C': /* disable attribute */
			cpm.auxStatus = 5
		case 'L', 'D': /* set line */ /* delete line */
			cpm.auxStatus = 6
		case '*', ' ': /* set pixel */ /* clear pixel */
			cpm.auxStatus = 8
		default: /* some true ANSI sequence? */
			cpm.auxStatus = 0
			fmt.Printf("%c%c", 0x1b, c)
		}
	case 2:
		cpm.y = c - ' ' + 1
		cpm.auxStatus = 3
	case 3:
		cpm.x = c - ' ' + 1
		cpm.auxStatus = 0
		fmt.Printf("\033[%d;%dH", cpm.y, cpm.x)
	case 4: /* <ESC>+B prefix */
		cpm.auxStatus = 0
		switch c {
		case '0': /* start reverse video */
			fmt.Printf("\033[7m")
		case '1': /* start half intensity */
			fmt.Printf("\033[1m")
		case '2': /* start blinking */
			fmt.Printf("\033[5m")
		case '3': /* start underlining */
			fmt.Printf("\033[4m")
		case '4': /* cursor on */
			fmt.Printf("\033[?25h")
		case '5': /* video mode on */
			// nop
		case '6': /* remember cursor position */
			fmt.Printf("\033[s")
		case '7': /* preserve status line */
			// nop
		default:
			fmt.Printf("%cB%c", 0x1b, c)
		}
	case 5: /* <ESC>+C prefix */
		cpm.auxStatus = 0
		switch c {
		case '0': /* stop reverse video */
			fmt.Printf("\033[27m")
		case '1': /* stop half intensity */
			fmt.Printf("\033[m")
		case '2': /* stop blinking */
			fmt.Printf("\033[25m")
		case '3': /* stop underlining */
			fmt.Printf("\033[24m")
		case '4': /* cursor off */
			fmt.Printf("\033[?25l")
		case '6': /* restore cursor position */
			fmt.Printf("\033[u")
		case '5': /* video mode off */
			// nop
		case '7': /* don't preserve status line */
			// nop
		default:
			fmt.Printf("%cC%c", 0x1b, c)
		}
		/* set/clear line/point */
	case 6:
		cpm.auxStatus++
	case 7:
		cpm.auxStatus++
	case 8:
		cpm.auxStatus++
	case 9:
		cpm.auxStatus = 0
	}

}

// SysCallRawIO handles both simple character output, and input.
func SysCallRawIO(cpm *CPM) error {

	// Blocking input by default
	block := true

	// Set $NON_BLOCK to change it
	if nb := os.Getenv("NON_BLOCK"); nb != "" {
		block = false
	}

	// Use our I/O package
	obj := cpmio.New()

	switch cpm.CPU.States.DE.Lo {
	case 0xFF:
		// Blocking input
		if block {
			cpm.CPU.States.AF.Hi, _ = obj.BlockForCharacter()
			return nil
		}

		// non-blocking, but CPU-heavy
		p, err := obj.IsPending()
		if err != nil {
			return err
		}
		if p {
			cpm.CPU.States.AF.Hi = obj.GetAvailableChar()
			return nil
		}
		cpm.CPU.States.AF.Hi = 0x00
		return nil
	case 0xFE:
		p, err := obj.IsPending()
		if err != nil {
			return err
		}
		if p {
			cpm.CPU.States.AF.Hi = 0xff
			return nil
		}
		cpm.CPU.States.AF.Hi = 0x00
		return nil
	case 0xFD:
		var err error
		cpm.CPU.States.AF.Hi, err = obj.BlockForCharacter()
		if err != nil {
			return err
		}
		return nil
	default:
		fmt.Printf("%c", 0x7f&cpm.CPU.States.DE.Lo)
	}
	return nil
}

// SysCallGetIOByte gets the IOByte, which is used to describe which devices
// are used for I/O.  No CP/M utilities use it, except for STAT and PIP.
//
// The IOByte lives at 0x0003 in RAM, so it is often accessed directly when it is used.
func SysCallGetIOByte(cpm *CPM) error {

	// Get the value
	c := cpm.Memory.Get(0x003)

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

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
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

// SysCallConsoleStatus fakes a test for pending console (character) input.
func SysCallConsoleStatus(cpm *CPM) error {

	// Use our I/O package
	obj := cpmio.New()

	// Is something waiting for us?
	p, err := obj.IsPending()
	if err != nil {
		return fmt.Errorf("error calling IsPending:%s", err)
	}

	if p {
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	// Nothing pending
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallDriveAllReset resets the drives.
//
// TODO: If there is a file named "$..." then we need to return 0xFF in A,
// which will be read by the CCP - as created by SUBMIT.COM
func SysCallDriveAllReset(cpm *CPM) error {

	// Reset disk and user-number
	cpm.currentDrive = 0
	cpm.userNumber = 0

	// Reset our DMA address to the default
	cpm.dma = 0x80

	// Return values:
	// HL = 0, B=0, A=0
	cpm.CPU.States.HL.Hi = 0x00
	cpm.CPU.States.HL.Lo = 0x00
	cpm.CPU.States.BC.Hi = 0x00
	cpm.CPU.States.AF.Hi = 0x00
	return nil
}

// SysCallDriveSet updates the current drive number
func SysCallDriveSet(cpm *CPM) error {
	// The drive number passed to this routine is 0 for A:, 1 for B: up to 15 for P:.
	cpm.currentDrive = (cpm.CPU.States.AF.Hi & 0x0F)

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
		fileName = string(cpm.currentDrive+'A') + "/" + fileName
	}

	// child logger with more details.
	l := cpm.Logger.With(
		slog.String("function", "SysCallFileOpen"),
		slog.String("name", name),
		slog.String("ext", ext),
		slog.String("drive", string(cpm.currentDrive+'A')),
		slog.String("result", fileName))

	// Now we open..
	file, err := os.OpenFile(fileName, os.O_RDWR, 0644)
	if err != nil {

		// We might fail to open a file because it doesn't
		// exist.
		if os.IsNotExist(err) {

			l.Debug("failed to open, file does not exist")

			cpm.CPU.States.AF.Hi = 0xFF
			return nil
		}

		// Ok a different error
		l.Debug("failed to open", slog.String("error", err.Error()))
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
	cpm.findFirstResults = []string{}
	cpm.findOffset = 0

	// Create a structure with the contents
	fcbPtr := fcb.FromBytes(xxx)

	dir := "."
	if cpm.Drives {
		dir = string(cpm.currentDrive+'A') + "/"
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
	x := fcb.FromString(res[0])
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
	x := fcb.FromString(res)
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

	dir := "."
	if cpm.Drives {
		dir = string(cpm.currentDrive+'A') + "/"
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

	// No matches on the glob-search
	if len(res) == 0 {
		// Return 0xFF for failure
		cpm.CPU.States.AF.Hi = 0xFF
		return nil
	}

	for _, path := range res {
		if cpm.Drives {
			path = string(cpm.currentDrive+'A') + "/" + path
		}

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
		fileName = string(cpm.currentDrive+'A') + "/" + fileName
	}

	// child logger with more details.
	l := cpm.Logger.With(
		slog.String("function", "SysCallMakeFile"),
		slog.String("name", name),
		slog.String("ext", ext),
		slog.String("drive", string(cpm.currentDrive+'A')),
		slog.String("result", fileName))

	// Create the file
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {

		l.Debug("failed to open", slog.String("error", err.Error()))
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
		fileName = string(cpm.currentDrive+'A') + "/" + fileName
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
		dstName = string(cpm.currentDrive+'A') + "/" + dstName
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
