// Package consolein is an abstraction over console input.
//
// We support two methods of getting input, whilst selectively
// disabling/enabling echo - the use of `termbox' and the use of
// the `stty` binary.
package consolein

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

// ErrInterrupted is returned if the user presses Ctrl-C when in our ReadLine function.
var ErrInterrupted error = fmt.Errorf("INTERRUPTED")

// ConsoleInput is the interface that must be implemented by anything
// that wishes to be used as an input driver.
//
// Providing this interface is implemented an object may register itself,
// by name, via the Register method.
//
// You can compare this interface to the corresponding ConsoleOutput one,
// that delegates everything to the drivers rather than having some wrapper
// methods building upon the drivers as we do here.
type ConsoleInput interface {

	// Setup performs any specific setup which is required.
	Setup()

	// TearDown performs any specific cleanup which is required.
	TearDown()

	// PendingInput returns true if there is pending input available to be read.
	PendingInput() bool

	// BlockForCharacterNoEcho reads a single character from the console, without
	// echoing it.
	BlockForCharacterNoEcho() (byte, error)

	// GetName will return the name of the driver.
	GetName() string
}

// This is a map of known-drivers
var handlers = struct {
	m map[string]Constructor
}{m: make(map[string]Constructor)}

// interruptCount is the count of consecutive Ctrl-Cs which will trigger a "reboot".
var interruptCount int = 2

// stuffed holds pending input
var stuffed string = ""

// history holds previous (line) input.
var history []string

// Constructor is the signature of a constructor-function
// which is used to instantiate an instance of a driver.
type Constructor func() ConsoleInput

// Register makes a console driver available, by name.
//
// When one needs to be created the constructor can be called
// to create an instance of it.
func Register(name string, obj Constructor) {
	// Downcase for consistency.
	name = strings.ToLower(name)

	handlers.m[name] = obj
}

// ConsoleIn holds our state, which is basically just a
// pointer to the object handling our input
type ConsoleIn struct {

	// driver is the thing that actually reads our output.
	driver ConsoleInput

	// systemPrefix is the prefix to use to trigger the execution
	// of system commands, on the host,  in the ReadLine function
	systemPrefix string
}

// New is our constructor, it creates an input device which uses
// the specified driver.
func New(name string) (*ConsoleIn, error) {

	// Downcase for consistency.
	name = strings.ToLower(name)

	// Do we have a constructor with the given name?
	ctor, ok := handlers.m[name]
	if !ok {
		return nil, fmt.Errorf("failed to lookup driver by name '%s'", name)
	}

	// OK we do, return ourselves with that driver.
	return &ConsoleIn{
		driver: ctor(),
	}, nil
}

// SetSystemCommandPrefix enables the use of system-commands in our readline
// function.
func (co *ConsoleIn) SetSystemCommandPrefix(str string) {
	co.systemPrefix = str
}

// GetSystemCommandPrefix returns the value of the system-command prefix.
func (co *ConsoleIn) GetSystemCommandPrefix() string {
	return co.systemPrefix
}

// GetDriver allows getting our driver at runtime.
func (co *ConsoleIn) GetDriver() ConsoleInput {
	return co.driver
}

// GetName returns the name of our selected driver.
func (co *ConsoleIn) GetName() string {
	return co.driver.GetName()
}

// GetDrivers returns all available driver-names.
func (co *ConsoleIn) GetDrivers() []string {
	valid := []string{}

	for x := range handlers.m {
		valid = append(valid, x)
	}
	return valid
}

// Setup proxies into our registered console-input driver.
func (co *ConsoleIn) Setup() {
	co.driver.Setup()
}

// TearDown proxies into our registered console-input driver.
func (co *ConsoleIn) TearDown() {
	co.driver.TearDown()
}

// StuffInput proxies into our registered console-input driver.
func (co *ConsoleIn) StuffInput(input string) {
	stuffed = input
}

// SetInterruptCount sets the number of consecutive Ctrl-C characters
// are required to trigger a reboot.
//
// This function DOES NOT proxy to our registered console-input driver.
func (co *ConsoleIn) SetInterruptCount(val int) {
	interruptCount = val
}

// GetInterruptCount retrieves the number of consecutive Ctrl-C characters are required to trigger a reboot.
//
// This function DOES NOT proxy to our registered console-input driver.
func (co *ConsoleIn) GetInterruptCount() int {
	return interruptCount
}

