package cpm

import (
	"fmt"
	"os"
)

// File is an interface that we mock so that we can fake failures to write and close
// our printer logfile.
type File interface {
	// Write is designed to write data, but it does not.
	// It will either do nothing, or return an error under testing.
	Write([]byte) (int, error)

	// Close is designed to close a file, but it does not.
	// It will either do nothing, or return an error under testing.
	Close() error
}

// opener is the factory that will allow creating either a real os.File, or a mockFile
var opener func(name string, flag int, perm os.FileMode) (File, error)

// prnC attempts to write the character specified to the "printer".
//
// We redirect printing to use a file, which defaults to "print.log", but
// which can be changed via the CLI argument
func (cpm *CPM) prnC(char uint8) error {

	if opener == nil {
		opener = func(name string, flag int, perm os.FileMode) (File, error) {
			return os.OpenFile(name, flag, perm)
		}
	}
	// If the file doesn't exist, create it.
	f, err := opener(cpm.prnPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("prnC: Failed to open file %s:%s", cpm.prnPath, err)
	}

	data := make([]byte, 1)
	data[0] = char
	_, err = f.Write(data)
	if err != nil {
		return fmt.Errorf("prnC: Failed to write to file %s:%s", cpm.prnPath, err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("prnC: Failed to close file %s:%s", cpm.prnPath, err)
	}

	return nil
}
