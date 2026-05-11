# Plan: spec-id-prefix-method

<!-- Metadata -->
<!-- Created: 2026-05-08T17:49:29Z -->
<!-- Commit: a3108b2 -->
<!-- Branch: main -->
<!-- Repository: https://github.com/gregoryhunt/spektacular.git -->
<!-- Feature Slug: 0020-spec-id-prefix-method -->

## Overview

Specification creation will produce prefixed, chronologically sortable names by default while still allowing teams to choose counter-based names or accept identifiers from external systems. This makes new specs easier to organize and safer to automate without changing the authoring workflow users already follow.

## Architecture & Design Decisions

The chosen design resolves the final specification name before the spec workflow is created. The creation request still starts with a human-readable name, but `spec new` converts that request into the canonical workflow name by applying an explicit identifier, a timestamp prefix, or the next counter prefix. The rest of the workflow continues to operate on one name, so the returned `spec_name`, saved state, status output, scaffold title, and file path all agree.

Identifier generation is owned by a small resolver with three supported methods: `timestamp`, `counter`, and `external`. The default is `timestamp`, explicit `id` input overrides generated methods, and `external` mode requires that the caller provide `id`. Timestamp prefixes use UTC and lexicographic formatting with second-bump collision handling; counter prefixes use a zero-padded numeric prefix backed by a persisted project counter.

