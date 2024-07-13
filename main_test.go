// Integration tests :)

package main

import (
	"strings"
	"testing"

	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/cpm"
)

// TestDriveChange ensures the drive-letter changes after
// changing drives.
func TestDriveChange(t *testing.T) {

	obj, err := cpm.New(cpm.WithConsoleDriver("logger"))
	if err != nil {
		t.Fatalf("Create CP/M failed")
	}

	// Load the CCP binary - resetting RAM in the process.
	err = obj.LoadCCP()
	if err != nil {
		t.Fatalf("load CCP failed")
	}

	obj.SetDrives(false)

	obj.StuffText("C:\r\nEXIT\r\n")
	// Run it
	err = obj.Execute([]string{})
	if err != nil && err != cpm.ErrHalt {
		t.Fatalf("failed to run: %s", err)
	}

	// Get our output handle
	helper := obj.GetOutputDriver()
	l, ok := helper.(*consoleout.OutputLoggingDriver)
	if !ok {
		t.Fatalf("failed to cast output driver")
	}

	// Get output written to the screen, and remove newlines
	out := l.GetOutput()
	out = strings.ReplaceAll(out, "\n", "")
	out = strings.ReplaceAll(out, "\r", "")
	if out != `A>C>C>` {

		t.Fatalf("unexpected output '%v'", out)
	}
}

// TestReadWriteRand invokes our help-samples to read/write
// records - via the external API.
func TestReadWriteRand(t *testing.T) {

	obj, err := cpm.New()
	if err != nil {
		t.Fatalf("Create CP/M failed")
	}

	// Load the CCP binary - resetting RAM in the process.
	err = obj.LoadCCP()
	if err != nil {
		t.Fatalf("load CCP failed")
	}

	obj.SetDrives(false)
	obj.SetDrivePath("A", "samples/")
	obj.StuffText("WRITE foo\nREAD foo\nEXIT\n")

	// Run it
	err = obj.Execute([]string{})
	if err != nil && err != cpm.ErrHalt {
		t.Fatalf("failed to run: %s", err)
	}
}

// TestCompleteLighthouse plays our Lighthouse game, to completion.
//
// It uses the fast/hacky solution rather than the slow/normal/real one
// just to cut down on the scripting.
//
// However it is a great test to see that things work as expected.
func TestCompleteLighthouse(t *testing.T) {

	obj, err := cpm.New(cpm.WithConsoleDriver("logger"))
	if err != nil {
		t.Fatalf("Create CP/M failed")
	}

	// Load the CCP binary - resetting RAM in the process.
	err = obj.LoadCCP()
	if err != nil {
		t.Fatalf("load CCP failed")
	}

	obj.SetDrives(false)
	obj.SetDrivePath("A", "dist/")
	obj.StuffText("LIHOUSE\nAAAA\ndown\nEXAMINE DESK\nTAKE METEOR\nUP\n\nn\nQUIT\n")

	// Run it
	err = obj.Execute([]string{})
	if err != nil && err != cpm.ErrHalt {
		t.Fatalf("failed to run: %s", err)
	}

	// Get our output handle
	helper := obj.GetOutputDriver()
	l, ok := helper.(*consoleout.OutputLoggingDriver)
	if !ok {
		t.Fatalf("failed to cast output driver")
	}

	// Get the text written to the screen
	out := l.GetOutput()

	// Ensure the game was completed - easy path.
	if !strings.Contains(out, "Congratulations") {
		t.Fatalf("failed to win")
	}
	if !strings.Contains(out, "You won") {
		t.Fatalf("failed to win")
	}
}
