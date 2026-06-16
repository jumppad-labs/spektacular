# Context: 000026_ripgrep-replace

## Current State Analysis

Search is implemented once, in `FileStore.Search` (internal/store/search.go:31-41). It
probes PATH for `rg` per call (`exec.LookPath`, :36); if found it shells out via
`searchRipgrep` (:62-104) with `--json --no-heading --fixed-strings --ignore-case
--glob=!**/conventions/**` over the store root (:66), decoding the NDJSON event stream
into `rgEvent` (:43-60); otherwise it falls back to `searchNative` (:108-146) — a
`filepath.WalkDir` + per-line `strings.Contains(strings.ToLower(line), needle)` scan
(:162) with `SkipDir` on any directory named `conventions` (:120-122). Both paths share
`trimExcerpt` (:176-186; whitespace collapse, 256-byte cap on a rune boundary).

Key facts about the starting point:

- The native path is already certified functionally equivalent to the rg path on
  `{Scope, Path, Excerpt}` by `TestSearch_RipgrepAndFallbackEquivalent`
  (internal/store/search_test.go:83-108). The equivalence projection (:42-54) deliberately
  excludes `Score` — only the rg path sets it (`float64(len(submatches))`, search.go:97);
  the native path leaves it 0.
- rg exit code 1 ("no matches") maps to an empty result (:69-73); other failures wrap as
  `"ripgrep search: %w"`. Empty query short-circuits to `(nil, nil)` (:32-34).
- The test seam is an unexported `forceFallback bool` on `FileStore`
  (internal/store/store.go:54-61), used by fallback-pinned tests.
- **Stale comment #1**: search.go:20-22 claims lines longer than `scanBufferBytes` (1MiB)
  "are skipped rather than aborting the whole scan" — false. `scanFile` returns
  `scanner.Err()` (:166-168), so `bufio.ErrTooLong` aborts the entire search via the walk
  callback (:125-128).
- **Stale comment #2**: search.go:117-119 cites the glob form `!conventions/**`, which
  plan 000025 verified does NOT work in rg; the working form `!**/conventions/**` is what
  the code actually passes.
- Undocumented asymmetry: the rg path inherits rg's default gitignore/hidden-file/binary
  filtering; the native path scans everything. Fixtures never exercise this. User decided:
  replicate binary skip only.
- Consumers are backend-agnostic: `Set.Search` (internal/knowledge/set.go:87-97) fans
  across scope stores in configured order, fail-fast with
  `"searching knowledge source %q: %w"`; CLI `knowledge search` (cmd/knowledge.go:117-134)
  emits the `{"hits":[...]}` envelope, normalising nil → `[]store.Hit{}`; skill templates
  reference only the CLI command.
- No rg install checks, version checks, or doctor probes exist anywhere; `exec.LookPath`
  at search.go:36 is the only detection, re-run per call.
- `internal/store/search.go` contains the codebase's only subprocess use.

## Per-Phase Technical Notes

### Phase 1.1: Native relevance scoring

**File changes**

- `internal/store/search.go:148-170` — widen `scanFile`'s return from `[]string` (matched
  lines) to matched lines with per-line counts; add a small unexported type, e.g.
  `type lineMatch struct { text string; count int }`, or return parallel slices — prefer
  the struct. Count = non-overlapping occurrences via `strings.Count(strings.ToLower(line), needle)`
  (needle is already lower-cased by the caller, search.go:109). Note `strings.Count`
  counts non-overlapping occurrences — matches rg's submatch arithmetic for fixed strings.
- `internal/store/search.go:129-139` — in `searchNative`, populate `Score: float64(m.count)`
  when building each `Hit` (today the field is omitted, defaulting to 0).
- `internal/store/store.go:27` — update `Hit.Score` doc comment from "0 when the backend
  has none" to the new invariant (per-line occurrence count) — make the semantic change
  here; final wording pass happens in Phase 2.2.
- `internal/store/search_test.go` — new test, e.g. `TestSearch_ScoreCountsLineOccurrences`:
  own fixture in `t.TempDir()` written via `FileStore.Write` (per repo test convention —
  tests own their filesystem), with a file whose line contains the needle twice (score 2),
  once (score 1), and once in different case (score 1). Hand-maintained expected scores —
  do NOT derive them by running the matcher (independent-oracle convention).

**Guard rails**

- `TestSearch_RipgrepAndFallbackEquivalent` (search_test.go:83-108) must stay green: its
  projection (:42-54) excludes Score, so no change needed there — do not add Score to the
  projection in this phase.
- Do not touch the rg path; both backends remain live until Phase 2.1.

**Complexity**: Low
**Token estimate**: ~15k
**Agent strategy**: Single agent, sequential execution.

### Phase 1.2: Resilient file handling — binary skip and long-line recovery

**File changes**

