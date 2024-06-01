package fcb

import (
	"fmt"
	"testing"
)

// TestFCBSize ensures our size matches expectations.
func TestFCBSize(t *testing.T) {
	x := FromString("blah")
	b := x.AsBytes()

	if len(b) != 36 {
		t.Fatalf("FCB struct is %d bytes", len(b))
	}
}

// Test we can convert an FCB to bytes, and back, without losing data in the round-trip.
func TestCopy(t *testing.T) {
	f1 := FromString("blah")
	copy(f1.Al[:], "0123456789abcdef")
	f1.Ex = 'X'
	f1.S1 = 'S'
	f1.S2 = '?'
	f1.RC = 'f'
	f1.R0 = 'R'
	f1.R1 = '0'
	f1.R2 = '1'
	f1.Cr = '*'
	b := f1.AsBytes()

	f2 := FromBytes(b)
	if fmt.Sprintf("%s", f2.Al) != "0123456789abcdef" {
		t.Fatalf("copy failed")
	}
	if f2.Ex != 'X' {
		t.Fatalf("copy failed")
	}
	if f2.S1 != 'S' {
		t.Fatalf("copy failed")
	}
	if f2.Cr != '*' {
		t.Fatalf("copy failed")
	}
	if f2.S2 != '?' {
		t.Fatalf("copy failed")
	}
	if f2.RC != 'f' {
		t.Fatalf("copy failed")
	}
	if f2.R0 != 'R' {
		t.Fatalf("copy failed")
	}
	if f2.R1 != '0' {
		t.Fatalf("copy failed")
	}
	if f2.R2 != '1' {
		t.Fatalf("copy failed")
	}

}

// TestFCBFromString is a trivial test to only cover the basics right now.
func TestFCBFromString(t *testing.T) {

	// Simple test to ensure the basic one works.
	f := FromString("b:foo")
	if f.Drive != 1 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "FOO" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}
	if f.GetType() != "   " {
		t.Fatalf("unexpected suffix '%v'", f.GetType())
	}

	// Try a long name, to confirm it is truncated
	f = FromString("c:this-is-a-long-name")
	if f.Drive != 2 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "THIS-IS-" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}
	if f.GetType() != "   " {
		t.Fatalf("unexpected suffix '%v'", f.GetType())
	}

	// Try a long suffix, to confirm it is truncated
	f = FromString("c:this-is-a-.long-name")
	if f.Drive != 2 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "THIS-IS-" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}
	if f.GetType() != "LON" {
		t.Fatalf("unexpected suffix '%v'", f.GetType())
	}

	// wildcard
	f = FromString("c:steve*.*")
	if f.Drive != 2 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "STEVE???" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}
	if f.GetType() != "???" {
		t.Fatalf("type wrong, got '%v'", f.GetName())
	}

	f = FromString("c:test.C*")
	if f.Drive != 2 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "TEST" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}
	if f.GetType() != "C??" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
	}

}

func TestDoesMatch(t *testing.T) {

	type testcase struct {
		// pattern contains a pattern
		pattern string

		// yes contains a list of filenames that should match that pattern
		yes []string

		// no contains a list of filenames that should NOT match that pattern
		no []string
	}

	tests := []testcase{
		{
			pattern: "*.com",
			yes:     []string{"A.COM", "B:FOO.COM"},
			no:      []string{"A", "BOB", "C.GO"},
		},
		{
			pattern: "A*",
			yes:     []string{"ANIMAL", "B:AUGUST"},
			no:      []string{"ANIMAL.COM", "BOB", "AURORA.COM"},
		},
		{
			pattern: "A*.*",
			yes:     []string{"ANIMAL.com", "B:AUGUST.com", "AURORA"},
			no:      []string{"Test", "BOB"},
		},
	}

	for _, test := range tests {

		f := FromString(test.pattern)

		for _, ei := range test.no {

			if f.DoesMatch(ei) {
				t.Fatalf("file %s matched pattern %s and it should not have done", ei, test.pattern)
			}
		}

		for _, joo := range test.yes {

			if !f.DoesMatch(joo) {
				t.Fatalf("file %s did not match pattern %s and it should have done", joo, test.pattern)
			}
		}
	}
}
