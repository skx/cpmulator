// This file implements the BIOS function-calls.
//
// These are documented online:
//
// * https://www.seasip.info/Cpm/bios.html

package cpm

import (
	"fmt"
	"log/slog"
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

// BiosSysCallConsoleInput should block for a single character of input,
// and return the character pressed in the A-register.
func BiosSysCallConsoleInput(cpm *CPM) error {

	out, err := cpm.input.BlockForCharacterWithEcho()
	if err != nil {
		return err
	}

	cpm.CPU.States.AF.Hi = out
	return nil
}

// BiosSysCallConsoleOutput should write a single character, in the C-register,
// to the console.
func BiosSysCallConsoleOutput(cpm *CPM) error {

	// Write the character in C to the screen.
	c := cpm.CPU.States.BC.Lo
	cpm.outC(c)

	return nil
}

// BiosSysCallPrintChar should print the specified character, in the C-register,
// to the printer.  We fake that and write to a file instead.
func BiosSysCallPrintChar(cpm *CPM) error {

	// Write the character in C to the printer
	c := cpm.CPU.States.BC.Lo

	// Write the character to the printer
	err := cpm.prnC(c)
	return err
}

// BiosSysCallPrinterStatus returns status of current printer device.
//
// This is fake, and always returns "ready".
func BiosSysCallPrinterStatus(cpm *CPM) error {

	// Ready
	cpm.CPU.States.AF.Hi = 0xFF
	return nil
}

// BiosSysCallScreenOutputStatus returns status of current screen output device.
//
// This is fake, and always returns "ready".
func BiosSysCallScreenOutputStatus(cpm *CPM) error {

	// Ready
	cpm.CPU.States.AF.Hi = 0xFF
	return nil
}

// BiosSysCallAuxInputStatus returns status of current auxiliary input device.
//
// This is fake, and always returns "ready".
func BiosSysCallAuxInputStatus(cpm *CPM) error {

	// Ready
	cpm.CPU.States.AF.Hi = 0xFF
	return nil
}

// BiosSysCallAuxOutputStatus returns status of current auxiliary output device.
//
// This is fake, and always returns "ready".
func BiosSysCallAuxOutputStatus(cpm *CPM) error {

	// Ready
	cpm.CPU.States.AF.Hi = 0xFF
	return nil
}

// BiosHandler is involved when a BIOS syscall needs to be executed,
// which is handled via a small trampoline.
//
// These are looked up in the BIOSSyscalls map.
func (cpm *CPM) BiosHandler(val uint8) {

	// Lookup the handler
	handler, ok := cpm.BIOSSyscalls[val]

	// If it doesn't exist we don't have it implemented.
	if !ok {
		cpm.Logger.Error("Unimplemented BIOS syscall",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)))

		// record the error
		cpm.biosErr = ErrUnimplemented
		// halt processing.
		cpm.CPU.HALT = true

		// stop now.
		return
	}

	// Log the call we're going to make
	cpm.Logger.Info("BIOS",
		slog.String("name", handler.Desc),
		slog.Int("syscall", int(val)),
		slog.String("syscallHex", fmt.Sprintf("0x%02X", val)),
		slog.Group("registers",
			slog.String("A", fmt.Sprintf("%02X", cpm.CPU.States.AF.Hi)),
			slog.String("B", fmt.Sprintf("%02X", cpm.CPU.States.BC.Hi)),
			slog.String("C", fmt.Sprintf("%02X", cpm.CPU.States.BC.Lo)),
			slog.String("D", fmt.Sprintf("%02X", cpm.CPU.States.DE.Hi)),
			slog.String("E", fmt.Sprintf("%02X", cpm.CPU.States.DE.Lo)),
			slog.String("F", fmt.Sprintf("%02X", cpm.CPU.States.AF.Lo)),
			slog.String("H", fmt.Sprintf("%02X", cpm.CPU.States.HL.Hi)),
			slog.String("L", fmt.Sprintf("%02X", cpm.CPU.States.HL.Lo)),
			slog.String("BC", fmt.Sprintf("%04X", cpm.CPU.States.BC.U16())),
			slog.String("DE", fmt.Sprintf("%04X", cpm.CPU.States.DE.U16())),
			slog.String("HL", fmt.Sprintf("%04X", cpm.CPU.States.HL.U16()))))

	// Otherwise invoke it, and look for any error
	err := handler.Handler(cpm)

	// If there was an error then record it for later notice.
	if err != nil {
		// record the error
		cpm.biosErr = err
		// halt processing.
		cpm.CPU.HALT = true
	}
}
