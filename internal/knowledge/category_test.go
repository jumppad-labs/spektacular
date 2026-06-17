package knowledge

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Criterion 1: every category in the registry has a non-empty Purpose,
// Boundary, and EntryShape — the descriptive contract a contributor relies on.
func TestCategories_AllHaveDescriptiveFields(t *testing.T) {
	for _, c := range Categories {
		require.NotEmpty(t, c.Purpose, "category %q must have a Purpose", c.Name)
		require.NotEmpty(t, c.Boundary, "category %q must have a Boundary", c.Name)
		require.NotEmpty(t, c.EntryShape, "category %q must have an EntryShape", c.Name)
	}
}

// Criterion 2: the registry contains exactly the six known categories, by name,
// in registry order.
func TestCategories_ContainsExactlyTheSixKnownCategories(t *testing.T) {
	names := make([]string, len(Categories))
	for i, c := range Categories {
		names[i] = c.Name
	}
	require.Equal(t, []string{
		"conventions",
		"glossary",
		"architecture",
		"gotchas",
		"learnings",
		"decisions",
	}, names)
}

// Criterion 3: glossary is declared always-applied and decisions is declared
// looked-up — the two tiers, read from the registry.
func TestCategoryByName_DeclaresExpectedTiers(t *testing.T) {
	glossary, ok := CategoryByName("glossary")
	require.True(t, ok)
	require.Equal(t, TierAlwaysApplied, glossary.Tier)

	decisions, ok := CategoryByName("decisions")
	require.True(t, ok)
	require.Equal(t, TierLookedUp, decisions.Tier)
}

// Criterion 4: AlwaysApplied returns exactly conventions then glossary, in
// registry order.
func TestAlwaysApplied_ReturnsConventionsAndGlossaryInOrder(t *testing.T) {
	require.Equal(t, []string{"conventions", "glossary"}, AlwaysApplied())
}

// Criterion 5: CategoryByName reports found for a known name and not-found,
// with a zero Category, for an unknown name.
func TestCategoryByName_FoundForKnownMissingForUnknown(t *testing.T) {
	c, ok := CategoryByName("architecture")
	require.True(t, ok)
	require.Equal(t, "architecture", c.Name)

	zero, ok := CategoryByName("nonexistent")
	require.False(t, ok)
	require.Equal(t, Category{}, zero)
}
