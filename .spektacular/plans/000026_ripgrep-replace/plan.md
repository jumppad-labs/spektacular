# Plan: 000026_ripgrep-replace

<!-- Metadata -->
<!-- Created: 2026-06-11T10:35:00Z -->
<!-- Commit: f0d74db -->
<!-- Branch: main -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

<!--
  OVERVIEW
  A concise 2-3 sentence summary of the plan. Answer:
    1. What is being built?
    2. What problem does it solve?
    3. Who benefits?
  No file paths, no commands, no implementation detail. A reviewer should be
  able to decide whether the plan is worth reading in full from this section
  alone.
-->
## Overview

This plan removes Spektacular's dependency on the external ripgrep tool by promoting the
existing built-in Go search path to be the only search implementation, closing the three
behavioural gaps (relevance scores, binary-file handling, long-line resilience) so users
notice no change in what they find. Search becomes fully self-contained: a fresh install
on a clean machine searches successfully with nothing else installed, and no search can
ever fail because an external tool is missing or the wrong version.

<!--
  CONVENTIONS
  The project conventions (always-apply standards) that bear on this work,
  each with a one-line reason it applies — and only the relevant ones, not
  the whole knowledge base. Conventions are loaded in full during discovery
  and the relevant subset is chosen as the design is locked, then confirmed
  by the user. Cite a convention inline in the sections it drives. If no
  conventions are relevant (or the project has none), state that plainly,
  e.g. "No project conventions apply to this feature." An empty or generic
  list is a visible signal the knowledge base was not consulted.
-->
## Conventions

No project conventions apply to this feature: the project's conventions store was loaded
in full during discovery and is empty.

<!--
  ARCHITECTURE & DESIGN DECISIONS
  The chosen design direction in 2-4 short paragraphs. Explain the shape of
  the solution, the key decisions and their trade-offs, and why the chosen
  direction beats the alternatives. Cross-reference
  research.md#alternatives-considered-and-rejected so readers can drill into
  the evidence for rejected options. This is plan.md's load-bearing section —
  a reviewer should be able to spot missing architectural patterns or design
  gaps from this section without needing to read context.md.
-->
## Architecture & Design Decisions

The chosen shape is an in-place promotion of the existing native search path. Today
`FileStore.Search` (internal/store/search.go:31-41) probes PATH for `rg` and shells out to
`searchRipgrep`, falling back to `searchNative` — a pure-Go `filepath.WalkDir` + per-line
case-insensitive substring scan that plan 000025's equivalence test already proved
functionally identical on `{Scope, Path, Excerpt}`. This feature deletes the external path
entirely — `searchRipgrep`, the `rgEvent` JSON decoder, the `exec.LookPath` dispatch, and
the now-meaningless `forceFallback` test seam on `FileStore` (internal/store/store.go:54-61)
— and makes the native scan the only implementation, removing the `os/exec` dependency from
the package so the "no external tool or process" constraint is satisfied by construction
rather than by configuration. No public API changes: `Store.Search(query) ([]Hit, error)`,
the `Hit` struct, the `Set.Search` fan-out (internal/knowledge/set.go:87-97), and the
`knowledge search` CLI envelope (cmd/knowledge.go:117-134) are all untouched.

Three deliberate behavioural decisions close the gaps between what rg-equipped users see
today and what the bare native path does, so the replacement is faithful to *observed*
behaviour rather than to the weaker fallback. First, `Hit.Score` is computed natively as
the count of non-overlapping case-folded occurrences of the query in the matching line,
replicating rg's per-line submatch count (search.go:97) instead of the fallback's
constant 0. Second, binary files are skipped using the git/ripgrep convention — a NUL byte
within the first 8000 bytes of a file means skip it — so curated text stores never surface
garbage excerpts; rg's hidden-file and gitignore filtering is deliberately *not*
replicated, because knowledge stores are content directories, not git working trees.
Third, an over-long line (>1MiB, `bufio.ErrTooLong`) now skips the remainder of that file
and lets the search continue, implementing the intent the code comment at search.go:20-22
always claimed; today such a line actually aborts the entire search. A scanner that hits
`ErrTooLong` is unrecoverable mid-file, so the skip is at file granularity — any other
scan error still fails the search loudly, preserving the fail-fast contract.

