# Working context — implement 000027_document-level-search

IMPLEMENT WORKFLOW in progress (started 2026-06-11). read_plan complete:
structural validation passed (all 10 sections, 5 phase checkboxes 1.1/1.2/
2.1/2.2/3.1 with resolving context.md links); drift check clean — every
path/symbol/line ref in plan.md and context.md verified against the working
tree. Changelog mode: no `## Changelog` in plan.md → first-phase invocation,
start at Phase 1.1.

## User decisions this session

- Plan choice: user picked 000027_document-level-search (not 000026) when
  /spek-implement was invoked without a name.
- **000026 prerequisite waived**: the plan requires the uncommitted
  000026_ripgrep-replace work to be committed before Phase 1.1; user chose
  "proceed without committing" — 000026 and 000027 changes will coexist
  uncommitted on branch f-ripgrep-replace. Do not commit unless asked.

## ALL 5 PHASES COMPLETE + test_plan written. Advancing to finished.

- update_repo_changelog: CHANGELOG.md 000027 entry already present (Phase
  3.1 wrote it as a spec-mandated lockstep surface) — left as-is, no dup.
- test_plan: live-KB open-question check done. Findings:
  - Metric 1 (motivating queries): "workflow steps"/"workflow state" →
    workflow-steps.md ranked FIRST (scores 38/37). Scoring works; no noise
    entry outranks the relevant one (open question resolved NO).
  - The spec's 3rd example "interrupted workflow" → 0 hits because the word
    "interrupted"/"interrupt" exists NOWHERE in the KB (correct AND-match,
    not a bug). USER said it was just an illustrative example; replaced with
    "scope label" → initial-idea.md FIRST, words on disjoint lines
    (scope 20/214/502/587/731, label 312/315/616/620) — a genuine
    document-level win (per-line would've returned 0 from it).
  - Metric 2: "workflow" → 9 ranked results (was 60 line hits).
  - Metric 3: "workflow" 9 results == 9 distinct non-binary/non-convention
    files containing the word.
  - Round trip: read of top hit scope/path returns full body (~38.6KB).
  All captured in 000027_document-level-search/test-plan.md.

## Phase 3.1 verify step — green (all 3 criteria pass; help text, skill
flow, template/rendered parity, changelog, full suite all verified)

## Phase 3.1 implement step — done

- Edited: spek-knowledge SKILL.md template (ranked per-document results,
  all-words matching note); 02-discovery.md (ranked results wording);
  skill_spawn-planning-agents.md (ranked per-document; also dropped stale
  "conventions" from the search-targets list — conventions are excluded
  from search since 000025; note as deviation); README search row + Hit
  struct block (Title/Excerpts/score meaning) + built-in Search bullet;
  CHANGELOG.md new top entry 000027_document-level-search.
