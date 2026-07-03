# Plan: 000031_agent-session-debug-log

<!-- Metadata -->
<!-- Created: 2026-07-02T16:15:24Z -->
<!-- Commit: 62b1af7 -->
<!-- Branch: b-debugging -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This plan adds an opt-in, off-by-default local record of every command an agent issues to Spektacular and every response it receives during a session, so a different agent can reconstruct what happened afterward without needing the original agent's memory of the session. It solves a real gap in how Spektacular's workflows run: entirely unsupervised, with no person watching, so when an agent stalls, repeats itself, or takes a wrong turn today, nothing captures what actually happened. Whoever later needs to diagnose a stuck or misbehaving session benefits, at no cost to normal operation — the record has zero effect on the tool's actual behavior when enabled, and needs no new dependency, service, or process to produce.

## Conventions

No project conventions apply to this feature. The project's `conventions/` knowledge area currently contains only its own placeholder README (`.spektacular/knowledge/conventions/README.md`, "This directory contains conventions documentation.") with no actual convention entries defined yet — confirmed by listing the directory during discovery, not skipped.

## Architecture & Design Decisions

The recorder is a thin, additive layer around the tool's existing single outcome choke point, `cmd/root.go`'s `runRoot()` (introduced by plan `000030_agent-actionable-errors`'s Phase 1.2 as the one place every command's success or failure already converges before reaching stdout). `runRoot()` captures `os.Args[1:]` verbatim as "the command the agent issued" before calling `rootCmd.Execute()`, and — only when the toggle is on — swaps `rootCmd`'s output stream for `io.MultiWriter(originalOut, captureBuf)` so every byte any of the tool's ~40 existing response call sites write (including the one `skillListCmd` outlier that bypasses `internal/output.Write`) is duplicated into an in-memory buffer without changing what the real stream receives. After `rootCmd.Execute()` returns, `runRoot()` has the full raw response bytes, the exit outcome, and — by reading `.spektacular/state.json` once immediately before `Execute()` and once immediately after — a before/after snapshot of the workflow engine's persisted state. A new, small package (proposed `internal/sessionlog`) owns turning that into one JSON Lines record appended to `.spektacular/debug/session-log.jsonl`: `argv`, `duration_ms`, `exit_code`, the full raw `response` bytes, and a `state_changed` signal derived from diffing `CurrentStep`/`CompletedSteps`/`UpdatedAt` rather than from the returned error's type alone. This directly closes a real gap: `Workflow.Goto` to the current step is a documented silent no-op (`.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md`) that returns `nil` and is indistinguishable from a genuine advance by error type — exactly the class of stuck-agent behavior this feature exists to make diagnosable after the fact (research.md#alternatives-considered-and-rejected, alternative 3).

The opt-in toggle reuses `internal/config.Config.Debug.Enabled` (`internal/config/config.go:39-42,92`), an existing, already-tested, currently-unwired field that already defaults to `false` and already round-trips through `.spektacular/config.yaml` — no new flag or environment variable is introduced, per the spec's Technical Approach preference to reuse an existing mechanism over adding a new one. `runRoot()` loads config independently of each subcommand's own `loadConfig()` call solely to read this one field before deciding whether to wire up the tee; a config-load failure at that point fails open (recording silently stays off for that invocation) rather than surfacing a new failure mode, since the spec's "Non-invasive" requirement means the recorder must never be capable of changing the primary command's outcome.

Session identity is derived, not minted: `session_id` is a pure function of `.spektacular/state.json`'s existing `(Kind, Name)` pair, computed from the same before-state read already needed for the outcome diff above — no new marker file, lock, or persisted counter is required. A session boundary occurs exactly when `(Kind, Name)` changes between two consecutive commands (a new workflow instance starts, or the kind switches); resuming the same named instance after a long real-world gap is deliberately treated as a continuation of the same session, not a new one, since the record's purpose is reconstructing what happened to one piece of work, not fragmenting on wall-clock idle time. Commands issued with no workflow state on disk yet (an ad hoc `knowledge search` before any workflow has run) fall back to a fixed sentinel session id. This approach was chosen directly on the user's steer during the architecture discussion — link the session boundary to the state the tool already tracks, rather than inventing a new idle-timeout heuristic requiring its own persisted marker (research.md#chosen-approach--evidence).

Two heavier alternatives were considered and rejected: hooking `cobra.PersistentPreRunE`/`PersistentPostRunE` for structured command/result capture, and teaching `internal/output.Write`/`WriteFailure` themselves to emit recorder events. Both were rejected primarily because `PostRun` never fires when `RunE` returns an error — meaning both approaches would still need root-level capture for every failure response anyway, adding cobra-lifecycle or shared-package plumbing (touching `internal/workflow`'s public `Config`/`ResultWriter` interfaces, or `internal/output`, both used across every command family) without eliminating the need for the simpler mechanism underneath. The chosen approach keeps the entire feature scoped to `cmd/root.go` plus one new package, touching neither `internal/workflow` nor `internal/output` nor any of the ~40 existing response call sites — the same "single choke point beats editing every call site" precedent plan `000030` already established for the adjacent error-formatting problem (research.md#alternatives-considered-and-rejected, alternatives 1-3).

Each record stores the tool's full raw response bytes verbatim rather than a structured subset — the simplest shape that satisfies the spec's "Self-contained record" and "Machine-extractable detail" requirements directly, and the only one that needs no special-casing for `file read`'s raw-bytes (non-JSON) success path or `skillListCmd`'s bypass of `output.Write`. The record file itself, `.spektacular/debug/session-log.jsonl`, lands in a directory already excluded from version control by the repo's root `.gitignore` (`.spektacular/debug/`), satisfying "must never be committed to VCS" with no new gitignore entry required; it is a single continuously-appended file (not one file per session) since the spec puts retention/rotation/cleanup explicitly out of scope, and a reader reconstructs one session by filtering lines on `session_id`.

## Component Breakdown

- **Debug toggle** (existing component, `internal/config`'s `Debug` section — gains a consumer, no schema change). Owns the on/off state of the whole feature: whether it is enabled, defaulted off, and how an operator sets it via the project's config file. Nothing else in the tool is allowed to decide independently whether recording happens — every other component treats this as the single source of truth for "is recording on right now."

- **Command outcome wrapper** (existing component, the tool's single response choke point — extended, not replaced). Already owns turning every command's outcome (success or failure) into the one response the agent receives. This plan extends its responsibility: when the debug toggle is on, it also captures the exact command the agent issued, duplicates the exact response bytes it was already about to send (without altering them), and takes a snapshot of workflow state immediately before and immediately after the command runs. It hands all of that, as one unit, to the session recorder. When the toggle is off, none of this extra work happens and the wrapper behaves exactly as it does today — recording is strictly additive to its existing responsibility, never a precondition for it.

- **Session recorder** (new component). Owns the shape and durability of the record itself: deciding which session a given command belongs to (derived from workflow state, not tracked independently), building one self-contained entry per command from what the command outcome wrapper hands it, deciding whether the command actually advanced anything (not just whether it returned an error), and appending that entry to the on-disk record. It owns the on-disk location and format of the record and is the only component that writes to it. Any failure inside this component (e.g. an unwritable disk) is contained here and never allowed to change the outcome the agent receives — that guarantee is this component's responsibility to uphold, not the wrapper's.

- **Workflow state store** (existing component, unchanged — becomes a read-only dependency of the session recorder). Already owns the durable record of which workflow is active, its kind, its name, and its progress. This plan adds no new reader-facing behavior to it; the session recorder simply reads its current on-disk state before and after a command, the same way any other caller would, to derive session identity and to tell a genuine advance apart from a rejected or no-op command.

## Data Structures & Interfaces

**`Event`** — one JSON Lines record, the serialization boundary between the command outcome wrapper and the on-disk record. One `Event` is written per completed CLI invocation while recording is enabled.

```go
type Event struct {
    Seq        int64          // monotonic within the record file — the tie-breaker "Unambiguous ordering" needs
    Timestamp  time.Time      // when the command completed
    SessionID  string         // see SessionID() below
    Command    []string       // os.Args[1:], verbatim — "the command the agent issued"
    DurationMS int64
    ExitCode   int
    Response   string         // the exact raw bytes written to stdout for this command, verbatim
    StateBefore *StateSnapshot // nil if no workflow state existed on disk yet
    StateAfter  *StateSnapshot // nil under the same condition
    Advanced    bool          // true only if StateAfter meaningfully differs from StateBefore
}
```

**`StateSnapshot`** — the small, read-only view of `.spektacular/state.json` the wrapper takes before and after a command, and the input to both `Advanced` and `SessionID`.

```go
type StateSnapshot struct {
    Kind           string
    Name           string
    CurrentStep    string
    CompletedSteps []string
}
```

**`SessionID(state *StateSnapshot) string`** — a pure function, not a stored value: derives the session a command belongs to directly from the current `(Kind, Name)` pair (`nil` state maps to a fixed sentinel). Same inputs always produce the same id, so no session registry, lock, or marker file exists anywhere in this design — the workflow state store already used elsewhere in the tool is the only source of truth for session identity.

**`Recorder`** — the contract the command outcome wrapper calls into; the only entry point the session recorder component exposes.

```go
func Record(logPath string, ev Event) // no error return, by design
```

`Record` never returns an error and never panics outward — any failure while building or writing an entry (an unwritable disk, a marshal error, whatever) is contained and dropped inside this one function. This is a deliberate type-level guarantee, not just a convention: the command outcome wrapper has no error value to check or propagate, so a recording failure structurally cannot become a command failure, which is how the plan enforces the spec's "Non-invasive" requirement rather than relying on callers to remember to ignore it.

No changes are made to `internal/output`'s `ErrorResponse`/`Write`/`WriteFailure` types, to `internal/workflow`'s `Data`/`Config`/`ResultWriter` interfaces, or to `internal/config.Config`'s schema — all three are read or used as they exist today.

## Implementation Detail

The plan introduces one new leaf package (proposed `internal/sessionlog`) following the same shape every other `internal/*` package in this codebase already uses — small, single-responsibility, no dependents beyond `cmd`. It owns the `Event`/`StateSnapshot` types, `SessionID`, and `Record` from § Data Structures & Interfaces, and nothing else; it does not import `internal/workflow`, `internal/output`, or `internal/config` — `cmd` builds the `StateSnapshot` values itself (from a file it already knows how to locate) and passes them in, keeping the new package a pure, easily-testable leaf with no knowledge of workflows, responses, or configuration.

`runRoot()` gains a clearly-bracketed recording concern rather than having its existing outcome-handling logic rewritten around it. The toggle check happens first and short-circuits: when `Debug.Enabled` is false — the default, and the overwhelmingly common case — `runRoot()`'s behavior is byte-for-byte what it is today, with no extra reads, no extra writes, and no extra allocations beyond the one boolean check. Only when the toggle is true does the wrapper do the extra work: read the state snapshot, swap in a tee'd writer, execute, read the state snapshot again, and hand the result to `sessionlog.Record`. This mirrors the shape `runRoot()` already has today — a short sequence of named steps around `rootCmd.Execute()` — rather than introducing a new control-flow pattern; a developer reading `cmd/root.go` should be able to tell at a glance that recording is optional, additive, and cannot be reached at all on the toggle-off path, not something woven through the core logic.

Capturing the response stream introduces this codebase's first use of `io.MultiWriter` (Go's standard library tee) — a well-known, idiomatic pattern rather than a new abstraction. It requires no new dependency: everything the recorder needs (`encoding/json`, `os`, `io`) is already in the standard library, satisfying the spec's "no new... dependency" constraint by construction, not just by intent.

Reading `.spektacular/state.json` for the before/after snapshot follows an existing pattern rather than introducing a new one: `cmd` already owns the path-construction helpers for that file (`dataDir`/`stateFilePath`-equivalent logic already lives in `cmd`, not in `internal/workflow`), so the session recorder's caller reads the same on-disk file the same way any other part of `cmd` already does, as a second independent, read-only consumer — no new export is added to `internal/workflow` for this, and no lock or coordination is needed since the read is a snapshot, not a write.

The on-disk record itself is this codebase's first JSON Lines writer: append one JSON object plus a trailing newline per call, opening the file with the standard append-mode flags. This is a new but minimal, self-contained pattern local to the one new package — it does not touch or generalize `internal/output`'s existing (and unrelated) single-JSON-object-per-response writer, keeping the two concerns — "the response the agent receives" and "the record of what happened" — visibly separate in the code the way they are separate in the spec.

## Dependencies

- **`cmd/root.go`'s `runRoot()` / `Execute()`** (internal, existing). The command outcome wrapper introduced by plan `000030_agent-actionable-errors` (Phase 1.2). This plan depends on it already existing and already being the single place every command's outcome converges — it is extended, not replaced, and needs no prior change beyond what plan `000030` already shipped.

- **`internal/config.Config.Debug`** (internal, existing). Provides the opt-in toggle this plan reuses as-is. No schema change needed — the field, its default (`false`), and its YAML round-trip already exist and are already tested.

- **`.spektacular/state.json` / `internal/workflow`'s state persistence** (internal, existing). Provides the on-disk `(Kind, Name, CurrentStep, CompletedSteps)` data the session recorder reads to derive session identity and the advanced/no-op signal. Read-only dependency — this plan does not modify `internal/workflow`, its persistence format, or its public interfaces.

- **`.spektacular/knowledge/gotchas/goto-to-current-step-is-a-silent-noop.md`** (project knowledge, existing). Not a code dependency, but the documented evidence this plan's outcome-detection design is built to correctly handle — cited here so a reviewer can re-verify the design against it without re-deriving the finding.

- **Go standard library only** (`encoding/json`, `io`, `os`, `time`). No new external library or module is introduced — satisfies the spec's "no new... runtime dependency" constraint directly; `go.mod` does not change.

- **No prior spec or plan must land first.** Spec `000030_agent-actionable-errors` explicitly deferred this exact feature ("A debug or session action-log for reconstructing agent behavior after the fact is out of scope; it is left for a separate, future spec") and has already shipped and merged, so its `runRoot()`/`ErrorResponse` foundation is already in place. No other in-flight spec or plan blocks this one.

## Testing Approach

Coverage concentrates on the new `internal/sessionlog` package (unit tests, no dependency on cobra or the workflow engine) and on `cmd/root.go`'s extended `runRoot()` (integration tests driven through the same `runRootCmd`-style in-process harness the existing command-family tests already use, per this project's established CLI test convention). `internal/sessionlog` is tested in isolation: `SessionID` as a pure function across a matrix of `(Kind, Name)` inputs including the no-state-yet case, and `Record` proving it never returns an error or panics even when the write target is unwritable — the type-level guarantee § Data Structures relies on is only real if a test forces the failure path and confirms the caller sees nothing. `runRoot()`'s integration tests drive real command sequences end-to-end (spec/plan/implement families, plus at least one non-workflow command) with recording enabled and assert on the resulting record file's contents, and separately assert that recording enabled vs. disabled produces byte-identical stdout and exit codes for the same command sequence — the load-bearing guarantee the whole feature is built around.

The load-bearing assertions, in plain language: every command issued while recording is on gets exactly one entry, in order; an entry exists and is complete for rejected, erroring, and no-op commands, not only successful ones; a `Goto` call to the already-current step — the documented silent-no-op case — is recorded with `Advanced: false` despite returning no error, proving the design's outcome signal is not just error-type inspection; entries for the same workflow instance share one `SessionID` and entries from a differently-named or differently-kinded instance never share it; and turning recording on changes nothing else about what the tool does. This directly follows the pattern plan `000030_agent-actionable-errors` used for its own wrapper tests (`cmd/root_test.go`'s `TestWrapper_*` family) — a small number of tests, each proving one cross-cutting guarantee across representative command families, rather than one test per command.

Deliberate gap: no dedicated performance/benchmark test. The design adds a fixed, small amount of work per command (two small file reads, one buffer copy, one JSON-line append) with no unbounded growth per call, which is a structural argument rather than a measured one — acceptable here because the spec's cost concern (see Success Metrics below) is about operator adoption behavior over time, not a specific latency budget, and no latency requirement exists to test against.

**Success Metrics** (from the spec):

- *"Given the record from a session where an agent got stuck or behaved unexpectedly, another agent can reconstruct what happened and identify the point things went wrong without needing a person to explain the session or without access to the original agent's own memory of it."* — **Manual, captured in the implementation test plan.** Whether a diagnosing agent can actually succeed at this is not mechanically assertable; the implement workflow's test plan will exercise this directly (deliberately induce the silent-no-op scenario or a rejected-transition scenario live, then hand only the resulting record to a fresh session and confirm it can name the stuck point), grounded in this plan's own automated tests proving the record's completeness, ordering, and no-op-detection (above) as the mechanical precondition that manual exercise depends on.
- *"Enabling this recording does not become something operators avoid using because of its cost or side effects — it is left on for as long as an investigation needs it without concern."* — **Split.** The "side effects" half is a **behavioural test**: the toggle-on/toggle-off byte-identical-output assertion above directly guarantees enabling recording changes nothing an operator or agent would notice. The "cost" / ongoing-adoption half is **manual, captured in the implementation test plan** — actual operator behavior over time cannot be observed from this session's test run, the same way plan `000030`'s analogous adoption metric was handled as an ongoing observation procedure rather than a one-time automated check.

## Milestones & Phases

### Milestone 1: Turning on recording captures every command and its exact response, with zero effect on what the tool actually does

**What changes**: An operator can opt in to a local record of a Spektacular session by setting the existing debug option in their project's config file — no new flag or environment variable is introduced. Once enabled, every command the tool receives and the exact response it returns are captured, in order, to a plain local file that Spektacular already keeps out of version control — including commands that are rejected, error, or otherwise make no forward progress, not only the ones that succeed. Nothing else about how the tool behaves changes: the same command produces the exact same output and exit code whether recording is on or off, and recording stays off by default.

#### - [x] Phase 1.1: Build the session record writer

A new, self-contained package owns the shape of one recorded entry and how it gets appended to disk — a plain local file, one JSON object per line, containing the command that was issued, the exact response returned, how long it took, and its exit code. This package knows nothing about workflows, commands, or configuration; it only knows how to turn one already-assembled entry into one line on disk, and it is built and proven correct entirely on its own, including proving that a write failure (a full disk, an unwritable path) can never surface as an error the rest of the tool has to react to.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-build-the-session-record-writer)

