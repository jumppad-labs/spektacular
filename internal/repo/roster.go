package repo

import (
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
)

// Roster projects the registry's identity plus each repo's own descriptive
// metadata into the shape workflow templates render as the repo roster.
// Resolved paths, materialization state, and staleness always come from
// `repo list`, which the rendered instructions direct the agent to run.
// Descriptive metadata is sourced the same way `repo list` sources it — from
// each repo's own config when it is materialized locally, absent otherwise —
// so an agent working across a multi-repo project sees exactly what listing
// reports. Refreshed from config on every invocation so the rendered roster
// is exactly as fresh as the registry itself.
func Roster(cfg config.Config, root string, git GitRunner) []map[string]any {
	roster := make([]map[string]any, 0, len(cfg.Repos))
	set, err := New(cfg, root, git)
	for _, r := range cfg.Repos {
		entry := map[string]any{
			"name":        r.Name,
			"description": "",
			"role":        "",
			"tags":        "",
			"deployment":  "",
		}
		if err == nil {
			if meta, ok := set.DescriptiveMetadata(r.Name); ok {
				entry["description"] = meta.Description
				entry["role"] = meta.Role
				entry["tags"] = strings.Join(meta.Tags, ", ")
				entry["deployment"] = meta.Deployment
			}
		}
		roster = append(roster, entry)
	}
	return roster
}
