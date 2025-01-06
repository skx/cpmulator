
#
# The files we wish to generate.
#
all: A/\#.COM A/!CCP.COM A/!CTRLC.COM A/!DEBUG.COM A/!HOSTCMD.COM A/!INPUT.COM A/!OUTPUT.COM A/!VERSION.COM

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

A/!CTRLC.COM: ctrlc.z80
	pasmo ctrlc.z80 A/!CTRLC.COM

A/!DEBUG.COM: debug.z80
	pasmo debug.z80 A/!DEBUG.COM

A/!HOSTCMD.COM: hostcmd.z80
	pasmo hostcmd.z80 A/!HOSTCMD.COM

A/!INPUT.COM: input.z80
	pasmo input.z80 A/!INPUT.COM

A/!OUTPUT.COM: output.z80
	pasmo output.z80 A/!OUTPUT.COM

A/!VERSION.COM: version.z80
	pasmo version.z80 A/!VERSION.COM
