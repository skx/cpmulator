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

	// InterruptCount holds the number of consecutive Ctrl-Cs which are necessary
	// to trigger an interrupt response from ReadLine
	InterruptCount int

	// stuffed holds fake input which has been forced into the buffer used
	// by ReadLine
	stuffed string

	// history holds previous (line) input.
	history []string
}

// New is our constructor.
func New() *ConsoleIn {
	t := &ConsoleIn{
		State:          Unknown,
		InterruptCount: 2,
	}
	return t
}

// StuffInput forces input into the buffer which our ReadLine function will
// return.  It is used solely for the AUTOEXEC.SUB behaviour by our driver.
func (ci *ConsoleIn) StuffInput(text string) {
	ci.stuffed = text
}

// SetInterruptCount updates the number of consecutive Ctrl-Cs which are necessary
// to trigger an interrupt in ReadLine.
func (ci *ConsoleIn) SetInterruptCount(val int) {
	ci.InterruptCount = val
}

// GetInterruptCount returns the number of consecutive Ctrl-Cs which are necessary
// to trigger an interrupt in ReadLine.
func (ci *ConsoleIn) GetInterruptCount() int {
	return ci.InterruptCount
}

// PendingInput returns true if there is pending input from STDIN..
//
// Note that we have to set RAW mode, without this input is laggy
// and zork doesn't run.
func (ci *ConsoleIn) PendingInput() bool {

	// Do we have faked/stuffed input to process?
	if len(ci.stuffed) > 0 {
		return true
	}

	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}

	// Platform-specific code in select_XXXX.go
	res := canSelect()

	// restore the state of the terminal to avoid mixing RAW/Cooked
	err = term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		return false
	}

	// Return true if we have something ready to read.
	return res
}

// BlockForCharacterNoEcho returns the next character from the console, blocking until
// one is available.
//
// NOTE: This function should not echo keystrokes which are entered.
func (ci *ConsoleIn) BlockForCharacterNoEcho() (byte, error) {

	// Do we have faked/stuffed input to process?
	if len(ci.stuffed) > 0 {
		c := ci.stuffed[0]
		ci.stuffed = ci.stuffed[1:]
		return c, nil
	}

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

	// Do we have faked/stuffed input to process?
	if len(ci.stuffed) > 0 {
		c := ci.stuffed[0]
		ci.stuffed = ci.stuffed[1:]
		return c, nil
	}

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
// length specified.
//
// Note: We should enable echo in this function.
//
// NOTE: A user pressing Ctrl-C will be caught, and this will trigger the BDOS
// function to reboot.  We have a variable holding the number of consecutive
// Ctrl-C characters are required to trigger this behaviour.
//
// NOTE: We erase the input buffer with ESC, and allow history movement via
// Ctrl-p and Ctrl-n.
func (ci *ConsoleIn) ReadLine(max uint8) (string, error) {

	// Text the user entered
	text := ""

	// count of consecutive Ctrl-C
	ctrlCount := 0

	// offset from history
	offset := 0

	// Erase the text the user has entered, both on the screen
	// and in the input buffer.
	eraseInput := func() {
		for len(text) > 0 {
			text = text[:len(text)-1]
			fmt.Printf("\b \b")
		}
	}

	// Wwe're expecting the user to enter a line of text,
	// but we process their input in terms of characters.
	//
	// We do that so that we can react to special characters
	// such as Esc, Ctrl-N, Ctrl-C, etc.
	//
	// We don't implement Readline, or anything too advanced,
	// but we make a decent effort regardless.
	for {

		// Get a character, with no echo.
		x, err := ci.BlockForCharacterNoEcho()
		if err != nil {
			return "", err
		}

		// Esc? or Ctrl-X
		if x == 27 || x == 24 {

			eraseInput()

			continue
		}

		// Ctrl-N?
		if x == 14 {
			if offset >= 1 {

				offset--

				eraseInput()

				if len(ci.history)-offset < len(ci.history) {
					// replace with a suitable value, and show it
					text = ci.history[len(ci.history)-offset]
					fmt.Printf("%s", text)
				}
			}
			continue
		}

		// Ctrl-P?
		if x == 16 {
			if offset >= len(ci.history) {
				continue
			}
			offset += 1

			eraseInput()

			// replace with a suitable value, and show it
			text = ci.history[len(ci.history)-offset]
			fmt.Printf("%s", text)

			continue
		}

		// Ctrl-C ?
		if x == 0x03 {

			// Ctrl-C should only take effect at the start of the line.
			// i.e. When the text is empty.
			if text == "" {
				ctrlCount += 1

				// If we've hit our limit of consecutive Ctrl-Cs
				// then we return the interrupted error-code
				if ctrlCount == ci.InterruptCount {
					return "", ErrInterrupted
				}
			}
			continue
		}

		// Not a ctrl-c so reset our count
		ctrlCount = 0

		// Newline?
		if x == '\n' || x == '\r' {

			if text != "" {
				// If we have no history, save it.
				if len(ci.history) == 0 {
					ci.history = append(ci.history, text)
				} else {
					// otherwise only add if different to previous entry.
					if text != ci.history[len(ci.history)-1] {
						ci.history = append(ci.history, text)
					}
				}
			}

			// Add the newline and return
			text += "\n"
			break
		}

		// Backspace / Delete? Remove a single character.
		if x == '\b' || x == 127 {

			// remove the character from our text, and overwrite on the console
			if len(text) > 0 {
				text = text[:len(text)-1]
				fmt.Printf("\b \b")
			}
			continue
		}

		// If the user has entered the maximum then we'll say their
		// input-time is over now.
		if len(text) >= int(max) {
			break
		}

		// Finally if it was a printable character we'll keep it.
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
