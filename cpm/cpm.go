// Package cpm is the main package for our emulator, it uses memory to
// emulate execution of things at the bios level.
//
// The package mostly contains the implementation of the syscalls that
// CP/M programs would expect - along with a little machinery to wire up
// the Z80 emulator we're using and deal with FCB structures.
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
	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/fcb"
	"github.com/skx/cpmulator/memory"
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

	// ErrUnimplemented will be used to handle a CP/M binary calling an unimplemented syscall.
	//
	// It should be handled and expected by callers.
	ErrUnimplemented = errors.New("UNIMPLEMENTED")
)

// CPMHandlerType contains the signature of a CP/M bios function.
//
// It is not expected that outside packages will want to add custom BIOS
// functions, or syscalls, but this is public so that it could be done if
// it was necessary.
type CPMHandlerType func(cpm *CPM) error

// CPMHandler contains details of a specific call we implement.
//
// While we mostly need a "number to handler", mapping having a name
// is useful for the logs we produce.
type CPMHandler struct {
	// Desc contain the human-readable description of the given CP/M syscall.
	Desc string

	// Handler contains the function which should be involved for this syscall.
	Handler CPMHandlerType
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

// CPM is the object that holds our emulator state
type CPM struct {

	// files is the cache we use for File handles
	files map[uint16]FileCache

	// dma contains the offset of the DMA area which is used
	// for block I/O.
	dma uint16

	// start contains the location to which we load our binaries,
	// and execute them from.  This is specifically a variable because
	// while all CP/M binaries are loaded at 0x0100 the CCP we can
	// launch uses a higher location - so that it isn't overwritten by
	// the programs it launches.
	start uint16

	// auxStatus handles storing the state for the auxilary / punch output
	// device.  This is used by MBASIC amongst other things, and we use it
	// to basically keep track of multibyte output
	auxStatus int

	// x holds the character X position, when using AUX I/O.
	// It is set/used by escape sequences.
	x uint8

	// y holds the character Y position, when using AUX I/O.
	// It is set/used by escape sequences.
	y uint8

	// Syscalls contains the syscalls we know how to emulate, indexed
	// by their ID.
	Syscalls map[uint8]CPMHandler

	// Memory contains the memory the system runs with.
	Memory *memory.Memory

	// CPU contains a pointer to the virtual CPU we use to execute
	// code.  The CP/M we're implementing is Z80-based, so we need to
	// be able to emulate that.
	CPU z80.CPU

	// Drives specifies whether we use sub-directories for the
	// CP/M drives we emulate, instead of the current working directory.
	Drives bool

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

	// findFirstResults is a sneaky cache of files that match a glob.
	//
	// For finding files CP/M uses "find first" to find the first result
	// then allows the programmer to call "find next", to continue the searching.
	//
	// This means we need to track state, the way we do this is to store the
	// results here, and bump the findOffset each time find-next is called.
	findFirstResults []string
	findOffset       int

	// Reader is where we get our STDIN from.
	//
	// TODO: We should have something similar for STDOUT.
	Reader *bufio.Reader

	// Logger holds a logger which we use for debugging and diagnostics.
	Logger *slog.Logger
}

// New returns a new emulation object
func New(logger *slog.Logger) *CPM {

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
	sys[3] = CPMHandler{
		Desc:    "A_READ",
		Handler: SysCallAuxRead,
	}
	sys[4] = CPMHandler{
		Desc:    "A_WRITE",
		Handler: SysCallAuxWrite,
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
	}
	sys[25] = CPMHandler{
		Desc:    "DRV_GET",
		Handler: SysCallDriveGet,
	}
	sys[26] = CPMHandler{
		Desc:    "F_DMAOFF",
		Handler: SysCallSetDMA,
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
	sys[34] = CPMHandler{
		Desc:    "F_WRITERAND",
		Handler: SysCallWriteRand,
	}

	// Create the object
	tmp := &CPM{
		Logger:   logger,
		Reader:   bufio.NewReader(os.Stdin),
		Syscalls: sys,
		dma:      0x0080,
		start:    0x0100,
		files:    make(map[uint16]FileCache),
	}
	return tmp
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
// we don't overlap with our CCP, or "large programs" loaded at 0x0100
func (cpm *CPM) fixupRAM() {
	i := 0
	CBIOS := 0xFE00
	NENTRY := 30

	SETMEM := func(a int, v int) {
		cpm.Memory.Set(uint16(a), uint8(v))
	}

	// We _should_ add "JUMP TO CCP" here, but instead
	// we terminate execution via a HALT (0x76) instruction.
	//
	// This works regardless of whether CCP is present or not
	//
	SETMEM(0x0000, 0x76) // 0xC3) /* JP CBIOS+3 */
	SETMEM(0x0001, ((CBIOS + 3) & 0xFF))
	SETMEM(0x0002, ((CBIOS + 3) >> 8))

	SETMEM(0x0003, 0x00) // IO/byte
	SETMEM(0x0004, 0x00) // Current drive

	/* fake BIOS entry points */
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
func (cpm *CPM) LoadCCP() {

	// Create 64K of memory, full of NOPs
	if cpm.Memory == nil {
		cpm.Memory = new(memory.Memory)
	}

	// Get our embedded CCP
	data := ccp.CCPBinary

	// The location in RAM of the binary
	var ccpEntrypoint uint16 = 0xDE00

	// Load it into memory
	cpm.Memory.SetRange(ccpEntrypoint, data...)

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
	cpm.start = ccpEntrypoint

	// patch low-memory so that RST instructions will
	// ultimately invoke our CP/M syscalls, via our "Out"
	// function.
	cpm.fixupRAM()
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
	cpm.CPU.States.BC.Lo = cpm.currentDrive

	// Setup a breakpoint on 0x0005 - the BIOS entrypoint.
	cpm.CPU.BreakPoints = map[uint16]struct{}{}
	cpm.CPU.BreakPoints[0x05] = struct{}{}

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

		// No error?  Then end - the CPU hit a HALT.
		if err == nil {
			return ErrHalt
		}

		// An error which wasn't a breakpoint?  Give up
		if err != z80.ErrBreakPoint {
			return fmt.Errorf("unexpected error running CPU %s", err)
		}

		// OK we have a breakpoint error to handle.
		//
		// That means we have a CP/M BIOS function to emulate,
		// the syscall identifier is stored in the C-register.
		syscall := cpm.CPU.States.BC.Lo

		//
		// Is there a syscall entry for this number?
		//
		handler, exists := cpm.Syscalls[syscall]

		//
		// Nope: That will stop execution with a fatal log
		//
		if !exists {

			cpm.Logger.Error("Unimplemented SysCall",
				slog.Int("syscall", int(syscall)),
				slog.String("syscallHex",
					fmt.Sprintf("0x%02X", syscall)),
			)
			return ErrUnimplemented
		}

		// Log the call we're going to make
		cpm.Logger.Info("SysCall",
			slog.String("name", handler.Desc),
			slog.Int("syscall", int(syscall)),
			slog.String("syscallHex",
				fmt.Sprintf("0x%02X", syscall)),
		)

		// Invoke the handler
		err = handler.Handler(cpm)

		// Are we being asked to terminate CP/M?  If so return
		if err == ErrExit {
			return nil
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

// SetDrives enables/disables the use of subdirectories upon the host system
// to represent CP/M drives
func (cpm *CPM) SetDrives(enabled bool) {
	cpm.Drives = true
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
func (cpm *CPM) Out(addr uint8, val uint8) {

	// We use port FF for CP/M calls - via
	// the compatibility instructions we deployed
	// in fixRAM.
	if addr != 0xFF {
		return
	}

	//
	// Is there a syscall entry for this number?
	//
	handler, exists := cpm.Syscalls[val]

	if !exists {
		cpm.Logger.Error("Unimplemented SysCall - Via I/O",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex",
				fmt.Sprintf("0x%02X", val)),
		)
		return
	}

	// Log the call we're going to make
	cpm.Logger.Info("SysCall via I/O",
		slog.String("name", handler.Desc),
		slog.Int("syscall", int(val)),
		slog.String("syscallHex",
			fmt.Sprintf("0x%02X", val)),
	)

	// Invoke the handler
	err := handler.Handler(cpm)
	if err != nil {
		fmt.Printf("ERROR Via I/O Handler: %s\n", err)
	}

}
