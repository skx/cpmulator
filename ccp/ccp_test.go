package ccp

import (
	"strings"
	"testing"
)

// TestCCPTrivial is a trivial test that we have contents
func TestCCPTrivial(t *testing.T) {

	// Test that we have two CCPs
	if len(ccps) != 2 {
		t.Fatalf("we should have two CCPs")
	}

	// Get each one
	for _, n := range ccps {

		//  Get the size
		bytes := n.Bytes

		// The CCPs are small, but bigger than 1k and smaller than 8k
		if len(bytes) < 1024 {
			t.Fatalf("CCP %s is too small got %d bytes", n.Name, len(bytes))
		}

		if len(bytes) > 8192 {
			t.Fatalf("CCP %s is too large", n.Name)
		}
	}

}

// TestInvalidCCP tests that a CCP with a bogus name isn't found, and
// that the error contains the known values which exist.
func TestInvalidCCP(t *testing.T) {

	_, err := Get("foo")
	if err == nil {
		t.Fatalf("expected failure to load CCP, but got it")
	}

	if !strings.Contains(err.Error(), "ccp") {
		t.Fatalf("error message didn't include valid ccp: ccp")
	}
	if !strings.Contains(err.Error(), "ccpz") {
		t.Fatalf("error message didn't include valid ccp: ccpz")
	}

}

func TestRetrieveAll(t *testing.T) {
	all := GetAll()

	for _, item := range all {
		obj, err := Get(item.Name)
		if err != nil {
			t.Fatalf("failure to get by name %v:%s", item, err)
		}
		if item.Name != obj.Name {
			t.Fatalf("bogus result")
		}
	}
}
