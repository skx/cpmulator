// Package CPM is the main package for our emulator, it uses memory to
// emulate execution of things at the bios level
package cpm

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/koron-go/z80"
	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
)

var (
	// ErrExit will be used to handle a CP/M binary calling Exit.
	//
	// It should be handled and expected by callers.
	ErrExit = errors.New("EXIT")

	// ErrUnimplemented will be used to handle a CP/M binary calling an unimplemented syscall.
	//
	// It should be handled and expected by callers.
	ErrUnimplemented = errors.New("UNIMPLEMENTED")
)

// CPMHandlerType contains the signature of a CP/M bios function
type CPMHandlerType func(cpm *CPM) error

// CPMHandler contains details of a specific call we implement.
type CPMHandler struct {
	// Desc contain the human-readable description of the given CP/M syscall.
	Desc string

	// Handler contains the function which should be involved for this syscall.
	Handler CPMHandlerType
}

// CPM is the object that holds our emulator state
type CPM struct {

	// Syscalls contains the syscalls we know how to emulate.
	Syscalls map[uint8]CPMHandler

	// Memory contains the memory the system runs with.
	Memory *memory.Memory

	// CPU contains our emulated CPU
	CPU z80.CPU

	// currentDrive contains the currently selected drive.
	// Valid values are 00-15, where
	// 0  -> A:
	// 1  -> B:
	// 15 -> P:
	currentDrive uint8

	// userNumber contains the current user number.
	//
	// Valid values are 00-15
	userNumber uint8

	// fileIsOpen records whether we have an open file.
	fileIsOpen bool

	// file has the handle to the open file, if fileIsOpen is true.
	file *os.File

	// findFirstResults is a sneaky cache of files that match a glob.
	//
	// For finding files CP/M uses "find first" to find the first result
	// then allows the programmer to call "find next", to continue the searching.
	//
	// This means we need to track state, the way we do this is to store the
	// results here, and bump the findOffset each time find-next is called.
	findFirstResults []string
	findOffset       int

	// Reader is where we get our STDIN from
	Reader *bufio.Reader

	// Filename holds the binary we're executing
	Filename string

	// Logger holds a logger, if null then no logs will be kept
	Logger *slog.Logger
}

// New returns a new emulation object
func New(filename string, logger *slog.Logger) *CPM {

	//
	// Create and populate our syscall table
	//
	sys := make(map[uint8]CPMHandler)
	sys[0] = CPMHandler{
		Desc:    "P_TERMCPM",
		Handler: SysCallExit,
	}
	sys[1] = CPMHandler{
		Desc:    "C_READ",
		Handler: SysCallReadChar,
	}
	sys[2] = CPMHandler{
		Desc:    "C_WRITE",
		Handler: SysCallWriteChar,
	}
	sys[6] = CPMHandler{
		Desc:    "C_RAWIO",
		Handler: SysCallRawIO,
	}
	sys[9] = CPMHandler{
		Desc:    "C_WRITESTRING",
		Handler: SysCallWriteString,
	}
	sys[10] = CPMHandler{
		Desc:    "C_READSTRING",
		Handler: SysCallReadString,
	}
	sys[14] = CPMHandler{
		Desc:    "DRV_SET",
		Handler: SysCallDriveSet,
	}
	sys[15] = CPMHandler{
		Desc:    "F_OPEN",
		Handler: SysCallFileOpen,
	}
	sys[16] = CPMHandler{
		Desc:    "F_CLOSE",
		Handler: SysCallFileClose,
	}
	sys[17] = CPMHandler{
		Desc:    "F_SFIRST",
		Handler: SysCallFindFirst,
	}
	sys[18] = CPMHandler{
		Desc:    "F_SNEXT",
		Handler: SysCallFindNext,
	}
	sys[19] = CPMHandler{
		Desc:    "F_DELETE",
		Handler: SysCallDeleteFile,
	}
	sys[22] = CPMHandler{
		Desc:    "F_MAKE",
		Handler: SysCallMakeFile,
	}
	sys[25] = CPMHandler{
		Desc:    "DRV_GET",
		Handler: SysCallDriveGet,
	}
	sys[31] = CPMHandler{
		Desc:    "DRV_DPB",
		Handler: SysCallGetDriveDPB,
	}
	sys[32] = CPMHandler{
		Desc:    "F_USERNUM",
		Handler: SysCallUserNumber,
	}
	sys[33] = CPMHandler{
		Desc:    "F_READRAND",
		Handler: SysCallReadRand,
	}

	// Create the object
	tmp := &CPM{
		Filename: filename,
		Logger:   logger,
		Reader:   bufio.NewReader(os.Stdin),
		Syscalls: sys,
	}
	return tmp
}

