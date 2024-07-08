package cpm

import (
	"os"
	"testing"

	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
)

func TestFileSize(t *testing.T) {

	create := func(name string, size int) error {
		f, err := os.Create(name)
		if err != nil {
			return err
		}
		defer f.Close()

		d := []byte{0x00}
		for size > 0 {
			f.Write(d)
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
		create(name, sz)

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
