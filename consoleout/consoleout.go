// Package consoleout is an abstruction over console output.
//
// We know we need an ANSI/RAW output, and we have an ADM-3A driver,
// so we want to create a factory that can instantiate and change a driver,
// given just a name.
package consoleout

import "fmt"

// ConsoleDriver is the interface that must be implemented by anything
// that wishes to be used as a console driver.
type ConsoleDriver interface {

	// PutCharacter will output the specified character to STDOUT.
	PutCharacter(c byte)

	// GetName will return the name of our driver.
	GetName() string
}

// ConsoleOut holds our state, which contains a pointer to the object
// handling the output.
type ConsoleOut struct {

	// driver is the thing that actually writes our output.
	driver ConsoleDriver
}

// This is a map of known-drivers
var handlers = struct {
	m map[string]Constructor
}{m: make(map[string]Constructor)}

// Constructor is the signature of a constructor-function.
type Constructor func() ConsoleDriver

// Register a test-type with a constructor.
func Register(name string, obj Constructor) {
	handlers.m[name] = obj
}

// New is our constructore, it creates an output device which uses
// the specified driver.
func New(name string) (*ConsoleOut, error) {

	// Do we have a constructor with the given name?
	ctor, ok := handlers.m[name]
	if !ok {
		return nil, fmt.Errorf("failed to lookup driver by name '%s'", name)
	}

	// OK we do, return ourselves with that driver.
	return &ConsoleOut{
		driver: ctor(),
	}, nil
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

// PutCharacter outputs a character, using our selected driver.
func (co *ConsoleOut) PutCharacter(c byte) {
	co.driver.PutCharacter(c)
}
