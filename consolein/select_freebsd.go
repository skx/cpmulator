//go:+build freebsd

package consolein

import (
	"os"
	"syscall"
)

// fdget returns index and offset of fd in fds.
func fdget(fd int, fds *syscall.FdSet) (index, offset int) {
	index = fd / (syscall.FD_SETSIZE / len(fds.X__fds_bits)) % len(fds.X__fds_bits)
	offset = fd % (syscall.FD_SETSIZE / len(fds.X__fds_bits))
	return
}

// fdset implements FD_SET macro.
func fdset(fd int, fds *syscall.FdSet) {
	idx, pos := fdget(fd, fds)
	fds.X__fds_bits[idx] = 1 << uint(pos)
}

// canSelect contains a platform-specific implementation of code that tries to use
// SELECT to read from STDIN.
func canSelect() bool {

	var readfds syscall.FdSet

	fdset(int(os.Stdin.Fd()), &readfds)

	// See if input is pending, for a while.
	err := syscall.Select(1, &readfds, nil, nil, &syscall.Timeval{Usec: 200})
	if err != nil {
		return false
	}

	return true
}
