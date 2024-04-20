// Package ccp contains an embeded CCP binary, which is used for command-line
// access when no specific binary is executed by cpmulator
package ccp

import (
	_ "embed"
)

//go:embed ccp.bin
var CCPBinary []uint8
