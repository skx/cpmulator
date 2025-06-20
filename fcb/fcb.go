// Package fcb contains helpers for reading, writing, and working with the CP/M FCB structure.
package fcb

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"unicode"
)

// SIZE contains the size of the FCB structure
var SIZE = 36

// FCB is a structure which is used to hold details about file entries, although
// later versions of CP/M support directories we do not.
//
// We largely focus upon Name, Type, and the various read/write offsets.  Most of
// the other fields are maintained but ignored.
type FCB struct {
	// Drive holds the drive letter for this entry.
	Drive uint8

	// Name holds the name of the file.
	Name [8]uint8

	// Type holds the suffix.
	Type [3]uint8

	// Ex holds the logical extent.
	Ex uint8

	// S1 is reserved, and ignored.
	S1 uint8

	// S2 is reserved, and ignored.
	S2 uint8

	// RC holds the record count.
	// (i.e. The size of the file in 128-byte records.)
	RC uint8

	// Allocation map, ignored.
	Al [16]uint8

	// Cr holds the current record offset.
	Cr uint8

	// R0, holds part of the random-record offset.
	R0 uint8

	// R1 holds part of the random-record offset.
	R1 uint8

	// R2 holds part of the random-record offset.
	R2 uint8
}

// Find is the structure which is returned for files found via FindFirst / FindNext.
//
// This structure exists to make it easy for us to work with both the path on the host,
// and the path within the CP/M disk.  Specifically we need to populate the size of
// files when we return their FCB entries from either call - and that means we need
// access to the host filesystem (i.e. cope when directories are used to represent
// drives).
type Find struct {
	// Host is the location on the host for the file.
	// This might refer to the current directory, or a drive-based sub-directory.
	Host string

	// Name is the name as CP/M would see it.
	// This will be upper-cased and in 8.3 format.
	Name string

	// Size contains the size of the file.
	Size int64
}

// GetName returns the name component of an FCB entry.
func (f *FCB) GetName() string {
	t := ""

	for _, c := range f.Name {
		if c != 0x00 {
			t += string(c)
		}
	}
	return strings.TrimSpace(t)
}

// GetType returns the type/extension component of an FCB entry.
//
// If the extension is null, or empty, we return the empty string.
func (f *FCB) GetType() string {
	t := ""

	for _, c := range f.Type {
		if unicode.IsPrint(rune(c)) {
			t += string(c)
		} else {
			t += " "
		}
	}
	return t
}

// GetFileName returns the name and suffix, but importantly it removes
// any trailing spaces.
func (f *FCB) GetFileName() string {
	name := f.GetName()
	ext := f.GetType()

	if ext != "" && ext != "   " {
		name += "."
		name += ext
	}

	return strings.TrimSpace(name)
}

// GetCacheKey returns a string which can be used for caching this
// object in some way - it's the name of the file, as seen by the
// CP/M system.
func (f *FCB) GetCacheKey() string {
	t := ""

	// Name
	for _, c := range f.Name {
		if unicode.IsPrint(rune(c)) {
			t += string(c)
		} else {
			t += " "
		}
	}

	// Suffix
	for _, c := range f.Type {
		if unicode.IsPrint(rune(c)) {
			t += string(c)
		} else {
			t += " "
		}
	}
	return t

}

// AsBytes returns the entry of the FCB in a format suitable
// for copying to RAM.
func (f *FCB) AsBytes() []uint8 {

	var r []uint8

	r = append(r, f.Drive)
	r = append(r, f.Name[:]...)
	r = append(r, f.Type[:]...)
	r = append(r, f.Ex)
	r = append(r, f.S1)
	r = append(r, f.S2)
	r = append(r, f.RC)
	r = append(r, f.Al[:]...)
	r = append(r, f.Cr)
	r = append(r, f.R0)
	r = append(r, f.R1)
	r = append(r, f.R2)

	return r
}

// SetSequentialOffset updates the offset used for sequential reads/writes
// to use the given value.
func (f *FCB) SetSequentialOffset(offset int64) {

	f.Cr = uint8((offset % 16384) / 128)
	f.Ex = uint8((offset % 524288) / 16384)
	f.S2 = uint8(offset / 524288)

	// confirm this works
	x := f.GetSequentialOffset()
	if x != offset {
		slog.Error("updating the sequential offset failed",
			slog.Int64("expected", offset),
			slog.Int64("real", x))
	}
}

// GetRandomOffset returns the value held in the random fields, R0, R1, and R2.
func (f *FCB) GetRandomOffset() int64 {
	return int64(int64(f.R2)<<16) | int64(int64(f.R1)<<8) | int64(f.R0)
}

// SetRandomOffset updates the value held in the random fields, R0, R1, and R2.
func (f *FCB) SetRandomOffset(offset int64) {

	// Now we set the "random record" which is R0,R1,R2
	f.R0 = uint8(offset & 0xFF)
	f.R1 = uint8(offset >> 8)
	f.R2 = uint8(offset >> 16)
}

