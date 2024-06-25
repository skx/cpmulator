package memory

import (
	"os"
	"testing"
)

// TestMemoryTrivial just does basic get/set tests
func TestMemoryTrivial(t *testing.T) {

	mem := new(Memory)

	// Set
	mem.Set(0x00, 0x01)
	mem.Set(0x01, 0x02)

	// Get
	if mem.Get(0x00) != 0x01 {
		t.Fatalf("failed to get expected result")
	}
	if mem.Get(0x01) != 0x02 {
		t.Fatalf("failed to get expected result")
	}
	// GetU16
	if mem.GetU16(0x00) != 0x0201 {
		t.Fatalf("failed to get expected result")
	}

	// Fill with 0xCD
	mem.FillRange(0x00, 0xFFFF, 0xCD)

	if mem.Get(0xFFFE) != 0xCD {
		t.Fatalf("failed to get expected result")
	}
	// GetU16
	if mem.GetU16(0x0100) != 0xCDCD {
		t.Fatalf("failed to get expected result")
	}

	// Get a random range
	out := mem.GetRange(0x300, 0x00FF)
	for _, d := range out {
		if d != 0xCD {
			t.Fatalf("wrong result in GetRange")
		}
	}

	// Put a (small) range
	out = []uint8{0x01, 0x02, 0x03}
	mem.SetRange(0x0000, out[:]...)

	if mem.Get(0x00) != 0x01 {
		t.Fatalf("failed to get expected result")
	}
	if mem.Get(0x01) != 0x02 {
		t.Fatalf("failed to get expected result")
	}
	// GetU16
	if mem.GetU16(0x00) != 0x0201 {
		t.Fatalf("failed to get expected result")
	}
	if mem.GetU16(0x02) != 0xCD03 {
		t.Fatalf("failed to get expected result")
	}
}

// TestLoadFile ensures we can load a file
func TestLoadFile(t *testing.T) {

	// Create memory
	mem := new(Memory)

	err := mem.LoadFile(0, "/this/file-does/not/exist")
	if err == nil {
		t.Fatalf("expected error, got none")
	}

	// Now write out a temporary file, with static contents.
	var file *os.File
	file, err = os.CreateTemp("", "tst-*.mem")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Write some known-text to the file
	_, err = file.WriteString("Steve Kemp")
	if err != nil {
		t.Fatalf("failed to write program to temporary file")
	}

	// Close the file
	file.Close()

	// Load the file
	err = mem.LoadFile(0, file.Name())
	if err != nil {
		t.Errorf("failed to load file")
	}

	// Confirm the contents are OK
	x := "Steve Kemp"
	for i, c := range x {
		chr := mem.Get(uint16(i))
		if string(chr) != string(c) {
			t.Fatalf("RAM had wrong contents at %d: %c != %c\n", i, c, chr)
		}
	}
}
