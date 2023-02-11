package atomicpaths_test

import (
	"os"
	"path/filepath"
	"testing"

	requirePkg "github.com/stretchr/testify/require"

	"github.com/aezhar/atomicpaths"
)

func TestFileWriteNew(t *testing.T) {
	require := requirePkg.New(t)

	p := filepath.Join(t.TempDir(), "foo")

	f, err := atomicpaths.CreateFile(p, 0o666)
	require.NoError(err)

	// Initially the file is temporary and does not exist yet
	// under the original path.
	require.NoFileExists(p)

	_, err = f.WriteString("Hello World!\n")
	require.NoError(err)

	// Committing the changes should not error.
	require.NoError(f.Commit())

	// After committing the file content the file should exist
	// now under the original path.
	require.FileExists(p)

	// Ensure the content is correct.
	content, err := os.ReadFile(p)
	require.NoError(err)

	require.Equal([]byte("Hello World!\n"), content)

	require.NoError(f.Close())
}

func TestFileOverwrite(t *testing.T) {
	require := requirePkg.New(t)

	p := filepath.Join(t.TempDir(), "foo")

	// Given there is already a file present.
	require.NoError(os.WriteFile(p, []byte("Hello World!\n"), 0o644))

	// When writing new content to the file ...
	f, err := atomicpaths.CreateFile(p, 0o666)
	require.NoError(err)

	_, err = f.WriteString("Foobar!\n")
	require.NoError(err)

	require.NoError(f.Sync())

	// Then the previous content should still be available.
	content, err := os.ReadFile(p)
	require.NoError(err)

	require.Equal([]byte("Hello World!\n"), content)

	// Unless the temporary file has been committed.
	require.NoError(f.Commit())

	content, err = os.ReadFile(p)
	require.NoError(err)

	require.Equal([]byte("Foobar!\n"), content)
}

func TestFileRollback(t *testing.T) {
	require := requirePkg.New(t)

	p := filepath.Join(t.TempDir(), "foo")

	f, err := atomicpaths.CreateFile(p, 0o666)
	require.NoError(err)

	require.NoError(f.Close())

	// Without committing the file it should not exist at the
	// original path.
	require.NoFileExists(p)
}
