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

	// fakeNewlines is used to control if we should use
	// an extra character alongside newlines.
	fakeNewlines bool

	// inNewline returns true if we're in the middle of a newline
	// and we need to inject a fake character.
	inNewline bool
}

// Setup reads the contents of the file specified by the
// environmental variable $INPUT_FILE, and saves it away as
// a source of fake console input.
//
// If no filename is chosen "input.txt" will be used as a default.
func (fi *FileInput) Setup() error {

	fileName := os.Getenv("INPUT_FILE")
	if fileName == "" {
		fileName = "input.txt"
	}

	dat, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	// Do we fake newline inputs?  If so set that up now
	if os.Getenv("INPUT_FAKE_NEWLINES") == "1" {
		fi.fakeNewlines = true
	}

	// Save our offset and data.
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
	time.Sleep(15 * time.Millisecond)

	// If our position is less than the size of the data then
	// we have data to read, so it is pending.
	return (fi.offset < len(fi.content))
}

// BlockForCharacterNoEcho returns the next character from the file we
// use to fake our input.
func (fi *FileInput) BlockForCharacterNoEcho() (byte, error) {

	// This is designed to ensure that we're not too responsive.
	time.Sleep(15 * time.Millisecond)

	// If we have input available
	if fi.offset < len(fi.content) {

		// Get the next character, and move past it.
		x := fi.content[fi.offset]
		fi.offset++

		// If we have to deal with \r\n instead
		// of just \n then deal with that.
		if fi.inNewline {
			fi.inNewline = false
			return '', nil
		}
		if x == '\n' && fi.fakeNewlines {
			fi.inNewline = true
		}

		// Also allow a sleep to happen.  Sigh.
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
