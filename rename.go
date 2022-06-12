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
	"path/filepath"
)

func renameToTemp(path string) (string, error) {
	for i := 0; i < 1000; i++ {
		tempName, err := makeTempName(path + ".original")
		if err != nil {
			return "", err
		}

		switch err := os.Rename(path, tempName); {
		case err == nil:
			// File was renamed successfully.
			return tempName, nil
		case err != nil && !errors.Is(err, fs.ErrExist):
			// Renaming failed for a reason other than the target exists.
			return "", err
		}
	}
	return "", ErrExhausted
}

func forceRemoveAll(p string) error {
	err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}

		if fi.Mode().Perm()&0600 != 0600 {
			return os.Chmod(path, fi.Mode().Perm()|0600)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return os.RemoveAll(p)
}

func move(oldPath, newPath string) error {
	switch err := os.Rename(oldPath, newPath); {
	case err == nil:
		// File was renamed successfully.
		return nil
	case err != nil && !errors.Is(err, fs.ErrExist):
		// Renaming failed for a reason other than the target exists.
		return err
	}

	// If newPath exists (aka. the "original"), try renaming it to a
	// new temporary name first, then renaming oldPath to the newPath,
	// and delete the original. If system crashes in between renaming
	// oldPath to newPath and deleting the "original" newPath, the original
	// file will still be available under the temporary name, so
	// users can recover their data.
	origTemp, err := renameToTemp(newPath)
	if err != nil {
		return err
	}

	if err := move(oldPath, newPath); err != nil {
		return err
	}
	return forceRemoveAll(origTemp)
}
