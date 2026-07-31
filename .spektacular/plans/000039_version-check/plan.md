---
created_date: "2026-07-31"
status: completed
closed_date: "2026-07-31"
---

# Plan: 000039_version-check

<!-- Metadata -->
<!-- Created: 2026-07-31T11:52:00Z -->
<!-- Commit: 8f25cd3 -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

When a user upgrades the Spektacular binary, the skills and agent files it previously installed into a repository can silently go stale — today nothing detects this, so users must remember to re-run init. This plan records the installing binary's version in a dedicated file at init time, adds a cheap `version check` command that reports match, mismatch, or missing-record in structured output, and puts a short preamble in every workflow skill so agents check freshness on each invocation and prompt the user to re-run init when the installation is out of date. Anyone using Spektacular across binary upgrades benefits: staleness announces itself at the next skill use instead of surfacing as mysteriously outdated behavior.

## Conventions

No project conventions apply to this feature: the knowledge base's always-applied tier currently contains only the category README placeholders, no concrete convention entries. Two codebase-enforced standards (guarded by existing tests rather than knowledge entries) do bear on the work and are honoured by the design:

- **Skill templates write the CLI as `{{command}}`, never a rendered command** — the new version-check block added to the four SKILL.md templates uses `{{command}} version check`; enforced by `cmd/init_test.go` and `internal/agent/instruction_surface_test.go`.
- **Commands emit the shared JSON envelope via `internal/output`** — `version check` uses `output.New(...).WriteResult(...)` so the `"error": false` discriminant is injected, and must be added to the envelope contract tables in `cmd/root_test.go`.

## Architecture & Design Decisions

The solution has three small parts, all inside existing seams. First, `init` records the installing binary's version in a new dedicated file, `.spektacular/version`, written in `runInit` (`cmd/init.go`) immediately after the config rewrite — the ldflags-injected `version` variable already lives in package `cmd` (`cmd/root.go:18-23`), so no plumbing or exporting is needed. The file contains the bare version string (e.g. `0.15.1`) followed by a newline; the sha suffix that `versionString()` appends for released builds is deliberately excluded so that two builds of the same release always match. The file is committed to git by default (no gitignore changes), making staleness team-visible: a teammate running an older binary against a freshly re-initialized repo is prompted too.

Second, a new `version check` subcommand (`cmd/version.go`, registered alongside the other commands in `cmd/root.go`) performs the comparison so no skill ever implements it. It reads `.spektacular/version`, trims whitespace, and compares for exact string equality against the current binary's `version` var. It always exits 0 and reports through the standard success envelope: `status` (`match`, `mismatch`, or `missing`), `installed_version`, `current_version`, and — for the two stale states — an `action` field carrying the exact instruction to relay ("the installed Spektacular files are out of date; ask the user to re-run init"). A missing, empty, or whitespace-only file is `missing`; any other unequal content is `mismatch`. Staleness is a state report, not a command failure, so the agent-actionable `ErrorResponse` path (exit 1) is reserved for genuine faults; this keeps `runRoot`'s error envelope meaningful and the check safe to chain in shell. The command never writes anything — refreshing the installation remains an explicit, user-initiated re-run of init, which overwrites `.spektacular/version` with the current binary's version and thereby clears the mismatch.

Third, each of the four workflow skill templates (`templates/skills/workflows/*/SKILL.md`) gains a short, uniform "version check first" block immediately after the YAML frontmatter, instructing the agent to run `{{command}} version check` before anything else, proceed silently on `match`, and on `mismatch`/`missing` tell the user the installed files are stale and ask them to re-run init before continuing. Skills render byte-identically for all three supported agents (claude, bob, codex) from a single template with only the `{{command}}` substitution (`internal/agent/skills.go:38-65`), so this lands everywhere with four template edits and zero per-agent work. The ~5-line block is duplicated rather than extracted into a mustache partial: no partial machinery exists today, and adding a `PartialProvider` for a block this small is more moving parts than the duplication it removes.