**Acceptance criteria**:
- [x] Given a sequence of entries handed to the writer, the resulting file contains one valid, independently-parseable JSON object per line, in the order they were given, each carrying a strictly increasing sequence number.
- [x] When the writer cannot write (its target location is not writable), no error is returned or raised to the caller — the failure is contained entirely inside the writer.

#### - [x] Phase 1.2: Wire recording into every command, gated by the existing debug option

Every command the tool runs now optionally records itself, controlled by the project's existing debug setting (already present in the config file, already off by default — no new flag or environment variable). When the setting is on, the exact command issued and the tool's exact response are captured for every invocation — including ones that are rejected, that error, or that otherwise make no forward progress, not just the ones that succeed. When the setting is off, or absent entirely, nothing changes: the tool's real output and exit code are identical to today, byte for byte, for every command family.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-wire-recording-into-every-command-gated-by-the-existing-debug-option)

**Acceptance criteria**:
- [x] With the debug setting off (the default), running any command produces no record file and no observable difference from today's behavior.
- [x] With the debug setting on, running a command that succeeds, one that is rejected, and one that errors each produce a corresponding recorded entry — none are silently dropped.
- [x] With the debug setting on, the response and exit code the caller actually receives are identical to what they would receive with it off, for a representative command from every command family.

### Milestone 2: The record distinguishes real progress from a stalled or rejected command, and keeps separate pieces of work separate

