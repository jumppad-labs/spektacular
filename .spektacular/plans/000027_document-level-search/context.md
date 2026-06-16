# Context: 000027_document-level-search

## Current State Analysis

The built-in scanner that plan 000026_ripgrep-replace promoted to sole search
implementation is the starting point; it sits uncommitted on branch
`f-ripgrep-replace` and was proven byte-identical to the old ripgrep backend
on 14 queries — the limitation motivating this plan is the matching model,
not the scanner.

- `internal/store/search.go:35-40` — `Search` entry: empty query →
  `(nil, nil)`; the WHOLE query string is one lower-cased needle (no
  tokenization, search.go:45).
- `internal/store/search.go:44-83` — `search`: `filepath.WalkDir` from store
  root; `conventions` dirs `SkipDir` (52-59); per-line fan-out loop builds
  one `Hit` per matching line (65-76) with `Score = float64(line count)`.
- `internal/store/search.go:87-131` — `lineMatch{text, count}` and
  `scanFile`: binary sniff of first 8000 bytes for NUL (103-111); scanner
  buffer 64 KiB initial / 1 MiB max (115); per-line
  `strings.Count(strings.ToLower(line), needle)` (118); on
  `bufio.ErrTooLong` keeps matches collected so far and skips the file's
  remainder without error (122-127).
- `internal/store/search.go:137-147` — `trimExcerpt`: whitespace collapse +
  256-byte cap (`maxExcerptBytes`, search.go:17) on a UTF-8 boundary.
- `internal/store/store.go:23-28` — `Hit{Scope, Path, Excerpt, Score}`;
  `Path` documented "pass to Read". `Store` interface includes `Search`
  (store.go:49). `Read` at store.go:85-95 (traversal-rejecting `abs`,
  `ErrNotFound`).
- `internal/knowledge/set.go:87-97` — `Set.Search` concatenates per-source
  hits in config order, documented "no ranking or dedup"; `Set.Read` at
  set.go:100-106; conventions served whole via set.go:141-160.
- `cmd/knowledge.go:20-25, 117-134` — `search <query>` (ExactArgs(1)),
  `{"hits": [...]}` envelope, nil→`[]store.Hit{}`; output schemas at 57-69;
  `read` at 136-158 with required scope+path (234-254).
- Non-test callers of `Search`/`store.Hit` are exactly: `Set.Search`
  (set.go:87-97) and `runKnowledgeSearch` (cmd/knowledge.go:125-133) —
  grep-verified; `internal/output/writer.go:24-88` serialises generically.
- Live-knowledge-base evidence (spec workflow): query "workflow" → 60
  per-line hits across 9 documents, 56 with score 1, 15 hits from raw JSON
  transcript files; "workflow steps" → 0 hits.

## Per-Phase Technical Notes

### Phase 1.1: Per-document aggregation and matching

**File changes**

- `internal/store/store.go:23-28` — reshape `Hit`: keep `Scope`/`Path`
  unchanged (doc comment "pass to Read" stays); add
  `Title string \`json:"title"\``; replace `Excerpt string` with
  `Excerpts []string \`json:"excerpts"\``; rewrite the `Score` comment to
  "sum of all query terms' case-insensitive occurrences across the
  document". Interface at `store.go:49` keeps its signature.
- `internal/store/search.go:27-40` — rewrite `Search` doc comment for
  document-level semantics; tokenize with `strings.Fields(strings.ToLower(query))`
  and short-circuit zero terms to `(nil, nil)` (covers empty and
  whitespace-only).
- `internal/store/search.go:44-83` — `search` takes the term slice; the
  WalkDir callback keeps the `conventions` SkipDir (52-59) and Rel fallback
  (66-69) untouched; replace the per-line fan-out loop (65-76) with: skip
  file unless every term's count > 0, else append one `Hit` with
  `Score = float64(sum of term counts)`, `Title = rel` (locator fallback
  only in this phase), `Excerpts =` first matching lines run through
  `trimExcerpt` (refined in Phase 1.2).
- `internal/store/search.go:85-131` — replace `lineMatch` with a per-file
  aggregate (e.g. `fileAggregate{counts []int, lines []candidateLine}`
  indexed by term position); `scanFile(path string, terms []string)` counts
  `strings.Count(loweredLine, term)` per term per line (118), accumulates
  totals, and records matching lines in file order. Binary sniff (103-111),
  buffer limits (115), and the `bufio.ErrTooLong` keep-and-skip return
  (122-127) are unchanged — on ErrTooLong return the aggregate collected so
  far.
