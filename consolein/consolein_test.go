package consolein

import (
	"testing"
)

func TestReadline(t *testing.T) {

	x := New()
	x.State = Echo

	// Simple readline
	// Here \x10 is the Ctrl-P which would use the previous history
	// as we're just created we have none so it is ignored.
	x.stuffed = "s\x10teve\n"
	out, err := x.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error")
	}
	if out != "steve" {
		t.Fatalf("Unexpected output '%s'", out)
	}

	// Ctrl-C at start of the line should trigger a reboot-error
	//	x.stuffed = string([]byte{0x03, 0x03, 0x00}a)
	x.stuffed = "\x03\x03steve"
	x.State = Echo
	_, err = x.ReadLine(20)
	if err != ErrInterrupted {
		t.Fatalf("unexpected error %s", err)
	}

	// Ctrl-C at the middle of a line should not
	x.stuffed = "steve\x03\x03steve\n"
	x.State = Echo
	out, err = x.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "stevesteve" {
		t.Fatalf("unexpected output %s", out)
	}

	// Ctrl-B overwrites
	x.stuffed = "steve\b\b\b\b\bHello\n"
	x.State = Echo
	out, err = x.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}

	// ESC resets input
	x.stuffed = "steve\x1BHello\n"
	x.State = Echo
	out, err = x.ReadLine(20)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}

	// Too much input?  We truncate
	x.stuffed = "I like to move it, move it\n"
	x.State = Echo
	out, err = x.ReadLine(5)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "I lik" {
		t.Fatalf("unexpected output %s", out)
	}

	// Add some history, and return the last value
	x.history = append(x.history, "I like to move it")
	x.stuffed = "ste\x10\n"
	x.State = Echo
	out, err = x.ReadLine(5)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "I lik" {
		t.Fatalf("unexpected output %s", out)
	}

	// Go back and forwardd in history

	x.stuffed = "\x10\x10\x10\x0e\n"
	x.State = Echo
	out, err = x.ReadLine(10)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	if out != "Hello" {
		t.Fatalf("unexpected output %s", out)
	}

}

func TestCtrlC(t *testing.T) {

	x := New()

	if x.InterruptCount != 2 {
		t.Fatalf("unexpected default interrupt count")
	}

	x.SetInterruptCount((3))
	if x.GetInterruptCount() != 3 {
		t.Fatalf("unexpected interrupt count")
	}
}

func TestPending(t *testing.T) {

	x := New()

	x.StuffInput("foo")
	if !x.PendingInput() {
		t.Fatalf("we should have pending input, but see none")
	}

}

// TestExec is just here for coverate
func TestExec(t *testing.T) {

	x := New()

	// No echo
	x.disableEcho()
	if x.State != NoEcho {
		t.Fatalf("unexpected state")
	}

	x.enableEcho()
	if x.State != Echo {
		t.Fatalf("unexpected state")
	}

	x.Reset()
	if x.State != Echo {
		t.Fatalf("unexpected state")
	}
}
