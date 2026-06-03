# Context: 000024_resume

## Current State Analysis

The three workflows share one FSM engine and one persisted state file; only their step lists differ.

- `internal/workflow/state.go:14-20` — `State{CurrentStep, CompletedSteps, CreatedAt, UpdatedAt, Data}` with canonical JSON tags. No `kind` field today.
- `internal/workflow/state.go:22-26` — `markCompleted`. `:28-39` `loadState` (returns error on missing/invalid file). `:41-54` `saveState` (MkdirAll + `MarshalIndent` + `WriteFile`, stamps `UpdatedAt`).
- `internal/workflow/workflow.go:13-21` — `Config{Command, DryRun, SpecDir, PlanDir}`. No `Kind`.
- `internal/workflow/workflow.go:65-84` — `New()` loads existing state or, in the else branch (`:75-84`), creates fresh `State` with `initialState = steps[0].Src[0]`. This fresh-create branch is where `Kind` gets stamped.
- `internal/workflow/workflow.go:104-116` — each step's callback wired as an `after_<name>` FSM callback; this is the exact invocation shape `renderStep` must mirror (`step.Callback(w.data, w.out, w.store, w.cfg)`, honour returned `nextStep` via `pendingGoto`).
- `internal/workflow/workflow.go:130-139` — `enter_state` marks `e.Src` completed (if a valid step) and persists on every transition unless `DryRun`.
- `internal/workflow/workflow.go:164-181` — `Goto(name)`. The no-op guard is `:168` `if w.Current() == name { return nil }` — the single line repurposed for re-render.
- `internal/workflow/workflow.go:238-256` — `StepNames()` / `NextStepName()`; `:259-272` `StepStatus()`.
- `cmd/spec.go:98-100` — `stateFilePath(dataDir) = filepath.Join(dataDir, "state.json")`, shared by all three command files.
- `cmd/root.go:54-60` — `dataDir()` returns `<cwd>/.spektacular`. `:62-72` `projectRoot()` = cwd. `:74-82` `init()` registers `specCmd`/`planCmd`/`implementCmd`.
- Entry-command prologues that unconditionally remove state: `cmd/spec.go:208-213`, `cmd/plan.go:112-117`, `cmd/implement.go:123-128` (each: `statePath := stateFilePath(dataDir)`; if `dryRun` append `.dryrun-tmp` else `_ = os.Remove(statePath)`).
- `wfCfg` construction (where `Kind` must be set): `cmd/spec.go:215`, `cmd/plan.go:119`, `cmd/implement.go:130`.
- `new` flag registration in `init()`: `cmd/spec.go:363-365` (region), `cmd/plan.go:274-279`, `cmd/implement.go:291-296`. `--force` is added here.
- `goto` handlers (already call `wf.Goto(stepVal)`): `cmd/spec.go:290`, `cmd/plan.go:197`, `cmd/implement.go:208`. No change needed — they inherit re-render automatically and do not require a kind.
- Result structs: `internal/steps/spec/result.go`, `internal/steps/plan/result.go`, `internal/steps/implement/result.go` (each `{Step, <kind>_path, <kind>_name, Instruction}`).
- `internal/stepkit/stepkit.go:1-93` — `WriteStepResult`, `PathStrategy`, `ResultBuilder`, mustache rendering against the embedded `templates` FS. `internal/steps/plan/steps.go:159` shows `stepkit.RenderTemplate("scaffold/plan.md", vars)` — the call shape to render `resume.md`.
- Step templates: `templates/steps/spec/01-08*.md`, `templates/steps/plan/01-16*.md`, `templates/steps/implement/01-08*.md` (plus each `*-finished.md`).
- Skill playbooks referencing `state.json`: `templates/skills/workflows/spek-new/SKILL.md:49`, `spek-plan/SKILL.md:33,41`, `spek-implement/SKILL.md:32,42`.
- Skill/template install tests: `cmd/init_test.go`, `internal/agent/*_test.go` (e.g. `instruction_surface_test.go`, `bob_test.go`, `claude`/`codex` install paths in `internal/agent/skills.go`).
- Test conventions: `internal/workflow/workflow_test.go` (engine; `t.TempDir()`, rebuild-from-disk at `:89-100`; same-step no-op at `:70-79`; `TestCompletedStepsTracked` at `:191-203`). `cmd/spec_test.go` (command driving via `rootCmd.SetArgs`, `t.Chdir`, JSON unmarshal of stdout; `writeSpecCommandConfig`/`writeSpecCommandFile` helpers; asserts `NoFileExists(state.json)` patterns at `:158,201,237,256`). No `cmd/plan_test.go` or `cmd/implement_test.go` for `new` detection yet — added in Phase 2.2.

## Per-Phase Technical Notes

### Phase 1.1: Record workflow kind in state

