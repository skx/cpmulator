# CCP

This directory contains the CCP we bundle within our emulator:

* [ccp.asm](ccp.asm)
  * The source-code, to be compiled by [pasmo]() with the included `Makefile`.
* [ccp.bin](ccp.bin)
  * The compiled binary, which is embedded in `cpmulator`, via [ccp.go](ccp.go).

CCP stands for "console command processor" and is basically the "shell".
