package atomicpaths

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
	return errors.Join(f.File.Close(), os.Remove(f.tempPath))
}

func regCloseUncommittedAfterSync(f *File) error {
	f.closeFn = regCloseAgainError
	return os.Remove(f.tempPath)
}

// File represents a temporary file that on success can be "committed"
// to the provided path and rolled back otherwise.
type File struct {
	*os.File

	tempPath, origPath string

	closeFn     func(f *File) error
	isCommitted bool
	isClosed    bool
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
	if f.isCommitted {
		return ErrAlreadyCommitted
	}

	if !f.isClosed {
		if err := f.File.Sync(); err != nil {
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		if err := f.File.Close(); err != nil {
			return fmt.Errorf("atomicpaths.commit: %w", err)
		}
		f.isClosed = true
	}

	if err := move(f.Name(), f.OriginalPath()); err != nil {
		f.closeFn = regCloseUncommittedAfterSync
		return fmt.Errorf("atomicpaths.commit: %w", err)
	}
	f.isCommitted = true

	f.closeFn = regCloseCommitted
	return nil
}

// CreateFile creates a temporary file that can be either atomically
// committed to the given path or discarded.
func CreateFile(origPath string, perm os.FileMode) (*File, error) {
	for i := 0; i < 1000; i++ {
		tempPath, err := makeTempName(origPath)
		if err != nil {
			return nil, err
		}

		f, err := os.OpenFile(tempPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return nil, err
		}

		af := &File{
			File:     f,
			tempPath: tempPath,
			origPath: origPath,
			closeFn:  regCloseUncommitted,
		}
		return af, nil
	}
	return nil, ErrExhausted
}
