package consoleout

import (
	"io"
	"os"
)

// OutputLoggingDriver holds our state.
type OutputLoggingDriver struct {

	// writer is where we send our output
	writer io.Writer

	// history stores our history
	history string
}

// GetName returns the name of this driver.
//
// This is part of the OutputDriver interface.
func (ol *OutputLoggingDriver) GetName() string {
	return "logger"
}

// PutCharacter writes the specified character to the console,
// as this is a recording-driver nothing happens and instead the output
// is discarded saved into our history
//
// This is part of the OutputDriver interface.
func (ol *OutputLoggingDriver) PutCharacter(c uint8) {
	ol.history += string(c)
}

// SetWriter will update the writer.
func (ol *OutputLoggingDriver) SetWriter(w io.Writer) {
	ol.writer = w
}

// GetOutput returns our history.
//
// This is part of the ConsoleRecorder interface
func (ol *OutputLoggingDriver) GetOutput() string {
	return ol.history
}

// init registers our driver, by name.
func init() {
	Register("logger", func() ConsoleDriver {
		return &OutputLoggingDriver{
			writer: os.Stdout,
		}
	})
}
