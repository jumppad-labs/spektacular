package templates

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbroglie/mustache"
	"github.com/jumppad-labs/spektacular/internal/config"
	"github.com/stretchr/testify/require"
)

// knowledgeSkillTemplate is the template the per-agent spek-knowledge copies
// are generated from.
const knowledgeSkillTemplate = "skills/workflows/spek-knowledge/SKILL.md"

// scopeAddressingPhrases are the ways a template used to address a knowledge
// entry by a scope label instead of by a tier and a store name: the `scope`
// key in a `knowledge read`/`write` --data payload, a `--scope` flag, a search
// result described as carrying a `scope` field, and the prose that asked an
// agent to pick a scope as the write destination.
//
// The list is deliberately narrow. The bare word "scope" is legitimate all
// over the corpus in an unrelated sense — "out of scope", "scope creep", "a
// scoped decision", "scoped to the repo(s) identified in Step 1" — so matching
// on it would fail on prose that has nothing to do with knowledge addressing.
// Every phrase below is therefore either punctuated as the CLI surface
// (`"scope":`, `--scope`, backticked `scope` as a result field) or is a
// complete clause that only occurs when a template is telling an agent where
// knowledge lives.
var scopeAddressingPhrases = []string{
	`"scope":`,
	"--scope",
	"`scope`",
	"the scope it came from",
	"tagged by scope",
	"available scopes",
	"target scope",
	"scope selection",
}

// knowledgePrecedencePhrases are the ways a template described one knowledge
// store overriding another. Nothing in the tool has ever implemented such an
// ordering, so no template may describe one. "precedence" on its own is not a
// phrase here: the knowledge skill states that there is *no* precedence
// between stores, and that sentence must keep passing.
var knowledgePrecedencePhrases = []string{
	"most specific source",
	"most-specific",
	"layered precedence",
	"specificity",
	"project → team → global",
}

// scopeVocabularyHits reports which of scopeAddressingPhrases appear in body.
func scopeVocabularyHits(body string) []string {
	var hits []string
	for _, phrase := range scopeAddressingPhrases {
		if strings.Contains(body, phrase) {
			hits = append(hits, phrase)
		}
	}
	return hits
}

// walkTemplateMarkdown visits every markdown file in the embedded corpus and
// returns how many it visited, so an empty walk cannot pass a sweep.
func walkTemplateMarkdown(t *testing.T, visit func(p, body string)) int {
	t.Helper()
	visited := 0
	err := fs.WalkDir(FS, ".", func(p string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if d.IsDir() || path.Ext(p) != ".md" {
			return nil
		}
		visited++
		visit(p, mustReadTemplate(t, p))
		return nil
	})
	require.NoError(t, err, "walking the embedded template corpus")
	require.GreaterOrEqual(t, visited, 40, "expected the walk to visit at least 40 templates")
	return visited
}

// Phase 3.2 criterion 2: no template anywhere in the corpus instructs an agent
// to address knowledge by a scope label. Knowledge is addressed by a tier and a
// store name, so the `scope` key, the `--scope` flag, and the prose that treats
// a scope as a destination must not appear in any template. The exemption list
// this sweep carried while Phase 3.1 was in flight is gone: every template it
// named has been rewritten.
func TestNoTemplateAddressesKnowledgeByScope(t *testing.T) {
	walkTemplateMarkdown(t, func(p, body string) {
		require.Emptyf(t, scopeVocabularyHits(body),
			"%s must address knowledge by tier and store name, not by scope: found %v", p, scopeVocabularyHits(body))
	})
}

// Phase 3.1 criterion 4: no template describes one knowledge store taking
// precedence over another. The skill's old `project` → `team` → `global`
// specificity chain is gone and nothing may reintroduce it.
func TestNoTemplateDescribesKnowledgeStorePrecedence(t *testing.T) {
	walkTemplateMarkdown(t, func(p, body string) {
		for _, phrase := range knowledgePrecedencePhrases {
			require.NotContainsf(t, body, phrase,
				"%s must not describe a precedence order between knowledge stores; found %q", p, phrase)
		}
	})

	// The replacement statement is present, so the chain was removed rather
	// than merely reworded.
	body := mustReadTemplate(t, knowledgeSkillTemplate)
	require.Contains(t, body, "There is no precedence between stores",
		"the knowledge skill must state that no store overrides another")
}

