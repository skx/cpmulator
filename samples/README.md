# Sample CP/M Binaries

This directory contains some sample code, written in Z80 assembly, which was useful in testing my understanding when writing the emulator.

Other CP/M code within the repository includes:

* The top-level [dist/](../dist) directory contains a complete program used to test the emulator.
* The top-level [static/](../static/) directory contains some binaries which are always available when launching our emulator.

More significant programs are available within the sister-repository:

* https://github.com/skx/cpm-dist



## Contents

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
* [terminal.z80](terminal.z80)
  * Show the dimensions of the terminal we're running within.
* [unimpl.z80](unimpl.z80)
  * Attempt to call a BDOS function which isn't implemented.
