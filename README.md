# cpmulator - A CP/M emulator written in golang

This repository contains a CP/M emulator, with integrated CCP, which is designed to run CP/M binaries:

* The project was initially created to run [my text-based adventure game](https://github.com/skx/lighthouse-of-doom/), which I wrote a few years ago, to amuse my child.
  * That was written in Z80 assembly language and initially targeted CP/M, although it was later ported to the ZX Spectrum.

Over time this project has become more complete, and I've now implemented enough functionity to run many of the well-known CP/M programs:

* The Aztec C-Compiler.
* Borland's Turbo Pascal
* Many early Infocom games:
  * Zork 1, 2, & 3.
  * Planetfall.
  * etc.
* Microsoft BASIC
* Wordstar

The biggest caveat is that I've not implemented any notion of disk-based access.  This means that, for example, opening, reading/writing, and closing files is absolutely fine, but any API call that refers to tracks, sectors, or disks will fail (with an "unimplemented syscall" error).

A companion repository contains a collection of vintage CP/M software you can use with this, or any other, CP/M emulator:

* [https://github.com/skx/cpm-dist](https://github.com/skx/cpm-dist)




# Installation & Versioning

This emulator is written using golang, so if you have a working golang toolchain you can install in the standard way:

```
go install github.com/skx/cpmulator@latest
```

If you were to clone this repository to your local system you could then build and install by running:

```
go build .
go install .
```

If neither of these options are suitable you may download a binary from [our release page](https://github.com/skx/cpmulator/releases).

Releases will be made as/when features seem to justify it, but it should be noted that I consider the CLI tool, and the emulator itself, the "product".  That means that the internal APIs will change around as/when necessary - so far changes have been minor, but I'm not averse to changing parameters to internal packages, or adding/renaming/removing methods as necessary without any regard for external users.




# Quick Start

* Build/Install this application.
* Clone the associated repository of binaries:
  * `git clone https://github.com/skx/cpm-dist.git /tmp/cpm-dist`
* Launch the emulator, pointing at the binaries:
  * `cpmulator -cd /tmp/cpm-dist -directories`
* Start something:
  * "B:", then "MBASIC" - to run BASIC.  (Type "SYSTEM" to exit.)
  * "G:", then "ZORK1" - to play zork1.
  * "P:", then "TURBO" - to run turbo pascal.
  * "E:", then "WS" - to run wordstar.




# Portability

The CP/M input handlers need to disable echoing when reading (single) characters from STDIN.  There isn't a simple and portable solution for this in golang, although the appropriate primitives exist so building such support isn't impossible.

Usage of is demonstrated in the standard library:

* [x/term package](https://pkg.go.dev/golang.org/x/term)
  * [ReadPassword](https://pkg.go.dev/golang.org/x/term#ReadPassword) - ReadPassword reads a line of input from a terminal without local echo.

Unfortunately there is no code there for reading only a _single character_, rather than a complete line.  In the interest of expediency I resort to executing the `stty` binary, rather than attempting to use the `x/term` package to manage echo/noecho state myself, and this means the code in this repository isn't 100% portable; it will work on Linux and MacOS hosts, but not Windows.

I've got an open bug about fixing the console (input), [#65](https://github.com/skx/cpmulator/issues/65).




# Usage

If you launch `cpmulator` with no arguments then one of the integrated CCPs ("console command processor") will be launched, dropping you into a familiar shell:

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

You can terminate the CCP by typing `EXIT`.  The following built-in commands are available:

<details>
<summary>Show the built-in commands of the default CCP:</summary>

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


Traditionally pressing `Ctrl-C` would reload the CCP, via a soft boot.  I think that combination is likely to be entered by accident, so in `cpmulator` pressing Ctrl-C _twice_ will reboot the CCP.

> I've added a binary `samples/ctrlc.com` which lets you change this at runtime, via an internal [BIOS extension](EXTENSIONS.md).  Run `ctrlc 0` to disable the Ctrl-C behaviour, or `ctrlc N` to require N consecutive Ctrl-C keystrokes to trigger the restart-behaviour.  Neat.

There are currently a pair of CCP implementations included within the emulator, and they can be selected via the `-ccp` command-line flag:

* "ccp"
  * This is the default, but you can choose it explicitly via `cpmulator -ccp=ccp ..`.
  * The original/default one, from Digital Research
* "ccpz"
  * Launch this via `cpmulate -ccp=ccpz ..`
  * An enhanced one with extra built-in commands.
  * Notably "GET 0100 FOO.COM" will load a binary into RAM, at address 0x100.  Then "JMP 0100" will launch it.
  * The prompt changes to show user-number, for example if you run "USER 3".
  * If a command isn't found in the current drive A: will be searched instead, which is handy.

You can also launch a binary directly by specifying it's path upon the command-line, followed by any optional arguments that the binary accepts or requires:

```
$ cpmulator /path/to/binary [optional-args]
```

Other options are shown in the output of `cpmulator -help`, but in brief:

* `-cd /path/to/directory`
  * Change to the given directory before running.
* `-directories`
  * Use directories on the host for drive-contents, discussed later in this document.
* `-log-path /path/to/file`
  * Output debug-logs to the given file, creating it if necessary.
* `-prn-path /path/to/file`
  * All output which CP/M sends to the "printer" will be written to the given file.
* `-syscalls`
  * Dump the list of implemented BDOS and BIOS syscalls.
* `-version`
  * Show our version number.




# Sample Binaries

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




# Drives vs. Directories

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




# Implemented Syscalls


You can see the list of implemented syscalls, along with a mention of how complete their implementation is, by running:

```
$ cpmulator -syscalls
BDOS
	00 P_TERMCPM
	01 C_READ
	02 C_WRITE
	03 A_READ
..snip..
BIOS
	00  BOOT
	01  WBOOT
..snip..
```

Items marked "FAKE" return "appropriate" values, rather than real values.  Or are otherwise incomplete.

> The only functions with significantly different behaviour are those which should send a single character to the printer (BDOS "L_WRITE" / BIOS "LIST"), they actually send their output to the file `print.log` in the current-directory, creating it if necessary.  (The path may be altered via the `-prn-path` command-line argument.)

The implementation of the syscalls is the core of our emulator, and they can be found here:

* [cpm/cpm_bdos.go](cpm/cpm_bdos.go) - BDOS functions.
  * https://www.seasip.info/Cpm/bdos.html
* [cpm/cpm_bios.go](cpm/cpm_bios.go) - BIOS functions.
  * https://www.seasip.info/Cpm/bios.html




# Debugging Failures & Tweaking Behaviour

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

The following environmental variables influence runtime behaviour:

| Variable    | Purpose                                                       |
|-------------|---------------------------------------------------------------|
| SIMPLE_CHAR | Avoid the attempted VT52 output conversion.                   |

For reference the memory map of our CP/M looks like this:

* 0x0000 - Start of RAM
* 0xDE00 - The CCP
* 0xF000 - The BDOS (fake)
* 0xFE00 - The BIOS (fake)




# Sample Programs

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




# References

* [Digital Research - CP/M Operating System Manual](http://www.gaby.de/cpm/manuals/archive/cpm22htm/)
  * Particularly the syscall reference in [Section 5: CP/M 2 System Interface](http://www.gaby.de/cpm/manuals/archive/cpm22htm/ch5.htm).




# Release Checklist

The testing that I should do before a release:

* [ ] Play lighthouse of doom to completion, either victory or death.
* [ ] Compile a program with ASM & LOAD.  Confirm it runs.
* [ ] Compile HELLO.C and ECHO.C with Aztec C Compiler.
  * [ ] Confirm the generated binaries run.
* [ ] Run BBC Basic, and play a game.
  * [ ] Test "SAVE" and "LOAD" commands.
  * [ ] Test saving tokenized AND raw versions. (i.e `SAVE "FOO"`, and `SAVE "FOO", A`.)
* [ ] Compile a program with Turbo Pascal.
  * [ ] Confirm the generated binary runs.
* [ ] Play Zork1 for a few turns.
  * [ ] Test SAVE and RESTORE commands, and confirm they work.
* [ ] Test BE.COM
* [ ] Test STAT.COM
* [ ] Test some built-in shell-commands; ERA, TYPE, and EXIT.
* [ ] Test `samples/INTEST.COM` `samples/READ.COM`, `samples/WRITE.COM`.
  * These demonstrate core primitives are not broken.



## Bugs?

Let me know by filing an issue.  If your program is "real" then it is highly likely it will try to invoke an unimplemented BIOS function.

Steve