- `internal/workflow/state.go:14-20` — add `Kind string \`json:"kind,omitempty"\`` as the first field of `State`.
- `internal/workflow/state.go:~26` — add method `func (s *State) InProgress() bool { return s.CurrentStep != "" && s.CurrentStep != "finished" }`.
- `internal/workflow/workflow.go:13-21` — add `Kind string` to `Config`.
- `internal/workflow/workflow.go:77-83` — in the fresh-create else branch, set `Kind: cfg.Kind` on the new `State`. Do not touch the load branch (`:72-74`).
- `cmd/spec.go:215`, `cmd/plan.go:119`, `cmd/implement.go:130` — set `Kind: "spec"` / `"plan"` / `"implement"` in each `workflow.Config{...}` literal so fresh state is stamped.
- Tests: `internal/workflow/workflow_test.go` — assert fresh `New(...)` state has the configured `Kind`; table-test `InProgress()` for an unfinished step (true) and `"finished"` (false). Use hand-written expected values.

**Complexity**: Low
**Token estimate**: ~15k
**Agent strategy**: Single agent, sequential.

### Phase 1.2: Re-render the current step on same-step goto

- `internal/workflow/workflow.go:167-181` — in `Goto`, replace the `:168` `return nil` with `return w.renderStep(name)`.
- Add `func (w *Workflow) renderStep(name string) error` — iterate `w.steps`, find the `StepConfig` whose `Name == name`, and if it has a `Callback`, invoke `step.Callback(w.data, w.out, w.store, w.cfg)` exactly as the `after_<name>` hook does at `:106-115`. Return its error. If the callback returns a non-empty `nextStep`, set `w.pendingGoto` and follow with `w.Goto(nextStep)` (mirrors `Next()` at `:157-159`) — defensive; current_step at rest is always a render step so this rarely fires.
- `internal/workflow/workflow_test.go:70-79` — rewrite `TestGotoSameStepIsNoop` into `TestGotoSameStepReRenders`: drive to a step whose callback records invocation, call `Goto(current)`, assert the callback ran and `Current()` + `CompletedSteps` are unchanged. Existing forward `Goto`/`Next` tests guard against regressions.
- **Open-question check belongs here**: read every callback in `internal/steps/spec/steps.go` and `internal/steps/implement/steps.go`. If any writes to the store/artifact/state as a render side effect, STOP and ask the user (plan.md § Open Questions).

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential.

### Phase 2.1: Detection primitives and the resume report

- New file `cmd/resume.go`:
  - `type ResumeReport struct { Resumable bool; Kind, Name, CurrentStep, Instruction string }` with JSON tags per plan.md § Data Structures.
  - `func resumeInstruction(kind, name, currentStep string) (string, error)` — renders `templates/steps/resume.md` via `stepkit.RenderTemplate("steps/resume.md", vars)` (shape per `internal/steps/plan/steps.go:159`).
- New template `templates/steps/resume.md` — mustache body rendered with `{{kind}}`, `{{name}}`, `{{current_step}}`; instructs the agent to (1) ask the user resume-vs-new, (2) on resume read `.spektacular/context.md` then run `{{command}} {{kind}} goto --data '{"step":"{{current_step}}"}'`, (3) on new run `{{command}} {{kind}} new --force`. Use the `{{command}}` placeholder convention (not a literal `go run .`).
- No `inferKind` and no kind registry — `Kind` comes straight from the loaded state.
- Tests: `cmd/resume_test.go` — assert the rendered instruction contains both follow-up commands and interpolates kind/name/current_step.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent (one cohesive new file + template).

### Phase 2.2: Entry commands detect, report, and force

- Add `--force` bool flag to each `new` command in `init()`: `cmd/spec.go:363-365`, `cmd/plan.go:274-279`, `cmd/implement.go:291-296`.
- New shared helper in `cmd/resume.go`, e.g. `func detectInProgress(statePath string) (*workflow.State, error)` — read the state file (`os.ReadFile` + `json.Unmarshal` into `workflow.State`, or a small exported loader); if the file is absent or `!state.InProgress()` return `(nil, nil)`; otherwise return the in-progress state.
- In each `new` prologue (`cmd/spec.go:208-213`, `cmd/plan.go:112-117`, `cmd/implement.go:123-128`): when not `--force` and not `--dry-run`, call `detectInProgress`. If it returns an in-progress state:
  - if `state.Kind == ""` → return a clear error (e.g. "existing workflow state has no kind marker; run with --force to start fresh"). No clobber.
  - else render the resume instruction and write a `ResumeReport`, then return (write nothing else).
  Otherwise `os.Remove(statePath)` and proceed as today. Keep the `.dryrun-tmp` branch unchanged (dry-run does not detect).
