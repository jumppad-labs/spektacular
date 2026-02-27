package claude

import (
	"testing"

	"github.com/jumppad-labs/spektacular/internal/runner"
	"github.com/stretchr/testify/require"
)

// Compile-time check that Claude implements runner.Runner.
var _ runner.Runner = (*Claude)(nil)

func TestNew_ReturnsNonNil(t *testing.T) {
	c := New()
	require.NotNil(t, c)
}
