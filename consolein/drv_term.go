// drv_term.go uses the Termbox library to handle console-based input.
//
// A goroutine is launched which collects any keyboard input and
// saves that to a buffer where it can be peeled off on-demand.
//
// The portability of this solution is unknown, however this driver
// _seems_ reasonable and is the default.

package consolein

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nsf/termbox-go"
	"golang.org/x/term"
)

// TermboxInput is our input-driver, using termbox
type TermboxInput struct {

	// oldState contains the state of the terminal, before switching to RAW mode
	oldState *term.State

	// Cancel holds a context which can be used to close our polling goroutine
	Cancel context.CancelFunc

	// stuffed holds fake input which has been forced into the buffer used
	// by ReadLine
	stuffed string

	// keyBuffer builds up keys read "in the background", via termbox
	keyBuffer []rune
}

// Setup ensures that the termbox init functions are called, and our
// terminal is set into RAW mode.
func (ti *TermboxInput) Setup() {

	var err error

	// switch STDIN into 'raw' mode - we must do this before
	// we setup termbox.
	ti.oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	// Setup the terminal.
	err = termbox.Init()
	if err != nil {
		panic(err)
	}

	// This is "Show Cursor" which termbox hides by default.
	//
	// Sigh.
	fmt.Printf("\x1b[?25h")

	// Allow our polling of keyboard to be canceled
	ctx, cancel := context.WithCancel(context.Background())
	ti.Cancel = cancel

	// Start polling for keyboard input "in the background".
	go ti.pollKeyboard(ctx)
}

// pollKeyboard runs in a goroutine and collects keyboard input
// into a buffer where it will be read from in the future.
func (ti *TermboxInput) pollKeyboard(ctx context.Context) {
	for {
		// Are we done?
		select {
		case <-ctx.Done():
			return
		default:
			// NOP
		}

		// Now look for keyboard input
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Ch != 0 {
				ti.keyBuffer = append(ti.keyBuffer, ev.Ch)
			} else {
				ti.keyBuffer = append(ti.keyBuffer, rune(ev.Key))
			}
		}
	}
}

// TearDown resets the state of the terminal, disables the background polling of characters
// and generally gets us ready for exit.
func (ti *TermboxInput) TearDown() {
	// Cancel the keyboard reading
	if ti.Cancel != nil {
		ti.Cancel()
	}

	// Terminate the GUI.
	termbox.Close()

	// Restore the terminal
	if ti.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), ti.oldState)
	}
}

// StuffInput inserts fake values into our input-buffer
func (ti *TermboxInput) StuffInput(input string) {
	ti.stuffed = input
}

// PendingInput returns true if there is pending input from STDIN.
func (ti *TermboxInput) PendingInput() bool {

	// Do we have faked/stuffed input to process?
	if len(ti.stuffed) > 0 {
		return true
	}

	// Otherwise only if we've read stuff.
	return len(ti.keyBuffer) > 0
}

// BlockForCharacterNoEcho returns the next character from the console, blocking until
// one is available.
//
// NOTE: This function should not echo keystrokes which are entered.
func (ti *TermboxInput) BlockForCharacterNoEcho() (byte, error) {

	// Do we have faked/stuffed input to process?
	if len(ti.stuffed) > 0 {
		c := ti.stuffed[0]
		ti.stuffed = ti.stuffed[1:]
		return c, nil
	}

	// Otherwise only if we've read stuff.
	for len(ti.keyBuffer) == 0 {
		time.Sleep(1 * time.Millisecond)
	}

	// Return the character
	c := ti.keyBuffer[0]
	ti.keyBuffer = ti.keyBuffer[1:]
	return byte(c), nil
}

// GetName is part of the module API, and returns the name of this driver.
func (ti *TermboxInput) GetName() string {
	return "term"
}

// init registers our driver, by name.
func init() {
	Register("term", func() ConsoleInput {
		return new(TermboxInput)
	})
}
