// drv_file creates a console input-driver which reads and
// returns fake console input from a file named "input.txt"
//
// The intent is that this driver will be useful for scripted
// automation.  We add a small delay to all operations just to
// make things seem a little real, and we replace "#" characters
// with a delay of a second too.

package consolein

import (
	"io"
	"os"
	"time"
)

// FileInput is an input-driver that returns fake "console input"
// by reading the content of the file "input.txt".
//
// It is primarily designed for testing and automation.  We make
// a tiny pause between our functions and for every input character
// that is a "#" character we sleep a single second.
//
// We do this because there are some commands that poll for console
// input and cancel, or otherwise process it.  For example the C
// compiler will poll for input when linking and if we don't give it
// some artificial delays we might find our pending input is swallowed
// at random - depending on the speed of the host.
type FileInput struct {

	// offset shows the offset into the buffer we're at
	offset int

	// content contains the content of the "input.txt" file
	content []byte
}

// Setup reads the file "input.txt" and saves the input away for
// future returning when polled.
//
// If the file "input.txt" does not exist in the PWD then an error
// will be returned.
func (fi *FileInput) Setup() error {

	fileName := os.Getenv("INPUT_FILE")
	if fileName == "" {
		fileName = "input.txt"
	}

	dat, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	fi.offset = 0
	fi.content = dat
	return nil
}

// TearDown is a NOP.
func (fi *FileInput) TearDown() error {
	return nil
}

// PendingInput returns true if there is pending input which we
// can return.  This is always true unless we've exhausted the contents
// of our input-file.
func (fi *FileInput) PendingInput() bool {

	// This is designed to ensure that we're not too responsive.
	time.Sleep(5 * time.Millisecond)

	// If our position is less than the size of the data then
	// we have data to read, so it is pending.
	return (fi.offset < len(fi.content))
}

// BlockForCharacterNoEcho returns the next character from the file we
// use to fake our input.
func (fi *FileInput) BlockForCharacterNoEcho() (byte, error) {

	// This is designed to ensure that we're not too responsive.
	time.Sleep(5 * time.Millisecond)

	if fi.offset < len(fi.content) {
		x := fi.content[fi.offset]
		fi.offset++

		if x == '#' {
			time.Sleep(1 * time.Second)
		}
		return x, nil
	}

	// Input is over.
	return 0x00, io.EOF
}

// GetName is part of the module API, and returns the name of this driver.
func (fi *FileInput) GetName() string {
	return "file"
}

// init registers our driver, by name.
func init() {
	Register("file", func() ConsoleInput {
		return new(FileInput)
	})
}