- No migration / stamp-on-write: fresh state always carries `kind` (Phase 1.1); legacy kind-less state is never silently upgraded.
- Tests: extend `cmd/spec_test.go` and add `cmd/plan_test.go` / `cmd/implement_test.go`:
  - `new` on in-progress slot → `ResumeReport` JSON with correct `kind`/`name`/`current_step`; assert the state file is byte-for-byte unchanged.
  - `new --force` → fresh first-step `Result`.
  - clean dir → normal first-step `Result`, `resumable` absent/false.
  - cross-kind: seed a spec state, run `plan new`, assert `kind:"spec"`.
  - kind-less: seed a state file with no `kind`, run `new`, assert a clear error and the file is untouched.

**Complexity**: Medium
**Token estimate**: ~35k
**Agent strategy**: 2-3 parallel agents (one per command file) after the shared helper lands; sequential integration of the helper first.

### Phase 3.1: Step templates refresh the working context

- Append a uniform directive block to each step instruction template under `templates/steps/spec/` (`01-overview.md`..`08-verification.md`), `templates/steps/plan/` (`01-overview.md`..`16-write_research.md`), and `templates/steps/implement/` (`01-read_plan.md`..`08-update_repo_changelog.md`). The `*-finished.md` terminal templates may be skipped (no advance follows).
- Directive wording (identical across all): instruct the agent to refresh `.spektacular/context.md` with the current working context (key decisions, findings so far, what is done, what is next) before calling the next `goto`, using its own file tools, and note the file is git-tracked.
- No Go changes. Verify rendered output via existing template-render tests if present; otherwise a spot render.

**Complexity**: Low (mechanical, broad)
**Token estimate**: ~25k
**Agent strategy**: 3 parallel agents (one per workflow directory), identical directive text.

### Phase 3.2: Resume-aware skill playbooks

- `templates/skills/workflows/spek-new/SKILL.md:49`, `spek-plan/SKILL.md:33,41`, `spek-implement/SKILL.md:32,42` — replace "inspect `.spektacular/state.json` for the active name" guidance with: call `{{command}} <kind> new`; if the response is a `ResumeReport` (`resumable: true`), ask the user resume-vs-new; on resume read `.spektacular/context.md` then `{{command}} <kind> goto --data '{"step":"<current_step>"}'`; on new `{{command}} <kind> new --force`.
- Keep wording consistent across the three playbooks; preserve the `{{command}}` placeholder convention.
- Verify against `cmd/init_test.go` and `internal/agent/*_test.go` install/surface tests; update any test that asserts the old `state.json` guidance text.

**Complexity**: Medium
**Token estimate**: ~30k
**Agent strategy**: Single agent, sequential (consistent wording across playbooks).

## Testing Strategy

Engine behaviour is proven in `internal/workflow/workflow_test.go` using `t.TempDir()` state files and rebuild-from-disk (construct a second `Workflow` from the same path to simulate a fresh process — the existing `TestAutoSaveOnTransition` pattern). Command behaviour is proven in `cmd/*_test.go` by driving `rootCmd` with `SetArgs`, `t.Chdir(t.TempDir())`, and unmarshalling stdout JSON — the existing `cmd/spec_test.go` pattern. The shared detection helper is tested once (`cmd/resume_test.go`); per-command tests confirm wiring only, avoiding redundant assertions. Oracles (expected step names, resumable flag, error outcomes) are hand-written constants, never derived from the subject's own `Steps()`. No test reads or writes `.spektacular/context.md` (agent-owned prose, no CLI contract); skill-playbook edits are covered by existing install tests, not new behavioural tests.

## Project References

- Spec: `.spektacular/specs/000024_resume.md`.
- Prior plan `000015_implementation` (`context.md`, `research.md`) — documents the deliberate collapse from per-workflow state dirs (`.spektacular/plan-<name>/state.json` + `findActivePlan`) to the single shared `.spektacular/state.json`; confirms the current single-file model and that `implement new` truncates it like the others.
- Prior plan `000017_plan_testing` — references the `State` struct shape and `completed_steps` ordering invariant.
- `internal/workflow/workflow_test.go:163-189` (`TestMultiSourceTransition`) — confirms `looplab/fsm` multi-source edges already work; relevant to the implement workflow's `analyze` loop, untouched here.

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phases 1.1, 1.2, 2.1, 3.1 are Low. Phases 2.2, 3.2 are Medium. No High-tier phases — the change is broad but shallow, with one load-bearing engine line.

## Migration Notes

No automatic migration. Fresh state written under this change always carries `kind` (stamped at creation). A pre-existing kind-less `state.json` (from before this change) is not inferred or upgraded: `new` detection rejects it with a clear error, and the user either starts fresh with `--force` or finishes the existing workflow via `goto` (which needs no kind). `loadState` still reads old files unchanged — the new `kind` field is `omitempty` and absent old files parse fine — so `goto`/`status` continue to work on them.

## Performance Considerations

None. All operations are single small-file reads/writes already performed today; detection adds at most one state-file read per `new` invocation, and re-render runs one already-cheap template render. No hot paths, no new I/O of consequence.