This direction beats the alternatives because it satisfies every spec constraint with the least new surface: recording in `config.yaml` or `state.json` violates the dedicated-file constraint or the file's lifecycle; returning an exit-1 error on mismatch conflates "check ran, result: stale" with "check broke"; an AGENTS.md managed section is ambient guidance that cannot guarantee the check fires at skill invocation, which is the spec's explicit trigger. The known dev-build limitation — `go run .` binaries all report the default `0.1.0`, so template drift between dev builds is undetectable by version comparison — is explicitly descoped along with the template-content-hash idea; in a dev repo both sides read `0.1.0`, so checks stay silent and produce no false alarms. See research.md#alternatives-considered-and-rejected for the evidence behind each rejected option.

## Component Breakdown

**Version Recorder (changed: init command)** — The existing init command gains one responsibility: after writing the project config and installing agent files, it writes the dedicated version file containing the current binary's bare version string. It owns the file's creation and overwrite-on-re-init semantics — re-running init with a newer binary is the one and only way the recorded version advances, which is what clears a mismatch. It reads the same build-time version variable the binary already reports for itself, so recorder and reporter can never drift.

**Version Check Command (new)** — A new CLI subcommand that owns the entire comparison: it reads the version file, normalizes it (trim whitespace), compares it for exact equality against the current binary's version, and reports the outcome through the standard success envelope as one of three states — match, mismatch, or missing (absent, empty, or unreadable file). For the two stale states it includes a ready-to-relay action message telling the agent to prompt the user to re-run init. It is strictly read-only — it never touches the version file or any installed files — and reserves the error envelope for genuine faults, so agents branch on the reported state, not on exit codes. It reuses the existing output writer for envelope emission and field filtering.

**Skill Version-Check Preamble (changed: workflow skill templates)** — Each of the four workflow skill templates gains a short, identical block immediately after its frontmatter instructing the agent to run the version check command first, proceed silently on a match, and on mismatch/missing relay the staleness message and stop until the user decides. It owns the "check fires at every skill invocation" guarantee required by the spec. Because all three supported agents install byte-identical renders of these templates, the preamble reaches every agent through the existing skill installer with no per-agent work.

**Output Writer (reused, unchanged)** — The existing JSON envelope writer supplies the success-envelope discriminant, indentation, and field filtering for the new command. No changes; the new command is an additional caller and gets added to the existing envelope contract tests that every JSON-emitting command family must appear in.

## Data Structures & Interfaces

**The version file (new serialization boundary)** — `.spektacular/version` is a plain-text file containing the bare version string of the binary that ran init, followed by a newline (e.g. `0.15.1\n`). No sha suffix, no structure, no frontmatter. It is the contract between the Version Recorder (sole writer) and the Version Check Command (sole reader): the reader trims surrounding whitespace before comparing, so trailing-newline and editor-whitespace differences never cause a false mismatch.

**VersionCheckResult (new)** — the JSON contract between the Version Check Command and the invoking agent, emitted through the standard success envelope (the `"error": false` discriminant is injected by the writer, never declared on the struct):

```go
type VersionCheckResult struct {
    Status           string `json:"status"`                      // "match" | "mismatch" | "missing"
    InstalledVersion string `json:"installed_version,omitempty"` // recorded in .spektacular/version; empty when missing
    CurrentVersion   string `json:"current_version"`             // the running binary's version
    Action           string `json:"action,omitempty"`            // set on mismatch/missing: instruction to relay to the user
}
```

`Status` is the field agents branch on. `Action` carries the ready-to-relay "installed files are out of date — re-run init" instruction so no skill composes its own wording; it is absent on `match`, keeping matching invocations output-minimal.

**Skill preamble contract (changed, textual interface)** — the version-check block added to each workflow SKILL.md template is an instruction contract, not a type: run `{{command}} version check` before any other command; on `status: "match"` continue with zero version-related output; on `"mismatch"` or `"missing"` relay the `action` message and wait for the user's decision before proceeding. The `{{command}}` placeholder is the only substitution, resolved by the existing skill installer.

