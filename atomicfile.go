package atomicpaths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

// File represents a temporary file that on success can be "committed"
// to the provided path and rolled back otherwise.
type File struct {
	*os.File

	closeFn  func() error
	commitFn func() error

	origPath string
	parentFd int
	state    state
}

func (f *File) closeAgainError() error {
	return os.ErrInvalid
}

func (f *File) closeCommitted() error {
	f.closeFn = f.closeAgainError
	return nil
}

func (f *File) closeUncommitted() (err error) {
	f.closeFn = f.closeAgainError
	f.commitFn = f.commitClosed

	if !f.state.is(closed) {
		multierr.AppendInto(&err, f.File.Close())
		f.state.set(closed)
	}

	if !f.state.is(placed) {
		multierr.AppendInto(&err, os.Remove(f.Name()))
		f.state.set(placed)
	}

	if !f.state.is(synced) {
		multierr.AppendInto(&err, unix.Close(f.parentFd))
		f.state.set(synced)
	}

	return
}

func (f *File) commitClosed() error {
	return ErrRolledBack
}

func (f *File) commitCommitted() error {
	return ErrAlreadyCommitted
}

func (f *File) commitUncommitted() error {
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
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}

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
		f.closeFn = f.closeCommitted
		f.commitFn = f.commitCommitted

		f.state.set(synced)
	}

	return nil
}

// OriginalPath returns the path to the original file.
func (f *File) OriginalPath() string {
	return f.origPath
}

// Close closes the File instance, removing any uncommitted temporary
// files.
func (f *File) Close() error {
	return f.closeFn()
}

// Commit flushes all unwritten changes to disk, closes the underlying
// temporary file, making it impossible to apply any changes, and
// commits the temporary file to the original path.
//
// Commit can be called repeatedly in case of an error to resolve
// the returned problem and try again until the changes have been
// committed successfully or abandoned.
func (f *File) Commit() error {
	return f.commitFn()
}

// CreateFile creates a temporary file that can be either atomically
// committed to the given path or discarded.
func CreateFile(origPath string, perm os.FileMode) (*File, error) {
	parentPath := filepath.Dir(origPath)

	parentFd, err := openParent(parentPath)
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
		}
		af.closeFn = af.closeUncommitted
		af.commitFn = af.commitUncommitted
		return af, nil
	}
	return nil, ErrExhausted
}
