# Research: 000027_document-level-search

## Alternatives considered and rejected

### Option A: Whole-file read + per-term `strings.Contains` over the full content

Read each file into memory, check every query term against the full text,
then re-derive title/excerpts.

**Rejected**: it discards the streaming long-line resilience the existing
scanner just earned (internal/store/search.go:122-127 keeps pre-long-line
matches and skips the rest without error, proven by
`TestSearch_OversizedLineSkipsFileNotSearch`,
internal/store/search_test.go:171-190); a full read of a pathological file
is unbounded memory, and avoiding that would mean duplicating the scanner's
edge-case handling (binary sniff, buffer caps) a second time.

### Option B: Per-term searches intersected after the fact (Set- or cmd-level post-processing)

Run the existing single-term scan once per query word and intersect the
per-file hit sets.

**Rejected**: N full `filepath.WalkDir` passes per query
(internal/store/search.go:48); no pass ever sees all terms at once, so
"lines containing more distinct terms" excerpt selection and a document
title are impossible without re-reading every matched file; per-document
score composition across independent scans is awkward.

### Option C: Sorting results in `cmd/knowledge.go` instead of `knowledge.Set`

**Rejected**: ranking must be global across scopes (spec: stronger matches
first, full result list), and `Set.Search` is the single point where all
scopes' hits meet (internal/knowledge/set.go:87-97, currently documented as
"no ranking or dedup"). Sorting in cmd would leave the library API returning
unranked results that every future consumer must re-sort.

### Option D: (from spec non-goals, pre-rejected — do not re-propose)

Approximate matching/stemming, query syntax (quotes/booleans), filename
matching, TF-IDF/semantic ranking, result caps/pagination. All weighed
during the spec workflow and excluded (spec § Non-Goals,
.spektacular/specs/000027_document-level-search.md:177-199).

## Chosen approach — evidence

Aggregate inside the existing per-file scan: tokenize the query into
lower-cased whitespace-separated terms, have the scanner track per-term
occurrence counts per file plus the best excerpt lines, and emit **one Hit
per file** that contains every term.

- `scanFile` already returns a per-file aggregate (`[]lineMatch`,
  internal/store/search.go:87-90, 96-131); the per-line→Hit fan-out is one
  localized loop (internal/store/search.go:65-76). Document-level matching
  is an evolution of `scanFile`'s return type, exactly as the spec's
  technical approach anticipates (spec lines 130-143).
- Single pass per file preserves all hardened behaviours for free: binary
  sniff (search.go:103-111, git's NUL convention), conventions `SkipDir`
  (search.go:52-59), 1 MiB line cap with keep-what-you-have semantics
  (search.go:122-127), excerpt budget `trimExcerpt` (search.go:137-147).
- Title can be captured during the same pass (first ATX heading line),
  falling back to the locator — needed because knowledge bases contain
  non-markdown entries (e.g. `architecture/bob_json/bob_stream.json`
  produced 15 of 60 "workflow" hits in spec-time evidence).
- Blast radius of the `Hit` shape change is fully mapped: `store.Hit`
  (internal/store/store.go:23-28) flows only through
  `Set.Search` (internal/knowledge/set.go:87-97) →
  `runKnowledgeSearch` (cmd/knowledge.go:117-134, schema at 57-60) →
  `output.WriteResult` (internal/output/writer.go:24-26). Grep confirms no
  other non-test caller of `.Search(` or `store.Hit`.
- Ranking: sort merged hits in `Set.Search` by Score descending with a
  deterministic tie-break (configured scope order, then path) so tests and
  agents see stable output. `Set.sources` already preserves config order
  (internal/knowledge/set.go:18-28, 89-95).
- Score: sum of all terms' case-insensitive occurrence counts across the
  document (spec technical approach, lines 140-143). Single-word queries
  then match exactly the same file set as today: per-line substring hit ⇔
  term occurs somewhere in the file (regression guarantee, spec lines
  52-53, 112-113).

## Files examined

- `internal/store/search.go:35-147` — current per-line engine: lower-cased
  needle is the WHOLE query string (no tokenization, search.go:45); one Hit
  per matching line (65-76); Score = per-line occurrence count (74); binary
  sniff 8000B (103-111); 1 MiB scan buffer (115); ErrTooLong keeps prior
  matches (122-127); trimExcerpt 256B cap (137-147).
- `internal/store/store.go:23-28` — `Hit{Scope,Path,Excerpt,Score}` with
  JSON tags; Path documented "pass to Read". :49 — `Store` interface
  includes `Search`; :76-95 — `Read` joins path against root, rejects
  traversal, `ErrNotFound` for missing (the untouchable read contract).
- `internal/knowledge/set.go:57-82` — Set construction, scope-tagged stores
  in config order; :87-97 — Search concatenates per-source hits, "no
  ranking or dedup" (the place ranking will live); :100-106 — Read(scope,
  path); :141-160 — conventions served whole, why search excludes them.