No existing types change: `config.Config`, the workflow `State`, and the `ErrorResponse` shape are untouched. The command also exposes the existing `--schema` self-description convention, returning the input/output schema of `VersionCheckResult` for agent discovery.

## Implementation Detail

No new patterns are introduced; every piece of the feature follows an established shape in the codebase, which is deliberate — the feature's whole job is to be boring plumbing that agents and reviewers can trust.

**The command follows the existing small-command idiom.** The new `version` command group is a bare cobra parent (mirroring the changelog group), with `check` as its only subcommand. The subcommand follows the standard shape: a result struct with JSON tags, emission through the shared output writer so the envelope discriminant and `--fields` filtering come for free, and a `--schema` short-circuit for agent discovery. Genuine faults (an unreadable file, for instance) surface as agent-actionable error responses; the three staleness states are ordinary results. A developer reading the command sees the same anatomy as every other command family.

**Init grows one write, in the same place it already writes metadata.** The version-file write slots into the init command's existing post-install sequence, next to the config rewrite, using the version variable that already lives in the same package. Re-init overwrites the file unconditionally — recording is idempotent and "re-run init" is the entire repair story. The init command's existing plain-text output convention is preserved; the new write just adds one line of the same reporting.

**Skill templates change uniformly and minimally.** The same short preamble block is inserted at the top of all four workflow skill templates, after the frontmatter. The block is intentionally duplicated rather than extracted: the template pipeline has no partial mechanism, and introducing one for five lines would be a bigger change than the duplication it removes. The existing instruction-surface guard tests and the `{{command}}`-substitution tests automatically extend their coverage to the new text.

**Comparison logic is a pure function.** The match/mismatch/missing classification takes the recorded content and the current version as plain string inputs and is trivially unit-testable without touching the package-level version variable — sidestepping the dev-build version ambiguity in tests by construction.

## Dependencies

All dependencies already exist in the codebase; nothing must land or change before this plan starts.

- **`github.com/spf13/cobra` (existing, no changes)** — command registration for the new `version` group and `check` subcommand; already provides `--version` for humans via the root command's `Version` field.
- **`internal/output` (existing, no changes)** — the shared JSON envelope writer supplying the success discriminant, `--fields` filtering, and the `ErrorResponse` shape for genuine faults; the new command is purely an additional caller.
- **Build-time version injection (existing, no changes)** — the ldflags-set `version` variable in the `cmd` package is the single source of the current binary version for both recording at init and comparing at check time; the Makefile and Dagger ldflags paths must simply remain pointing at that package (fixed by the earlier "Fix version number" commit, PR #24).
- **Skill install pipeline (existing, no changes)** — `internal/agent`'s `installWorkflowSkills` and the `{{command}}` mustache substitution deliver the new preamble block to all three agents; only the four SKILL.md templates themselves change.
- **`internal/project` init scaffold (existing, touched only if the write lands there)** — the version-file write goes in the init command beside the config rewrite; the project scaffold itself needs no structural change.
- **`github.com/stretchr/testify` (existing, no changes)** — assertions for the new command and init tests, following the package's `require` convention.
- **Upstream spec** — `000039_version-check` (completed 2026-07-31) is the source of truth for scope; no other spec or plan must land first.

## Testing Approach

All testing is standard Go unit and command-level testing with testify's `require`, in isolated temporary directories per the package's existing convention — every test creates and chdirs into its own scratch directory and renders state through the production code paths, never against a real checkout.

**Unit tests: the classification function.** The match/mismatch/missing decision is a pure function over two strings, tested exhaustively with hand-written expected values: exact match, match with surrounding whitespace in the file content, differing versions, empty content, whitespace-only content. This is where the spec's edge-case requirement (missing/empty/unparseable recorded version is treated as stale, never an error) is pinned down.

**Command-level tests: `version check`.** Driven through the production root-command wrapper so the tests observe the real envelope and exit code. Load-bearing assertions: each of the three states produces exit 0, the success discriminant, the correct `status`, and — for the stale states — a non-empty `action` naming re-init; a match result carries no action text (the spec's "matching versions stay silent" contract); the check never creates or modifies the version file or any installed file (the spec's read-only constraint, asserted by comparing directory state before and after a mismatch invocation); genuine faults return the agent-actionable error shape. The new command family is also added to the existing envelope contract tables (stdout-only, discriminant present, printed exactly once), which is where cross-command output guarantees live.

**Init tests: recording and repair.** Extensions to the existing init test suite assert the version file exists with the binary's version after a fresh init, that re-running init overwrites a stale recorded version (the repair path — the spec's "re-initializing clears the mismatch"), and that the write is idempotent alongside init's existing idempotency contract. Because the recorded value comes from the same package-level variable the check compares against, tests inject known version values rather than deriving expectations from the variable itself, keeping the oracle independent of the subject.