// PendingInput proxies into our registered console-input driver.
func (co *ConsoleIn) PendingInput() bool {

	// if there is stuffed input we have something ready to read
	if len(stuffed) > 0 {
		return true
	}

	return co.driver.PendingInput()
}

// BlockForCharacterNoEcho proxies into our registered console-input driver.
func (co *ConsoleIn) BlockForCharacterNoEcho() (byte, error) {

	// Do we have faked/stuffed input to process?
	if len(stuffed) > 0 {
		c := stuffed[0]
		stuffed = stuffed[1:]
		return c, nil
	}

	return co.driver.BlockForCharacterNoEcho()
}

// BlockForCharacterWithEcho blocks for input and shows that input before it
// is returned.
//
// This function DOES NOT proxy to our registered console-input driver.
func (co *ConsoleIn) BlockForCharacterWithEcho() (byte, error) {

	// Do we have faked/stuffed input to process?
	if len(stuffed) > 0 {
		c := stuffed[0]
		stuffed = stuffed[1:]
		fmt.Printf("%c", c)
		return c, nil
	}

	c, err := co.driver.BlockForCharacterNoEcho()
	if err == nil {
		fmt.Printf("%c", c)
	}
	return c, err
}

// ReadLine handles the input of a single line of text.
//
// This function DOES NOT proxy to our registered console-input driver.
func (co *ConsoleIn) ReadLine(max uint8) (string, error) {
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

	// We're expecting the user to enter a line of text,
	// but we process their input in terms of characters.
	//
	// We do that so that we can react to special characters
	// such as Esc, Ctrl-N, Ctrl-C, etc.
	//
	// We don't implement Readline, or anything too advanced,
	// but we make a decent effort regardless.
	for {

		// Get a character, with no echo.
		x, err := co.BlockForCharacterNoEcho()
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

				if len(history)-offset < len(history) {
					// replace with a suitable value, and show it
					text = history[len(history)-offset]
					fmt.Printf("%s", text)
				}
			}
			continue
		}

		// Ctrl-P?
		if x == 16 {
			if offset >= len(history) {
				continue
			}
			offset += 1

			eraseInput()

			// replace with a suitable value, and show it
			text = history[len(history)-offset]
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
				if ctrlCount == interruptCount {
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
				if len(history) == 0 {
					history = append(history, text)
				} else {
					// otherwise only add if different to previous entry.
					if text != history[len(history)-1] {
						history = append(history, text)
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

	//
	// Execution of commands, if enabled
	//
	if co.systemPrefix != "" && strings.HasPrefix(text, co.systemPrefix) {

		// Strip the prefix, and any spaces.
		text = text[len(co.systemPrefix):]
		text = strings.TrimSpace(text)

		// cd is a special command.
		if strings.HasPrefix(text, "cd ") {
			bits := strings.Split(text, " ")
			if len(bits) >= 2 {
				dir := bits[1]
				err := os.Chdir(dir)
				if err != nil {
					fmt.Printf("\r\nError changing to directory %s: %s\r\n", dir, err)
				}
			}
			return co.ReadLine(max)
		}

		// Split the command, naively.
		var bits []string
		bits = strings.Split(text, " ")

		// Of course we might be using the shell.
		useShell := false
		if strings.Contains(text, ">") || strings.Contains(text, "&") || strings.Contains(text, "|") || strings.Contains(text, "<") {
			useShell = true
		}

		// If we are we wrap the command.
		if useShell {
			bits = []string{"bash", "-c", text}
		}

		// Prepare to run the command, capturing STDOUT & STDERR
		cmd := exec.Command(bits[0], bits[1:]...)
		var execOut bytes.Buffer
		var execErr bytes.Buffer
		cmd.Stdout = &execOut
		cmd.Stderr = &execErr

		// Actually run the command
		err := cmd.Run()
		if err != nil {
			fmt.Printf("\r\nerror running command '%s' %s%s\r\n", text, err.Error(), execErr.Bytes())
		} else {
			out := execOut.String()
			out += execErr.String()
			out = strings.ReplaceAll(out, "\n", "\n\r")
			fmt.Printf("\r\n%s\r\n", out)
		}

		// Read the input again, since we "stole" it via the exec handling.
		return co.ReadLine(max)
	}

	// Return the text
	return text, nil
}
