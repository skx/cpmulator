package cpm

import (
	"io/ioutil"
	"os"
	"testing"
)

// TestSimple ensures the most basic program runs
func TestSimple(t *testing.T) {

	// Create a new CP/M program
	var obj *CPM
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
	file, err = ioutil.TempFile("", "tst-*.com")
	if err != nil {
		t.Fatalf("failed to create tmpoerary file")
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

	// Now load the binary
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
