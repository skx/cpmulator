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
# Run the end to end tests
#
.PHONY: test
test:
	./test/run-tests.sh

#
# Run the golang tests
#
.PHONY: tests
tests:
	go test ./...


#
# cpmulator is fast to build.
#
cpmulator: $(wildcard *.go */*.go) ccp
	go build .
