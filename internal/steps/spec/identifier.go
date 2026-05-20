package spec

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/store"
)

var counterPrefixRE = regexp.MustCompile(`^(\d{6})_`)

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
	SpecDir string // configured spec directory; defaults to config.DefaultSpecDir when empty
	Store   store.Store
	Now     func() time.Time
}

// IdentifierResult is the canonical spec name.
type IdentifierResult struct {
	Name string
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

	specDir := req.SpecDir
	if specDir == "" {
		specDir = config.DefaultSpecDir
	}

	if req.ID != "" {
		id, err := NormalizeIdentifierPart("id", req.ID)
		if err != nil {
			return IdentifierResult{}, err
		}
		resolved, err := resolveWithPrefix(req.Store, specDir, id, name)
		if err != nil {
			return IdentifierResult{}, err
		}
		return IdentifierResult{Name: resolved}, nil
	}

	switch method {
	case IDMethodExternal:
		return IdentifierResult{}, fmt.Errorf("id is required when spec.id_method is %q", IDMethodExternal)
	case IDMethodTimestamp:
		return resolveTimestamp(req, specDir, name)
	case IDMethodCounter:
		return resolveCounter(req, specDir, name)
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

func resolveTimestamp(req IdentifierRequest, specDir, name string) (IdentifierResult, error) {
	now := req.Now
	if now == nil {
		now = time.Now
	}

	timestamp := now().UTC()
	for {
		resolved := fmt.Sprintf("%s-%s", timestamp.Format("20060102150405"), name)
		exists, err := specExists(req.Store, specDir, resolved)
		if err != nil {
			return IdentifierResult{}, err
		}
		if !exists {
			return IdentifierResult{Name: resolved}, nil
		}
		timestamp = timestamp.Add(time.Second)
	}
}

func resolveCounter(req IdentifierRequest, specDir, name string) (IdentifierResult, error) {
	next, err := nextCounterFromStore(req.Store, specDir)
	if err != nil {
		return IdentifierResult{}, err
	}
	for {
		resolved := fmt.Sprintf("%06d_%s", next, name)
		exists, err := specExists(req.Store, specDir, resolved)
		if err != nil {
			return IdentifierResult{}, err
		}
		if !exists {
			return IdentifierResult{Name: resolved}, nil
		}
		next++
	}
}

// nextCounterFromStore scans the specs directory for filenames whose names
// start with `<digits>-` and returns max+1. Returns 1 when no such files
// exist (or the directory does not yet exist).
func nextCounterFromStore(st store.Store, specDir string) (int, error) {
	if st == nil {
		return 0, fmt.Errorf("store required for spec identifier resolution")
	}
	entries, err := st.List(specDir)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 1, nil
		}
		return 0, err
	}
	max := 0
	for _, e := range entries {
		base := strings.TrimSuffix(e.Name, ".md")
		m := counterPrefixRE.FindStringSubmatch(base)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

func resolveWithPrefix(st store.Store, specDir, prefix, name string) (string, error) {
	resolved := fmt.Sprintf("%s-%s", prefix, name)
	exists, err := specExists(st, specDir, resolved)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("spec %q already exists", resolved)
	}
	return resolved, nil
}

func specExists(st store.Store, specDir, name string) (bool, error) {
	if st == nil {
		return false, fmt.Errorf("store required for spec identifier resolution")
	}
	return st.Exists(SpecFilePath(specDir, name)), nil
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