// Phase 3.1 criterion 1: before an entry is recorded on the user's behalf the
// contribute flow's propose-then-confirm checkpoint presents a destination
// that states the tier and the store name as well as the path, and waits for
// explicit approval of it.
func TestKnowledgeSkillContributeCheckpointNamesTierStoreAndPath(t *testing.T) {
	body := mustReadTemplate(t, knowledgeSkillTemplate)

	require.Contains(t, body, "**Show the user the destination and the body before writing.**",
		"the contribute flow must show the destination before writing")
	require.Contains(t, body, "The destination must state the **tier**, the **store name**, and the **path**",
		"the checkpoint's destination must state tier, store name and path")
	require.Contains(t, body, "Wait for **explicit confirmation**",
		"the checkpoint must wait for explicit confirmation of that destination")

	// The write the checkpoint approves carries the same three-part address,
	// so what was approved is what is written.
	require.Contains(t, body, `knowledge write --data '{"tier":"<tier>","name":"<name>","path":"<category>/<slug>.md"}'`,
		"the contribute write must address the entry by tier, name and path")
	require.Contains(t, body, `knowledge read --data '{"tier":"<tier>","name":"<name>","path":"<path>"}'`,
		"reads must address the entry by tier, name and path")
}

// Phase 3.1 criterion 3: the skill enumerates the configured stores before
// proposing a destination, and picks only from the names that enumeration
// returns.
func TestKnowledgeSkillEnumeratesStoresBeforeProposing(t *testing.T) {
	body := mustReadTemplate(t, knowledgeSkillTemplate)

	sources := strings.Index(body, "{{command}} knowledge sources")
	require.NotEqual(t, -1, sources, "the skill must run `knowledge sources`")
	require.Contains(t, body, "enumerate the configured stores by `tier` and `name`",
		"`knowledge sources` must be described as enumerating stores by tier and name")
	require.Contains(t, body, "authoritative set of writable destinations and of valid `--filter` names",
		"`knowledge sources` must be the authoritative list of destinations and filter names")
	require.Contains(t, body, "choose only from the names it returns, never a name you inferred",
		"the skill must forbid inventing a store name")

	// The enumeration comes first: it precedes the step that decides which
	// store to write to, and that step draws its tier and name from it.
	decide := strings.Index(body, "Decide (or ask the user) which **store** to write to")
	require.NotEqual(t, -1, decide, "the contribute flow must decide which store to write to")
	require.Less(t, sources, decide, "the skill must enumerate the stores before choosing one")
	require.Contains(t, body, "a `tier` and a `name` from the enumeration in step 1",
		"the chosen store must come from the enumeration")
}

// unconditionalLoadPhrases are the ways a template could claim that some
// knowledge store loads whether or not the agent named it. The discovery step
// used to carry exactly such a claim — "project-owned sources always load
// regardless of `--repo`" — and it was deleted because narrowing is now
// honoured with no exceptions. These needles catch a restatement of the idea.
//
// The corpus contains no occurrence of any of them today, and none collides
// with the legitimate vocabulary around it: "always-applied" (the category
// name, used in ~10 templates) never reads as "always load", because every
// template phrases it as "Load the project's **always-applied** knowledge".
var unconditionalLoadPhrases = []string{
	"always load",
	"always loads",
	"always loaded",
	"load regardless",
	"loads regardless",
	"loaded regardless",
	"always included regardless",
	"always contribute",
}

// narrowingTerms are the words a "regardless of ..." clause would have to
// reach for to be talking about knowledge narrowing rather than about
// something unrelated. The sweep below flags "regardless" only when one of
// these appears on the same line, so a future template may still write
// "regardless" in ordinary prose without tripping the guard.
var narrowingTerms = []string{
	"--repo",
	"--filter",
	"--tier",
	"narrowing",
	"knowledge",
	"store",
	"load",
}

// Phase 3.2 criterion 3: no workflow step still claims that some stores load
// regardless of the narrowing given. A store the agent does not name
// contributes nothing, with no exceptions, so no template may promise that
// anything arrives unasked for.
func TestNoTemplateClaimsStoresLoadRegardlessOfNarrowing(t *testing.T) {
	walkTemplateMarkdown(t, func(p, body string) {
		lower := strings.ToLower(body)
		for _, phrase := range unconditionalLoadPhrases {
			require.NotContainsf(t, lower, phrase,
				"%s must not claim a knowledge store loads unconditionally; found %q", p, phrase)
		}

		for _, line := range strings.Split(lower, "\n") {
			if !strings.Contains(line, "regardless") {
				continue
			}
			for _, term := range narrowingTerms {
				require.NotContainsf(t, line, term,
					"%s must not say knowledge arrives regardless of the narrowing; found %q on the line %q", p, term, strings.TrimSpace(line))
			}
		}
	})

	// The replacement statement is present, so the claim was removed rather
	// than merely reworded into a synonym these needles miss.
	body := mustReadTemplate(t, planDiscoveryStep)
	require.Contains(t, body, "The narrowing is honoured with no exceptions",
		"the discovery step must state that narrowing is honoured with no exceptions")
	require.Contains(t, body, "a store you do not name contributes nothing",
		"the discovery step must say an unnamed store contributes nothing")
}

