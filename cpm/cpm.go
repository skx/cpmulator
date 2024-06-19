// Package cpm is the main package for our emulator, it uses memory to
// emulate execution of things at the bios level.
//
// The package mostly contains the implementation of the syscalls that
// CP/M programs would expect - along with a little machinery to wire up
// the Z80 emulator we're using and deal with FCB structures.
package cpm

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/koron-go/z80"
	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/consoleout"
	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
	cpmver "github.com/skx/cpmulator/version"
)

var (
	// ErrExit will be used to handle a CP/M binary calling Exit.
	//
	// It should be handled and expected by callers.
	ErrExit = errors.New("EXIT")

	// ErrHalt will be used to note that the Z80 emulator executed a HALT
	// operation, and that terminated the execution of code.
	//
	// It should be handled and expected by callers.
	ErrHalt = errors.New("HALT")

	// ErrBoot will be used to note that the Z80 emulator executed code
	// at 0x0000 - i.e. a boot attempt
	//
	// It should be handled and expected by callers.
	ErrBoot = errors.New("BOOT")

	// ErrUnimplemented will be used to handle a CP/M binary calling an unimplemented syscall.
	//
	// It should be handled and expected by callers.
	ErrUnimplemented = errors.New("UNIMPLEMENTED")
)

// CPMHandlerType contains the signature of a function we use to
// emulate a CP/M BIOS or BDOS function.
//
// It is not expected that outside packages will want to add custom BIOS
// functions, or syscalls, but this is public so that it could be done if
// necessary.
type CPMHandlerType func(cpm *CPM) error

// CPMHandler contains details of a specific call we implement.
//
// While we mostly need a "number to handler", mapping having a name
// is useful for the logs we produce, and we mark those functions that
// don't do 100% of what they should as "Fake".
type CPMHandler struct {
	// Desc contain the human-readable name of the given CP/M syscall.
	Desc string

	// Handler contains the function which should be invoked for
	// this syscall.
	Handler CPMHandlerType

	// Fake stores a quick comment on the completeness of the syscall
	// implementation.  If Fake is set to true then the syscall is
	// faked, or otherwise incompletely implemented.
	//
	// This might mean completely bogus behaviour, or it might mean
	// "good enough, even if wrong".
	Fake bool
}

// FileCache is used to cache filehandles, against FCB addresses.
//
// This is primarily done as a speed optimization.
type FileCache struct {
	// name holds the name file, when it was opened/created.
	name string

	// handle has the file object.
	handle *os.File
}

// CPM is the object that holds our emulator state.
type CPM struct {

	// biosErr holds any error created by a BIOS handler.
	//
	// We need this because the handlers we use for BIOS operations
	// cannot return an error - due to the interface used in the z80
	// emulator.
	biosErr error

	// ccp contains the name of the CCP we should load
	ccp string

	// Should we be quiet?
	quiet bool

	// files is the cache we use for File handles.
	files map[uint16]FileCache

	// virtual contains a reference to a static filesystem which
	// is embedded within our binary, if any.
	static embed.FS

	// input is our interface for reading from the console.
	//
	// This needs to take account of echo/no-echo status.
	input *consolein.ConsoleIn

	// output is used for writing characters to the conolse
	output consoleout.ConsoleDriver

	// dma contains the address of the DMA area in RAM.
	//
	// The DMA area is used for all file I/O, and is 128 bytes in length.
	dma uint16

	// prnPath contains the filename to write all printer-output to.
	prnPath string

	// start contains the location to which we load our binaries,
	// and execute them from.  This is specifically a variable because
	// while all CP/M binaries are loaded at 0x0100 the CCP we can
	// launch uses a higher location - so that it isn't overwritten by
	// the programs it launches.
	start uint16

	// BDOSSyscalls contains details of the BDOS syscalls we
	// know how to emulate, indexed by their ID.
	BDOSSyscalls map[uint8]CPMHandler

	// BIOSSyscalls contains details of the BIOS syscalls we
	// know how to emulate, indexed by their ID.
	BIOSSyscalls map[uint8]CPMHandler

	// Memory contains the memory the system runs with.
	Memory *memory.Memory

	// CPU contains a pointer to the virtual CPU we use to execute
	// code.  The CP/M we're implementing is Z80-based, so we need to
	// be able to emulate that.
	CPU z80.CPU

	// Drives specifies the local paths for each directory.
	drives map[string]string

	// currentDrive contains the currently selected drive.
	// Valid values are 0-15, where they work in the obvious way:
	// 0  -> A:
	// 1  -> B:
	// 15 -> P:
	currentDrive uint8

	// userNumber contains the current user number.
	//
	// Valid values are 0-15.
	userNumber uint8

	// findFirstResults is a sneaky cache of files that match a glob.
	//
	// For finding files CP/M uses "find first" to find the first result
	// then allows the programmer to call "find next", to continue the searching.
	//
	// This means we need to track state, the way we do this is to store the
	// results here, and bump the findOffset each time find-next is called.
	findFirstResults []fcb.FCBFind

	// findOffset contains the index into findFirstResults which is
	// to be read next.
	findOffset int

	// Logger holds a logger which we use for debugging and diagnostics.
	Logger *slog.Logger
}

