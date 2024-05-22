# BIOS Extensions

Traditionally there are two _reserved_ BIOS functions, RESERVE1(31) and RESERVE2(32), I've claimed the first as virtual syscalls that the emulator will handle.

They can be called like so:

    ; Set the function-number to call in H
    ld h, 0xab

    ; Invoke the BIOS function
    ld a, 31
    out (0xff), a

Currently we have one function implemented, demonstrated in [samples/ctrlc.z80](samples/ctrlc.z80):



## Function 0x01

* If C == 0xFF return the value of the Ctrl-C count in A.
* IF C != 0xFF set the Ctrl-C count to be C.

Example:

    ;: get the value
    LD H, 0x01
    LD C, 0xFF
    LD A, 31
    OUT (0xFF), A
    ;; Now A has the result

    ;; Set the value to 4
    LD H, 0x01
    LD C, 0x04
    LD A, 31
    OUT (0xFF), A



## Function 0x02

On entry DE points to a text-string, terminated by NULL, which represents the name of the
console output driver to use.

Demonstrated in [samples/console.z80](samples/console.z80)



## Function 0x00
* TODO