- `cmd/knowledge.go:20-25,117-134` — `search <query>` ExactArgs(1),
  `{"hits": [...]}` envelope, nil→`[]store.Hit{}`; :57-69 — output schemas
  for search and read; :136-158, 234-254 — read command, scope+path both
  required.
- `internal/output/writer.go:24-88` — generic JSON marshal/indent,
  top-level `--fields` filtering; indifferent to Hit shape.
- `internal/store/search_test.go` — fixtures built in-code via
  `writeSearchFixture` (:21-36, includes conventions/ trap file with the
  needle); hand-maintained score oracle (:119-143); binary-skip asserts
  exact path slice (:147-166, order-sensitive via WalkDir lexical order);
  oversized-line test uses ElementsMatch (:171-190). Tests-own-filesystem
  and independent-oracle conventions visible throughout.
- `internal/knowledge/set_test.go` — Set.Search fan-out + scope tagging
  tests (will gain ranking/tie-break assertions).
- `cmd/knowledge_test.go` — CLI-level JSON output tests for search/read.
- `internal/agent/instruction_surface_test.go:94` — asserts "knowledge
  search" appears in rendered instruction surfaces (string presence only).

## External references

- git's binary-detection convention (NUL within the first 8000 bytes) —
  already the basis of search.go:23-25; no new external references needed.
  The feature is deliberately self-contained (spec constraint, lines
  71-77): standard library only.

## Prior plans / specs consulted

- **Plan 000026_ripgrep-replace** (`go run . plan file read
  000026_ripgrep-replace/plan.md`) — landed on this same branch
  (uncommitted): promoted the built-in Go scanner to sole search
  implementation, proved byte-identical parity with rg on 14 queries.
  Lesson: the limitation motivating 000027 is the matching model, not the
  scanner; build on the scanner, don't rebuild it.
- **Spec 000027_document-level-search** (`go run . spec file read
  000027_document-level-search.md`) — source of truth; non-goals
  pre-reject four alternative families; success metrics are tied to this
  repo's own knowledge base.
- **Spec/plan 000025_convention-aware-planning** — origin of the
  conventions/ search exclusion that must be preserved.

## Open assumptions

- **Tokenization is `strings.Fields` on the raw query** (whitespace split,
  lower-cased). A whitespace-only query yields zero terms and is treated
  like the empty query: empty results, no error (NOT match-everything).
- **Title = first ATX-heading line** (`#`-prefixed, trimmed of `#` and
  whitespace) encountered during the scan; entries with none (e.g. JSON
  files) fall back to the locator. Heading text after an over-long line is
  unreachable (scanner abandons the file) — locator fallback covers it.
- **A term occurring only after an over-long line is missed**, so such a
  document may fail to match a query it textually contains. This preserves
  existing behaviour; spec only requires the search to succeed and other
  entries to be returned.
- **Hit shape is** `{scope, path, title, score, excerpts: [...]}` with at
  most three excerpts per result (spec requires ≥1 excerpt; the "no result
  caps" non-goal applies to results, not excerpts-per-result).
- **`Store` interface Search signature changes ripple nowhere else**:
  FileStore is the only implementation and knowledge.Set the only caller
  (grep-verified, non-test).
- **Knowledge bases stay small** (single-digit entries today; standing
  assumption since search was built) — in-memory per-file term aggregation
  and full-result sorting are fine.
- **Occurrence-count scoring ranks the relevant entry first on the live
  knowledge base** — NOT verifiable until implementation; carried as the
  plan's sole open question. If a noise entry (raw JSON transcript)
  outranks the relevant entry for a motivating query, STOP and ask.

## Rehydration cues

- `go run . spec file read 000027_document-level-search.md` — re-read the
  spec (requirements, constraints, non-goals, success metrics).
- `go run . plan file read 000026_ripgrep-replace/plan.md` — predecessor
  plan; scanner design rationale and proven edge-case behaviour.
- Re-read `internal/store/search.go` + `search_test.go` in full (≈150 +
  190 lines) — the entire engine and its oracle tests.
- `internal/knowledge/set.go:87-106` — ranking insertion point and read
  contract.
- `cmd/knowledge.go:57-69,117-158` — CLI schemas and envelopes.
- `go run . knowledge search workflow` — reproduce the motivating noise
  (per-line hits, JSON-file pollution) against the live knowledge base;
  `go run . knowledge search "workflow steps"` — reproduce the motivating
  zero-result miss (until implemented).
- Shipped consumers needing lockstep update: `cmd/knowledge.go`
  help+schema; `templates/skills/workflows/spek-knowledge/SKILL.md` and
  rendered `.claude/skills/spek-knowledge/SKILL.md`;
  `templates/steps/plan/02-discovery.md`;
  `templates/skills/skill_spawn-planning-agents.md`; `README.md:164`;
  `CHANGELOG.md`. (Other workflow skills verified to contain zero
  "knowledge" mentions.)
- `go run . knowledge conventions` returned `[]` — no project conventions
  exist; the plan's Conventions section states that plainly.
