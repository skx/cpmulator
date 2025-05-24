// drv_error is a console input-driver which only returns errors.
//
// This driver is only used for testing purposes.

package consolein

import "fmt"

var (
	// ErrorInputName contains the name of this driver.
	ErrorInputName = "error"
)

// ErrorInput is an input-driver that only returns errors, and
// is used for testing.
type ErrorInput struct {
}

// Setup is a NOP.
func (ei *ErrorInput) Setup() error {
	return nil
}

// TearDown is a NOP.
func (ei *ErrorInput) TearDown() error {
	return nil
}

// PendingInput always pretends input is pending.
//
// However when input is polled for, via BlockForCharacterNoEcho,
// an error will always be returned.
func (ei *ErrorInput) PendingInput() bool {
	return true
}

// GetName returns the name of this driver, "error".
func (ei *ErrorInput) GetName() string {
	return ErrorInputName
}

// BlockForCharacterNoEcho always returns an error when
// invoked to read pending input.
func (ei *ErrorInput) BlockForCharacterNoEcho() (byte, error) {
	return 0x00, fmt.Errorf("DRV_ERROR")
}

// init registers our driver, by name.
func init() {
	Register(ErrorInputName, func() ConsoleInput {
		return new(ErrorInput)
	})
}
