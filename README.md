# go-cpm - A CP/M emulator written in golang

A couple of years ago I wrote a text-based adventure game, to amuse my child.  The game was written in Z80 assembly, for CP/M, and later ported to the ZX Spectrum:

* https://github.com/skx/lighthouse-of-doom/

As the game is written in Z80 assembly and only uses a couple of BIOS functions for interfacing with CP/M, it should be possible to get it running with only a Z80 CPU emulator, along with the implementation of a couple of CP/M BIOS functions.

This repository is the result, a portable and minimal emulator for CP/M that supports enough to run my game.




# Credits

99% of the functionality of this repository comes from the Z80 emulator library I'm using:

* https://github.com/koron-go/z80




# Limitations

This CP/M emulator is extremely basic:

* It loads a binary at 0x0100, which is the starting address for CP/M binaries.
* It implements only four syscalls (i.e. BIOS functions):
  * Read a single character from the console.
  * Read a line of input from the console.
  * Output a character to the console.
  * Output a $-terminated string to the console.

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

After that you may launch the binary you wish to run under the emulator - in my case that would be `lihouse.com` from [the release page](https://github.com/skx/lighthouse-of-doom/releases):

```
$ go-cpm lihouse.com
```



## Bugs?

Let me know - if your program is "real" then expect failure, life is hard.


Steve
