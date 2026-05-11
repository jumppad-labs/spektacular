# Research: spec-id-prefix-method

## Alternatives considered and rejected

### Option A: Implement prefixes only in installed skills or command wrappers

This would update `spek-new` instructions or agent-specific wrappers to prepend timestamps, counters, or external ids before invoking the binary.

**Rejected**: the specification requires native binary behavior, and wrappers are not the canonical execution surface. Current generated skill guidance invokes the binary with a JSON payload (`templates/skills/workflows/spek-new/SKILL.md:25-31`), while init installs multiple agent surfaces (`cmd/init.go:47-50`). Putting naming rules in wrappers would duplicate behavior across agents and leave external callers of the binary without the feature.

### Option B: Prefix only the file path inside the spec workflow step

This would keep workflow data `name` as the user-supplied name and make the spec `new()` step write a different prefixed path.

**Rejected**: the current workflow uses the same data name for path construction and returned metadata. `SpecFilePath(name)` builds `specs/<name>.md` (`internal/steps/spec/steps.go:11-14`), `new()` writes that path (`internal/steps/spec/steps.go:71-77`), strategy output reports `spec_name` and `spec_path` from the same name (`internal/steps/spec/strategy.go:14-18`), and status reconstructs the path from workflow data (`cmd/spec.go:274-286`). Splitting path identity from workflow identity would make follow-up workflows and status output ambiguous.

### Option C: Add separate command flags for identifier behavior

This would add flags such as `--id` or `--method` beside the existing JSON payload rather than extending `--data`.

**Rejected**: the current command contract is JSON input plus schema introspection. `spec new` reads a JSON object from `--data` (`cmd/spec.go:151-162`) and reports its JSON schema (`cmd/spec.go:136-148`). The local CLI architecture note recommends raw JSON payloads, runtime schema introspection, and strong input validation for agent-facing commands (`.spektacular/knowledge/architecture/cli-design-for-ai-agents.md:11-29`). Adding parallel flags would make the command harder for agents and external systems to discover and validate.

### Option D: Store the counter in workflow state

This would track the current counter in the existing workflow state file.

**Rejected**: the current project uses a shared workflow state file that is truncated on each workflow start (`cmd/spec.go:176-180`, `.spektacular/plans/15_implementation/context.md:11-13`). Putting a project-level counter there risks losing the counter whenever another workflow starts. The chosen persistent location is config, where the identifier method already lives.

### Option E: Derive counter values only from existing filenames

This would scan existing spec filenames and select the next number without storing counter state.

**Rejected**: after timestamp prefixes become the default, filename scanning risks confusing timestamp prefixes with counter prefixes unless the parser becomes more complex. It also makes counter behavior depend on historical files rather than the user's configured current counter. A persisted `spec.counter` value gives counter mode an explicit source of truth while still allowing collision checks to avoid overwrites.

## Chosen approach — evidence

- Resolving the canonical name before workflow creation matches existing workflow design because all downstream path and status output already flows from the workflow data `name` (`cmd/spec.go:183-187`, `cmd/spec.go:274-286`, `internal/steps/spec/strategy.go:14-18`).
- The spec workflow `new()` step already writes a scaffold using `SpecFilePath(name)` (`internal/steps/spec/steps.go:63-80`), so passing a canonical name lets this behavior remain simple.
- Config loading already unmarshals into `NewDefault()` (`internal/config/config.go:35-49`), which supports additive optional config fields without breaking older config files.
- Project init writes the default config from `config.NewDefault()` (`internal/project/init.go:38-44`), so adding the default method and counter there also makes them visible to new projects.
- The file store already rejects path traversal (`internal/store/store.go:48-55`) and can check candidate paths for collision through `Exists` (`internal/store/store.go:112-119`).
- The command schema types already support pattern, enum, and max length fields (`cmd/spec.go:21-39`), enough to expose the optional id input without schema machinery changes.
- `plan new` and `implement new` validate workflow names with the shared `nameRegexp` shape (`cmd/plan.go:89-97`, `cmd/implement.go:93-100`), so spec-created names must be normalized into that compatible shape rather than preserving uppercase or special characters.

## Files examined

