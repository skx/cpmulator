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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
)

// ConsoleIn holds our state
type ConsoleIn struct {
	// State holds our current state
	State Status
}

// New is our constructor
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
func (io *ConsoleIn) BlockForCharacterNoEcho() (byte, error) {

	// Do we need to change state?  If so then do it.
	if io.State != NoEcho {
		io.disableEcho()
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
func (io *ConsoleIn) BlockForCharacterWithEcho() (byte, error) {

	// Do we need to change state?  If so then do it.
	if io.State != Echo {
		io.enableEcho()
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
func (io *ConsoleIn) ReadLine(max uint8) (string, error) {

	// Do we need to change state?  If so then do it.
	if io.State != Echo {
		io.enableEcho()
	}

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

// Reset restores echo.
func (io *ConsoleIn) Reset() {
	io.enableEcho()
}

// disableEcho is the single place where we disable echoing.
func (io *ConsoleIn) disableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	io.State = NoEcho
}

// enableEcho is the single place where we enable echoing.
func (io *ConsoleIn) enableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "echo").Run()
	io.State = Echo
}
