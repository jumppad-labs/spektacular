# Context: implement workflow for 000039_version-check

## Workflow state

- Implement workflow started via spek-implement skill; plan chosen by user:
  000039_version-check. Currently past read_plan, heading into analyze.
- First-phase invocation: plan.md has no `## Changelog` section yet; start at
  Phase 1.1 (first unchecked phase).
- User's standing preference: drive straight through steps, skip per-step
  confirmations on simple features; only stop for real design decisions.

## read_plan results

- Structural validation passed: all 10 required plan.md sections present;
  phases 1.1 / 2.1 / 3.1 all have technical-detail links resolving to
  matching `### Phase N.M:` headings in context.md.
- Drift check passed: every file/symbol referenced in plan+context exists
  (cmd/root.go version var :20-23, versionString :37, runInit :21,
  dataDir :209, loadConfig :196, workflowSkills/installWorkflowSkills in
  internal/agent/skills.go, schemaProp/schemaObj/commandSchema in cmd/spec.go,
  runRootCmd cmd/root_test.go:68, envelope contract tests ~:92/:219/:332,
  4 SKILL.md templates, changelog bare-parent pattern). cmd/version.go does
  not exist yet — it is the planned new file.
- Spec coverage check passed: all 6 requirements + 6 acceptance criteria of
  spec 000039_version-check are covered by phases 1.1/2.1/3.1. No descoping
  needed.
- Watch-point from plan: the three envelope contract tests are t.Run-subtest
  tables; adding a two-word `version check` family should be a trivial
  subtest addition. STOP and ask only if it needs harness restructuring.

## Key design facts (from plan docs)

- `.spektacular/version`: bare version string + newline (no sha), written in
  runInit after config rewrite; committed to git; re-init overwrites — that's
  the whole repair path.
- `version check` (new cmd/version.go): bare cobra parent `version` +
  `check` subcommand; VersionCheckResult{status, installed_version(omitempty),
  current_version, action(omitempty)}; exit 0 for all three states
  (match/mismatch/missing); ErrorResponse (exit 1) only for genuine faults
  (unreadable file); pure classifyVersion(recorded, current) function;
  --schema short-circuit per cmd/skill.go pattern; read-only.
- Preamble: identical ~5-line block after frontmatter in all 4
  templates/skills/workflows/*/SKILL.md using `{{command}} version check`;
  no mustache partial. Dogfood re-init (`go run . init claude`) in Phase 3.1.
- Tests: oracle independence (assert literal "0.1.0" dev default, don't
  derive via versionString); t.TempDir()+t.Chdir convention; add family to
  three envelope contract tables.

## Phase 1.1 — COMPLETE (checked in plan.md, changelog entry written)

- cmd/version.go created: versionFilePath(dataDir), writeVersionFile(path,v)
  writing v+"\n" mode 0644 — Phase 2.1 extends this same file with the
  version command. cmd/init.go: version file written after config rewrite;
  output line `  Version:  <path> (<version>)` after the Project line.
- Tests in cmd/init_test.go: TestInit_Claude extended (version file ==
  "0.1.0\n"; fresh dir covers pre-feature-repo criterion),
  TestInit_RewritesStaleVersionFile new, TestInit_Idempotent extended.
- Verify: go build / go test ./... / go vet all green; Makefile lint = go
  vet only, no golangci-lint config.
- plan.md: Phase 1.1 heading + all 4 criteria checked; `## Changelog`
  section created at end of plan.md with the 1.1 entry.
- Now looping to analyze for Phase 2.1 (version check command). Remaining
  unchecked: 2.1, 3.1. Auto-looping per user's standing preference.

## Phase 2.1 — COMPLETE (checked in plan.md, changelog entry written)

- Heading + all 5 criteria checked; changelog entry appended. Verified all
  green. Now looping to analyze for Phase 3.1 (skill preamble + dogfood
  re-init) — the final phase.

## Phase 3.1 — COMPLETE (checked in plan.md, changelog entry written)

- Heading + all 4 criteria checked; changelog entry appended. Zero
  unchecked phases remain.

## Closing steps

- CHANGELOG.md (repo root): `## 000039_version-check` section prepended
  (user-facing summary).
- test-plan.md committed to plan store: two manual procedures — (1) live
  agent relays the stale prompt on skill invocation in a mismatched repo
  (release-manager check per release/agent), (2) field observation of
  staleness-attributable bug reports. Metric 2 (no false alarms) is
  automated, not restated.
- Feature changelog record committed to the changelog store
  (000039_version-check.md): what was built (3 parts), why it matters,
  deviations: none (watch-points resolved favorably; resetVersionCheckFlags
  added for flag hygiene).
- Spec reconciled: all 6 requirements + 6 acceptance criteria flipped to
  [x] — every checkbox is delivered by phases 1.1/2.1/3.1 per the plan
  changelog (live-agent relay verification remains a manual test-plan
  item, but the mechanism itself shipped). Advancing to finished.

## Phase 3.1 analyze results

- No mismatches. Templates: spek-new/plan/implement have STOP blockquote
  right after frontmatter; spek-knowledge has `# What this skill does`.
  Preamble inserts between the closing `---` and that first content, in all
  four, using `{{command}} version check`.
- Agent tests: claude/bob/codex_test.go each have a skillAssertions loop
  over installed SKILL.md — extend with a "version check" Contains
  assertion (test step's job). instruction_surface_test.go walks
  skills/workflows automatically.
- Dogfood re-init (`go run . init claude`) happens at the end of the
  implement step for this phase; review diff before proceeding.

## Phase 3.1 implement results

- Identical 4-line blockquote preamble ("Version check first.") inserted
  after frontmatter in all four templates/skills/workflows/*/SKILL.md,
  before the STOP block (spek-new/plan/implement) / before "# What this
  skill does" (spek-knowledge). Uses `{{command}} version check` and
  `{{command}} init <agent>`.