This direction beats the alternatives because the replacement is already written and
verified: promoting it is a deletion plus three small, separately testable behaviours.
Extracting a dedicated searcher package was rejected as speculative — `FileStore.Search`
is the sole consumer, and plan 000025 already established the precedent of keeping search
logic inline rather than abstracting it. Adopting a third-party library was rejected on
the evidence twice over: no maintained, importable, license-compatible pure-Go grep
library exists, and an indexing engine like bleve is categorically the wrong shape for a
live substring scan over a small store. Matching idiom stays `strings.ToLower` +
`strings.Contains` — not exact Unicode simple folding, but it is precisely what the
equivalence test certified as "the same case-insensitivity" users have today. See
research.md#alternatives-considered-and-rejected for the full evidence trail.

The blast radius is one package plus its tests and docs: the two rg-conditional tests
(internal/store/search_test.go:83-108, 135-150) are retired with their fixture semantics —
conventions exclusion, empty-query and no-match behaviour — re-homed in
backend-independent tests alongside new coverage for the three behaviours above; the
stale comments at search.go:20-22 (long-line "skip" claim) and search.go:119 (broken
`!conventions/**` glob form) are corrected; and README.md:212 stops describing a
ripgrep preference that no longer exists.

<!--
  COMPONENT BREAKDOWN
  The components (new or changed) that make up the solution, with their
  responsibilities and how they interact. One bullet or short paragraph per
  component. Name the component, state what it owns, and describe its
  relationship to the other components. Do not list file paths or line
  numbers here — component responsibilities, not implementation sites.
-->
## Component Breakdown

- **Store search (changed)** — `FileStore.Search` in the store package becomes the single,
  in-process search implementation. It owns the empty-query short-circuit, the recursive
  walk of the store root, the conventions-directory exclusion, and the assembly of
  scope-tagged hits (relative path, trimmed excerpt, score). The dual-backend dispatch is
  removed: there is no longer a probe for an external binary and no alternate code path.
- **File scanner (changed)** — the per-file scanning helper inside the store package. It
  owns binary detection (NUL byte in the leading bytes → skip the file), per-line
  case-insensitive literal matching, per-line match counting (the new Score source), and
  over-long-line handling (an oversized line skips the remainder of that file without
  failing the search; all other errors still fail loudly).
- **Excerpt trimming (unchanged)** — the existing shared helper that collapses whitespace
  and caps excerpts at the byte budget on a rune boundary. Already backend-independent;
  the surviving search path keeps using it as-is.
- **Ripgrep adapter (removed)** — the external-process backend: the rg invocation and
  JSON-event decoding, the binary-on-PATH dispatch, plus the test-only fallback-forcing
  seam on the store struct that existed solely to pin the two backends against each other.
  All deleted; the store package drops its subprocess dependency entirely.
- **Knowledge set & CLI (unchanged)** — the scope fan-out that concatenates per-store hits
  and the `knowledge search` command with its JSON envelope are deliberately untouched;
  they observe only the `Store.Search` contract, which keeps its signature and hit shape.
- **Search tests (changed)** — the rg-conditional equivalence and empty-result tests are
  retired; their guarantees (conventions exclusion, empty-query/no-match behaviour) move
  into backend-independent tests, joined by new coverage for scoring, binary skip, and
  long-line skip.
- **Documentation (changed)** — the README's description of search stops mentioning an
  external-tool preference and describes the built-in implementation.

<!--
  DATA STRUCTURES & INTERFACES
  The types, interface signatures, and serialization boundaries introduced or
  changed by the plan. Show type shapes in pseudocode or a short code block
  where useful. Focus on the contract between components, not internal
  representation detail.
