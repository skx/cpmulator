package consolein

import (
	"io"
	"os"
	"testing"
)

func TestFileSetup(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("newline: both\n--\n#hi\n#"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 5 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi\n"+string(byte(0x00)) {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}

	if ch.PendingInput() {
		t.Fatalf("expected no pending input (we read the text)")
	}

	// After we've read all the characters we should just get EOF
	c = 0
	for c < 10 {
		_, err = ch.BlockForCharacterNoEcho()
		if err != io.EOF {
			t.Fatalf("expected EOF, got %v", err)
		}
		c++
	}

	sErr = ch.TearDown()
	if sErr != nil {
		t.Fatalf("failed to teardown")
	}
}

func TestSetupFail(t *testing.T) {

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	// input.txt doesn't exist so we get an error..
	sErr := ch.Setup()
	if sErr == nil {
		t.Fatalf("expected error, got none")
	}

	sErr = ch.TearDown()
	if sErr != nil {
		t.Fatalf("failed to teardown")
	}

}

// TestNoOptions tests some small data with no options
func TestNoOptions(t *testing.T) {

	f := new(FileInput)

	// 0
	out := f.parseOptions([]byte{})
	if len(out) != 0 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 0 {
		t.Fatalf("unexpected options present")
	}

	// 1
	out = f.parseOptions([]byte{0x01})
	if len(out) != 1 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 0 {
		t.Fatalf("unexpected options present")
	}

	// 2
	out = f.parseOptions([]byte{0x01, 0x02})
	if len(out) != 2 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 0 {
		t.Fatalf("unexpected options present")
	}

}

// TestOptions tests some small options
func TestOptions(t *testing.T) {

	f := new(FileInput)

	// One option.
	out := f.parseOptions([]byte("Foo: bar\n--\none"))
	if len(out) != 3 {
		t.Fatalf("error got the wrong data: %s", out)
	}
	if len(f.options) != 1 {
		t.Fatalf("unexpected options present")
	}
	if f.options["foo"] != "bar" {
		t.Fatalf("bogus options %v", f.options)
	}

	// One comment
	f = new(FileInput)
	out = f.parseOptions([]byte("# Foo: bar\n--\none"))
	if len(out) != 3 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 0 {
		t.Fatalf("unexpected options present")
	}

	// Comment and option
	f = new(FileInput)
	out = f.parseOptions([]byte("# Test\nFoo: bar\nsteve:kemp   \n--\none"))
	if len(out) != 3 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 2 {
		t.Fatalf("unexpected options present")
	}
	if f.options["foo"] != "bar" {
		t.Fatalf("bogus options %v", f.options)
	}
	if f.options["steve"] != "kemp" {
		t.Fatalf("bogus options %v", f.options)
	}

	// Most recent option takes precedence
	f = new(FileInput)
	out = f.parseOptions([]byte("# Test\nFoo: bar\nsteve:kemp   \nFoo: steve--\none"))
	if len(out) != 3 {
		t.Fatalf("error got the wrong data: %v", out)
	}
	if len(f.options) != 2 {
		t.Fatalf("unexpected options present")
	}
	if f.options["foo"] != "steve" {
		t.Fatalf("bogus options %v", f.options)
	}
	if f.options["steve"] != "kemp" {
		t.Fatalf("bogus options %v", f.options)
	}

}

// TestNewlineN ensures "newline: n" returns the expected character. (\n)
func TestNewlineN(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("newline: n\n--\nhi\n"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 3 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi\n" {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}

}

// TestNewlineM ensures "newline: m" returns the expected character. (Ctrl-M)
func TestNewlineM(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("newline: m\n--\nhi\n"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 3 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi"+string('') {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}
}

// TestNewlineBoth ensures "newline: both" returns both expected characters.
func TestNewlineBoth(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("newline: both\n--\nhi\n"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 4 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi"+string('')+"\n" {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}
}

// TestNewlineBogus ensures "newline: foo" returns just \n by default
func TestNewlineBogus(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("newline: bogus\n--\nhi\n"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 3 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi\n" {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}
}

// TestNewlineMissing ensures we returns just \n by default
func TestNewlineMissing(t *testing.T) {

	// Create a temporary file
	file, err := os.CreateTemp("", "in.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("nothing: bogus\n--\nhi\n"))
	if err != nil {
		t.Fatalf("failed to write to temporary file")
	}

	t.Setenv("INPUT_FILE", file.Name())

	// Create a helper
	x := FileInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	if !ch.PendingInput() {
		t.Fatalf("expected pending input (a)")
	}

	c := 0
	str := ""

	for c < 3 {
		var out byte
		out, err = ch.BlockForCharacterNoEcho()
		if err != nil {
			t.Fatalf("failed to get character")
		}
		str += string(out)

		c++
	}
	if str != "hi\n" {
		t.Fatalf("error in string, got '%v' '%s'", str, str)
	}
}
