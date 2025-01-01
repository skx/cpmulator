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



## Function 0x04: Get/Set Quiet Flag

* Removed.  Obsolete.



## Function 0x05: Get Terminal Size

* Returns the height of the terminal in H.
* Returns the width of the terminal in L.



## Function 0x06: Get/Set Debug flag

* If C is 0x01 debug-mode is enabled.
* If C is 0x00 debug-mode is disabled.
* If C is 0xFF debug-mode is queried.
   * 0x00 means it is not active.
   * 0x01 means it is enabled.

Debug mode shows a summary of syscalls, and their results.  It is faster
than using the logfile and it is useful to be able to toggle it at runtime.

Demonstrated in [static/debug.z80](static/debug.z80)



## Function 0x07: Get/Set Console Input Driver

On entry DE points to a text-string, terminated by NULL, which represents the name of the
console input driver to use.

If DE is 0x0000 then the DMA area is filled with the name of the current driver, NULL-terminated.

Demonstrated in [static/input.z80](static/input.z80)

See also function 0x02.
