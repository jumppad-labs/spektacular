# Working context — plan 000026_ripgrep-replace

PLAN WORKFLOW COMPLETE (2026-06-11). All three documents committed to the plan store via
`go run . plan file write`: 000026_ripgrep-replace/{plan,context,research}.md. Working
directory .spektacular/work/000026_ripgrep-replace/ removed after successful writes.

## Key decisions (carried forward for implement workflow)
- Direction: in-place promote-and-delete — remove the rg path (searchRipgrep, rgEvent,
  LookPath dispatch, forceFallback seam), promote the existing native scan; no new
  packages, no library (none viable), no consumer changes.
- Three user-decided behaviours (2026-06-11):
  1. Score = per-line count of non-overlapping case-insensitive query occurrences
     (replicates rg's submatch count; native currently emits 0).
  2. Binary files skipped via NUL byte in first ~8000 bytes (git/rg convention); rg's
     hidden-file/gitignore filtering deliberately NOT replicated.
  3. bufio.ErrTooLong skips the rest of that file and search continues (today it aborts
     the whole search; comment at search.go:20-22 was stale/false).
- 4 phases / 2 milestones: 1.1 scoring, 1.2 binary+long-line resilience (own fixtures —
  do NOT extend the shared equivalence fixture while the rg path is alive), 2.1 rg-path
  removal + test re-homing, 2.2 docs (README:212, stale comments incl. search.go:119
  broken-glob citation).
- Open questions: none (healthy empty); STOP-and-ask guard on research.md § Open
  assumptions (e.g. consumer depending on rg Score arithmetic).
- Project conventions store empty — none apply.

## Learnings
- Workflow note: a Bash `cd` into .spektacular/tmp persisted across calls and broke
  `go run .` (no Go files); always run plan CLI commands from the repo root.
- Verification caught: missing `## Project References` in assembled context doc, and a
  shell-command mention in plan.md § Conventions (not allowed in plan.md).
