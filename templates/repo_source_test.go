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

// repoListConsumers are the step templates that need to know where a repo's
// code lives. None of them carry a roster: each sends the agent to
// `repo list`, the single live source for it.
var repoListConsumers = []string{
	"steps/spec/00b-interview.md",
	"steps/plan/02-discovery.md",
	"steps/plan/03-architecture.md",
	"steps/implement/01-read_plan.md",
	"steps/implement/10-update_feature_changelog.md",
	"steps/resume.md",
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

// No template caches the roster: `repo list` is the single live source for
// where a repo's code lives, so neither a step nor the working context file
// may carry a copy that can go stale against the registry.
func TestNoTemplateRendersItsOwnRoster(t *testing.T) {
	visited := 0
	for _, dir := range []string{"steps", "skills"} {
		err := fs.WalkDir(FS, dir, func(p string, d fs.DirEntry, err error) error {
			require.NoError(t, err)
			if d.IsDir() || path.Ext(p) != ".md" {
				return nil
			}
			visited++
			body := mustReadTemplate(t, p)
			require.NotContainsf(t, body, "{{#repos}}",
				"%s must send the agent to `repo list`, not render its own roster", p)
			require.NotContainsf(t, body, "## Repos",
				"%s must not point at a Repos section; the working context no longer carries one", p)
			return nil
		})
		require.NoErrorf(t, err, "walking %s", dir)
	}
	require.GreaterOrEqual(t, visited, 40, "expected the walk to visit at least 40 step and skill templates")
}

// The working context template that used to render the roster is gone: the
// CLI owns no part of context.md any more.
func TestNoContextRosterTemplateExists(t *testing.T) {
	_, err := fs.ReadFile(FS, "context/repos.md")
	require.Error(t, err, "templates/context/repos.md must not exist; `repo list` is the only roster")
}

// Every step that needs to know where a repo's code lives sends the agent to
// `repo list`, which reports the registry as it stands right now rather than
// as it stood when some earlier command wrote a file.
func TestRepoListConsumersSendTheAgentToRepoList(t *testing.T) {
	for _, f := range repoListConsumers {
		body := mustReadTemplate(t, f)
		require.Containsf(t, body, "{{config.command}} repo list", "%s must send the agent to `repo list`", f)
		require.Containsf(t, body, "`root`", "%s must name the root that repo list reports", f)
	}
}

// The implement workflow's opening step directs every code-touching step for
// the rest of the run into each repo's source, so the later steps need not
// repeat it.
func TestImplementOpeningStepDirectsWorkToEachRepoSource(t *testing.T) {
	body := mustReadTemplate(t, "steps/implement/01-read_plan.md")
	require.Contains(t, body, "**Where the code lives.**", "the read_plan step must open with the where-the-code-lives block")
	require.Contains(t, body, "the rest of this workflow", "the read_plan step must scope the direction to the whole workflow")
	require.Contains(t, body, "never in whatever directory you started in", "the read_plan step must rule out the starting directory")
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
