package cpm

import (
	"context"
	"os"
	"testing"

	"github.com/skx/cpmulator/memory"
)

// TestSimple ensures the most basic program runs
func TestSimple(t *testing.T) {

	// Create a new CP/M helper
	obj, err := New(WithOutputDriver("null"), WithInputDriver("stty"))
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
	if obj.GetInputDriver().GetName() != "stty" {
		t.Fatalf("console driver name mismatch!")
	}
	if obj.GetCCPName() != "ccp" {
		t.Fatalf("ccp name mismatch!")
	}

	// Ensure our BDOS and BIOS don't equal each other
	if obj.GetBDOSAddress() == obj.GetBIOSAddress() {
		t.Fatalf("BIOS and BDOS should be different!")
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
	if err != ErrBoot {
		t.Fatalf("failed to run binary %v!", err)
	}

	defer func() {
		tErr := obj.IOTearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

}

func TestBogusConstructor(t *testing.T) {

	_, err := New(WithOutputDriver("bogus"))
	if err == nil {
		t.Fatalf("expected error, bogus console driver, got none")
	}

	_, err = New(WithInputDriver("bogus"))
	if err == nil {
		t.Fatalf("expected error, bogus console driver, got none")
	}

	x, _ := New(WithInputDriver("stty"))
	sErr := x.IOSetup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := x.IOTearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	ctx := context.Background()
	_, err = New(WithContext(ctx))
	if err != nil {
		t.Fatalf("loading a context failed")
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
	// Ensure we have memory
	obj.Memory = new(memory.Memory)

	obj.In(0x12)
	obj.CPU.HALT = false
	obj.syscallErr = nil
	obj.Out(0x12, 0x34)

	// Valid: COLDBOOT
	obj.CPU.HALT = false
	obj.syscallErr = nil
	obj.Out(0xFF, 0x00)
	if obj.syscallErr != ErrBoot {
		t.Fatalf("unexpected error")
	}
	// Valid: WARMBOOT
	obj.CPU.HALT = false
	obj.syscallErr = nil
	obj.CPU.States.AF.Hi = 0x01
	obj.CPU.States.BC.Lo = 0x01
	obj.Out(0xFF, 0x01)
	if obj.syscallErr != ErrBoot {
		t.Fatalf("unexpected error")
	}

	// Invalid
	obj.CPU.HALT = false
	obj.syscallErr = nil
	obj.CPU.States.AF.Hi = 0xFE
	obj.CPU.States.BC.Lo = 0xFE
	obj.Out(0xFE, 0xFE)
	if obj.syscallErr != ErrUnimplemented {
		t.Fatalf("expected unimplemented, got %s", obj.syscallErr)
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

func TestHostExec(t *testing.T) {

	// Create a new CP/M helper
	obj, err := New(WithOutputDriver("null"), WithHostExec("!#!"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.input.GetSystemCommandPrefix() != "!#!" {
		t.Fatalf("WithHostExec didn't work as expected")
	}
}

func TestAddressOveride(t *testing.T) {

	// Create a new CP/M helper - default
	obj, err := New(WithOutputDriver("null"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.bdosAddress != 0xFA00 {
		t.Fatalf("default BDOS address is wrong, got %04X", obj.bdosAddress)
	}

	// Create a new CP/M helper - with env set
	t.Setenv("BDOS_ADDRESS", "0x1234")
	obj, err = New(WithOutputDriver("null"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	if obj.bdosAddress != 0x1234 {
		t.Fatalf("updated BDOS address is wrong")
	}

	// Create a new CP/M helper - with bogus env set
	t.Setenv("BDOS_ADDRESS", "steve")
	obj, err = New(WithOutputDriver("null"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	if obj.bdosAddress != 0xFA00 {
		t.Fatalf("updated BDOS address is wrong")
	}

}
