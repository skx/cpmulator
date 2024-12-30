// cpmulator entry-point / driver

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"

	cpmccp "github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/cpm"
	"github.com/skx/cpmulator/static"
	cpmver "github.com/skx/cpmulator/version"
)

var (

	// log holds our logging handle
	log *slog.Logger
)

func main() {

	//
	// Parse the command-line flags for this driver-application
	//
	ccp := flag.String("ccp", "ccpz", "The name of the CCP that we should run (ccp vs. ccpz).")
	cd := flag.String("cd", "", "Change to this directory before launching")
	console := flag.String("console", "", "The name of the console output driver to use (-list-output-drivers will show valid choices).")
	createDirectories := flag.Bool("create", false, "Create subdirectories on the host computer for each CP/M drive.")
	input := flag.String("input", cpm.DefaultInputDriver, "The name of the console input driver to use (-list-input-drivers will show valid choices).")
	output := flag.String("output", cpm.DefaultOutputDriver, "The name of the console output driver to use (-list-output-drivers will show valid choices).")
	logAll := flag.Bool("log-all", false, "Log all function invocations, including the noisy console I/O ones.")
	logPath := flag.String("log-path", "", "Specify the file to write debug logs to.")
	prnPath := flag.String("prn-path", "print.log", "Specify the file to write printer-output to.")
	showVersion := flag.Bool("version", false, "Report our version, and exit.")
	useDirectories := flag.Bool("directories", false, "Use subdirectories on the host computer for CP/M drives.")

	// listing
	listCcps := flag.Bool("list-ccp", false, "Dump the list of embedded CCPs, and exit.")
	listOutput := flag.Bool("list-output-drivers", false, "Dump the list of valid console output drivers, and exit.")
	listInput := flag.Bool("list-input-drivers", false, "Dump the list of valid console input drivers, and exit.")
	listSyscalls := flag.Bool("list-syscalls", false, "Dump the list of implemented BIOS/BDOS syscall functions, and exit.")

	// drives
	drive := make(map[string]*string)
	drive["A"] = flag.String("drive-a", "", "The path to the directory for A:")
	drive["B"] = flag.String("drive-b", "", "The path to the directory for B:")
	drive["C"] = flag.String("drive-c", "", "The path to the directory for C:")
	drive["D"] = flag.String("drive-d", "", "The path to the directory for D:")
	drive["E"] = flag.String("drive-e", "", "The path to the directory for E:")
	drive["F"] = flag.String("drive-f", "", "The path to the directory for F:")
	drive["G"] = flag.String("drive-g", "", "The path to the directory for G:")
	drive["H"] = flag.String("drive-h", "", "The path to the directory for H:")
	drive["I"] = flag.String("drive-i", "", "The path to the directory for I:")
	drive["J"] = flag.String("drive-j", "", "The path to the directory for J:")
	drive["K"] = flag.String("drive-k", "", "The path to the directory for K:")
	drive["L"] = flag.String("drive-l", "", "The path to the directory for L:")
	drive["M"] = flag.String("drive-m", "", "The path to the directory for M:")
	drive["N"] = flag.String("drive-n", "", "The path to the directory for N:")
	drive["O"] = flag.String("drive-o", "", "The path to the directory for O:")
	drive["P"] = flag.String("drive-p", "", "The path to the directory for P:")

	flag.Parse()

	// Are we dumping CCPs?
	if *listCcps {
		x := cpmccp.GetAll()
		for _, x := range x {
			fmt.Printf("%5s %-10s %04X bytes, entry-point %04X\n", x.Name, x.Description, len(x.Bytes), x.Start)
		}
		return
	}

	// Are we dumping console input drivers?
	if *listInput {
		obj, _ := consolein.New("null")
		valid := obj.GetDrivers()
		slices.Sort(valid)

		for _, name := range valid {
			suffix := ""
			if name == cpm.DefaultInputDriver {
				suffix = "\t[default]"
			}
			fmt.Printf("%s%s\n", name, suffix)
		}
		return
	}

	// Are we dumping console output drivers?
	if *listOutput {
		obj, _ := consoleout.New("null")
		valid := obj.GetDrivers()

		slices.Sort(valid)

		for _, name := range valid {
			suffix := ""
			if name == cpm.DefaultOutputDriver {
				suffix = "\t[default]"
			}
			fmt.Printf("%s%s\n", name, suffix)
		}
		return
	}

	// Are we dumping syscalls?
	if *listSyscalls {

		// dumper is a helper to dump the contents of
		// the given map in a human readable fashion.
		dumper := func(name string, arg map[uint8]cpm.CPMHandler) {

			// Get the syscalls in sorted order
			ids := []int{}
			for i := range arg {
				ids = append(ids, int(i))
			}
			sort.Ints(ids)

			// Now show them.
			fmt.Printf("%s syscalls:\n", name)
			for _, id := range ids {
				ent := arg[uint8(id)]
				fake := ""
				if ent.Fake {
					fake = "FAKE"
				}
				fmt.Printf("\t%03d %-20s %s\n", int(id), ent.Desc, fake)
			}
		}

		// Create helper - with defaults.
		c, err := cpm.New()
		if err != nil {
			fmt.Printf("error creating CPM object: %s\n", err)
			return
		}

		dumper("BDOS", c.BDOSSyscalls)
		dumper("BIOS", c.BIOSSyscalls)
		return
	}

	// show version
	if *showVersion {
		fmt.Printf("%s\n", cpmver.GetVersionBanner())
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

	// Set the logger now we've updated as appropriate.
	slog.SetDefault(log)

	// We used to use "-console", but now we prefer "-output" to match with "-input".
	out := *output
	if *console != "" {
		out = *console
		fmt.Printf("WARNING: -console is a deprecated flag, prefer to use -output.\r\n")
	}

	// Create a new emulator.
	obj, err := cpm.New(cpm.WithPrinterPath(*prnPath),
		cpm.WithOutputDriver(out),
		cpm.WithInputDriver(*input),
		cpm.WithCCP(*ccp))
	if err != nil {
		fmt.Printf("error creating CPM object: %s\n", err)
		return
	}

	// Are we logging noisy functions?
	if *logAll {
		obj.LogNoisy()
	}

	// I/O SETUP
	obj.IOSetup()

	// I/O TearDown
	// When we're finishing we'll reset some (console) state.
	defer obj.IOTearDown()

	// change directory?
	//
	// NOTE: We deliberately do this after setting up the logfile.
	if *cd != "" {
		err := os.Chdir(*cd)
		if err != nil {
			fmt.Printf("failed to change to %s:%s\n", *cd, err)
			return
		}
	}

	// Should we create child-directories?  If so, do so.
	//
	// NOTE: This is also done deliberately after the changing
	// of directory.
	if *createDirectories {
		for _, d := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
			if _, err := os.Stat(d); os.IsNotExist(err) {
				_ = os.Mkdir(d, 0755)
			}
		}
	}

	// Load any embedded files within our binary
	obj.SetStaticFilesystem(static.GetContent())

	// Default to not using subdirectories for drives
	obj.SetDrives(false)

	// Are we using drives?
	if *useDirectories {

		// Enable the use of directories.
		obj.SetDrives(true)

		// Count how many drives exist - if zero show a warning
		found := 0
		for _, d := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
			if _, err := os.Stat(d); err == nil {
				found++
			}
		}
		if found == 0 {
			fmt.Printf("WARNING: You've chosen to use subdirectories as drives.\r\n")
			fmt.Printf("         i.e. A/ would be used for the contents of A:\r\n")
			fmt.Printf("         i.e. B/ would be used for the contents of B:\r\n")
			fmt.Printf("\r\n")
			fmt.Printf("         However no drive-directories are present!\r\n")
			fmt.Printf("\r\n")
			fmt.Printf("You could fix this, like so:\r\n")
			fmt.Printf("         mkdir A\r\n")
			fmt.Printf("         mkdir B\r\n")
			fmt.Printf("         mkdir C\r\n")
			fmt.Printf("         etc\r\n")
			fmt.Printf("\r\n")
			fmt.Printf("Or you could launch this program with the '-create' flag.\r\n")
			fmt.Printf("That would automatically create directories for drives A-P.\r\n")
		}
	}

	// Do we have custom paths?  If so set them.
	for d, pth := range drive {
		if pth != nil && *pth != "" {
			obj.SetDrivePath(d, *pth)
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
				fmt.Printf("\n")
				return
			}

			// Reboot attempt, also fine
			if err == cpm.ErrBoot {
				fmt.Printf("\n")
				return
			}

			// Deliberate stop of execution.
			if err == cpm.ErrExit {
				fmt.Printf("\n")
				return
			}

			fmt.Printf("Error running %s [%s]: %s\n",
				program, strings.Join(args, ","), err)
		}

		fmt.Printf("\n")
		return
	}

	// Show a startup-banner.
	fmt.Printf("\ncpmulator %s\r\nCCP:%s input driver:%s output driver:%s\n", cpmver.GetVersionString(), obj.GetCCPName(), obj.GetInputDriver().GetName(), obj.GetOutputDriver().GetName())

	// We will load AUTOEXEC.SUB, once, if it exists (*)
	//
	// * - Terms and conditions apply.
	obj.RunAutoExec()

	// We load and re-run eternally - because many binaries the CCP
	// would launch would end with "exit" which would otherwise cause
	// our emulation to terminate
	//
	// Large binaries would also overwrite the CCP in RAM, so we can't
	// just jump back to the entry-point for that.
	//
	for {

		// Load the CCP binary - resetting RAM in the process.
		err := obj.LoadCCP()
		if err != nil {
			fmt.Printf("error loading CCP: %s\n", err)
			return
		}

		// Run the CCP, which will often load a child-binary.
		// The child-binary will call "P_TERMCPM" which will cause
		// the CCP to terminate.
		err = obj.Execute(args)
		if err != nil {

			// Start the loop again, which will reload the CCP
			// and jump to it.  Effectively rebooting.
			if err == cpm.ErrBoot {
				continue
			}

			// Deliberate stop of execution.
			if err == cpm.ErrHalt {
				fmt.Printf("\n")
				return
			}

			fmt.Printf("\nError running CCP: %s\n", err)
			return
		}
	}
}
