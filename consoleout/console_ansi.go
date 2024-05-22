package consoleout

import "fmt"

// AnsiOutputDriver holds our state.
type AnsiOutputDriver struct {
}

// GetName returns the name of this driver.
//
// This is part of the OutputDriver interface.
func (ad *AnsiOutputDriver) GetName() string {
	return "ansi"
}

// PutCharacter writes the specified character to the console.
//
// This is part of the OutputDriver interface.
func (ad *AnsiOutputDriver) PutCharacter(c uint8) {
	fmt.Printf("%c", c)
}

// init registers our driver, by name.
func init() {
	Register("ansi", func() ConsoleDriver {
		return &AnsiOutputDriver{}
	})
}
