package consoleout

import (
	"io"
	"os"
)

// NullOutputDriver holds our state.
type NullOutputDriver struct {

	// writer is where we send our output
	writer io.Writer
}

// GetName returns the name of this driver.
//
// This is part of the OutputDriver interface.
func (no *NullOutputDriver) GetName() string {
	return "null"
}

// PutCharacter writes the specified character to the console,
// as this is a null-driver nothing happens and instead the output
// is discarded.
//
// This is part of the OutputDriver interface.
func (no *NullOutputDriver) PutCharacter(c uint8) {
	// NOTHING HAppens
}

// SetWriter will update the writer.
func (no *NullOutputDriver) SetWriter(w io.Writer) {
	no.writer = w
}

// init registers our driver, by name.
func init() {
	Register("null", func() ConsoleDriver {
		return &NullOutputDriver{
			writer: os.Stdout,
		}
	})
}
