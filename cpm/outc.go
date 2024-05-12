package cpm

import (
	"fmt"
	"os"
)

// outC attempts to write a single character output, but converting to
// ANSI from vt. This means tracking state and handling multi-byte
// output properly.
//
// This is all a bit sleazy.
func (cpm *CPM) outC(c uint8) {

	if os.Getenv("SIMPLE_CHAR") != "" {
		fmt.Printf("%c", c)
		return
	}

	switch cpm.auxStatus {
	case 0:
		switch c {
		case 0x07: /* BEL: flash screen */
			fmt.Printf("\033[?5h\033[?5l")
		case 0x7f: /* DEL: echo BS, space, BS */
			fmt.Printf("\b \b")
		case 0x1a: /* adm3a clear screen */
			fmt.Printf("\033[H\033[2J")
		case 0x0c: /* vt52 clear screen */
			fmt.Printf("\033[H\033[2J")
		case 0x1e: /* adm3a cursor home */
			fmt.Printf("\033[H")
		case 0x1b:
			cpm.auxStatus = 1 /* esc-prefix */
		case 1:
			cpm.auxStatus = 2 /* cursor motion prefix */
		case 2: /* insert line */
			fmt.Printf("\033[L")
		case 3: /* delete line */
			fmt.Printf("\033[M")
		case 0x18, 5: /* clear to eol */
			fmt.Printf("\033[K")
		case 0x12, 0x13:
			// nop
		default:
			fmt.Printf("%c", c)
		}
	case 1: /* we had an esc-prefix */
		switch c {
		case 0x1b:
			fmt.Printf("%c", c)
		case '=', 'Y':
			cpm.auxStatus = 2
		case 'E': /* insert line */
			fmt.Printf("\033[L")
		case 'R': /* delete line */
			fmt.Printf("\033[M")
		case 'B': /* enable attribute */
			cpm.auxStatus = 4
		case 'C': /* disable attribute */
			cpm.auxStatus = 5
		case 'L', 'D': /* set line */ /* delete line */
			cpm.auxStatus = 6
		case '*', ' ': /* set pixel */ /* clear pixel */
			cpm.auxStatus = 8
		default: /* some true ANSI sequence? */
			cpm.auxStatus = 0
			fmt.Printf("%c%c", 0x1b, c)
		}
	case 2:
		cpm.y = c - ' ' + 1
		cpm.auxStatus = 3
	case 3:
		cpm.x = c - ' ' + 1
		cpm.auxStatus = 0
		fmt.Printf("\033[%d;%dH", cpm.y, cpm.x)
	case 4: /* <ESC>+B prefix */
		cpm.auxStatus = 0
		switch c {
		case '0': /* start reverse video */
			fmt.Printf("\033[7m")
		case '1': /* start half intensity */
			fmt.Printf("\033[1m")
		case '2': /* start blinking */
			fmt.Printf("\033[5m")
		case '3': /* start underlining */
			fmt.Printf("\033[4m")
		case '4': /* cursor on */
			fmt.Printf("\033[?25h")
		case '5': /* video mode on */
			// nop
		case '6': /* remember cursor position */
			fmt.Printf("\033[s")
		case '7': /* preserve status line */
			// nop
		default:
			fmt.Printf("%cB%c", 0x1b, c)
		}
	case 5: /* <ESC>+C prefix */
		cpm.auxStatus = 0
		switch c {
		case '0': /* stop reverse video */
			fmt.Printf("\033[27m")
		case '1': /* stop half intensity */
			fmt.Printf("\033[m")
		case '2': /* stop blinking */
			fmt.Printf("\033[25m")
		case '3': /* stop underlining */
			fmt.Printf("\033[24m")
		case '4': /* cursor off */
			fmt.Printf("\033[?25l")
		case '6': /* restore cursor position */
			fmt.Printf("\033[u")
		case '5': /* video mode off */
			// nop
		case '7': /* don't preserve status line */
			// nop
		default:
			fmt.Printf("%cC%c", 0x1b, c)
		}
		/* set/clear line/point */
	case 6:
		cpm.auxStatus++
	case 7:
		cpm.auxStatus++
	case 8:
		cpm.auxStatus++
	case 9:
		cpm.auxStatus = 0
	}

}