- `internal/store/search_test.go` — adapt to the new shape:
  `TestSearch_ExcerptWithinBudget` (39-51) iterates `Excerpts`;
  round-trip (64-82), case/conventions (86-114), binary (147-166, exact
  path slice still `[]string{"plain.txt"}`), oversized (171-190) all keep
  their assertions against `Hit.Path` sets.
  `TestSearch_ScoreCountsLineOccurrences` (119-143) becomes a per-document
  oracle: `twice.txt` (2 occurrences on one line) → one hit score 2, plus a
  new fixture file with occurrences spread over lines → one hit with summed
  score. Add: scattered multi-word match (words never co-located on a
  line), missing-word exclusion, exactly-one-hit-per-file with words on
  many lines, and a single-word regression test whose hand-maintained
  oracle lists the expected file set for the shared fixture
  (`writeSearchFixture`, 21-36). Follow the package's conventions:
  fixtures in `t.TempDir()`, hand-maintained expected values, no redundant
  assertions.

**Complexity**: Medium
**Token estimate**: ~30k
**Agent strategy**: 2 parallel agents — one rewrites `search.go` +
`store.go`, one prepares the test rewrite against the agreed `Hit` shape —
then sequential integration to reconcile and run the package tests.

### Phase 1.2: Titles and best-line excerpts

**File changes**

- `internal/store/search.go` (scan loop, currently 116-121) — while
  scanning, capture the first line whose trimmed form starts with `#`
  followed by a space or only `#`s (ATX heading), strip the leading `#`s
  and whitespace, store as the aggregate's title; lines after an over-long
  line are unreachable, so a late heading falls back to locator (accepted
  in plan architecture).
- `internal/store/search.go` (aggregate + Hit construction) — track per
  candidate line its distinct-term count and total occurrences; select
  excerpts by distinct terms desc, total occurrences desc, file order asc;
  introduce `const maxExcerptsPerHit = 3` beside `maxExcerptBytes`
  (search.go:17); keep `trimExcerpt` (137-147) as the single budget
  enforcement point. Bound the candidate set as the scan proceeds (keep
  only the current top candidates) so memory stays line-bounded, matching
  the scanner's bounded-memory property.
- `internal/store/search.go` Hit construction — `Title` = captured heading
  text, else the locator (`rel`).
- `internal/store/search_test.go` — add: title from first heading
  (fixture md file with heading not on the matching line); locator
  fallback (fixture without any heading, e.g. JSON-like content);
  excerpt preference (a line with two distinct terms beats two lines with
  one each); excerpt cap (file with >3 matching lines yields exactly 3
  excerpts); budget already covered by the adapted
  `TestSearch_ExcerptWithinBudget` — do not re-assert (no redundant
  assertions).

**Complexity**: Low
**Token estimate**: ~15k
**Agent strategy**: Single agent, sequential execution.

### Phase 2.1: Globally ranked results across all knowledge sources

**File changes**

- `internal/knowledge/set.go:84-97` — `Set.Search`: collect hits per source
  remembering the source's configured index; after the loop,
  `sort.SliceStable` by `Score` desc, then source index asc, then `Path`
  asc; replace the "It performs no ranking or dedup" doc comment (84-86)
  with the ordering contract. Error propagation (92) unchanged.
- `internal/knowledge/set_test.go` — add ranking tests: two sources with
  differing scores → stronger first regardless of source order; equal
  scores → source config order then path, stable across repeated calls.
  Keep existing fan-out/scope-tagging assertions; the store-level
  round-trip test already guards `Read` — assert the set-level
  search→`Set.Read` round trip only if not already covered there (check
  before adding; no redundant assertions).

**Complexity**: Low
**Token estimate**: ~10k
**Agent strategy**: Single agent, sequential execution.

### Phase 2.2: The search command speaks the new result shape

**File changes**

- `cmd/knowledge.go:57-60` — update the search output schema: `hits` items
  described as per-document results with `scope`, `path`, `title`, `score`,
  `excerpts` (array). Read schemas (62-69) untouched.
- `cmd/knowledge.go:20-25` — update the search subcommand's `Short`/help
  wording to "ranked, one result per matching document". Envelope and
  nil→`[]store.Hit{}` normalisation (117-134) unchanged.
- `cmd/knowledge_test.go` — update expected JSON for search output (new
  fields, ranked order); keep read-command tests untouched as the contract
  guard.

**Complexity**: Low
**Token estimate**: ~8k
**Agent strategy**: Single agent, sequential execution.

### Phase 3.1: Shipped documentation moves in lockstep

**File changes**

- `templates/skills/workflows/spek-knowledge/SKILL.md:26-28,49-50` —
  describe ranked per-document hits (scope, path, title, score, excerpts)
  and the unchanged search→read chaining; keep the `{{command}}`
  placeholder convention (never the rendered `go run .`).
- `.claude/skills/spek-knowledge/SKILL.md` — the rendered copy in this
  repo: regenerate via the project's init/render path rather than hand
  editing, so template and copy cannot drift.
- `templates/steps/plan/02-discovery.md:9,18` — update the description of
  search results (scope-tagged stays true; now ranked per-document).
- `templates/skills/skill_spawn-planning-agents.md:18` — verify the "hits
  are tagged by scope" wording still holds (it does) and adjust any
  per-line phrasing.