-->
## Data Structures & Interfaces

No new types or interfaces are introduced — the design's core property is that every
existing contract is preserved while internal types are deleted:

- **`Store` interface (unchanged)** — `Search(query string) ([]Hit, error)` keeps its
  signature and semantics: empty query and no-match both return an empty result with a nil
  error; any store-level failure is an error, never a silent partial result.
- **`Hit` (shape unchanged, one field's semantics firmed up)** — remains
  `{Scope, Path, Excerpt, Score}` with the same JSON tags (`scope/path/excerpt/score`), so
  the `knowledge search` CLI envelope `{"hits": [...]}` is byte-compatible. `Score`
  changes from "populated only by the external backend, 0 otherwise" to "always the
  per-line count of non-overlapping case-insensitive query occurrences" — its doc comment
  is updated to state the new invariant.
- **`rgEvent` (removed)** — the internal decoder struct for the external tool's JSON event
  stream is deleted along with its only consumer.
- **`FileStore` (one field removed)** — drops the test-only `forceFallback` flag; the
  struct otherwise keeps its root and scope fields as-is.

Serialization boundaries: the only one that remains — the CLI's JSON output — is
unchanged. The deleted rg JSON-event parsing was the other serialization boundary, and it
disappears with the subprocess.

<!--
  IMPLEMENTATION DETAIL
  High-level only. Sketch new patterns being introduced, major code-shape
  changes, and code-structure UX — enough for a reviewer to spot missing
  patterns or design gaps. This is NOT per-phase file:line work — that
  belongs in context.md. If you find yourself writing "in file X at line Y",
  stop and move it to context.md.
-->
## Implementation Detail

This change is **deletion-led**: the dominant code-shape change is collapsing a
dual-backend dispatch into a single code path. The search entry point stops being a router
("probe for an external binary, pick a backend") and becomes a direct call to the
walk-and-scan implementation. A developer reading the changed code experiences one linear
story — walk, exclude conventions, scan, build hits — instead of two parallel
implementations with a contract holding them together. The package also sheds its
subprocess machinery entirely; nothing in the store package executes external processes
any more, which makes the spec's constraint structurally true rather than policed.

The per-file scanner grows three behaviours, each following an existing pattern rather
than introducing a new one:

- **Binary sniff before scanning** — a leading-bytes NUL check (the git/ripgrep
  convention) decides skip-or-scan per file. This is a quiet skip, matching the
  established pattern that exclusions (conventions directories) are silent, not errors.
- **Recoverable vs. fatal scan errors** — the scanner discriminates the oversized-line
  sentinel error from all other failures: oversized lines end that file's scan and the
  walk continues; everything else still fails the whole search, preserving the existing
  fail-fast contract ("never silently return a subset" — except where explicitly
  documented as a skip).
- **Match counting** — the scanner reports matched lines along with their per-line
  occurrence count, which becomes the hit's score. This slightly widens the scanner's
  return shape from "matched lines" to "matched lines with counts".

The test suite shifts model: today the load-bearing test is *differential* (two backends
pinned against each other on shared fixtures); with one backend that becomes impossible,
so the same fixtures are re-pointed at *direct behavioural* assertions — conventions
exclusion, case-insensitivity, empty-query/no-match semantics, excerpt budget — plus new
direct coverage for scoring, binary skip, and long-line skip. Test construction keeps the
established convention: fixtures built in temp dirs through the production write path,
never by walking real directories.

No new module boundaries, no new interfaces, no new packages: existing patterns (fail-fast
store errors, shared excerpt trimming, scope-tagged hits) are followed throughout.

<!--
  DEPENDENCIES
  The internal packages, external libraries, upstream specs, or prior plans
  this work depends on. One bullet per dependency with a one-line note on
  what it provides and whether it needs any changes.
-->
## Dependencies

This work is notable for *removing* a dependency; what remains is small:

- **Go standard library only** (`path/filepath`, `bufio`, `bytes`, `strings`,
  `unicode/utf8`, `io/fs`, `os`) — provides the walk, scan, and matching primitives.
  Already imported by the store package; no version or module changes. The `os/exec`
  import is *removed*.
- **ripgrep (`rg`) external binary — dependency being eliminated** — currently an optional
  runtime accelerator probed per call; this plan deletes the integration. Nothing needs rg
  to be present (or absent) for the work to land.
- **`github.com/stretchr/testify`** — existing test-only dependency; the reworked and new
  tests keep using `require`. No changes.
- **Upstream specs/plans (all landed, nothing blocking)**:
  - *Spec 000026_ripgrep-replace* — the spec this plan implements; approved.
  - *Plan 000020_context* — built the search feature and the native path being promoted;
    landed.
  - *Plan 000025_convention-aware-planning* — established the conventions-exclusion
    semantics the replacement must preserve; landed.
- **Downstream consumers (no changes required)** — the knowledge set fan-out, the
  `knowledge search` CLI command, and the agent skill templates all consume the unchanged
  `Store.Search` contract; none need modification beyond what this plan itself does.

No dependency must land or change before this plan starts.

<!--
  TESTING APPROACH
  High-level overview of the testing strategy: what kinds of tests
  (unit, integration, contract, regression), which components get the most
  coverage, and what the load-bearing assertions are. Per-phase testing
  detail — which specific tests live in which specific files — stays in
  context.md.
-->
## Testing Approach

**Strategy: re-point the existing differential suite at direct behavioural assertions,
then add targeted coverage for the three new behaviours.** All tests are unit and
CLI-integration tests in the project's established style: Go `testing` +
`testify/require`, fixtures built in temp dirs through the production write path (tests
own their filesystem), hand-maintained expected values as oracles.

**Test types:**

- **Regression (unchanged tests)** — the backend-independent store tests (scope/locator
  round-trip, excerpt budget, trim behaviour), knowledge-set fan-out tests, and the CLI
  integration test that pins the JSON envelope all pass unchanged. These are the primary
  "no regression" guard: they assert search behaviour through the public contract without
  caring which backend runs.
- **Behavioural unit tests (reworked)** — the retired rg-vs-native equivalence tests
  donate their fixtures and guarantees to direct assertions: case-insensitive literal
  matching (an uppercase needle in a nested file is found by a lowercase query),
  conventions-directory exclusion at any depth, empty query → empty result with no error,
  no-match → empty result with no error.
- **Behavioural unit tests (new)** — one per new behaviour: Score equals the per-line
  count of non-overlapping case-insensitive occurrences; a file with a NUL byte in its
  leading bytes is skipped while sibling text files still match; a file containing an
  oversized line is skipped from that point without failing the search, and other files'
  hits still return.

**Most coverage goes to the store search component** — it's the only component whose
implementation changes; the knowledge set and CLI layers are deliberately exercised only
through their existing tests since their code is untouched (no redundant re-assertion of
the same guarantees through a second mechanism).

**Success metrics from the spec, each made verifiable:**

1. *"Zero external runtime dependencies for search: a fresh install on a clean machine
   performs search successfully with nothing else installed."* — **Behavioural test**,
   plus a structural guarantee. The reworked unit and CLI tests prove search returns
   correct results running entirely in-process — after this change there is no code path
   that can consult PATH or spawn a process, so any machine that can run the test suite is
   a "clean machine" for search purposes. The structural half — the store package no
   longer imports subprocess machinery — is verified at review/compile time by the
   deletion itself.
2. *"No regression in search results: the existing search-related tests pass with the
   built-in implementation."* — **Behavioural test**: all existing backend-independent
   search tests pass unchanged. The two rg-conditional tests are retired by design, not
   regressed — their fixture semantics are preserved in the reworked direct tests, which
   is recorded so a reviewer can trace each retired guarantee to its new home.

**Deliberate gaps:** no performance tests (explicit spec non-goal); no Unicode
case-folding edge-case tests beyond what `strings.ToLower` semantics give (matching idiom
is unchanged from the certified-equivalent native path); no test that rg's
hidden-file/gitignore filtering is absent (that filtering was an undocumented rg-path side
effect, decided against replicating).

<!--
  MILESTONES & PHASES
  2-4 milestones. Each milestone leads with a "What changes" summary
  paragraph describing the user-visible difference when the milestone lands.
  Each phase has a 2-4 sentence plain-language summary, a *Technical detail:*
  link to context.md, and an **Acceptance criteria**: checkbox list with
  outcome statements (not shell commands). No file:line references in
  plan.md phase content — those live in context.md.
-->
## Milestones & Phases

### Milestone 1: Built-in search matches what users see today

**What changes**: The pure-Go search path gains the three behaviours that today only show
up (or behave differently) when the external tool runs: results carry a meaningful
relevance score (per-line match count), binary files are quietly skipped instead of
surfacing garbage excerpts, and a pathologically long line no longer kills the entire
search — it skips that file and keeps going. Users without ripgrep installed get strictly
better results immediately; users with ripgrep notice nothing yet.

**Validation point:** new behavioural unit tests for scoring, binary skip, and long-line
skip pass; the full existing suite — including the rg-vs-native equivalence test, still
alive at this point — stays green.

#### - [ ] Phase 1.1: Native relevance scoring

The built-in search path starts computing a relevance score for every hit: the number of
times the query appears in the matching line, exactly what users with the external tool
installed see today. Until now the built-in path always reported zero, so removing the
external tool later would have silently degraded result metadata. After this phase the
two backends agree on scores too, not just on which lines match.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-native-relevance-scoring)

