# Context: spec-id-prefix-method

## Current State Analysis

- `cmd/spec.go:19` defines `nameRegexp` as lowercase alphanumeric plus hyphen/underscore. It currently validates only the requested spec name, not an external identifier.
- `cmd/spec.go:136-197` owns `spec new`: it reports schema, parses `--data`, validates `name`, loads config, removes the shared state file for non-dry-run creation, creates a workflow with `store.NewFileStore(dataDir)`, and sets workflow data key `name` to the parsed input name.
- `cmd/spec.go:137-148` reports a schema with only `name` as input, so external callers cannot discover an `id` field yet.
- `cmd/spec.go:176-187` removes state before workflow creation and then sets `name` directly from input. New validation must happen before state removal so validation failures do not clobber active state.
- `cmd/spec.go:259-292` builds status paths from the workflow data `name`, so putting the canonical name in state makes status output naturally follow the new naming behavior.
- `internal/steps/spec/steps.go:11-14` builds the store-relative spec path as `specs/<name>.md`.
- `internal/steps/spec/steps.go:63-80` writes the initial scaffold for `new()` using the workflow data `name`; it should not need to know which identifier method was used.
- `internal/steps/spec/strategy.go:14-18` reports `spec_path` and `spec_name` from the workflow name. This supports resolving the final name before workflow start.
- `internal/config/config.go:18-32` defines the top-level config and defaults. `FromYAMLFile` unmarshals into `NewDefault()`, so missing fields can inherit defaults.
- `internal/config/config_test.go:11-49` covers default values, env expansion, missing-file errors, and round-trip behavior.
- `cmd/root.go:30-42` loads `.spektacular/config.yaml`, returning defaults when the file is absent and errors when it is invalid.
- `internal/project/init.go:38-44` writes default config only if it does not exist. Updating `NewDefault()` updates newly initialized projects.
- `cmd/init.go:36-44` loads, mutates, and writes config during `init`, so new fields must survive re-init paths.
- `internal/store/store.go:48-55` rejects paths that escape the store root; identifier validation should still reject unsafe characters before path construction.
- `internal/store/store.go:112-119` checks path existence, which is enough for timestamp, counter, and explicit-id collision detection.
- `internal/steps/spec/steps_test.go:112-122` verifies that the spec `new()` step writes a scaffold at `SpecFilePath(name)`.
- `cmd/implement_test.go:38-51` shows the command-test pattern for resetting `rootCmd` output, error, and args between invocations.
- `templates/skills/workflows/spek-new/SKILL.md:8-31` says the workflow writes `.spektacular/specs/<name>.md`; this needs to change because the returned `spec_name` can now include a generated or supplied prefix.
- `.agents/skills/spek-new/SKILL.md:8-31` is tracked in this repository and currently has the same stale assumption as the template, so it should be updated directly as part of this plan.
- `README.md:130-145` has a config section that is already stale relative to the current config struct, but it is the user-facing place to document the new spec id method.

## Per-Phase Technical Notes

### Phase 1.1: Configurable ID Methods

- `internal/config/config.go:13-23` - add a new nested spec config type with `id_method` and `counter` YAML fields, then add that field to `Config`.
- `internal/config/config.go:25-33` - set `NewDefault().Spec.IDMethod` to `timestamp` and `NewDefault().Spec.Counter` to `0`.
- `internal/config/config.go:35-49` - no loader rewrite is expected; because loading unmarshals into defaults, existing config files should inherit `timestamp` when `spec.id_method` is omitted. Add tests to prove this rather than changing the loader shape.
- `internal/config/config_test.go:11-16` - extend default assertions to include `cfg.Spec.IDMethod == "timestamp"`.
- `internal/config/config_test.go:18-30` - add a test loading YAML with only `command` to prove missing `spec` config defaults to timestamp.
- `internal/config/config_test.go:37-49` - extend round-trip assertions to include `Spec.IDMethod` and `Spec.Counter`.
- `internal/project/init.go:38-44` - no direct edit should be needed beyond the config default, but verify the generated config file includes `spec.id_method`.
- `cmd/init.go:36-44` and `cmd/init_test.go:147-168` - verify re-init with a config file that lacks `spec` still writes a valid config and preserves custom command behavior.

**Complexity**: Low
**Token estimate**: ~6k
**Agent strategy**: Single agent, sequential execution.

### Phase 1.2: Identifier Resolution Rules

- `internal/steps/spec/steps.go:11-14` - keep `SpecFilePath(name)` unchanged and treat `name` as already canonical.
- `internal/steps/spec/` - create a new resolver source file in the spec steps package so `cmd/spec.go` can call it through the existing spec import. Suggested contract shape:

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

