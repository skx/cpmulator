// cpmulator entry-point / driver

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/skx/cpmulator/cpm"
	cpmio "github.com/skx/cpmulator/io"
)

// log holds our logging handle
var log *slog.Logger

// restoreEcho is designed to ensure we leave our terminal in a good state,
// when we terminate, by enabling console-echoing if it had been disabled.
func restoreEcho() {
	// Use our I/O package
	obj := cpmio.New(log)
	obj.Restore()
}

func main() {

	defer restoreEcho()
	//
	// Parse the command-line flags for this driver-application
	//
	cd := flag.String("cd", "", "Change to this directory before launching")
	createDirectories := flag.Bool("create", false, "Create subdirectories on the host computer for each CP/M drive.")
	useDirectories := flag.Bool("directories", false, "Use subdirectories on the host computer for CP/M drives.")
	logPath := flag.String("log-path", "", "Specify the file to write debug logs to.")
	prnPath := flag.String("prn-path", "print.log", "Specify the file to write printer-output to.")
	syscalls := flag.Bool("syscalls", false, "List the syscalls we implement.")
	flag.Parse()

	// Are we dumping syscalls?
	if *syscalls {

		// Create helper
		c := cpm.New(nil, "print.log")

		// Get syscalls in sorted order
		ids := []int{}
		for i := range c.Syscalls {
			ids = append(ids, int(i))
		}
		sort.Ints(ids)

		fmt.Printf("BDOS\n")
		for id := range ids {
			ent := c.Syscalls[uint8(id)]
			fake := ""
			if ent.Fake {
				fake = "FAKE"
			}
			fmt.Printf("\t%02d %-20s %s\n", int(id), ent.Desc, fake)
		}
		fmt.Printf("BIOS\n")
		fmt.Printf("\t00  BOOT                FAKE\n")
		fmt.Printf("\t01  WBOOT               FAKE\n")
		fmt.Printf("\t02  CONST\n")
		fmt.Printf("\t03  CONIN\n")
		fmt.Printf("\t04  CONOUT\n")

		return
	}
	// Default program to execute, and arguments to pass to program
	program := ""
	args := []string{}

	// If we have a program
	if len(flag.Args()) > 0 {
		program = flag.Args()[0]
		if len(flag.Args()) > 1 {
			args = flag.Args()[1:]
		}
	}

	// Setup our logging level - default to warnings or higher.
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelWarn)

	// The default log behaviour is to show critical issues to STDERR
	logFile := os.Stderr

	// But if we have a logfile, we'll write there
	if *logPath != "" {

		var err error
		logFile, err = os.OpenFile(*logPath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("failed to open logfile for writing %s:%s\n", *logPath, err)
			return
		}

		// And that will trigger more verbose output
		lvl.Set(slog.LevelDebug)

		defer logFile.Close()
	}

	// Create our logging handler, using the level we've just setup.
	log = slog.New(
		slog.NewJSONHandler(
			logFile,
			&slog.HandlerOptions{
				Level: lvl,
			}))

	// Create a new emulator.
	obj := cpm.New(log, *prnPath)

	// change directory?
	if *cd != "" {
		err := os.Chdir(*cd)
		if err != nil {
			fmt.Printf("failed to change to %s:%s\n", *cd, err)
			return
		}
	}

	// Should we create child-directories?  If so, do so.
	if *createDirectories {
		for _, d := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
			if _, err := os.Stat(d); os.IsNotExist(err) {
				_ = os.Mkdir(d, 0755)
			}
		}
	}

	// Are we using drives?
	if *useDirectories {

		// Enable drives
		obj.SetDrives(true)

		// Count how many drives exist - if zero show a warning
		found := 0
		for _, d := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
			if _, err := os.Stat(d); err == nil {
				found++
			}
		}
		if found == 0 {
			fmt.Printf("WARNING: You've chosen to  directories as drives.\n")
			fmt.Printf("         i.e. A/ would be used for the contents of A:\n")
			fmt.Printf("         i.e. B/ would be used for the contents of B:\n")
			fmt.Printf("\n")
			fmt.Printf("No drive-directories are present, you could fix this:\n")
			fmt.Printf("         mkdir A\n")
			fmt.Printf("         mkdir B\n")
			fmt.Printf("         mkdir C\n")
			fmt.Printf("         etc\n")
			fmt.Printf("\n")
			fmt.Printf("Run this program with '-create' to automatically create these directories.\n")
		}
	}

	// Load the binary, if we were given one.
	if program != "" {

		err := obj.LoadBinary(program)
		if err != nil {
			fmt.Printf("Error loading program %s:%s\n", program, err)
			return
		}

		err = obj.Execute(args)
		if err != nil {

			// Deliberate stop of execution
			if err == cpm.ErrHalt {
				return
			}

			// Deliberate stop of execution.
			if err == cpm.ErrExit {
				return
			}

			fmt.Printf("Error running %s [%s]: %s\n",
				program, strings.Join(args, ","), err)
		}
		return

	}

	// We load and re-run eternally - because many binaries the CCP
	// would launch would end with "exit" which would otherwise cause
	// our emulation to terminate
	//
	// Large binaries would also overwrite the CCP in RAM, so we can't
	// just jump back to the entry-point for that.
	//
	for {
		// Load the CCP binary - reseting RAM
		obj.LoadCCP()

		// Run the CCP, which will often load a child-binary.
		// The child-binary will call "P_TERMCPM" which will cause
		// the CCP to terminate.
		err := obj.Execute(args)
		if err != nil {

			// Deliberate stop of execution.
			if err == cpm.ErrHalt {
				return
			}

			fmt.Printf("\nError running CCP: %s\n", err)
			return
		}
	}
}
