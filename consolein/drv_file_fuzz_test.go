package consolein

import (
	"io"
	"testing"
	"time"
)

// FuzzOptionsParser does some simple fuzz-testing of our options-parser.
func FuzzOptionsParser(f *testing.F) {

	// empty + whitespace
	f.Add([]byte(nil))
	f.Add([]byte(""))
	f.Add([]byte(`\n\r\t`))

	// One option
	f.Add([]byte("foo:bar\n--\nbar"))

	// Two options
	f.Add([]byte("foo:bar\nbar:baz'\n--\nbar"))

	// misc
	f.Add([]byte("foo:bar\nbar:baz--\nbar"))
	f.Add([]byte("foo:bar\nbar:\n--\nbaz--\nbar"))
	f.Add([]byte("foo\n--\nbaz--\nbarfoo\n--\nbaz--\nbarfoo\n--\nbaz--\nbarfoo\n--\nbaz--\nbar"))
	f.Add([]byte("foo\n--\n#"))
	f.Add([]byte("foo\n--\n##"))
	f.Add([]byte("foo\n--\n###"))
	f.Add([]byte("#"))
	f.Add([]byte("##"))
	f.Add([]byte("####"))

	f.Fuzz(func(t *testing.T, input []byte) {

		// Create a new object using the (fuzzed) input
		tmp := new(FileInput)

		// We don't want to deal with long-sleeps
		tmp.delaySmall = 1 * time.Millisecond
		tmp.delayLarge = 1 * time.Millisecond

		tmp.parseOptions(input)

		// Make sure we can get a character, or EOF.
		//
		// any other error/failure is noteworthy.
		_, err := tmp.BlockForCharacterNoEcho()
		if err != nil && err != io.EOF {
			t.Fatalf("failed to read character %v:%v", input, err)
		}
	})
}
