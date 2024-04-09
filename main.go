package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/koron-go/z80"
)

// Start is an address where a program starts.
const Start = 0x0100

// Reader lets us get console input
var reader *bufio.Reader

// Page 0:
// ref. http://ngs.no.coocan.jp/doc/wiki.cgi/datapack?page=12%BE%CF+%B3%B0%C9%F4%A5%D7%A5%ED%A5%B0%A5%E9%A5%E0%A4%CE%B4%C4%B6%AD#p2
var bios0000 = []byte{
	0xc3, 0x03, 0xff, 0x00, 0x00, 0xc3, 0x06, 0xfe,
}

// source: _z80/minibios.asm
var biosFE06 = []byte{
	0x79, 0xfe, 0x02, 0x28, 0x05, 0xfe, 0x09, 0x28, 0x05, 0x76, 0x7b, 0xd3,
	0x00, 0xc9, 0x1a, 0xfe, 0x24, 0xc8, 0xd3, 0x00, 0x13, 0x18, 0xf7,
}

// page for stop code.
var biosFF03 = []byte{
	0x76,
}

// Memory provides 64K bytes array memory.
type Memory struct {
	buf [65536]uint8
}

// Set sets a byte at addr of memory.
func (m *Memory) Set(addr uint16, value uint8) {
	m.buf[addr] = value
}

// Get returns a byte at addr of memory.
func (m *Memory) Get(addr uint16) uint8 {
	return m.buf[addr]
}

func (m *Memory) readU16(addr uint16) uint16 {
	l := m.Get(addr)
	h := m.Get(addr + 1)
	return toU16(l, h)
}

// put puts "data" block from addr.
func (m *Memory) put(addr uint16, data ...uint8) {
	copy(m.buf[int(addr):int(addr)+len(data)], data)
}

// LoadFile loads a file from "Start" (0x0100) as program.
func (m *Memory) LoadFile(name string) error {
	prog, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	m.put(Start, prog...)
	return nil
}

func toU16(l, h uint8) uint16 {
	return (uint16(h) << 8) | uint16(l)
}

// runCPM loads and executes the given .COM file
func runCPM(path string) error {

	m := new(Memory)
	m.put(0x0000, bios0000...)
	m.put(0xfe06, biosFE06...)
	m.put(0xff03, biosFF03...)

	err := m.LoadFile(path)
	if err != nil {
		return (fmt.Errorf("failed to load %s: %s", path, err))
	}

	stt := z80.States{SPR: z80.SPR{PC: 0x100}}

	cpu := z80.CPU{
		States: stt,
		Memory: m,
	}

	// Breakpoints
	breakpoints := []uint16{
		// CP/M BIOS address
		0x05,
	}

	if len(breakpoints) > 0 {
		cpu.BreakPoints = map[uint16]struct{}{}
		for _, v := range breakpoints {
			cpu.BreakPoints[v] = struct{}{}
		}
	}

	for {
		err := cpu.Run(context.Background())
		if err != nil {
			if err == z80.ErrBreakPoint {

				// 0x00 - Exit!
				if cpu.States.BC.Lo == 0x00 {
					// EXIT!
					return nil
				}
				// 0x01 - Read a key, result returned in A
				if cpu.States.BC.Lo == 0x01 {
					// TODO: We're always returning "n" for no here.
					cpu.States.AF.Hi = 'n'

					// Return from call
					cpu.PC = m.readU16(cpu.SP)
					// pop stack back.  Fun
					cpu.SP += 2

					continue
				}

				// 0x01 - Print a character, from E.
				if cpu.States.BC.Lo == 0x02 {
					fmt.Printf("%c", (cpu.States.DE.Lo))
					continue
				}

				// 0x0A - Read line of input - buffer in DE
				if cpu.States.BC.Lo == 0x0A {

					text, _ := reader.ReadString('\n')
					cpu.Memory.Set(cpu.States.DE.U16()+1, uint8(len(text)))

					i := 0
					for i < len(text) {
						cpu.Memory.Set(uint16(cpu.States.DE.U16()+2+uint16(i)), text[i])
						i++
					}

					// Return from call
					cpu.PC = m.readU16(cpu.SP)
					// pop stack back.  Fun
					cpu.SP += 2
					continue
				}
				fmt.Printf("Breakpoint: %04X\n", cpu.States.PC)
				fmt.Printf("Unknown break code: %02X\n", cpu.States.BC.Lo)
			}
		}
	}

	return nil
}

func main() {

	// Ensure we've been given the name of a file
	if len(os.Args) != 2 {
		fmt.Printf("Usage: go-cpm path/to/file.com\n")
		return
	}

	// Populate the global reader
	reader = bufio.NewReader(os.Stdin)

	// Load the binary
	err := runCPM(os.Args[1])
	if err != nil {
		fmt.Printf("Error running %s: %s\n", os.Args[1], err)
	}
}