- `internal/steps/spec/` - in the new resolver source file, define method constants for `timestamp`, `counter`, and `external`; treat an empty method as timestamp.
- `internal/steps/spec/` - add a normalization/validation helper shared by requested names and explicit IDs. It must reject empty raw values, leading or trailing whitespace (`raw != strings.TrimSpace(raw)`), path separators, control characters, and values that normalize to empty. It must lower-case ASCII input, preserve alphanumeric characters and underscores, and replace accepted separator/special characters such as `.`, `@`, `-`, and internal whitespace runs with a single `-`.
- `internal/steps/spec/` - compose final names with a hyphen separator: `<prefix>-<name>`.
- `internal/steps/spec/` - timestamp mode should use `req.Now().UTC().Format("20060102150405")`; default `Now` to `time.Now` if omitted. Before returning, check whether `SpecFilePath(candidate)` exists; if it does, add one second and retry until the candidate is unused.
- `internal/steps/spec/` - counter mode should start from `req.Counter + 1`, format as six digits, and check for collisions. If the target exists because the stored counter is stale, keep incrementing until an unused target is found and return that value as the new current counter.
- `internal/steps/spec/` - explicit id mode should also check the target path and fail on collision rather than overwrite an existing spec.
- `internal/store/store.go:112-119` - use `Exists(SpecFilePath(candidate))` for timestamp, counter, and explicit-id collision detection.
- `internal/steps/spec/` - create a new resolver test file with deterministic tests for default timestamp, timestamp collision bumping, explicit timestamp method, counter from configured value, stale counter collision bumping, explicit id override with normalization, external missing id, external with normalized id, unknown method, leading/trailing whitespace rejection, slash/control-character rejection, and nil store in generated modes.

**Complexity**: Medium
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential execution; tests should be written alongside the resolver because the behavior is compact and state-free.

### Phase 2.1: Native Spec Creation Wiring

- `cmd/spec.go:21-39` - `schemaProp` already supports type, enum, pattern, and maxLength. Reuse it to describe optional `id` in the `spec new --schema` input.
- `cmd/spec.go:136-148` - update schema properties to include `id` with a filename-safe pattern and max length. Keep `Required` as only `name`.
- `cmd/spec.go:151-159` - extend the anonymous input struct with `ID string `json:"id"``.
- `cmd/spec.go:163-165` - replace the direct requested-name regex check with the shared normalization/validation helper so names like `Team.Alpha` normalize to `team-alpha` while names with leading/trailing whitespace fail.
- `cmd/root.go:30-42` - add a helper that returns the config path, or otherwise expose the path to `runSpecNew`, because counter mode must write the updated `spec.counter` value back to `.spektacular/config.yaml`.
- `cmd/spec.go:167-186` - create the store before workflow creation, resolve the canonical name with config and input id, persist any counter update when not dry-run, then remove state only after resolution and counter persistence succeed.
- `cmd/spec.go:183-187` - pass the same store instance to `workflow.New` and set workflow data `name` to the resolved canonical name, not the requested suffix.
- `cmd/spec.go:193-195` - keep `output.WriteError` behavior for workflow errors. Resolver/config validation errors can return directly, matching current validation style.
- `internal/steps/spec/steps.go:71-77` - no method branching should be added here. The step writes the scaffold using the canonical name it receives.
- `internal/steps/spec/strategy.go:14-18` - no direct change expected; verify returned `spec_path` and `spec_name` reflect the canonical name.
- `cmd/spec.go:176-180` - preserve dry-run behavior. Dry-run should resolve and report the canonical name but avoid removing state, writing a spec file, or updating the persisted counter.

**Complexity**: Medium
**Token estimate**: ~10k
**Agent strategy**: Single agent, sequential execution because command state ordering and resolver use are tightly coupled.

### Phase 2.2: Public Command Contract Coverage

- `cmd/` - create a new spec command test file rather than overloading `internal/steps/spec` tests. Mirror the `setupImplementCmd` pattern from `cmd/implement_test.go:38-51`.
- `cmd/` - add helper setup that creates `.spektacular/`, writes optional config YAML, executes `rootCmd`, and decodes JSON output.
- `cmd/` - test default creation creates exactly one spec file whose basename matches `^\d{14}-billing-export\.md$`, returns `step == "overview"`, and returns matching normalized `spec_name` and `spec_path`.
- `cmd/` - test timestamp collision by pre-creating the first timestamp target and asserting creation bumps to the next second.
- `cmd/` - test counter mode with config `spec.id_method: counter` and `spec.counter: 7` produces `000008-billing-export.md` and persists `spec.counter: 8`.
- `cmd/` - test stale counter collision by pre-creating `000008-billing-export.md` and asserting the command creates `000009-billing-export.md` and persists `9`.
- `cmd/` - test explicit `id` input like `EXT.User@123` creates `ext-user-123-billing-export.md` even when config is timestamp or counter.
- `cmd/` - test external mode without id returns an error and leaves both the specs directory and shared state file absent or unchanged.
- `cmd/` - test external mode with id succeeds.
- `cmd/` - test `spec new --schema` contains `"id"` and keeps `"name"` required.
- `cmd/` - test unsafe and untrimmed name/id values fail before file creation.
- `internal/steps/spec/steps_test.go:112-122` - keep the existing scaffold test as-is or update only if helper names move; it should continue to prove `new()` writes `SpecFilePath(name)`.