// planDiscoveryStep is the plan step that performs the always-applied load.
const planDiscoveryStep = "steps/plan/02-discovery.md"

// knowledgeInvocations returns every `<command> knowledge ...` invocation in
// body, each truncated at the end of its backticked code span (or its line).
// Scoping to the invocation itself matters because `--repo` is still a
// legitimate flag elsewhere in the corpus — `changelog file write --repo
// <name>` routes a derived release note into a member repo's own changelog
// store — so a whole-file `--repo` ban would be wrong. Only the knowledge
// commands must be free of it.
func knowledgeInvocations(body string) []string {
	var found []string
	for _, prefix := range []string{"{{config.command}} knowledge ", "{{command}} knowledge "} {
		rest := body
		for {
			i := strings.Index(rest, prefix)
			if i == -1 {
				break
			}
			rest = rest[i+len(prefix):]
			end := len(rest)
			if j := strings.IndexAny(rest, "`\n"); j != -1 {
				end = j
			}
			found = append(found, prefix+rest[:end])
		}
	}
	return found
}

// Phase 3.2 criterion 2 (the discovery step specifically): the always-applied
// load narrows by a tier and a repeatable filter, and no knowledge command
// anywhere in the corpus still narrows with `--repo`.
func TestDiscoveryStepNarrowsAlwaysAppliedByTierAndFilter(t *testing.T) {
	body := mustReadTemplate(t, planDiscoveryStep)

	require.Contains(t, body, "{{config.command}} knowledge always-applied --tier repo --filter <name>",
		"the discovery step must narrow the always-applied load with --tier repo --filter")
	require.Contains(t, body, "repeat `--filter` for each target repo",
		"the discovery step must say --filter is repeatable")

	// A plan that needs shared knowledge asks for it explicitly and names the
	// stores, rather than relying on them arriving on their own.
	require.Contains(t, body, "`--tier all`",
		"the discovery step must tell a plan needing shared knowledge to ask for --tier all")
	require.Contains(t, body, "knowledge sources",
		"the discovery step must point at `knowledge sources` for the available store names")

	// Search hits carry a tier and a store name, and a read takes the
	// three-part address built from them.
	require.Contains(t, body, "tagged with the tier and store name it came from",
		"search hits must be described as carrying a tier and a store name")
	require.Contains(t, body, `knowledge read --data '{"tier":"<tier>","name":"<name>","path":"<path>"}'`,
		"the discovery step's read example must use the three-part address")

	// The knowledge-write proposal names a tier and a store, not a scope.
	require.Contains(t, body, "propose to the user a target tier and store name",
		"the discovery step must propose a tier and store name before writing knowledge")

	// The architecture step's back-reference carries the same narrowing.
	require.Contains(t, mustReadTemplate(t, "steps/plan/03-architecture.md"),
		"knowledge always-applied --tier repo --filter <name>",
		"the architecture step's always-applied reference must carry the narrowing")

	// No knowledge command narrows by `--repo` anywhere in the corpus.
	walkTemplateMarkdown(t, func(p, b string) {
		for _, invocation := range knowledgeInvocations(b) {
			require.NotContainsf(t, invocation, "--repo",
				"%s narrows a knowledge command with --repo; knowledge narrows by --tier and --filter: %q", p, invocation)
		}
	})
}

// managedAgentGuidanceSections is the hand-maintained set of managed sections
// `init` installs into a project's AGENTS.md, mapping each template under
// `templates/agents/` to the `## ` heading `internal/agent`'s
// installManagedSection locates it by. Hand-maintained on purpose: adding a
// section without updating this map fails the test below.
var managedAgentGuidanceSections = map[string]string{
	"agents/repo-sources.md":         "## Where the Code Lives — Read This First",
	"agents/memory-context.md":       "## Memory & Context",
	"agents/spec-trigger.md":         "## Spec-Worthy Discussion Recognition",
	"agents/historical-artifacts.md": "## Historical Artifacts: Specs and Plans as Archaeology",
	"agents/draft-presentation.md":   "## Presenting Drafts and Confirmations",
	"agents/knowledge-trigger.md":    "## Knowledge-Worthy Discovery Recognition",
}

