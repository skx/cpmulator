// Package FCB contains helpers for reading, writing, and working with the CP/M FCB structure.
package fcb

import (
	"strings"
)

// FCB is a placeholder struct which is slowly in the process of being used.
type FCB struct {
	// Drive holds the drive letter for this entry.
	Drive uint8

	// Name holds the name of the file.
	Name [8]uint8

	// Type holds the suffix.
	Type [3]uint8

	Ex uint8
	S1 uint8
	S2 uint8
	RC uint8
	Al [16]uint8
	Cr uint8 // FCB_CURRENT_RECORD_OFFSET
	R0 uint8 // FCB_RANDOM_RECORD_OFFSET
	R1 uint8
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
