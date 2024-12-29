package cpm

import (
	"strings"
	"testing"

	"github.com/skx/cpmulator/memory"
)

func TestStatus(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("1.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	err = BiosSysCallConsoleStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("console status was wrong")
	}

	c.input.StuffInput("S")
	err = BiosSysCallConsoleStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("console status was wrong")
	}

	err = BiosSysCallPrinterStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("printer status was wrong")
	}

	err = BiosSysCallScreenOutputStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("screen status was wrong")
	}

	err = BiosSysCallAuxInputStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("aux input status was wrong")
	}

	err = BiosSysCallAuxOutputStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	if c.CPU.States.AF.Hi != 0xFF {
		t.Fatalf("aux output status was wrong")
	}

}

func TestCustom(t *testing.T) {

	// Get a null/space terminated string from memory
	getStringFromMemory := func(cpm *CPM, addr uint16) string {
		str := ""
		x := cpm.Memory.Get(addr)
		for x != ' ' && x != 0x00 {
			str += string(x)
			addr++
			x = cpm.Memory.Get(addr)
		}

		// Useful when the CCP has passed a string, because
		// that uppercases all input
		return strings.ToLower(str)
	}

	// Create a new helper
	c, err := New(WithPrinterPath("2.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	c.Memory = new(memory.Memory)

	// Invalid call
	c.CPU.HL.SetU16(0x1234)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// 0x0000
	c.CPU.HL.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.CPU.States.HL.Hi != 'S' {
		t.Fatalf("Reserved1: HL != S")
	}
	if c.CPU.States.HL.Lo != 'K' {
		t.Fatalf("Reserved1: HL != K")
	}

	// 0x0001
	c.CPU.States.HL.SetU16(0x0001)
	c.CPU.States.BC.Lo = 0xFF
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.CPU.States.AF.Hi != 2 {
		t.Fatalf("Reserved1: Wrong default count for Ctrl.C")
	}

	// change the value
	c.CPU.States.BC.Lo = 0xFE
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	// read it back
	c.CPU.States.BC.Lo = 0xFF
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.CPU.States.AF.Hi != 0xFE {
		t.Fatalf("Reserved1: Wrong Ctrl-C value - update failed")
	}

	// 0x0002
	// Get/Set console driver
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str := getStringFromMemory(c, c.dma)
	if str != "adm-3a" {
		t.Fatalf("wrong console driver '%s'", str)
	}

	// set to "null"
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'n', 'u', 'l', 'l', ' '}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "steve" - this will fail
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 'e', 'v', 'e', ' '}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// read it back
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "null" {
		t.Fatalf("wrong console driver '%s'", str)
	}

	// 0x0003
	// Get/Set CCP
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "ccp" {
		t.Fatalf("wrong ccp name '%s'", str)
	}

	// set to "ccpz"
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'c', 'c', 'p', 'z', ' '}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "steve" - this will fail
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 'e', 'v', 'e', ' '}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// read it back
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "ccpz" {
		t.Fatalf("wrong ccp name '%s'", str)
	}

	// 0x0004
	// Retired.
	c.CPU.States.HL.SetU16(0x0004)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// 0x0005
	// Terminal width
	c.CPU.States.HL.SetU16(0x0005)
	_ = BiosSysCallReserved1(c)

	// 0x0006
	// Debug Flag
	c.CPU.States.HL.SetU16(0x0006)
	c.CPU.States.BC.Lo = 0xFF
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	// Set it
	c.CPU.States.HL.SetU16(0x0006)
	c.CPU.States.BC.Lo = 0x00
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	// Make sure it worked
	c.CPU.States.HL.SetU16(0x0006)
	c.CPU.States.BC.Lo = 0xFF
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.CPU.States.BC.Lo != 0x00 {
		t.Fatalf("setting flag failed")
	}

	c.CPU.States.HL.SetU16(0x0006)
	c.CPU.States.BC.Lo = 0x01
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	// Make sure it worked
	c.CPU.States.HL.SetU16(0x0006)
	c.CPU.States.BC.Lo = 0xFF
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.CPU.States.BC.Lo != 0x01 {
		t.Fatalf("setting flag failed")
	}

}

func TestBIOSConsoleInput(t *testing.T) {
	// Create a new helper
	c, err := New(WithPrinterPath("3.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.IOTearDown()

	c.input.StuffInput("s")
	err = BiosSysCallConsoleInput(c)
	if err != nil {
		t.Fatalf("failed to call CP/M")
	}
	if c.CPU.States.AF.Hi != 's' {
		t.Fatalf("got the wrong input")
	}

}

// Test that an error comes by setting an impossible filename for the printer
func TestBIOSError(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("C:/fdf/fdsfd/fdsf·\\332\fdsf/fsdf.invlaid"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.Memory = new(memory.Memory)
	c.fixupRAM()
	defer c.IOTearDown()

	c.simpleDebug = true

	// Ensure we get an error when printing
	err = c.prnC('s')
	if err == nil {
		t.Fatalf("expected error writing to impossible file")
	}

	// 5 == LIST / BiosSysCallPrintChar
	if c.biosErr != nil {
		t.Fatalf("found an error we didn't expect")
	}

	// This will fail
	c.BiosHandler(5)

	// So the error should be set
	if c.biosErr == nil {
		t.Fatalf("found no error, but we should have done")
	}

	// 15 == LISTST == BiosSysCallPrinterStatus
	c.BiosHandler(15)

}
