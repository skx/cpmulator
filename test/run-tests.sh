#!/bin/bash
#
# Driver: Run all test-cases.
#


#
# For each test-case we see
#
for i in test/*.in ; do

    # Get the name, and run that test
    name=$(basename "$i" .in)

    ./test/run-test.sh "${name}"

    if [ $? -eq 1 ]; then
        echo "Previous test, ${name}, failed.  Aborting"
        exit 1
    fi

done