**What changes**: Beyond capturing what was asked and what was returned, each entry in the record now also says whether that command actually moved anything forward — catching, in particular, the case where a request looks successful but silently made no progress at all, which is exactly the kind of thing that leaves an agent stuck without anyone noticing. Entries belonging to the same piece of work stay grouped together, and entries from a different or later piece of work never get mixed in with an earlier one, so a diagnosing agent reading the record afterward can tell whether and where things went wrong without having to untangle two unrelated stories from the same file.

#### - [x] Phase 2.1: Record whether a command actually advanced anything

Each recorded entry now also states whether the command actually changed anything, determined by comparing the tool's own tracked progress immediately before and immediately after the command ran — not by looking at whether an error was returned. This specifically catches the case where a command reports success but silently made no progress at all (repeating a step that is already the current one), which today looks identical to a real advance and is exactly the kind of thing that leaves an agent stuck without any visible sign of it.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-record-whether-a-command-actually-advanced-anything)

**Acceptance criteria**:
- [x] A command that genuinely advances a piece of work in progress is recorded as having advanced it.
- [x] A command that repeats the step already current — which returns no error and looks like a normal response — is recorded as not having advanced anything.
- [x] A command unrelated to any piece of work in progress (no tracked progress exists yet) is recorded without error, clearly showing there was nothing to advance.

