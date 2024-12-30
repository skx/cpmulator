package consolein

import "testing"

func TestReadlineSTTY(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	// Simple readline
	// Here \x10 is the Ctrl-P which would use the previous history
	// as we're just created we have none so it is ignored.
	ch.StuffInput("s\x10teve\n")
	out, err := ch.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if out != "steve" {
		t.Fatalf("Unexpected output '%s'", out)
	}

	// Ctrl-C at start of the line should trigger a reboot-error
	//	x.stuffed = string([]byte{0x03, 0x03, 0x00}a)
	ch.StuffInput("\x03\x03steve")
	_, err = ch.ReadLine(20)
	if err != ErrInterrupted {
		t.Fatalf("unexpected error %s", err)
	}

	// Ctrl-C at the middle of a line should not
	ch.StuffInput("steve\x03\x03steve\n")
	out, err = ch.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "stevesteve" {
		t.Fatalf("unexpected output %s", out)
	}

	// Ctrl-B overwrites
	ch.StuffInput("steve\b\b\b\b\bHello\n")
	out, err = ch.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}

	// ESC resets input
	ch.StuffInput("steve\x1BHello\n")
	out, err = ch.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}

	// Too much input?  We truncate
	ch.StuffInput("I like to move it, move it\n")
	out, err = ch.ReadLine(5)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "I lik" {
		t.Fatalf("unexpected output %s", out)
	}

	// Add some history, and return the last value
	history = append(history, "I like to move it")
	ch.StuffInput("ste\x10\n")
	out, err = ch.ReadLine(5)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "I lik" {
		t.Fatalf("unexpected output %s", out)
	}

	// Go back and forward in history
	ch.StuffInput("\x10\x10\x10\x0e\n")
	out, err = ch.ReadLine(10)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}
}

// TestOverview just calls most of the methods, as an overview, to bump coverage.
func TestOverview(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	ch.Setup()
	defer ch.TearDown()

	ch.StuffInput("1.2.3.4.5.6.7.8.9.0\n")

	if !ch.PendingInput() {
		t.Fatalf("should have pending input")
	}

	c, err := ch.BlockForCharacterNoEcho()
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if c != '1' {
		t.Fatalf("wrong character")
	}

	c, err = ch.BlockForCharacterWithEcho()
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if c != '.' {
		t.Fatalf("wrong character")
	}
}

func TestCtrlC(t *testing.T) {

	ch := ConsoleIn{}

	if interruptCount != 2 {
		t.Fatalf("unexpected default interrupt count")
	}

	ch.SetInterruptCount((3))
	if ch.GetInterruptCount() != 3 {
		t.Fatalf("unexpected interrupt count")
	}
}

func TestPending(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	ch.Setup()
	defer ch.TearDown()

	ch.StuffInput("foo")
	if !ch.PendingInput() {
		t.Fatalf("we should have pending input, but see none")
	}

}

// TestDriverRegistration performs some sanity-check on our driver-registration.
func TestDriverRegistration(t *testing.T) {

	if len(handlers.m) != 2 {
		t.Fatalf("wrong number of handlers")
	}

	_, ok := handlers.m["term"]
	if !ok {
		t.Fatalf("failed to find expected handler, term")
	}
	_, err := New("term")
	if err != nil {
		t.Fatalf("failed to find expected handler, term")
	}

	_, ok = handlers.m["stty"]
	if !ok {
		t.Fatalf("failed to find expected handler, stty")
	}
	_, err = New("stty")
	if err != nil {
		t.Fatalf("failed to find expected handler, term")
	}

	_, ok = handlers.m["bogus"]
	if ok {
		t.Fatalf("found unexpected handler!")
	}
	_, err = New("bogus")
	if err == nil {
		t.Fatalf("failed to find expected handler, term")
	}

	obj, err2 := New("stty")
	if err2 != nil {
		t.Fatalf("error looking up driver")
	}
	drv := obj.GetDriver()
	if drv.GetName() != "stty" {
		t.Fatalf("naming mismatch on driver!")
	}
	if obj.GetName() != "stty" {
		t.Fatalf("naming mismatch on driver!")
	}
	if len(obj.GetDrivers()) != 2 {
		t.Fatalf("driver count is wrong")
	}

}
