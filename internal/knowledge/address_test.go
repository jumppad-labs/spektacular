package knowledge

import (
	"testing"

	"github.com/jumppad-labs/spektacular/internal/output"
	"github.com/stretchr/testify/require"
)

// requireRefusal asserts err is a structured knowledge refusal carrying the
// expected code and a non-empty next action. It returns the envelope so a
// caller can make further assertions about the guidance it offers.
func requireRefusal(t *testing.T, err error, code string) *output.ErrorResponse {
	t.Helper()
	require.Error(t, err)
	var envelope *output.ErrorResponse
	require.ErrorAs(t, err, &envelope)
	require.Equal(t, code, envelope.Code)
	require.NotEmpty(t, envelope.NextAction, "a refusal must tell the caller how to reissue the request")
	return envelope
}

// Criteria 1 and 6: an address that does not name exactly one store is refused,
// with the code that says which part is wrong. An omitted tier, an omitted
// name, both omitted, a tier that is neither of the two addressable tiers, and
// the fan-out tier "all" are each rejected.
func TestAddress_ValidateRefusesAddressesThatDoNotNameOneStore(t *testing.T) {
	cases := map[string]struct {
		addr Address
		code string
	}{
		"no tier":       {Address{Name: "docs"}, ErrCodeTierRequired},
		"no name":       {Address{Tier: TierRepo}, ErrCodeNameRequired},
		"neither":       {Address{}, ErrCodeTierRequired},
		"unknown tier":  {Address{Tier: "team", Name: "docs"}, ErrCodeTierInvalid},
		"fan-out tier":  {Address{Tier: TierAll, Name: "docs"}, ErrCodeTierInvalid},
		"all, no name":  {Address{Tier: TierAll}, ErrCodeTierInvalid},
		"unknown, none": {Address{Tier: "team"}, ErrCodeTierInvalid},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			requireRefusal(t, tc.addr.Validate(), tc.code)
		})
	}
}

// Criterion 6: the two addressable tiers are accepted when paired with a name.
func TestAddress_ValidateAcceptsBothAddressableTiers(t *testing.T) {
	require.NoError(t, Address{Tier: TierProject, Name: "team"}.Validate())
	require.NoError(t, Address{Tier: TierRepo, Name: "docs"}.Validate())
}

// A selector may name the fan-out tier and may carry no filter at all, since it
// names a set of stores rather than one. An absent or unrecognised tier is
// still refused.
func TestSelector_ValidateAcceptsFanOutAndRefusesUnknownTier(t *testing.T) {
	require.NoError(t, Selector{Tier: TierAll}.Validate())
	require.NoError(t, Selector{Tier: TierRepo, Filter: []string{"docs"}}.Validate())

	requireRefusal(t, Selector{}.Validate(), ErrCodeTierRequired)
	requireRefusal(t, Selector{Tier: "team"}.Validate(), ErrCodeTierInvalid)
}
