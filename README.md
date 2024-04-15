# cpmulator - A CP/M emulator written in golang

A couple of years ago I wrote [a text-based adventure game](https://github.com/skx/lighthouse-of-doom/) in Z80 assembly for CP/M, to amuse my child.  As only a handful of CP/M BIOS functions were used I reasoned it should be possible to emulate those BIOS calls alongside a basic Z80 emulator, and play my game on a modern computer.

This repository is the result, a minimal/portable emulator for CP/M that supports _just enough_ CP/M-functionality to run my game, as well as the ZORK games from Infocom!

* **NOTE**: My game was later ported to the ZX Spectrum.




# Credits

90% of the functionality of this repository comes from the Z80 emulator library I'm using:

* https://github.com/koron-go/z80




# Limitations

This CP/M emulator is extremely basic, I initially implemented just those primitives required to play _my_ game.

Later I added more BIOS functions such that it is now possible to run the Z-machine from Infocom:

* This means you can play Zork 1
* This means you can play Zork 2
* This means you can play Zork 3
* This means you can play The Hitchhikers guide to the galacy

**NOTE**: At the point I can successfully run Zork I will slow down the updates to this repository, but pull-requests to add functionality will always be welcome!



## Notes On Filesystem Functions

Traditionally CP/M would upper-case the command-line arguments it received, which means that you will ALWAYS need to work with filenames in upper-case.

I don't handle different drive-letters, or user-areas.  If you were to run the file-creation code each of these functions would create "./FOO.TXT":

* `cpmulater samples/create.com foo.txt`
* `cpmulater samples/create.com B:foo.txt`
* `cpmulater samples/create.com C:foo.txt`
* `cpmulater samples/create.com D:foo.txt`



## Debugging

When the emulator is asked to execute an unimplemented BIOS call it will abort with a fatal error, for example:

```
$ ./cpmulator FOO.COM
{"time":"2024-04-14T15:39:34.560609302+03:00",
  "level":"ERROR",
  "msg":"Unimplemented syscall",
  "syscall":15,
  "syscallHex":"0x0F"}
Error running FOO.COM: UNIMPLEMENTED
```

You can see a lot of the functions it did successfully emulate and handle by setting the environmental variable DEBUG to a non-empty value, this will generate a log to STDERR where you can save it:

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




# Usage

This emulator is written using golong, so if you have a working golang deployment you can install in the standard way:

```
go install github.com/skx/cpmulator@latest
```

If you were to clone this repository you could build and install by running:

```
go install .
```

If neither of these options sufficed you may download the latest binary from [our release page](https://github.com/skx/cpmulator/releases).




# Usage

Usage only requires that you have a CP/M binary (Z80) which you wish to execute:

```
$ cpmulator /path/to/binary [optional-args]
```

Note that the ZORK games are distributed as two files; an executable and a data-file.  For example if you wish to play ZORK, or a similar game you'll want to have the two files in the same directory:

* ZORK1.COM
  * The filename of this doesn't matter.
* ZORK1.DAT
  * This **must** be named ZORK1.DAT, in upper-case, and be in the current directory.

```
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
..
```



## Sample Programs

You'll see some Z80 assembly programs beneath [samples](samples/) which are used to check my understanding.  If you have the `pasmo` compiler enabled you can build them all by running "make".

In case you don't I've added the compiled versions.



## Bugs?

Let me know - if your program is "real" then expect failure, life is hard.


Steve