**Template guard tests: the preamble.** The existing instruction-surface and `{{command}}`-substitution guard tests automatically cover the new preamble text; the only addition is an assertion that each installed skill contains the version-check instruction, so a future template edit cannot silently drop it.

**Success metrics mapping.** (1) "Users are informed of stale files on their next skill invocation" — behavioural at the CLI boundary: a mismatched recording yields `status: mismatch` plus the relay-ready action message, and the installed preamble instructs the agent to run the check first; whether a live agent actually relays the prompt is agent behaviour, flagged as **manual — captured in the implementation test plan**. (2) "No false alarms when installation matches" — behavioural: the match state asserts no action text and no version-related output beyond the status report. (3) "Reports of skills behaving oddly after upgrades drop to zero" — a field-observation metric with no automatable assertion: **manual — captured in the implementation test plan**.

**Deliberate gaps.** No end-to-end test drives a real agent through a skill invocation — the preamble is instruction text, and its effect on agent behaviour is not assertable in CI. No semver parsing or ordering tests exist because the design deliberately uses exact string equality.

## Milestones & Phases

### Milestone 1: Init records which binary installed the files

**What changes**: Running init in a repository now leaves behind a record of the exact binary version that performed the installation, stored in a dedicated version file inside the tool's dot-directory. Re-running init at any time refreshes the record to the current binary. On its own this milestone changes nothing about day-to-day use — no checks, no prompts — but every repository initialized from here on carries the information the rest of the feature needs, and re-init is already the complete repair mechanism.

**Validation point**: After running init in a fresh repository, the version file exists and contains the running binary's version; after re-running init with a different recorded value in place, the file holds the new binary's version; the existing init idempotency guarantees still hold.

#### - [x] Phase 1.1: Record the installing version at init

Teach init to write the dedicated version file containing the running binary's version, immediately alongside the configuration it already writes. Re-running init overwrites the file unconditionally, making re-init the complete repair path for a stale installation. Init's existing human-readable output gains one line reporting the recorded version.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-record-the-installing-version-at-init)

**Acceptance criteria**:

- [x] After init runs in a fresh repository, the version file exists inside the tool's dot-directory and contains exactly the running binary's version.
- [x] Re-running init over an existing installation replaces a stale recorded version with the current binary's version.
- [x] Init's existing idempotency guarantee still holds — running it twice leaves all sibling files untouched.
- [x] Init still succeeds and records correctly in a repository that predates the feature (no version file present).

### Milestone 2: A single cheap command answers "is this installation stale?"

**What changes**: A new `version check` command tells anyone — agent or human — whether the repository's installed files match the binary in use, in one call with structured output. It reports one of three states (match, mismatch, or missing record), includes both versions, and on a stale result carries a ready-made instruction to re-run init. It never modifies anything, and repositories initialized before this feature existed simply report as stale rather than erroring. At this point the staleness question is answerable on demand, but nothing asks it automatically yet.

**Validation point**: Invoking the command in repositories with a matching, differing, absent, and empty version file yields the correct state and versions in the standard JSON envelope with exit code 0, leaves every file untouched, and the stale states carry the re-run-init instruction while the match state carries no action text.

