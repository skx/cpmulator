package cpm

import (
	"strings"
	"testing"

	"github.com/skx/cpmulator/memory"
)

func TestMisc(t *testing.T) {
	// Create a new helper
	c, err := New(WithPrinterPath("1.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}

	// Punch (auxout) does nothing.
	err = BiosSysCallPunch(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}

	// Reader (auxin) should return Ctrl-Z
	err = BiosSysCallReader(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 26 {
		t.Fatalf("auxin result was wrong %02X", c.CPU.States.AF.Hi)
	}
}

func TestStatus(t *testing.T) {

	// Create a new helper
	c, err := New(WithPrinterPath("1.log"))
	if err != nil {
		t.Fatalf("failed to create CPM")
	}
	c.StuffText("")
	err = BiosSysCallConsoleStatus(c)
	if err != nil {
		t.Fatalf("failed to call CPM")
	}
	if c.CPU.States.AF.Hi != 0x00 {
		t.Fatalf("console status was wrong %02X", c.CPU.States.AF.Hi)
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

	if BiosSysCallReserved1NOP(c) != nil {
		t.Fatalf("unexpected error")
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
	// Get/Set console output driver
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str := getStringFromMemory(c, c.dma)
	if str != DefaultOutputDriver {
		t.Fatalf("wrong console driver '%s'", str)
	}

	// set to "null"
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'n', 'u', 'l', 'l', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "null" - a second time
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'n', 'u', 'l', 'l', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "null" - again with empty options
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'n', 'u', 'l', 'l', ':', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to ""
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "null" - again with options
	c.CPU.States.HL.SetU16(0x0002)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{':', 'f', 'o', 0x00}...)
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

	// set to "ccpz", again
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'c', 'c', 'p', 'z', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "ccpz", again
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'c', 'c', 'p', 'z', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "ccpz", again
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'C', 'C', 'P', 'Z', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to an empty string
	c.CPU.States.HL.SetU16(0x0003)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{0x00}...)
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
	//  Simple Debug - Retired.

	// 0x0007
	// Get/Set console input driver
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != DefaultInputDriver {
		t.Fatalf("wrong console driver '%s'", str)
	}

	// set to "file" this will fail as the file doesn't exist
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'f', 'i', 'l', 'e', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err == nil {
		t.Fatalf("expected error, got none.")
	}

	// set to "stty"
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 't', 'y', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function %s", err)
	}

	// set to "stty", again
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 't', 'y', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function %s", err)
	}

	// set to "stty", again this time with options too
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{':', 'f', 'o', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function %s", err)
	}

	// set to an empty string
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function %s", err)
	}

	// set to "stty", again this time with empty options
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 't', 'y', ':', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// set to "steve" - this will fail
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'s', 't', 'e', 'v', 'e', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err == nil {
		t.Fatalf("error calling reserved function")
	}

	// read it back
	c.CPU.States.HL.SetU16(0x0007)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "stty" {
		t.Fatalf("wrong console driver '%s'", str)
	}

	// 0x0008
	// Get/Set host command prefix
	c.CPU.States.HL.SetU16(0x0008)
	c.CPU.States.DE.SetU16(0x0000)
	c.Memory.Set(c.dma, 0x00)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "" {
		t.Fatalf("unexpected systemcommandprefix '%s'", str)
	}

	// set to "!!"
	c.CPU.States.HL.SetU16(0x0008)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'!', '!', ' ', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// confirm it worked
	c.CPU.States.HL.SetU16(0x0008)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "!!" {
		t.Fatalf("unexpected systemcommandprefix '%s'", str)
	}
	if c.input.GetSystemCommandPrefix() != "!!" {
		t.Fatalf("unexpected mismatch in systemcommandprefix, got '%v'", c.input.GetSystemCommandPrefix())
	}

	// set to "/clear"
	c.CPU.States.HL.SetU16(0x0008)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'/', 'c', 'l', 'e', 'a', 'r', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}

	// confirm it worked
	if c.input.GetSystemCommandPrefix() != "" {
		t.Fatalf("unexpected mismatch in systemcommandprefix")
	}

	// 0x0009
	// Disable things
	c.CPU.States.HL.SetU16(0x0009)
	n := 1
	for n < 6 {
		c.CPU.States.DE.SetU16(uint16(n))
		err = BiosSysCallReserved1(c)
		if err != nil {
			t.Fatalf("error calling reserved function")
		}
		n++
	}

	// 0x000A
	// Change printer path.
	// set to "prn.log"
	c.CPU.States.HL.SetU16(0x000A)
	c.CPU.States.DE.SetU16(0xFE00)
	c.Memory.SetRange(0xFE00, []byte{'p', 'r', 'n', '.', 'l', 'o', 'g', 0x00}...)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	if c.prnPath != "prn.log" {
		t.Fatalf("changing printer path failed")
	}

	// Reset the value, and retrieve it
	c.prnPath = "the.sky.log"

	c.CPU.States.HL.SetU16(0x000A)
	c.CPU.States.DE.SetU16(0x0000)
	err = BiosSysCallReserved1(c)
	if err != nil {
		t.Fatalf("error calling reserved function")
	}
	str = getStringFromMemory(c, c.dma)
	if str != "the.sky.log" {
		t.Fatalf("unexpected printer path '%s'", str)
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

	defer func() {
		tErr := c.IOTearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

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
	defer func() {
		tErr := c.IOTearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	// Ensure we get an error when printing
	err = c.prnC('s')
	if err == nil {
		t.Fatalf("expected error writing to impossible file")
	}

	// 5 == LIST / BiosSysCallPrintChar
	if c.syscallErr != nil {
		t.Fatalf("found an error we didn't expect")
	}

	// This will fail
	c.CPU.States.AF.Hi = 0x05
	c.Out(0xFF, 5)

	// So the error should be set
	if c.syscallErr == nil {
		t.Fatalf("found no error, but we should have done")
	}

	// 15 == LISTST == BiosSysCallPrinterStatus
	c.CPU.States.BC.SetU16(15)
	c.Out(0xFF, 15)

}
