;; common.inc - Some common routines held in one place for consistency.
;;
;; We're not going to go extreme and save every byte so some of the code
;; here is not used by every binary we embed.
;;


;; Test that we're running under cpmulator by calling the
;; "is cpmulator" function.
;;
;; If we're running under out emulator, return.
;;
;; Otherwise show an error message and exit the process.
exit_if_not_cpmulator:
        ld hl, 0x0000
        ld a, 31
        out (0xff), a

        ;; We expect SKX to appear in registers HLA
        cp 'X'
        jr nz, not_cpmulator

        ld a, h
        cp 'S'
        jr nz, not_cpmulator

        ld a, l
        cp 'K'
        ret z

        ;; Fall-through
not_cpmulator:
        ld de, WRONG_EMULATOR
        ld  c, BDOS_OUTPUT_STRING
        call   BDOS_ENTRY_POINT
        ;; exit
        rst 0x00

WRONG_EMULATOR:
        db "This binary is not running under cpmulator, aborting.", 0x0a, 0x0d, "$"



;; Output the string pointed to by HL to the console.
;;
;; Continue printing until a NULL by has been returned.
print_string:
        ld a, (hl)            ; Get the character
        cp 0                  ; Is it null?
        ret z                 ; If so return
        inc hl
          push hl             ; Save our index
          ld      c, 0x02     ; setup for printing
          ld e, a
          call    0x0005      ; print the character
          pop hl
        jr print_string       ; repeat, forever.