- `internal/store/search.go:150-170` (`scanFile`) — two additions:
  1. **Binary sniff**: after `os.Open`, read up to 8000 bytes (git's `FIRST_FEW_BYTES`
     convention, mirrors rg's NUL-byte binary detection); if `bytes.IndexByte(buf, 0) >= 0`,
     return no matches, nil error (quiet skip). Then continue scanning from the start —
     either `file.Seek(0, io.SeekStart)` before constructing the scanner, or wrap with
     `io.MultiReader(bytes.NewReader(buf), file)` to avoid the seek. Either is fine; pick
     one and keep it simple.
  2. **Long-line recovery**: at the `scanner.Err()` check (:166-168), treat
     `errors.Is(err, bufio.ErrTooLong)` as end-of-file — return the matches collected so
     far with nil error (a Scanner is unrecoverable after ErrTooLong, so the skip is
     file-granularity by necessity). All other scanner errors still return the error
     (fail-fast contract preserved; walk-level errors at search.go:113-115 and open errors
     at :151-153 unchanged).
- `internal/store/search.go:20-22` — rewrite the `scanBufferBytes` comment: it currently
  claims lines longer than the buffer "are skipped rather than aborting the whole scan",
  which is false today; after this phase the accurate wording is "a file containing a
  longer line is scanned up to that line and then skipped".
- `internal/store/search_test.go` — two new tests with their OWN fixtures (do not extend
  `writeSearchFixture`, :22-37 — a >1MiB line would make the rg and native paths diverge
  while the equivalence test is still alive, and rg-skipped binaries are incidental there):
  - `TestSearch_SkipsBinaryFiles`: store with a NUL-containing file that includes the
    needle (must NOT hit) and a sibling text file with the needle (must hit). Write the
    binary fixture through `FileStore.Write` (content is `[]byte`-compatible via string
    with `\x00`).
  - `TestSearch_OversizedLineSkipsFileNotSearch`: store with one file containing a
    matching line followed by a >1MiB (`scanBufferBytes`) line, plus a normal matching
    sibling file. Assert: search returns nil error; sibling's hit present; first file's
    pre-long-line match present.
  - Both run via `forceFallback` (store.go:54-61) while it still exists, mirroring
    `TestSearch_ScopeAndLocatorRoundTrip` (:112-131); Phase 2.1 removes the flag from
    these tests.

**Complexity**: Low
**Token estimate**: ~20k
**Agent strategy**: Single agent, sequential execution.

### Phase 2.1: Remove the external tool integration

**File changes**

- `internal/store/search.go`:
  - Delete the rg dispatch in `Search` (:35-39) — body becomes empty-query check then a
    direct call to the walk/scan (optionally inline `searchNative` into `Search` or keep
    the helper; keep the helper name meaningful, e.g. rename `searchNative` → `search` or
    fold in — implementer's choice, smallest clear diff wins).
  - Delete `rgEvent` (:43-60) and `searchRipgrep` (:62-104) entirely.
  - Drop now-unused imports: `encoding/json`, `os/exec`; re-check `errors` and `bytes` —
    Phase 1.2 added `errors.Is` and `bytes.IndexByte` uses, so both likely stay. Verify
    with goimports.
  - Fix stale comment :117-119 — remove the "Mirrors the ripgrep --glob=!conventions/**
    exclusion" sentence (the glob form cited is the broken one anyway, per plan 000025).
- `internal/store/store.go:54-61` — delete the `forceFallback` field and its doc comment
  from `FileStore`.
- `internal/store/search_test.go`:
  - Delete `TestSearch_RipgrepAndFallbackEquivalent` (:83-108) and
    `TestSearch_RipgrepEmptyResults` (:135-150).
  - Re-home their backend-independent guarantees as direct tests over the existing
    `writeSearchFixture` (:22-37): case-insensitive matching finds `nested/deep.txt`'s
    uppercase NEEDLE; no hit path starts with `conventions/`; no-match query → empty +
    nil error; empty query → empty + nil error. (Empty/no-match partially covered by
    `TestSearch_ScopeAndLocatorRoundTrip` :112-131 — extend or add ONE direct test, do not
    duplicate assertions that existing tests already make through the same mechanism.)
  - Remove every `forceFallback = true` line (fixture helpers and tests from Phases
    1.1/1.2, plus :58-69, :112-131 usages).
- `cmd/knowledge_test.go:138-158` — no code change; after this phase it deterministically
  exercises the native path. Verify it passes.
- Repo-wide check: `os/exec` must no longer be imported anywhere in `internal/store`
  (it was the package's only subprocess use; `grep -r "os/exec" internal/` should hit
  nothing).

**Complexity**: Medium (deletion + test re-homing touches two source files and one test
file with interleaved concerns)
**Token estimate**: ~25k
**Agent strategy**: Single agent, sequential execution — the deletions and test rework are
tightly coupled to one package; parallel agents would conflict in the same files.

### Phase 2.2: Documentation matches the built-in search

**File changes**

- `README.md:212` — replace "Search prefers the ripgrep binary when present and falls
  back to a native Go scan" (current wording approximate; read before editing) with a
  description of the built-in search: in-process walk, case-insensitive literal matching,
  conventions excluded, binary files skipped.
- `internal/store/search.go:24-30` — rewrite `Search`'s doc comment: drop the dual-path /
  "no caller can observe which ran" contract; document in-process scanning, empty-query
  semantics, conventions exclusion, binary skip, long-line file skip, and score semantics.
- `internal/store/store.go:23-28` — final wording pass on `Hit.Score` doc (see Phase 1.1;
  state "per-line count of case-insensitive query occurrences").
- Sweep `internal/store/` comments for remaining rg/ripgrep mentions
  (`grep -rn -i ripgrep internal/store/` → only historical plan/spec docs should mention
  it afterwards).

**Complexity**: Low
**Token estimate**: ~10k
**Agent strategy**: Single agent, sequential execution.

## Testing Strategy

Overall: regression via the untouched backend-independent suite; reworked direct tests
inherit the retired differential tests' guarantees; new direct tests cover the three new
behaviours. All tests: Go `testing` + `testify/require`, fixtures in `t.TempDir()` via the
production write path, hand-maintained oracles.

Per phase:

- **Phase 1.1** — `TestSearch_ScoreCountsLineOccurrences` (new, own fixture): expected
  scores 2 / 1 / 1 hand-written. Regression guard: full suite incl. the still-alive
  equivalence test (Score excluded from its projection).
- **Phase 1.2** — `TestSearch_SkipsBinaryFiles` and
  `TestSearch_OversizedLineSkipsFileNotSearch` (new, own fixtures, forced fallback while
  the seam exists). Must NOT extend `writeSearchFixture` — a >1MiB line diverges rg vs
  native while the equivalence test is alive. Regression guard: full suite.
- **Phase 2.1** — delete the two rg-conditional tests; add/extend direct tests over
  `writeSearchFixture` for case-insensitivity and conventions exclusion; avoid duplicating
  empty/no-match assertions already made by `TestSearch_ScopeAndLocatorRoundTrip`. After
  this phase no test skips on missing rg and no test references the seam.
  `cmd/knowledge_test.go` passes unchanged, now deterministically on the native path.
- **Phase 2.2** — docs only; full suite is the regression guard.

Spec success metrics → verification mapping (carried from plan.md § Testing Approach):
metric 1 (zero external deps) = behavioural via the in-process suite + structural (no
`os/exec` import in `internal/store`); metric 2 (no regression) = behavioural via the
unchanged backend-independent tests, with retired differential tests' guarantees traced to
their re-homed direct tests. No manual test-plan items.

## Project References

- Spec (source of truth): `000026_ripgrep-replace.md` in the spec store — read via
  `go run . spec file read 000026_ripgrep-replace.md`.
- Prior plan **000020_context** (plan store: `000020_context/{plan,context,research}.md`) —
  built the search feature; records why rg was chosen, the `--fixed-strings --ignore-case`
  gotcha, the "knowledge directories are small" assumption, and the no-index non-goal.
- Prior plan **000025_convention-aware-planning** (plan store) — conventions-exclusion
  semantics; the `!**/conventions/**` vs `!conventions/**` glob gotcha.
- Implementation surface: `internal/store/search.go`, `internal/store/store.go`,
  `internal/store/search_test.go`; consumers `internal/knowledge/set.go`,
  `cmd/knowledge.go`, `cmd/knowledge_test.go`; docs `README.md:212`.
- External: ripgrep GUIDE.md (binary detection / `-i` semantics,
  https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md); git's 8000-byte
  `buffer_is_binary()` convention; golang/go#52022 (no `strings.ContainsFold`).
- Project conventions store: empty at planning time (no conventions to honour).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase estimates: 1.1 ~15k (Low), 1.2 ~20k (Low), 2.1 ~25k (Medium, but single-agent —
tightly coupled deletions in one package), 2.2 ~10k (Low). Total ~70k.

## Migration Notes

None. No data migration, no config migration, no breaking CLI changes. Users with rg
installed lose nothing; rg simply stops being consulted. The `Hit.Score` JSON field keeps
its name and type; its value source changes from rg submatch count to the native
occurrence count (identical for fixed strings).

## Performance Considerations

Performance is an explicit spec non-goal. The native walk-and-scan runs per query with no
index or cache, resting on plan 000020's recorded assumption that knowledge directories
are small. The binary sniff adds one ≤8000-byte read per file — negligible. If a future
store grows large enough that scan latency matters, that is a new spec (indexing was
explicitly excluded in 000020 and remains out of scope here).
