package version

import (
	"strings"
	"testing"
)

// TestVersion is a nop-test that performs coverage of our version package.
func TestVersion(t *testing.T) {
	x := GetVersionString()
	y := GetVersionBanner()

	// Banner should have our version
	if !strings.Contains(y, x) {
		t.Fatalf("banner doesn't contain our version")
	}
}
