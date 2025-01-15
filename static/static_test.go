package static

import (
	"strings"
	"testing"
)

// TestStatic just ensures we have some files.
func TestStatic(t *testing.T) {

	// Read the subdirectory
	files, err := GetContent().ReadDir("A")
	if err != nil {
		t.Fatalf("error reading contents")
	}

	// Ensure each file is a .COM files
	for _, entry := range files {
		name := entry.Name()
		if !strings.HasSuffix(name, ".COM") {
			t.Fatalf("file '%s' is not a .COM file", name)
		}
	}
}

// TestEmpty ensures we have no files.
func TestEmpty(t *testing.T) {

	// Read the subdirectory
	files, err := GetEmptyContent().ReadDir(".")
	if err != nil {
		t.Fatalf("error reading contents")
	}

	if len(files) > 0 {
		t.Fatalf("got files, but expected none")
	}
}
