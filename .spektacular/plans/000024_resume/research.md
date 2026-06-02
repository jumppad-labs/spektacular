# Research: 000024_resume

## Alternatives considered and rejected

### Option A: Per-kind state files (one file per kind)

Store state in separate files keyed by kind — e.g. `.spektacular/state/spec.json`, `state/plan.json`, `state/implement.json` — so an in-progress spec and an in-progress plan coexist without clobbering each other. This is one of the two storage options the spec's technical approach explicitly offers and would satisfy AC #8 (per-kind independence) literally.

**Rejected**: The user explicitly chose a single state file with git as the recovery mechanism ("there should be a single state … Git commits can be used to manage the state files"). Per-kind files reintroduce a multi-file layout the codebase deliberately collapsed away — see `.spektacular/plans/000015_implementation/context.md` (documents the move from per-workflow `.spektacular/plan-<name>/state.json` + `findActivePlan` to the single shared `.spektacular/state.json`). AC #8 is instead honoured by surfacing (not ignoring) a cross-kind in-progress workflow.

### Option B: Per-(kind, name) state

Store state per workflow instance — e.g. `.spektacular/state/plan/<name>.json` — so multiple in-progress plans coexist and start-new for a different name leaves earlier ones fully resumable. This most literally satisfies spec AC #5 ("start-new keeps the old one … still exists unchanged").

**Rejected**: Same user directive (single state slot, git for recovery). It also complicates detection — `new` would have to scan for any in-progress instance of the kind and the resume prompt could surface several — contradicting the spec's singular "an in-progress workflow of that kind" framing. Recovery of an overwritten workflow is handled by `git checkout` of the prior `state.json`/`context.md` instead.

### Option C: Dedicated `<kind> resume` subcommand

Add a new `resume` subcommand per kind that re-renders the current step, rather than overloading `goto`.

**Rejected**: The engine already has `Goto`, whose same-step branch is a no-op (`internal/workflow/workflow.go:168`). Repurposing that branch to re-render reuses the command the agent already calls and adds zero new CLI surface. The user confirmed reusing `goto` and that resume must target the *current* step (its work "would not yet have been persisted"). A new subcommand was unnecessary.

### Option D: CLI-managed working-context file

Have the CLI own `.spektacular/context.md` — written via `goto --stdin/--file` plumbing and embedded in the resume report — mirroring how `state.json` and plan documents are CLI-managed.

**Rejected**: The user decided "for now it is ok for the llm to write and read freely to this file using their internal tools." The context is free-form prose for agent rehydration; inventing a serialization contract and CLI plumbing for it is deferred (plan.md § Out of Scope).

### Option E: Resume by advancing to the next step

On resume, point the agent at `goto <next_step>` (the step after the interrupted one), which works with `goto` entirely unchanged.

**Rejected**: This marks the interrupted step completed and skips re-showing it, violating spec AC #3 ("advance to the first not-yet-completed step" = `current_step`). The user confirmed resume must target the current step because its work was never persisted.

### Option F: Infer the kind of legacy kind-less state

When a pre-existing `state.json` carries no `kind` marker, recover it by fingerprinting `current_step` + `completed_steps` against each kind's `StepNames()` (the lists are largely disjoint; `completed_steps` disambiguates shared names like `verification`), then stamp it on the next write.

**Rejected**: The user chose to keep this simple — "we don't need inferKind for legacy state, if not kind, just error." Inference adds a heuristic that can misclassify (an empty/sparse completed-list on a shared step is genuinely ambiguous) and a stamp-on-write migration path, for a transient edge that only exists for workflows already mid-flight at upgrade time. Erroring on a kind-less in-progress state — leaving the user to `--force` or finish via `goto` — is simpler and never guesses wrong.

## Chosen approach — evidence

- Per-step persistence already exists: `internal/workflow/workflow.go:130-139` (`enter_state` saves on every transition unless `DryRun`) and `commitTerminal` at `:187-203` — so spec AC #1 is largely satisfied today; the feature builds on it.
- The blocker is concrete and localised: `cmd/spec.go:212`, `cmd/plan.go:116`, `cmd/implement.go:127` each `_ = os.Remove(statePath)` before starting.
- Re-render is a one-line change at a real no-op: `internal/workflow/workflow.go:168`. The callback-invocation shape to reuse is at `:104-116`.
- Re-render safety: plan step callbacks are render-only / read-only — `internal/steps/plan/steps.go:84-286` (each calls `writeStep`/`RenderTemplate`; `planDocStillScaffold` at `:202-212` only *reads* the store). This is the evidence the design relies on; the spec/implement callbacks must be confirmed during Phase 1.2 (plan.md § Open Questions).
- Single shared engine confirms DRY is feasible: `internal/workflow/workflow.go:45-143` (`Workflow` + `New`), driven by per-kind `Steps()` in `internal/steps/{spec,plan,implement}/steps.go`.
- Kind is always available without inference: it is stamped at creation in the fresh-create branch (`internal/workflow/workflow.go:75-84`) from `Config.Kind`, so detection and cross-kind reporting read `state.Kind` directly; no fingerprint of `StepNames()` is needed.

