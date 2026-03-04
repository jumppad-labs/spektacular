package runner

import (
	"fmt"
	"sort"
)

var registry = map[string]func() Runner{}

// Register adds a runner constructor for a given command name.
// It is typically called from an init() function in the runner's package.
func Register(name string, constructor func() Runner) {
	registry[name] = constructor
}

// NewRunner returns a Runner for the given command name.
func NewRunner(command string) (Runner, error) {
	constructor, ok := registry[command]
	if !ok {
		return nil, fmt.Errorf("unsupported runner: %q (available: %v)", command, registeredNames())
	}
	return constructor(), nil
}

func registeredNames() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
