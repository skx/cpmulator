// Package ccp contains a pair of embedded CCP binaries, which can
// be used by the emulator as shells.
package ccp

import (
	_ "embed"
	"fmt"
	"strings"
)

// ccps contains the global array of the CCP variants we have.
var ccps []Flavour

// Flavour contains details about a possible CCP the user might run.
type Flavour struct {
	// Name has the name of the CCP.
	Name string

	// Description has the description of the CCP.
	Description string

	// Bytes contains the raw binary content.
	Bytes []uint8

	// Origin contains the start/load location of the CCP.
	Start uint16
}

//go:embed DR.BIN
var ccpBin []uint8

//go:embed CCPZ.BIN
var ccpzBin []uint8

// init sets up our global ccp array, by adding the two embedded CCPs to
// the array, with suitable names/offsets.
func init() {
	ccps = append(ccps, Flavour{
		Name:        "ccp",
		Description: "CP/M v2.2",
		Start:       0xDE00,
		Bytes:       ccpBin,
	})

	ccps = append(ccps, Flavour{
		Name:        "ccpz",
		Description: "CCPZ v4.1skx",
		Start:       0xDE00,
		Bytes:       ccpzBin,
	})
}

// GetAll returns the details of all known CCPs we have embedded.
func GetAll() []Flavour {
	return ccps
}

// Get returns the CCP version specified, by name, if it exists.
//
// If the given name is invalid then an error will be returned.
func Get(name string) (Flavour, error) {

	valid := []string{}

	for _, ent := range ccps {

		if strings.ToLower(name) == ent.Name {
			return ent, nil
		}
		valid = append(valid, ent.Name)
	}

	return Flavour{}, fmt.Errorf("ccp %s not found - valid choices are: %s", name, strings.Join(valid, ","))
}
