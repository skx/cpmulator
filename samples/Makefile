
#
# The files we wish to generate.
#
all: cli-args.com


#
# Cleanup
#
clean:
	rm *.com


#
# Convert an .asm file into a .com file, via pasmo.
#
%.com: %.z80
	pasmo $< $@
