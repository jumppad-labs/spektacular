---
created_date: "2026-09-04"
status: completed
closed_date: "2026-09-04"
---

# Knowledge is addressed by tier and store name

## What was built

Spektacular's knowledge base now has a way to say *which* knowledge you mean.

A project can bring several repositories together, and knowledge accumulates in two places: each
repository has its own, about its own code, and the project has shared stores that span them. Every
store used to be labelled the same way, with no way to tell them apart, so an entry written for one
repository silently landed in another, results never said where they came from, and anything reading
knowledge received everything merged together.

Every request now states a **tier** — the project's shared knowledge, the repositories' own, or both
— and names one **store** within it.

- **Writing states where an entry belongs, and lands there.** A write that leaves the tier or the
  store name unstated is refused outright rather than landing somewhere plausible, and the refusal
  lists the stores available in that tier so it can be reissued immediately.
- **Every result reports its origin.** Listing entries, search hits, read results, conventions and
  always-applied entries all carry the tier and store they came from. A hit therefore carries exactly
  what a read needs, so a result can be fetched back without a second lookup and with no risk of
  retrieving a same-named entry from a different store.
- **Retrieval can be narrowed, and the narrowing is honoured with no exceptions.** Searching,
  listing, conventions and the always-applied load take `--tier <project|repo|all>` and a repeatable
  `--filter <name>`, replacing the narrower `--repo` option. A store that is not named is not
  included, and no store is included merely because it was hard to attribute. Planning work on one
  repository therefore stops loading the standing rules of repositories it is not touching.
- **Configuration says which store is which.** A repository declares its single knowledge store as
  one provider block in `repo.yaml`, matching the `changelog:` block beside it; the project names
  each of its shared stores under `knowledge.sources`.
- **The published interface describes all of it.** Each knowledge command's `--schema` output now
  advertises the tier, store name and narrowing fields it accepts or returns, so a caller can
  discover the addressing without reading documentation or source.
- **The agent's route was rewritten to match.** The `spek-knowledge` skill works in tiers and store
  names throughout, and before recording an entry on a user's behalf it shows the tier and store it
  will be written to alongside the path, so approval is given for a destination rather than a
  filename. The planning workflow's knowledge load moved onto the new narrowing.
- **The documentation explains the model** on the site's knowledge-base reference page, across five
  further site pages whose examples would otherwise instruct a reader to write a configuration the
  tool now rejects, and in the command repository's own readme and in-repo knowledge document.

Two knowledge entries about the documentation site's layout were recorded into that repository's own
store through the delivered commands, which is both the last requirement and the proof the addressing
works: it is precisely the write that could not be expressed before.

## Why it matters

Anyone recording knowledge in a multi-repository project benefits, as does anyone whose planning
quality or cost suffers from unrelated knowledge crowding in. The defect was silent: an entry written
for one repository became unreachable behind another, and nothing reported that anything had gone
wrong. In this project's own knowledge base, reading a convention that existed only in the
documentation repository's store returned "not found", because that store sat behind the command
repository's in a first-match scan. That same read now succeeds when addressed to `docs`, and
correctly reports not-found when addressed to `spektacular`.

## Breaking change

Configuration written in the previous form is **rejected with an actionable error rather than
migrated**, so this is a breaking change existing projects correct by hand, once. Any knowledge
operation against a stale file fails with an error naming the file, what was found in it, and the
block now required; nothing on disk is rewritten, so the same failure repeats until a person edits
it. Two one-time edits:

1. In each registered repository's `repo.yaml`, replace the `knowledge.sources` list with a single
   provider block:
   ```yaml
   knowledge:
     provider: file
     config:
       location: knowledge
   ```
2. In the project's `config.yaml`, rename each shared store's `scope:` key to `name:`. A project
   declaring no `knowledge:` block needs no change.

A repository scaffolded by the tool is accepted with no correction, and upgrading from the older
single-file configuration still produces a repository configuration that loads cleanly.

## Deviations from the plan

- **Two changes moved forward from later phases into Phase 1.1**, because Go will not compile a
  half-changed package and two of that phase's own acceptance criteria were untestable otherwise:
  `SourceInfo` gained its tier and name fields (planned for 1.3), and the read/write `--data` payload
  moved onto the three-part address (planned for 1.4). Everything else in those phases landed where
  planned.
- **Phase 2.4's configuration corrections were made during Phase 2.3**, because the plan's own
  dependency section makes the ordering a hard constraint: every knowledge command in the working
  tree fails from the moment the rejection lands. Phase 2.4 became a verification pass.
- **Two harbor end-to-end fixtures had to be corrected** that no phase named
  (`tests/harbor/implement-workflow/environment/{repo,docs-repo}.yaml`). They are not exercised by
  `go test ./...` and worked only by accident, since the superseded key parsed into an empty block
  whose synthesised default happened to land on the same path.
- **Three pre-existing defects were fixed along the way**, each on a criterion path of this work: a
  failed knowledge read surfaced as `internal_error` with an empty `next_action`, violating the
  repository's own error convention; `knowledge search --schema` required a positional query, so the
  one command whose narrowing matters most had an interface a caller could not discover; and
  `aggregateKnowledgeSources` wrapped every footprint failure in a `repo_footprint` error whose
  remediation ("run `repo add` to repair") could not fix a superseded knowledge block.
- **Phase 3.1's regeneration touched two files beyond its scope**, both put to the user, who chose to
  keep them: `.bob/skills/spek-implement/SKILL.md` and `.bob/skills/spek-plan/SKILL.md` had been
  stale since an earlier commit regenerated only the `.claude` copies. A stray-period typo in the
  source template was fixed at the same time, at the user's request.
- **The two supplied knowledge entries were written from scratch.** The plan anticipated this: the
  original drafts no longer existed, so both bodies were drafted, presented in full, and approved
  before either write. They were grounded in the site's actual stylesheet and components rather than
  in the plan's one-line summary.
