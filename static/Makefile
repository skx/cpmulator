
#
# The files we wish to generate.
#
all: A/\#.COM A/!CCP.COM A/!CTRLC.COM A/!DEBUG.COM A/!DISABLE.COM A/!HOSTCMD.COM A/!INPUT.COM A/!OUTPUT.COM A/!PRNPATH.COM A/!VERSION.COM

# cleanup
clean:
	rm A/*.COM

#
# How to build them all - repetitive.
#
A/\#.COM: comment.z80
	pasmo comment.z80 A/#.COM

A/!CCP.COM: ccp.z80 common.inc
	pasmo ccp.z80 A/!CCP.COM

A/!CTRLC.COM: ctrlc.z80 common.inc
	pasmo ctrlc.z80 A/!CTRLC.COM

A/!DEBUG.COM: debug.z80 common.inc
	pasmo debug.z80 A/!DEBUG.COM

A/!DISABLE.COM: disable.z80 common.inc
	pasmo disable.z80 A/!DISABLE.COM

A/!HOSTCMD.COM: hostcmd.z80 common.inc
	pasmo hostcmd.z80 A/!HOSTCMD.COM

A/!INPUT.COM: input.z80 common.inc
	pasmo input.z80 A/!INPUT.COM

A/!OUTPUT.COM: output.z80 common.inc
	pasmo output.z80 A/!OUTPUT.COM

A/!PRNPATH.COM: prnpath.z80 common.inc
	pasmo prnpath.z80 A/!PRNPATH.COM

A/!VERSION.COM: version.z80 common.inc
	pasmo version.z80 A/!VERSION.COM
