package spec

import (
	"testing"
	"time"

	"github.com/jumppad-labs/spektacular/internal/store"
	"github.com/stretchr/testify/require"
)

func identifierStore(t *testing.T) store.Store {
	t.Helper()
	return store.NewFileStore(t.TempDir())
}

func writeExistingSpec(t *testing.T, st store.Store, name string) {
	t.Helper()
	require.NoError(t, st.Write(SpecFilePath(name), []byte("existing")))
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
	require.Equal(t, 0, got.Counter)
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
	require.Equal(t, 0, got.Counter)
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

func TestResolveIdentifier_CounterUsesNextValue(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:    "billing-export",
		Method:  IDMethodCounter,
		Counter: 7,
		Store:   st,
	})

	require.NoError(t, err)
	require.Equal(t, "000008-billing-export", got.Name)
	require.Equal(t, 8, got.Counter)
}

func TestResolveIdentifier_CounterCollisionBumpsValue(t *testing.T) {
	st := identifierStore(t)
	writeExistingSpec(t, st, "000008-billing-export")

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:    "billing-export",
		Method:  IDMethodCounter,
		Counter: 7,
		Store:   st,
	})

	require.NoError(t, err)
	require.Equal(t, "000009-billing-export", got.Name)
	require.Equal(t, 9, got.Counter)
}

func TestResolveIdentifier_ExplicitIDOverridesGeneratedMethod(t *testing.T) {
	st := identifierStore(t)

	got, err := ResolveIdentifier(IdentifierRequest{
		Name:    "Billing Export",
		ID:      "EXT.User@123",
		Method:  IDMethodCounter,
		Counter: 7,
		Store:   st,
	})

	require.NoError(t, err)
	require.Equal(t, "ext-user-123-billing-export", got.Name)
	require.Equal(t, 7, got.Counter)
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
