// Package consoleout is an abstraction over console output.
//
// We know we need an ANSI/RAW output, and we have an ADM-3A driver,
// so we want to create a factory that can instantiate and change a driver,
// given just a name.
package consoleout

import (
	"fmt"
	"io"
	"strings"
)

// ConsoleOutput is the interface that must be implemented by anything
// that wishes to be used as a console driver.
//
// Providing this interface is implemented an object may register itself,
// by name, via the Register method.
//
// You can compare this to the ConsoleInput interface, which is similar, although
// in that case the wrapper which creates the instances also implements some common methods.
type ConsoleOutput interface {

	// PutCharacter will output the specified character to the defined writer.
	//
	// The writer will default to STDOUT, but can be changed, via SetWriter.
	PutCharacter(c uint8)

	// GetName will return the name of the driver.
	GetName() string

	// SetWriter will update the writer.
	SetWriter(io.Writer)
}

// ConsoleRecorder is an interface that allows returning the contents that
// have been previously sent to the console.
//
// This is used solely for integration tests.
type ConsoleRecorder interface {

	// GetOutput returns the contents which have been displayed.
	GetOutput() string

	// Reset removes any stored state.
	Reset()
}

// This is a map of known-drivers
var handlers = struct {
	m map[string]Constructor
}{m: make(map[string]Constructor)}

// Constructor is the signature of a constructor-function
// which is used to instantiate an instance of a driver.
type Constructor func() ConsoleOutput

// Register makes a console driver available, by name.
//
// When one needs to be created the constructor can be called
// to create an instance of it.
func Register(name string, obj Constructor) {
	// Downcase for consistency.
	name = strings.ToLower(name)

	handlers.m[name] = obj
}

// ConsoleOut holds our state, which is basically just a
// pointer to the object handling our output.
type ConsoleOut struct {

	// driver is the thing that actually writes our output.
	driver ConsoleOutput

	// options store per-driver options which might be passed in the
	// constructor.  Right now these are undocumented
	options string
}

// New is our constructor, it creates an output device which uses
// the specified driver.
func New(name string) (*ConsoleOut, error) {

	// Do we have trailing options?
	options := ""

	// If we do save them
	val := strings.Split(name, ":")
	if len(val) == 2 {
		name = val[0]
		options = val[1]
	}

	// Downcase for consistency.
	name = strings.ToLower(name)

	// Do we have a constructor with the given name?
	ctor, ok := handlers.m[name]
	if !ok {
		return nil, fmt.Errorf("failed to lookup driver by name '%s'", name)
	}

	// OK we do, return ourselves with that driver.
	return &ConsoleOut{
		driver:  ctor(),
		options: options,
	}, nil
}

// GetDriver allows getting our driver at runtime.
func (co *ConsoleOut) GetDriver() ConsoleOutput {
	return co.driver
}

// WriteString writes the given string, character by character, via our
// selected output driver.
func (co *ConsoleOut) WriteString(str string) {
	for _, c := range str {
		co.PutCharacter(uint8(c))
	}
}

// ChangeDriver allows changing our driver at runtime.
func (co *ConsoleOut) ChangeDriver(name string) error {

	// Do we have a constructor with the given name?
	ctor, ok := handlers.m[name]
	if !ok {
		return fmt.Errorf("failed to lookup driver by name '%s'", name)
	}

	// change the driver by creating a new object
	co.driver = ctor()
	return nil
}

// GetName returns the name of our selected driver.
func (co *ConsoleOut) GetName() string {
	return co.driver.GetName()
}

// GetDrivers returns all available driver-names.
//
// We hide the internal "null", and "logger" drivers.
func (co *ConsoleOut) GetDrivers() []string {
	valid := []string{}

	for x := range handlers.m {
		if x != "null" && x != "logger" {
			valid = append(valid, x)
		}
	}
	return valid
}

// PutCharacter outputs a character, using our selected driver.
func (co *ConsoleOut) PutCharacter(c byte) {

	// If we have no options then just output the
	// character and have an early return.
	if co.options == "" {
		co.driver.PutCharacter(c)
		return
	}

	// Options only change our newline handling at the moment.
	// so anything that is a different character can also get
	// printed and an early termination.
	if c != '\r' && c != '\n' {
		co.driver.PutCharacter(c)
		return
	}

	// Right so we've got a CR or a LF, and we have non-empty options.
	if c == '\r' {

		// NO CR allowed?  Ignore the character
		if strings.Contains(co.options, "CR=NONE") {
			return
		}

		// CR should do "both"?  Do that
		if strings.Contains(co.options, "CR=BOTH") {
			co.driver.PutCharacter('\r')
			co.driver.PutCharacter('\n')
			return
		}

		// CR is just CR?  Okay
		if strings.Contains(co.options, "CR=CR") {
			co.driver.PutCharacter('\r')
			return
		}

		// CR is actually LF?  Okay
		if strings.Contains(co.options, "CR=LF") {
			co.driver.PutCharacter('\n')
			return
		}

	}
	if c == '\n' {

		// No LF allowed?  Ignore the character
		if strings.Contains(co.options, "LF=NONE") {
			return
		}

		// LF should do "both"?   Do that.
		if strings.Contains(co.options, "LF=BOTH") {
			co.driver.PutCharacter('\r')
			co.driver.PutCharacter('\n')
			return
		}

		// LF is just LF?  Okay.
		if strings.Contains(co.options, "LF=LF") {
			co.driver.PutCharacter('\n')
			return
		}

		// LF is actually CR?  Okay
		if strings.Contains(co.options, "LF=CR") {
			co.driver.PutCharacter('\r')
			return
		}
	}

	//
	// At this point we had CR or LF and yet none of our
	// options made a change.
	//
	// Just print the character.
	//
	co.driver.PutCharacter(c)
}
