package cpm

import (
	"fmt"
	"log/slog"

	cpmio "github.com/skx/cpmulator/io"
)

// BiosHandler is involved when a BIOS syscall needs to be executed,
// which is handled via trapping RST instructions and using a short
// trampoline.
//
// The functions here are far fewer than those in the cpm_bdos.go files.
func (cpm *CPM) BiosHandler(val uint8) {

	switch val {
	case 00:
		cpm.Logger.Info("BIOS syscall 0x00",
			slog.String("BIOS", "BOOT"))

		// Set entry-point to 0x0000 which will result in
		// a boot-trap.
		cpm.CPU.States.PC = 0x0000
	case 01:
		cpm.Logger.Info("BIOS syscall 0x01",
			slog.String("BIOS", "WBOOT"))

		// Set entry-point to 0x0000 which will result in
		// a boot-trap.
		cpm.CPU.States.PC = 0x0000
	case 02:
		// Returns its status in A; 0 if no character is ready, 0FFh if one is.
		cpm.Logger.Info("BIOS syscall 0x02",
			slog.String("BIOS", "CONST"))

		// Nothing pending - FAKE
		cpm.CPU.States.AF.Hi = 0x00

	case 03:
		cpm.Logger.Info("BIOS syscall 0x03",
			slog.String("BIOS", "CONIN"))
		// Wait until the keyboard is ready to provide a character, and return it in A.
		// Use our I/O package
		obj := cpmio.New(cpm.Logger)

		out, err := obj.BlockForCharacter()
		if err != nil {
			// record the error
			cpm.ioErr = err
			// halt processing.
			cpm.CPU.HALT = true
		}
		cpm.CPU.States.AF.Hi = out

	case 04:
		cpm.Logger.Info("BIOS syscall 0x04",
			slog.String("BIOS", "CONOUT"))

		// Write the character in C to the screen.
		c := cpm.CPU.States.BC.Lo
		cpm.outC(c)
	case 05:
		cpm.Logger.Info("BIOS syscall 0x05",
			slog.String("BIOS", "LIST"))

		// Write the character in C to the printer
		err := cpm.prnC(cpm.CPU.States.BC.Lo)
		if err != nil {
			// record the error
			cpm.ioErr = err
			// halt processing.
			cpm.CPU.HALT = true
		}

	default:
		cpm.Logger.Error("Unimplemented BIOS syscall",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)))
	}
}
