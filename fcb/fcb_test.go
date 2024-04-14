package fcb

import "testing"

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
	if f.GetType() != "" {
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
	if f.GetType() != "" {
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
	f = FromString("c:steve*")
	if f.Drive != 2 {
		t.Fatalf("drive wrong")
	}
	if f.GetName() != "STEVE???" {
		t.Fatalf("name wrong, got '%v'", f.GetName())
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
