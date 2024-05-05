# Brief notes on Debugging



## Enabling Logging

The emulator can be configured to log all the syscalls, or bios requests, that it was asked to carry out.

To enable logging set the environmental variable `DEBUG` to any non-empty value, and redirect STDERR to capture these logs into a file:

```sh
DEBUG=1 cpmulator [args] 2>logs.out
```

The logging is produced via the golang `slog` package, and will be written in JSON format for ease of processing.  However note that each line is a distinct record and we don't have an array of logs.

You can convert the flat file to a JSON array object using `jq` like so:

```sh
jq --slurp '.' logs.out > logs.json
```

Once you have the logs as a JSON array you can use the `jq` tool to count unique syscalls, or perform further processing like so:

```sh
cat logs.json | jq '.[].name' | sort | grep -v null | uniq --count  | sort --numeric-sort
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

There will be two kinds of syscalls logged:

* Those that are invoked via the traditional `call 0x0005`.
  * This will be logged as `SysCall`.
* Those that are invoked via "`RST XX`" instructions.
  * This will be logged as `IOSysCall`.

In both cases the register values will be logged, individually and as pairs so you can examine them in whichever way you see fit.



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
