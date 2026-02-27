package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, ".spektacular", "specs")
	err := os.MkdirAll(specsDir, 0755)
	require.NoError(t, err)
	return dir
}

func TestCreate_WritesSpecFile(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "my-feature", "", "")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "My Feature")
}

func TestCreate_DefaultsTitle_FromName(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "cool-thing", "", "")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "Cool Thing")
}

func TestCreate_UsesProvidedTitle(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "feature", "My Custom Title", "")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "My Custom Title")
}

func TestCreate_UsesProvidedDescription(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "feature", "", "A custom description")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "A custom description")
}

func TestCreate_FileAlreadyExists_ReturnsError(t *testing.T) {
	dir := setupProject(t)

	_, err := Create(dir, "duplicate", "", "")
	require.NoError(t, err)

	_, err = Create(dir, "duplicate", "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestCreate_AppendsExtension(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "no-ext", "", "")
	require.NoError(t, err)
	require.Equal(t, "no-ext.md", filepath.Base(path))
}

func TestCreate_DoesNotDuplicateExtension(t *testing.T) {
	dir := setupProject(t)

	path, err := Create(dir, "with-ext.md", "", "")
	require.NoError(t, err)
	require.Equal(t, "with-ext.md", filepath.Base(path))
}