#### - [x] Phase 2.2: Group recorded entries by the piece of work they belong to

Every recorded entry is now tagged with which piece of work it belongs to, derived automatically from the tool's own tracking of what is currently being worked on — no separate session setup or identifier is required from whoever is issuing the commands. Commands that belong to the same piece of work always share the same tag, even across a long gap and even if the work is picked back up later; commands belonging to a different or unrelated piece of work never share that tag, so a record accumulated over time can always be split back out into the distinct pieces of work it covers.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-group-recorded-entries-by-the-piece-of-work-they-belong-to)

**Acceptance criteria**:
- [x] Every entry recorded while working on the same named piece of work carries the same tag, including entries recorded before and after resuming it later.
- [x] Entries recorded while working on two differently named (or differently kinded) pieces of work never carry the same tag.
- [x] A command issued before any piece of work has been started yet is still recorded, under a consistent placeholder tag rather than being dropped or erroring.

## Open Questions

None. A genuine pass over the design surfaced no uncertainty that can only be resolved once implementation begins — every design fork identified during discovery and architecture was either resolved with the user directly (chosen architecture, session-boundary derivation, response-capture granularity) or settled by reading the actual code and, where a question was cheap to verify directly, by running it against the real binary rather than parking it:

- Whether cobra's own built-in `--help`/`--version` output would bypass the response tee (since it isn't written through `internal/output.Write` like every other response) was checked directly against the built binary: cobra's default help printer also writes through `cmd.OutOrStdout()`, the same writer `runRoot()` tees, so it is captured like any other response with no special-casing needed.
- The one genuine implementer judgment call surfaced while drafting phases — whether `SessionID` should derive from the before- or after-snapshot of workflow state — was resolved during the phases step itself (§ Phase 2.2 in context.md: use the after-snapshot) rather than left open, once the founding-call edge case was reasoned through.
- Sequence-number derivation across process boundaries (each CLI invocation is a separate process with no shared memory) was resolved during phases drafting: derive `Seq` from the existing record file's current line count at write time, accepting that this means each write briefly reads what it has already written — acceptable given the spec's own Non-Goals rule out any retention/rotation concern, so unbounded file growth is already an accepted trade-off, not a new one this introduces.

