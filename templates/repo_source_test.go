package templates

import (
	"io/fs"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

// repoSourceStandInPhrases are the ways step and skill templates used to
// point an agent at a repo's code without naming it: the directory the agent
// happened to be running in, the project's working tree standing in for the
// colocated repo, and a release note written to a root CHANGELOG.md in the
// code. Plan 000046 replaced every one of them with the roster's per-repo
// source, and this list keeps them from coming back.
var repoSourceStandInPhrases = []string{
	"directory you are running in",
	"shares this project's working tree",
	"root `CHANGELOG.md`",
	"project root's `CHANGELOG.md`",
	"repo-level `CHANGELOG.md`",
	"update_repo_changelog",
	"resolved root",
	"resolved local path",
}

// rosterTemplates are the templates that must render the repo roster with
// each repo's source: the three plan/spec steps that already carried the
// roster, the implement workflow's opening step (the roster is rendered once
// there and carried forward by the agent rather than repeated in every
// code-touching step), and the changelog step that attributes changes to
// repos.
var rosterTemplates = []string{
	"steps/spec/00b-interview.md",
	"steps/plan/02-discovery.md",
	"steps/plan/03-architecture.md",
	"steps/implement/01-read_plan.md",
	"steps/implement/10-update_feature_changelog.md",
}

// Phase 3.2 criterion 1: no step or skill template contains a stand-in for a
// repo's source, and the walk visits enough files that an empty walk cannot
// pass.
func TestNoTemplateUsesARepoSourceStandIn(t *testing.T) {
	visited := 0
	for _, dir := range []string{"steps", "skills"} {
		err := fs.WalkDir(FS, dir, func(p string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() || path.Ext(p) != ".md" {
				return nil
			}
			visited++
			body := mustReadTemplate(t, p)
			for _, phrase := range repoSourceStandInPhrases {
				require.NotContainsf(t, body, phrase,
					"%s must not use the stand-in phrase %q; direct the agent to the repo's source instead", p, phrase)
			}
			return nil
		})
		require.NoErrorf(t, err, "walking %s", dir)
	}
	require.GreaterOrEqual(t, visited, 40, "expected the walk to visit at least 40 step and skill templates")
}

// Phase 3.2 criteria 2 and 3: the spec interview, the plan discovery and
// architecture steps, and the code-touching implement steps all render the
// roster with each repo's source, and every one carries the no-repos
// fallback.
func TestRosterTemplatesRenderEachRepoSource(t *testing.T) {
	for _, f := range rosterTemplates {
		body := mustReadTemplate(t, f)
		require.Containsf(t, body, "{{#repos}}", "%s must iterate the repo roster", f)
		require.Containsf(t, body, "{{#source}} (source: `{{source}}`){{/source}}", "%s must render each repo's source on its roster line", f)
		require.Containsf(t, body, "{{^repos}}", "%s must carry a fallback for a project with no registered repos", f)
		require.Containsf(t, body, "No repos are registered in this project's configuration", "%s must render the no-repos fallback", f)
		require.NotContainsf(t, body, "colocated repo only", "%s must not describe the fallback as the colocated repo", f)
	}
}

// Phase 3.2 criterion 3: the implement workflow's opening step directs all
// code-touching work into each repo's source for the rest of the workflow.
// The later steps rely on the agent carrying that roster forward, so only the
// opening step carries the block.
func TestImplementOpeningStepDirectsWorkToEachRepoSource(t *testing.T) {
	body := mustReadTemplate(t, "steps/implement/01-read_plan.md")
	require.Contains(t, body, "**Where the code lives.**", "the read_plan step must open with the where-the-code-lives block")
	require.Contains(t, body, "For the rest of this workflow", "the read_plan step must scope the direction to the whole workflow")
	require.Contains(t, body, "in each repo's source listed below", "the read_plan step must direct work into each repo's source")
}

// Phase 3.2 criterion 4: the changelog steps define an unprefixed path as
// belonging to the only registered repo and require a prefix whenever more
// than one repo is registered.
func TestChangelogStepsDefinePrefixRuleByRegisteredRepoCount(t *testing.T) {
	for _, f := range []string{
		"steps/implement/07-update_changelog.md",
		"steps/implement/10-update_feature_changelog.md",
	} {
		body := mustReadTemplate(t, f)
		require.Containsf(t, body, "whenever more than one repo is registered", "%s must require prefixes when more than one repo is registered", f)
		require.Containsf(t, body, "belong", "%s must say which repo an unprefixed path belongs to", f)
		require.Containsf(t, body, "the only registered repo", "%s must define unprefixed paths as the only registered repo's", f)
	}
}
