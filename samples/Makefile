
#
# The files we wish to generate.
#
all: ccp.com cli-args.com console.com create.com \
     ctrlc.com delete.com drive.com find.com \
     intest.com quiet.com read.com ret.com test.com \
     user-num.com version.com write.com


#
# Cleanup
#
clean:
	rm *.com


#
# Convert an .asm file into a .com file, via pasmo.
#
%.com: %.z80
	pasmo "$<" "$@"