- Rendered .claude/skills/* regenerated via `go run . init claude`
  (.claude/skills is GITIGNORED — rendered copies are not tracked; init
  also rewrote AGENTS.md managed section, no diff). Sweep clean; go test
  ./... exit 0.
- Per plan testing strategy, 3.1 has no new automated tests — existing
  rendered-template tests + manual sweep cover it.

## Phase 3.1 analyze — done (Low, main context)

- Rendered skill copy is byte-identical to template with {{command}}→
  "go run ." ; regeneration path = `go run . init claude`
  (installWorkflowSkills, internal/agent/skills.go).
- Edit list: spek-knowledge SKILL.md template :26 (hits description);
  regenerate .claude copy via init; templates/steps/plan/02-discovery.md:9;
  templates/skills/skill_spawn-planning-agents.md:18 (verify/adjust);
  README.md:164 (search row) AND README.md:199-205 Hit struct code block
  (Excerpt→Excerpts/Title/score meaning — found by sweep, not in plan's
  list) + :214 "literal matching" bullet; CHANGELOG.md new 000027 entry on
  top. skill_verify-implementation.md "excerpt" hits are failure-output
  excerpts — unrelated, leave.

## Phases 2.1 + 2.2 — COMPLETE (Milestone 2 done). Remaining: 3.1 only.

- 2.2 deviation recorded in plan changelog: schemaProp gained optional
  Properties map (cmd/spec.go) — needed to express object item schemas.

## Phase 2.2 test step — done

- cmd/knowledge_test.go: knowledgeHit gained Title; search test now
  asserts exact ordered hit list incl. title (path-fallback — fixture has
  no headings; heading titles covered at store level), score, excerpts,
  and the documented tie order (project before team).
  New: TestKnowledgeSearch_SchemaDeclaresPerDocumentHitFields (search
  --schema needs a dummy positional arg — ExactArgs validates before the
  schema short-circuit) and
  TestKnowledgeSearch_EmptyAndNoMatchQueriesReturnEmptyHits (JSONEq
  {"hits": []} pins non-nil array). go test ./... green.

## Phase 2.2 implement step — done (code only, tests next)

- cmd/spec.go: schemaProp gained Properties map (omitempty) so array items
  can describe object fields — shared by all --schema output, additive.
- cmd/knowledge.go: search Short → "returning ranked, one-per-document
  results"; knowledgeSearchOutputSchema hits.items now object{scope, path,
  title, score:number, excerpts:array<string>}. Envelope, nil→[] and read
  command untouched. Build + cmd tests green.

## Phase 2.2 analyze — done (Low, main context)

- schemaProp (cmd/spec.go:27-33) has Type/Enum/Pattern/MaxLen/Items but NO
  nested Properties — expressing per-hit object fields requires adding
  `Properties map[string]*schemaProp json:"properties,omitempty"` to
  schemaProp (minimal, omitempty, backwards compatible). No test pins the
  search schema JSON (the --schema CLI test exercises `read` only).
- Changes: knowledgeSearchOutputSchema hits.items → object{scope, path,
  title, score(number), excerpts(array of string)}; search cmd Short →
  ranked per-document wording; envelope + nil→[] untouched.

## Phase 2.1 — COMPLETE (ticked + changelog entry). Remaining: 2.2, 3.1.

## Phase 2.1 verify step — green (build/test/vet pass, 3/3 criteria asserted)

## Phase 2.1 test step — done

- set_test.go gained TestSet_SearchRanksAcrossSourcesByScore (later
  source's strong match outranks earlier source's weak match, exact
  order), TestSet_SearchTieBreaksBySourceOrderThenPath (fixture chosen so
  walk order ≠ path order: notes/zz.md walks before notes.md but sorts
  after; called twice for determinism), TestSet_SearchHitRoundTripsThroughRead
  (hit scope/path → set.Read returns full literal content). All green;
  go test ./... green.

## Phase 2.1 implement step — done (code only, tests next)

- set.go: Set.Search now collects (hit, source index) pairs and
  sort.SliceStable's them — Score desc, source index asc, Path asc — and
  returns the flattened slice; doc comment now states the ordering
  contract ("the returned slice is the display order"). "sort" import
  added. Build + existing knowledge tests green.

## Phase 2.1 analyze — done (Low, main context)

- Set.Search at set.go:87-97 unchanged since drift gate. set_test.go
  helpers: writeFile, twoScopeSet (project+team scopes, both contain
  "compass"). No existing set-level search→read chaining test — criterion
  3 needs one; store-level round trip covers Store.Read only.
- Approach: collect (hit, source index) pairs, sort.SliceStable score
  desc / source idx asc / path asc; rewrite "no ranking" doc comment.

## Phases 1.1 + 1.2 — COMPLETE (Milestone 1 done)

- Both phases ticked in plan.md with changelog entries (2026-06-11).
  Remaining: 2.1 (Set ranking), 2.2 (CLI schema/help), 3.1 (docs sweep).
- Phase 1.2 discovery worth keeping: headingText rejects "#hashtag";
  bare "#" first heading wins the slot with empty text → locator fallback;
  candidate insertion after equal entries preserves file-order tie-break.

## Phase 1.2 verify step — green

- build/test/vet all pass; 13/13 TestSearch tests green; all 3 criteria
  mapped to passing tests. Verifier noted "excerpt contains a query word"
  is only indirectly asserted — accepted: true by construction (candidates
  are matching lines) and pinned by the exact-string excerpt oracle;
  adding a direct assertion would breach no-redundant-assertions.

## Phase 1.2 test step — done

- search_test.go gained: TestSearch_TitleFromFirstHeading (first heading
  wins over later ones), TestSearch_TitleFallsBackToLocator,
  TestSearch_ExcerptPrefersLineWithMoreTerms (Excerpts[0] is the two-term
  line even when single-term lines precede it), TestSearch_ExcerptCountCapped
  (exactly 3). go test ./... fully green.

## Phase 1.2 implement step — done (code only, tests next)

- search.go: maxExcerptsPerHit=3 const added beside maxExcerptBytes;
  fileAggregate gained title/titleSet/best ([]candidateLine{text,
  distinct, total}) replacing lines []string; addCandidate keeps a
  bounded sorted top-3 (distinct desc, total desc, file order asc —
  insertion after equal entries preserves file order); headingText helper
  recognises ATX headings (trimmed "#"+space-or-alone; "#hashtag"
  rejected; first heading wins via titleSet even if its text is empty →
  locator fallback). Hit.Title = heading text else rel.
- go build + existing store tests green. Title/excerpt-selection tests to
  be added in test step.

## Phase 1.1 — COMPLETE (plan checkboxes ticked, changelog entry written)

- update_plan: Phase 1.1 heading + all 4 criteria ticked in plan.md.
- update_changelog: `## Changelog` section created in plan.md with the
  2026-06-11 Phase 1.1 entry (deviations: 000026-commit prerequisite
  waived by user; cmd/knowledge_test.go adapted early). 4 phases remain
  (1.2, 2.1, 2.2, 3.1).
- Looping to analyze for Phase 1.2 without prompting, per the user's
  standing skip-confirmation-on-simple-features preference.
- Phase 1.2 reminders: hoist inline literal 3 → maxExcerptsPerHit const;
  extend fileAggregate lines with per-line distinct-term + occurrence
  counts for best-line selection; capture first ATX heading as Title.
- Phase 1.2 analyze: done in main context (Low complexity); all changes
  confined to internal/store/search.go (+ tests), which Phase 1.1 just
  rewrote — no drift. Selection order: distinct terms desc, total
  occurrences desc, file order asc; candidate set bounded during scan.

## Phase 1.1 verify step — green

- go build / go test ./... / go vet (= make lint) all pass; all four
  acceptance criteria mapped to passing tests by the verify sub-agent.

## Phase 1.1 test step — done

- search_test.go adapted (Excerpts iteration, whitespace-only query case,
  score test renamed TestSearch_ScoreSumsOccurrencesAcrossDocument with
  spread.txt summed-score fixture + Len pin) and extended: scattered
  multi-word match, missing-word exclusion, one-hit-per-document with
  hand-computed score, single-word regression oracle
  (TestSearch_SingleWordMatchesSameFiles: needle → {top.txt,
  nested/deep.txt, long.txt}).
- cmd/knowledge_test.go minimally adapted (local struct excerpt→excerpts
  array; assertions on Excerpts[0]); CLI title/score/ranking assertions
  deliberately deferred to Phase 2.2.
- `go test ./...` fully green.

## Phase 1.1 implement step — done (code only, tests next)

- store.go: Hit reshaped — added Title, Excerpt→Excerpts []string, Score
  comment now "sum of all query terms' occurrences across the document".
- search.go: Search tokenizes via strings.Fields(strings.ToLower(query)),
  zero terms → (nil,nil); search(terms) emits one Hit per file where every
  term count > 0, Score = summed counts, Title = rel (heading capture is
  Phase 1.2), Excerpts = first matching lines (≤3, literal cap — the
  maxExcerptsPerHit const is scheduled for Phase 1.2's best-line
  selection), each trimExcerpt'd. lineMatch replaced by
  fileAggregate{counts []int, lines []string}; scanFile(path, terms)
  keeps binary sniff / 1MiB buffer / ErrTooLong keep-and-skip unchanged
  (binary → zero aggregate → no match).
- `go build ./...` passes; internal/store/search_test.go and
  cmd/knowledge_test.go (:16,:155-157) still reference old Hit shape —
  both to be adapted in the test step.

## Analyze findings (Phase 1.1 — current phase)

- All Phase 1.1 file:line refs verified current. Only non-test user of
  Hit.Excerpt is the fan-out loop at internal/store/search.go:73.
- Cross-phase learning: cmd/knowledge_test.go:16 declares a local hit
  struct with `excerpt` json field and asserts on it (:155-157). Hit shape
  change in 1.1 breaks it before Phase 2.2 — adapt it minimally in 1.1
  (excerpts array) to keep the full suite green; 2.2 owns schema/help/
  ranking assertions. internal/knowledge/set_test.go does not touch
  Excerpt.

## Key decisions (from plan workflow, carried forward)

- Design: tokenize query (strings.Fields, lower-cased; zero terms → empty
  result), single-pass per-file aggregation in scanFile (per-term counts,
  first-ATX-heading title, bounded best-excerpt candidates), match = all
  terms present, one Hit per document. Hit = {scope, path, title, score,
  excerpts[≤3]}; scope/path untouched (read contract preserved). Score =
  sum of term occurrences. Ranking in Set.Search: score desc, tie-break
  source order then path.
- 3 milestones / 5 phases: 1.1 aggregation+matching (Medium, Hit shape
  changes once), 1.2 titles+excerpts, 2.1 Set ranking, 2.2 CLI
  schema/help, 3.1 lockstep docs sweep (templates, rendered skill copy via
  init/render not hand-edit, README:164, CHANGELOG).
- Sole open question (impl-time): does occurrence-count scoring rank the
  relevant entry FIRST on the live KB given noisy JSON transcript entries?
  If not → STOP and ask user.
- Success metrics: fixture-based behavioural tests for mechanisms; live-KB
  outcomes are manual — captured in the implementation test plan
  (tests-own-filesystem convention forbids reading the live KB).
- No project conventions exist (conventions store empty).
