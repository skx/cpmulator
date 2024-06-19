//go:build unix

package consolein

import (
	"os"

	"golang.org/x/sys/unix"
)

// canSelect contains a platform-specific implementation of code that tries to use
// SELECT to read from STDIN.
func canSelect() bool {

	fds := &unix.FdSet{}
	fds.Set(int(os.Stdin.Fd()))

	// See if input is pending, for a while.
	tv := unix.Timeval{Usec: 200}

	// via select with timeout
	nRead, err := unix.Select(1, fds, nil, nil, &tv)
	if err != nil {
		return false
	}

	return (nRead > 0)
}
