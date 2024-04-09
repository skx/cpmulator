# go-cpm - A CP/M emulator written in golang

A couple of years ago I wrote a text-based adventure game, to amuse my child.  The game was written in Z80 assembly, for CP/M, and later ported to the ZX Spectrum:

* https://github.com/skx/lighthouse-of-doom/

It occurred to me recently that the game is written in Z80 assembly and only uses a couple of BIOS functions for interfacing with CP/M:

* Await a single keystroke and return it.
* Output a character to the console.
* Read a line of input from the console.

Bearing that in mind it seemed obvious that I could use an existing Z80 emulator to allow the game to be played on any system that could run golang, and this is the result




# Credits

99% of the functionality of this repository comes from the Z80 emulator library I'm using:

* https://github.com/koron-go/z80




# Limitations

This CP/M emulator is extremely basic:

* It loads a binary at 0x0100, which is the starting address for CP/M binaries.
* It implements only three syscalls (i.e. BIOS functions)
  * Read a character from the console - we force the user to press a key THEN press RETURN.
  * Read a line of input from the console.
  * Output a character.

It will no doubt fail to execute any _real_ CP/M binaries.



## Usage

Build and install in the standard way:

```
go install .
```

Or:

```
go install github.com/skx/go-cpm@latest
```

Then download the latest CP/M release of the game from here (the file will be `lihouse.com`):

* https://github.com/skx/lighthouse-of-doom/

After that launch the game:

```
$ go-cpm lihouse.com
```



## Bugs?

Let me know - if your program is "real" then expect failure, life is hard.


Steve
