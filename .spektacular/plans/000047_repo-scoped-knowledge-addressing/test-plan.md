---
created_date: "2026-09-04"
status: completed
closed_date: "2026-09-04"
---

# Test Plan: 000047_repo-scoped-knowledge-addressing

Three of the spec's four success metrics are covered by automated behavioural tests and are not
restated here; they live in the suite:

- *No knowledge entry lands in the wrong store* — `internal/knowledge/set_test.go`
  (`TestSet_WriteIsReadableOnlyAtTheAddressItWasWrittenTo`,
  `TestSet_WriteWithoutAFullAddressRecordsNothing`,
  `TestSet_WriteWithoutAFullAddressRefusedWithASingleStore`) and `cmd/knowledge_test.go`
  (`TestKnowledgeWrite_WithoutAFullAddressRecordsNothing`,
  `TestKnowledgeReadWrite_AmbiguousRequestIsRefusedNotResolvedToFirstRepo`).
- *Planning loads less irrelevant knowledge* — `internal/knowledge/set_test.go`
  (`TestSet_NarrowedAlwaysAppliedLoadIsStrictlySmaller`).
- *A caller can predict what a request covers* — `internal/knowledge/set_test.go`
  (`TestSet_SelectorCoverageMatrixIsUniformAcrossRetrievalPaths`, nine cells across all four
  retrieval paths).

One metric is manual by nature, because it measures the act of delivery rather than a property of
the code.

## Metric: the dogfood write succeeds first time

**What to measure.** Whether recording a knowledge entry into a *named* repository's store, through
the delivered command surface, lands in the intended store on the first attempt, with no second try
and no fallback to editing files under the store directory by hand. The threshold is exact: **zero
retries and zero fallbacks, for every entry recorded.**

**Why it cannot be automated.** The durable half of the guarantee is already asserted behaviourally
above. What remains is an observation about using the tool: whether the address a person states is
the address the entry reaches without correction. A test that wrote and read back would only repeat
the assertions already in the suite.

**Setup.** A project registering at least two repositories, each with its own knowledge store, so a
misdirected write has somewhere wrong to land. In this repository, `go run . knowledge sources`
reports `repo/spektacular` and `repo/docs`, which satisfies it.

**How.**

1. Enumerate the stores and choose the destination from that listing rather than from memory:
   ```
   go run . knowledge sources
   ```
2. Stage the entry body at `.spektacular/tmp/<slug>.md`, then record it, naming the tier and the
   store:
   ```
   go run . knowledge write \
     --data '{"tier":"repo","name":"<store>","path":"<category>/<slug>.md"}' \
     --file .spektacular/tmp/<slug>.md
   ```
3. Read it back at the same address:
   ```
   go run . knowledge read --data '{"tier":"repo","name":"<store>","path":"<category>/<slug>.md"}'
   ```
4. Confirm it is absent from every other store, using the same path against each other name that
   step 1 reported:
   ```
   go run . knowledge read --data '{"tier":"repo","name":"<other-store>","path":"<category>/<slug>.md"}'
   ```
5. Confirm search attributes it correctly:
   ```
   go run . knowledge search "<distinctive phrase from the entry>"
   ```

**Expected result.**

- Step 2 returns `"error": false` and echoes back the tier, name and path that were requested.
- Step 3 returns the exact content staged in step 2.
- Step 4 returns `"code": "knowledge_entry_not_found"` for every other store.
- Step 5 returns a hit whose `tier` and `name` are the store written to.
- The entry's file exists under that repository's own footprint and under no other.
- **No step needed to be repeated, and no entry was created by editing a file directly.**

**Who / when.** The implementer, once per release that changes knowledge addressing, and any time
the `spek-knowledge` skill's contribute flow is edited, since that flow is what a person actually
follows.

**Result of the run performed for this change (2026-09-04).** **Pass.** Two entries were recorded
into the `docs` repository's store through the skill and the delivered commands:
`conventions/site-layout.md` and `decisions/frame-width-flow.md`. Both writes returned
`"error": false` on the first attempt. Both read back at
`{"tier":"repo","name":"docs","path":…}` with the content supplied; both returned
`knowledge_entry_not_found` against `{"tier":"repo","name":"spektacular",…}`; a search for a
distinctive phrase reported tier `repo` and name `docs`; and both files landed under
`spektacular-website/.spektacular/knowledge/`. Neither write needed a second attempt and neither
fell back to editing a file directly.

Worth recording alongside the result: this is the write that could not be expressed at all before
this change. The reproduction in the plan's research shows the same read returning "not found",
because the `docs` store was unreachable behind `spektacular` in a first-match scan.
