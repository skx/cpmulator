# cpmulator - A CP/M emulator written in golang

A couple of years ago I wrote [a text-based adventure game](https://github.com/skx/lighthouse-of-doom/) in Z80 assembly for CP/M, to amuse my child.  As the game is written in Z80 assembly and only uses a couple of the CP/M BIOS functions I reasoned it should be possible to emulate just a few functions to get the game playable via golang - which would make it portable to Windows, MacOS, and other systems.

* **NOTE**: My game was later ported to the ZX Spectrum.

This repository is the result, a minimal/portable emulator for CP/M that supports just enough of the CP/M BIOS functions to run my game.

My intention is to implement sufficient functionality to run ZORK and HITCHHIKERS, and then stop active work upon it.




# Credits

95% of the functionality of this repository comes from the Z80 emulator library I'm using:

* https://github.com/koron-go/z80




# Limitations

This CP/M emulator is extremely basic:

* It loads a binary at 0x0100, which is the starting address for CP/M binaries.
* Initially I implements only the four syscalls (i.e. BIOS functions) I needed:
  * Read a single character from the console.
  * Read a line of input from the console.
  * Output a character to the console.
  * Output a $-terminated string to the console.
* I've since started to add more, and there are open issues you can use to track progress of the obvious missing calls (primarily file I/O related functinality):
  * [Open issues](https://github.com/skx/cpmulator/issues)

**NOTE**: At the point I can successfully run Zork I will slow down the updates to this repository, but pull-requests to add functionality will be welcome regardless.



## Debugging

When the emulator is asked to execute an unimplemented BIOS call it will abort with a fatal error, for example:

```
$ ./cpmulator ZORK1.COM
{"time":"2024-04-14T15:39:34.560609302+03:00",
  "level":"ERROR",
  "msg":"Unimplemented syscall",
  "syscall":15,
  "syscallHex":"0x0F"}
Error running ZORK1.COM: UNIMPLEMENTED
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