**Acceptance criteria**:

- [ ] A search hit's score equals the number of non-overlapping, case-insensitive
      occurrences of the query in the matching line (e.g. a line containing the needle
      twice scores 2).
- [ ] Hits on lines containing the query once score exactly 1, regardless of letter case.
- [ ] All previously passing search tests still pass, including the backend-equivalence
      test, which continues to ignore scores.

#### - [ ] Phase 1.2: Resilient file handling — binary skip and long-line recovery

The built-in search becomes robust against awkward store contents. Files that are binary
(detected the same way git and ripgrep detect them) are quietly skipped instead of being
scanned for text matches, and a file containing a pathologically long line no longer
aborts the entire search — the scanner skips the rest of that file and carries on with the
others. Today such a line fails the whole query, contradicting what the code's own
documentation promises.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-resilient-file-handling--binary-skip-and-long-line-recovery)

**Acceptance criteria**:

- [ ] A file whose leading bytes contain a NUL byte produces no hits, while text files in
      the same store still match normally.
- [ ] A store containing one file with an oversized line still returns hits from its other
      files, and returns any hits found in the oversized file before the long line; the
      search reports success, not an error.
- [ ] Genuine failures (an unreadable file or directory) still fail the search with an
      error rather than being skipped.
- [ ] All previously passing search tests still pass.

