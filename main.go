// entry point

package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/skx/go-cpm/cpm"
)

func main() {

	// Ensure we've been given the name of a file
	if len(os.Args) < 2 {
		fmt.Printf("Usage: go-cpm path/to/file.com [args]\n")
		return
	}

	// Setup our logging level - default to warnings or higher
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelWarn)

	// But show "everything" if $DEBUG is non.empty
	if os.Getenv("DEBUG") != "" {
		lvl.Set(slog.LevelDebug)
	}

	//
	// Create our logging handler, using the level we've just setup
	//
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	//
	// Create a new emulator.
	//
	cpm := cpm.New(os.Args[1], log)

	//
	// Run the binary we've been given.
	//
	err := cpm.Execute(os.Args[2:])
	if err != nil {
		fmt.Printf("Error running %s: %s\n", os.Args[1], err)
	}
}
