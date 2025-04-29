// This file implements the BIOS function-calls.
//
// These are documented online:
//
// * https://www.seasip.info/Cpm/bios.html

package cpm

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/version"
)

// BiosSysCallColdBoot handles a cold boot.
func BiosSysCallColdBoot(cpm *CPM) error {
	// Reset all registers
	cpm.CPU.AF.SetU16(0)
	cpm.CPU.BC.SetU16(0)
	cpm.CPU.DE.SetU16(0)
	cpm.CPU.HL.SetU16(0)

	// Reset the stack on a cold-boot.
	cpm.CPU.SP = 0xFFFF

	// Reset the drive and user-number on a cold-boot.
	cpm.currentDrive = 0
	cpm.Memory.Set(0x0004, 0)

	// DMA gets reset
	cpm.dma = DefaultDMAAddress

	return ErrBoot
}

// BiosSysCallWarmBoot handles a warm boot.
func BiosSysCallWarmBoot(cpm *CPM) error {

	// Reset all registers
	cpm.CPU.AF.SetU16(0)
	cpm.CPU.BC.SetU16(0)
	cpm.CPU.DE.SetU16(0)
	cpm.CPU.HL.SetU16(0)

	// DMA gets reset
	cpm.dma = DefaultDMAAddress

	return ErrBoot
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
	de := cpm.CPU.States.DE.U16()
	c := cpm.CPU.States.BC.Lo

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
		for x != 0x00 {
			str += string(x)
			addr++
			x = cpm.Memory.Get(addr)
		}

		// Trim leading and trailing whitespace
		str = strings.TrimSpace(str)

		// Lower-case because the CCP will upper-case CLI arguments
		return strings.ToLower(str)
	}

	// Several of our routines are called with DE set to NULL,
	// and that means "store the value in the DMA area".
	//
	// This routine resets the DMA area to NULL, and then sets
	// the passed in value there.
	setStringInDMA := func(str string) {

		// Fill the DMA area with NULL bytes
		cpm.Memory.FillRange(cpm.dma, 127, 0x00)

		// now populate with our console driver.
		for i, c := range str {
			cpm.Memory.Set(cpm.dma+uint16(i), uint8(c))
		}
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

		// Set it in the DMA area
		setStringInDMA(vers)

		return nil

	// Get/Set the ctrl-c flag
	case 0x0001:
		if c == 0xFF {
			cpm.CPU.States.AF.Hi = uint8(cpm.input.GetInterruptCount())
		} else {
			cpm.input.SetInterruptCount(int(c))
		}

	// Get/Set the output driver.
	case 0x0002:

		// If DE is null then we're just being asked to return
		// the current value of the driver.
		if de == 0x0000 {
			setStringInDMA(cpm.output.GetName())
			return nil
		}

		// DE is not-null so we're going to try to change to the given
		// value.  Get the value
		str := getStringFromMemory(de)

		if str == "" {
			cpm.output.WriteString("Ignoring empty parameter.\r\n")
			return nil
		}

		// Get the old value
		old := cpm.output.GetName()

		// Is there a change?
		if old == str {
			cpm.output.WriteString("The output driver is already " + str + ", doing nothing.\r\n")
			return nil
		}

		// Create the new output driver
		driver, err := consoleout.New(str)

		// If it failed we're not going to terminate the syscall, or
		// the emulator, just ignore the attempt.
		if err != nil {
			cpm.output.WriteString("Changing output driver failed, " + err.Error() + ".\r\n")
			return nil
		}

		// Do we have options?  If so show them too
		options := ""
		val := strings.Split(str, ":")
		nm := str
		if len(val) == 2 {
			if len(val[1]) > 0 {
				options = " with options '" + val[1] + "'"
				nm = val[0]
			}
		}

		cpm.output = driver
		if nm != old {

			cpm.output.WriteString("The output driver has been changed from " + old + " to " + driver.GetName() + options + ".\r\n")
			return nil
		}

		if len(val) == 2 {
			cpm.output.WriteString("Options changed to " + val[1] + " for " + driver.GetName() + ".\r\n")
		}
		return nil

	// Get/Set the CCP
	case 0x0003:

		// If DE is null then we're just being asked to return
		// the current CCP name
		if de == 0x0000 {
			setStringInDMA(cpm.ccp)
			return nil
		}

		// Get the string pointed to by DE
		str := getStringFromMemory(de)

		if str == "" {
			cpm.output.WriteString("Ignoring empty parameter.\r\n")
			return nil
		}

		// If there is no change do nothing
		if str == cpm.ccp {
			cpm.output.WriteString("CCP is already set to " + str + ", doing nothing.\r\n")
			return nil
		}

		// See if the CCP exists
		entry, err := ccp.Get(str)
		if err != nil {
			cpm.output.WriteString("Error changing CCP to " + str + ", " + err.Error() + "\r\n")
			return nil
		}

		// old value
		old := cpm.ccp
		cpm.ccp = str

		if old != str {
			cpm.output.WriteString(fmt.Sprintf("CCP changed to %s [%s] Size:0x%04X Entry-Point:0x%04X\r\n", str, entry.Description, len(entry.Bytes), entry.Start))
		}
		return nil

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

	// Get/Set the input driver.
	case 0x0007:

		// If DE is null then we're just being asked to return
		// the current value of the driver.
		if de == 0x0000 {
			setStringInDMA(cpm.input.GetName())
			return nil
		}

		// DE is not-null so we're going to try to change to the given
		// value.  Get the value.
		str := getStringFromMemory(de)

		if str == "" {
			cpm.output.WriteString("Ignoring empty parameter.\r\n")
			return nil
		}

		// Get the old value
		oldName := cpm.input.GetName()

		// Is there a change?
		if oldName == str {
			cpm.output.WriteString("The input driver is already " + str + ", doing nothing.\r\n")
			return nil
		}

		// Okay now we tear-down the old driver, and we explicitly do this
		// before we create the new one.
		//
		// This might mean we have console output going to a weird place if
		// we have failures..
		_ = cpm.input.TearDown()

		// Create the new driver
		driver, err := consolein.New(str)
		if err != nil {
			cpm.output.WriteString("Error creating the new driver, " + err.Error() + ".\r\n")
			return err
		}

		// We need to setup the new driver.
		//
		// If this fails we also abort the change-attempt.
		err = driver.Setup()
		if err != nil {
			cpm.output.WriteString("Error setting up the new driver, " + err.Error() + ".\r\n")
			return err
		}

		// Do we have options?  If so show them too
		options := ""
		val := strings.Split(str, ":")
		nm := str
		if len(val) == 2 {
			if len(val[1]) > 0 {
				options = " with options '" + val[1] + "'"
				nm = val[0]
			}
		}

		cpm.input = driver
		if nm != oldName {
			cpm.output.WriteString("Input driver changed from " + oldName + " to " + driver.GetName() + options + ".\r\n")
			return nil
		}

		if len(val) == 2 {
			cpm.output.WriteString("Options changed to " + val[1] + " for " + driver.GetName() + ".\r\n")
		}
		return nil

	// Set the host prefix
	case 0x0008:

		// If DE is null then we're just being asked to return
		// the current value of the prefix, if any.
		if de == 0x0000 {
			setStringInDMA(cpm.input.GetSystemCommandPrefix())
			return nil
		}

		// Get the string pointed to by DE
		str := getStringFromMemory(de)

		// set it
		if str == "/clear" {
			cpm.input.SetSystemCommandPrefix("")
		} else {
			cpm.input.SetSystemCommandPrefix(str)
		}

	// Disable extensions / filesystem
	case 0x0009:

		switch de {

		case 0x0001:
			// Just filesystem
			cpm.static = embed.FS{}
			cpm.output.WriteString("The embedded filesystem has been disabled.\r\n")

		case 0x0002:
			// Just BIOS
			cpm.BIOSSyscalls[31] = Handler{
				Desc:    "RESERVE1_DISABLED",
				Handler: BiosSysCallReserved1NOP,
				Fake:    true,
			}
			cpm.output.WriteString("The BIOS extensions have been disabled.\r\n")

		case 0x0003:
			// Both
			cpm.static = embed.FS{}
			cpm.BIOSSyscalls[31] = Handler{
				Desc:    "RESERVE1_DISABLED",
				Handler: BiosSysCallReserved1NOP,
				Fake:    true,
			}
			cpm.output.WriteString("The embedded filesystem and our BIOS extensions have been disabled.\r\n")

		case 0x0004:
			// Both, but quietly.
			cpm.static = embed.FS{}
			cpm.BIOSSyscalls[31] = Handler{
				Desc:    "RESERVE1_DISABLED",
				Handler: BiosSysCallReserved1NOP,
				Fake:    true,
			}
		default:
			cpm.output.WriteString(fmt.Sprintf("Unknown action for the disable BIOS function: %04X\r\n", de))
		}

		// Set the printer-logfile
	case 0x000a:

		// If DE is null then we're just being asked to return
		// the current value of filename.
		if de == 0x0000 {
			setStringInDMA(cpm.prnPath)
			return nil
		}

		// Get the string pointed to by DE
		str := getStringFromMemory(de)

		// set it
		cpm.prnPath = str
		return nil

	default:
		cpm.output.WriteString(fmt.Sprintf("Ignoring unknown custom BIOS function HL:%04X\r\n", hl))
	}

	return nil
}

// BiosSysCallReserved1NOP is used when one of our extended BIOS syscalls was used to
// disable the use of additional future syscalls.
func BiosSysCallReserved1NOP(cpm *CPM) error {
	return nil
}
