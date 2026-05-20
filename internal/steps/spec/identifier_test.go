package spec

import (
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

func identifierStore(t *testing.T) store.Store {
	t.Helper()
	return store.NewFileStore(t.TempDir(), "project")
}

func writeExistingSpec(t *testing.T, st store.Store, name string) {
	t.Helper()
	// Generated-mode resolution scans the default spec directory when the
	// request carries no SpecDir, so the fixture must land in the same place.
	writeExistingSpecIn(t, st, config.DefaultSpecDir, name)
}

func writeExistingSpecIn(t *testing.T, st store.Store, dir, name string) {
	t.Helper()
	require.NoError(t, st.Write(SpecFilePath(dir, name), []byte("existing")))
}

func fixedIdentifierTime() time.Time {
	return time.Date(2026, time.May, 8, 21, 2, 3, 0, time.FixedZone("EDT", -4*60*60))
}

func TestResolveIdentifier_DefaultTimestamp(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:  "Billing.Export",
		Store: st,
		Now:   fixedIdentifierTime,
	})

	require.NoError(t, err)
	require.Equal(t, "20260509010203-billing-export", got.Name)
}

func TestResolveIdentifier_TimestampCollisionBumpsSeconds(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "20260509010203-billing-export")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing-export",
		Method: IDMethodTimestamp,
		Store:  st,
		Now:    fixedIdentifierTime,
	})

	require.NoError(t, err)
	require.Equal(t, "20260509010204-billing-export", got.Name)
}

func TestResolveIdentifier_ExplicitTimestampMethod(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "Billing@Export",
		Method: IDMethodTimestamp,
		Store:  st,
		Now:    fixedIdentifierTime,
	})

	require.NoError(t, err)
	require.Equal(t, "20260509010203-billing-export", got.Name)
}

func TestResolveIdentifier_CounterEmptyStoreStartsAtOne(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing-export",
		Method: IDMethodCounter,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "000001_billing-export", got.Name)
}

func TestResolveIdentifier_CounterUsesMaxFromStorePlusOne(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "000007_old-feature")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing-export",
		Method: IDMethodCounter,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "000008_billing-export", got.Name)
}

func TestResolveIdentifier_CounterIgnoresTimestampAndExternalPrefixes(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "000003_old-feature")
	writeExistingSpec(t, st, "20260509010203-other-feature")
	writeExistingSpec(t, st, "ext-123-external")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing-export",
		Method: IDMethodCounter,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "000004_billing-export", got.Name)
}

func TestResolveIdentifier_CounterCollisionBumpsValue(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "000007_old-feature")
	writeExistingSpec(t, st, "000008_billing-export")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing-export",
		Method: IDMethodCounter,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "000009_billing-export", got.Name)
}

// TestResolveIdentifier_CounterEnumeratesConfiguredSpecDir asserts that counter
// resolution scans the configured (non-default) spec directory for existing
// specs (Phase 2.2, criterion 1).
func TestResolveIdentifier_CounterEnumeratesConfiguredSpecDir(t *testing.T) {
	st := identifierStore(t)
	// Existing counter-prefixed spec under the non-default directory.
	writeExistingSpecIn(t, st, "my-specs", "000007_old-feature")
	// A decoy under the default "specs" dir must be ignored.
	writeExistingSpecIn(t, st, "specs", "000099_decoy")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:    "billing-export",
		Method:  IDMethodCounter,
		SpecDir: "my-specs",
		Store:   st,
	})

	require.NoError(t, err)
	require.Equal(t, "000008_billing-export", got.Name)
}

func TestResolveIdentifier_ExplicitIDOverridesGeneratedMethod(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "Billing Export",
		ID:     "EXT.User@123",
		Method: IDMethodCounter,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "ext-user-123-billing-export", got.Name)
}

func TestResolveIdentifier_ExplicitIDCollisionFails(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "ext-123-billing")

	_, err := ResolveIdentifier(IdentifierRequest{
		Name:  "billing",
		ID:    "EXT.123",
		Store: st,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestResolveIdentifier_ExternalRequiresID(t *testing.T) {
	st := identifierStore(t)

	_, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing",
		Method: IDMethodExternal,
		Store:  st,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "id is required")
}

func TestResolveIdentifier_ExternalUsesNormalizedID(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:   "Billing Export",
		ID:     "EXT.User@123",
		Method: IDMethodExternal,
		Store:  st,
	})

	require.NoError(t, err)
	require.Equal(t, "ext-user-123-billing-export", got.Name)
}

func TestResolveIdentifier_UnknownMethodReturnsError(t *testing.T) {
	st := identifierStore(t)

	_, err := ResolveIdentifier(IdentifierRequest{
		Name:   "billing",
		Method: "unsupported",
		Store:  st,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported spec.id_method")
}

func TestResolveIdentifier_RejectsUnsafeOrUntrimmedValuesBeforeStoreUse(t *testing.T) {
	tests := []struct {
		name string
		req  IdentifierRequest
		want string
	}{
		{
			name: "leading whitespace name",
			req:  IdentifierRequest{Name: " billing", Method: IDMethodTimestamp},
			want: "leading or trailing whitespace",
		},
		{
			name: "trailing whitespace id",
			req:  IdentifierRequest{Name: "billing", ID: "ext ", Method: IDMethodTimestamp},
			want: "leading or trailing whitespace",
		},
		{
			name: "path separator name",
			req:  IdentifierRequest{Name: "billing/export", Method: IDMethodTimestamp},
			want: "path separators",
		},
		{
			name: "control character id",
			req:  IdentifierRequest{Name: "billing", ID: "bad\nid", Method: IDMethodTimestamp},
			want: "control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveIdentifier(tt.req)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestResolveIdentifier_NilStoreFailsGeneratedModes(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{name: "default timestamp"},
		{name: "timestamp", method: IDMethodTimestamp},
		{name: "counter", method: IDMethodCounter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ResolveIdentifier(IdentifierRequest{
				Name:   "billing",
				Method: tt.method,
				Now:    fixedIdentifierTime,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "store required")
		})
	}
}
