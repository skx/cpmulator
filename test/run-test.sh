#!/bin/bash
#
# Run a single test:
#
#  1.  feed some pre-cooked input to CPMUlater
#
#  2.  Collect all the output.
#
#  3.  Ensure that every line of patterns we expect was found in the output
#


#
# Sanity-check
#
if [ ! -d ../cpm-dist ] ; then
    echo "cpm-dist is not present at ../cpm-dist"
    echo "Aborting"
    exit 1
fi

if [ ! -x "$(command -v ansifilter)" ]; then
    echo "ansifilter is not in your \$PATH"
    echo "  brew install ansifilter"
    echo "  apt-get install ansifilter"
    echo " etc"
    exit 1
fi

echo "Running test case: $1"
input=test/$1.in
pattern=test/$1.pat
output=test/$1.out


#
# Ensure we have a test-input and a set of patterns
#
if [ ! -e "$input" ] ; then
    echo "  TATAL: Test $1 has no input."
    exit 1
fi

if [ ! -e "$pattern" ] ; then
    echo "  TATAL: Test $1 has no patterns to look for."
    exit 1
fi


#
# Remove any output from previous runs.
#
if [ -e "$output" ] ; then
    rm "$output"
fi


#
# Spawn run the emulator with the cooked input.
#
export INPUT_FILE="${input}"
start=$(date +%s)
echo " Starting $(date)"
./cpmulator -input file  -cd ../cpm-dist/ -directories -timeout 120 -ccp ccp | ansifilter &> "$output"
end=$(date +%s)
runtime=$((end-start))
echo " Completed in ${runtime} seconds"


#
#  Test that the patterns we expect are present in the output.
#
while read -r line; do
    if ! grep -q "$line" "$output"; then
        echo "  FAIL: $line"
        echo "  Test output saved in $output"
        exit 1
    else
        echo "  OK: $line"
    fi
done < "$pattern"


#
# All done
#
exit 0
