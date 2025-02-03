# Test

This directory contains a some simple test-cases which can be used upon a Linux/MacOS host to run functional tests of the emulator.

The general approach is to fake input to the emulator, such that we tell it to run "A:FOO", or similar, and confirm we get the output we expect.



## Assumptions

* For this to work you'll need to have `cpm-dist` cloned in the same parent-directory to this repository.
* You'll need to create a suitable pre-cooked input file.



## Usage

To run all tests, from the parent-directory, execute:

```
$ ./test/run-tests.sh
```

To run a specific test:

```
$ ./test/run-test.sh zork1
```



## Implementation

The idea is that to write a new test there will be two files created:

* `foo.in`
  * This will contain the input which the test should pass to the emulator.
  * **NOTE**: That a `#` character will insert a 1 second delay in input reading.
* `foo.pat`
  * This file contains one regular expression per line.
  * If the output of running the test DOES NOT MATCH each pattern that is a fatal error.
  * **NOTE**: Don't forget the trailing newline.



## Configuration

We've had to had some configuration to the test-cases to work with assumptions, at the moment
the only real config options relate to newline handling.

* `newline` specifies which kind of line-endings to send:
  * "n"
    * A newline is \n
  * "m"
    * A newline is Ctrl-M
  * "both"
    * A newline is the pair "Ctrl-m" & "\n", in that order.
* `pause-on-newline:true`
  * Will sleep for five seconds after sending a newline.
