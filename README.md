# cpmulator - A CP/M emulator written in golang

This repository contains a CP/M emulator, with integrated CCP, which is primarily designed to launch simple CP/M binaries:

* The project was created to run [a text-based adventure game](https://github.com/skx/lighthouse-of-doom/) I wrote a few years ago, to amuse my child.
  * This was written in Z80 assembly language and initially targeted at CP/M, although it was later ported to the ZX Spectrum.
* Over time it has become more functional:
  * It can now run ZORK 1, 2, & 3, as well as The Hitchhiker's guide to the galaxy and other similar games.

I've implemented enough of the BIOS functions to run simple well-behaved utilities, but I've not implemented any notion of disk access - only file-based I/O.




# Installation

This emulator is written using golang, so if you have a working golang toolchain you can install in the standard way:

```
go install github.com/skx/cpmulator@latest
```

If you were to clone this repository to your local system you could then build and install by running:

```
go install .
```

If neither of these options sufficed you may download the latest binary from [our release page](https://github.com/skx/cpmulator/releases).




# Usage

If you launch `cpmulator` with no arguments then the integrated CCP ("console command processor") will be launched, dropping you into a familiar shell - note here the filenames are presented in the old-school way, as per the later note on filesystems and filenames:

```sh
$ cpmulator
A>
A>DIR *
A:         .GIT |         .GIT |         .GIT | LICENSE
A: README  .MD  | CCP     .    | CPM     .    | CPMULATO.
A: FCB     .    | GO      .MOD | GO      .SUM | MAIN    .GO
A: MEMORY  .    | SAMPLES .

A>TYPE LICENSE
The MIT License (MIT)
..
A>
```

You can terminate the CCP by pressing Ctrl-C, or typing `EXIT`.  The following built-in commands are available - but note that file-renaming is not yet supported, this is noted in #37.

<details>
<summary>Show the CCP built-in commands:</summary>

* `CLS`
  * Clear the screen.
* `DIR`
  * List files, by default this uses "`*.*`", so files without suffixes will be hidden.
    * Prefer "`DIR *`" if you want to see _everything_.
* `EXIT` / `HALT` / `QUIT`
  * Terminate the CCP.
* `ERA`
  * Erase the named files.
* `TYPE`
  * View the contents of the named file - wildcards are not permitted.
* `REN`
  * Rename files, so "`REN NEW = OLD`" - again wildcards are not permitted, nor is cross-drive renaming.  This will fail until I've implemented F_RENAME, as tracked in #37.

</details>


You can also launch a binary directly, by specifying it's path upon the command-line, followed by any optional arguments:

```
$ cpmulator /path/to/binary [optional-args]
```

I've placed some games within the `dist/` directory, so you can launch them easily like so:

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

Note that the Infocom games are distributed as two files; an executable and a data-file.  When you launch the game you'll need to be in the same directory as both the game and the data-file.



## Drives vs. Directories

By default when you launch `cpmulator` with no arguments you'll be presented with the CCP interface, with A: as the current drive and A:, B:, C:, and all other drives, will refer to the current-working directory where you launched the emulator from.  This is perhaps the most practical way to get started, but it means that files are unique across drives:

* i.e. "`A:FOO`" is the same as "`B:FOO`"
  * In short "`FOO`" will exist on drives `A:` all the way through to `P:`.

If you prefer you can configure drives to have their contents from sub-directories upon the host system (i.e. the machine you're running on).  This means you need to create subdirectories, and the contents of those directories will only be visible on the appropriate drives:

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

It isn't currently possibly to point different drives to arbitrary paths on your computer, but that might be considered if you have a use-case for it.



## Debugging Failures

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

You can see a log of the functions it did successfully emulate and handle by setting the environmental variable DEBUG to a non-empty value, this will generate a log to STDERR where you can save it:

```
$ DEBUG=1 ./cpmulator ZORK1.COM  2>log.log
..
$ cat log.log
{"time":"2024-04-14T15:41:20.62879931+03:00",
 "level":"INFO",
 "msg":"Calling BIOS emulation",
 "name":"DRV_GET",
 "syscall":25,
 "syscallHex":"0x19"
}
{"time":"2024-04-14T15:41:20.628908173+03:00",
 "level":"INFO",
 "msg":"Calling BIOS emulation",
 "name":"DRV_SET",
 "syscall":14,
 "syscallHex":"0x0E"
}

```



## Sample Programs

You'll see some Z80 assembly programs beneath [samples](samples/) which are used to check my understanding.  If you have the `pasmo` compiler enabled you can build them all by running "make".

In case you don't I've added ensured I also commit the generated binaries to the git repository.




# Credits

* Much of the functionality of this repository comes from the [excellent Z80 emulator library](https://github.com/koron-go/z80) it is using, written by @koron-go.
* The CCP comes from [my fork](https://github.com/skx/z80-playground-cpm-fat/) of the origianl [cpm-fat](https://github.com/z80playground/cpm-fat/)
  * However this is largely unchanged from the [original CCP](http://www.cpm.z80.de/source.html) from Digital Research.
  * I've not looked at the differences in-depth, but I don't think there are any significant changes.



## Bugs?

Let me know by filing an issue.  If your program is "real" then it is highly likely it will try to invoke an unimplemented BIOS function.

Outstanding issues I'm aware of:

* Inconsistent handling of Drives in FCB entries.
  * There seems to be no suffering here, but ..
* I don't implement some of the basic BIOS calls that might be useful
  * Get free RAM, etc, etc.
  * These will be added over time as their absence causes program-failures.
* PIP.COM doesn't work for file copying; `PIP A:NEW.COM=B:ORIG.COM`.
   * This should copy "ORIG.COM" to "NEW.COM", but aborts due to lack of `F_RANDREC` (and probably more syscalls).
   * You can [read about PIP](https://www.shaels.net/index.php/cpm80-22-documents/using-cpm/6-pip-utility) if you're interested.  It is a useful tool!

Steve
