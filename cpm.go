// cpm.go - Implement the BIOS callbacks

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/koron-go/z80"
	"golang.org/x/term"
)

// currentDrive contains the currently selected drive.
// Valid values are 00-15, where
// 0  -> A:
// 1  -> B:
// 15 -> P:
var currentDrive uint8

// userNumber contains the current user number.
//
// Valid values are 00-15
var userNumber uint8

// runCPM loads and executes the given .COM file
func runCPM(path string, args []string) error {

	// Create 64K of memory, full of NOPs
	m := new(Memory)

	// Load our binary into it
	err := m.LoadFile(path)
	if err != nil {
		return (fmt.Errorf("failed to load %s: %s", path, err))
	}

	// Convert our array of CLI arguments to a string
	cli := strings.Join(args, " ")
	cli = strings.TrimSpace(cli)

	//
	// By default any command-line arguments need to be copied
	// to 0x0080 - as a pascal-prefixed string.
	//
	// If there are arguments the default FCBs need to be updated
	// appropriately too.
	//
	// Default to emptying the FCBs and leaving the CLI args empty.
	//
	// DMA area / CLI Args
	m.FillRange(0x0080, 32, 0x00)
	// FCB1
	m.FillRange(0x005C+1, 11, ' ')
	// FCB2
	m.FillRange(0x006C+1, 11, ' ')

	// Poke in the CLI argument as a Pascal string.
	// (i.e. length prefixed)
	if len(cli) > 0 {

		// Setup the CLI arguments - these are set as a pascal string
		// (i.e. first byte is the length, then the data follows).
		m.put(0x0080, uint8(len(cli)))
		for i, c := range cli {
			m.put(0x0081+uint16(i), uint8(c))
		}

		// Now setup FCB1
		if len(args) > 0 {
			// TODO handle this properly
			for i, c := range args[0] {
				m.put(0x005C+1+uint16(i), uint8(c))
			}
		}

		// Now setup FCB2
		if len(args) > 1 {
			// TODO handle this properly
			for i, c := range args[1] {
				m.put(0x006C+1+uint16(i), uint8(c))
			}
		}
	}

	// Create the CPU, pointing to our memory
	// starting point for PC will be the binary entry-point
	cpu := z80.CPU{
		States: z80.States{SPR: z80.SPR{PC: 0x100}},
		Memory: m,
	}

	// Setup a breakpoint on 0x0005
	// That's the BIOS entrypoint
	cpu.BreakPoints = map[uint16]struct{}{}
	cpu.BreakPoints[0x05] = struct{}{}

	// Helper to return from a CALL instruction
	//
	// Pop the return address from the stack and
	// return execution there.
	callReturn := func() {
		// Return from call
		cpu.PC = m.GetU16(cpu.SP)
		// pop stack back.  Fun
		cpu.SP += 2
	}

	// Run forever :)
	for {

		// Run until we hit an error
		err := cpu.Run(context.Background())

		// No error?  Then end - the CPU hit a HALT.
		if err == nil {
			return nil
		}

		// An error which wasn't a breakpoint?  Give up
		if err != z80.ErrBreakPoint {
			return fmt.Errorf("unexpected error running CPU %s", err)
		}

		// OK we have a breakpoint error to handle.
		//
		// That means we have a CP/M BIOS function to emulate.
		function := cpu.States.BC.Lo

		// 0x00 - Exit!
		if function == 0x00 {
			// EXIT!
			return nil
		}

		// 0x01 - Read a key, result returned in A
		if function == 0x01 {

			// switch stdin into 'raw' mode
			oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("error making raw terminal %s", err)
			}

			// read only a single byte
			b := make([]byte, 1)
			_, err = os.Stdin.Read(b)
			if err != nil {
				return fmt.Errorf("error reading a byte from stdin %s", err)
			}

			// restore the state of the terminal to avoid mixing RAW/Cooked
			term.Restore(int(os.Stdin.Fd()), oldState)

			// Return the character
			cpu.States.AF.Hi = b[0]

			callReturn()
			continue
		}

		// 0x02 - Print a character, from E.
		if function == 0x02 {
			fmt.Printf("%c", (cpu.States.DE.Lo))
			callReturn()
			continue
		}

		// 0x09 - Write a string of $-terminated text - address in DE
		if function == 0x09 {
			addr := cpu.States.DE.U16()

			c := m.Get(addr)
			for c != '$' {
				fmt.Printf("%c", c)
				addr++
				c = m.Get(addr)
			}
			callReturn()
			continue
		}

		// 0x0A - Read line of input - buffer in DE
		if function == 0x0A {

			addr := cpu.States.DE.U16()

			text, err := reader.ReadString('\n')
			if err != nil {
				return (fmt.Errorf("error reading from STDIN:%s", err))
			}

			// remove trailing newline
			text = strings.TrimSuffix(text, "\n")

			// addr[0] is the size of the input buffer
			// addr[1] should be the size of input read, set it:
			cpu.Memory.Set(addr+1, uint8(len(text)))

			// addr[2+] should be the text
			i := 0
			for i < len(text) {
				cpu.Memory.Set(uint16(addr+2+uint16(i)), text[i])
				i++
			}

			callReturn()
			continue
		}

		// 14 (DRV_SET) - Select disc
		if function == 0x0E {
			// The drive number passed to this routine is 0 for A:, 1 for B: up to 15 for P:.
			currentDrive = (cpu.States.AF.Hi & 0x0F)

			// Success means we return 0x00 in A
			cpu.States.AF.Hi = 0x00

			callReturn()
			continue
		}
		// 25 (DRV_GET)  - Return current drive
		if function == 0x19 {

			cpu.States.AF.Hi = currentDrive

			callReturn()
			continue
		}

		// 32 (F_USERNUM) - get/set user number
		if function == 0x20 {

			// We're either setting or getting
			//
			// If the value is 0xFF we return it, otherwise we set
			if cpu.States.DE.Lo != 0xFF {

				// Set the number - masked, because valid values are 0-15
				userNumber = (cpu.States.DE.Lo & 0x0F)
			}

			// Return the current number, which might have changed
			cpu.States.AF.Hi = userNumber

			callReturn()
			continue
		}

		fmt.Printf("Breakpoint called %04X - Unimplemented BIOS call C:%02X\n", cpu.States.PC, cpu.States.BC.Lo)
	}
}
