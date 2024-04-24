// Package fcb contains helpers for reading, writing, and working with the CP/M FCB structure.
package fcb

import (
	"os"
	"strings"
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
		if c != 0x00 {
			t += string(c)
		}
	}
	return strings.TrimSpace(t)
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

// GetSequentialOffset returns the offset the FCB contains for
// the sequential read/write calls - as used by the BDOS functions
// F_READ and F_WRITE.
//
// IncreaseSequentialOffset updates the value.
func (f *FCB) GetSequentialOffset() int64 {

	// Helpers
	BlkS2 := 4096
	BlkEx := 128
	MaxS2 := 15
	blkSize := 128

	offset := int64((int(f.S2)&MaxS2)*BlkS2*blkSize +
		int(f.Ex)*BlkEx*blkSize +
		int(f.Cr)*blkSize)
	return offset
}

// IncreaseSequentialOffset updates the read/write offset which
// would be used for the sequential read functions.
func (f *FCB) IncreaseSequentialOffset() {

	MaxCR := 128
	MaxEX := 31

	f.S2 &= 0x7F // reset unmodified flag
	f.Cr++
	if int(f.Cr) > MaxCR {
		f.Cr = 1
		f.Ex++
	}
	if int(f.Ex) > MaxEX {
		f.Ex = 0
		f.S2++
	}
	f.RC++

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

// GetMatches returns the files matching the pattern in the given FCB record.
//
// We try to do this by converting the entries of the named directory into FCBs
// after ignoring those with impossible formats - i.e. not FILENAME.EXT length.
func (f *FCB) GetMatches(prefix string) ([]string, error) {
	var ret []string

	t := string(f.Type[0]) + string(f.Type[1]) + string(f.Type[2])
	if t == "" || t == "   " {
		t = "???"
	}

	// Find files in the directory
	files, err := os.ReadDir(prefix)
	if err != nil {
		return ret, err
	}

	// For each file
	for _, file := range files {

		orig := file.Name()

		// Ignore directories
		if file.IsDir() {
			continue
		}

		// Name needs to be upper-cased
		name := strings.ToUpper(file.Name())

		// is the name too long?
		if len(name) > 8+3 {
			continue
		}

		// Having a .extension is fine, but if the
		// suffix is longer than three characters we're
		// not going to use it.
		parts := strings.Split(name, ".")
		if len(parts) == 2 {
			// filename is over 8 characters
			if len(parts[0]) > 8 {
				continue
			}
			// suffix is over 3 characters
			if len(parts[1]) > 3 {
				continue
			}
		}

		include := true
		// OK make an fcb
		tmp := FromString(name)
		for i, c := range tmp.Name {
			if (f.Name[i] != c) && (f.Name[i] != '?') {
				include = false
			}
		}
		for i, c := range tmp.Type {
			if (t[i] != c) && (t[i] != '?') {
				include = false
			}
		}
		// Does it match? Then add the original name
		if include {
			ret = append(ret, orig)
		}
	}
	// Find files in the current directory.
	return ret, nil
}
