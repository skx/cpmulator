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

	"github.com/koron-go/z80"
	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/version"
)

// BiosSysCallColdBoot handles a cold boot.
func BiosSysCallColdBoot(cpm *CPM) error {

	// Set entry-point to 0x0000 which will result in
	// a boot-trap.
	cpm.CPU.States.PC = 0x0000
	return nil
}

// BiosSysCallWarmBoot handles a warm boot.
func BiosSysCallWarmBoot(cpm *CPM) error {

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
	cpm.CPU.States.AF.Hi = out
	return err
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

// BiosSysCallReserved1 is a helper to get/set the values of the CPM interpreter from
// within the system.  Neat.
func BiosSysCallReserved1(cpm *CPM) error {

	// H is used to specify the function.
	//
	// HL == 0
	//    Are we running under cpmulator?  We always say yes!
	//
	// HL == 1
	//    C == 0xff to get the ctrl-c count
	//    C != 0xff to set the ctrl-c count
	// ...
	//
	hl := cpm.CPU.States.HL.U16()
	c := cpm.CPU.States.BC.Lo
	de := cpm.CPU.States.DE.U16()

	//
	// Helper to read a null/space terminated string from
	// memory.
	//
	// Here because several of our custom syscalls need to
	// read a string from the caller.
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

	// Is this a CPMUlator?
	case 0x0000:
		// Magic values in the registers
		cpm.CPU.States.HL.Hi = 'S'
		cpm.CPU.States.HL.Lo = 'K'
		cpm.CPU.States.AF.Hi = 'X'

		// Get our version
		vers := version.GetVersionBanner()
		vers = strings.ReplaceAll(vers, "\n", "\n\r")

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

	// Get/Set the ctrl-c flag
	case 0x0001:
		if c == 0xFF {
			cpm.CPU.States.AF.Hi = uint8(cpm.input.GetInterruptCount())
		} else {
			cpm.input.SetInterruptCount(int(c))
		}

	// Get/Set the console driver.
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

		if old != str {
			fmt.Printf("Console driver changed from %s to %s.\n", old, driver.GetName())
		}

	// Get/Set the CCP
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

		if old != str {
			fmt.Printf("CCP changed to %s [%s] Size:0x%04X Entry-Point:0x%04X\n", str, entry.Description, len(entry.Bytes), entry.Start)
		}

	// Get/Set the quiet flag
	case 0x0004:

		// Retired.
		return nil

	// Get terminal size in HL
	case 0x0005:
		width, height, err := term.GetSize(int(os.Stdin.Fd()))

		// This will fail on tests, and Windows probably.
		cpm.CPU.States.HL.Hi = uint8(height)
		cpm.CPU.States.HL.Lo = uint8(width)

		if err != nil {
			return err
		}

	// Get/Set the debug-flag
	case 0x0006:

		// if C == 00
		//   Disable debug
		//
		// if C == 01
		//   Enable debug
		//
		// If C == 0xFF
		//   Return the statues of the flag in C.
		//
		if c == 0x00 {
			cpm.simpleDebug = false
		}
		if c == 0x01 {
			cpm.simpleDebug = true
		}
		if c == 0xFF {
			if cpm.simpleDebug {
				cpm.CPU.States.BC.Lo = 0x01
			} else {
				cpm.CPU.States.BC.Lo = 0x00
			}
		}

	default:
		fmt.Printf("Unknown custom BIOS function HL:%04X, ignoring", hl)
	}

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
		slog.Error("Unimplemented BIOS syscall",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)))

		// record the error
		cpm.biosErr = ErrUnimplemented
		// halt processing.
		cpm.CPU.HALT = true

		// stop now.
		return
	}

	if !handler.Noisy {

		// show the function being invoked.
		if cpm.simpleDebug {
			fmt.Printf("%03d %s\n", val, handler.Desc)
		}

		// Log the call we're going to make
		slog.Info("BIOS",
			slog.String("name", handler.Desc),
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)),
			slog.Group("registers",
				slog.String("AF", fmt.Sprintf("%04X", cpm.CPU.States.AF.U16())),
				slog.String("BC", fmt.Sprintf("%04X", cpm.CPU.States.BC.U16())),
				slog.String("DE", fmt.Sprintf("%04X", cpm.CPU.States.DE.U16())),
				slog.String("HL", fmt.Sprintf("%04X", cpm.CPU.States.HL.U16()))))
	}

	// Otherwise invoke it, and look for any error
	err := handler.Handler(cpm)

	// If there was an error then record it for later notice.
	if err != nil {
		// record the error
		cpm.biosErr = err
		// halt processing.
		cpm.CPU.HALT = true
	}

	// If A == 0x00 then we set the zero flag
	if cpm.CPU.States.AF.Hi == 0x00 {
		cpm.CPU.SetFlag(z80.FlagZ)
	} else {
		cpm.CPU.ResetFlag(z80.FlagZ)
	}

}
