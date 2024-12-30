// Package ccp contains a pair of embedded CCP binaries, which can
// be used by the emulator as shells.
//
// At build-time we include "*.BIN" from the ccp/ directory, which
// means it's easy to add a new CCP driver - however we must also
// ensure there is a matching name-entry added to the code, so it isn't
// 100% automatic.
package ccp

import (
	"embed"
	"fmt"
	"strings"
)

// Flavour contains details about a possible CCP the user might run.
type Flavour struct {
	// Name contains the public-facing name of the CCP.
	//
	// NOTE: This name is visible to end-users, and will be used in the "-ccp" command-line flag,
	// or as the name when changing at run-time via the "A:!CCP.COM" binary.
	Name string

	// Description contains the description of the CCP.
	Description string

	// Bytes contains the raw binary content.
	Bytes []uint8

	// Start specifies the memory-address, within RAM, to which the raw bytes should be loaded and to which control should be passed.
	//
	// (i.e. This must match the ORG specified in the CCP source code.)
	Start uint16
}

var (
	// ccps contains the global array of the CCP variants we have.
	ccps []Flavour

	//go:embed *.BIN
	ccpFiles embed.FS
)

// init sets up our global ccp array, by adding the two embedded CCPs to
// the array, with suitable names/offsets.
func init() {

	// Load the CCP from DR
	ccp, _ := ccpFiles.ReadFile("DR.BIN")
	ccps = append(ccps, Flavour{
		Name:        "ccp",
		Description: "CP/M v2.2skx",
		Start:       0xDE00,
		Bytes:       ccp,
	})

	// Load the alternative CCP
	ccpz, _ := ccpFiles.ReadFile("CCPZ.BIN")
	ccps = append(ccps, Flavour{
		Name:        "ccpz",
		Description: "CCPZ v4.1skx",
		Start:       0xDE00,
		Bytes:       ccpz,
	})
}

// GetAll returns the details of all known CCPs we have embedded.
func GetAll() []Flavour {
	return ccps
}

// Get returns the CCP version specified, by name, if it exists.
//
// If the given name is invalid then an error will be returned instead.
func Get(name string) (Flavour, error) {

	valid := []string{}

	for _, ent := range ccps {

		// When changing at runtime, via "CCP.COM", we will have had
		// the name upper-cased by the CCP so we need to downcase here.
		if strings.ToLower(name) == ent.Name {
			return ent, nil
		}
		valid = append(valid, ent.Name)
	}

	return Flavour{}, fmt.Errorf("ccp %s not found - valid choices are: %s", name, strings.Join(valid, ","))
}
