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
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// IO is used to hold our package state
type IO struct {
	Logger *slog.Logger
}

// New is our package constructor.
func New(log *slog.Logger) *IO {
	return &IO{Logger: log}
}

// disableEcho is the single place where we disable echoing.
func (io *IO) disableEcho() {
	err := exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	if err != nil {
		io.Logger.Debug("disableEcho",
			slog.String("error", err.Error()))
	}
}

// enableEcho is the single place where we enable echoing.
func (io *IO) enableEcho() {
	err := exec.Command("stty", "-F", "/dev/tty", "echo").Run()
	if err != nil {
		io.Logger.Debug("enableEcho",
			slog.String("error", err.Error()))
	}
}

// Restore enables echoing.
func (io *IO) Restore() {
	io.enableEcho()
}

// BlockForCharacter returns the next character from the console, blocking until
// one is available.
//
// NOTE: This function should not echo keystrokes which are entered.
func (io *IO) BlockForCharacter() (byte, error) {

	io.disableEcho()

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

	io.enableEcho()

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

	io.disableEcho()

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
	_ = os.Stdin.SetDeadline(time.Now().Add(time.Millisecond * 50))

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

	return 0x00, fmt.Errorf("can't happen")
}

// ReadLine reads a line of input from the console, truncating to the
// length specified.  (The user can enter more than is allowed but no
// buffer-overruns will occur!)
//
// Note: We should enable echo in this function.
func (io *IO) ReadLine(max uint8) (string, error) {

	io.enableEcho()

	// Create a  reader
	reader := bufio.NewReader(os.Stdin)

	// Read a line of text
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading from STDIN:%s", err)
	}

	// remove any trailing newline
	text = strings.TrimSuffix(text, "\n")

	// Too much entered?  Truncate the text.
	if len(text) > int(max) {
		text = text[:max]
	}

	// Return the text
	return text, err
}
