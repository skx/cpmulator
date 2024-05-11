// Package io is designed to collect the code that reads from STDIN.
//
// There are, broadly speaking, two things we need to do here:
//
// * See if there is any available (single-character) input.
//
// * Read a single byte of input.
//
// This package is explicitly not used for _line_ based IO (yet).
package io

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/term"
)

// IO is used to hold our package state
type IO struct {
}

// New is our package constructor.
func New() *IO {
	return &IO{}
}

// BlockForCharacter returns the next character from the console, blocking until
// one is available.
//
// NOTE: This function should not echo keystrokes which are entered.
func (io *IO) BlockForCharacter() (byte, error) {

	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	defer func() {
		//  display entered characters on the screen
		exec.Command("stty", "-F", "/dev/tty", "echo").Run()

	}()

	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0x00, fmt.Errorf("error making raw terminal %s", err)
	}

	// read only a single byte
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return 0x00, fmt.Errorf("error reading a byte from stdin %s", err)
	}

	// restore the state of the terminal to avoid mixing RAW/Cooked
	err = term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		return 0x00, fmt.Errorf("error restoring terminal state %s", err)
	}

	// Return the character we read
	return b[0], nil
}

// BlockForCharacterWithEcho returns the next character from the console,
// blocking until one is available.
//
// NOTE: Characters should be echo'd as they are input.
func (io *IO) BlockForCharacterWithEcho() (byte, error) {

	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 0x00, fmt.Errorf("error making raw terminal %s", err)
	}

	// read only a single byte
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	if err != nil {
		return 0x00, fmt.Errorf("error reading a byte from stdin %s", err)
	}

	// restore the state of the terminal to avoid mixing RAW/Cooked
	err = term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		return 0x00, fmt.Errorf("error restoring terminal state %s", err)
	}

	fmt.Printf("%c", b[0])
	return b[0], nil
}

// GetCharOrNull retrieves any input that is available, returning NULL
// if nothing is pending.
//
// Note: We should disable echo in this function.
func (io *IO) GetCharOrNull() (uint8, error) {

	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	defer func() {
		//  display entered characters on the screen
		exec.Command("stty", "-F", "/dev/tty", "echo").Run()

	}()

	// Set STDIN to be non-blocking.
	if err1 := syscall.SetNonblock(0, true); err1 != nil {

		return 0x00, fmt.Errorf("failed to set non-blocking stdin %s", err1)
	}

	// Switch STDIN into 'raw' mode.
	oldState, err2 := term.MakeRaw(int(os.Stdin.Fd()))
	if err2 != nil {
		return 0x00, fmt.Errorf("error making raw terminal %s", err2)
	}

	// We'll read only a single byte of input, into this buffer
	b := make([]byte, 1)

	// NOTE: This doesn't work without the non-blocking mode having been
	// set previously.
	_ = os.Stdin.SetDeadline(time.Now().Add(time.Millisecond * 10))

	// Try the read
	n, err := os.Stdin.Read(b)

	// restore the state of the terminal to avoid mixing RAW/Cooked
	err3 := term.Restore(int(os.Stdin.Fd()), oldState)
	if err3 != nil {
		return 0x00, fmt.Errorf("error restoring terminal state %s", err3)
	}

	// restore the non-blocking state
	if err4 := syscall.SetNonblock(0, false); err4 != nil {
		return 0x00, fmt.Errorf("failed to restore non-blocking %s", err4)
	}

	// If we got a timeout, or some other error, then we assume
	// there is no character pending
	if err != nil {
		return 0x00, nil
	}

	// If we read one byte, as we hoped to do then we can record
	// that byte, and return it
	if n == 1 {
		return b[0], nil
	}

	// Can't happen?
	return 0x00, fmt.Errorf("impossible condition")
}
