package cmd

import (
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/jumppad-labs/spektacular/internal/repo"
	"github.com/jumppad-labs/spektacular/internal/workingcontext"
)

// refreshRoster computes the repo roster (identity, descriptive metadata,
// and each repo's resolved source) from config and, unless dryRun, rewrites
// the CLI-managed "## Repos" block at the top of .spektacular/context.md
// from it. Every workflow `new` and `goto` calls this before doing anything
// else, so the file an agent reads first — whether it is starting a fresh
// workflow or resuming one mid-way — always says where each repo's code
// lives. It never clones or runs git.
func refreshRoster(cfg config.Config, root string, dryRun bool) ([]map[string]any, error) {
	roster := repo.Roster(cfg, root, repoGit)
	if dryRun {
		return roster, nil
	}
	if err := workingcontext.WriteRoster(workingcontext.Path(root), cfg.Command, roster); err != nil {
		return nil, err
	}
	return roster, nil
}
