
#
# The files we wish to generate.
#
all: cli-args.com create.com \
     delete.com filesize.com find.com \
     intest.com read.com ret.com tsize.com \
     write.com


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