- `README.md:164-165` — command table: search row describes ranked
  per-document results; read row unchanged.
- `CHANGELOG.md` — entry under the current unreleased heading describing
  document-level search (note: this file already carries uncommitted
  000026 entries on this branch).
- `internal/agent/instruction_surface_test.go:94` and
  `internal/steps/plan/steps_test.go` — confirm string-presence assertions
  still pass after template edits; update expectations only if wording
  they pin has changed.
- Sweep check: `grep -ri "excerpt\b\|per-line\|one result per" templates/
  README.md .claude/skills/ cmd/` (and the `templates/templates.go` embeds
  compile) to confirm no shipped surface still describes the old shape.

**Complexity**: Low
**Token estimate**: ~12k
**Agent strategy**: Single agent, sequential execution (the files are
interdependent in wording; one agent keeps the vocabulary consistent).

## Testing Strategy

Conventions for every phase: fixtures live in the test's own `t.TempDir()`
(never the live knowledge base), expected values are hand-maintained
oracles, and a behaviour fully guarded at a lower layer is not re-asserted
at a higher one.

- **Phase 1.1** carries the matching-model load: scattered multi-word
  match, missing-word exclusion, one-hit-per-document, hand-computed
  document scores, the single-word regression oracle over the shared
  fixture, and the preserved edge cases (empty/whitespace query,
  conventions exclusion, binary skip, over-long-line resilience) — all in
  `internal/store/search_test.go`.
- **Phase 1.2** adds title and excerpt assertions in the same file:
  heading title, locator fallback, distinct-term preference, the 3-excerpt
  cap; the byte budget is already asserted by the adapted excerpt-budget
  test.
- **Phase 2.1** asserts the ranking contract in
  `internal/knowledge/set_test.go`: score-descending across sources,
  deterministic tie-break (source order, then path), stability across
  runs; search→read round trip checked once at the appropriate layer.
- **Phase 2.2** asserts the wire contract in `cmd/knowledge_test.go`: new
  fields in the `hits` envelope, schema/emission agreement, empty list
  (not error) for empty and no-match queries.
- **Phase 3.1** relies on the existing rendered-template tests
  (`internal/agent/instruction_surface_test.go`,
  `internal/steps/plan/steps_test.go`) plus a manual grep sweep for stale
  per-line descriptions.
- **Success metrics**: mechanisms are covered by the fixture tests above;
  the three live-knowledge-base outcomes ("workflow steps"/"workflow
  state"/"interrupted workflow" ranked first; "workflow" single-digit
  ranked results; single-word discovery set unchanged) are **manual —
  captured in the implementation test plan**, per the plan's Testing
  Approach.

## Project References

- Spec (source of truth): `go run . spec file read
  000027_document-level-search.md` — requirements, constraints, non-goals,
  success metrics.
- Predecessor plan: `go run . plan file read 000026_ripgrep-replace/plan.md`
  (+ `research.md`, `context.md`) — scanner design rationale and the
  hardened edge-case behaviours this plan inherits.
- Decision log for this plan: `research.md` in this plan directory
  (`go run . plan file read 000027_document-level-search/research.md`) —
  rejected alternatives, evidence, open assumptions, rehydration cues.
- Conventions check: `go run . knowledge conventions` returned `[]` — no
  project conventions exist.
- Origin of the conventions-exclusion behaviour:
  spec/plan 000025_convention-aware-planning.
- Key packages: `internal/store` (engine + `Hit`), `internal/knowledge`
  (Set fan-out/ranking + read contract), `cmd` (knowledge subcommands),
  `internal/output` (generic JSON writer, untouched), `templates/`
  (shipped instruction surfaces).

## Token Management Strategy

| Tier | Token Budget | Agent Strategy |
|------|-------------|----------------|
| Low | ~10k | Single agent, sequential |
| Medium | ~25k | 2-3 parallel agents |
| High | ~50k+ | Parallel analysis, sequential integration |

Phase 1.1 is the only Medium phase (~30k); all other phases are Low
(~8-15k). Total estimate ~75k across five phases.

## Migration Notes

No data migration. The `Hit` JSON shape changes (`excerpt` → `excerpts`,
new `title`, rescoped `score`), which is a breaking change for any external
consumer of `knowledge search` output — sanctioned by the spec, with all
shipped consumers updated in lockstep in Phase 3.1. Prerequisite ordering:
the uncommitted 000026_ripgrep-replace implementation on this branch must
land before Phase 1.1 starts.

## Performance Considerations

Knowledge bases are small by standing assumption (single-digit entries
today); no performance work is in scope. The design preserves the scanner's
bounded-memory properties: one streaming pass per file, line-bounded
buffers (1 MiB cap), and a bounded excerpt-candidate set per file instead
of collecting all matching lines. Ranking sorts the full merged result
slice in memory, which is fine at knowledge-base scale and explicitly
covered by the spec's "no smarter relevance, limits, or pagination"
non-goal.
