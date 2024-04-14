# cpmulator - A CP/M emulator written in golang

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
* Initially I implements only the four syscalls (i.e. BIOS functions) I needed:
  * Read a single character from the console.
  * Read a line of input from the console.
  * Output a character to the console.
  * Output a $-terminated string to the console.
* I've since added a couple more.
  * But there is currently no file/disk functionality present.
  * I'm undecided if I wish to add them all, though running Zork would be cool.

It will no doubt fail to execute any _real_ CP/M binaries.



## Usage

Build and install in the standard way:

```
go install .
```

Or:

```
go install github.com/skx/cpmulator@latest
```

After that you may launch the binary you wish to run under the emulator - in my case that would be `lihouse.com` from [the release page](https://github.com/skx/lighthouse-of-doom/releases):

```
$ cpmulator lihouse.com
```


## Sample Programs

You'll see some Z80 assembly programs beneath [samples](samples/) which are used to check my understanding.  If you have the `pasmo` compiler enabled you can build them all by running "make".


## Bugs?

Let me know - if your program is "real" then expect failure, life is hard.


Steve
