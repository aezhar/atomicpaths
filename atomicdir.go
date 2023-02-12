// Copyright 2022 individual contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package atomicpaths

import (
	"errors"
	"io/fs"
	"os"
)

func dirCloseAgainError(d *Dir) error { return os.ErrInvalid }

func dirCloseCommitted(d *Dir) error {
	d.closeFn = dirCloseAgainError
	return nil
}

func dirCloseUncommitted(d *Dir) error {
	d.closeFn = dirCloseAgainError
	return os.RemoveAll(d.tempPath)
}

// Dir represents a temporary directory that can on success be "committed"
// to the provided path and rolled back otherwise.
type Dir struct {
	tempPath, origPath string

	closeFn     func(f *Dir) error
	isCommitted bool
}

// Name returns the path to the temporary directory to be modified.
func (d *Dir) Name() string { return d.tempPath }

// OriginalPath returns the path to the original directory.
func (d *Dir) OriginalPath() string { return d.origPath }

// Close closes the Dir instance, removing any uncommitted temporary
// files.
func (d *Dir) Close() error { return d.closeFn(d) }

// Commit commits the temporary directory to the original path by
// deleting the original path, if necessary, and moving the temporary
// directory to the original's path location.
func (d *Dir) Commit() error {
	if d.isCommitted {
		return ErrCommitted
	}

	if err := move(d.Name(), d.OriginalPath()); err != nil {
		return err
	}
	d.closeFn = dirCloseCommitted
	d.isCommitted = true
	return nil
}

func CreateDir(origPath string, perm fs.FileMode) (*Dir, error) {
	for i := 0; i < 1000; i++ {
		tempPath, err := makeTempPath(origPath)
		if err != nil {
			return nil, err
		}

		if err := os.Mkdir(tempPath, perm); err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return nil, err
		}

		return &Dir{
			closeFn:  dirCloseUncommitted,
			tempPath: tempPath,
			origPath: origPath,
		}, nil
	}
	return nil, ErrExhausted
}
