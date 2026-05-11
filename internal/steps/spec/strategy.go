package spec

import (
	"path/filepath"

	"github.com/jumppad-labs/spektacular/internal/artifact"
	"github.com/jumppad-labs/spektacular/internal/stepkit"
	"github.com/jumppad-labs/spektacular/internal/workflow"
)

// strategy implements stepkit.PathStrategy for the spec workflow.
type strategy struct{}

func (strategy) PrimaryPathField() string { return "spec_path" }

func (strategy) PathVars(instanceName, storeRoot string) map[string]any {
	return map[string]any{
		"spec_path": filepath.Join(storeRoot, SpecFilePath(instanceName)),
		"spec_name": instanceName,
	}
}

func (s strategy) PathVarsWithConfig(instanceName, storeRoot string, cfg workflow.Config) map[string]any {
	path, err := artifact.Path(cfg.Project, artifact.KindSpec, instanceName)
	if err != nil {
		return s.PathVars(instanceName, storeRoot)
	}
	return map[string]any{
		"spec_path": filepath.Join(storeRoot, path),
		"spec_name": instanceName,
	}
}

// Compile-time interface check.
var _ stepkit.PathStrategy = strategy{}