- `.spektacular/specs/spec-id-prefix-method.md:1` - source specification for timestamp default, counter mode, explicit id override, config method, and non-goals.
- `.spektacular/knowledge/architecture/cli-design-for-ai-agents.md:11` - local architecture note favoring JSON payloads and runtime schema introspection for agent-facing CLIs.
- `.spektacular/knowledge/conventions.md:11` - local conventions call for tests on new functionality and README updates for user-facing changes.
- `.spektacular/plans/13_agent-cli-improvements/plan.md:34` - prior plan established schema introspection and JSON `--data` as command design.
- `.spektacular/plans/14_spektacular-store/plan.md:220` - prior plan describes store-backed workflow writes and validates spec file creation through the store.
- `.spektacular/plans/15_implementation/context.md:11` - prior context confirms spec and plan workflows share `.spektacular/state.json`.
- `cmd/spec.go:19` - existing requested-name regex is lowercase alphanumeric, hyphen, and underscore.
- `cmd/spec.go:136` - `runSpecNew` owns schema, input parsing, config loading, state removal, workflow creation, and initial workflow data.
- `cmd/spec.go:259` - status output derives spec path from the workflow data name.
- `cmd/plan.go:89` - plan creation currently accepts only lowercase workflow names, so spec names returned from this feature must be normalized for follow-up workflows.
- `cmd/implement.go:93` - implement creation uses the same workflow-name validation shape as plan creation.
- `cmd/root.go:30` - config loading returns defaults when `.spektacular/config.yaml` is absent.
- `cmd/init.go:36` - init loads, mutates, and writes config, so new config fields must survive re-init.
- `cmd/implement_test.go:38` - command tests reset root command state with a helper pattern that can be reused for spec command tests.
- `internal/config/config.go:18` - top-level config currently has `Command`, `Agent`, and `Debug`.
- `internal/config/config.go:25` - defaults are centralized in `NewDefault()`.
- `internal/config/config_test.go:11` - existing config tests cover default and round-trip behavior.
- `internal/project/init.go:38` - default config writing happens through `config.NewDefault()`.
- `internal/project/init_test.go:33` - project init tests confirm config file creation.
- `internal/steps/spec/steps.go:11` - `SpecFilePath` is the path helper for spec files.
- `internal/steps/spec/steps.go:63` - spec `new()` writes the scaffold and should continue to receive one canonical name.
- `internal/steps/spec/steps_test.go:112` - existing test asserts scaffold writing through `SpecFilePath`.
- `internal/steps/spec/strategy.go:14` - path strategy reports spec path and name from the workflow instance name.
- `internal/store/store.go:48` - store path resolution rejects paths that escape the root.
- `internal/store/store.go:112` - store existence checks support collision detection for timestamp, counter, and explicit-id targets.
- `templates/skills/workflows/spek-new/SKILL.md:8` - generated spec skill currently assumes the requested name is the final filename.
- `README.md:130` - README has the user-facing config section to update for `spec.id_method`.

## External references

None fetched during this planning pass. The local architecture note at `.spektacular/knowledge/architecture/cli-design-for-ai-agents.md` cites an external CLI design article and was used only for local design principles.

## Prior plans / specs consulted

- `.spektacular/specs/spec-id-prefix-method.md` - defines the required behavior and explicitly excludes existing spec migration, external-system integrations, workflow content changes, and plan/implementation artifact naming.
- `.spektacular/plans/13_agent-cli-improvements/plan.md` - confirms the project direction for JSON `--data`, schema introspection, dry-run behavior, and agent-facing validation.
- `.spektacular/plans/14_spektacular-store/plan.md` - confirms spec creation is store-backed and that path traversal protection belongs in the store layer while command/spec validation remains useful.
- `.spektacular/plans/15_implementation/context.md` - confirms current workflow state is shared and should not be split or migrated for this feature.

## Open assumptions

None. The reviewed gaps have been resolved in the plan: externally supplied-id mode is `external`, timestamp collisions bump by whole seconds, counter mode uses persisted config state, accepted names and ids are normalized without trimming, and tracked/generated skill guidance is updated.

## Rehydration cues

- Re-read `.spektacular/specs/spec-id-prefix-method.md` for the feature contract.
- Re-read `cmd/spec.go`, `internal/config/config.go`, `internal/steps/spec/steps.go`, `internal/steps/spec/strategy.go`, and `internal/store/store.go` before implementation.
- Use `rg -n "spec new|SpecFilePath|nameRegexp|config.yaml|id_method|counter|schema" cmd internal templates README.md .spektacular` to rebuild the local map.
- Use `go run . spec new --schema` after implementation to verify schema visibility.
- Focus first on resolver tests, then command tests, then full regression.