- Dogfood re-init run: .claude/skills/*/SKILL.md re-rendered with the
  preamble (`go run . version check`), .spektacular/version written with
  0.1.0, AGENTS.md managed sections re-rendered. `go run . version check`
  in this repo → status match, silent. Note: AGENTS.md and
  templates/agents/memory-context.md had pre-existing uncommitted user
  modifications from before this session.
- Tests (sub-agent): "spektacular version check" containment assertion
  added inside the skillAssertions loops of claude/bob/codex_test.go (12
  rendered files); TestInit_CustomCommand asserts "go run . version check"
  (substitution with non-default command). No template-walking duplicate
  added. go test ./internal/agent/ ./cmd/ passes.
- Verify (sub-agent): build/test/vet green; `go run . version check` in
  this repo → match; every .claude/skills/*/SKILL.md contains the
  preamble. Phase 3.1 verified.

## Phase 2.1 analyze results

- No mismatches. --schema pattern: cmd/skill.go:28-47 (Bool flag "schema",
  build commandSchema, output.Write(w, s, "")); schema types cmd/spec.go
  (schemaProp/schemaObj/commandSchema); success path
  output.New(cmd.OutOrStdout(), globalFields).WriteResult(result).
- Watch-point RESOLVED: envelope contract tests are plain t.Run subtests
  (TestWrapper_SuccessAndFailureBothStreamOnStdoutOnly,
  TestWrapper_ErrorDiscriminantAndExitCode,
  TestWrapper_FailureIsPrintedExactlyOnceWithNoCobraBoilerplate in
  cmd/root_test.go) — add a "version" subtest to each; no harness change.
- Failure path for contract tests: make .spektacular/version a DIRECTORY →
  os.ReadFile fails non-IsNotExist → ErrorResponse exit 1.
- config.NewDefault(): Command "spektacular", Agent "" — action text
  suggests "`<command> init <agent>`", appending agent only when non-empty.
- ErrorResponse is returned as error from RunE (runRoot envelope-izes);
  NewError(...).WithResource(...).WithNextAction(...).

## Phase 2.1 implement results

- cmd/version.go extended: VersionCheckResult struct; versionCmd bare
  parent + versionCheckCmd; runVersionCheck with --schema short-circuit
  (status enum in schema), dataDir()+os.ReadFile, IsNotExist→missing,
  other read error→ErrorResponse "version_file_unreadable"; pure
  classifyVersion(recorded,current) (trim; ""→missing; ==→match; else
  mismatch); staleAction(status) composes relay text with
  `<cfg.Command> init [<cfg.Agent>]` (agent appended only when set).
  init() registers schema flag + subcommand; rootCmd.AddCommand(versionCmd)
  in cmd/root.go.
- Smoke-tested all three states + --schema against a built binary in
  scratch dir: correct status/action/exit 0; build+vet clean.
- Tests (sub-agent): new cmd/version_test.go — TestClassifyVersion table,
  TestVersionCheck_{Match,Mismatch,Missing,WhitespaceOnlyFile,
  UnreadableFileIsGenuineFault,Schema}; read-only guarantee asserted in
  mismatch test; resetVersionCheckFlags added for --schema flag hygiene.
  cmd/root_test.go: "version" subtests added to all three wrapper contract
  tests (failure representative: version file as directory → exit 1).
  go test ./cmd/ passes.
- Verify (sub-agent): go build / go test ./... / go vet all green.
  Phase 2.1 verified.

## Learnings

- Pre-existing bug (out of scope, mention to user at end):
  templates/agents/memory-context.md:3 hardcodes `spektacular init` instead
  of `{{command}} init`.
