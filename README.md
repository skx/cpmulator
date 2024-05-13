# cpmulator - A CP/M emulator written in golang

This repository contains a CP/M emulator, with integrated CCP, which is primarily designed to launch simple CP/M binaries:

* The project was initially created to run [my text-based adventure game](https://github.com/skx/lighthouse-of-doom/), which I wrote a few years ago, to amuse my child.
  * This was written in Z80 assembly language and initially targeted at CP/M, although it was later ported to the ZX Spectrum.

Over time this project has become more complete, and more complex.  I've implemented enough of the BIOS functions to run simple binaries and also several of the well-known programs of the time:

* The Aztec C-Compiler.
  * You can edit and compile code within the emulator, then run it!
* Borland's Turbo Pascal
  * You can edit and compile code within the emulator, then run it!
* Microsoft BASIC
* Zork 1, 2, & 3

The biggest caveat of the emulator is that I've not implemented any notion of disk-based access.  This means that, for example opening, reading/writing, and closing files is absolutely fine, but any API call that refers to tracks, sectors, or disks will fail.

A companion repository contains a collection of vintage CP/M software you can use with this, or any other, emulator:

* [https://github.com/skx/cpm-dist](https://github.com/skx/cpm-dist)




# Installation

This emulator is written using golang, so if you have a working golang toolchain you can install in the standard way:

```
go install github.com/skx/cpmulator@latest
```

If you were to clone this repository to your local system you could then build and install by running:

```
go install .
```

If neither of these options are suitable you may download the latest binary from [the release page](https://github.com/skx/cpmulator/releases).




# Portability

The CP/M input handlers need to disable echoing when reading (single) characters from STDIN.  There isn't a simple and portable solution for this in golang - so I've resorted to the naive approach of executing `stty` when necessary.

This means the code in this repository isn't 100% portable; it will work on Linux and MacOS hosts, but not Windows.

There _is_ code to disable echoing in the golang standard library, for example you can [consider the code in the readPassword function](https://cs.opensource.google/go/x/term/+/refs/tags/v0.20.0:term_unix.go) in `x/term`.  Unfortunately the facilities there are only sufficient for reading a _line_, not a _character_.




# Usage

If you launch `cpmulator` with no arguments then the integrated CCP ("console command processor") will be launched, dropping you into a familiar shell:

```sh
$ cpmulator
A>dir

A: LICENSE .    | README  .MD  | CPMULATO.    | GO      .MOD
A: GO      .SUM | MAIN    .GO  | RET     .COM

A>TYPE LICENSE
The MIT License (MIT)
..
A>
```

You can terminate the CCP by pressing Ctrl-C, or typing `EXIT`.  The following built-in commands are available:

<details>
<summary>Show the CCP built-in commands:</summary>

* `CLS`
  * Clear the screen.
* `DIR`
  * List files, by default this uses "`*.*`".
    * Try "`DIR *.COM`" if you want to see something more specific, for example.
* `EXIT` / `HALT` / `QUIT`
  * Terminate the CCP.
* `ERA`
  * Erase the named files, wildcards are permitted.
* `TYPE`
  * View the contents of the named file - wildcards are not permitted.
* `REN`
  * Rename files, so "`REN NEW=OLD`" - again note that wildcards are not permitted, nor is cross-drive renaming.

</details>


You can also launch a binary directly by specifying it's path upon the command-line, followed by any optional arguments that the binary accepts or requires:

```
$ cpmulator /path/to/binary [optional-args]
```



## Sample Binaries

I've placed some games within the `dist/` directory, to make it easier for you to get started:

```sh
$ cd dist/
$ cpmulator ZORK1.COM
ZORK I: The Great Underground Empire
Copyright (c) 1981, 1982, 1983 Infocom, Inc. All rights
reserved.
ZORK is a registered trademark of Infocom, Inc.
Revision 88 / Serial number 840726

West of House
You are standing in an open field west of a white house, with
a boarded front door.
There is a small mailbox here.

>
```

A companion repository contains a larger collection of vintage CP/M software you can use with this emulator:

* [https://github.com/skx/cpm-dist](https://github.com/skx/cpm-dist)



## Drives vs. Directories

By default when you launch `cpmulator` with no arguments you'll be presented with the CCP interface, with A: as the current drive.   In this mode A:, B:, C:, and all other drives, will refer to the current-working directory where you launched the emulator from (i.e. they have the same view of files).  This is perhaps the most practical way to get started, but it means that files are repeated across drives:

* i.e. "`A:FOO`" is the same as "`B:FOO`", and if you delete "`C:FOO`" you'll find it has vanished from all drives.
  * In short "`FOO`" will exist on drives `A:` all the way through to `P:`.

If you prefer you may configure drives to be distinct, each drive referring to a distinct sub-directory upon the host system (i.e. the machine you're running on):

```sh
$ mkdir A/  ; touch A/LS.COM ; touch A/FOO.COM
$ mkdir B/  ; touch B/DU.COM ; touch B/BAR.COM
$ mkdir G/  ; touch G/ME.COM ; touch G/BAZ.COM
```

Now if you launch the emulator you'll see only the files which _should_ be visible on the appropriate drive:


```sh
$ cpmulator -directories
A>DIR A:
A: FOO     .COM | LS      .COM

A>DIR B:
B: BAR     .COM | DU      .COM

A>DIR G:
G: BAZ     .COM | ME      .COM

A>DIR E:
No file
```

A companion repository contains a larger collection of vintage CP/M software you can use with this emulator:

* [https://github.com/skx/cpm-dist](https://github.com/skx/cpm-dist)

This is arranged into subdirectories, on the assumption you'll run with the `-directories` flag, and the drives are thus used as a means of organization.  For example you might want to look at games, on the `G:` drive, or the BASIC interpreters on the `B:` drive:

```
frodo ~/Repos/github.com/skx/cpm-dist $ cpmulator  -directories
A>g:
G>dir *.com
G: HITCH   .COM | LEATHER .COM | LIHOUSE .COM | PLANET  .COM
G: ZORK1   .COM | ZORK2   .COM | ZORK3   .COM

G>dir b:*.com
B: MBASIC  .COM | OBASIC  .COM | TBASIC  .COM
```

Note that it isn't currently possibly to point different drives to arbitrary paths on your computer, but that might be considered if you have a use-case for it.



## Implemented Syscalls


You can see the list of implemented syscalls, along with a mention of how complete their implementation is, by running:

```
$ cpmulator -syscalls
00 P_TERMCPM
01 C_READ
02 C_WRITE
03 A_READ
04 A_WRITE
05 L_WRITE
06 C_RAWIO
07 GET_IOBYTE
08 SET_IOBYTE
09 C_WRITESTRING
10 C_READSTRING
11 C_STAT               FAKE
12 S_BDOSVER
..
31 DRV_DPB              FAKE
32 F_USERNUM
33 F_READRAND
34 F_WRITERAND
```

Items marked "FAKE" return "appropriate" values, rather than real values.  Or are otherwise incomplete.  The only function with significantly different behaviour is `L_WRITE` which is designed to send a single character to a connected printer - that function appends the character to the file `print.log` in the current-directory, creating it if necessary.



## Debugging Failures & Tweaking Behaviour

When an unimplemented BIOS call is attempted the program it will abort with a fatal error, for example:

```
$ ./cpmulator FOO.COM
{"time":"2024-04-14T15:39:34.560609302+03:00",
  "level":"ERROR",
  "msg":"Unimplemented syscall",
  "syscall":255,
  "syscallHex":"0xFF"}
Error running FOO.COM: UNIMPLEMENTED
```

If things are _mostly_ working, but something is not quite producing the correct result then we have some notes on debugging:

* [DEBUGGING.md](DEBUGGING.md)

Here is the complete list of environmental variables which influence behaviour:

| Variable    | Purpose                                                       |
|-------------|---------------------------------------------------------------|
| DEBUG       | Send a log of CP/M syscalls to STDERR                         |
| SIMPLE_CHAR | Avoid the attempted VT52 output conversion.                   |



## Sample Programs

You'll see some Z80 assembly programs beneath [samples](samples/) which are used to check my understanding.  If you have the `pasmo` compiler enabled you can build them all by running "make", in case you don't I've also committed the generated binaries.




# Credits

* Much of the functionality of this repository comes from the [excellent Z80 emulator library](https://github.com/koron-go/z80) it is using, written by [@koron-go](https://github.com/koron-go).
* The CCP comes from [my fork](https://github.com/skx/z80-playground-cpm-fat/) of the original [cpm-fat](https://github.com/z80playground/cpm-fat/)
  * However this is largely unchanged from the [original CCP](http://www.cpm.z80.de/source.html) from Digital Research, although I did add the `CLS`, `EXIT`, `HALT` & `QUIT` built-in commands.

When I was uncertain of how to implement a specific system call the following two emulators were also useful:

* [https://github.com/ivanizag/iz-cpm](https://github.com/ivanizag/iz-cpm)
  * Portable CP/M emulation to run CP/M 2.2 binaries for Z80.
    * Has a handy "download" script to fetch some CP/M binaries, including BASIC, Turbo Pascal, and WordStar.
  * Written in Rust.
* [https://github.com/jhallen/cpm](https://github.com/jhallen/cpm)
  * Run CP/M commands in Linux/Cygwin with this Z80 / BDOS / ADM-3A emulator.
  * Written in C.



## References

* [Digital Research - CP/M Operating System Manual](http://www.gaby.de/cpm/manuals/archive/cpm22htm/)
  * Particularly the syscall reference in [Section 5: CP/M 2 System Interface](http://www.gaby.de/cpm/manuals/archive/cpm22htm/ch5.htm).
* Sample code
  * https://github.com/Laci1953/RC2014-CPM/tree/main



## Bugs?

Let me know by filing an issue.  If your program is "real" then it is highly likely it will try to invoke an unimplemented BIOS function.


Steve