## Out of Scope

- **Automatic detection or diagnosis of loops or problems.** This plan only produces the record; identifying that an agent got stuck, and where, is left entirely to whichever agent reads the record afterward. (Spec § Non-Goals.)
- **A viewer, UI, or dashboard for browsing records.** The record is a plain local file, read directly by whoever or whatever consumes it — no browsing tool is built. (Spec § Non-Goals.)
- **Automatic redaction or scrubbing of sensitive content from the record.** Keeping the record out of version control (already covered by this plan, § Architecture & Design Decisions) is separate from sanitizing what it contains, and no content-filtering is added. (Spec § Non-Goals.)
- **A retention, rotation, or cleanup policy for old records.** The record file grows without bound for as long as recording stays enabled; nothing in this plan trims, rotates, or expires old entries. This is also why the record deliberately needs no session registry or marker file of its own — see § Data Structures & Interfaces. (Spec § Non-Goals.)
- **Using the record to automatically correct or heal an agent's behavior.** Nothing reads the record back into the running tool; it is write-only from the tool's perspective. (Spec § Non-Goals.)
- **Capturing the agent's own internal reasoning or thought process.** Only the tool's own observable side of each interaction — the command it received and the response it returned — is captured; the tool has no access to an agent's reasoning to capture in the first place. (Spec § Non-Goals.)
- **Any change to `internal/config`'s config schema, or to `internal/workflow`'s public interfaces or persistence format.** The existing `Debug.Enabled` field and the existing `state.json` shape are reused exactly as they are today — this plan adds a new consumer of each, not a new field or a new export. (Committed during this plan's Dependencies and Implementation Detail sections; not a spec Non-Goal, but an explicit scope boundary chosen to keep the change's blast radius minimal.)
- **A new opt-in mechanism (flag or environment variable).** Considered and explicitly rejected during the architecture step in favor of reusing the existing `debug.enabled` config setting — see research.md § Alternatives considered and rejected. Not something this plan adds alongside the reused toggle.

