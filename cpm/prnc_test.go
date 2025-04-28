package cpm

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

var (
	// ErrClose is the error we send when mocking a File Close operation.
	ErrClose = fmt.Errorf("CLOSE")

	// ErrWrite is the error we send when mocking a File Write operation.
	ErrWrite = fmt.Errorf("WRITE")
)

// mockFile is a structure used for mocking file failures
type mockFile struct {
	// failWrite will trigger the Write method, from our File interface, to return an error.
	failWrite bool

	// failClose will trigger the Close method, from our File interface, to return an error.
	failClose bool
}

// Write is the mocked Write method from the File interface, used for testing.
func (m *mockFile) Write(p []byte) (int, error) {
	if m.failWrite {
		return 0, ErrWrite
	}
	return len(p), nil
}

// Close is the mocked Write method from the File interface, used for testing.
func (m *mockFile) Close() error {
	if m.failClose {
		return ErrClose
	}
	return nil
}

// TestPrinterOutput tests that printer output goes to the file as
// expected.
func TestPrinterOutput(t *testing.T) {

	// Create a printer-output file
	file, err := os.CreateTemp("", "tst-*.prn")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err = New(WithPrinterPath(file.Name()))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.prnPath != file.Name() {
		t.Fatalf("unexpected filename for printer log")
	}

	// Now output some characters
	err = obj.prnC('s')
	if err != nil {
		t.Fatalf("failed to write character to printer-file")
	}

	obj.CPU.States.DE.Lo = 'k'
	err = BdosSysCallPrinterWrite(obj)
	if err != nil {
		t.Fatalf("failed to write character to printer-file")
	}

	obj.CPU.States.BC.Lo = 'x'
	err = BiosSysCallPrintChar(obj)
	if err != nil {
		t.Fatalf("failed to write character to printer-file")
	}

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

// TestWriteFail tests what happens when a file Write method fails.
func TestWriteFail(t *testing.T) {

	opener = func(name string, flag int, perm os.FileMode) (File, error) {
		return &mockFile{failWrite: true}, nil
	}

	// Create a printer-output file
	file, err := os.CreateTemp("", "tst-*.prn")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err = New(WithPrinterPath(file.Name()))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.prnPath != file.Name() {
		t.Fatalf("unexpected filename for printer log")
	}

	// Now output a character, which we expect to fail.
	err = obj.prnC('s')
	if err == nil {
		t.Fatalf("expected error, got none %s", err)
	}
	if !strings.Contains(err.Error(), ErrWrite.Error()) {
		t.Fatalf("got an error, but the wrong one")
	}
}

// TestWriteClose tests what happens when a file Close method fails.
func TestWriteClose(t *testing.T) {

	opener = func(name string, flag int, perm os.FileMode) (File, error) {
		return &mockFile{failClose: true}, nil
	}

	// Create a printer-output file
	file, err := os.CreateTemp("", "tst-*.prn")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err = New(WithPrinterPath(file.Name()))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.prnPath != file.Name() {
		t.Fatalf("unexpected filename for printer log")
	}

	// Now output a character, which we expect to fail.
	err = obj.prnC('s')
	if err == nil {
		t.Fatalf("expected error, got none %s", err)
	}
	if !strings.Contains(err.Error(), ErrClose.Error()) {
		t.Fatalf("got an error, but the wrong one %v", err)
	}
}
