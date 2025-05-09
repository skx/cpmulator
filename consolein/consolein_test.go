package consolein

import (
	"os"
	"testing"
)

func TestReadlineSTTY(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	defer func() {
		tErr := ch.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

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

	// Get known drivers
	known := ch.GetDrivers()
	if len(known) != 2 {
		t.Fatalf("unexpected number of public drivers")
	}
	if ch.GetDriver().GetName() != "stty" {
		t.Fatalf("driver has wrong name")
	}
	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := ch.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

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

	// Get the current prefix
	cur := ch.GetSystemCommandPrefix()

	// change it
	ch.SetSystemCommandPrefix("foo")
	if ch.GetSystemCommandPrefix() != "foo" {
		t.Fatalf("failed to change command prefix")
	}

	if ch.GetSystemCommandPrefix() == cur {
		t.Fatalf("failed to change command prefix")
	}
}

// TestCtrlC tests that we can get/set the ctrl-c counter
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

// TestPending tests that we can fake the pending input via stuffing
func TestPending(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := ch.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	ch.StuffInput("foo")
	if !ch.PendingInput() {
		t.Fatalf("we should have pending input, but see none")
	}

}

// TestDriverRegistration performs some sanity-check on our driver-registration.
func TestDriverRegistration(t *testing.T) {

	expectedCount := 4
	found := len(handlers.m)
	if found != expectedCount {
		t.Fatalf("wrong number of handlers.  found %d, expected %d", found, expectedCount)
	}

	_, ok := handlers.m["term"]
	if !ok {
		t.Fatalf("failed to find expected handler, term")
	}
	_, err := New("term")
	if err != nil {
		t.Fatalf("failed to find expected handler, term")
	}

	_, ok = handlers.m["file"]
	if !ok {
		t.Fatalf("failed to find expected handler, file")
	}
	_, err = New("file")
	if err != nil {
		t.Fatalf("failed to find expected handler, file")
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

	//
	// stty
	//
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

	//
	// term
	//
	obj, err2 = New("term")
	if err2 != nil {
		t.Fatalf("error looking up driver")
	}
	drv = obj.GetDriver()
	if drv.GetName() != "term" {
		t.Fatalf("naming mismatch on driver!")
	}
	if obj.GetName() != "term" {
		t.Fatalf("naming mismatch on driver!")
	}

	//
	// file
	//
	obj, err2 = New("file")
	if err2 != nil {
		t.Fatalf("error looking up driver")
	}
	drv = obj.GetDriver()
	if drv.GetName() != "file" {
		t.Fatalf("naming mismatch on driver!")
	}
	if obj.GetName() != "file" {
		t.Fatalf("naming mismatch on driver!")
	}

	//
	// NOTE:
	//
	// We expect to find two less than the number of available
	// drivers in this call, because we hide the "file"-driver and the "error-driver".
	//
	found = len(obj.GetDrivers())
	if found != expectedCount-2 {
		t.Fatalf("wrong number of handlers.  found %d, expected %d", found, expectedCount)
	}
}

// TestSimpleExec executes a simple command
func TestSimpleExec(t *testing.T) {

	cwd := func() string {
		pwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get CWD %s", err)
		}
		return pwd
	}

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := ch.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	// Setup input to run "cd ."
	ch.SetSystemCommandPrefix("!!")
	ch.StuffInput("!!cd .\ninput\n")

	out, err := ch.ReadLine(199)
	if err != nil {
		t.Fatalf("error reading input %s", err)
	}
	if out != "input" {
		t.Fatalf("unexpected input %s", out)
	}

	// Setup input to run a CD that will fail
	ch.SetSystemCommandPrefix("!!")
	ch.StuffInput("!!cd /this(path)/is/missing\ninput\n")
	_, err = ch.ReadLine(199)
	if err != ErrShowOutput {
		t.Fatalf("got error, but not the one we expected: %s", err)
	}

	// Get the CWD before we change
	before := cwd()
	ch.StuffInput("!!cd ..\ninput2\n")

	out, err = ch.ReadLine(199)
	if err != nil {
		t.Fatalf("error reading input %s", err)
	}
	if out != "input2" {
		t.Fatalf("unexpected input %s", out)
	}

	// Confirm we changed directory
	after := cwd()
	if after == before {
		t.Fatalf("failed to change directory")
	}
}

// TestSimpleExecNoOutput executes a simple command which results in no output
func TestSimpleExecNoOutput(t *testing.T) {

	// Create a helper
	x := STTYInput{}

	ch := ConsoleIn{}
	ch.driver = &x

	sErr := ch.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := ch.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	// Create a temporary file
	file, err := os.CreateTemp("", "me.txt")
	if err != nil {
		t.Fatalf("failed to create temporary file")
	}
	defer os.Remove(file.Name())

	// Setup input to run "touch .." which will produce no ouptut
	ch.SetSystemCommandPrefix("!!")
	ch.StuffInput("")
	ch.StuffInput("!!touch " + file.Name() + "\ntouched\n")

	out, err := ch.ReadLine(199)
	if err != nil {
		t.Fatalf("error reading input %s: %s", err, out)
	}
	if out != "touched" {
		t.Fatalf("unexpected input %s", out)
	}

}

// TestFactoryOptions ensures we have some options
func TestFactoryOptions(t *testing.T) {

	d, e := New("stty:CAKE/IS/A/LIE")
	if e != nil {
		t.Fatalf("failed to lookup driver by name %s", e)
	}
	if d.GetName() != "stty" {
		t.Fatalf("setting options broke the name")
	}
}

// TestErrorDriver just tests the error-driver returns errors.
func TestErrorDriver(t *testing.T) {

	d, e := New("error")
	if e != nil {
		t.Fatalf("failed to lookup driver by name %s", e)
	}
	if d.GetName() != "error" {
		t.Fatalf("wrong name for driver")
	}
	if !d.PendingInput() {
		t.Fatalf("expected pending input, got none")
	}
	if d.Setup() != nil {
		t.Fatalf("unexpected error in method")
	}
	if d.TearDown() != nil {
		t.Fatalf("unexpected error in method")
	}

	_, e2 := d.BlockForCharacterNoEcho()
	if e2 == nil {
		t.Fatal("expected error, got none")
	}

	ch := ConsoleIn{}
	x := ErrorInput{}
	ch.driver = &x
	_, e2 = ch.BlockForCharacterWithEcho()
	if e2 == nil {
		t.Fatal("expected error, got none")
	}

	_, e2 = ch.ReadLine(3)
	if e2 == nil {
		t.Fatalf("expected error, got none")
	}
}

// TestCommandExecution does a minimal host-command run
func TestCommandExecution(t *testing.T) {

	d, e := New("stty")
	if e != nil {
		t.Fatalf("failed to lookup driver by name %s", e)
	}

	sErr := d.Setup()
	if sErr != nil {
		t.Fatalf("failed to setup driver %s", sErr.Error())
	}

	defer func() {
		tErr := d.TearDown()
		if tErr != nil {
			t.Fatalf("teardown failed %s", tErr.Error())
		}
	}()

	// Run "env" which should be fine
	d.StuffInput("")
	d.StuffInput("!!env\n")
	d.SetSystemCommandPrefix("!!")

	_, _ = d.ReadLine(10)

	// Run "/not/found" which should fail
	d.StuffInput("")
	d.StuffInput("!!/this/is/not/not/found\n")
	d.SetSystemCommandPrefix("!!")

	_, _ = d.ReadLine(30)
}
