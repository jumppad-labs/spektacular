package repo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRelativeToConfig_ExpressesATargetRelativeToTheConfigFolder pins the
// translation a registry location must go through: an entry's location is
// resolved from the folder holding config.yaml, so a target sitting beside
// the project must be written as a path climbing out of that folder.
func TestRelativeToConfig_ExpressesATargetRelativeToTheConfigFolder(t *testing.T) {
	// The project keeps its config in /p/.spektacular, and the target is the
	// sibling /code: two levels up, then down into code.
	require.Equal(t, "../../code", relativeToConfig("/p", "/p/../code"))
}

// TestRelativeToConfig_KeepsAPathThatHasNoRelativeForm covers the fallback:
// no relative path leads from a relative base to an absolute target, so the
// absolute path is kept rather than a broken relative one being invented.
func TestRelativeToConfig_KeepsAPathThatHasNoRelativeForm(t *testing.T) {
	require.Equal(t, "/code", relativeToConfig("p", "/code"))
}

// TestStringSlice_ReadsTagsInBothShapesTheyArriveIn covers every shape a list
// of answers reaches registration in. Values gathered in the session that
// asked for them are a typed slice; the same values read back out of the
// JSON-persisted workflow data after an interruption are a slice of any.
func TestStringSlice_ReadsTagsInBothShapesTheyArriveIn(t *testing.T) {
	t.Run("typed slice from the gathering session", func(t *testing.T) {
		data := &testData{values: map[string]any{"tags": []string{"docs", "markdown"}}}
		require.Equal(t, []string{"docs", "markdown"}, stringSlice(data, "tags"))
	})

	t.Run("slice of any after a JSON round trip", func(t *testing.T) {
		data := &testData{values: map[string]any{"tags": []any{"docs", "markdown"}}}
		require.Equal(t, []string{"docs", "markdown"}, stringSlice(data, "tags"))
	})

	t.Run("non-string members are skipped", func(t *testing.T) {
		data := &testData{values: map[string]any{"tags": []any{"docs", 7, nil, "markdown"}}}
		require.Equal(t, []string{"docs", "markdown"}, stringSlice(data, "tags"))
	})

	t.Run("a missing key yields nothing", func(t *testing.T) {
		data := &testData{values: map[string]any{}}
		require.Nil(t, stringSlice(data, "tags"))
	})
}
