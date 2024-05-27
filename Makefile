###
##  Hacky makefile which does too much recompilation.
##
##  "go build" / "go install" will do the right thing, unless
## you're changing the CCP or modifying the static binaries we
## added.
##
###


ALL: ccp static cpmulator


#
# CCP is fast to build.
#
.PHONY: ccp
ccp: $(wildcard ccp/*.ASM)
	cd ccp && make

#
# Static helpers are fast to build.
#
.PHONY: static
static: $(wildcard ccp/*.z80)
	cd static && make


#
# cpmulator is fast to build.
#
cpmulator: $(wildcard *.go */*.go)
	go build .
