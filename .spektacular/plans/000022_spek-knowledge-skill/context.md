# Context: 000005_spek-knowledge-skill

## Current State Analysis

Spektacular already exposes a complete knowledge-base CRUD surface
via the `spektacular knowledge` command tree, and a complete
agent-skill distribution mechanism via `spektacular init`. The new
`spek-knowledge` skill plugs into both without modifying either.

**Knowledge CRUD surface** (consumer of the new skill's
orchestration):

- `cmd/knowledge.go:106` (`runKnowledgeSearch`) — accepts a positional
  query, fans across every configured source via `knowledge.Set.Search`,
  returns hits tagged with their `scope` field.
- `cmd/knowledge.go:125` (`runKnowledgeRead`) — accepts
  `--data '{"scope","path"}'` and returns the entry contents.
- `cmd/knowledge.go:168` (`runKnowledgeWrite`) — accepts
  `--data '{"scope","path"}'` plus `--file <path>` (or stdin) and
  writes the body into the named scope.
- `cmd/knowledge.go:191` (`runKnowledgeSources`) — returns the
  configured scopes and their locations, ordered as configured.
- `internal/knowledge/set.go:77` (`Set.Search`) — fans across scoped
  stores and emits results tagged with scope (this is what makes
  "search results identify their scope" free).
- `internal/knowledge/set.go:100` (`Set.Write`) — writes into the
  named scope only, leaving other scopes untouched.
- `internal/knowledge/set.go:126` (`Set.Sources`) — drives the
  scope-enumeration step before any contribution.

**Skill distribution mechanism** (the registry hook points):

- `internal/agent/skills.go:25-29` — `workflowSkills` slice. Currently
  three entries (`spek-new`, `spek-plan`, `spek-implement`). The new
  skill adds a fourth.
- `internal/agent/skills.go:39` — `installWorkflowSkills` iterates the
  slice, reads each template, renders `{{command}}`, writes one
  SKILL.md per skill into the agent's skills dir, emits one progress
  line per file.
- `internal/agent/commands.go:18-22` — `workflowDescriptions` map.
  Currently three entries; bob's command-wrapper renderer consumes it.
- `internal/agent/commands.go:28` — `installCommandWrappers` iterates
  `workflowSkills` and renders one wrapper per skill using the
  description from `workflowDescriptions`.
- `internal/agent/claude.go:13`, `internal/agent/codex.go:13`,
  `internal/agent/bob.go:13` — per-agent `Install` methods. Claude and
  codex only install skills (no command wrappers). Bob installs both.
- `templates/skills/workflows/spek-new/SKILL.md`,
  `templates/skills/workflows/spek-plan/SKILL.md`,
  `templates/skills/workflows/spek-implement/SKILL.md` — the three
  existing skill templates. The new skill template lives alongside at
  `templates/skills/workflows/spek-knowledge/SKILL.md`.

**Tests covering the install path** (all need a fourth-skill update):

- `internal/agent/claude_test.go:25-29` — `skillAssertions` map: three
  entries today, four after this plan.
- `internal/agent/codex_test.go:25-29` — same shape.
- `internal/agent/bob_test.go:25-29` — same shape, scoped to
  `.bob/skills/`.
- `internal/agent/bob_test.go:41-45` — `commandAssertions` map for
  bob's per-skill command wrapper: three entries today, four after.
- `internal/agent/bob_test.go:58` — list of forbidden
  stripped-basename filenames (`new.md`, `plan.md`, `implement.md`);
  adding `knowledge.md` here keeps the prefix-preservation guarantee
  honest for the new skill too.
- `internal/agent/instruction_surface_test.go:31-46`
  (`TestEmbeddedTemplatesAvoidStdinInstructionSurface`) — walks the
  `skills/workflows/` and `steps/` template trees and forbids the old
  stdin/heredoc patterns. No code change needed: the walk
  automatically picks up the new template once it exists. The new
  template body must avoid the forbidden substrings listed at
  `internal/agent/instruction_surface_test.go:19-26`.
- `internal/agent/instruction_surface_test.go:51-68`
  (`TestRenderedSkillsAvoidStdinInstructionSurface`) — same forbidden
  list, but applied to the rendered `.claude/skills/` tree under the
  project root. No code change needed; the dogfooded skills tree will
  need to be regenerated with `go run . init claude` after the plan
  lands so this test continues to pass against fresh output.

**Spec / requirements that map directly to existing CLI calls** —
no implementation needed for these; the skill body just has to
invoke them correctly:

- "Look up entries on demand" → `knowledge search`, optionally
  followed by `knowledge read`.
- "Contribute new entries on demand" → `knowledge sources`
  (enumerate), then `knowledge write` after confirmation.
- "Update existing entries" → `knowledge read` then `knowledge write`
  to same scope/path after confirmation.
- "Search results identify their scope" → already in `Set.Search`
  output (scope-tagged hits).
- "Scope selection is explicit" → already in `Set.Sources` output
  (configured scopes are enumerable before any write).
- "Must support all Spektacular-supported agents" → `workflowSkills`
  iteration covers all three agents identically.

