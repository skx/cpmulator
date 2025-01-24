#!/bin/bash
#
# This script is designed to test that the emulator
# works "fully".
#
# It does this by pasting fake input into the generated
# binary and testing real binaries
#

#
# This is the fake input that we're going to pass to the
# emulator
#
export INPUT_FILE=test/test.in

#
# Make sure we've built the emulator.
#
go build .

#
# Make sure we have our input
#
if [ ! -e "${INPUT_FILE}" ] ; then
    echo "The test input-file, ${INPUT_FILE}, is missing.  Aborting"
    exit 1
fi

#
# Clear any previous output
#
rm -f test/test.out

#
# Run the emulator, saving the output
#
./cpmulator -input file -cd ../cpm-dist/ -directories -ccp ccp | tee test.out

#
# Now make sure we got some sensible output.
#

# Did we load the lighthouse of doom?
if ! grep -q "small torch" test.out; then
    echo "We failed to find the small torch"
    exit 1
fi

# An obvious test is to make sure that we compiled and ran the "echo" binary.
#
# We can look for this by confirming we got UPPER-CASE output.
if ! grep -q CAKE test.out; then
    echo "We failed to find 'CAKE' which implies the echo didn't work"
    exit 1
fi


echo "ALL TESTS GOOD!"
exit 0