Configuration stays additive through a nested spec config block, so existing config files remain valid and new projects expose the default method and counter value. The command schema is expanded to document optional `id` input for agents and external callers. Rejected options are captured in [research.md#alternatives-considered-and-rejected](./research.md#alternatives-considered-and-rejected), especially the alternatives of doing this in skills or mutating only the file path inside the workflow.

## Component Breakdown

- **Spec identifier resolver** owns method validation, normalization, prefix generation, collision checks, explicit identifier validation, and final canonical name composition. It is the only component that needs to know how timestamp, counter, and external identifier modes differ.
- **Project configuration** owns the optional default method and current counter value for spec identifiers. It provides defaults when fields are omitted and persists counter progress for counter mode.
- **Spec creation command adapter** owns JSON input parsing, schema reporting, loading configuration, invoking the resolver, and starting the workflow with the canonical name.
- **Existing spec workflow** continues to own scaffold rendering, step instructions, state persistence, and final spec file writes. It receives the canonical name and does not branch on identifier method.
- **Guidance and tests** keep installed skills, docs, and regression coverage aligned with the new creation contract so agents know to trust the returned spec path and spec name.

## Data Structures & Interfaces

The config contract gains a nested spec block and the creation input gains an optional `id` value:

```go
type Config struct {
    Command string      `yaml:"command"`
    Agent   string      `yaml:"agent"`
    Debug   DebugConfig `yaml:"debug"`
    Spec    struct {
        IDMethod string `yaml:"id_method"`
        Counter  int    `yaml:"counter"`
    } `yaml:"spec"`
}
```

The resolver contract accepts the requested name, optional id, configured method, store access for counter mode, and a clock for timestamp mode. It returns the canonical workflow name or a validation error:

```go
request := struct {
    Name   string
    ID     string
    Method string
    Counter int
    Store  store.Store
    Now    func() time.Time
}{}

result := struct {
    Name    string
    Counter int
}{}
```

Supported method values are `timestamp`, `counter`, and `external`. An empty method is equivalent to `timestamp`. Names and identifiers are normalized by lowercasing and replacing accepted separator characters with `-`; leading or trailing whitespace is a validation error, not something the system trims.

## Implementation Detail

The implementation introduces one focused naming boundary before workflow state is created. Command parsing validates the requested suffix name, loads configuration, creates the project store, resolves the canonical spec name, and only then removes or writes workflow state. This ordering ensures configuration errors, invalid identifiers, or missing external identifiers fail without creating a spec file or clobbering active state.

The resolver treats explicit `id` input as a universal override after normalization. Generated methods only run when no id is provided; `external` mode is the opposite contract and errors unless id is provided. Input validation rejects empty values, leading or trailing whitespace, path separators, and control characters, then normalizes accepted values by lowercasing and replacing separators such as `.`, `@`, and internal whitespace with `-`.

Counter generation reads the current project counter from config, increments it, formats it with six digits, and persists the new current value as part of successful counter allocation. Timestamp generation uses UTC seconds in `YYYYMMDDHHMMSS` form so alphabetical order matches creation order; if a timestamp-created filename already exists, the resolver adds one second and checks again until it finds an unused filename. Existing file write behavior remains owned by the workflow and store, but the command must not allow generated or explicit names to overwrite an existing spec.

## Dependencies

- **Cobra command handling** provides the existing subcommand, flag, and schema surfaces; it needs a narrow input/schema update for optional id.
- **YAML config loading** provides default-backed config parsing; it needs one nested config block and tests for omission/round-trip behavior.
- **Project file store** provides path traversal protection and existence checks; it is reused for collision detection without API changes.
- **Spec workflow steps** provide scaffold writing and status/path output; they should continue to receive one canonical name.
- **Embedded skill templates and README docs** provide user and agent guidance; they need wording updates so generated names and optional ids are visible.
- **Standard library time and parsing utilities** are sufficient for timestamp formatting, counter parsing, and validation; no new third-party dependency is required.

## Testing Approach

Testing concentrates on the resolver and command boundary because those are the places where behavior changes. Resolver unit tests should use deterministic clocks and temp stores to prove method defaults, timestamp formatting, timestamp collision bumping, counter incrementing from config, explicit id normalization, external mode requirements, and invalid identifiers.

Command-level tests should exercise `spec new` as an external caller sees it: JSON input, schema output, file creation, returned path/name, dry-run behavior, and failure without side effects. Config tests should prove old config files remain valid and new defaults round-trip.

Existing workflow tests remain valuable regression coverage. They should continue to pass with canonical names because the workflow should not need to understand how those names were chosen.

## Milestones & Phases

### Milestone 1: Identifier Rules Are Defined

**What changes**: The project gains a clear, testable contract for how spec identifiers are configured, generated, and validated. This milestone does not change the user-facing creation flow yet; it creates the safe foundation that later phases wire into creation.

#### - [x] Phase 1.1: Configurable ID Methods

Add the optional spec identifier method and current counter value to configuration with timestamp and zero as defaults. Existing config files continue to load without edits, while newly written configs expose the default method and counter clearly. This phase establishes the persistent project preference and counter state before any creation behavior depends on them.

*Technical detail:* [context.md#phase-11-configurable-id-methods](./context.md#phase-11-configurable-id-methods)

**Acceptance criteria**:

- [x] Projects without a spec identifier setting still load with timestamp behavior selected.
- [x] New or round-tripped config output includes the spec identifier method and current counter without losing existing command, agent, or debug settings.
- [x] Invalid or unknown method values are rejected when spec creation tries to use them.

#### - [x] Phase 1.2: Identifier Resolution Rules

Add the resolver that turns a requested spec name into the canonical spec name. It normalizes accepted names and ids without trimming, covers timestamp, counter, and external identifier methods, treats a passed id as an override for generated methods, and returns any updated counter value. This phase makes the naming behavior deterministic and isolated before command wiring.

*Technical detail:* [context.md#phase-12-identifier-resolution-rules](./context.md#phase-12-identifier-resolution-rules)

**Acceptance criteria**:

- [x] Timestamp resolution produces a sortable timestamp prefix followed by the requested name.
- [x] Timestamp resolution bumps by seconds when the initial timestamp target already exists.
- [x] Counter resolution uses the next persisted counter value and returns the updated current counter.
- [x] Passed identifiers are normalized, accepted for generated methods, and required for external mode.
- [x] Unsafe or untrimmed name and identifier values are rejected before any file path is built.

### Milestone 2: Spec Creation Uses Canonical Names

**What changes**: Users and external systems get the new naming behavior through the native spec creation command. The returned spec path, active workflow state, and created file all use the same canonical name.

#### - [x] Phase 2.1: Native Spec Creation Wiring

Wire the resolver into spec creation before workflow state is created. The command accepts optional id input, applies the configured method, persists counter changes when counter mode allocates a value, and starts the existing workflow with the resolved canonical name. Failure cases stop before any spec file or workflow state is written.

*Technical detail:* [context.md#phase-21-native-spec-creation-wiring](./context.md#phase-21-native-spec-creation-wiring)

**Acceptance criteria**:

- [x] Creating a spec without id or config uses a timestamp-prefixed filename.
- [x] Creating a spec with id uses the normalized id as the prefix regardless of generated method configuration.
- [x] External mode without id fails clearly and leaves no new spec file behind.
- [x] Counter mode advances the persisted current counter when creation succeeds.
- [x] Dry-run creation reports the same canonical name shape without writing files.

#### - [x] Phase 2.2: Public Command Contract Coverage

Add command-level coverage for the behavior external callers rely on. These tests verify the JSON input contract, schema visibility, returned metadata, and filesystem outcomes. This phase catches regressions that pure resolver tests cannot see.

*Technical detail:* [context.md#phase-22-public-command-contract-coverage](./context.md#phase-22-public-command-contract-coverage)

**Acceptance criteria**:

- [x] The creation schema documents both `name` and optional `id`.
- [x] Timestamp, timestamp collision, counter, explicit id normalization, and external-mode cases are observable through command output.
- [x] Validation failures do not leave behind a created spec or new workflow state.

### Milestone 3: Guidance And Regression Safety Are Updated

**What changes**: Agents, users, and future maintainers can see and trust the new naming behavior. Documentation and generated guidance explain that callers should use the spec name and path returned by the command.

#### - [x] Phase 3.1: Generated Guidance And Documentation

Update the generated and tracked spec workflow guidance plus user-facing docs to describe prefixed names, normalized optional id input, and the config method. This keeps installed agent instructions aligned with the binary-owned behavior without requiring wrapper logic.

*Technical detail:* [context.md#phase-31-generated-guidance-and-documentation](./context.md#phase-31-generated-guidance-and-documentation)

**Acceptance criteria**:

- [x] Generated spec workflow guidance no longer assumes the requested name is always the final filename.
- [x] The tracked local `spek-new` skill is updated alongside the template so repository agents see the new behavior immediately.
- [x] User-facing documentation shows the config method values and optional id input.
- [x] Documentation tells callers to use returned spec metadata for follow-up workflows.

#### - [x] Phase 3.2: Final Regression Pass

Run the focused package tests and the full test suite after all changes are integrated. This phase verifies the new behavior and checks that existing spec, plan, init, config, and store workflows still operate normally.

*Technical detail:* [context.md#phase-32-final-regression-pass](./context.md#phase-32-final-regression-pass)

**Acceptance criteria**:

- [x] All resolver, config, command, and existing workflow tests pass.
- [x] A manual spec creation produces a prefixed spec path and a usable next workflow state using the returned normalized `spec_name`.
- [x] No existing plan or implementation artifact naming behavior changes.

## Open Questions

None.

## Out of Scope

- Renaming or migrating existing specification files is out of scope.
- Building a direct integration with any specific external system is out of scope.
- Changing the contents or step order of the specification authoring workflow is out of scope.
- Applying the same identifier method to plans or implementation artifacts is out of scope.
- Reformatting legacy unpadded or underscore-separated numeric spec filenames is out of scope.
- Guaranteeing gapless counter values is out of scope; failed filesystem writes after counter allocation may leave skipped numbers.

## Changelog

### FINAL SUMMARY

Implemented native spec identifier prefixes across configuration, resolver logic, `spec new`, command tests, generated guidance, and user-facing docs. New spec creation now defaults to timestamp prefixes, supports persisted counter prefixes, accepts optional external ids, and requires ids for external mode while returning canonical `spec_name` and `spec_path` for follow-up workflows.

**Total phases**: 6/6 completed

**Notable deviations from the plan**: None

### 2026-05-09 - Phase 1.1: Configurable ID Methods

**What was done**: Added `spec.id_method` and `spec.counter` to project configuration with `timestamp` and `0` defaults. Config loading now validates the configured spec ID method, and init/spec command tests cover old config compatibility, round-tripped output, generated config defaults, and `spec new` rejection for unsupported methods.

**Deviations**: None

**Files changed**:
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/project/init_test.go`
- `cmd/init_test.go`
- `cmd/spec_test.go`

**Discoveries**: Method validation belongs at config load time for this phase because every command that starts a workflow already loads config first; later resolver wiring can reuse the exported config method constants instead of duplicating strings.

### 2026-05-09 - Phase 1.2: Identifier Resolution Rules

**What was done**: Added a spec identifier resolver that normalizes names and optional ids, supports timestamp, counter, and external modes, checks collisions through the project store, and bumps timestamp seconds or counter values until generated names are unused. The resolver returns the canonical spec name plus the current counter value that command wiring can persist later.

**Deviations**: None

**Files changed**:
- `internal/steps/spec/identifier.go`
- `internal/steps/spec/identifier_test.go`

**Discoveries**: Explicit id overrides intentionally leave the returned counter unchanged, so Phase 2.1 can persist `spec.counter` only when counter allocation actually advances it.

### 2026-05-09 - Phase 2.1: Native Spec Creation Wiring

**What was done**: Wired `spec new` through the identifier resolver so new specs use canonical timestamp, counter, or external-id names before workflow state is created. Counter mode now persists the advanced `spec.counter` on successful non-dry-run creation, while dry-run resolves and reports the same canonical name without writing specs, state, or counter updates.

**Deviations**: None

**Files changed**:
- `cmd/root.go`
- `cmd/spec.go`
- `cmd/spec_test.go`
- `internal/steps/spec/identifier.go`

**Discoveries**: Auxiliary `--stdin` and `--file` workflow data must not be allowed to override the resolved `name`; the command now applies extra workflow data first and then restores the canonical name before starting the workflow.

### 2026-05-09 - Phase 2.2: Public Command Contract Coverage

**What was done**: Expanded command-level tests to cover `spec new --schema`, timestamp defaults, timestamp collision bumping, counter and stale-counter behavior, explicit id normalization, external mode success and failure, dry-run output, and validation failures without spec/state side effects. Added a small injectable clock for `spec new` so timestamp collision behavior is deterministic in tests.

**Deviations**: None

**Files changed**:
- `cmd/spec.go`
- `cmd/spec_test.go`

**Discoveries**: Cobra flag values can persist between same-process command tests, so the spec command test helper now resets `schema`, `dry-run`, and input flags around each invocation.

### 2026-05-09 - Phase 3.1: Generated Guidance And Documentation

**What was done**: Updated generated and tracked `spek-new` guidance so agents use returned `spec_name` and `spec_path` instead of assuming the requested name is final. README now documents optional external ids, returned metadata usage, and the `spec.id_method` values; the repo changelog includes the user-facing behavior change.

**Deviations**: None

**Files changed**:
- `templates/skills/workflows/spek-new/SKILL.md`
- `.agents/skills/spek-new/SKILL.md`
- `README.md`
- `CHANGELOG.md`

**Discoveries**: The README command examples were still using older shorthand commands, so the usage section was updated to the current workflow command shape while documenting the new identifier behavior.

### 2026-05-09 - Phase 3.2: Final Regression Pass

**What was done**: Ran focused regression packages, the full Go test suite, and a manual smoke path in a temp project. The smoke created a timestamp-prefixed spec and then started `plan new` with the returned normalized `spec_name`.

**Deviations**: None

**Files changed**:
- `.spektacular/plans/spec-id-prefix-method/plan.md`

**Discoveries**: `go run` cannot target this module from outside the module directory in the local environment, so the manual smoke used a temporary built binary instead.
