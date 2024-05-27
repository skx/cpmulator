// Package static is a hierarchy of files that are added to
// the generated emulator.
//
// The intention is that we can ship a number of binary CP/M
// files within our emulator.
package static

import "embed"

//go:embed */*
var Content embed.FS