## Files examined

- `internal/workflow/state.go:14-54` — `State` struct + `loadState`/`saveState`; no `kind` field; load returns error on missing file.
- `internal/workflow/workflow.go:13-21,65-143,164-203,238-272` — `Config`, `New` (fresh-create branch `:75-84`), `Goto` no-op `:168`, `enter_state`/`commitTerminal`, `StepNames`/`StepStatus`.
- `internal/workflow/workflow_test.go:70-79,89-100,191-203` — same-step no-op test (to change), rebuild-from-disk pattern, completed-step tracking.
- `cmd/root.go:54-82` — `dataDir`, `projectRoot`, command registration.
- `cmd/spec.go:98-100,196-294,363-365` — `stateFilePath`, `new`/`goto` handlers, flag init.
- `cmd/plan.go:68-201,274-282` — `new`/`goto`/`status` handlers (prologue `:112-117`, wfCfg `:119`), flag init.
- `cmd/implement.go:72-212,291-296` — `new`/`goto` handlers (prologue `:123-128`, plan-precondition `:118-121`), flag init.
- `internal/steps/plan/steps.go:28-286` — plan `Steps()`, `writeStep`, read-only doc checks.
- `internal/stepkit/stepkit.go:1-93` — shared rendering pipeline + `RenderTemplate` usage shape.
- `templates/steps/{spec,plan,implement}/*.md` — step instruction templates (targets for the context directive).
- `templates/skills/workflows/spek-{new,plan,implement}/SKILL.md` — playbooks referencing `state.json` (`spek-new:49`, `spek-plan:33,41`, `spek-implement:32,42`).
- `cmd/spec_test.go:1-258` — command test conventions (`t.Chdir`, `SetArgs`, JSON unmarshal, `NoFileExists(state.json)` assertions).

## External references

- `looplab/fsm` — the FSM library underlying the engine. Why it mattered: confirmed that re-render must sidestep the FSM (an X→X event is not defined, which is why the current code guards with a no-op rather than firing a transition); multi-source edges already work (`TestMultiSourceTransition`).

## Prior plans / specs consulted

- `.spektacular/specs/000024_resume.md` — the source spec; requirements, AC #1–#8, and the technical-approach hints (detect-instead-of-delete, kind marker, per-kind storage option, migration risk).
- `.spektacular/plans/000015_implementation/{context.md,research.md}` — documents the historical collapse from per-workflow state directories to the single shared `state.json`; confirms `implement new` truncates the shared file like `spec`/`plan`, and that `findActivePlan`/per-workflow dirs no longer exist. Directly informs the single-slot decision.
- `.spektacular/plans/000017_plan_testing/{plan.md,context.md,research.md}` — references the `State` struct shape and the `completed_steps == Steps() order` invariant.

## Open assumptions

- **Spec and implement step callbacks are render-only / read-only.** Verified for the plan workflow only. If any spec/implement callback writes as a side effect of rendering, re-render would repeat it on resume — the implement workflow must STOP and ask (plan.md § Open Questions, checked in Phase 1.2).
- **Dry-run keeps today's behaviour** (operates on `.dryrun-tmp`, no detection). Assumed acceptable; not exercised by new tests.
- **A kind-less in-progress state is rare and transient** (only workflows already mid-flight at upgrade time). Assumed acceptable to handle by erroring rather than inferring; if this proves disruptive in practice, the rejected Option F could be revisited.

## Rehydration cues

- Re-read the spec: `.spektacular/specs/000024_resume.md`.
- Re-read this plan set: `go run . plan file read 000024_resume/plan.md` (and `context.md`, `research.md`).
- Engine entry points to re-open: `internal/workflow/workflow.go:65-143` (`New`), `:164-181` (`Goto`, no-op at `:168`), `:130-139` (`enter_state`); `internal/workflow/state.go:14-54`.
- Command prologues to change: `cmd/spec.go:208-213`, `cmd/plan.go:112-117`, `cmd/implement.go:123-128`; flag init at `cmd/spec.go:363-365`, `cmd/plan.go:274-279`, `cmd/implement.go:291-296`.
- Confirm callback safety before Phase 1.2: read `internal/steps/spec/steps.go` and `internal/steps/implement/steps.go` end-to-end.
- Test patterns to mirror: `internal/workflow/workflow_test.go` (engine, rebuild-from-disk) and `cmd/spec_test.go` (command driving).
