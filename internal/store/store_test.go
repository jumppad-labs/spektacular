package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *FileStore {
	t.Helper()
	return NewFileStore(t.TempDir(), "project")
}

func TestWrite_CreatesFileAndParentDirs(t *testing.T) {
	st := newTestStore(t)
	err := st.Write("subdir/file.txt", []byte("hello"))
	require.NoError(t, err)
	data, err := st.Read("subdir/file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("hello"), data)
}

func TestWrite_OverwritesExisting(t *testing.T) {
	st := newTestStore(t)
	require.NoError(t, st.Write("file.txt", []byte("v1")))
	require.NoError(t, st.Write("file.txt", []byte("v2")))
	data, err := st.Read("file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("v2"), data)
}

func TestRead_ReturnsErrNotFoundForMissing(t *testing.T) {
	st := newTestStore(t)
	_, err := st.Read("missing.txt")
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestDelete_RemovesFile(t *testing.T) {
	st := newTestStore(t)
	require.NoError(t, st.Write("file.txt", []byte("data")))
	require.NoError(t, st.Delete("file.txt"))
	require.False(t, st.Exists("file.txt"))
}

func TestDelete_IdempotentOnMissing(t *testing.T) {
	st := newTestStore(t)
	err := st.Delete("nonexistent.txt")
	require.NoError(t, err)
}

func TestList_DistinguishesFilesFromDirectories(t *testing.T) {
	st := newTestStore(t)
	require.NoError(t, st.Write("dir/a.txt", []byte("a")))
	require.NoError(t, st.Write("dir/b.txt", []byte("b")))
	require.NoError(t, st.Write("dir/sub/c.txt", []byte("c")))
	entries, err := st.List("dir")
	require.NoError(t, err)
	require.ElementsMatch(t, []DirEntry{
		{Name: "a.txt", IsDir: false},
		{Name: "b.txt", IsDir: false},
		{Name: "sub", IsDir: true},
	}, entries)
}

func TestList_ReturnsErrNotFoundForMissingDir(t *testing.T) {
	st := newTestStore(t)
	_, err := st.List("nodir")
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestExists_TrueForFile(t *testing.T) {
	st := newTestStore(t)
	require.NoError(t, st.Write("file.txt", []byte("x")))
	require.True(t, st.Exists("file.txt"))
}

func TestExists_FalseForMissing(t *testing.T) {
	st := newTestStore(t)
	require.False(t, st.Exists("missing.txt"))
}

func TestRoot_ReturnsAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	st := NewFileStore(dir, "project")
	require.Equal(t, filepath.Clean(dir), st.Root())
}

func TestNewFileStore_RecordsScope(t *testing.T) {
	st := NewFileStore(t.TempDir(), "project")
	require.Equal(t, "project", st.Scope())
}

var _ Store = (*FileStore)(nil)

func TestPathTraversal_Rejected(t *testing.T) {
	st := newTestStore(t)
	_, err := st.Read("../escape.txt")
	require.Error(t, err)
	err = st.Write("../escape.txt", []byte("x"))
	require.Error(t, err)
	err = st.Delete("../escape.txt")
	require.Error(t, err)
	_, err = st.List("../escape")
	require.Error(t, err)
}
