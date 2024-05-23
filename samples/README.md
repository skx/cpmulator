# Sample CP/M Binaries

This directory contains some sample code, written in Z80 assembly.

The top-level [dist/](../dist) directory contains more useful programs, including the various ZORK games, although some binaries here are useful for users.



## cpmulator Specific Utilities

I've implemented [a custom BIOS function](../EXTENSIONS.md), with a number of extensions, and these extensions are useful within the CCP.

The following binaries are designed to work with the extensions and make functional/useful changes to the CCP environment:

* [ccp.z80](ccp.asm)
  * Change the CCP in-use at runtime.
* [console.z80](console.z80)
  * Toggle between ADM-3A and ANSI console output.
* [ctrlc.z80](ctrlc.z80)
  * By default we reboot the CCP whenever the user presses Ctrl-C twice in a row.
  * Here you can tweak that behaviour to change the number of consecutive Ctrl-Cs that will reboot.
    * Require only a single Ctrl-C (`ctrlc 1`)
    * Disable the Ctrl-C reboot behaviour entirely (`ctrlc 0`)
* [test.z80](test.z80)
  * A program that determines whether it is running under cpmulator.
  * If so it shows the version banner.
* [quiet.z80](quiet.z80)
  * Disable the startup banner.
  * "cpmulator unreleased loaded CCP ccpz, with adm-3a output driver", or similar.



## Other Testing-Commands

These were primarily written to compare behaviour across emulators, or test my understanding of how things were supposed to work.

* [cli-args.z80](cli-args.z80)
  * Shows command-line arguments passed to binaries launched from CCP
* [create.z80](create.z80)
  * Create a file, given the name.
* [delete.z80](delete.z80)
  * Delete files matching the given name/pattern.
* [intest.z80](intest.z80)
  * Test the various character/line input methods for correctness.
* [read.z80](read.z80) & [write.z80](write.z80)
  * Write populates the named file with 255 records of fixed content.
  * Read processes the named file and aborts if the records contain surprising content.
  * Used to test sequential read/write operations.
* [ret.z80](ret.z80)
  * Terminate the execution of a binary in four different ways.
