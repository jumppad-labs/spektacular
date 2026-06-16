# Research: 000026_ripgrep-replace

## Alternatives considered and rejected

### Option: Pure-Go grep "library" packages

Import an existing grep-as-a-library Go package instead of owning the scan code.

**Rejected**: none viable; confirmed by two independent investigations (plan 000020
research, May 2026 web search; re-confirmed June 2026):

- `github.com/monochromegane/the_platinum_searcher` (pt): dead since Sep 2018 (v2.2.0);
  only entry point is `PlatinumSearcher.Run(args []string) int` — argv-shaped, prints to
  stdout, not a programmatic API; 25 transitive imports.
  https://pkg.go.dev/github.com/monochromegane/the_platinum_searcher
- `github.com/svent/sift`: GPL-3.0 and `package main` — a CLI, not importable.
  https://github.com/svent/sift
- `mvdan.cc/gogrep`: searches Go AST, not text; archived Jun 2021. Wrong category.
- Micro-packages (`rwxrob/grep`, `rastasheep/go-grep`, `itzmanish/grep-go`,
  `tanqiangyes/grep-go`): single-digit-star toys, no binary detection, no scanner-limit
  handling — importing any is worse than ~150 lines of stdlib.
- `u-root` grep, `google/codesearch`, `sourcegraph/zoekt`: `package main` or
  index-build-then-query designs — wrong shape for a live directory walk.

### Option: Full-text index engine (bleve)

`github.com/blevesearch/bleve/v2` — actively maintained full-text indexing engine.

**Rejected**: 28 direct module requires (zapx ×7, RoaringBitmap, go-faiss, protobuf,
bbolt…); analyze→index→query architecture makes plain substring matching *awkward*
(wildcard queries over an inverted index). Categorically overkill for case-insensitive
substring over a small store; plan 000020 already declared "no trigram or inverted index
is built or maintained" out of scope. https://github.com/blevesearch/bleve

### Option: Regexp-based matching

`regexp.MustCompile("(?i)"+regexp.QuoteMeta(q))` — zero-dep, Unicode folding for free.

**Rejected**: spec non-goal forbids new regex capability, and the existing native path's
`strings.ToLower`+`strings.Contains` idiom is already proven equivalent by the rg/native
equivalence test (internal/store/search_test.go:83-108). No reason to switch idioms.

### Option: Keep rg as an optional fast path

Retain the dual-backend dispatch, with the native path as the only guaranteed one.

**Rejected**: explicitly rejected in the spec workflow (user decision recorded in spec
context): clean replacement, external path removed entirely. Removal also satisfies
000025's environment-independence constraint by construction.

### Option: Searcher package extraction

Pull walk/scan/match into a new reusable package (e.g. `internal/grep`).

