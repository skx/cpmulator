# Sample CP/M Binaries

This directory contains some sample code, written in Z80 assembly, which were useful to my understanding when developing the emulator.


The top-level [dist/](../dist) directory contains more useful programs, including the various ZORK games.



## Highlights

* [ctrlc.z80](ctrlc.z80)
  * I've implemented [a custom BIOS function](../EXTENSIONS.md) to get/set the number of Ctrl-C presses which will reboot the CCP.
  * This utility invokes that function to get/set the value.
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
