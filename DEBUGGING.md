# Brief notes on Debugging



## Enabling Logging

The emulator can be configured to log all the syscalls, or bios requests, that it was asked to carry out.

To enable logging specify the path to a file, with the `-log-path` command-line argument:

```sh
cpmulator -log-path debug.log [args] 2>logs.out
```

The logging is produced via the golang `slog` package, and will be written in JSON format for ease of processing.  However note that each line is a distinct record and we don't have an array of logs.

You can convert the flat file to a JSON array object using `jq` like so:

```sh
jq --slurp '.' debug.log > debug.json
```

Once you have the logs as a JSON array you can use the `jq` tool to count unique syscalls, or perform further processing like so:

```sh
cat debug.json | jq '.[].name' | sort | grep -v null | uniq --count  | sort --numeric-sort
```
This will give output like this:

```
      1 "F_CLOSE"
      2 "C_READ"
      2 "F_SFIRST"
      2 "P_TERMCPM"
      3 "DRV_ALLRESET"
      4 "DUMP"
      4 "F_OPEN"
      4 "LIHOUSE"
      5 "F_SNEXT"
      8 "DRV_SET"
     14 "C_READSTRING"
     14 "DRV_GET"
     60 "F_USERNUM"
    105 "F_READRAND"
    180 "F_READ"
    293 "F_DMAOFF"
  61728 "C_WRITE"
```



## Notes on Syscalls

There will be two kinds of syscalls BIOS calls and BDOS calls.

We deploy a fake BIOS jump-table and a fake BDOS when the emulator launches.  By default the BIOS jump-table is located at 0xFE00 and the BDOS is located at 0xF000.

* The BIOS syscalls jump to some code that stores the syscall number in the A-register
  * Then OUT 0xFF, A is executed.
  * This sends configures the emulator to carry out the appropriate action.
* The BDOS entrypoint has a tiny piece of code that just runs "OUT (C),C"
  * This out instruction is trapped by the emulator and the syscall number can be taken from the C-register.



## Register Values

Register values matter for some calls, for example the `C_RAWIO` function is invoked when the `C` register contains `06`, however the behaviour is modified by the value of the `E` register:

* If `E` is `0xFF`
  * Read a character from STDIN, blocking.
* If `E` is `0xFE`
  * Test if pending input is available, without blocking.
* Other special cases
  * Have different actions.
* Finally if no special handling is setup then output the character in `E` to STDOUT.

Most other syscalls are more static, but there are examples where a call's behaviour requires seeing the input register such as `F_USERNUM`.
