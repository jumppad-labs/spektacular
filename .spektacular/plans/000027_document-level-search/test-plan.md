# Test Plan: 000027_document-level-search

The three success metrics from the spec are each split per the plan's Testing
Approach: the **mechanism** is covered by automated behavioural tests on
fixtures (in `internal/store/search_test.go`, `internal/knowledge/set_test.go`,
and `cmd/knowledge_test.go`), and the **live-knowledge-base outcome** is a
manual check captured here. The procedures below run against this repository's
own knowledge base (the `project` scope at `.spektacular/knowledge`), which the
automated tests deliberately never read.

A note on the spec's example queries: the spec illustrated success metric #1
with "workflow steps", "workflow state", and "interrupted workflow". The first
two work as written. "interrupted workflow" returns zero results because the
word "interrupted" (and even the stem "interrupt") appears nowhere in this
knowledge base — with document-level AND-matching, a query word that exists in
no entry can match nothing. This is correct behaviour, not a scoring failure;
the third example query is replaced below with **"scope label"**, a real
two-word phrase whose words are genuinely scattered across different lines of
the relevant entry, which is the property metric #1 is meant to demonstrate.

Who/when: run by the implementer (or a reviewer) once, pre-merge, on the
`f-ripgrep-replace` branch with the feature built. All commands assume the
repository root as the working directory; substitute `{{command}}` with the
configured CLI invocation (`go run .` in this repo).

---

## Metric 1 — The motivating queries now succeed

**What to measure:** A multi-word query whose words are scattered across
different lines of an entry now finds that entry and ranks it first, where the
old per-line search returned nothing.

**How:**

```
go run . knowledge search "workflow steps"
go run . knowledge search "workflow state"
go run . knowledge search "scope label"
```

For the "scope label" case, confirm the words really are scattered (so the
match is only possible at the document level):

```
grep -in 'scope' .spektacular/knowledge/architecture/initial-idea.md
grep -in 'label' .spektacular/knowledge/architecture/initial-idea.md
```

**Expected result:**

- `workflow steps` → `architecture/workflow-steps.md` is the **first** result
  (observed score 38).
- `workflow state` → `architecture/workflow-steps.md` is the **first** result
  (observed score 37).
- `scope label` → `architecture/initial-idea.md` is the **first** result
  (observed score 9), and the two `grep` calls show "scope" and "label" on
  disjoint line numbers (observed: scope on 20/214/502/587/731, label on
  312/315/616/620 — never the same line). Each result carries a non-empty
  `excerpts` array and a `title`.

**Pass condition:** each query returns the relevant architecture entry ranked
first, and the "scope label" words are confirmed never to co-occur on a single
line. If a noise entry (e.g. a raw JSON transcript file) ever outranks the
relevant entry for a motivating query, STOP and raise it with the user — match
strength is deliberately simple occurrence counting, and re-ranking or
knowledge-base curation is out of scope for this plan.

## Metric 2 — Broad queries become triageable

**What to measure:** A broad single-word query returns one ranked result per
matching entry, not dozens of repetitive per-line hits.

**How:**

```
go run . knowledge search "workflow" | python3 -c "import json,sys; print(len(json.load(sys.stdin)['hits']))"
```

**Expected result:** a single-digit count — one result per matching document —
with `architecture/workflow-steps.md` ranked first (observed: 9 ranked
results, where the old per-line search produced 60 line hits across 9
documents).

**Pass condition:** the result count equals the number of matching documents
(not lines), is single-digit, and the strongest-matching entry is first.

## Metric 3 — No discovery regression

**What to measure:** A single-word query still surfaces exactly the entries it
surfaced before — one result per document that contains the word, with
conventions and binary files excluded.

**How:**

```
# ranked results for a single word:
go run . knowledge search "workflow" | python3 -c "import json,sys; print(len(json.load(sys.stdin)['hits']))"
# distinct text files containing that word, excluding conventions/:
grep -rilI "workflow" .spektacular/knowledge | grep -v "/conventions/" | wc -l
```

**Expected result:** the two counts are equal (observed: 9 and 9). The
document set returned by search equals the set of non-binary, non-convention
files that contain the word.

**Pass condition:** counts match exactly. The single-word regression guarantee
is additionally proven by construction in the automated oracle test
`TestSearch_SingleWordMatchesSameFiles`; this live check confirms it on the
real knowledge base.

## Round-trip guard (search → read), live KB

**What to measure:** A returned hit's `scope` and `path` still fetch the full
entry through the unchanged read command.

**How:**

```
go run . knowledge read --data '{"scope":"project","path":"architecture/initial-idea.md"}'
```

(using the top hit's scope/path from the "scope label" query)

**Expected result:** the full entry body is returned (observed content length
~38.6 KB), confirming the search→read contract is unchanged.

**Pass condition:** read returns the entry's full content without error.
