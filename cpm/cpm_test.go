package cpm

import (
	"os"
	"testing"
)

// TestSimple ensures the most basic program runs
func TestSimple(t *testing.T) {

	// Create a new CP/M helper
	obj, err := New("xx", "null", "ccp")
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	// Confirm the output
	if obj.GetOutputDriver() != "null" {
		t.Fatalf("console driver name mismatch!")
	}
	if obj.GetCCPName() != "ccp" {
		t.Fatalf("ccp name mismatch!")
	}

	// Create a temporary file with our "RET" program in it.
	var file *os.File
	file, err = os.CreateTemp("", "tst-*.com")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Write "RET" to the file
	_, err = file.WriteString("0xC9")
	if err != nil {
		t.Fatalf("failed to write program to temporary file")
	}

	// Close the file
	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file")
	}

	// Attempt to load an invalid binary
	err = obj.LoadBinary("this/fil/does/not/exist")
	if err == nil {
		t.Fatalf("expected an error loading a bogus binary, got none")
	}

	// Now load the real binary
	err = obj.LoadBinary(file.Name())
	if err != nil {
		t.Fatalf("failed to load binary")
	}

	// Finally launch it
	err = obj.Execute([]string{})
	if err != nil {
		t.Fatalf("failed to run binary!")
	}

	defer obj.Cleanup()
}

func TestLoadCCP(t *testing.T) {

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err := New("xx", "null", "ccp")
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	err = obj.LoadCCP()
	if err != nil {
		t.Fatalf("failed to load CCP")
	}

	// Create a new CP/M helper - invalid
	obj, err = New("xx", "null", "ccp-invalid")
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	err = obj.LoadCCP()
	if err == nil {
		t.Fatalf("expected an error loading invalid CCP, got none")
	}

}

// TestPrinterOutput tests that printer output goes to the file as
// expected.
func TestPrinterOutpu(t *testing.T) {

	// Create a printer-output file
	file, err := os.CreateTemp("", "tst-*.prn")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err = New(file.Name(), "null", "ccp")
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	// Now output some characters
	obj.prnC('s')
	obj.prnC('k')
	obj.prnC('x')

	// Read back the file.
	var data []byte
	data, err = os.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("failed to read from file")
	}

	if string(data) != "skx" {
		t.Fatalf("printer output had the wrong content")
	}
}
