package cpm

import (
	"errors"
	"os"
	"testing"

	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
)

func TestUnimplemented(t *testing.T) {
	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.Cleanup()

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
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.Cleanup()

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
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	c.fixupRAM()
	c.SetDrives(false)
	defer c.Cleanup()

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
	for true {
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
}

func TestReadLine(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Stuff some fake input
	c.input = consolein.New()
	c.input.StuffInput("steve")

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

}

func TestDriveGetSet(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Get the drive, wahtever it is
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
	c.CPU.States.AF.Hi = 0xff
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

func TestSetDMA(t *testing.T) {

	// Create a new helper
	c, err := New()
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

func TestUserNumbeR(t *testing.T) {

	// Create a new helper
	c, err := New()
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
	c.CPU.States.DE.Lo = 0xff
	err = BdosSysCallUserNumber(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 05 {
		t.Fatalf("retriving user number failed")
	}

}
func TestDriveReset(t *testing.T) {

	getState := func() uint8 {
		// Create a new helper
		c, err := New()
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
	if getState() != 0xff {
		t.Fatalf("getState != 0xff")
	}
}
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
	c, err := New()
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
	c, err := New()
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
	c.CPU.States.DE.Lo = 0xfe
	err = BdosSysCallSetIOByte(c)
	if err != nil {
		t.Fatalf("error in CPM call")
	}

	// get it
	err = BdosSysCallGetIOByte(c)
	if err != nil {
		t.Fatalf("error in CPM call")
	}

	if c.CPU.States.AF.Hi != 0xfe {
		t.Fatalf("unexpected updated IO byte")
	}
}

func TestBDOSCoverage(t *testing.T) {

	// Create a new helper
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

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
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)

	// Files created in "."
	c.SetDrives(false)

	fileExists := func(path string) bool {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
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
	c.CPU.States.DE.SetU16(0xcdcd)
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
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("expected error with empty file")
	}

	// Delete the file, if it was present
	os.Remove(name)
}

func TestDelete(t *testing.T) {

	// Create a new helper
	c, err := New()
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
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
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

func TestRename(t *testing.T) {

	// Create a new helper
	c, err := New()
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
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
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
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("renaming to an impossible name succeeded")
	}

}
