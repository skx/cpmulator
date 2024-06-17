// This file implements the BIOS function-calls.
//
// These are documented online:
//
// * https://www.seasip.info/Cpm/bios.html

package cpm

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/version"
)

// BiosSysCallBoot handles a warm/cold boot.
func BiosSysCallBoot(cpm *CPM) error {

	// Set entry-point to 0x0000 which will result in
	// a boot-trap.
	cpm.CPU.States.PC = 0x0000
	return nil
}

// BiosSysCallConsoleStatus should return 0x00 if there is no input
// pending, otherwise 0xFF.
func BiosSysCallConsoleStatus(cpm *CPM) error {

	if cpm.input.PendingInput() {
		cpm.CPU.States.AF.Hi = 0xFF
	} else {
		cpm.CPU.States.AF.Hi = 0x00
	}

	return nil
}

// BiosSysCallConsoleInput should block for a single character of input,
// and return the character pressed in the A-register.
func BiosSysCallConsoleInput(cpm *CPM) error {

	out, err := cpm.input.BlockForCharacterNoEcho()
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
	cpm.output.PutCharacter(c)

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

// BiosSysCallReserved1 is a helper to get/set the values of the CPM interpreter from
// within the system.  Neat.
func BiosSysCallReserved1(cpm *CPM) error {

	// H is used to specify the function.
	//
	// HL == 0
	//    Are we running under CPMUlater?  We always say yes!
	//
	// HL == 1
	//    C == 0xff to get the ctrl-c count
	//    C != 0xff to set the ctrl-c count
	//
	// HL == 2
	//    DE points to a string containing the console driver to use.
	//
	// HL == 3
	//    DE points to a string containing the CCP to use.
	//
	// HL == 4
	//    C == 0 - Quiet mode on.
	//    C != 0 - Quiet mode off
	//
	hl := cpm.CPU.States.HL.U16()
	c := cpm.CPU.States.BC.Lo
	de := cpm.CPU.States.DE.U16()

	//
	// Helper to read a null/space terminated string from
	// memory.
	//
	// Here because our custom syscalls read a string when
	// setting both CCP and DisplayDriver.
	//
	getStringFromMemory := func(addr uint16) string {
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

	switch hl {

	case 0x0000:
		// Magic values in the registers
		cpm.CPU.States.HL.Hi = 'S'
		cpm.CPU.States.HL.Lo = 'K'
		cpm.CPU.States.AF.Hi = 'X'

		// Get our version
		vers := version.GetVersionBanner()

		// Fill the DMA area with NULL bytes
		addr := cpm.dma

		end := addr + uint16(127)
		for end > addr {
			cpm.Memory.Set(end, 0x00)
			end--
		}

		// now populate with our name/version/information
		for i, c := range vers {
			cpm.Memory.Set(addr+uint16(i), uint8(c))
		}
		return nil

	case 0x0001:
		if c == 0xFF {
			cpm.CPU.States.AF.Hi = uint8(cpm.input.GetInterruptCount())
		} else {
			cpm.input.SetInterruptCount(int(c))
		}

	case 0x0002:

		if de == 0x0000 {
			// Fill the DMA area with NULL bytes
			addr := cpm.dma

			end := addr + uint16(127)
			for end > addr {
				cpm.Memory.Set(end, 0x00)
				end--
			}

			// now populate with our console driver
			str := cpm.output.GetName()
			for i, c := range str {
				cpm.Memory.Set(addr+uint16(i), uint8(c))
			}
			return nil
		}

		// Get the string pointed to by DE
		str := getStringFromMemory(de)

		// Output driver needs to be created
		driver, err := consoleout.New(str)

		// If it failed we're not going to terminate the syscall, or
		// the emulator, just ignore the attempt.
		if err != nil {
			fmt.Printf("%s", err)
			return nil
		}

		old := cpm.output.GetName()
		cpm.output = driver

		// when running quietly don't show any output
		if cpm.quiet {
			return nil
		}

		if old != str {
			fmt.Printf("Console driver changed from %s to %s.\n", old, driver.GetName())
		} else {
			fmt.Printf("Console driver is already %s, making no change.\n", str)
		}

	case 0x0003:

		if de == 0x0000 {
			// Fill the DMA area with NULL bytes
			addr := cpm.dma

			end := addr + uint16(127)
			for end > addr {
				cpm.Memory.Set(end, 0x00)
				end--
			}

			// now populate with our CCP
			str := cpm.ccp
			for i, c := range str {
				cpm.Memory.Set(addr+uint16(i), uint8(c))
			}
			return nil
		}

		// Get the string pointed to by DE
		str := getStringFromMemory(de)

		// See if the CCP exists
		entry, err := ccp.Get(str)
		if err != nil {
			fmt.Printf("Invalid CCP name %s\n", str)
			return nil
		}

		// old value
		old := cpm.ccp
		cpm.ccp = str

		// when running quietly don't show any output
		if cpm.quiet {
			return nil
		}

		if old != str {
			fmt.Printf("CCP changed to %s [%s] Size:0x%04X Entry-Point:0x%04X\n", str, entry.Description, len(entry.Bytes), entry.Start)
		} else {
			fmt.Printf("CCP is already %s, making no change.\n", str)
		}

	case 0x0004:

		// if C == 00
		//   Set the quiet flag to be true
		//
		// If C == 0xFF
		//   Return the statues of the flag in C (0 = quiet, 1 = non-quiet)
		//
		// Set the quiet flag
		if c == 0x00 {
			cpm.SetQuiet(true)
		}
		if c == 0x01 {
			cpm.SetQuiet(false)
		}
		if c == 0xFF {
			if cpm.GetQuiet() {
				cpm.CPU.States.BC.Lo = 0x00
			} else {
				cpm.CPU.States.BC.Lo = 0x01
			}

		}

	case 0x0005:
		width, height, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		cpm.CPU.States.HL.Hi = uint8(height)
		cpm.CPU.States.HL.Lo = uint8(width)

	default:
		fmt.Printf("Unknown custom BIOS function HL:%04X, ignoring", hl)
	}

	return nil
}
