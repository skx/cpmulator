package cpm

import (
	"fmt"
	"os"
)

// prnC attempts to write the character specified to the "printer".
//
// We redirect printing to use a file, which defaults to "print.log", but
// which can be changed via the CLI argument
func (cpm *CPM) prnC(char uint8) error {

	// If the file doesn't exist, create it.
	f, err := os.OpenFile(cpm.prnPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
