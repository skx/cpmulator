package consoleout

import (
	"fmt"
	"io"
	"os"
)

// Adm3AOutputDriver holds our state.
type Adm3AOutputDriver struct {

	// status contains our state, in the state-machine
	status int

	// x stores the cursor X
	x uint8

	// y stores the cursor Y
	y uint8

	// writer is where we send our output
	writer io.Writer
}

// GetName returns the name of this driver.
//
// This is part of the OutputDriver interface.
func (a3a *Adm3AOutputDriver) GetName() string {
	return "adm-3a"
}

// PutCharacter writes the character to the console.
//
// This is part of the OutputDriver interface.
func (a3a *Adm3AOutputDriver) PutCharacter(c uint8) {

	switch a3a.status {
	case 0:
		switch c {
		case 0x07: /* BEL: flash screen */
			fmt.Fprintf(a3a.writer, "\033[?5h\033[?5l")
		case 0x7F: /* DEL: echo BS, space, BS */
			fmt.Fprintf(a3a.writer, "\b \b")
		case 0x1A: /* adm3a clear screen */
			fmt.Fprintf(a3a.writer, "\033[H\033[2J")
		case 0x0C: /* vt52 clear screen */
			fmt.Fprintf(a3a.writer, "\033[H\033[2J")
		case 0x1E: /* adm3a cursor home */
			fmt.Fprintf(a3a.writer, "\033[H")
		case 0x1B:
			a3a.status = 1 /* esc-prefix */
		case 1:
			a3a.status = 2 /* cursor motion prefix */
		case 2: /* insert line */
			fmt.Fprintf(a3a.writer, "\033[L")
		case 3: /* delete line */
			fmt.Fprintf(a3a.writer, "\033[M")
		case 0x18, 5: /* clear to eol */
			fmt.Fprintf(a3a.writer, "\033[K")
		case 0x12, 0x13:
			// nop
		default:
			fmt.Fprintf(a3a.writer, "%c", c)
		}
	case 1: /* we had an esc-prefix */
		switch c {
		case 0x1B:
			fmt.Fprintf(a3a.writer, "%c", c)
		case '=', 'Y':
			a3a.status = 2
		case 'E': /* insert line */
			fmt.Fprintf(a3a.writer, "\033[L")
		case 'R': /* delete line */
			fmt.Fprintf(a3a.writer, "\033[M")
		case 'B': /* enable attribute */
			a3a.status = 4
		case 'C': /* disable attribute */
			a3a.status = 5
		case 'L', 'D': /* set line */ /* delete line */
			a3a.status = 6
		case '*', ' ': /* set pixel */ /* clear pixel */
			a3a.status = 8
		default: /* some true ANSI sequence? */
			a3a.status = 0
			fmt.Fprintf(a3a.writer, "%c%c", 0x1B, c)
		}
	case 2:
		a3a.y = c - ' ' + 1
		a3a.status = 3
	case 3:
		a3a.x = c - ' ' + 1
		a3a.status = 0
		fmt.Fprintf(a3a.writer, "\033[%d;%dH", a3a.y, a3a.x)
	case 4: /* <ESC>+B prefix */
		a3a.status = 0
		switch c {
		case '0': /* start reverse video */
			fmt.Fprintf(a3a.writer, "\033[7m")
		case '1': /* start half intensity */
			fmt.Fprintf(a3a.writer, "\033[1m")
		case '2': /* start blinking */
			fmt.Fprintf(a3a.writer, "\033[5m")
		case '3': /* start underlining */
			fmt.Fprintf(a3a.writer, "\033[4m")
		case '4': /* cursor on */
			fmt.Fprintf(a3a.writer, "\033[?25h")
		case '5': /* video mode on */
			// nop
		case '6': /* remember cursor position */
			fmt.Fprintf(a3a.writer, "\033[s")
		case '7': /* preserve status line */
			// nop
		default:
			fmt.Fprintf(a3a.writer, "%cB%c", 0x1B, c)
		}
	case 5: /* <ESC>+C prefix */
		a3a.status = 0
		switch c {
		case '0': /* stop reverse video */
			fmt.Fprintf(a3a.writer, "\033[27m")
		case '1': /* stop half intensity */
			fmt.Fprintf(a3a.writer, "\033[m")
		case '2': /* stop blinking */
			fmt.Fprintf(a3a.writer, "\033[25m")
		case '3': /* stop underlining */
			fmt.Fprintf(a3a.writer, "\033[24m")
		case '4': /* cursor off */
			fmt.Fprintf(a3a.writer, "\033[?25l")
		case '6': /* restore cursor position */
			fmt.Fprintf(a3a.writer, "\033[u")
		case '5': /* video mode off */
			// nop
		case '7': /* don't preserve status line */
			// nop
		default:
			fmt.Fprintf(a3a.writer, "%cC%c", 0x1B, c)
		}
		/* set/clear line/point */
	case 6:
		a3a.status++
	case 7:
		a3a.status++
	case 8:
		a3a.status++
	case 9:
		a3a.status = 0
	}

}

// SetWriter will update the writer.
func (a3a *Adm3AOutputDriver) SetWriter(w io.Writer) {
	a3a.writer = w
}

// init registers our driver, by name.
func init() {
	Register("adm-3a", func() ConsoleOutput {
		return &Adm3AOutputDriver{
			writer: os.Stdout,
		}
	})
}
