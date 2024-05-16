// Package io is designed to collect the code that reads from STDIN.
//
// There are three functions we need to care about:
//
// * Block for a character, and return it.
//
// * Block for a character, and return it, but disable echo first.
//
// * Read a single line of input.
//
// There are functions for polling console status in CP/M, however it
// seems to work just fine if we fake their results - which means this
// package is simpler than it would otherwise need to be.
package io

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

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
