# CCP

CCP stands for "console command processor" and is basically the "shell".

This directory contains the two CCP flavours:



## DR CCP

* [CCP.ASM](CCP.ASM)
  * The source-code, to be compiled by pasmo with the included `Makefile`.
* `CCP.BIN`
  * The compiled binary, which is embedded in `cpmulator`, via [ccp.go](ccp.go).



## CCPZ

* [CCPZ.ASM](CCPZ.ASM)
  * The source-code which can be actually compiled using `cpmulator`, and the ASM.COM included within [cpm-dist](https://github.com/skx/cpm-dist).
* `CCPZ.BIN`
  * The compiled binary, which is embedded in `cpmulator`, via [ccp.go](ccp.go).
