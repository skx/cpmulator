# Embedded Resources

This directory contains some binaries which are embedded into the main `cpmulator` binary, the intention is that these binaries will always
appear present upon the local disc.  These make use of our [custom BIOS functions](../EXTENSIONS.md), making them tied to this emulator and useless without it.

Other CP/M code within the repository includes:

* The top-level [dist/](../dist) directory contains a complete program used to test the emulator.
* The top-level [samples/](../samples/) directory contains some code which was useful to me to test my understanding, when writing the emulator.

More significant programs are available within the sister-repository:

* https://github.com/skx/cpm-dist



## Contents

The embedded resources do not have 100% full functionality, you cannot bundle a game such as ZORK, because not all I/O primitives work upon them, but simple binaries to be executed by the CCP work just fine.


* [ccp.z80](ccp.z80)
  * Change the CCP in-use at runtime.
* [comment.z80](comment.z80)
  * Source to a program that does nothing, output to "`#.COM`".
  * This is used to allow "`# FOO`" to act as a comment inside submit-files.
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