// Phase 3.2 criterion 4: no new managed agent-guidance section was added for
// the knowledge model. The new vocabulary landed by editing the two existing
// sections that mention it, so the set of managed sections is unchanged.
//
// A managed section exists only if it has a template under `templates/agents/`
// (installManagedSection renders one to produce it), so pinning that directory
// to an exact set is what makes an added section fail here.
func TestManagedAgentGuidanceSectionsAreUnchanged(t *testing.T) {
	entries, err := fs.ReadDir(FS, "agents")
	require.NoError(t, err, "reading the managed agent-guidance templates")

	onDisk := map[string]string{}
	for _, e := range entries {
		require.Falsef(t, e.IsDir(), "agents/%s must be a template file, not a directory", e.Name())
		p := path.Join("agents", e.Name())
		body := mustReadTemplate(t, p)

		heading, _, ok := strings.Cut(body, "\n")
		require.Truef(t, ok, "%s must open with its `## ` heading line", p)
		onDisk[p] = heading
	}

	require.Equal(t, managedAgentGuidanceSections, onDisk,
		"the set of managed agent-guidance sections changed; no new section may be added for the knowledge model")

	// The two sections that do mention the knowledge model carry the new
	// vocabulary, so the change landed by editing them rather than by adding
	// a section alongside them.
	for _, p := range []string{"agents/memory-context.md", "agents/knowledge-trigger.md"} {
		require.Containsf(t, mustReadTemplate(t, p), "handles tier and store selection",
			"%s must describe the skill as handling tier and store selection", p)
	}

	// This project's own AGENTS.md carries those sections and no others, so a
	// section installed without a template would still be caught.
	agents, err := os.ReadFile(filepath.Join("..", "AGENTS.md"))
	require.NoError(t, err, "reading this project's AGENTS.md")

	installed := map[string]bool{}
	for _, line := range strings.Split(string(agents), "\n") {
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimRight(line, "\r \t")
			require.Falsef(t, installed[heading], "AGENTS.md carries %q more than once", heading)
			installed[heading] = true
		}
	}

	expected := map[string]bool{}
	for _, heading := range managedAgentGuidanceSections {
		expected[heading] = true
	}
	require.Equal(t, expected, installed,
		"AGENTS.md must carry exactly the managed sections and no others")
}

// generatedSkillCopies are the per-agent skill copies this repository dogfoods
// and tracks in git. Each is produced by `internal/agent.installWorkflowSkills`
// rendering the corresponding template's `{{command}}` placeholder.
var generatedSkillCopies = []string{
	".claude/skills",
	".bob/skills",
}

// generatedSkillNames maps each installed skill directory to the template it
// is generated from. Hand-maintained to match
// `internal/agent/skills.go`'s workflowSkills.
var generatedSkillNames = map[string]string{
	"spek-new":       "skills/workflows/spek-new/SKILL.md",
	"spek-plan":      "skills/workflows/spek-plan/SKILL.md",
	"spek-implement": "skills/workflows/spek-implement/SKILL.md",
	"spek-knowledge": "skills/workflows/spek-knowledge/SKILL.md",
}

// Phase 3.1 criterion 5: every per-agent copy of a workflow skill is byte-for
// byte the template with `{{command}}` rendered to this project's configured
// command, so no agent can be handed a hand-edited copy carrying the old
// vocabulary.
//
// The checked-in copies under `.claude/skills/` and `.bob/skills/` ARE the
// artifacts under test, so — like `cmd/docs_test.go`'s repository-root
// documentation assertions — they are read read-only from the repository root
// relative to this package's directory. They are tracked in git, so they are
// present after a clean checkout.
func TestGeneratedSkillCopiesMatchTheirTemplates(t *testing.T) {
	cfg, err := config.FromYAMLFile(filepath.Join("..", ".spektacular", "config.yaml"))
	require.NoError(t, err, "reading this project's own configuration")
	require.NotEmpty(t, cfg.Command, "the project config must set the command the copies were rendered with")

	for _, skillsDir := range generatedSkillCopies {
		for name, tmplPath := range generatedSkillNames {
			t.Run(skillsDir+"/"+name, func(t *testing.T) {
				rendered, err := mustache.Render(mustReadTemplate(t, tmplPath), map[string]string{"command": cfg.Command})
				require.NoError(t, err, "rendering %s", tmplPath)

				copyPath := filepath.Join("..", filepath.FromSlash(skillsDir), name, "SKILL.md")
				onDisk, err := os.ReadFile(copyPath)
				require.NoErrorf(t, err, "%s must exist; regenerate it with `%s init`", copyPath, cfg.Command)

				require.Equalf(t, rendered, string(onDisk),
					"%s has drifted from %s; regenerate it rather than editing it by hand", copyPath, tmplPath)
			})
		}
	}
}
