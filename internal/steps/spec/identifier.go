package spec

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
)

const (
	IDMethodTimestamp = config.SpecIDMethodTimestamp
	IDMethodCounter   = config.SpecIDMethodCounter
	IDMethodExternal  = config.SpecIDMethodExternal

	MaxIdentifierPartLength = 64
)

// IdentifierRequest describes the data needed to resolve a canonical spec name.
type IdentifierRequest struct {
	Name    string
	ID      string
	Method  string
	Counter int
	Store   store.Store
	Now     func() time.Time
}

// IdentifierResult is the canonical spec name and any updated counter state.
type IdentifierResult struct {
	Name    string
	Counter int
}

// ResolveIdentifier turns a requested spec name plus optional id into a
// canonical spec name.
func ResolveIdentifier(req IdentifierRequest) (IdentifierResult, error) {
	name, err := NormalizeIdentifierPart("name", req.Name)
	if err != nil {
		return IdentifierResult{}, err
	}

	method := req.Method
	if method == "" {
		method = IDMethodTimestamp
	}
	if err := validateMethod(method); err != nil {
		return IdentifierResult{}, err
	}

	if req.ID != "" {
		id, err := NormalizeIdentifierPart("id", req.ID)
		if err != nil {
			return IdentifierResult{}, err
		}
		resolved, err := resolveWithPrefix(req.Store, id, name)
		if err != nil {
			return IdentifierResult{}, err
		}
		return IdentifierResult{Name: resolved, Counter: req.Counter}, nil
	}

	switch method {
	case IDMethodExternal:
		return IdentifierResult{}, fmt.Errorf("id is required when spec.id_method is %q", IDMethodExternal)
	case IDMethodTimestamp:
		return resolveTimestamp(req, name)
	case IDMethodCounter:
		return resolveCounter(req, name)
	default:
		return IdentifierResult{}, fmt.Errorf("unsupported spec.id_method %q", method)
	}
}

// NormalizeIdentifierPart normalizes one user-provided name/id component.
func NormalizeIdentifierPart(label, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if len(raw) > MaxIdentifierPartLength {
		return "", fmt.Errorf("%s must be at most %d characters", label, MaxIdentifierPartLength)
	}
	if raw != strings.TrimSpace(raw) {
		return "", fmt.Errorf("%s must not have leading or trailing whitespace", label)
	}

	var b strings.Builder
	lastHyphen := false
	for _, r := range raw {
		switch {
		case r == '/' || r == '\\':
			return "", fmt.Errorf("%s must not contain path separators", label)
		case unicode.IsControl(r):
			return "", fmt.Errorf("%s must not contain control characters", label)
		case isASCIIAlnum(r):
			b.WriteRune(toASCIILower(r))
			lastHyphen = false
		case r == '_':
			b.WriteRune(r)
			lastHyphen = false
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		default:
			return "", fmt.Errorf("%s contains unsupported character %q", label, r)
		}
	}

	out := b.String()
	if out == "" {
		return "", fmt.Errorf("%s normalizes to empty", label)
	}
	return out, nil
}

func validateMethod(method string) error {
	switch method {
	case IDMethodTimestamp, IDMethodCounter, IDMethodExternal:
		return nil
	default:
		return fmt.Errorf("unsupported spec.id_method %q", method)
	}
}

func resolveTimestamp(req IdentifierRequest, name string) (IdentifierResult, error) {
	now := req.Now
	if now == nil {
		now = time.Now
	}

	timestamp := now().UTC()
	for {
		resolved := fmt.Sprintf("%s-%s", timestamp.Format("20060102150405"), name)
		exists, err := specExists(req.Store, resolved)
		if err != nil {
			return IdentifierResult{}, err
		}
		if !exists {
			return IdentifierResult{Name: resolved, Counter: req.Counter}, nil
		}
		timestamp = timestamp.Add(time.Second)
	}
}

func resolveCounter(req IdentifierRequest, name string) (IdentifierResult, error) {
	counter := req.Counter + 1
	for {
		resolved := fmt.Sprintf("%06d-%s", counter, name)
		exists, err := specExists(req.Store, resolved)
		if err != nil {
			return IdentifierResult{}, err
		}
		if !exists {
			return IdentifierResult{Name: resolved, Counter: counter}, nil
		}
		counter++
	}
}

func resolveWithPrefix(st store.Store, prefix, name string) (string, error) {
	resolved := fmt.Sprintf("%s-%s", prefix, name)
	exists, err := specExists(st, resolved)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("spec %q already exists", resolved)
	}
	return resolved, nil
}

func specExists(st store.Store, name string) (bool, error) {
	if st == nil {
		return false, fmt.Errorf("store required for spec identifier resolution")
	}
	return st.Exists(SpecFilePath(name)), nil
}

func isASCIIAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func toASCIILower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
