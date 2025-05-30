// Package static is a hierarchy of files that are added to
// the generated emulator.
//
// The intention is that we can ship a number of binary CP/M
// files within our emulator.
package static

import "embed"

//go:embed */*
var content embed.FS

// empty has no contents.
var empty embed.FS

// GetContent returns the embedded filesystem we store within this package.
func GetContent() embed.FS {
	return content
}

// GetEmptyContent returns the embedded filesystem we store within this package which has no contents.
func GetEmptyContent() embed.FS {
	return empty
}