## Changelog

### 2026-07-02 — Phase 1.1: Build the session record writer

**What was done**: Added a new `internal/sessionlog` package with an `Event` struct and a
`Record(logPath string, ev Event)` function that appends one JSON line per completed CLI
invocation to a local log file, deriving each entry's sequence number from the existing line
count of the target file. `Record` has no error return by design — every internal failure
(unwritable directory, marshal error, unwritable file) is swallowed silently so a recording
failure can never surface as a command failure.

**Deviations**: None.

**Files changed**:
- `internal/sessionlog/record.go`
- `internal/sessionlog/record_test.go`

**Discoveries**: Linux reports `ENOTDIR` (not `ENOENT`) when a path component is a file rather
than a directory, so `os.IsNotExist` is the wrong check for that failure mode in tests —
`require.Error` is the correct assertion. This package deliberately does not import
`internal/workflow`, `internal/output`, or `internal/config` — it stays a pure leaf package;
`Event` will grow `SessionID`/`StateBefore`/`StateAfter`/`Advanced` fields in Phases 2.1/2.2, and
`cmd/root.go`'s `runRoot()` (Phase 1.2, next) is where those values get built and this package
gets wired in.

### 2026-07-02 — Phase 1.2: Wire recording into every command, gated by the existing debug option

**What was done**: Extended `cmd/root.go`'s `runRoot()` to optionally record every command's
outcome, gated by `internal/config.Config.Debug.Enabled`. When on, it captures `os.Args[1:]` and a
start time, tees `rootCmd`'s output writer via `io.MultiWriter` so the exact response bytes any
subcommand writes are duplicated into an in-memory buffer without altering what the real stream
receives, and — after `Execute()` returns and any failure has been written through the still-teed
writer — builds a `sessionlog.Event` and appends it via `sessionlog.Record` to
`.spektacular/debug/session-log.jsonl`, then restores the original writer. When the toggle is off
(the default), behavior is byte-for-byte identical to before this phase, save for one additional
`loadConfig()` call whose result is simply discarded when disabled.

