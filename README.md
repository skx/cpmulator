# cpmulator - A CP/M emulator written in golang

This repository contains a CP/M emulator, with integrated CCP ("Console Command Processor", i.e. shell), which is designed to run CP/M binaries.  The project was initially created to run [a text-based adventure game](https://github.com/skx/lighthouse-of-doom/) which I wrote a few years ago.  (That game was written in Z80 assembly language targeting CP/M, later it was ported to the ZX Spectrum.)

Over time this project has become more complete, and I've now implemented enough functionity to run many of the well-known CP/M programs:

* The Aztec C-Compiler.
* Borland's Turbo Pascal.
* Many early Infocom games:
  * Zork 1, 2, & 3.
  * Planetfall.
  * etc.
* BBC and Microsoft BASIC.
* Wordstar.

As things stand this project is "complete".  I'd like to increase the test-coverage for my own reassurance, but I've now reached a point where all the binaries I've tried to execute work as expected.  If you find a program that _doesn't_ work please [open an issue](https://github.com/skx/cpmulator/issues), otherwise I suspect ongoing development will be minimal, and sporadic.

> **NOTE** I've not implemented any notion of disk-support.  This means that opening, reading/writing, and closing files is absolutely fine, but any API call that refers to tracks, sectors, or disks will fail (with an "unimplemented syscall" error).

A companion repository contains a collection of vintage CP/M software you can use with this, or any other, CP/M emulator:

* [https://github.com/skx/cpm-dist](https://github.com/skx/cpm-dist)




# Installation & Versioning

This emulator is written using golang, so if you have a working golang toolchain you can install in the standard way:

```
go install github.com/skx/cpmulator@latest
```

If you were to clone this repository to your local system you could build, and install it, by running:

```
go build .
go install .
```

If neither of these options are suitable you may download a binary from [our release page](https://github.com/skx/cpmulator/releases).

Releases will be made as/when features seem to justify it, but it should be noted that I consider the CLI tool, and the emulator itself, the "product".  That means that the internal APIs will change around as/when necessary - so far changes have been minor, but I'm not averse to changing parameters to internal packages, or adding/renaming/removing methods as necessary without any regard for external users.




# Quick Start



## Docker-based Quick Start

If you've got docker installed you can launch things by running:

```sh
docker run --pull=always -t -i ghcr.io/skx/cpmulator:master
```

**NOTE** You must run with the `-t` and `-i` flags, otherwise the console will be broken.

Once launched you can explore, for example to run Zork type:

* `G:`
  * This will change to the G:-drive, which is where I've placed games.
* `ZORK1`
  * This will launch the Zork1 binary.



## Local Quick Start

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




# Usage

If you launch `cpmulator` with no arguments then the default CCP will be launched, dropping you into a shell:

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
<summary>Show the standard built-in commands of the default CCP:</summary>

* `CLS`
  * Clear the screen.
* `DIR`
  * Try "`DIR *.COM`" if you want to see only executables, for example.
* `EXIT` / `HALT` / `QUIT`
  * Terminate the CCP.
* `ERA`
  * Erase the named files, wildcards are permitted.
* `TYPE`
  * View the contents of the named file - wildcards are not permitted.
* `REN`
  * Rename files, so "`REN NEW=OLD`" - again note that wildcards are not permitted, nor is cross-drive renaming.

</details>

There are a pair of CCP implementations included within the emulator, and they can be selected via the `-ccp` command-line flag:

* "ccp"
  * The original CCP from Digital Research.
  * Launch the emulator via `cpmulator -ccp=ccp ..`, or use `A:!CCP CCP` to change to it at run-time.
* "ccpz"
  * An enhanced CCP, which is the default
  * "`GET 0100 FOO.COM`" will load a binary into RAM, at address 0x100.  Then "`JMP 0100`" will launch it.
  * There are also built-in `PEEK` and `POKE` commands which can show/set memory contents.
  * `LIST file.ext` will "print" the contents of `file.ext`, but see the note later about printer-output (TLDR; We write it to `print.log`.)
  * The prompt will show the currently-selected user-number, for example if you run "USER 3".
  * If a command isn't found in the current drive A: will be searched instead, which is handy.
  * Finally any line beginning with the comment-character (`#`) will be ignored, which is useful for commenting purposes inside SUBMIT files.

You can also launch a binary directly by specifying it's path upon the command-line, followed by any optional arguments that the binary accepts or requires:

```
$ cpmulator /path/to/binary [optional-args]
```



## Command Line Flags

There are many available command-line options, which are shown in the output of `cpmulator -help`, but the following summary shows the most important/useful options:

* `-cd /path/to/directory`
  * Change to the given directory before running.
* `-directories`
  * Use directories on the host for drive-contents, discussed later in this document.
* `-embed`
  * Enable/Disable the embedded binaries we unconditionally add to the A:-drive.  (The utilities to change the output driver, toggle debugging, etc.)
* `-log-path /path/to/file`
  * Output debug-logs to the given file, creating it if necessary.
  * **NOTE**: You can run `A:!DEBUG 1` to enable "quick debug logging", and `A:!DEBUG 0` to turn it back off again, at runtime.
* `-prn-path /path/to/file`
  * All output which CP/M sends to the "printer" will be written to the given file.
* `-list-syscalls`
  * Dump the list of implemented BDOS and BIOS syscalls.
* `-list-input-drivers` and `-list-output-drivers` to see the available I/O driver-names, which may then be selected via the `-input` and `-output` flags.
* `-version`
  * Show the version number of the emulator, and exit.



## Startup Processing

When the CCP is launched for interactive execution, we allow commands to be executed at startup:

* If `SUBMIT.COM` **and** `AUTOEXEC.SUB` exist on A:
* Then the contents of `AUTOEXEC.SUB` will be executed.
  * We secretly run "`SUBMIT AUTOEXEC`" to achieve this.

This allows you to customize the emulator, or perform other "one-time" setup via the options described in the next section.



## Runtime Behaviour Changes

There are a small number of [extensions](EXTENSIONS.md) added to the BIOS functionality we provide, and these extensions allow changing some aspects of the emulator at runtime.

The behaviour changing is achieved by having a small number of .COM files invoke the extension functions, and these binaries are embedded within our emulator to improve ease of use, via the [static/](static/) directory in our source-tree.  This means no matter what you'll always find some binaries installed on A:, despite not being present in reality - you can disable the appearance of these binaries via the `-embed=false` command-line flag.

> **NOTE** To avoid naming collisions all our embedded binaries are named with a `!` prefix, except for `#.COM` which is designed to be used as a comment-binary.


### CCP Handling

We default to loading the enhanced CCP, but allow the original from Digital Research to be used via the `-ccp` command-line flag.   The binary `A:!CCP.COM` lets you change CCP at runtime.


### Ctrl-C Handling

Traditionally pressing `Ctrl-C` would reload the CCP, via a soft boot.  I think that combination is likely to be entered by accident, so in `cpmulator` we default to requiring you to press Ctrl-C _twice_ in a row to reboot the CCP.

The binary `A:!CTRLC.COM` which lets you change this at runtime.  Run `A:!CTRLC 0` to disable the Ctrl-C behaviour, or `A:!CTRLC N` to require N consecutive Ctrl-C keystrokes to trigger the restart-behaviour (max: 9).


### Console Input

We default to using the portable `termbox-go`-based input-handler, this can be changed via the `-input` command-line flag at startup.  Additionally it can be changed at runtime via `A:!INPUT.COM`.

Run `A:!INPUT stty` to use the non-portable Unix-centric approach which provides a scrollback, and uses the system's `stty` binary to enable/disable character echoing.


### Console Output

We default to pretending our output device is an ADM-3A terminal, this can be changed via the `-output` command-line flag at startup.  Additionally it can be changed at runtime via `A:!OUTPUT.COM`.

Run `A:!OUTPUT ansi` to disable the output emulation, or `A:!OUTPUT adm-3a` to restore it.

You'll see that the [cpm-dist](https://github.com/skx/cpm-dist) repository contains a version of Wordstar, and that behaves differently depending on the selected output handler.  Changing the handler at run-time is a neat bit of behaviour.


### Debug Handling

We expect that all _real_ debugging will involve the comprehensive logfile which is created via the `-log-path` argument to the emulator, however we
have a "quick debug" option which will merely log the syscalls which are invoked, and this has the advantage that it can be enabled, or disabled, at
runtime.

`A:!DEBUG.COM` will show the state of the flag, and it can be enabled with `A:!DEBUG 1` or disabled with `!DEBUG 0`.

Finally `A:!VERSION.COM` will show you the version of the emulator you're running.




# Drives vs. Directories

By default when you launch `cpmulator` with no arguments you'll be presented with the CCP interface, with A: as the current drive.   In this mode A:, B:, C:, and all other drives, will refer to the current-working directory from which you launched the emulator.  This is perhaps the most practical way to get started, but it means that files are repeated across drives:

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

You can also point specific drives to particular paths via the `-drive-X` command-line arguments.  For example the following would have A: and B: pointed to custom paths, and C:-P: using the current working directory:

```
$ cpmulator -ccp=ccpz -drive-a /tmp -drive-b ~/Repos/github.com/skx/cpm-dist/G/
```




# Implemented Syscalls


You can see the list of implemented syscalls, along with a mention of how complete their implementation is, by running:

```
$ cpmulator -list-syscalls
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




# Portability

The CP/M input handlers need to disable echoing when reading (single) characters from STDIN.  There isn't a simple and portable solution for this in golang, although the appropriate primitives exist so building such support isn't impossible, it just relies upon writing per-environment support, using something like the [ReadPassword](https://pkg.go.dev/golang.org/x/term#ReadPassword) function from the standard-library.

I sidestepped this whole problem initially, just invoking the `stty` binary to enable/disable the echoing of characters on-demand, but that only works on Linux, BSD, and Mac hosts.  To be properly portable I had to use the [termbox](https://github.com/nsf/termbox-go) library for all input, but that means we get no scrollback/history so there's a tradeoff to be made.

By default input will be read via `termbox` but you may you specify a different driver via the CLI arguments:

* `cpmulator -input xxx`
  * Use the input-driver named `xxx`.
* `cpmulator -list-input-drivers`
  * List all available input-drivers.




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

For reference the memory map of our CP/M looks like this:

* 0x0000 - Start of RAM
* 0xDE00 - The CCP
* 0xF000 - The BDOS (fake)
* 0xFE00 - The BIOS (fake)




# Credits and References


* Much of the functionality of this repository comes from the [excellent Z80 emulator library](https://github.com/koron-go/z80) it is using, written by [@koron-go](https://github.com/koron-go).
* The default CCP comes from [my fork](https://github.com/skx/z80-playground-cpm-fat/) of the original [cpm-fat](https://github.com/z80playground/cpm-fat/)
  * However this is largely unchanged from the [original CCP](http://www.cpm.z80.de/source.html) from Digital Research, although I did add the `CLS`, `EXIT`, `HALT` & `QUIT` built-in commands.
* Reference Documentation
  * [CP/M BDOS function reference](https://www.seasip.info/Cpm/bdos.html).
  * [CP/M BIOS function reference](https://www.seasip.info/Cpm/bios.html).
  * [CP/M Operating System Manual](http://www.gaby.de/cpm/manuals/archive/cpm22htm/)
    * Contains an assembler reference, DDT commands, etc, etc.
* Other emulators which were useful resources when some functionality was unclear:
  * [https://github.com/ivanizag/iz-cpm](https://github.com/ivanizag/iz-cpm)
    * Portable CP/M emulation to run CP/M 2.2 binaries for Z80.
    * Has a handy "download" script to fetch some CP/M binaries, including BASIC, Turbo Pascal, and WordStar.
    * Written in Rust.
  * [https://github.com/jhallen/cpm](https://github.com/jhallen/cpm)
    * Run CP/M commands in Linux/Cygwin with this Z80 / BDOS / ADM-3A emulator.
    * Written in C.




# Release Checklist

The testing that I should do before a release:

* [ ] Confirm DDT can be used to trace execution of a simple binary.
* [ ] Confirm the A1 Apple Emulator can be launched, and BASIC will run.
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



## Bugs?

Let me know by filing an issue.

Steve
