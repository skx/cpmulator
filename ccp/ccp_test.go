package ccp

import (
	"strings"
	"testing"
)

// TestCCPTrivial is a trivial test that we have contents
func TestCCPTrivial(t *testing.T) {

	if len(CCPBinary) < 1000 {
		t.Fatalf("CCP is too small")
	}

	if !strings.Contains(string(CCPBinary), "Digital Research") {
		t.Fatalf("missing expected copyright")
	}
}
