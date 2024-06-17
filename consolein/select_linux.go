//go:build linux

package consolein

import (
	"os"
	"syscall"
)

// canSelect contains a platform-specific implementation of code that tries to use
// SELECT to read from STDIN.
func canSelect() bool {

	var readfds syscall.FdSet

	fd := os.Stdin.Fd()
	readfds.Bits[fd/64] |= 1 << (fd % 64)

	// See if input is pending, for a while.
	nRead, err := syscall.Select(1, &readfds, nil, nil, &syscall.Timeval{Usec: 200})
	if err != nil {
		return false
	}

	return (nRead > 0)
}
