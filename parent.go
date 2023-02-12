package atomicpaths

import (
	"golang.org/x/sys/unix"
)

func openParent(path string) (fd int, err error) {
	flags := unix.O_RDONLY
	flags |= unix.O_DIRECTORY
	flags |= unix.O_CLOEXEC
	fd, err = unix.Open(path, flags, 0)
	return
}