## Per-Phase Technical Notes

### Phase 1.1: Register `spek-knowledge` as a fourth workflow skill

**File changes**:

- `internal/agent/skills.go:25-29` — extend the `workflowSkills`
  slice with a fourth entry:
  ```go
  {Name: "spek-knowledge", TemplatePath: "skills/workflows/spek-knowledge/SKILL.md"},
  ```
  The existing iteration at `skills.go:40-63` then handles render,
  mkdir, write, and progress output without modification.

- `internal/agent/commands.go:18-22` — extend
  `workflowDescriptions` with the new description. Match the
  description used in the new SKILL.md frontmatter (Phase 2.1
  finalises wording — Phase 1.1 may use a placeholder description
  consistent with the spec's "Overview" line, e.g. `"Search,
  contribute to, or update the project's knowledge base."`). The
  iteration at `commands.go:40-57` picks it up for bob's wrapper
  rendering.

- `templates/skills/workflows/spek-knowledge/SKILL.md` *(new file)*
  — frontmatter mirrors `spek-plan/SKILL.md` (lines 1-4). Body for
  Phase 1.1 is a minimal placeholder of the form: short "what this
  skill does" paragraph naming the four CRUD commands
  (`{{command}} knowledge search`, `read`, `sources`, `write`); a
  "TBD: full intent-routing playbook lands in Phase 2.1" stub
  marker. The placeholder MUST avoid the forbidden substrings in
  `internal/agent/instruction_surface_test.go:19-26` (no `cat … |
  spektacular knowledge write` patterns). Body uses `{{command}}`
  mustache variable consistently — never the literal string
  `spektacular` — so the rendered file passes the existing
  no-leftover-mustache assertion at `claude_test.go:36`,
  `codex_test.go:36`, `bob_test.go:36, 53`.

- `internal/agent/claude_test.go:25-29` — extend `skillAssertions`
  map with `"spek-knowledge": "knowledge"` (the rendered body must
  reference the `knowledge` CLI namespace at minimum). The iteration
  at lines 30-37 automatically asserts the file exists, contains the
  expected substring, and contains no leftover `{{command}}`.

- `internal/agent/codex_test.go:25-29` — parallel extension; same
  expected substring `"knowledge"`. Iteration loop at lines 30-37
  picks it up.

- `internal/agent/bob_test.go:25-29` — parallel extension for the
  `.bob/skills/` tree.

- `internal/agent/bob_test.go:41-45` — extend `commandAssertions`
  with `"spek-knowledge.md": "`spek-knowledge` skill"` so the
  wrapper rendering is asserted. The iteration at lines 46-54 picks
  it up.

- `internal/agent/bob_test.go:58` — add `"knowledge.md"` to the
  list of forbidden stripped filenames in the iteration at lines
  58-60 so bob's `spek-` prefix-preservation guarantee covers the
  new skill.

**Complexity**: Low

**Token estimate**: ~3k tokens

**Agent strategy**: Single agent, sequential. The diff is small and
tightly coupled — one source edit, four test edits, one new template
file — all touching `internal/agent/` plus `templates/`. Parallel
agents buy nothing and increase merge friction.

### Phase 2.1: Author the `spek-knowledge` skill body

**File changes**:

- `templates/skills/workflows/spek-knowledge/SKILL.md` — replace the
  Phase 1.1 placeholder body with the full natural-language playbook.
  Body sections:

  1. **Frontmatter** — name + description (unchanged from Phase 1.1).
  2. **What this skill does** — explain that the skill orchestrates
     the existing `{{command}} knowledge` CRUD surface for ad-hoc
     read, contribute, and update operations on the configured
     knowledge store, without starting a spec/plan/implement flow.
  3. **When to invoke** — natural-language trigger surface: any user
     reference to the knowledge base, knowledge entries, or phrases
     like "remember", "recall", "what do we know about ...". Single
     skill handles all three intents.
  4. **Intent: lookup** — detect when the user wants to read or
     search. Run `{{command}} knowledge search <query>`. Optionally
     follow with `{{command}} knowledge read --data '{"scope":"…","path":"…"}'`
     for a full read of a promising hit. Present results with the
     `scope` field labeled in the agent's output to the user.
  5. **Intent: contribute** — detect when the user wants to record
     something new. Run `{{command}} knowledge sources` to enumerate
     scopes. Ask the user (or infer from intent) which scope, title,
     and body. Stage the body under `.spektacular/tmp/<slug>.md`
     using the `Write` tool. Show the user the proposed scope, path,
     and body. **Wait for explicit confirmation.** Only after yes,
     run `{{command}} knowledge write --data
     '{"scope":"…","path":"…"}' --file <staged>`. Remove the scratch
     file after a successful write.
  6. **Intent: update** — detect when the user wants to revise an
     existing entry. Run `{{command}} knowledge search` (or `list`)
     to find the target. Read its current content via `{{command}}
     knowledge read`. Apply the user's revision intent to produce
     new content. Stage under `.spektacular/tmp/`. Show the user the
     scope, path, and proposed new body (or a diff). **Wait for
     explicit confirmation.** Then `{{command}} knowledge write` to
     the same scope and path. Remove the scratch file.
  7. **Decline handling** — if the user declines or asks for
     changes, do not invoke `knowledge write`. Loop back to refine
     the proposal or stop.

  The body must:
  - Use `{{command}}` consistently — never bare `spektacular`.
  - Avoid the forbidden substrings at
    `internal/agent/instruction_surface_test.go:19-26` — use
    `--file <staged>` exclusively, never `cat … | {{command}}
    knowledge write`.
  - Be self-contained and agent-agnostic — no Claude-specific tool
    references that other agents won't have.

- `internal/agent/commands.go:18-22` — finalise the description
  string if Phase 1.1 used a placeholder. The description appears
  in bob's slash-command menu — should be one short sentence.

- `internal/agent/instruction_surface_test.go` — no code change
  required; the existing walks (lines 32-45 and 57-67) automatically
  cover the new template once it exists. Verify by running the
  tests; if they pass, no edit needed.

- *(Optional)* Append a new test to
  `internal/agent/instruction_surface_test.go` (after the existing
  `TestRenderedSkillsAvoidStdinInstructionSurface` at lines 51-68)
  asserting the rendered `spek-knowledge` body contains all four
  CRUD invocations (`knowledge search`, `knowledge read`,
  `knowledge sources`, `knowledge write`). This guards against
  future edits accidentally dropping a load-bearing reference.

- Dogfooded `.claude/skills/spek-knowledge/SKILL.md` — regenerate by
  running `go run . init claude` in the spektacular repo root after
  Phase 2.1 lands, so the project's own rendered skills tree stays
  in sync. Required for
  `TestRenderedSkillsAvoidStdinInstructionSurface` to pass.

**Complexity**: Low

**Token estimate**: ~5k tokens

**Agent strategy**: Single agent, sequential. Body authoring is a
single creative act; no parallel opportunity. Behavioural
verification is end-to-end at the Milestone 2 acceptance check, not
automated.

## Testing Strategy

Three test families cover the change:

1. **Per-agent install tests** (`claude_test.go`, `codex_test.go`,
   `bob_test.go`) — extended by one map entry each in Phase 1.1.
   These exercise the full render-and-write path against a
   `t.TempDir()`. The shared fixture-FS substitution at
   `internal/agent/skills.go:33` (`sourceFS`) is reused unchanged.

2. **Instruction-surface tests** (`instruction_surface_test.go`) —
   no code change needed in Phase 1.1 or Phase 2.1; the walk-based
   shape automatically covers the new template. The new template
   body must respect the forbidden-substring list at lines 19-26.
   An optional positive-assertion test in Phase 2.1 documents which
   CRUD invocations are load-bearing.

3. **End-to-end behavioural verification** — manual, at the
   Milestone 2 acceptance check. The Acceptance Criteria checkboxes
   in plan.md drive the script: run the skill in claude, then in
   codex (or bob), against this repo's `.spektacular/knowledge/`
   store. Observe scope labelling, propose-then-confirm behaviour,
   and persistence outcomes.

No new test infrastructure. No changes to the `testify` /
`require` / `t.TempDir()` conventions already used throughout
`internal/agent/`.

## Project References

- Spec: `.spektacular/specs/000005_spek-knowledge-skill.md` (this
  repo).
- Plan: `.spektacular/plans/000005_spek-knowledge-skill/plan.md`
  (this repo).
- Research log: `.spektacular/plans/000005_spek-knowledge-skill/research.md`
  (this repo, written at the next step).
- Target codebase root:
  `/home/nicj/code/github.com/jumppad-labs/spektacular/` (Go CLI
  source).
- Existing workflow skill templates for reference shape:
  `templates/skills/workflows/spek-new/SKILL.md`,
  `templates/skills/workflows/spek-plan/SKILL.md`,
  `templates/skills/workflows/spek-implement/SKILL.md`.
- Knowledge command source: `cmd/knowledge.go` and
  `internal/knowledge/set.go`.
- Memory: `feedback_thin_skill_over_workflow` —
  records the design preference that shaped this plan.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Both phases in this plan are Low (~3k and ~5k respectively) and
implementable by a single sequential agent. No phase warrants
parallel orchestration; the diff surface is too small to subdivide
cleanly.

## Migration Notes

Existing Spektacular projects that want the new skill must re-run
`spektacular init <agent>` after this plan lands. There is no
automated migration step. The re-init is idempotent — it overwrites
the four (post-plan) workflow skill files; it does not touch
project-specific state, knowledge entries, specs, plans, or
`config.yaml`'s agent assignment.

Users who do not re-run init keep their existing three skills and do
not gain `spek-knowledge`. No regression — they simply do not get
the new capability until they opt in.

## Performance Considerations

None. The skill is a static template installed once at init time;
runtime behaviour is just shell invocations of existing
`spektacular knowledge` commands, which are already performance-
constrained by file IO against the local knowledge store (typically
a single-digit number of small markdown files). The change adds no
hot paths, no new IO patterns, and no new caching surfaces.
