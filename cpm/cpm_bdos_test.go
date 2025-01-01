package cpm

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
	"github.com/skx/cpmulator/static"
)

// Flaky with our new implementation.
func TestConsoleInput(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("11.log"), WithInputDriver("term"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.IOTearDown()

	// ReadChar
	c.StuffText("s")
	err = BdosSysCallReadChar(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 's' {
		t.Fatalf("got the wrong input")
	}

	// AuxRead
	c.StuffText("k")
	err = BdosSysCallAuxRead(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 'k' {
		t.Fatalf("got the wrong input")
	}

	// RawIO
	c.StuffText("x")
	c.CPU.States.DE.Lo = 0xFF
	err = BdosSysCallRawIO(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 'x' {
		t.Fatalf("got the wrong input")
	}

	c.StuffText("x")
	c.CPU.States.DE.Lo = 0xFE
	err = BdosSysCallRawIO(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("got the wrong response")
	}

	c.StuffText("1")
	c.CPU.States.DE.Lo = 0xFD
	err = BdosSysCallRawIO(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != '1' {
		t.Fatalf("got the wrong response")
	}

	c.StuffText("1")
	c.CPU.States.DE.Lo = 42
	err = BdosSysCallRawIO(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}

	c.StuffText("1")
	err = BdosSysCallConsoleStatus(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("got the wrong response")
	}
}

func TestUnimplemented(t *testing.T) {
	// Create a new helper
	c, err := New(WithPrinterPath("12.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.IOTearDown()

	// Create a binary
	var file *os.File
	file, err = os.CreateTemp("", "tst-*.com")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Make a call to BDOS function 99 - unimplemented
	_, err = file.Write([]byte{0x0E, 0x63, 0xCD, 0x05, 0x00})

	if err != nil {
		t.Fatalf("failed to write program to temporary file")
	}

	// Close the file
	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file")
	}

	// Attempt to load the binary
	err = c.LoadBinary(file.Name())
	if err != nil {
		t.Fatalf("error loading a binary")
	}

	// Finally launch it
	c.simpleDebug = true
	err = c.Execute([]string{"foo", "bar", "baz"})
	if err == nil {
		t.Fatalf("expected an error, got none")
	}
	if err != ErrUnimplemented {
		t.Fatalf("got an error, but the wrong one: %v\n", err)
	}
}

// TestBoot  ensures that a "jmp 0x0000" ends the emulation
func TestBoot(t *testing.T) {
	// Create a new helper
	c, err := New(WithPrinterPath("13.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.IOTearDown()

	// Create a binary
	var file *os.File
	file, err = os.CreateTemp("", "tst-*.com")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// CALL 0x0000
	_, err = file.Write([]byte{0xCD, 0x00, 0x00})

	if err != nil {
		t.Fatalf("failed to write program to temporary file")
	}

	// Close the file
	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file")
	}

	// Attempt to load the binary
	err = c.LoadBinary(file.Name())
	if err != nil {
		t.Fatalf("error loading a binary")
	}

	// Finally launch it
	c.simpleDebug = true
	err = c.Execute([]string{"foo", "bar", "baz"})
	if err == nil {
		t.Fatalf("expected an error, got none")
	}
	if err != ErrBoot {
		t.Fatalf("got an error, but the wrong one: %v\n", err)
	}

}

// TestFind invokes FindFirst and FindNext
func TestFind(t *testing.T) {
	// Create a new helper
	c, err := New(WithPrinterPath("14.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	c.fixupRAM()
	c.SetDrives(false)
	c.SetStaticFilesystem(static.GetContent())
	defer c.IOTearDown()

	// Create a pattern in an FCB
	name := "*.GO"
	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 5

	// Save it into RAM
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	found := 0

	// Call FindFirst
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFindFirst(c)

	if err != nil {
		t.Fatalf("error calling find first:err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("error calling find first:A")
	}
	found++

	// Now we call findNext, until it fails
	for {
		c.CPU.States.DE.SetU16(0x0200)
		err = BdosSysCallFindNext(c)

		if err != nil {
			t.Fatalf("error calling find next:err")
		}
		if c.CPU.States.AF.Hi != 0x00 {
			break
		}
		found++

	}

	if found != 5 {
		t.Fatalf("found wrong number of files, got %d", found)
	}

	// Try again, this time looking at embedded resources
	// Create a pattern in an FCB
	name = "A:!*.COM"
	fcbPtr = fcb.FromString(name)

	// Save it into RAM
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	found = 0

	// Call FindFirst
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFindFirst(c)

	if err != nil {
		t.Fatalf("error calling find first:err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("error calling find first:A")
	}
	found++

	// Now we call findNext, until it fails
	for {
		c.CPU.States.DE.SetU16(0x0200)
		err = BdosSysCallFindNext(c)

		if err != nil {
			t.Fatalf("error calling find next:err")
		}
		if c.CPU.States.AF.Hi != 0x00 {
			break
		}
		found++

	}

	if found != 6 {
		t.Fatalf("found wrong number of embedded files, got %d", found)
	}

	//
	// Now try to find files that won't exist
	//
	name = "*.SKX"
	fcbPtr = fcb.FromString(name)

	// Save it into RAM
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call FindFirst
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFindFirst(c)

	if err != nil {
		t.Fatalf("error calling find first:err")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("Expected no matches, but got something else?")
	}

	//
	// Finally try to match files with a bogus
	// directory mapping
	//
	c.SetDrives(true)
	c.SetDrivePath("A", "//½§<¿¿\\z<zZ!2/fdsf/<fð¿¿fdsf\fdsf")
	c.SetDrivePath("B", "//½§<¿¿\\z<zZ!2/fdsf/<fð¿¿fdsf\fdsf")
	name = "B:*.CFM"
	fcbPtr = fcb.FromString(name)
	fcbPtr.Drive = 1
	// Save it into RAM
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call FindFirst
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFindFirst(c)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("error calling find first:A")
	}
}

// TestReadLine does a minimal test on STDIN reading, via the faked/stuffed
// contents.  We can't use this trick for single bytes though.
func TestReadLine(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("15.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Stuff some fake input
	c.input, _ = consolein.New("stty")
	c.StuffText("steve\n")

	// Setup a buffer, so we can read 5 characters
	c.Memory.Set(0x0100, 5)
	c.CPU.States.DE.SetU16(0x0100)

	// Read it
	err = BdosSysCallReadString(c)
	if err != nil {
		t.Fatalf("error reading CPM")
	}

	// How much did we get
	got := c.Memory.Get(0x0101)
	if got != 05 {
		t.Fatalf("returned wrong amount")
	}

	// What did we get?
	text := ""
	i := 0
	for i < int(got) {
		text += string(c.Memory.Get(uint16(0x0102 + i)))
		i++
	}

	if text != "steve" {
		t.Fatalf("wrong text received")
	}

	//
	// Ctrl-C should trigger a reboot, of course
	//
	c.CPU.DE.SetU16(0x0000)
	c.StuffText("\x03\x03foo\n")

	err = BdosSysCallReadString(c)
	if err != ErrBoot {
		t.Fatalf("expected reboot from Ctrl-C, got %v", err)
	}
}

// TestDriveGetSet tests getting/setting the current drive.
func TestDriveGetSet(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("16.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Get the drive, whatever it is
	err = BdosSysCallDriveGet(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}

	// Set a drive
	c.CPU.States.AF.Hi = 3
	err = BdosSysCallDriveSet(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	cur := c.CPU.States.AF.Hi

	// Get the (updated)
	err = BdosSysCallDriveGet(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 3 || c.CPU.States.AF.Hi == cur {
		t.Fatalf("setting the drive failed got %d", c.CPU.States.AF.Hi)
	}

	// Set a drive to a bogus value
	c.CPU.States.AF.Hi = 0xFF
	err = BdosSysCallDriveSet(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}

	// Get the (updated)
	err = BdosSysCallDriveGet(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 15 {
		t.Fatalf("setting the drive failed got %d - should have been P:", c.CPU.States.AF.Hi)
	}

}

// TestSetDMA tests that we can update the DMA address.
func TestSetDMA(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("16.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	if c.dma != 0x80 {
		t.Fatalf("bogus initial DMA")
	}

	c.CPU.States.DE.SetU16(0x1234)
	err = BdosSysCallSetDMA(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	if c.dma != 0x1234 {
		t.Fatalf("failed ot update dma")
	}
}

// TestUserNumber tests we can get/set the current user-number.
func TestUserNumber(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("17.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// set to user 5
	c.CPU.States.DE.Lo = 5
	err = BdosSysCallUserNumber(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	// get the user
	if c.userNumber != 5 {
		t.Fatalf("failed to set user number")
	}

	// now get properly
	c.CPU.States.DE.Lo = 0xFF
	err = BdosSysCallUserNumber(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 05 {
		t.Fatalf("retriving user number failed")
	}
}

// TestDriveReset tests that when a $-file is present the
// drive reset returns a different result, such that SUBMIT.COM
// works - well more specifically so the CCP recognizes the file
// that submit.com created.
func TestDriveReset(t *testing.T) {

	getState := func() uint8 {
		// Create a new helper
		c, err := New(WithPrinterPath("19.log"))
		if err != nil {
			t.Fatalf("failed to create CPM")
		}
		c.Memory = new(memory.Memory)

		c.SetDrives(false)

		err = BdosSysCallDriveAllReset(c)
		if err != nil {
			t.Fatalf("reset drive call failed")
		}
		return c.CPU.States.AF.Hi
	}

	if getState() != 0x00 {
		t.Fatalf("getState != 0")
	}

	// Create a file with "$" in it
	name := "NA$E.$$$"
	_, err := os.Create(name)
	if err != nil {
		t.Fatalf("failed to create $-file")
	}
	defer os.Remove(name)

	// Now reset again and we should see 0xFF to trigger
	// the submit.com behaviour
	if getState() != 0xFF {
		t.Fatalf("getState != 0xFF")
	}
}

// TestFileSize is incomplete because it doesn't handle
// virtual files - TODO
func TestFileSize(t *testing.T) {

	create := func(name string, size int) error {
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()

		d := []byte{0x00}
		for size > 0 {
			_, _ = f.Write(d)
			size--
		}
		return nil
	}

	// Create a new helper
	c, err := New(WithPrinterPath("20.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Call the function
	for _, sz := range []int{128, 256, 41088} {

		// Create a named file with the given size
		name := "TEST.TXT"
		err := create(name, sz)
		if err != nil {
			t.Fatalf("failed to create")
		}

		// Create an FCB
		fcbPtr := fcb.FromString(name)

		// Write the FCB into memory
		c.Memory.SetRange(0x000, fcbPtr.AsBytes()...)
		c.CPU.States.DE.Lo = 0x00
		c.CPU.States.DE.Hi = 0x00

		// Get the file-size
		err = BdosSysCallFileSize(c)

		if err != nil {
			t.Fatalf("failed to get file size: %s\n", err)
		}
		// Now the fcb size should be populated
		// Read it back
		xxx := c.Memory.GetRange(0x0000, fcb.SIZE)
		fcbPtr = fcb.FromBytes(xxx)

		n := int(int(fcbPtr.R2)<<16) | int(int(fcbPtr.R1)<<8) | int(fcbPtr.R0)
		if n*128 != sz {
			t.Fatalf("size was wrong expected %d, got %d", sz, n)
		}
		os.Remove(name)
	}

	// Get the file-size of a failed file
}

// TestIOByte tests the get/set of the IO byte
func TestIOByte(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("21.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// default is zero
	err = BdosSysCallGetIOByte(c)
	if err != nil {
		t.Fatalf("error in CPM call")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("unexpected initial IO byte")
	}

	// set it
	c.CPU.States.DE.Lo = 0xFE
	err = BdosSysCallSetIOByte(c)
	if err != nil {
		t.Fatalf("error in CPM call")
	}

	// get it
	err = BdosSysCallGetIOByte(c)
	if err != nil {
		t.Fatalf("error in CPM call")
	}

	if c.CPU.States.AF.Hi != 0xFE {
		t.Fatalf("unexpected updated IO byte")
	}
}

// TestBDOSCoverage adds calls to the trivial functions that
// don't really get implemented.  It just calls them to
// increase our coverage.
func TestBDOSCoverage(t *testing.T) {

	// Create a printer-output file
	file, err := os.CreateTemp("", "tst-*.prn")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Create a new helper - redirect the printer log because
	// we'll be invoking BdosSysCallPrinterWrite
	c, err := New(WithPrinterPath(file.Name()))

	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	for _, handler := range c.BDOSSyscalls {

		if handler.Fake {
			err = handler.Handler(c)
			if err != nil {
				t.Fatalf("error calling %s\n", handler.Desc)
			}
		}
	}
	err = BdosSysCallBDOSVersion(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallDirectScreenFunctions(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallDriveAlloc(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallDriveROVec(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallDriveReset(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallDriveSetRO(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallErrorMode(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallGetDriveDPB(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallLoginVec(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallSetFileAttributes(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallTime(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
}

// TestMakeCloseFile tests our file creation handler, and our close function too
func TestMakeFile(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("23.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	fileExists := func(path string) bool {
		if _, err2 := os.Stat(path); errors.Is(err2, os.ErrNotExist) {
			return false
		}
		return true
	}

	// Create an FCB pointing to a file
	name := "MAKE.ME"
	if fileExists(name) {
		t.Fatalf("file already exists")
	}

	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 5
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call the creation function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallMakeFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("creation failed")
	}
	if !fileExists(name) {
		t.Fatalf("failed to create file")
	}

	// Now we've created it the file will be open
	// Close it.
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileClose(c)
	if err != nil {
		t.Fatalf("failed to close file, after creation")
	}

	// Why not also try to close a file that is
	// not open?
	c.CPU.States.DE.SetU16(0xCDCD)
	err = BdosSysCallFileClose(c)
	if err != nil {
		t.Fatalf("failed to close file which wasn't open")
	}

	// Try to create a file from a null FCB
	// Here we're relying on the RAM being nulls
	c.CPU.States.DE.SetU16(0x1200)
	err = BdosSysCallMakeFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("expected error with empty file")
	}

	// Delete the file, if it was present
	os.Remove(name)
}

func TestCloseDollar(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	fileExists := func(path string) bool {
		if _, err2 := os.Stat(path); errors.Is(err2, os.ErrNotExist) {
			return false
		}
		return true
	}

	// Create an FCB pointing to a file
	name := "KEMP$.$ME"
	if fileExists(name) {
		t.Fatalf("file already exists")
	}

	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 5
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call the creation function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallMakeFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("creation failed")
	}
	if !fileExists(name) {
		t.Fatalf("failed to create file")
	}

	//
	// Now we've created the file we'll write to it.
	//
	// Fill the DMA area
	c.Memory.Set(c.dma+0, 'S')
	c.Memory.Set(c.dma+1, 't')
	c.Memory.Set(c.dma+2, 'e')
	c.Memory.Set(c.dma+3, 'v')
	c.Memory.Set(c.dma+4, 'e')
	c.Memory.Set(c.dma+5, 0x00)
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallWrite(c)
	if err != nil {
		t.Fatalf("got error writing to file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("got error writing to file: A")
	}

	// Change the file size - first get the updated size
	xxx := c.Memory.GetRange(0x0200, fcb.SIZE)
	fcbPtr = fcb.FromBytes(xxx)
	fcbPtr.RC = 0
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Now we've created it the file will be open
	// Close it.
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileClose(c)
	if err != nil {
		t.Fatalf("failed to close file, after creation")
	}

	// Delete the file, if it was present
	os.Remove(name)

}

// TestDelete tests we can delete a file.
func TestDelete(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("24.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	// Create a file
	name := "DELETE.ME"
	_, err = os.Create(name)
	if err != nil {
		t.Fatalf("failed to create $-file")
	}

	fileExists := func(path string) bool {
		if _, err2 := os.Stat(path); errors.Is(err2, os.ErrNotExist) {
			return false
		}
		return true
	}

	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 3
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call the deletion function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallDeleteFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("delete failed")
	}
	if fileExists(name) {
		t.Fatalf("failed to delete file")
	}

	// Delete the file, if it was present
	os.Remove(name)

}

// TestRename tests we can perform a simple rename.
func TestRename(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("25.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	// Create a file
	name := "BEFORE"
	_, err = os.Create(name)
	if err != nil {
		t.Fatalf("failed to create file")
	}

	fileExists := func(path string) bool {
		if _, err2 := os.Stat(path); errors.Is(err2, os.ErrNotExist) {
			return false
		}
		return true
	}

	// Src
	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 3
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Dst
	dstPtr := fcb.FromString("AFTER")
	dstPtr.Drive = 6
	c.Memory.SetRange(0x0200+16, dstPtr.AsBytes()...)

	// Call the rename function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallRenameFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("rename failed")
	}
	if fileExists("BEFORE") {
		t.Fatalf("file still exists")
	}
	if !fileExists("AFTER") {
		t.Fatalf("file rename didn't create it")
	}

	// Delete both files, if present.
	os.Remove("BEFORE")
	os.Remove("AFTER")

	// Try to rename to a file that can't work
	dstPtr = fcb.FromString("/.>/>dsd:")
	dstPtr.Drive = 6
	c.Memory.SetRange(0x0200+16, dstPtr.AsBytes()...)

	// Call the rename function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallRenameFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("renaming to an impossible name succeeded")
	}

}

// TestWriteFile tests writing a sequential record to an open file.
func TestWriteFile(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("26.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	fileExists := func(path string) bool {
		if _, err2 := os.Stat(path); errors.Is(err2, os.ErrNotExist) {
			return false
		}
		return true
	}

	// Create an FCB pointing to a file
	name := "WRITE.ME"
	if fileExists(name) {
		t.Fatalf("file already exists")
	}

	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 5
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call the creation function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallMakeFile(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("creation failed")
	}
	if !fileExists(name) {
		t.Fatalf("failed to create file")
	}

	//
	// Now we've created the file we'll write
	// to it.
	//
	// Fill the DMA area
	c.Memory.Set(c.dma+0, 'S')
	c.Memory.Set(c.dma+1, 't')
	c.Memory.Set(c.dma+2, 'e')
	c.Memory.Set(c.dma+3, 'v')
	c.Memory.Set(c.dma+4, 'e')
	c.Memory.Set(c.dma+5, 0x00)

	err = BdosSysCallWrite(c)
	if err != nil {
		t.Fatalf("got error writing to file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("got error writing to file: A")
	}

	// Now we've created it the file will be open
	// Close it.
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileClose(c)
	if err != nil {
		t.Fatalf("failed to close file, after creation")
	}

	// Now we can open it and confirm it has the data we expect
	var data []byte
	data, err = os.ReadFile(name)
	if err != nil {
		t.Fatalf("failed to read file: %s", err)
	}

	if len(data) != 128 {
		t.Fatalf("file created isn't a multiple of the block size")
	}

	if data[0] != 'S' {
		t.Fatalf("wrong contents")
	}
	if data[1] != 't' {
		t.Fatalf("wrong contents")
	}
	if data[2] != 'e' {
		t.Fatalf("wrong contents")
	}
	if data[3] != 'v' {
		t.Fatalf("wrong contents")
	}
	if data[4] != 'e' {
		t.Fatalf("wrong contents")
	}
	if data[5] != 0x00 {
		t.Fatalf("wrong contents")
	}

	// Delete the file, if it was present
	os.Remove(name)

	//
	// Now try to write to a file that is not open
	//
	data = make([]byte, 200)
	f := fcb.FromBytes(data)
	c.Memory.SetRange(0x0200, f.AsBytes()...)
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallWrite(c)
	if err != nil {
		t.Fatalf("error calling cp/m")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("expected A-reg to hold an error")
	}
}

// TestReadFile tests reading a sequential record from an open file.
func TestReadFile(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	//
	// Now try to read from a file that is not open
	//
	data := make([]byte, 200)
	f := fcb.FromBytes(data)
	c.Memory.SetRange(0x0200, f.AsBytes()...)
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallRead(c)
	if err != nil {
		t.Fatalf("error calling cp/m")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("expected A-reg to hold an error")
	}

}

// TestFileOpen ensures we can open files.
func TestFileOpen(t *testing.T) {
	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	c.SetDrives(false)
	defer c.IOTearDown()

	// Create a binary

	var file *os.File
	name := "OPEN.ME"
	file, err = os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to create test-file")
	}
	file.Close()
	defer os.Remove(name)

	// Create an FCB pointing to the file
	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 05
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call Open
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("failed to open file: A=%02X", c.CPU.States.AF.Hi)
	}

	// Try to open a file that doesn't exist
	fcbPtr = fcb.FromString("invalid.txt")
	fcbPtr.Drive = 9
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call Open
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open file: err")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("failed to open file: A=%02X", c.CPU.States.AF.Hi)
	}

	// Try to open an embedded file
	c.SetStaticFilesystem(static.GetContent())
	fcbPtr = fcb.FromString("A:!CTRLC.COM")
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call Open
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open embedded file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("failed to open embedded file: A=%02X", c.CPU.States.AF.Hi)
	}

	// Close the file
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileClose(c)
	if err != nil {
		t.Fatalf("failed to close embedded file: err")
	}

	// Try to open a file with no name
	data := make([]byte, 200)
	f := fcb.FromBytes(data)
	c.Memory.SetRange(0x0200, f.AsBytes()...)
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open file: err")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("failed to open file: A=%02X", c.CPU.States.AF.Hi)
	}

}

func TestRead(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	// Create a file
	name := "READ.ME"

	// Create a binary
	var file *os.File
	file, err = os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to create file")
	}
	defer os.Remove(name)

	// Write some data there
	_, err = file.Write([]byte{0x01, 0x02, 0xCD, 0xFF})
	if err != nil {
		t.Fatalf("failed to write to file")
	}

	// Close the file
	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file")
	}

	fcbPtr := fcb.FromString(name)
	fcbPtr.Drive = 5
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Open the file
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("failed to open file: A=%02X", c.CPU.States.AF.Hi)
	}

	// Call the read function
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallRead(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}

	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("read failed")
	}

	// Ensure the data is correct
	if c.Memory.Get(c.dma) != 0x01 {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+1) != 0x02 {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+2) != 0xCD {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+3) != 0xFF {
		t.Fatalf("wrong value read")
	}

	//
	// Try a random read too, just for fun
	//
	err = BdosSysCallReadRand(c)
	if err != nil {
		t.Fatalf("failed read rand")
	}
	if c.Memory.Get(c.dma) != 0x01 {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+1) != 0x02 {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+2) != 0xCD {
		t.Fatalf("wrong value read")
	}
	if c.Memory.Get(c.dma+3) != 0xFF {
		t.Fatalf("wrong value read")
	}

	//
	// Read from a virtual file
	//
	c.SetStaticFilesystem(static.GetContent())
	fcbPtr = fcb.FromString("A:!CTRLC.COM")
	c.Memory.SetRange(0x0200, fcbPtr.AsBytes()...)

	// Call Open
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallFileOpen(c)
	if err != nil {
		t.Fatalf("failed to open embedded file: err")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("failed to open embedded file: A=%02X", c.CPU.States.AF.Hi)
	}

	// Call read
	c.CPU.States.DE.SetU16(0x0200)
	err = BdosSysCallRead(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("read (virtual) failed")
	}

	//
	// Try a random read too, just for fun
	//
	err = BdosSysCallReadRand(c)
	if err != nil {
		t.Fatalf("error calling CP/M")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("read rand (virtual) failed")
	}

}

func TestTicks(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	// Ensure we have a launch time that was in the past.
	if !c.launchTime.Before(time.Now()) {
		t.Fatalf("time travel isn't possible")
	}

	// Call the function
	err = BdosSysCallUptime(c)
	if err != nil {
		t.Fatalf("unexpected error getting ticks")
	}

	// Get the before time.
	a := c.CPU.States.HL.U16()

	// Call the function
	err = BdosSysCallUptime(c)
	if err != nil {
		t.Fatalf("unexpected error getting ticks")
	}

	// Get the after time.
	b := c.CPU.States.HL.U16()

	if a == b {
		t.Fatalf("no time has passed %d %d", a, b)
	}
}
