// entry point

package main

import (
	"bufio"
	"fmt"
	"os"
)

// Reader lets us get console input
var reader *bufio.Reader

func main() {

	// Ensure we've been given the name of a file
	if len(os.Args) < 2 {
		fmt.Printf("Usage: go-cpm path/to/file.com [args]\n")
		return
	}

	// Populate the global reader
	reader = bufio.NewReader(os.Stdin)

	// Load the binary
	err := runCPM(os.Args[1], os.Args[2:])
	if err != nil {
		fmt.Printf("Error running %s: %s\n", os.Args[1], err)
	}
}
