// Package consolein handles the reading of console input
// for our emulator.
//
// The package supports the minimum required functionality
// we need - which boils down to reading a single character
// of input, with and without echo, and reading a line of text.
//
// Note that no output functions are handled by this package,
// it is exclusively used for input.
package consolein

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// Status is used to record our current state
type Status int

var (
	// Unknown means we don't know the status of echo/noecho
	Unknown Status = 0

	// Echo means that input will echo characters.
	Echo Status = 1

	// NoEcho means that input will not echo characters.
	NoEcho Status = 2

	// ErrInterrupted is returned if the user presses Ctrl-C when in our ReadLine function.
	ErrInterrupted = fmt.Errorf("INTERRUPTED")
)

// ConsoleIn holds our state
type ConsoleIn struct {
	// State holds our current echo state; either Echo, NoEcho, or Unknown.
	State Status
}

// New is our constructor.
func New() *ConsoleIn {
	t := &ConsoleIn{
		State: Unknown,
	}
	return t
}

// BlockForCharacterNoEcho returns the next character from the console, blocking until
// one is available.
//
// NOTE: This function should not echo keystrokes which are entered.
func (ci *ConsoleIn) BlockForCharacterNoEcho() (byte, error) {

	// Do we need to change state?  If so then do it.
	if ci.State != NoEcho {
		ci.disableEcho()
	}

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
func (ci *ConsoleIn) BlockForCharacterWithEcho() (byte, error) {

	// Do we need to change state?  If so then do it.
	if ci.State != Echo {
		ci.enableEcho()
	}

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
//
// NOTE: A user pressing Ctrl-C will be caught, and this will trigger the BDOS
// function to reboot.
func (ci *ConsoleIn) ReadLine(max uint8) (string, error) {

	// Do we need to change state?  If so then do it.
	if ci.State != Echo {
		ci.enableEcho()
	}

	// Text the user entered
	text := ""

	// count of consecutive Ctrl-C
	ctrlCount := 0

	// Hacky input-loop
	for {

		// Get a character, with no echo.
		x, err := ci.BlockForCharacterNoEcho()
		if err != nil {
			return "", err
		}

		// Ctrl-C ?
		//
		if x == 0x03 {
			ctrlCount += 1

			// Twice in a row will reboot
			if ctrlCount == 2 {
				return "", ErrInterrupted
			}
			continue
		}

		// Not a ctrl-c so reset our count
		ctrlCount = 0

		// Newline?
		if x == '\n' || x == '\r' {
			text += "\n"
			break
		}

		// Backspace / Delete?
		if x == '\b' || x == 127 {

			// remove the character from our text, and overwrite on the console
			if len(text) > 0 {
				text = text[:len(text)-1]
				fmt.Printf("\b \b")
			}
			continue
		}

		// If the user has entered the maximum then we'll
		// avoid further input
		if len(text) >= int(max) {
			continue
		}
		// Otherwise if it was a printable character we'll keep it.
		if unicode.IsPrint(rune(x)) {
			fmt.Printf("%c", x)
			text += string(x)
		}
	}

	// remove any trailing newline
	text = strings.TrimSuffix(text, "\n")

	// Too much entered?  Truncate the text.
	if len(text) > int(max) {
		text = text[:max]
	}

	// Return the text
	return text, nil
}

// Reset restores echo.
func (ci *ConsoleIn) Reset() {
	ci.enableEcho()
}

// disableEcho is the single place where we disable echoing.
func (ci *ConsoleIn) disableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	ci.State = NoEcho
}

// enableEcho is the single place where we enable echoing.
func (ci *ConsoleIn) enableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "echo").Run()
	ci.State = Echo
}
