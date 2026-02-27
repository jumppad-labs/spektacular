package runner

import (
	"fmt"
	"sort"

	"github.com/jumppad-labs/spektacular/internal/config"
)

var registry = map[string]func() Runner{}

// Register adds a runner constructor for a given command name.
// It is typically called from an init() function in the runner's package.
func Register(name string, constructor func() Runner) {
	registry[name] = constructor
}

// NewRunner returns a Runner for the agent command specified in the config.
func NewRunner(cfg config.Config) (Runner, error) {
	constructor, ok := registry[cfg.Agent.Command]
	if !ok {
		return nil, fmt.Errorf("unsupported runner: %q (available: %v)", cfg.Agent.Command, registeredNames())
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
