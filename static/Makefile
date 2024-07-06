
#
# The files we wish to generate.
#
all: A/\#.COM A/!CCP.COM A/!CONSOLE.COM A/!CTRLC.COM A/!DEBUG.COM 

# cleanup
clean:
	rm A/*.COM

#
# How to build them all - repetitive.
#
A/\#.COM: comment.z80
	pasmo comment.z80 A/#.COM

A/!CCP.COM: ccp.z80
	pasmo ccp.z80 A/!CCP.COM

A/!CONSOLE.COM: console.z80
	pasmo console.z80 A/!CONSOLE.COM

A/!CTRLC.COM: ctrlc.z80
	pasmo ctrlc.z80 A/!CTRLC.COM

A/!DEBUG.COM: debug.z80
	pasmo debug.z80 A/!DEBUG.COM