**Deviations**: None from the architecture — the config-load-once-at-the-top / tee /
restore-after-failure-write sequence matches context.md's Phase 1.2 notes exactly. One drift note:
context.md's phrasing implied `runRoot()` already had some pre-`Execute()` logic to extend; in
reality it had none — the wiring is greenfield inside the function, not a modification of existing
logic. This didn't change what was built, only how it read during analysis.

**Files changed**:
- `cmd/root.go`
- `cmd/root_test.go`

**Discoveries**: `rootCmd.OutOrStdout()` must be read *before* `Execute()` runs — `orig` cannot be
assumed to be `os.Stdout`, because `cmd/implement_test.go`'s `setupImplementCmd` pre-sets
`rootCmd.SetOut(buf)` before tests call `runRoot()`, and the tee needs to compose with that
override, not bypass it. The tee must stay installed through `output.WriteFailure`'s write too, not
just through `Execute()`, or a failing command's response would never reach the recorded buffer.
`runRoot()`'s recorded `Event.Command` is always the real process `os.Args[1:]` — inside a test
binary this is constant regardless of which subcommand a helper like `runRootCmd` drives via
`rootCmd.SetArgs(...)`, which only affects cobra's own parsing, not `os.Args`. Verification
surfaced two real gaps between the shipped tests and the plan's literal acceptance-criteria
wording (both closed before this entry was written): criterion 2 asks for three distinct outcomes
(succeeds/rejected/errors) — malformed `--data` JSON hits a genuinely different code path
(`cmd/spec.go`'s own `json.Unmarshal` failing, a plain `error`) than a validation-rejected
`*output.ErrorResponse`, so a dedicated "errors" subtest was added; criterion 3 asks for "every
command family," not just one — the byte-identical test was rewritten table-driven across
spec/plan/implement/knowledge/skill. `init` is deliberately excluded from that table: it only
writes `.spektacular/config.yaml` if none already exists, so pre-seeding a debug-on config to
flip the toggle would make the on/off runs asymmetric for a reason unrelated to this feature.

### 2026-07-02 — Phase 2.1: Record whether a command actually advanced anything

**What was done**: Extended `internal/sessionlog.Event` with `StateBefore *StateSnapshot`,
`StateAfter *StateSnapshot`, and `Advanced bool`, and added `StateSnapshot{Kind, Name, CurrentStep,
CompletedSteps}`. In `cmd/root.go`, added `readStateSnapshot()` (reads `.spektacular/state.json`
via the existing `dataDir()`/`stateFilePath()` helpers, returns nil on any error or missing file)
and `stateAdvanced(before, after)` (a pure diff: nil/non-nil mismatch is always an advance; both
nil is never an advance; otherwise compares Kind/Name/CurrentStep/len(CompletedSteps)). `runRoot()`
now captures a state snapshot immediately before `Execute()` and immediately after, and records
both plus the derived `Advanced` flag alongside the existing command/response capture.

**Deviations**: None from the architecture. `internal/sessionlog` still does not import
`internal/workflow` — `cmd` builds the snapshot itself from the raw state file, as planned.

**Files changed**:
- `internal/sessionlog/record.go`
- `internal/sessionlog/record_test.go` (mechanical fix: updated Phase 1.1's
  `TestRecordRoundTripsAllFields` oracle to include the 3 new fields — not new coverage for this
  phase, just keeping a pre-existing test in sync with the struct it exercises)
- `cmd/root.go`
- `cmd/root_test.go`

**Discoveries**: Manually verified end-to-end against the actual documented
`goto-to-current-step-is-a-silent-noop` gotcha using a built binary in a scratch project: `spec new`
→ `advanced=true, state_before=null`; the following `spec goto` to the already-current step →
`advanced=false`, identical before/after snapshots, despite exit code 0 and no error — this is the
concrete proof-of-concept the whole feature exists to enable. This scenario is also independently
corroborated by the pre-existing `internal/workflow/workflow_test.go`'s
`TestGotoSameStepReRenders`, confirming it's a real, reproducible no-op rather than a contrived
test case.

### 2026-07-02 — Phase 2.2: Group recorded entries by the piece of work they belong to

**What was done**: Added `SessionID(state *StateSnapshot) string` to `internal/sessionlog` — a
pure function deriving a session tag from `(Kind, Name)`, falling back to the fixed sentinel
`"no-active-workflow"` when no workflow state exists yet. Added `SessionID string` to `Event`.
Wired `sessionlog.SessionID(stateAfter)` into the `Event` built in `cmd/root.go`'s `runRoot()` —
deliberately from `stateAfter`, not `stateBefore`, so a workflow's founding call is grouped with
its own instance's later calls rather than wrongly sentinel-tagged alone.

**Deviations**: None. The one implementer judgment call the plan flagged as already resolved
during planning (before- vs after-snapshot for `SessionID`) was implemented exactly as decided:
after-snapshot.

**Files changed**:
- `internal/sessionlog/record.go`
- `internal/sessionlog/record_test.go` (mechanical fix: updated `TestRecordRoundTripsAllFields`'s
  oracle again to include `session_id` — same pattern as Phase 2.1's fix, not new coverage for
  this phase)
- `cmd/root.go`
- `cmd/root_test.go`

**Discoveries**: Manually verified end-to-end with a built binary: `knowledge sources` (no
workflow state) → `session_id="no-active-workflow"`; `spec new` (founding call) →
`session_id="spec:000001_billing"`; a following `spec goto` on the same instance → identical
`session_id`. This repo's own default `spec.id_method` is `timestamp`, not `counter` — tests that
need deterministic, assertable resolved spec names must explicitly set `id_method: counter` in
their config fixture.

This was the final phase of the plan. All 4 phases (1.1, 1.2, 2.1, 2.2) are now implemented,
tested, and verified — build/vet/gofmt/full-suite all green with no regressions across any phase.
