package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file tests the skill command family's serving of library skills
// (templates/skills/skill_*.md via `skill <name>` and `skill list`), focused
// on Phase 4.2's repo-management and cross-repo orchestration content. The
// wrapper/envelope mechanics themselves are covered in root_test.go.

// skillProject seeds a minimal project fixture in a fresh temp dir and chdirs
// into it, so skill invocations never read or touch this repository's own
// .spektacular state.
func skillProject(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	writeSpecCommandConfig(t, dir, "")
}

// fetchSkillInstructions runs `skill <name>` through the production wrapper
// and returns the served instructions from the decoded success envelope.
func fetchSkillInstructions(t *testing.T, name string) string {
	t.Helper()
	stdout, stderr, code := runRootCmd(t, "skill", name)
	require.Equal(t, 0, code)
	require.Empty(t, stderr)

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &m))
	require.Equal(t, false, m["error"])
	require.Equal(t, name, m["name"], "the served skill must identify itself by name")

	instructions, ok := m["instructions"].(string)
	require.True(t, ok, "the envelope must carry string instructions")
	require.NotEmpty(t, instructions)
	return instructions
}

// Criterion 4: `skill manage-repos` returns the repo-management skill's
// content — registration, inspection, and the never-fetch/pull rule.
func TestSkillManageRepos_IsRetrievable(t *testing.T) {
	skillProject(t)
	instructions := fetchSkillInstructions(t, "manage-repos")

	require.Contains(t, instructions, "repo add",
		"manage-repos must document the `repo add` registration command")
	require.Contains(t, instructions, "repo list",
		"manage-repos must document the `repo list` inspection command")
	require.Contains(t, instructions, "never fetches or pulls",
		"manage-repos must state Spektacular never fetches or pulls on its own")
}

// Criterion 4: manage-repos is a library skill served raw — its content
// carries no command placeholder in either form.
func TestSkillManageRepos_ServedWithoutCommandPlaceholders(t *testing.T) {
	skillProject(t)
	instructions := fetchSkillInstructions(t, "manage-repos")

	require.NotContains(t, instructions, "{{command}}",
		"library skills are served unrendered and must not carry the {{command}} placeholder")
	require.NotContains(t, instructions, "{{config.command}}",
		"library skills are served unrendered and must not carry the {{config.command}} placeholder")
}

// Criterion 4: the skill listing includes manage-repos alongside the existing
// skills.
func TestSkillList_IncludesManageRepos(t *testing.T) {
	skillProject(t)
	stdout, stderr, code := runRootCmd(t, "skill", "list")
	require.Equal(t, 0, code)
	require.Empty(t, stderr)

	var listing struct {
		Skills []string `json:"skills"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &listing))
	require.Contains(t, listing.Skills, "manage-repos",
		"the listing must include the repo-management skill")
	require.Contains(t, listing.Skills, "spawn-planning-agents",
		"the listing must still include the existing planning skill")
	require.Contains(t, listing.Skills, "spawn-implementation-agents",
		"the listing must still include the existing implementation skill")
}

// Criterion 4: the unknown-skill error's available-skills list names
// manage-repos, so an agent probing with a wrong name is steered to it.
func TestSkillUnknownName_ErrorListsManageRepos(t *testing.T) {
	skillProject(t)
	stdout, stderr, code := runRootCmd(t, "skill", "does-not-exist")
	require.Equal(t, 1, code)
	require.Empty(t, stderr)
	require.Contains(t, stdout, "manage-repos",
		"the unknown-skill error must list manage-repos among the available skills")
}

// Criterion 3: the spawn-implementation-agents skill directs carrying each
// piece of attributed work out inside its repo's resolved root, keeping one
// agent's changes per repo so verification and changelog derivation stay
// per-repo.
func TestSkillSpawnImplementationAgents_DirectsWorkToAttributedRepo(t *testing.T) {
	skillProject(t)
	instructions := fetchSkillInstructions(t, "spawn-implementation-agents")

	require.Contains(t, instructions, "Work belongs in its attributed repo",
		"the skill must open with the attributed-repo rule")
	require.Contains(t, instructions, "carry that work out inside that repo's resolved root",
		"attributed work must be carried out in the repo's resolved root")
	require.Contains(t, instructions, "repo list",
		"resolved local paths must come from the `repo list` command")
	require.Contains(t, instructions, "changelog derivation stay per-repo",
		"changes must stay per-repo so derived changelog entries follow the attribution")
}

// Criteria 1 and 2: the spawn-planning-agents skill fans research out per
// registered repo, using each repo's metadata to scope which repos a question
// belongs to.
func TestSkillSpawnPlanningAgents_FansResearchOutPerRepo(t *testing.T) {
	skillProject(t)
	instructions := fetchSkillInstructions(t, "spawn-planning-agents")

	require.Contains(t, instructions, "Multi-repo projects",
		"the skill must carry the multi-repo fan-out paragraph")
	require.Contains(t, instructions, "fan the research out per repo",
		"research must fan out across registered repos")
	require.Contains(t, instructions, "repo list",
		"resolved local paths must come from the `repo list` command")
	require.Contains(t, instructions, "description, role, and tags",
		"repo metadata must scope which repos a research question is relevant to")
}
