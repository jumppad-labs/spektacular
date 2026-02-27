// Package defaults embeds the static asset files bundled with spektacular.
package defaults

import (
	"embed"
	"fmt"
)

//go:embed files files/.gitignore
var FS embed.FS

// ReadFile returns the content of a named embedded file.
// name is relative to the "files/" prefix, e.g. "agents/planner.md".
func ReadFile(name string) ([]byte, error) {
	data, err := FS.ReadFile("files/" + name)
	if err != nil {
		return nil, fmt.Errorf("embedded file %q not found: %w", name, err)
	}
	return data, nil
}

// MustReadFile returns the content of a named embedded file, panicking on error.
func MustReadFile(name string) []byte {
	data, err := ReadFile(name)
	if err != nil {
		panic(err)
	}
	return data
}
