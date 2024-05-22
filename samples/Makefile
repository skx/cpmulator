
#
# The files we wish to generate.
#
all: cli-args.com ctrlc.com create.com delete.com drive.com find.com intest.com read.com ret.com user-num.com version.com write.com


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