**Complexity**: Medium
**Token estimate**: ~14k
**Agent strategy**: Single agent. Command tests share global `rootCmd`; keep them sequential and isolated with cleanup.

### Phase 3.1: Generated Guidance And Documentation

- `templates/skills/workflows/spek-new/SKILL.md:8` - update wording so it says the workflow produces the spec file returned by `spec_path`, not always `.spektacular/specs/<name>.md`.
- `templates/skills/workflows/spek-new/SKILL.md:25-31` - add a short note that external callers may pass `id` in the JSON payload and that agents should use the returned `spec_name` for later plan workflows.
- `.agents/skills/spek-new/SKILL.md:8-31` - update the tracked local skill alongside the template so Codex in this repository sees the new behavior immediately.
- `README.md:130-145` - replace or augment the stale config example with a minimal current config example that includes `spec.id_method: timestamp` and documents valid values `timestamp`, `counter`, and `external`.
- `README.md` near spec creation usage - document `spec new --data '{"name":"billing-export","id":"EXT-123"}'` as the external-id override example if the surrounding command reference exists.
- `CHANGELOG.md` - if implementation conventions require changelog updates for user-facing CLI behavior, add a concise entry describing timestamp default prefixes and optional external IDs.

**Complexity**: Low
**Token estimate**: ~7k
**Agent strategy**: Single agent, sequential execution.

### Phase 3.2: Final Regression Pass

- `internal/config/config_test.go` - run the config package tests after config changes.
- `internal/steps/spec/` package tests and `internal/steps/spec/steps_test.go` - run the spec package tests after resolver and workflow checks.
- `cmd/` package tests - run command tests after command wiring and schema changes.
- `cmd/init_test.go` and `internal/project/init_test.go` - run init tests after config defaults and docs/guidance changes.
- Full repository test suite - run once after focused tests pass.
- Manual smoke path - from a temp project or with dry-run where appropriate, verify a spec creation returns a prefixed `spec_name` and that the next plan workflow can target that returned name.

**Complexity**: Low
**Token estimate**: ~5k
**Agent strategy**: Single agent. Parallel test execution is handled by Go; no multi-agent split is needed.

## Testing Strategy

Use three layers of coverage. First, pure resolver tests should prove naming rules with no Cobra or workflow involvement. Second, command tests should prove external behavior through JSON input, schema output, filesystem writes, and state side effects. Third, existing workflow, config, init, and store tests should serve as regression coverage so the feature does not accidentally change unrelated workflow behavior.

The most important cases are timestamp default behavior, timestamp collision bumping, explicit id normalization, counter incrementing and persistence, stale counter collision handling, external mode requiring id, unsafe or untrimmed input rejection, existing config compatibility, and schema discoverability.

## Project References

- Specification: `.spektacular/specs/spec-id-prefix-method.md`
- Plan namespace from workflow: `spec-id-prefix-method`
- Derived feature slug from existing plans: `0020-spec-id-prefix-method`
- Metadata: created `2026-05-08T17:49:29Z`, commit `a3108b2`, branch `main`, repository `https://github.com/gregoryhunt/spektacular.git`

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | Single agent with focused file reads; optional split only between docs and tests |
| High | ~50k+ | Not expected for this feature |

## Migration Notes

Existing config files remain valid because the new fields are optional and default to timestamp plus counter zero. Existing specification files are not renamed. The default behavior for new spec creation changes from unprefixed names to timestamp-prefixed names, and accepted names/ids are normalized, so callers that previously assumed the requested name was the final filename must use the returned `spec_name` or `spec_path`.

## Performance Considerations

Timestamp and explicit-id modes are constant-time unless they encounter a collision, in which case they perform repeated existence checks until an unused target is found or the explicit-id collision fails. Counter mode uses the persisted counter and only performs existence checks while avoiding stale-counter collisions, so it no longer needs to scan all spec filenames. If a project accumulates enough collisions to make repeated checks expensive, that indicates the counter or timestamp source is stale and should be investigated separately.
