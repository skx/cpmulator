package memory

import (
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
