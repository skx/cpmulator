package consoleout

import (
	"fmt"
	"io"
	"os"
)

// AnsiOutputDriver holds our state.
type AnsiOutputDriver struct {
	// writer is where we send our output
	writer io.Writer
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
	fmt.Fprintf(ad.writer, "%c", c)
}

// SetWriter will update the writer.
func (ad *AnsiOutputDriver) SetWriter(w io.Writer) {
	ad.writer = w
}

// init registers our driver, by name.
func init() {
	Register("ansi", func() ConsoleOutput {
		return &AnsiOutputDriver{
			writer: os.Stdout,
		}
	})
}
