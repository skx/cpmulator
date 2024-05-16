package cpm

import (
	"fmt"
	"log/slog"

	cpmio "github.com/skx/cpmulator/io"
)

// BiosSysCallBoot handles a warm/cold boot.
func BiosSysCallBoot(cpm *CPM) error {

	// Set entry-point to 0x0000 which will result in
	// a boot-trap.
	cpm.CPU.States.PC = 0x0000
	return nil
}

// BiosSysCallConsoleStatus should return 0x00 if there is no input
// pending, otherwise 0xFF.  We fake it
func BiosSysCallConsoleStatus(cpm *CPM) error {
	cpm.CPU.States.AF.Hi = 0x00

	return nil
}

func BiosSysCallConsoleInput(cpm *CPM) error {

	// Wait until the keyboard is ready to provide a character, and return it in A.
	// Use our I/O package
	obj := cpmio.New(cpm.Logger)

	out, err := obj.BlockForCharacter()
	if err != nil {
		return err
	}

	cpm.CPU.States.AF.Hi = out
	return nil
}

func BiosSysCallConsoleOutput(cpm *CPM) error {

	// Write the character in C to the screen.
	c := cpm.CPU.States.BC.Lo
	cpm.outC(c)

	return nil
}

func BiosSysCallPrintChar(cpm *CPM) error {

	// Write the character in C to the printer
	c := cpm.CPU.States.BC.Lo

	// Write the character to the printer
	err := cpm.prnC(c)
	return err
}

// BiosHandler is involved when a BIOS syscall needs to be executed,
// which is handled via trapping RST instructions and using a short
// trampoline.
//
// The functions here are far fewer than those in the cpm_bdos.go files.
func (cpm *CPM) BiosHandler(val uint8) {

	// Lookup the handler
	handler, ok := cpm.BIOSSyscalls[val]

	// If it doesn't exist we don't have it implemented.
	if !ok {
		cpm.Logger.Error("Unimplemented BIOS syscall",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)))
	}

	// Otherwise invoke it, and look for any error
	err := handler.Handler(cpm)

	// If there was an error then record it for later notice.
	if err != nil {
		// record the error
		cpm.ioErr = err
		// halt processing.
		cpm.CPU.HALT = true
	}
}