### Milestone 2: Search is fully self-contained; no external tool ever needed

**What changes**: The external-tool integration is removed end to end: Spektacular no
longer looks for, runs, or depends on ripgrep, and search behaves identically on every
machine. The differential tests that pinned the two backends together are retired with
their guarantees re-homed in direct tests, and the README and code commentary stop
describing an external-tool preference that no longer exists. This is the spec's
deliverable: a fresh install on a clean machine searches successfully with nothing else
installed.

**Validation point:** the full test suite passes with no rg-conditional skips; the store
package contains no subprocess code; README and doc comments describe only the built-in
implementation.

#### - [ ] Phase 2.1: Remove the external tool integration

Spektacular stops looking for or running ripgrep entirely: the external backend, the
machinery that chose between backends, and the test seam that pinned the two backends
against each other are all deleted, making the built-in path the only path. The
differential tests retire, with each guarantee they carried — case-insensitive matching,
exclusion of convention files, clean empty results for no-match and empty queries —
re-homed in direct tests that run on every machine, no longer skipped when the external
tool is absent.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-remove-the-external-tool-integration)

**Acceptance criteria**:

- [ ] Searching a populated store on a machine with no external search tool installed
      returns the same matches, excerpts, and scores as on a machine that has one — there
      is no longer any code that consults the environment for a search tool or starts an
      external process.
