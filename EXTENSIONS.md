# BIOS Extensions

Traditionally there are two _reserved_ BIOS functions, RESERVE1(31) and RESERVE2(32), I've claimed the first as virtual syscalls that the emulator will handle.

They can be called like so:

    ; Set the function-number to call in HL
    ld hl, 0x00

    ; Invoke the BIOS function
    ld a, 31
    out (0xff), a

We've implemented a small number of custom BIOS calls, documented below.



## Function 0x00: CPMUlator?

Test to see if the code is running under cpmulator, the return value is split into two parts:

* Registers are set to specific values:
  * H -> `S`
  * L -> `K`
  * A -> `X`
* The DMA buffer is filled with a text-banner, null-terminated.



## Function 0x01: Get/Set Ctrl-C Count

* If C == 0xFF return the value of the Ctrl-C count in A.
* IF C != 0xFF set the Ctrl-C count to be C.

Example:

    ;: get the value
    LD HL, 0x01
    LD C, 0xFF
    LD A, 31
    OUT (0xFF), A
    ;; Now A has the result

    ;; Set the value to 4
    LD HL, 0x01
    LD C, 0x04
    LD A, 31
    OUT (0xFF), A

Demonstrated in [static/ctrlc.z80](static/ctrlc.z80).



## Function 0x02: Get/Set Console Output Driver

On entry DE points to a text-string, terminated by NULL, which represents the name of the
console output driver to use.

If DE is 0x0000 then the DMA area is filled with the name of the current driver, NULL-terminated.

Demonstrated in [static/output.z80](static/output.z80)

See also function 0x07.



## Function 0x03: Get/Set CCP

On entry DE points to a text-string, terminated by NULL, which represents the name of the
CCP to use.

If DE is 0x0000 then the DMA area is filled with the name of the currently active CCP, NULL-terminated.

Demonstrated in [static/ccp.z80](static/ccp.z80)



## Function 0x04: NOP

This is an obsolete function, which does nothing.



## Function 0x05: Get Terminal Size

* Returns the height of the terminal in H.
* Returns the width of the terminal in L.



## Function 0x06: NOP

This is an obsolete function, which does nothing.



## Function 0x07: Get/Set Console Input Driver

On entry DE points to a text-string, terminated by NULL, which represents the name of the
console input driver to use.

If DE is 0x0000 then the DMA area is filled with the name of the current driver, NULL-terminated.

Demonstrated in [static/input.z80](static/input.z80)

See also function 0x02.



## Function 0x08: Get/Set Prefix for running commands on the host

On entry DE points to a text-string, terminated by NULL, which represents the prefix which will
be used to allow executing commands on the host-system.

For example if you were to run `!hostcmd !!` then enter `!!uptime` within the CCP prompt you'd
actually see the output of running the `uptime` command.

If DE is 0x0000 then the DMA area is filled with the name of the current prefix, NULL-terminated.

Demonstrated in [static/hostcmd.z80](static/hostcmd.z80)



## Function 0x09: Disable the BIOS extensions documented in this page.

This function is used to disable the embedded filesystem we use to host our utility functions, and
the BIOS extensions documented upon this page.  On entry DE is used to determine what to disable:

* 0x0001 - Disable the embedded filesystem.
* 0x0002 - Disable the custom BIOS functions.
* 0x0003 - Disable both the embedded filesystem, and the custom BIOS functions.
* 0x0004 - Disable both the embedded filesystem, and the custom BIOS functions, but do so quietly.

Demonstrated in [static/disable.z80](static/disable.z80)



## Function 0x0A: Get/Set Printer Log Path

On entry DE points to a text-string, terminated by NULL, which represents the name of the
file to write printer-output to.

If DE is 0x0000 then the DMA area is filled with the name of the printer log-file, NULL-terminated.

Demonstrated in [static/prnpath.z80](static/prnpath.z80)

See also function 0x02.
