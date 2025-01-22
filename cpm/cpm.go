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
	"strconv"
	"strings"
	"time"

	"github.com/koron-go/z80"
	"github.com/skx/cpmulator/ccp"
	"github.com/skx/cpmulator/consolein"
	"github.com/skx/cpmulator/consoleout"
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

	// ErrBoot will be used to note that the Z80 emulator executed code
	// at 0x0000 - i.e. a boot attempt
	//
	// It should be handled and expected by callers.
	ErrBoot = errors.New("BOOT")

	// ErrUnimplemented will be used to handle a CP/M binary calling an unimplemented syscall.
	//
	// It should be handled and expected by callers.
	ErrUnimplemented = errors.New("UNIMPLEMENTED")

	// DefaultInputDriver contains the name of the default console input driver.
	DefaultInputDriver string = "term"

	// DefaultOutputDriver contains the name of the default console output driver.
	DefaultOutputDriver string = "adm-3a"
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
// While we mostly need a "number to handler", having a name as well
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

	// Noisy is set when a given function is noisy, that is it is called
	// a lot and the debugging logs from the function are not so useful.
	//
	// This is primarily used to mark console I/O functions, and disables
	// their logging by default.  Logging these functions is still possible,
	// but requires a call to LogNoisy()
	Noisy bool
}

// FileCache is used to cache filehandles on the host-side of the system,
// which have been opened by the CP/M binary/CCP.
type FileCache struct {
	// name holds the name of the file, when it was opened/created,
	// on the host-side.
	name string

	// handle has the file handle of the opened file.
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

	// files is the cache we use for File handles.
	files map[uint16]FileCache

	// virtual contains a reference to a static filesystem which
	// is embedded within our binary, if any.
	static embed.FS

	// input is our interface for reading from the console.
	//
	// This needs to take account of echo/no-echo status.
	input *consolein.ConsoleIn

	// output is used for writing characters to the console.
	output *consoleout.ConsoleOut

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

	// biosAddress contains the address of the BIOS we've faked.
	//
	// This might need to be moved, in rare situations.
	biosAddress uint16

	// bdosAddress contains the address of the fake BDOS we've deployed.
	//
	// This might need to be moved, in rare situations.
	bdosAddress uint16

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

	// simpleDebug is used to just output the name of syscalls made.
	//
	// For real debugging we expect the caller to use our Logger, via
	// the logfile
	simpleDebug bool

	// launchTime is the time at which the application was launched
	launchTime time.Time
}

// ccpoption defines a config-setting option for our constructor.
//
// We use the decorator-pattern to allow flexible updates for the
// configuration values we allow.
type cpmoption func(*CPM) error

// WithCCP lets the default CCP to be changed in our constructor.
func WithCCP(name string) cpmoption {
	return func(c *CPM) error {
		c.ccp = name
		return nil
	}
}

// WithPrinterPath allows the printer output to changed in our constructor.
func WithPrinterPath(path string) cpmoption {
	return func(c *CPM) error {
		c.prnPath = path
		return nil
	}
}

// WithOutputDriver allows the default console output driver to be changed in our constructor.
func WithOutputDriver(name string) cpmoption {

	return func(c *CPM) error {

		driver, err := consoleout.New(name)
		if err != nil {
			return err
		}

		c.output = driver
		return nil
	}
}

// WithInputDriver allows the default console input driver to be changed in our constructor.
func WithInputDriver(name string) cpmoption {

	return func(c *CPM) error {

		driver, err := consolein.New(name)
		if err != nil {
			return err
		}

		c.input = driver
		return nil
	}
}

// WithHostExec allows executing commands on the host, by prefixing them with a
// custom prefix in the Readline primitive.
func WithHostExec(prefix string) cpmoption {
	return func(c *CPM) error {
		c.input.SetSystemCommandPrefix(prefix)
		return nil
	}
}

