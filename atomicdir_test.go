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

package atomicpaths_test

import (
	"io/fs"
	"os"
	"testing"

	assertpkg "github.com/stretchr/testify/assert"
	requirepkg "github.com/stretchr/testify/require"

	"github.com/aezhar/atomicpaths"
)

func TestCreateDir(t *testing.T) {
	assert := assertpkg.New(t)

	tempDir := t.TempDir() + "/dirname"

	d, err := atomicpaths.CreateDir(tempDir, 0700)
	if !assert.NoError(err) {
		return
	}

	assert.NotEmpty(d.Name())
	assert.DirExists(d.Name())

	assert.NoDirExists(tempDir)
	assert.Equal(tempDir, d.OriginalPath())
}

func TestCreateDir_CommitNew(t *testing.T) {
	tt := []struct {
		name      string
		createDir bool
	}{
		{"new", false},
		{"existing", true},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			require := requirepkg.New(t)

			tempDir := t.TempDir() + "/dirname"
			if tc.createDir {
				require.NoError(os.MkdirAll(tempDir, 0700))
				require.NoError(os.WriteFile(tempDir+"/foo", []byte("baa"), 0700))

				require.FileExists(tempDir + "/foo")
			} else {
				require.NoDirExists(tempDir)
			}

			d, err := atomicpaths.CreateDir(tempDir, 0700)
			require.NoError(err)

			// Commit temporary directory to the original path with no
			// directory currently present at the original path. Should work.
			require.NoError(d.Commit())

			require.DirExists(tempDir)
			require.NoDirExists(d.Name())

			// Calling Commit a second time should result in an error,
			// since the Dir object is dead and there is nothing to be committed.
			require.ErrorIs(d.Commit(), atomicpaths.ErrAlreadyCommitted)

			// Calling close now should not result in an error since the
			// changes were committed successfully.
			require.NoError(d.Close())

			// Calling close a second time though should report ErrInvalid.
			require.ErrorIs(d.Close(), fs.ErrInvalid)
		})
	}
}