#### - [x] Phase 2.1: Add the version check command

Add the new command that performs the comparison so no skill ever has to: it reads the recorded version, compares it to the running binary's version, and reports match, mismatch, or missing through the standard JSON envelope with exit code 0. Stale results carry a ready-to-relay instruction to re-run init; the command itself never modifies anything. The comparison itself is a pure, exhaustively unit-tested function, and the command joins the project-wide output contract tests.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-add-the-version-check-command)

**Acceptance criteria**:

- [x] With a recorded version equal to the binary's, the command reports a match with both versions and no action text.
- [x] With a differing recorded version, the command reports a mismatch, both versions, and an instruction to re-run init.
- [x] With no version file, or an empty/whitespace-only one, the command reports the record as missing with the same re-run-init instruction, and does not error.
- [x] The command leaves the version file and all installed files byte-for-byte unchanged in every state.
- [x] The command's output follows the same JSON envelope rules as every other command family, including schema self-description and field filtering.

### Milestone 3: Every skill invocation checks freshness automatically

**What changes**: All four workflow skills, on every invocation and for every supported agent, now begin by running the version check. When the installation matches the binary, the user notices nothing — the skill proceeds exactly as before. When it doesn't, the agent tells the user their installed Spektacular files are out of date and asks them to re-run init before continuing. This is the milestone that delivers the feature's promise: nobody has to remember to re-initialize after upgrading, because the next skill use says so.

**Validation point**: Freshly installed skills for each agent contain the version-check preamble with the configured command substituted; the guard tests over the instruction surface pass; invoking a skill in a mismatched repository produces the stale prompt, and after re-running init the same skill invocation proceeds with no version-related output.

#### - [x] Phase 3.1: Add the version-check preamble to workflow skills

Insert the same short preamble at the top of all four workflow skill templates: run the version check first, continue silently on a match, and on a stale result tell the user their installed files are out of date and ask them to re-run init before proceeding. Because all supported agents install identical renders of these templates, one template change per skill reaches every agent. Finally, re-initialize this repository so its own installed skills carry the preamble and its version file exists.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-add-the-version-check-preamble-to-workflow-skills)

**Acceptance criteria**:

- [x] Every freshly installed workflow skill, for each supported agent, begins with the version-check instruction with the configured command correctly substituted.
- [x] The templates keep the command placeholder unrendered, and the whole instruction surface still passes the existing guard tests.
- [x] A guard exists so that removing the preamble from any skill template fails the test suite.
- [x] This repository's own installed skills contain the preamble and its version file records the dev binary's version, and invoking a skill here produces no version warning.

## Open Questions

No open questions. All design decisions were resolved during planning: the version file's name and location, the recorded value (bare version, no sha), exact-string comparison semantics, the three-state success-envelope contract, the committed-to-git choice, the per-template preamble insertion, and the descoping of the template-content-hash signal for dev builds.

One watch-point for the implementer (not a question): whether the existing envelope contract tables in the command test suite accept a two-word command family (`version check`) without harness changes — only discoverable when wiring the tests. If the harness needs more than a trivial table entry, STOP and ask before restructuring shared test code.

## Out of Scope

- **Detecting hand-edits to installed files** — the check compares recorded install version to binary version only, never file contents (spec Non-Goal). No drift detection for locally modified skills; re-running init silently overwrites them, as it does today.
- **Checking on every CLI command run** — the check fires only at skill invocation, driven by the skill preamble (spec Non-Goal). No hook in the binary's root command path.
- **Migration tooling for pre-feature repositories** — repositories initialized before this feature simply report `missing` and see the re-run-init prompt until re-initialized (spec Non-Goal); nothing backfills version files.
- **Template content hashing for dev builds** — descoped during planning (spec Technical Approach flagged it as optional). All `go run .` builds report the default `0.1.0`, so version comparison cannot detect template drift between dev builds; in practice dev repos match silently. If dev-time drift detection is ever wanted, it needs a new spec covering a scoped hash of the embedded template FS (which currently embeds test files too, via `go:embed all:*`).
- **Auto-refreshing installed files on mismatch** — expressly forbidden by the spec's constraints; the check is read-only and repair is always an explicit user-run init.
- **Semver-aware comparison (upgrade vs downgrade detection)** — the check is exact string equality; any difference is simply "mismatch". No ordering, no "newer than" messaging.
- **Fixing the pre-existing `templates/agents/memory-context.md` bug** — its "Managed by" line hardcodes `spektacular init` where sibling templates use `{{command}} init`; noticed during research, unrelated to this feature, left for a separate trivial fix.

