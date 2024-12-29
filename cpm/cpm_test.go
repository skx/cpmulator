package cpm

import (
	"os"
	"testing"

	"github.com/skx/cpmulator/memory"
)

// TestSimple ensures the most basic program runs
func TestSimple(t *testing.T) {

	// Create a new CP/M helper
	obj, err := New(WithConsoleDriver("null"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	// Ensure we have memory
	obj.Memory = new(memory.Memory)

	// Write a simple character to the output
	//
	// Since our driver is "null" this will be silently discarded
	err = BiosSysCallConsoleOutput(obj)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	err = BdosSysCallAuxWrite(obj)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	err = BdosSysCallWriteChar(obj)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	// Write a string of three bytes to the console - again discarded
	obj.CPU.States.DE.SetU16(0xFE00)
	obj.Memory.Set(0xFE00, 's')
	obj.Memory.Set(0xFE01, 'k')
	obj.Memory.Set(0xFE02, 'x')
	obj.Memory.Set(0xFE03, '$')
	err = BdosSysCallWriteString(obj)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	// Confirm the output driver is null, as expected
	if obj.GetOutputDriver().GetName() != "null" {
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
	_, err = file.Write([]byte{0xC9})
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

	// Now load the real binary - but first of all remove
	// the RAM
	obj.Memory = nil
	err = obj.LoadBinary(file.Name())
	if err != nil {
		t.Fatalf("failed to load binary")
	}

	// Finally launch it
	err = obj.Execute([]string{})
	if err != nil {
		t.Fatalf("failed to run binary!")
	}

	defer obj.IOTearDown()
}

func TestBogusConstructor(t *testing.T) {

	_, err := New(WithConsoleDriver("bogus"))
	if err == nil {
		t.Fatalf("expected error, bogus console driver, got none")
	}
}

func TestLoadCCP(t *testing.T) {

	// Create a new CP/M helper - valid
	var obj *CPM
	obj, err := New()
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	err = obj.LoadCCP()
	if err != nil {
		t.Fatalf("failed to load CCP")
	}

	// Create a new CP/M helper - invalid
	obj, err = New(WithCCP("ccp-invalid"))
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

// TestLogNoisy tests that functions are updated appropriately.
func TestLogNoisy(t *testing.T) {

	obj, err := New()
	if err != nil {
		t.Fatalf("failed to create CP/M object")
	}

	// Count the syscalls that are noisy
	count := func() int {
		noisy := 0

		for _, syscall := range obj.BIOSSyscalls {
			if syscall.Noisy {
				noisy++
			}
		}
		for _, syscall := range obj.BDOSSyscalls {
			if syscall.Noisy {
				noisy++
			}
		}
		return noisy
	}

	// Count before
	before := count()
	if before <= 0 {
		t.Fatalf("count of noisy functions was bogus")
	}

	// Change the logging
	obj.LogNoisy()

	// Count after
	after := count()
	if after != 0 {
		t.Fatalf("count of noisy functions was bogus")
	}
}

func TestDrives(t *testing.T) {

	obj, err := New()
	if err != nil {
		t.Fatalf("failed to create CP/M object")
	}

	// All drives will be "."
	obj.SetDrives(false)

	// Confirm it
	for _, x := range obj.drives {
		if x != "." {
			t.Fatalf("drive path wrong")
		}
	}

	// All drives will be the letter
	obj.SetDrives(true)

	// Confirm it
	for k, x := range obj.drives {
		if x != k {
			t.Fatalf("drive path wrong")
		}
	}

	// Bonus
	if obj.drives["A"] != "A" {
		t.Fatalf("bogus A:")
	}
	obj.SetDrivePath("A", "STEVE")
	if obj.drives["A"] != "STEVE" {
		t.Fatalf("bogus A:")
	}
}

// TestCPMCoverage is just coverage messup
func TestCPMCoverage(t *testing.T) {

	obj, err := New()
	if err != nil {
		t.Fatalf("failed to create CP/M object")
	}

	obj.In(0x12)
	obj.Out(0x12, 0x34)

	// Valid: COLDBOOT
	obj.Out(0xFF, 0x00)
	if obj.biosErr != nil {
		t.Fatalf("unexpected error")
	}
	// Valid: WARMBOOT
	obj.Out(0xFF, 0x01)
	if obj.biosErr != nil {
		t.Fatalf("unexpected error")
	}

	// Invalid
	obj.Out(0xFF, 0xFF)
	if obj.biosErr != ErrUnimplemented {
		t.Fatalf("expected unimplemented, got %s", obj.biosErr)
	}
}

func TestAutoExec(t *testing.T) {

	obj, err := New()
	if err != nil {
		t.Fatalf("failed to create CP/M object")
	}

	// All drives will be in PWD
	obj.SetDrives(false)

	// Ensure there's _something_ to read
	obj.StuffText("nothing\n")
	obj.RunAutoExec()

	var out string
	out, err = obj.input.ReadLine(200)
	if err != nil {
		t.Fatalf("failed to call ReadLine")
	}
	if out != "nothing" {
		t.Fatalf("strange input read: %s\n", out)
	}

	// Now create the two files which would drive the
	// submit-handler.
	var file1 *os.File
	var file2 *os.File
	file1, err = os.OpenFile("SUBMIT.COM", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to create file %s", err)
	}
	file2, err = os.OpenFile("AUTOEXEC.SUB", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("failed to create file %s", err)
	}

	// Close the files, once they exist
	file1.Close()
	file2.Close()

	defer func() {
		os.Remove("SUBMIT.COM")
		os.Remove("AUTOEXEC.SUB")
	}()

	// Ensure there's _something_ to read
	obj.StuffText("nothing\n")
	obj.RunAutoExec()

	out, err = obj.input.ReadLine(200)
	if err != nil {
		t.Fatalf("failed to call ReadLine")
	}
	if out != "SUBMIT AUTOEXEC" {
		t.Fatalf("strange input read: '%s'\n", out)
	}

}
