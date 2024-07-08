package cpm

import "testing"

func TestStatus(t *testing.T) {

	// Create a new helper
	c, _ := New()

	BiosSysCallConsoleStatus(c)
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("console status was wrong")
	}

	BiosSysCallPrinterStatus(c)
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("printer status was wrong")
	}

	BiosSysCallScreenOutputStatus(c)
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("screen status was wrong")
	}

	BiosSysCallAuxInputStatus(c)
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("aux input status was wrong")
	}

	BiosSysCallAuxOutputStatus(c)
	if c.CPU.States.AF.Hi != 0xff {
		t.Fatalf("aux output status was wrong")
	}

}