## Changelog

### 2026-07-31 — Phase 1.1: Record the installing version at init

**What was done**: Init now writes `.spektacular/version` containing the running binary's bare version string (newline-terminated) immediately after the config rewrite, overwriting any previous record, and reports the recorded version with one extra plain-text output line. Helpers `versionFilePath` and `writeVersionFile` live in the new `cmd/version.go`, which Phase 2.1 will extend with the `version check` command.

**Deviations**: None.

**Files changed**:
- `cmd/version.go` (new)
- `cmd/init.go`
- `cmd/init_test.go`

**Discoveries**: Nothing unexpected — the codebase matched the plan's file:line references exactly. The pre-feature-repo acceptance criterion is covered by the fresh-init test (a fresh temp dir has no version file), so no separate test was added.

### 2026-07-31 — Phase 2.1: Add the version check command

**What was done**: Added the `version` command group with a `check` subcommand that reads `.spektacular/version`, classifies it against the running binary's version via the pure `classifyVersion` function, and reports `match`, `mismatch`, or `missing` through the standard JSON envelope with exit 0. Stale states carry an `action` string suggesting the configured init command (`<command> init <agent>`); an unreadable version file (non-not-exist error) returns the agent-actionable `version_file_unreadable` error with exit 1. The command supports `--schema` self-description and `--fields` filtering, and joined the three project-wide envelope contract tests.

**Deviations**: None.

**Files changed**:
- `cmd/version.go`
- `cmd/root.go`
- `cmd/version_test.go` (new)
- `cmd/root_test.go`

**Discoveries**: The plan's watch-point resolved cleanly — the envelope contract tests are plain `t.Run` subtests, so the two-word `version check` family needed only new subtests, no harness change. The `--schema` flag lives on the package-level command and persists across in-process invocations, so `cmd/version_test.go` adds `resetVersionCheckFlags` (mirroring `resetSpecCommandFlags`) to prevent flag-state leakage. A version file that is a directory is the convenient test fixture for the genuine-fault path.

### 2026-07-31 — Phase 3.1: Add the version-check preamble to workflow skills

**What was done**: Inserted an identical "Version check first" blockquote immediately after the frontmatter of all four workflow skill templates, instructing the agent to run `{{command}} version check` before anything else, stay silent on a match, and on mismatch/missing relay the `action` message and wait for the user to re-run init. Re-initialized this repository so its own installed skills carry the preamble and `.spektacular/version` records the dev binary's `0.1.0` (version check here reports a silent match).

**Deviations**: None.

**Files changed**:
- `templates/skills/workflows/spek-new/SKILL.md`
- `templates/skills/workflows/spek-plan/SKILL.md`
- `templates/skills/workflows/spek-implement/SKILL.md`
- `templates/skills/workflows/spek-knowledge/SKILL.md`
- `internal/agent/claude_test.go`
- `internal/agent/bob_test.go`
- `internal/agent/codex_test.go`
- `cmd/init_test.go`
- `.claude/skills/*/SKILL.md`, `AGENTS.md`, `.spektacular/version` (dogfood re-init renders)

**Discoveries**: The per-agent install tests' existing `skillAssertions` loops cover all four skills for all three agents, so a single containment assertion per loop guards the preamble in 12 rendered files — no separate template-walking test was needed. `TestInit_CustomCommand` doubles as proof the preamble uses the `{{command}}` placeholder rather than a hardcoded command.
