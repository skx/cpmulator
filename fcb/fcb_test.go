package fcb

import (
	"fmt"
	"sort"
	"testing"
)

// TestFCBSize ensures our size matches expectations.
func TestFCBSize(t *testing.T) {
	x := FromString("blah")
	b := x.AsBytes()

	if len(b) != 36 {
		t.Fatalf("FCB struct is %d bytes", len(b))
	}

	if x.GetFileName() != "BLAH" {
		t.Fatalf("wrong name returned, got %v", x.GetFileName())
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
	if f.GetCacheKey() != "FOO        " {
		t.Fatalf("name wrong, got '%v'", f.GetCacheKey())
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
	if f.GetFileName() != "THIS-IS-.LON" {
		t.Fatalf("wrong name returned, got %v", f.GetFileName())
	}
	if f.GetCacheKey() != "THIS-IS-LON" {
		t.Fatalf("wrong cache returned, got %v", f.GetCacheKey())
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

	f = FromString("")
	f.Name[0] = 0x00
	f.Name[1] = 0x01
	f.Type[0] = 0x00
	f.Type[1] = 0x01
	if f.GetCacheKey() != "           " {
		t.Fatalf("wrong cache returned, got %v", f.GetCacheKey())
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

// TestGetMatches ensures we can use our matcher.
func TestGetMatches(t *testing.T) {

	f := FromString("*.GO")

	out, err := f.GetMatches("..")
	if err != nil {
		t.Fatalf("failed to get matches")
	}

	if len(out) < 10 {
		t.Fatalf("unexpected number of matches got %d", len(out))
	}

	// sort the files - so we can be predictable
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	// first file, alphabetically
	if out[0].Host != "../ccp/ccp.go" {
		t.Fatalf("unexpected name %s", out[0].Host)
	}

	found := false
	for _, e := range out {
		if e.Host == "../static/static.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("failed to find static.go")
	}

	_, err = f.GetMatches("!>>//path/not/found")
	if err == nil {
		t.Fatalf("expected error on bogus directory, got none")
	}
}

// TestOffset does a trivial test that increases go in steps of 128
func TestOffset(t *testing.T) {

	f := FromString("test")

	// before
	cur := f.GetSequentialOffset()
	if cur != 0 {
		t.Fatalf("unexpected initial offset")
	}

	// bump
	f.SetSequentialOffset(128)

	// after
	after := f.GetSequentialOffset()
	if after == 0 {
		t.Fatalf("unexpected offset after increase")
	}

	// Should have gone up by 128
	if after-128 != cur {
		t.Fatalf("offset should rise by 128")
	}

	// Do a bunch more increases
	remain := 128 * 128
	for remain > 0 {
		f.SetSequentialOffset(128 + f.GetSequentialOffset())
		remain--
	}

	if f.GetSequentialOffset()%128 != 0 {
		t.Fatalf("weird remainder - we should rise in 128-steps")
	}

}

// TestSuffix ensures that the non-printable extensions are replaced with spaces, as expected.
func TestSuffix(t *testing.T) {

	b := make([]byte, 128)
	f := FromBytes(b)

	typ := f.GetType()
	if typ != "   " {
		t.Fatalf("type was weird '%s'", typ)
	}
}

// TestIssue238 tests that #238 is closed - files that were too
// long were showing up.
func TestIssue238(t *testing.T) {

	f := FromString("*.*")

	if f.DoesMatch("DOCKERFILE") {
		t.Fatalf("Dockerfile showed up, and it shouldn't have done.")
	}
	if f.DoesMatch("cpmulator") {
		t.Fatalf("Ourself showed up, and it shouldn't have done.")
	}

}

func TestRandomOffset(t *testing.T) {

	f := FromString("test")

	// before
	cur := f.GetRandomOffset()
	if cur != 0 {
		t.Fatalf("unexpected initial offset")
	}

	// bump
	f.SetRandomOffset(12832)

	// after
	after := f.GetRandomOffset()
	if after == 0 {
		t.Fatalf("unexpected offset after increase")
	}
	if after != 12832 {
		t.Fatalf("unexpected offset after increase")
	}
}
