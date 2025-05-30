// drv_file creates a console input-driver which reads and
// returns fake input from a file.  It is used for end-to-end
// or functional-testing of our emulator.

package consolein

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var (
	// FileInputName contains the name of this driver
	FileInputName = "file"
)

// FileInput is an input-driver that returns fake console input
// by reading and returning the contents of a file ("input.txt"
// by default, but this may be changed).
//
// The input-driver is primarily designed for testing and automation.
// We make a tiny pause between our functions and for every input
// we allow some conditional support for changing the line-endings
// which are used when replaying the various test-cases.
type FileInput struct {

	// content contains the content of the file we're returning
	// input from.
	content []byte

	// offset shows the offset into the buffer we're at.
	offset int

	// a test-case can set an arbitrary number of options and here
	// is where we record them.
	options map[string]string

	// fakeInput is input we should return in the future.
	//
	// This is used to return fake Ctrl-M characters when
	// newlines are hit, if required.  It is general-purpose
	// though so we could fake/modify other input options.
	fakeInput string

	// delayUntil is used to see if we're in the middle of a delay,
	// where we pretend we have no input.
	delayUntil time.Time

	// delaySmall is the time we delay before polling input or characters
	delaySmall time.Duration

	// delayLarge is the time we delay when we see '#' in the input file
	delayLarge time.Duration
}

// Setup reads the contents of the file specified by the
// environmental variable $INPUT_FILE, and saves it away as
// a source of fake console input.
//
// If no filename is chosen "input.txt" will be used as a default.
func (fi *FileInput) Setup() error {

	// We allow the input file to be overridden from the
	// default via the environmental-variable.
	fileName := os.Getenv("INPUT_FILE")
	if fileName == "" {
		fileName = "input.txt"
	}

	// Read the content.
	dat, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	// Create a map for storing per-test options
	fi.options = make(map[string]string)

	// The data might be updated to strip off the header.
	fi.offset = 0
	fi.content = fi.parseOptions(dat)

	// We're not delaying by default.
	fi.delayUntil = time.Now()

	// Setup the default delay times
	fi.delaySmall = time.Millisecond * 15
	fi.delayLarge = time.Second * 5

	return nil
}

// parseOptions strips out any options from the given data, recording them
// internally and returning the data after that.
func (fi *FileInput) parseOptions(data []byte) []byte {

	// Ensure that we have a map to store options.
	if fi.options == nil {
		fi.options = make(map[string]string)
	}

	// Length and current offset
	l := len(data)
	i := 0
	position := -1

	// Do we find "--\n" in the data?  If not then there are no options
	for i < l {
		if data[i] == '-' &&
			(i+1) < l &&
			data[i+1] == '-' &&
			(i+2) < l &&
			data[i+2] == '\n' &&
			position < 0 {
			position = i
		}

		i++
	}

	// We didn't find "--" so we can just return the data as-is
	// because there are no options.
	if position < 0 {
		return data
	}

	// Header is 0 - position.
	// Text is position + 3 (the length of "--\n").
	header := data[0:position]
	data = data[position+3:]

	// Split the header by newlines and process
	h := string(header)
	for _, line := range strings.Split(h, "\n") {

		// Trim any leading/trailing whitespace.
		line = strings.TrimSpace(line)

		// lines in the header prefixed by "#" are comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// otherwise the header is key:val pairs
		d := strings.Split(line, ":")
		if len(d) == 2 {
			key := d[0]
			val := d[1]

			// Trim leading/trailing space and down-case.
			key = strings.ToLower(strings.TrimSpace(key))
			val = strings.ToLower(strings.TrimSpace(val))

			// save away
			fi.options[key] = val
		}
	}

	return data
}

// TearDown is a NOP.
func (fi *FileInput) TearDown() error {
	return nil
}

// PendingInput returns true if there is pending input which we
// can return.  This is always true unless we've exhausted the contents
// of our input-file.
func (fi *FileInput) PendingInput() bool {

	time.Sleep(fi.delaySmall)

	// If we're not in a delay period return the real result
	if time.Now().After(fi.delayUntil) {
		return (fi.offset < len(fi.content))
	}

	// We're in a delay period, so just pretend nothing is happening.
	return false
}

// BlockForCharacterNoEcho returns the next character from the file we
// use to fake our input.
func (fi *FileInput) BlockForCharacterNoEcho() (byte, error) {

	time.Sleep(fi.delaySmall)

	// Ensure we block, of we're supposed to.
	for !time.Now().After(fi.delayUntil) {
		time.Sleep(fi.delaySmall)
	}

	// If we have to deal with \r\n instead of just \n handle that first.
	if len(fi.fakeInput) > 0 {
		c := fi.fakeInput[0]
		fi.fakeInput = fi.fakeInput[1:]
		return c, nil
	}

	// If we have input available
	if fi.offset < len(fi.content) {

		// Get the next character, and move past it.
		x := fi.content[fi.offset]
		fi.offset++

		// We've found a newline in our input.
		//
		// We allow newlines to be handled a couple of different
		// ways, and optionally trigger a delay, so we'll have to
		// handle that here.
		//
		if x == '\n' {

			// Does we pause on newlines?
			pause, pausePresent := fi.options["pause-on-newline"]
			if pausePresent && pause == "true" {
				fi.delayUntil = time.Now().Add(fi.delayLarge)
			}

			// Does newline handling have special config?
			opt, ok := fi.options["newline"]

			// Nope.  Return the newline
			if !ok {
				return x, nil
			}

			// Look at way we have been configured to return newlines.
			switch opt {
			case "n":
				// newline: n -> just return "\n"
				return x, nil
			case "m":
				// newline: m -> just return "\r"
				return '\r', nil
			case "both":
				// newline: both -> first return "\r" then "\n"
				fi.fakeInput = "\n" + fi.fakeInput
				return '\r', nil
			default:
				return x, fmt.Errorf("unknown setting 'newline:%s' in test-case", opt)
			}
		}

		return x, nil
	}

	// Input is over.
	return 0x00, io.EOF
}

// GetName is part of the module API, and returns the name of this driver.
func (fi *FileInput) GetName() string {
	return FileInputName
}

// init registers our driver, by name.
func init() {
	Register(FileInputName, func() ConsoleInput {
		return new(FileInput)
	})
}
