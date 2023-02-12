package atomicpaths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type stateFlags int

func (s *stateFlags) is(f stateFlags) bool {
	return *s&f == f
}

func (s *stateFlags) set(f stateFlags) {
	*s |= f
}

const (
	placed stateFlags = 1 << iota
	closed
	synced
)

func regCloseAgainError(*File) error {
	return os.ErrInvalid
}

func regCloseCommitted(f *File) error {
	f.closeFn = regCloseAgainError
	return nil
}

func regCloseUncommitted(f *File) error {
	f.closeFn = regCloseAgainError
	return errors.Join(f.File.Close(), os.Remove(f.Name()))
}

func regCloseUncommittedAfterSync(f *File) error {
	f.closeFn = regCloseAgainError
	return os.Remove(f.Name())
}

// File represents a temporary file that on success can be "committed"
// to the provided path and rolled back otherwise.
type File struct {
	*os.File

	origPath string
	parentFd int
	closeFn  func(f *File) error

	state stateFlags
}

// OriginalPath returns the path to the original file.
func (f *File) OriginalPath() string {
	return f.origPath
}

// Close closes the File instance, removing any uncommitted temporary
// files.
func (f *File) Close() error { return f.closeFn(f) }

// Commit flushes all unwritten changes to disk, closes the underlying
// temporary file, making it impossible to apply any changes, and
// commits the temporary file to the original path.
func (f *File) Commit() error {
	if f.state.is(synced) {
		return ErrAlreadyCommitted
	}

	if !f.state.is(closed) {
		if err := f.File.Sync(); err != nil {
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		if err := f.File.Close(); err != nil {
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		f.state.set(closed)
	}

	if !f.state.is(placed) {
		oldName := filepath.Base(f.Name())
		newName := filepath.Base(f.OriginalPath())
		if err := rename(f.parentFd, oldName, newName); err != nil {
			f.closeFn = regCloseUncommittedAfterSync
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		f.closeFn = regCloseCommitted

		f.state.set(placed)
	}

	if !f.state.is(synced) {
		if err := unix.Fsync(f.parentFd); err != nil {
			err = &fs.PathError{Op: "sync", Path: filepath.Dir(f.Name()), Err: err}
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		if err := unix.Close(f.parentFd); err != nil {
			err = &fs.PathError{Op: "close", Path: filepath.Dir(f.Name()), Err: err}
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}

		f.state.set(synced)
	}

	return nil
}

// CreateFile creates a temporary file that can be either atomically
// committed to the given path or discarded.
func CreateFile(origPath string, perm os.FileMode) (*File, error) {
	parentPath := filepath.Dir(origPath)

	flags := unix.O_RDONLY
	flags |= unix.O_DIRECTORY
	flags |= unix.O_CLOEXEC
	parentFd, err := unix.Open(parentPath, flags, 0)
	if err != nil {
		return nil, err
	}

	origName := filepath.Base(origPath)
	for i := 0; i < 1000; i++ {
		tempName, err := makeTempName(origName)
		if err != nil {
			return nil, err
		}

		flags := unix.O_RDWR
		flags |= unix.O_CREAT
		flags |= unix.O_EXCL
		flags |= unix.O_CLOEXEC
		fileFd, err := unix.Openat(parentFd, tempName, flags, uint32(perm))
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return nil, err
		}

		tempPath := filepath.Join(parentPath, tempName)
		af := &File{
			File:     os.NewFile(uintptr(fileFd), tempPath),
			parentFd: parentFd,
			origPath: origPath,
			closeFn:  regCloseUncommitted,
		}
		return af, nil
	}
	return nil, ErrExhausted
}
