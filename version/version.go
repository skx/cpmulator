// Package version exists solely so that we can store the version of this application
// in one location, despite needing it in two places within the application.
//
// First, and foremost, we need the version to be available by the main.go driver-package,
// but secondly we also want to report our version in one of our expanded BIOS functions.
//
// Duplicating the version number/tag in two places is a recipe for drift and confusion,
// so this internal-package is the result.
package version

import "fmt"

var (
	// version is populated with our release tag, via a Github Action.
	//
	// See .github/build in the source distribution for details.
	version = "unreleased"
)

// GetVersionBanner returns a banner which is suitable for printing, to show our name,
// version, and homepage link.
func GetVersionBanner() string {

	str := fmt.Sprintf("cpmulator %s\n%s\n", version, "https://github.com/skx/cpmulator/")
	return str
}

// GetVersionString returns our version number as a string.
func GetVersionString() string {
	return version
}