// GetSequentialOffset returns the offset the FCB contains for
// the sequential read/write calls - as used by the BDOS functions
// F_READ and F_WRITE.
func (f *FCB) GetSequentialOffset() int64 {

	offset := (int64(f.S2) * 524288) +
		(int64(f.Ex) * 16384) +
		(int64(f.Cr) * 128)
	return offset
}

// FromString returns an FCB entry from the given string.
//
// This is currently just used for processing command-line arguments.
func FromString(str string) FCB {

	// Return value
	tmp := FCB{}

	// Filenames are always upper-case
	str = strings.ToUpper(str)

	// Does the string have a drive-prefix?
	if len(str) > 2 && str[1] == ':' {
		tmp.Drive = str[0] - 'A'
		str = str[2:]
	} else {
		tmp.Drive = 0x00
	}

	// Suffix defaults to "   "
	copy(tmp.Type[:], "   ")

	// Now we have to parse the string.
	//
	// 1. is there a suffix?
	parts := strings.Split(str, ".")

	// No suffix?
	if len(parts) == 1 {
		t := ""

		// pad the value
		name := parts[0]
		for len(name) < 8 {
			name += " "
		}

		// process to change "*" to "????"
		for _, c := range name {
			if c == '*' {
				t += "?????????"
				break
			} else {
				t += string(c)
			}
		}

		// Copy the result into place, noting that copy will truncate
		copy(tmp.Name[:], t)
	}
	if len(parts) == 2 {
		t := ""

		// pad the value
		name := parts[0]
		for len(name) < 8 {
			name += " "
		}

		// process to change "*" to "????"
		for _, c := range name {
			if c == '*' {
				t += "?????????"
				break
			} else {
				t += string(c)
			}
		}

		// Copy the result into place, noting that copy will truncate
		copy(tmp.Name[:], t)

		// pad the value
		ext := parts[1]
		for len(ext) < 3 {
			ext += " "
		}

		// process to change "*" to "????"
		t = ""
		for _, c := range ext {
			if c == '*' {
				t += "???"
				break
			} else {
				t += string(c)
			}
		}

		// Copy the result into place, noting that copy will truncate
		copy(tmp.Type[:], t)
	}

	return tmp
}

// FromBytes returns an FCB entry from the given bytes
func FromBytes(bytes []uint8) FCB {
	// Return value
	tmp := FCB{}

	tmp.Drive = bytes[0]
	copy(tmp.Name[:], bytes[1:])
	copy(tmp.Type[:], bytes[9:])
	tmp.Ex = bytes[12]
	tmp.S1 = bytes[13]
	tmp.S2 = bytes[14]
	tmp.RC = bytes[15]
	copy(tmp.Al[:], bytes[16:])
	tmp.Cr = bytes[32]
	tmp.R0 = bytes[33]
	tmp.R1 = bytes[34]
	tmp.R2 = bytes[35]

	return tmp
}

// DoesMatch returns true if the filename specified matches the pattern in the FCB.
func (f *FCB) DoesMatch(name string) bool {

	// If the file doesn't have a dot then it can't be visible if it is too long
	if len(name) > 8 && !strings.Contains(name, ".") {
		return false
	}

	// Having a .extension is fine, but if the
	// suffix is longer than three characters we're
	// not going to use it.
	parts := strings.Split(name, ".")
	if len(parts) == 2 {
		// filename is over 8 characters
		if len(parts[0]) > 8 {
			return false
		}
		// suffix is over 3 characters
		if len(parts[1]) > 3 {
			return false
		}
	}

	// Create a temporary FCB for the specified filename.
	tmp := FromString(name)

	// Now test if the name we've got matches that in the
	// search-pattern: Name.
	//
	// Either a literal match, or a wildcard match with "?".
	for i, c := range f.Name {
		if (tmp.Name[i] != c) && (f.Name[i] != '?') {
			return false
		}
	}

	// Repeat for the suffix.
	for i, c := range f.Type {
		if (tmp.Type[i] != c) && (f.Type[i] != '?') {
			return false
		}
	}

	// Got a match
	return true
}

// GetMatches returns the files matching the pattern in the given FCB record.
//
// We try to do this by converting the entries of the named directory into FCBs
// after ignoring those with impossible formats - i.e. not FILENAME.EXT length.
func (f *FCB) GetMatches(prefix string) ([]Find, error) {
	var ret []Find

	err := filepath.Walk(prefix, func(path string, info fs.FileInfo, err error) error {

		if err != nil {
			return err
		}

		// Ignore directories, we only care about files.
		if info.IsDir() {
			return nil
		}

		// Upper-case, and remove prefix.
		name := filepath.Base(strings.ToUpper(path))

		if f.DoesMatch(name) {

			var ent Find

			// Populate the host-path before we do anything else.
			ent.Host = filepath.Join(path)

			// populate the name.
			ent.Name = name

			// populate the size.
			ent.Size = info.Size()

			// append
			ret = append(ret, ent)
		}

		return nil
	})

	// Return the entries we found, if any.
	return ret, err
}