**Rejected** at the architecture step: speculative — `FileStore.Search`
(internal/store/search.go:31) is the sole consumer, and plan 000025 established the
precedent of keeping search logic inline ("no shared abstraction for the two search
paths") rather than abstracting it.

### Option: Set-layer post-filter for conventions exclusion

Filter hits by path prefix at the knowledge-set layer instead of in the store walk.

**Rejected** back in plan 000025 (research.md): exclusion belongs in `FileStore.Search`
itself; a post-filter would still walk/scan the conventions files wastefully. The native
SkipDir at internal/store/search.go:120-122 is the surviving mechanism.

## Chosen approach — evidence

Promote the existing native path; delete the rg path. Evidence the native path already
satisfies the spec:

- `internal/store/search.go:108-146` (`searchNative`) — `filepath.WalkDir(f.root)`, SkipDir
  on any dir named `conventions` (:120-122), per-line case-insensitive literal match
  (`strings.Contains(strings.ToLower(line), needle)`, :162), one Hit per matching line,
  excerpt via shared `trimExcerpt` (:176-186, 256-byte cap, whitespace collapse, rune-safe).
- `internal/store/search.go:31-41` — dispatch: `forceFallback` seam → `exec.LookPath("rg")`
  → `searchRipgrep`, else `searchNative`. Empty query → `(nil, nil)` (:32-34).
- Equivalence contract: `internal/store/search_test.go:83-108`
  (`TestSearch_RipgrepAndFallbackEquivalent`) proves the two paths agree on
  `{Scope, Path, Excerpt}` over a fixture covering case-insensitivity, nesting, long lines,
  and conventions exclusion — the native path is *already verified* functionally equivalent.
- 000025 constraint (spec §Constraints): results must be identical with or without the
  accelerated tool — the codebase was already required to keep native ≡ rg.
- rg invocation being replaced: `internal/store/search.go:66` — `--json --no-heading
  --fixed-strings --ignore-case --glob=!**/conventions/**` over `f.root`; exit 1 → empty
  (:69-73); Score = per-line submatch count (:97).
- Consumers are backend-agnostic: `internal/knowledge/set.go:87-97` (`Set.Search` fans
  across scope stores, concatenates, fail-fast), `cmd/knowledge.go:117-134`
  (`knowledge search` CLI, `{"hits":[...]}` envelope), skill templates reference only the
  CLI (`templates/skills/workflows/spek-knowledge/SKILL.md:26,49`,
  `templates/skills/skill_spawn-planning-agents.md:18`, `templates/steps/plan/02-discovery.md:18`).
  No consumer observes the backend.

User decisions on the three behavioural deltas (rg path vs native path), resolved
2026-06-11:

1. **Score: replicate rg's per-line match count natively** — count non-overlapping
   case-folded occurrences of the needle per matching line (rg sets
   `float64(len(submatches))`, search.go:97; native currently leaves 0). Preserves what
   rg-equipped users observe today.
2. **File filtering: skip binary files only** — add rg-style NUL-byte binary detection
   (NUL in first ~8000 bytes → skip file, mirroring git/rg convention). Do NOT replicate
   rg's hidden-file/gitignore filtering — knowledge stores are not git working trees.
3. **Long lines: skip the file's remainder, keep searching** — implement the documented
   intent at search.go:20-22 (comment claims skip; code actually aborts via
   `scanner.Err()` at :166-168). Treat `bufio.ErrTooLong` as "skip this file", any other
   scan error still fails the search.

## Files examined

- `internal/store/search.go:1-186` — whole search implementation; dual-path dispatch,
  rg invocation/parsing, native walk, `trimExcerpt`. Stale comments: :20-22 (claims
  long-line skip; code aborts), :119 (cites broken glob form `!conventions/**`; working
  form is `!**/conventions/**` per plan 000025).
- `internal/store/store.go:23-28` — `Hit{Scope, Path, Excerpt, Score}` JSON-tagged
  `scope/path/excerpt/score`; :27 documents Score "0 when the backend has none".
- `internal/store/store.go:49` — `Store` interface `Search(query string) ([]Hit, error)`.
- `internal/store/store.go:54-61` — `forceFallback` test seam on `FileStore`; becomes
  meaningless once only one path exists — remove with the rg path.
- `internal/store/search_test.go:14-17,22-37` — fixture `writeSearchFixture`: t.TempDir +
  production `FileStore.Write`; covers match, no-match, nested uppercase NEEDLE
  (case-insensitivity oracle), >256-byte line, `conventions/style.md` (exclusion oracle).
- `internal/store/search_test.go:42-54` — equivalence projection deliberately excludes
  Score (only rg path sets it).
- `internal/store/search_test.go:57-79` — excerpt budget + trimExcerpt unit tests
  (backend-independent; survive unchanged).
- `internal/store/search_test.go:83-108,135-150` — rg-guarded tests
  (`TestSearch_RipgrepAndFallbackEquivalent`, `TestSearch_RipgrepEmptyResults`); skip when
  rg absent; obsolete once the rg path is deleted — their fixture assertions (conventions
  exclusion, no-match-empty, empty-query-empty) are the semantics to retain in replacements.
- `internal/store/search_test.go:112-131` — `TestSearch_ScopeAndLocatorRoundTrip` (forced
  fallback; survives, drop the forceFallback usage).
- `internal/knowledge/set.go:57-97` — `NewSet` (one FileStore per configured scope,
  fail-fast stat), `Set.Search` fan-out in configured order, error wrap
  `"searching knowledge source %q"`.
- `internal/knowledge/set_test.go:25-116` — two-scope fixtures via t.TempDir +
  config.SourceConfig; scope-tagging assertions; backend-agnostic, survive unchanged.
- `cmd/knowledge.go:20-25,57-60,105-134` — `knowledge search` command, output schema
  declaration, `{"hits":[...]}` envelope, nil→`[]store.Hit{}` normalisation.
- `cmd/knowledge_test.go:12-18,52-85,138-158` — CLI integration test pins envelope field
  names; runs whichever backend the host has (currently rg on dev machines) — after the
  change it deterministically exercises the native path.
- `internal/config/config.go:23,33,36,209-237` — knowledge source config; only
  `ProviderFile` supported; default `project` scope at `.spektacular/knowledge`.
- `README.md:212` — documents "prefers ripgrep with native fallback"; needs updating.

## External references

- ripgrep GUIDE.md / rg(1) (https://github.com/BurntSushi/ripgrep/blob/master/GUIDE.md) —
  reference model: binary = file contains NUL byte; `-i` is Unicode *simple* case folding;
  `-F` literal. Confirms NUL-check + case-folded substring is a faithful semantic clone.
- git `buffer_is_binary()` — checks first 8000 bytes for `\x00`
  (https://www.codestudy.net/blog/how-to-determine-if-git-handles-a-file-as-binary-or-as-text/) —
  the binary-detection convention to copy.
- bufio.Scanner docs + golang-nuts (https://pkg.go.dev/bufio) — `bufio.ErrTooLong` is
  unrecoverable for a Scanner: after it fires the scanner stops; "skip and continue within
  the same scanner" is impossible — skip must operate at file granularity (or switch to
  bufio.Reader). Shaped decision #3 (skip file's remainder).
- golang/go#52022 (https://github.com/golang/go/issues/52022) — stdlib has no
  `strings.ContainsFold`; `ToLower`-both-sides is the pragmatic idiom, not exact Unicode
  simple folding (Turkish İ, ς/σ edge cases). Current native path already uses it and the
  equivalence test passes — acceptable fidelity, no change of idiom needed.
- pkg.go.dev pages for pt/sift/gogrep/bleve (URLs in Alternatives) — library rejection
  evidence.
- "Beating grep with Go" (healeycodes.com) — cited by plan 000020 as confirmation the
  native path had to be hand-written; no library to pull.

## Prior plans / specs consulted

(read via `go run . plan file read` / `go run . spec file read`)

- **000020_context** (spec + plan + research) — origin of the rg choice and of
  `searchNative`. Learned: rg chosen for speed with native fallback *required* so there is
  "no hard runtime dependency on an external binary"; `--fixed-strings --ignore-case`
  added during implementation so rg matches the native literal scan; Score equivalence
  deliberately excluded from tests; rg exit 1 = no matches; "knowledge directories are
  small" recorded as the justification that walk-and-scan performance is acceptable;
  no-index/no-ranking explicitly out of scope; `forceFallback` chosen over a package-level
  lookPath var as the test seam.
- **000025_convention-aware-planning** (spec + plan + research) — conventions exclusion in
  both paths, deliberately duplicated ("two implementations, one equivalence test"), no
  shared abstraction. Gotcha: rg glob `!conventions/**` did NOT work; `!**/conventions/**`
  is the verified form. Constraint: identical results whether or not the accelerated tool
  is present — pre-authorises this replacement.
- **000026_ripgrep-replace** (spec, the subject) — clean replacement, zero external
  process, functional parity incl. conventions exclusion; non-goals: no regex/fuzzy/ranking,
  performance not a goal; success metrics: clean-machine search works, existing
  search-related tests pass.
- **000022_spek-knowledge-skill / 000023_context-over-memory** — consumers only; skill
  layer sees only the CLI envelope, unaffected by a backend swap.

## Open assumptions

- **Knowledge stores stay small** (carried from 000020): walk-and-scan per query with no
  index remains acceptable. If implement finds a perf-critical caller, STOP and ask.
- **`strings.ToLower`-based matching is "the same case-insensitivity" the spec demands.**
  It is not exact Unicode simple folding (rg's `-i`), but it is what the current native
  path does and the equivalence test passes on the shared fixtures. Assumed acceptable; if
  a reviewer demands exact rg folding semantics, that's a `unicode.SimpleFold` rune-scan
  (~30 lines) — STOP and ask before adding it.
- **No store content depends on rg's hidden-file/gitignore filtering** (user decided not
  to replicate it). Assumed knowledge stores contain no hidden/gitignored files whose
  sudden appearance in results would surprise. Fixtures contain none today.
- **Score consumers tolerate the native count.** Per-line non-overlapping occurrence count
  ≈ rg's submatch count for fixed strings; assumed no consumer depends on rg's exact
  submatch arithmetic (overlapping needles like "aa" in "aaa" differ; rg counts
  non-overlapping too, so expected identical — unverified against rg edge cases).
- **`cmd/knowledge_test.go` passes unchanged** once the backend is native-only (it asserts
  envelope shape and excerpt content, not Score values). Verified by reading assertions,
  not by running without rg.

## Rehydration cues

- `go run . spec file read 000026_ripgrep-replace.md` — the spec (source of truth).
- Read `internal/store/search.go` and `internal/store/search_test.go` in full (~190 +
  ~150 lines) — the entire battleground.
- `go run . plan file read 000020_context/research.md` and
  `go run . plan file read 000025_convention-aware-planning/plan.md` — why rg existed and
  why `!**/conventions/**`.
- `go run . knowledge conventions` — currently returns `{"conventions": []}` (empty; no
  project conventions to honour).
- Re-check consumers: `internal/knowledge/set.go:87-97`, `cmd/knowledge.go:117-134`.
- User decisions of 2026-06-11 (Score=native match count; binary-skip only; long line
  skips file) are recorded above and in `.spektacular/context.md`.
