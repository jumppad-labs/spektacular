package repo

import (
	"strings"

	"github.com/jumppad-labs/spektacular/internal/config"
)

// Roster projects the registry's identity plus each repo's own descriptive
// metadata and the resolved location of its code into the shape workflow
// templates render as the repo roster. Descriptive metadata and the source
// are read the same git-free way `repo list` reads them — from each repo's
// own config when its location is on disk, absent otherwise — so an agent
// working across a multi-repo project sees exactly what listing reports.
// The source is rendered into instructions (an earlier decision kept
// resolved paths out of them) because the roster is refreshed from config on
// every `new` and `goto`, so the value is at most one invocation old; a
// repo whose code is not on disk (an uncloned git source) carries an empty
// source, and the instructions direct the agent to `repo list` for it.
// Materialization state and staleness still come only from `repo list`.
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
			"source":      "",
		}
		if err == nil {
			if meta, ok := set.DescriptiveMetadata(r.Name); ok {
				entry["description"] = meta.Description
				entry["role"] = meta.Role
				entry["tags"] = strings.Join(meta.Tags, ", ")
				entry["deployment"] = meta.Deployment
			}
			if source, ok := set.LocalSource(r.Name); ok {
				entry["source"] = source
			}
		}
		roster = append(roster, entry)
	}
	return roster
}