// New returns a new emulation object.  We support default options,
// and new defaults may be specified via WithOutputDriver, etc, etc.
func New(options ...cpmoption) (*CPM, error) {

	//
	// Create and populate our syscall table for the BDOS syscalls.
	//
	bdos := make(map[uint8]CPMHandler)
	bdos[0] = CPMHandler{
		Desc:    "P_TERMCPM",
		Handler: BdosSysCallExit,
	}
	bdos[1] = CPMHandler{
		Desc:    "C_READ",
		Handler: BdosSysCallReadChar,
		Noisy:   true,
	}
	bdos[2] = CPMHandler{
		Desc:    "C_WRITE",
		Handler: BdosSysCallWriteChar,
		Noisy:   true,
	}
	bdos[3] = CPMHandler{
		Desc:    "A_READ",
		Handler: BdosSysCallAuxRead,
		Noisy:   true,
	}
	bdos[4] = CPMHandler{
		Desc:    "A_WRITE",
		Handler: BdosSysCallAuxWrite,
		Noisy:   true,
	}
	bdos[5] = CPMHandler{
		Desc:    "L_WRITE",
		Handler: BdosSysCallPrinterWrite,
		Fake:    true,
		Noisy:   true,
	}
	bdos[6] = CPMHandler{
		Desc:    "C_RAWIO",
		Handler: BdosSysCallRawIO,
		Noisy:   true,
	}
	bdos[7] = CPMHandler{
		Desc:    "GET_IOBYTE",
		Handler: BdosSysCallGetIOByte,
	}
	bdos[8] = CPMHandler{
		Desc:    "SET_IOBYTE",
		Handler: BdosSysCallSetIOByte,
	}
	bdos[9] = CPMHandler{
		Desc:    "C_WRITESTRING",
		Handler: BdosSysCallWriteString,
	}
	bdos[10] = CPMHandler{
		Desc:    "C_READSTRING",
		Handler: BdosSysCallReadString,
	}
	bdos[11] = CPMHandler{
		Desc:    "C_STAT",
		Handler: BdosSysCallConsoleStatus,
		Noisy:   true,
	}
	bdos[12] = CPMHandler{
		Desc:    "S_BDOSVER",
		Handler: BdosSysCallBDOSVersion,
	}
	bdos[13] = CPMHandler{
		Desc:    "DRV_ALLRESET",
		Handler: BdosSysCallDriveAllReset,
	}
	bdos[14] = CPMHandler{
		Desc:    "DRV_SET",
		Handler: BdosSysCallDriveSet,
	}
	bdos[15] = CPMHandler{
		Desc:    "F_OPEN",
		Handler: BdosSysCallFileOpen,
	}
	bdos[16] = CPMHandler{
		Desc:    "F_CLOSE",
		Handler: BdosSysCallFileClose,
	}
	bdos[17] = CPMHandler{
		Desc:    "F_SFIRST",
		Handler: BdosSysCallFindFirst,
	}
	bdos[18] = CPMHandler{
		Desc:    "F_SNEXT",
		Handler: BdosSysCallFindNext,
	}
	bdos[19] = CPMHandler{
		Desc:    "F_DELETE",
		Handler: BdosSysCallDeleteFile,
	}
	bdos[20] = CPMHandler{
		Desc:    "F_READ",
		Handler: BdosSysCallRead,
	}
	bdos[21] = CPMHandler{
		Desc:    "F_WRITE",
		Handler: BdosSysCallWrite,
	}
	bdos[22] = CPMHandler{
		Desc:    "F_MAKE",
		Handler: BdosSysCallMakeFile,
	}
	bdos[23] = CPMHandler{
		Desc:    "F_RENAME",
		Handler: BdosSysCallRenameFile,
	}
	bdos[24] = CPMHandler{
		Desc:    "DRV_LOGINVEC",
		Handler: BdosSysCallLoginVec,
		Fake:    true,
	}
	bdos[25] = CPMHandler{
		Desc:    "DRV_GET",
		Handler: BdosSysCallDriveGet,
	}
	bdos[26] = CPMHandler{
		Desc:    "F_DMAOFF",
		Handler: BdosSysCallSetDMA,
	}
	bdos[27] = CPMHandler{
		Desc:    "DRV_ALLOCVEC",
		Handler: BdosSysCallDriveAlloc,
		Fake:    true,
	}
	bdos[28] = CPMHandler{
		Desc:    "DRV_SETRO",
		Handler: BdosSysCallDriveSetRO,
		Fake:    true,
	}
	bdos[29] = CPMHandler{
		Desc:    "DRV_ROVEC",
		Handler: BdosSysCallDriveROVec,
		Fake:    true,
	}
	bdos[30] = CPMHandler{
		Desc:    "F_ATTRIB",
		Handler: BdosSysCallSetFileAttributes,
		Fake:    true,
	}
	bdos[31] = CPMHandler{
		Desc:    "DRV_DPB",
		Handler: BdosSysCallGetDriveDPB,
		Fake:    true,
	}
	bdos[32] = CPMHandler{
		Desc:    "F_USERNUM",
		Handler: BdosSysCallUserNumber,
	}
	bdos[33] = CPMHandler{
		Desc:    "F_READRAND",
		Handler: BdosSysCallReadRand,
	}
	bdos[34] = CPMHandler{
		Desc:    "F_WRITERAND",
		Handler: BdosSysCallWriteRand,
	}
	bdos[35] = CPMHandler{
		Desc:    "F_SIZE",
		Handler: BdosSysCallFileSize,
	}
	bdos[36] = CPMHandler{
		Desc:    "F_RANDREC",
		Handler: BdosSysCallRandRecord,
	}
	bdos[37] = CPMHandler{
		Desc:    "DRV_RESET",
		Handler: BdosSysCallDriveReset,
		Fake:    true,
	}
	bdos[40] = CPMHandler{
		Desc:    "F_WRITEZF",
		Handler: BdosSysCallWriteRand,

		// We don't zero-pad
		Fake: true,
	}
	bdos[45] = CPMHandler{
		Desc:    "F_ERRMODE",
		Handler: BdosSysCallErrorMode,
		Fake:    true,
	}
	bdos[105] = CPMHandler{
		Desc:    "T_GET",
		Handler: BdosSysCallTime,
		Fake:    true,
	}
	bdos[113] = CPMHandler{ // used by Turbo Pascal
		Desc:    "DirectScreenFunctions",
		Handler: BdosSysCallDirectScreenFunctions,
		Fake:    true,
	}
	bdos[248] = CPMHandler{ // used by BBC BASIC v5
		Desc:    "F_UPTIME",
		Handler: BdosSysCallUptime,
		Fake:    true,
	}

	//
	// Create and populate our syscall table for the BIOS syscalls.
	//
	bios := make(map[uint8]CPMHandler)
	bios[0] = CPMHandler{
		Desc:    "BOOT",
		Handler: BiosSysCallColdBoot,
	}
	bios[1] = CPMHandler{
		Desc:    "WBOOT",
		Handler: BiosSysCallWarmBoot,
	}
	bios[2] = CPMHandler{
		Desc:    "CONST",
		Handler: BiosSysCallConsoleStatus,
		Noisy:   true,
	}
	bios[3] = CPMHandler{
		Desc:    "CONIN",
		Handler: BiosSysCallConsoleInput,
		Noisy:   true,
	}
	bios[4] = CPMHandler{
		Desc:    "CONOUT",
		Handler: BiosSysCallConsoleOutput,
		Noisy:   true,
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
		Noisy:   true,
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

	// Default output driver
	oDriver, err := consoleout.New(DefaultOutputDriver)
	if err != nil {
		return nil, err
	}

	// Default input driver
	iDriver, err := consolein.New(DefaultInputDriver)
	if err != nil {
		return nil, err
	}

	// Helper to return a number, if it is present in the environment
	envNumber := func(name string, defValue uint16) uint16 {

		val := os.Getenv(name)
		if val == "" {
			return defValue
		}

		// base is implied, so "0xFEFE" works.
		num, err := strconv.ParseInt(val, 00, 32)
		if err != nil {
			return defValue
		}

		// Truncate.
		return uint16(num & 0xFFFF)
	}

	// Create the emulator object and return it
	tmp := &CPM{
		BDOSSyscalls: bdos,
		BIOSSyscalls: bios,
		ccp:          "ccp", // default
		dma:          0x0080,
		drives:       make(map[string]string),
		files:        make(map[uint16]FileCache),
		input:        iDriver,       // default
		output:       oDriver,       // default
		prnPath:      "printer.log", // default
		start:        0x0100,
		launchTime:   time.Now(),
		biosAddress:  envNumber("BIOS_ADDRESS", 0xFE00),
		bdosAddress:  envNumber("BDOS_ADDRESS", 0xF000),
	}

	// Allow options to override our defaults
	for _, option := range options {
		err := option(tmp)
		if err != nil {
			return tmp, err
		}
	}

	return tmp, nil
}

// IOSetup ensures that our I/O is ready.
func (cpm *CPM) IOSetup() {
	cpm.input.Setup()
}

// IOTearDown cleans up the state of the terminal, if necessary.
func (cpm *CPM) IOTearDown() {
	cpm.input.TearDown()
}

// GetInputDriver returns the configured input driver.
func (cpm *CPM) GetInputDriver() consolein.ConsoleInput {
	return cpm.input.GetDriver()
}

// GetOutputDriver returns the configured output driver.
func (cpm *CPM) GetOutputDriver() consoleout.ConsoleOutput {
	return cpm.output.GetDriver()
}

// GetCCPName returns the name of the CCP we've been configured to load.
func (cpm *CPM) GetCCPName() string {
	return cpm.ccp
}

// GetBIOSAddress returns the address the fake BIOS is deployed at.
//
// This is used for the startup banner.
func (cpm *CPM) GetBIOSAddress() uint16 {
	return cpm.biosAddress
}

// GetBDOSAddress returns the address the fake BDOS is deployed at.
//
// This is used for the startup banner.
func (cpm *CPM) GetBDOSAddress() uint16 {
	return cpm.bdosAddress
}

// LogNoisy enables logging support for each of the functions which
// would otherwise be disabled
func (cpm *CPM) LogNoisy() {

	for k, e := range cpm.BDOSSyscalls {
		e.Noisy = false
		cpm.BDOSSyscalls[k] = e
	}
	for k, e := range cpm.BIOSSyscalls {
		e.Noisy = false
		cpm.BIOSSyscalls[k] = e
	}
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
// handle "badly behaved" programs that invoke CP/M functions by working out
// the address of the BIOS/BDOS and jumping there directly.
//
// We put some code to call the handlers via faked OUT 0xFF,N - where N
// is the syscall to run.
//
// The precise region we patch is unimportant, but we want to make sure
// we don't overlap with our CCP, or "large programs" loaded at 0x0100.
//
// We store the addresses we can use in the cpm-structure, and they can
// be changed by the user via the BIOS_ADDRESS and BDOS_ADDRESS environmental
// variables.
func (cpm *CPM) fixupRAM() {
	i := 0

	// The two addresses which are important
	BIOS := int(cpm.biosAddress)
	BDOS := int(cpm.bdosAddress)

	NENTRY := 30

	SETMEM := func(a int, v int) {
		cpm.Memory.Set(uint16(a), uint8(v))
	}

	SETMEM(0x0000, 0xC3)                /* JMP */
	SETMEM(0x0001, ((BIOS + 3) & 0xFF)) /* Fake address of entry-point */
	SETMEM(0x0002, ((BIOS + 3) >> 8))

	SETMEM(BIOS, 0xC9) // RET

	// Now we setup the initial values of the I/O byte
	SETMEM(0x0003, 0x00)

	// We setup a fake jump here, because 0x0006 is sometimes
	// used to find the free RAM and we pretend our BDOS is at 0xDC00
	SETMEM(0x0005, 0xC3)            /* JMP */
	SETMEM(0x0006, ((BDOS) & 0xFF)) /* Fake Address of entry point */
	SETMEM(0x0007, ((BDOS) >> 8))

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
		SETMEM(BIOS+3*i, 0xC3)
		SETMEM(BIOS+3*i+1, (BIOS+NENTRY*3+i*5)&0xFF)
		SETMEM(BIOS+3*i+2, (BIOS+NENTRY*3+i*5)>>8)

		/* LD A,<bios-call> - start of bios-entry */
		SETMEM(BIOS+NENTRY*3+i*5+0, 0x3E)
		SETMEM(BIOS+NENTRY*3+i*5+1, i)

		/* OUT A,0FFH - we use port 0xFF to fake the BIOS call */
		SETMEM(BIOS+NENTRY*3+i*5+2, 0xD3)
		SETMEM(BIOS+NENTRY*3+i*5+3, 0xFF)

		/* RET - end of bios-entry */
		SETMEM(BIOS+NENTRY*3+i*5+4, 0xC9)
		i++
	}

	SETMEM(BDOS+0, 0x00) // NOP
	SETMEM(BDOS+1, 0x00) // NOP
	SETMEM(BDOS+2, 0x00) // NOP
	SETMEM(BDOS+3, 0x00) // NOP
	SETMEM(BDOS+4, 0x00) // NOP
	SETMEM(BDOS+5, 0x00) // NOP
	SETMEM(BDOS+6, 0xED) // OUT (C), C
	SETMEM(BDOS+7, 0x49) //    ""
	SETMEM(BDOS+8, 0xC9) // RET
	SETMEM(BDOS+9, 0x00) // NOP

}

// LoadCCP loads the CCP into RAM, to be executed instead of an external binary.
//
// This function modifies the "start" attribute, to ensure the CCP is loaded
// and executed at a higher address than the default of 0x0100.
func (cpm *CPM) LoadCCP() error {

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
		slog.Debug("Closing handle in FileCache",
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
				SP: 0xFFFF,
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

	for {
		// Reset the state of any saved error and the halt-flag.
		cpm.biosErr = nil
		cpm.CPU.HALT = false

		// Launch the Z80 emulator.
		//
		// This will basically run forever, or until the CPU is halted
		// in one of our handlers.
		err := cpm.CPU.Run(context.Background())

		// If the errors are both empty, but the CPU is halted then we exit.
		if err == nil && cpm.biosErr == nil && cpm.CPU.HALT {
			return ErrHalt
		}

		// An unimplemented system-call was encountered.
		// That's a fatal-error
		if err == ErrUnimplemented || cpm.biosErr == ErrUnimplemented {
			return ErrUnimplemented
		}

		// The virtual machine was rebooted.
		if err == ErrBoot || cpm.biosErr == ErrBoot {
			return ErrBoot
		}

		// Emulation was terminated.
		if err == ErrHalt || cpm.biosErr == ErrHalt {
			return ErrHalt
		}

		// The emulator exits.
		// NOTE: We send a different return code here.
		if err == ErrExit || cpm.biosErr == ErrExit {
			return ErrBoot
		}
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
	cpm.input.StuffInput("SUBMIT AUTOEXEC\n")
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
	slog.Debug("I/O IN",
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

	if cpm.CPU.HALT {
		fmt.Printf("Out() called when CPU is halted\r\n")
		return
	}
	if cpm.biosErr != nil {
		fmt.Printf("Out() with pending error: %v\r\n", cpm.biosErr)
		return
	}

	//
	// Reset our state
	//
	cpm.CPU.HALT = false
	cpm.biosErr = nil

	//
	// We're going to lookup a handler, based on the
	// register that makes sense.
	//
	var handler CPMHandler

	// Did we find it?  And if so what type was it?
	var ok bool
	var callType string

	//
	// We use port FF for BIOS calls.
	//
	// Any other port is a BDOS call.
	//
	if addr == 0xFF {

		// If the port and the value don't match then something
		// fishy is going on.
		//
		// Or a CP/M binary is legitimately using an OUT instruction.
		//
		// To make sense we log this as a fatal error, and bail.
		if val != cpm.CPU.AF.Hi {
			slog.Error("Out() mismatch for "+callType+" handler",
				slog.Int("Value", int(val)),
				slog.Group("registers",
					slog.String("AF", fmt.Sprintf("%04X", cpm.CPU.States.AF.U16())),
				))
			return
		}

		callType = "BIOS"

		// Lookup the handler
		handler, ok = cpm.BIOSSyscalls[val]

	} else {

		// If the port and the value don't match then something
		// fishy is going on.
		//
		// Or a CP/M binary is legitimately using an OUT instruction.
		//
		// To make sense we log this as a fatal error, and bail.
		if val != cpm.CPU.BC.Lo {
			slog.Error("Out() mismatch for "+callType+" handler",
				slog.Int("Value", int(val)),
				slog.Group("registers",
					slog.String("BC", fmt.Sprintf("%04X", cpm.CPU.States.BC.U16())),
				))
			return
		}

		callType = "BDOS"

		//
		// Is there a syscall entry for this number?
		//
		handler, ok = cpm.BDOSSyscalls[cpm.CPU.BC.Lo]

	}

	// If a handler was not found then we've not implemented
	// the given syscall.
	//
	// Log the failure and return.
	if !ok {
		slog.Error("Unimplemented "+callType+" syscall",
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)))

		cpm.biosErr = ErrUnimplemented
		cpm.CPU.HALT = true
		return
	}

	// Log the call, if we should.
	if !handler.Noisy {

		// show the function being invoked.
		if cpm.simpleDebug {
			fmt.Printf("%03d %s %s\n", val, callType, handler.Desc)
		}

		// Log the call we're going to make
		slog.Info(callType,
			slog.String("name", handler.Desc),
			slog.String("type", callType),
			slog.Int("syscall", int(val)),
			slog.String("syscallHex", fmt.Sprintf("0x%02X", val)),
			slog.Group("registers",
				slog.String("AF", fmt.Sprintf("%04X", cpm.CPU.States.AF.U16())),
				slog.String("BC", fmt.Sprintf("%04X", cpm.CPU.States.BC.U16())),
				slog.String("DE", fmt.Sprintf("%04X", cpm.CPU.States.DE.U16())),
				slog.String("HL", fmt.Sprintf("%04X", cpm.CPU.States.HL.U16()))))
	}

	// Invoke the handler, and save any error that we receive.
	//
	// If an error was returned we halt the emulation of the virtual Z80.
	cpm.biosErr = handler.Handler(cpm)
	if cpm.biosErr != nil {
		cpm.CPU.HALT = true
	}

	// If A == 0x00 then we set the zero flag.
	//
	// Not even sure if this is necessary..
	if cpm.CPU.States.AF.Hi == 0x00 {
		cpm.CPU.SetFlag(z80.FlagZ)
	} else {
		cpm.CPU.ResetFlag(z80.FlagZ)
	}
}

// StuffText inserts text into the read-buffer of the console
// input-driver.
//
// This is used for two purposes; to drive the "SUBMIT AUTOEXEC"
// integration at run-time, and to support integration tests to be
// written.
func (cpm *CPM) StuffText(input string) {
	cpm.input.StuffInput(input)
}
