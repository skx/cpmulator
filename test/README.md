# Test

This directory contains a simple testing script which can be used upon a Linux/MacOS host to run functional tests of the emulator.

The general approach is to fake input to the emulator, such that we tell it to run "A:FOO", or similar, and confirm we get the output we expect.



## Assumptions

* For this to work you'll need to have `cpm-dist` cloned in the same parent-directory to this repository.
* You'll need to create a suitable pre-cooked input file.



## Usage

From the parent-directory just run the script:

```
$ ./test/test.sh
..
ALL TESTS GOOD!
```
