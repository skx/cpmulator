
        ;; Test that we're running under cpmulator by calling the
        ;; "is cpmulator" function.
exit_if_not_cpmulator:
        ld HL, 0x0000
        ld a, 31
        out (0xff), a

        ;; We expect SKX to appear in registers HLA
        CP 'X'
        jr nz, not_cpmulator

        LD A, H
        CP 'S'
        jr nz, not_cpmulator

        LD A, L
        CP 'K'
        jr nz, not_cpmulator

        ret

WRONG_EMULATOR:
        db "This binary is not running under cpmulator, aborting.", 0x0a, 0x0d, "$"

not_cpmulator:
        LD DE, WRONG_EMULATOR
        LD C, BDOS_OUTPUT_STRING
        call BDOS_ENTRY_POINT
        rst 0x00