// New returns a new emulation object
func New(logger *slog.Logger, prn string, condriver string, ccp string) (*CPM, error) {

	//
	// Create and populate our syscall table for the BDOS syscalls.
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
	sys[3] = CPMHandler{
		Desc:    "A_READ",
		Handler: SysCallAuxRead,
	}
	sys[4] = CPMHandler{
		Desc:    "A_WRITE",
		Handler: SysCallAuxWrite,
	}
	sys[5] = CPMHandler{
		Desc:    "L_WRITE",
		Handler: SysCallPrinterWrite,
		Fake:    true,
	}
	sys[6] = CPMHandler{
		Desc:    "C_RAWIO",
		Handler: SysCallRawIO,
	}
	sys[7] = CPMHandler{
		Desc:    "GET_IOBYTE",
		Handler: SysCallGetIOByte,
	}
	sys[8] = CPMHandler{
		Desc:    "SET_IOBYTE",
		Handler: SysCallSetIOByte,
	}
	sys[9] = CPMHandler{
		Desc:    "C_WRITESTRING",
		Handler: SysCallWriteString,
	}
	sys[10] = CPMHandler{
		Desc:    "C_READSTRING",
		Handler: SysCallReadString,
	}
	sys[11] = CPMHandler{
		Desc:    "C_STAT",
		Handler: SysCallConsoleStatus,
	}
	sys[12] = CPMHandler{
		Desc:    "S_BDOSVER",
		Handler: SysCallBDOSVersion,
	}
	sys[13] = CPMHandler{
		Desc:    "DRV_ALLRESET",
		Handler: SysCallDriveAllReset,
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
	sys[20] = CPMHandler{
		Desc:    "F_READ",
		Handler: SysCallRead,
	}
	sys[21] = CPMHandler{
		Desc:    "F_WRITE",
		Handler: SysCallWrite,
	}
	sys[22] = CPMHandler{
		Desc:    "F_MAKE",
		Handler: SysCallMakeFile,
	}
	sys[23] = CPMHandler{
		Desc:    "F_RENAME",
		Handler: SysCallRenameFile,
	}
	sys[24] = CPMHandler{
		Desc:    "DRV_LOGINVEC",
		Handler: SysCallLoginVec,
		Fake:    true,
	}
	sys[25] = CPMHandler{
		Desc:    "DRV_GET",
		Handler: SysCallDriveGet,
	}
	sys[26] = CPMHandler{
		Desc:    "F_DMAOFF",
		Handler: SysCallSetDMA,
	}
	sys[27] = CPMHandler{
		Desc:    "DRV_ALLOCVEC",
		Handler: SysCallDriveAlloc,
		Fake:    true,
	}
	sys[28] = CPMHandler{
		Desc:    "DRV_SETRO",
		Handler: SysCallDriveSetRO,
		Fake:    true,
	}
	sys[29] = CPMHandler{
		Desc:    "DRV_ROVEC",
		Handler: SysCallDriveROVec,
		Fake:    true,
	}
	sys[30] = CPMHandler{
		Desc:    "F_ATTRIB",
		Handler: SysCallSetFileAttributes,
		Fake:    true,
	}
	sys[31] = CPMHandler{
		Desc:    "DRV_DPB",
		Handler: SysCallGetDriveDPB,
		Fake:    true,
	}
	sys[32] = CPMHandler{
		Desc:    "F_USERNUM",
		Handler: SysCallUserNumber,
	}
	sys[33] = CPMHandler{
		Desc:    "F_READRAND",
		Handler: SysCallReadRand,
	}
	sys[34] = CPMHandler{
		Desc:    "F_WRITERAND",
		Handler: SysCallWriteRand,
	}
	sys[35] = CPMHandler{
		Desc:    "F_SIZE",
		Handler: SysCallFileSize,
	}
	sys[36] = CPMHandler{
		Desc:    "F_RANDREC",
		Handler: SysCallRandRecord,
	}
	sys[37] = CPMHandler{
		Desc:    "DRV_RESET",
		Handler: SysCallDriveReset,
		Fake:    true,
	}
	sys[45] = CPMHandler{
		Desc:    "F_ERRMODE",
		Handler: SysCallErrorMode,
		Fake:    true,
	}
	sys[105] = CPMHandler{
		Desc:    "T_GET",
		Handler: SysCallTime,
		Fake:    true,
	}
	sys[113] = CPMHandler{ // used by Turbo Pascal
		Desc:    "DirectScreenFunctions",
		Handler: SysCallDirectScreenFunctions,
		Fake:    true,
	}

	//
	// Create and populate our syscall table for the BIOS syscalls.
	//
	bios := make(map[uint8]CPMHandler)
	bios[0] = CPMHandler{
		Desc:    "BOOT",
		Handler: BiosSysCallBoot,
	}
	bios[1] = CPMHandler{
		Desc:    "WBOOT",
		Handler: BiosSysCallBoot,
	}
	bios[2] = CPMHandler{
		Desc:    "CONST",
		Handler: BiosSysCallConsoleStatus,
	}
	bios[3] = CPMHandler{
		Desc:    "CONIN",
		Handler: BiosSysCallConsoleInput,
	}
	bios[4] = CPMHandler{
		Desc:    "CONOUT",
		Handler: BiosSysCallConsoleOutput,
	}
	bios[5] = CPMHandler{
		Desc:    "LIST",
		Handler: BiosSysCallPrintChar,
		Fake:    true,
	}
	bios[15] = CPMHandler{
		Desc:    "LISTST",
		Handler: BiosSysCallPrinterStatus,
		Fake:    true,
	}
	bios[17] = CPMHandler{
		Desc:    "CONOST",
		Handler: BiosSysCallScreenOutputStatus,
		Fake:    true,
	}
	bios[18] = CPMHandler{
		Desc:    "AUXIST",
		Handler: BiosSysCallAuxInputStatus,
		Fake:    true,
	}
	bios[19] = CPMHandler{
		Desc:    "AUXOST",
		Handler: BiosSysCallAuxOutputStatus,
		Fake:    true,
	}
	bios[31] = CPMHandler{
		Desc:    "RESERVE1",
		Handler: BiosSysCallReserved1,
		Fake:    true,
	}

	// Output driver needs to be created
	driver, err := consoleout.New(condriver)
	if err != nil {
		return nil, err
	}

	// Create the emulator object and return it
	tmp := &CPM{
		BDOSSyscalls: sys,
		BIOSSyscalls: bios,
		Logger:       logger,
		ccp:          ccp,
		dma:          0x0080,
		drives:       make(map[string]string),
		files:        make(map[uint16]FileCache),
		input:        consolein.New(),
		output:       driver,
		prnPath:      prn,
		start:        0x0100,
	}
	return tmp, nil
}

// Cleanup cleans up the state of the terminal, if necessary.
func (cpm *CPM) Cleanup() {
	cpm.input.Reset()
}

// GetOutputDriver returns the name of our configured output driver.
func (cpm *CPM) GetOutputDriver() string {
	return cpm.output.GetName()
}

// GetCCPName returns the name of the CCP we've been configured to load.
func (cpm *CPM) GetCCPName() string {
	return cpm.ccp
}

// GetQuiet returns the status of the quiet-flag.
func (cpm *CPM) GetQuiet() bool {
	return cpm.quiet
}

// SetQuiet updates the state of the quiet-flag.
func (cpm *CPM) SetQuiet(state bool) {
	cpm.quiet = state
}

// LoadBinary loads the given CP/M binary at the default address of 0x0100,
// where it can then be launched by Execute.
func (cpm *CPM) LoadBinary(filename string) error {

	// Create 64K of memory, full of NOPs
	if cpm.Memory == nil {
		cpm.Memory = new(memory.Memory)
	}

	// Load our binary into the memory
	err := cpm.Memory.LoadFile(cpm.start, filename)
	if err != nil {
		return (fmt.Errorf("failed to load %s: %s", filename, err))
	}

	//
	// Any command-line arguments need to be copied to the DMA area,
	// which defaults to 0x0080, as a pascal-prefixed string.
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

	// patch low-memory so that RST instructions will
	// ultimately invoke our CP/M syscalls, via our "Out"
	// function.
	cpm.fixupRAM()

	return nil
}

// fixupRAM is misnamed - but it patches the RAM with Z80 code to
// handle "badly behaved" programs that invoke CP/M functions via RST XX
// instructions, rather than calls to 0x0005
//
// We put some code to call the handlers via faked OUT 0xFF,N - where N
// is the syscall to run.
//
// The precise region we patch is unimportant, but we want to make sure
// we don't overlap with our CCP, or "large programs" loaded at 0x0100.
func (cpm *CPM) fixupRAM() {
	i := 0
	CBIOS := 0xFE00
	BDOS := 0xF000
	NENTRY := 30

	SETMEM := func(a int, v int) {
		cpm.Memory.Set(uint16(a), uint8(v))
	}

	// We _should_ add "JUMP TO CCP" here, but instead
	// we terminate execution via a HALT (0x76) instruction.
	//
	// This works regardless of whether CCP is present or not.
	//
	// We do the same thing for the jump at 0x0005 because
	// turbo pascal, and other programs presumably, look at
	// the following address to see how much free RAM is available.
	//
	SETMEM(0x0000, 0x76)                 /* HALT */
	SETMEM(0x0001, ((CBIOS + 3) & 0xFF)) /* Fake address of entry-point */
	SETMEM(0x0002, ((CBIOS + 3) >> 8))

	// We setup a fake jump here, because 0x0006 is sometimes
	// used to find the free RAM and we pretend our BDOS is at 0xDC00
	SETMEM(0x0005, 0x76)                /* HALT */
	SETMEM(0x0006, ((BDOS + 6) & 0xFF)) /* Fake Address of entry point */
	SETMEM(0x0007, ((BDOS + 6) >> 8))

	// Now we setup the initial values of the I/O byte
	SETMEM(0x0003, 0x00)

	// fake BIOS entry points for 30 syscalls.
	//
	// These are setup so that the RST instructions magically
	// end up at our handlers - our Z80 emulator allows us to trap
	// IN and OUT instructions, and later you'll see that we redirect
	// OUT(0xff, N) to invoke our handler(s).
	//
	// See the function:
	//
	//     func (cpm *CPM) Out(addr uint8, val uint8)
	//
	for i < 30 {
		/* JP <bios-entry> */
		SETMEM(CBIOS+3*i, 0xC3)
		SETMEM(CBIOS+3*i+1, (CBIOS+NENTRY*3+i*5)&0xFF)
		SETMEM(CBIOS+3*i+2, (CBIOS+NENTRY*3+i*5)>>8)

		/* LD A,<bios-call> - start of bios-entry */
		SETMEM(CBIOS+NENTRY*3+i*5, 0x3E)
		SETMEM(CBIOS+NENTRY*3+i*5+1, i)

		/* OUT A,0FFH - we use port 0xFF to fake the BIOS call */
		SETMEM(CBIOS+NENTRY*3+i*5+2, 0xD3)
		SETMEM(CBIOS+NENTRY*3+i*5+3, 0xFF)

		/* RET - end of bios-entry */
		SETMEM(CBIOS+NENTRY*3+i*5+4, 0xC9)
		i++
	}

}

// LoadCCP loads the CCP into RAM, to be executed instead of an external binary.
//
// This function modifies the "start" attribute, to ensure the CCP is loaded
// and executed at a higher address than the default of 0x0100.
func (cpm *CPM) LoadCCP() error {

	// Show a startup-banner.
	if !cpm.GetQuiet() {
		fmt.Printf("\ncpmulator %s loaded CCP %s, with %s output driver\n", cpmver.GetVersionString(), cpm.GetCCPName(), cpm.GetOutputDriver())
	}

	// Create 64K of memory, full of NOPs
	if cpm.Memory == nil {
		cpm.Memory = new(memory.Memory)
	}

	//
	// Get our helper to find the CCP to load
	//
	helper, err := ccp.Get(cpm.ccp)

	if err != nil {
		return fmt.Errorf("error retrieving CCP by name: %s", err)
	}

	// Load it into memory
	cpm.Memory.SetRange(helper.Start, helper.Bytes...)

	// DMA area / CLI Args are going to be unset.
	cpm.Memory.Set(0x0080, 0x00)
	cpm.Memory.FillRange(0x0081, 31, 0x00)

	// FCB1: Default drive, spaces for filenames.
	cpm.Memory.Set(0x005C, 0x00)
	cpm.Memory.FillRange(0x005C+1, 11, ' ')

	// FCB2: Default drive, spaces for filenames.
	cpm.Memory.Set(0x006C, 0x00)
	cpm.Memory.FillRange(0x006C+1, 11, ' ')

	// Ensure our starting point is what we expect
	cpm.start = helper.Start

	// patch low-memory so that RST instructions will
	// ultimately invoke our CP/M syscalls, via our "Out"
	// function.
	cpm.fixupRAM()

	return nil
}

// Execute executes our named binary, with the specified arguments.
//
// The function will not return until the process being executed terminates,
// and any error will be returned.
func (cpm *CPM) Execute(args []string) error {

	// Reset any cached filehandles.
	//
	// This is only required when running the CCP, as there we're persistent.
	for fcb, obj := range cpm.files {
		cpm.Logger.Debug("Closing handle in FileCache",
			slog.String("path", obj.name),
			slog.Int("fcb", int(fcb)))
		obj.handle.Close()
	}
	cpm.files = make(map[uint16]FileCache)

	// Create the CPU, pointing to our memory, and setting the initial program counter
	// to point to our expected entry-point.
	cpm.CPU = z80.CPU{
		States: z80.States{
			SPR: z80.SPR{
				PC: cpm.start,
			},
		},
		Memory: cpm.Memory,
		IO:     cpm,
	}

	//
	// This is a bit of a cheat, but the CCP we're using
	// assumes that the C register contains the user-number
	// and drive number to run from when it is launched.
	//
	// If we set this to the currentDrive (which will default
	// to A) we'll maintain state despite restarts, as we reuse
	// this processing object - so drive-changes will update that
	// value from the default when it is changed.
	//
	cpm.CPU.States.BC.Lo = cpm.userNumber<<4 | cpm.currentDrive

	// Set the same value in RAM
	cpm.Memory.Set(0x0004, cpm.CPU.States.BC.Lo)

	CBIOS := uint16(0xFE00)
	BDOS := uint16(0xF000)

	// Setup our breakpoints.
	//
	// We configure two:
	//
	//  0x0000 - is the boot address of the Z80 processor.
	//  0x0005 - The CPM BDOS entrypoint.
	//
	cpm.CPU.BreakPoints = make(map[uint16]struct{})
	cpm.CPU.BreakPoints[CBIOS] = struct{}{}
	cpm.CPU.BreakPoints[CBIOS+3] = struct{}{}
	cpm.CPU.BreakPoints[BDOS] = struct{}{}
	cpm.CPU.BreakPoints[BDOS+6] = struct{}{}
	cpm.CPU.BreakPoints[0x0005] = struct{}{}

	// Convert our array of CLI arguments to a string.
	cli := strings.Join(args, " ")
	cli = strings.TrimSpace(strings.ToUpper(cli))

	// Setup FCB1 if we have a first argument
	if len(args) > 0 {
		x := fcb.FromString(args[0])
		cpm.Memory.SetRange(0x005C, x.AsBytes()...)
	}

	// Setup FCB2 if we have a second argument
	if len(args) > 1 {
		x := fcb.FromString(args[1])
		cpm.Memory.SetRange(0x006C, x.AsBytes()...)
	}

	// Poke in the CLI argument as a Pascal string.
	// (i.e. length prefixed)
	if len(cli) > 0 {

		// Setup the CLI arguments - these are set as a pascal string
		// (i.e. first byte is the length, then the data follows).
		cpm.Memory.Set(0x0080, uint8(len(cli)))
		for i, c := range cli {
			cpm.Memory.SetRange(0x0081+uint16(i), uint8(c))
		}
	}

	// Run forever :)
	for {
		// Run until we hit an error
		err := cpm.CPU.Run(context.Background())

		// If we ended up here because the I/O handler received
		// an error, and then HALTed the emulator we'll process it
		// here.
		if cpm.biosErr != nil {
			err = cpm.biosErr
			cpm.biosErr = nil
		}

		// Reboot?
		if cpm.CPU.PC == 0x0000 {
			return ErrBoot
		}

		// No error?  Then end - the CPU hit a HALT.
		if err == nil {
			return ErrHalt
		}

		// Are we being asked to terminate CP/M?  If so return
		if err == ErrExit {
			return nil
		}

		// An error which wasn't a breakpoint?  Give up
		if err != z80.ErrBreakPoint {
			return fmt.Errorf("unexpected error running CPU %s", err)
		}

		// OK we have a breakpoint error to handle.
		//
		// That means we have a CP/M BDOS function to emulate,
		// the syscall identifier is stored in the C-register.
		syscall := cpm.CPU.States.BC.Lo

		//
		// Is there a syscall entry for this number?
		//
		handler, exists := cpm.BDOSSyscalls[syscall]

		//
		// Nope: That will stop execution with a fatal log
		//
		if !exists {

			cpm.Logger.Error("Unimplemented BDOS Syscall",
				slog.Int("syscall", int(syscall)),
				slog.String("syscallHex",
					fmt.Sprintf("0x%02X", syscall)),
			)
			return ErrUnimplemented
		}

		// Log the call we're going to make
		cpm.Logger.Info("BDOS",
			slog.String("name", handler.Desc),
			slog.Int("syscall", int(syscall)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", syscall)),
			slog.Group("registers",
				slog.String("A", fmt.Sprintf("%02X", cpm.CPU.States.AF.Hi)),
				slog.String("B", fmt.Sprintf("%02X", cpm.CPU.States.BC.Hi)),
				slog.String("C", fmt.Sprintf("%02X", cpm.CPU.States.BC.Lo)),
				slog.String("D", fmt.Sprintf("%02X", cpm.CPU.States.DE.Hi)),
				slog.String("E", fmt.Sprintf("%02X", cpm.CPU.States.DE.Lo)),
				slog.String("F", fmt.Sprintf("%02X", cpm.CPU.States.AF.Lo)),
				slog.String("H", fmt.Sprintf("%02X", cpm.CPU.States.HL.Hi)),
				slog.String("L", fmt.Sprintf("%02X", cpm.CPU.States.HL.Lo)),
				slog.String("BC", fmt.Sprintf("%04X", cpm.CPU.States.BC.U16())),
				slog.String("DE", fmt.Sprintf("%04X", cpm.CPU.States.DE.U16())),
				slog.String("HL", fmt.Sprintf("%04X", cpm.CPU.States.HL.U16()))))

		// Invoke the handler
		err = handler.Handler(cpm)

		// Are we being asked to terminate CP/M?  If so return
		if err == ErrExit {
			return nil
		}

		// Are we to reboot?
		if err == ErrBoot {
			cpm.CPU.PC = 0x0000
			continue
		}

		// Any other error is fatal.
		if err != nil {
			return err
		}

		// Return from call by getting the return address
		// from the stack, and updating the instruction pointer
		// to continue executing from there.
		cpm.CPU.PC = cpm.Memory.GetU16(cpm.CPU.SP)

		// We need to remove the entry from the stack to cleanup
		cpm.CPU.SP += 2
	}
}

// RunAutoExec is called once, if we're running in CCP-mode, rather than running
// a simple binary.
//
// If A:SUBMIT.COM and A:AUTOEXEC.SUB exist then we stuff the input-buffer with
// a command to process them.
func (cpm *CPM) RunAutoExec() {

	// These files must be present
	files := []string{"SUBMIT.COM", "AUTOEXEC.SUB"}

	// If one of the files is missing we return
	// without doing anything.
	for _, name := range files {

		// Get the local prefix.
		prefix := cpm.drives[string(cpm.currentDrive+'A')]

		// Add the name
		dst := filepath.Join(prefix, name)

		// Open it to see if it exists.
		handle, err := os.OpenFile(dst, os.O_RDONLY, 0644)
		if err != nil {

			// We're assuming "file not found",
			// or similar, here.
			return
		}

		handle.Close()
	}

	// OK we have both files
	cpm.input.StuffInput("SUBMIT AUTOEXEC")
}

// SetStaticFilesystem allows adding a reference to an embedded filesyste,.
func (cpm *CPM) SetStaticFilesystem(fs embed.FS) {
	cpm.static = fs
}

// SetDrives enables/disables the use of subdirectories upon the host system
// to represent CP/M drives.
//
// We use a map to handle the drive->path mappings, and if directories are
// not used we just store "." in the appropriate entry.
func (cpm *CPM) SetDrives(enabled bool) {

	for _, c := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P"} {
		if enabled {
			cpm.drives[c] = c
		} else {
			cpm.drives[c] = "."
		}
	}
}

// SetDrivePath allows a caller to setup a custom path for a given drive.
func (cpm *CPM) SetDrivePath(drive string, path string) {
	cpm.drives[drive] = path
}

// In is called to handle the I/O reading of a Z80 port.
//
// This is called by our embedded Z80 emulator.
func (cpm *CPM) In(addr uint8) uint8 {
	cpm.Logger.Debug("I/O IN",
		slog.Int("port", int(addr)))

	return 0
}

// Out is called to handle the I/O writing to a Z80 port.
//
// This is called by our embedded Z80 emulator, and this will be
// used by any system which used RST instructions to invoke the
// CP/M syscalls, rather than using "CALL 0x0005".  Notable offenders
// include Microsoft's BASIC.
//
// The functions called here BIOS functions, NOT BDOS functions.
//
// BDOS functions are implemented in our Execute method, via a lookup of
// the C register.  The functions here are invoked with their number in the
// A register and there are far far fewer of them.
func (cpm *CPM) Out(addr uint8, val uint8) {

	// We use port FF for CP/M calls - via
	// the compatibility instructions we deployed
	// in fixRAM.
	if addr != 0xFF {
		return
	}

	//
	// Invoke the handler, in cpm_bios.go
	//
	cpm.BiosHandler(val)
}