// Execute executes our named binary, with the specified arguments.
//
// The function will not return until the process being executed terminates,
// and any error will be returned.
func (cpm *CPM) Execute(args []string) error {

	// Create 64K of memory, full of NOPs
	cpm.Memory = new(memory.Memory)

	// Load our binary into it
	err := cpm.Memory.LoadFile(cpm.Filename)
	if err != nil {
		return (fmt.Errorf("failed to load %s: %s", cpm.Filename, err))
	}

	// Convert our array of CLI arguments to a string.
	cli := strings.Join(args, " ")
	cli = strings.TrimSpace(strings.ToUpper(cli))

	//
	// By default any command-line arguments need to be copied
	// to 0x0080 - as a pascal-prefixed string.
	//
	// If there are arguments the default FCBs need to be updated
	// appropriately too.
	//
	// Default to emptying the FCBs and leaving the CLI args empty.
	//
	// DMA area / CLI Args
	cpm.Memory.Set(0x0080, 0x00)
	cpm.Memory.FillRange(0x0081, 31, 0x00)

	// FCB1: Default drive, spaces for filenames.
	cpm.Memory.Set(0x005C, 0x00)
	cpm.Memory.FillRange(0x005C+1, 11, ' ')

	// FCB2: Default drive, spaces for filenames.
	cpm.Memory.Set(0x006C, 0x00)
	cpm.Memory.FillRange(0x006C+1, 11, ' ')

	// Now setup FCB1 if we have a first argument
	if len(args) > 0 {
		x := fcb.FromString(args[0])
		cpm.Memory.PutRange(0x005C, x.AsBytes()[:]...)
	}

	// Now setup FCB2 if we have a second argument
	if len(args) > 1 {
		x := fcb.FromString(args[1])
		cpm.Memory.PutRange(0x006C, x.AsBytes()[:]...)
	}

	// Poke in the CLI argument as a Pascal string.
	// (i.e. length prefixed)
	if len(cli) > 0 {

		// Setup the CLI arguments - these are set as a pascal string
		// (i.e. first byte is the length, then the data follows).
		cpm.Memory.Set(0x0080, uint8(len(cli)))
		for i, c := range cli {
			cpm.Memory.PutRange(0x0081+uint16(i), uint8(c))
		}
	}

	// Create the CPU, pointing to our memory
	// starting point for PC will be the binary entry-point
	cpm.CPU = z80.CPU{
		States: z80.States{SPR: z80.SPR{PC: 0x100}},
		Memory: cpm.Memory,
	}

	// Setup a breakpoint on 0x0005
	// That's the BIOS entrypoint
	cpm.CPU.BreakPoints = map[uint16]struct{}{}
	cpm.CPU.BreakPoints[0x05] = struct{}{}

	// Run forever :)
	for {

		// Run until we hit an error
		err := cpm.CPU.Run(context.Background())

		// No error?  Then end - the CPU hit a HALT.
		if err == nil {
			return nil
		}

		// An error which wasn't a breakpoint?  Give up
		if err != z80.ErrBreakPoint {
			return fmt.Errorf("unexpected error running CPU %s", err)
		}

		// OK we have a breakpoint error to handle.
		//
		// That means we have a CP/M BIOS function to emulate, the syscall
		// identifier is stored in the C-register.  Get it.
		syscall := cpm.CPU.States.BC.Lo

		//
		// Is there a syscall entry for this number?
		//
		handler, exists := cpm.Syscalls[syscall]

		if exists {
			cpm.Logger.Info("Calling BIOS emulation",
				slog.String("name", handler.Desc),
				slog.Int("syscall", int(syscall)),
				slog.String("syscallHex", fmt.Sprintf("0x%02X", syscall)),
			)

			err := handler.Handler(cpm)
			if err == ErrExit {
				return nil
			}
			if err != nil {
				return err
			}

			// Return from call
			cpm.CPU.PC = cpm.Memory.GetU16(cpm.CPU.SP)
			// pop stack back.  Fun
			cpm.CPU.SP += 2

		} else {

			// Unknown opcode
			cpm.Logger.Error("Unimplemented syscall",
				slog.Int("syscall", int(syscall)),
				slog.String("syscallHex", fmt.Sprintf("0x%02X", syscall)),
			)
			return ErrUnimplemented
		}
	}
}