- [ ] Convention files still never appear in search results, at any nesting depth.
- [ ] An empty query and a no-match query both return an empty result without error.
- [ ] The test suite contains no tests that skip themselves when the external tool is
      missing, and no test reaches for a backend-selection seam.

#### - [ ] Phase 2.2: Documentation matches the built-in search

Everything a reader encounters — the README's feature description and the code's own
commentary — describes a single built-in search, with the external-tool preference and the
two known-stale comments (the long-line claim that was previously false, and a reference
to a superseded exclusion pattern) corrected. A newcomer reading the docs or the code gets
an accurate picture of how search works.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-documentation-matches-the-built-in-search)

**Acceptance criteria**:

- [ ] The README describes search as built-in and self-contained, with no mention of
      preferring or falling back from an external tool.
- [ ] The search code's doc comments describe only the implemented behaviour: in-process
      scanning, score semantics, binary-file skipping, and long-line file skipping.
- [ ] No comment in the store package references ripgrep flags, globs, or backend
      equivalence.

<!--
  OPEN QUESTIONS
  Strictly for questions that genuinely cannot be resolved until
  implementation begins. Anything resolvable by asking the user, reading the
  code, or running a quick experiment must be resolved now — not parked
  here. If this section is empty, that is the expected outcome of a healthy
  planning pass.
-->
## Open Questions

None — after a genuine pass, there are no implementation-time-only uncertainties. Every
question surfaced during discovery was resolvable before implementation and was resolved:
the three behavioural deltas (score semantics, binary-file handling, long-line handling)
by user decision; library viability by research; the current code's shape by reading it;
the test-fixture sequencing hazard by analysis during phase planning. The code being
promoted already exists and is equivalence-tested, and the additions are small, locally
testable behaviours.

Standing guard from research.md § Open assumptions still applies: if the implementer finds
any recorded assumption wrong (e.g. a consumer that depends on rg's exact Score
arithmetic, or store content relying on hidden-file filtering), STOP and ask rather than
improvising.

<!--
  OUT OF SCOPE
  Explicit exclusions agreed during planning. Each bullet states what is NOT
  being done and, where useful, where it is tracked instead. This is as
  important as the requirements — it prevents scope creep and sets clear
  expectations for reviewers.
-->
## Out of Scope

- **No new search capabilities** — no regex, fuzzy matching, or relevance ranking beyond
  the like-for-like per-line occurrence count. (Spec § Non-Goals; ranking/indexing also
  explicitly excluded back in plan 000020_context.)
- **No performance work** — no benchmarks, no optimization, no guarantee the built-in scan
  is as fast as ripgrep. Matching current behaviour is the bar. (Spec § Non-Goals.)
- **No exact Unicode case-folding semantics** — matching keeps the existing
  `ToLower`-based idiom rather than replicating rg's Unicode simple folding (Turkish İ,
  ς/σ edge cases). If exact folding is ever demanded, it is a small self-contained
  follow-up, not this plan.
- **No hidden-file or gitignore filtering** — rg's implicit filtering is deliberately not
  replicated (user decision during discovery): knowledge stores are content directories,
  not git working trees. Only binary files are skipped.
- **No searcher package extraction** — search logic stays inline in the store package
  (architecture decision; sole consumer). Revisit only if a second consumer appears.
- **No search-index or caching layer** — every query remains a fresh walk-and-scan,
  resting on the recorded "knowledge stores stay small" assumption from plan 000020_context.
- **No changes to consumers** — the knowledge set fan-out, the `knowledge search` CLI
  envelope, and the agent skill templates are untouched; any envelope evolution is future
  work.
