package main

import (
	"strings"
)

// FCB is a placeholder struct.
type FCB struct {
	// Drive holds the drive letter for this entry.
	// 0 is A, 1 is B, etc.
	// Max value 15.
	Drive uint8

	// Name holds the name of the file
	Name [8]uint8

	// Type holds the suffix
	Type [3]uint8

	// Don't yet care about the rest of the entry.
	Rest [24]uint8
}

// GetName returns the name component of an FCB entry.
func (f *FCB) GetName() string {
	t := ""

	for _, c := range f.Name {
		if c != 0x00 {
			t += string(c)
		}
	}
	return t
}

// GetType returns the type/extension component of an FCB entry.
func (f *FCB) GetType() string {
	t := ""

	for _, c := range f.Type {
		if c != 0x00 {
			t += string(c)
		}
	}
	return t
}

// AsBytes returns the entry of the FCB in a format suitable
// for copying to RAM
func (f *FCB) AsBytes() []uint8 {

	var r []uint8

	r = append(r, f.Drive)
	r = append(r, f.Name[:]...)
	r = append(r, f.Type[:]...)
	r = append(r, f.Rest[:]...)

	return r
}

// FCBFromString returns an FCB entry from the given string.
//
// This is currently just used for processing command-line arguments.
func FCBFromString(str string) FCB {

	// Return value
	tmp := FCB{}

	// Filenames are always upper-case
	str = strings.ToUpper(str)

	// Does the string have a drive-prefix?
	if len(str) > 2 && str[1] == ':' {
		tmp.Drive = str[0] - 'A'
		str = str[2:]
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
			} else {
				t += string(c)
			}
		}

		// Copy the result into place, noting that copy will truncate
		copy(tmp.Type[:], t)
	}

	return tmp
}
