
#
# The files we wish to generate.
#
all: A/\#.COM A/CCP.COM A/CONSOLE.COM A/CTRLC.COM A/DEBUG.COM A/QUIET.COM

# cleanup
clean:
	rm A/*.COM

#
# How to build them all - repetitive.
#
A/\#.COM:
	pasmo comment.z80 A/#.COM

A/CCP.COM:
	pasmo ccp.z80 A/CCP.COM

A/CONSOLE.COM:
	pasmo console.z80 A/CONSOLE.COM

A/CTRLC.COM:
	pasmo ctrlc.z80 A/CTRLC.COM

A/DEBUG.COM:
	pasmo debug.z80 A/DEBUG.COM

A/QUIET.COM:
	pasmo quiet.z80 A/QUIET.COM
