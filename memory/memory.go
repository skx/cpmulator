// Package memory is a package that provides the 64k of RAM
// within which the emulator executes its programs.
package memory

import "os"

// Memory is our structure for representing the 64k of RAM
// that we run our programs within.
type Memory struct {
	buf [65536]uint8
}

// FillRange fills an area of memory with the given byte
func (m *Memory) FillRange(addr uint16, size int, char uint8) {
	for size > 0 {
		m.buf[addr] = char
		addr++
		size--
	}
}

// Get returns a byte at addr of memory.
func (m *Memory) Get(addr uint16) uint8 {
	return m.buf[addr]
}

// GetRange returns the contents of a given range
func (m *Memory) GetRange(addr uint16, size int) []uint8 {
	var ret []uint8
	for size > 0 {
		ret = append(ret, m.buf[addr])
		addr++
		size--
	}
	return ret
}

// GetU16 returns a word from the given address of memory.
func (m *Memory) GetU16(addr uint16) uint16 {
	l := m.Get(addr)
	h := m.Get(addr + 1)
	return (uint16(h) << 8) | uint16(l)
}

// LoadFile loads a file into the RAM, at the specified offset.
//
// Before loading the file all memory is filled with 0x00 (NOP).
func (m *Memory) LoadFile(offset uint16, name string) error {

	// Fill the 64k with NOP instructions
	for i := range m.buf {
		m.buf[i] = 0x00
	}

	// Load the binary
	prog, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	// Put it into the starting locatioe.
	m.SetRange(offset, prog...)

	return nil
}

// Set sets a byte at addr of memory.
func (m *Memory) Set(addr uint16, value uint8) {
	m.buf[addr] = value
}

// SetRange copies bytes from the given data to the specified
// starting address in RAM.
func (m *Memory) SetRange(addr uint16, data ...uint8) {
	copy(m.buf[int(addr):int(addr)+len(data)], data)
}
