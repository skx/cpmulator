//go:build unix

// drv_stty creates a console input-driver which uses the
// `stty` binary to set our echo/no-echo state.
//
// This is obviously not portable outwith Unix-like systems.

package consolein

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

var (
	// STTYInputName contains the name of this driver.
	STTYInputName = "stty"
)

// EchoStatus is used to record our current state.
type EchoStatus int

var (
	// Unknown means we don't know the status of echo/noecho
	Unknown EchoStatus = 0

	// Echo means that input will echo characters.
	Echo EchoStatus = 1

	// NoEcho means that input will not echo characters.
	NoEcho EchoStatus = 2
)

// STTYInput is an input-driver that executes the 'stty' binary
// to toggle between echoing character input, and disabling the
// echo.
//
// This is slow, as you can imagine, and non-portable outwith Unix-like
// systems.  To mitigate against the speed-issue we keep track of "echo"
// versus "noecho" states, to minimise the executions.
type STTYInput struct {

	// state holds our state
	state EchoStatus
}

// Setup is a NOP.
func (si *STTYInput) Setup() error {
	return nil
}

// TearDown resets the state of the terminal.
func (si *STTYInput) TearDown() error {
	if si.state != Echo {
		si.enableEcho()
	}

	return nil
}

// canSelect contains a platform-specific implementation of code that tries to use
// SELECT to read from STDIN.
func canSelect() bool {

	fds := new(unix.FdSet)
	fds.Set(int(os.Stdin.Fd()))

	// See if input is pending, for a while.
	tv := unix.Timeval{Usec: 200}

	// via select with timeout
	nRead, err := unix.Select(1, fds, nil, nil, &tv)
	if err != nil {
		return false
	}

	return (nRead > 0)
}

// PendingInput returns true if there is pending input from STDIN..
//
// Note that we have to set RAW mode, without this input is laggy
// and zork doesn't run.
func (si *STTYInput) PendingInput() bool {

	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}

	// Can we read from STDIN?
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
func (si *STTYInput) BlockForCharacterNoEcho() (byte, error) {

	// Do we need to change state?  If so then do it.
	if si.state != NoEcho {
		si.disableEcho()
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

// disableEcho is the single place where we disable echoing.
func (si *STTYInput) disableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	si.state = NoEcho
}

// enableEcho is the single place where we enable echoing.
func (si *STTYInput) enableEcho() {
	_ = exec.Command("stty", "-F", "/dev/tty", "echo").Run()
	si.state = Echo
}

// GetName is part of the module API, and returns the name of this driver.
func (si *STTYInput) GetName() string {
	return STTYInputName
}

// init registers our driver, by name.
func init() {
	Register(STTYInputName, func() ConsoleInput {
		return new(STTYInput)
	})
}
