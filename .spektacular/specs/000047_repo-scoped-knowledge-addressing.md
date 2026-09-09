---
created_date: "2026-09-03"
status: completed
closed_date: "2026-09-03"
---

# Feature: 000047_repo-scoped-knowledge-addressing

## Overview

A project can bring several repositories together, and knowledge accumulates
in two different places: each repository has its own, about its own codebase,
and the project has shared stores that apply across all of them. Today those
two kinds are not distinguished, every store is labelled the same way, and
there is no way to say which one you mean. An entry written for one
repository silently lands in another, results never say where they came from,
and anything reading knowledge receives everything merged together whether it
is relevant or not.

This work introduces one way of saying which knowledge you mean: whether you
want the project's shared knowledge, a repository's own, or both, and which
one in particular. Writing an entry says where it belongs, so it lands there;
every result says where it came from; and retrieval can be narrowed to chosen
stores, so planning work on one repository is not loaded with the standing
rules of repositories it is not touching. Anyone recording knowledge in a
multi-repository project benefits, as does anyone whose planning quality or
cost suffers from unrelated knowledge crowding in.

## Requirements

- [x] **Knowledge is addressed by tier**
  Every knowledge request must state which tier it concerns: the project's
  shared knowledge, the repositories' own knowledge, or both.

- [x] **A store is identified by name within its tier**
  Each store must be identifiable by a name unique within its tier: a
  repository's own store by that repository's registered name, and a shared
  store by a name the project gives it. The same name identifies the same
  store on every operation that accepts one.

- [x] **A repository has exactly one knowledge store**
  A registered repository must contribute exactly one knowledge store,
  holding knowledge about that repository's own codebase. A repository must
  not be able to declare several, nor to name or label the one it has.

- [x] **Shared knowledge is declared by the project**
  Knowledge that is not about a single repository's codebase must be
  declared by the project, which may declare any number of such stores, each
  under its own name.

- [x] **Writing states the tier and the store**
  Users must state both the tier and the store name when writing a knowledge
  entry. A write that leaves either unstated is refused, whether or not what
  was given happens to identify a single store.

- [x] **A refused write explains how to retry**
  When a write is refused for want of a tier or a store name, the system
  must report the names available in the relevant tier, so the caller can
  reissue the write without having to inspect configuration separately.

- [x] **Reading addresses a single store**
  Users must be able to read a specific entry by stating its tier, its store
  name, and its location, so an entry is retrieved from the store it
  actually lives in.

- [x] **Every knowledge result reports its tier and store**
  Results returned by listing, searching, and reading must each report the
  tier and the store name they came from, so no result is returned without
  identifying where it came from.

- [x] **A search result can be retrieved without further disambiguation**
  Users must be able to take any single result from a search or a listing
  and read that exact entry using only the information the result carries,
  with no additional lookup and no risk of retrieving a same-named entry
  from a different store.

- [x] **Retrieval can be narrowed to chosen stores**
  Users must be able to narrow searching, listing, and the loading of
  always-applied knowledge to a chosen set of store names, in addition to
  choosing a tier. Narrowing must be optional, and omitting it must consider
  every store the chosen tier covers.

- [x] **Narrowing is honoured uniformly**
  No store may be exempt from a narrowing that does not name it, and no
  store may require naming in order to be included when no narrowing is
  given. A caller can therefore predict exactly which stores a request
  covers from the tier and names it gave.

- [x] **An outdated configuration fails loudly and says what to change**
  A repository or project whose configuration still declares knowledge in
  the superseded form must be rejected with an error naming the
  configuration file at fault, what was found, and what is now required.

- [x] **Recording an entry on the user's behalf shows where it will land**
  Before an entry is recorded on the user's behalf, the user must be shown
  the tier and store it will be written to, alongside its location, and must
  approve that destination.

- [x] **The published interface describes the addressing fields**
  Every knowledge operation must describe the tier, store name, and
  narrowing it accepts or returns in the machine-readable interface it
  publishes, so a caller can discover them without reading documentation or
  source.

- [x] **Knowledge supplied for this feature is recorded through it**
  Two knowledge entries supplied as input to this work, one recording the
  documentation site's layout conventions and one recording the reasoning
  behind its choice of text width, must be recorded into the documentation
  repository's own store using the delivered behaviour rather than by
  editing files directly.

- [x] **The documentation explains the addressing model**
  The product documentation must explain the two tiers and what belongs in
  each, how a store is named, how writing an entry states its destination,
  and how retrieval is narrowed.

## Constraints

- Cannot migrate, rewrite, or alias a configuration written in the
  superseded form. Such a configuration must be rejected with an actionable
  error and corrected by hand, in preference to a silent upgrade or an
  indefinite period in which both forms are accepted.

- Cannot resolve an under-specified write by choosing a store on the
  caller's behalf, including by falling back to configuration order, the
  first registered repository, the most recently used store, or the
  repository the current working directory sits in.

- Must not introduce a managed agent-rules section for the knowledge model.
  The existing managed guidance already directs the agent to the knowledge
  skill, and the detail belongs in that skill.

- Must honor the documentation repository's authoring conventions for any
  documentation written, including its prohibition on em dashes in authored
  prose and its rules on authoring page content through components rather
  than layout markup.

- Documentation changes must build and typecheck cleanly with the
  documentation site's existing toolchain, introducing no new site tooling
  or publishing mechanism.

## Acceptance Criteria

- [x] **A write missing the tier or the store name is refused**
  A write that omits either the tier or the store name fails and records
  nothing, both where several stores exist and where only one does. In each
  case a subsequent read for that entry finds nothing.

- [x] **The refusal names the available stores**
  The failure message from a refused write lists the store names available
  in the relevant tier, and those names match the ones reported by the
  command that enumerates configured knowledge stores.

- [x] **A write to a repository's own store lands there**
  A write naming the repository tier and a registered repository succeeds,
  and reading the same location back with that tier and name returns the
  content supplied. Reading the same location with any other name, or in the
  other tier, finds nothing.

- [x] **A write to a shared store lands there**
  In a project declaring a shared store, a write naming the project tier and
  that store's name succeeds, and reading it back with the same tier and
  name returns the content supplied.

- [x] **Two repositories with entries at the same location stay distinct**
  Given entries at the same location in two registered repositories, reading
  with the first repository named returns the first repository's content and
  reading with the second named returns the second's. Neither read returns
  the other's content.

- [x] **A repository declaring more than one store is rejected**
  A repository whose configuration declares more than one knowledge store,
  or names or labels the store it declares, is rejected with an error, and
  no knowledge operation against that project succeeds until it is
  corrected.

- [x] **Listing reports tier and store for every entry**
  Listing knowledge in a project with two registered repositories and a
  shared store returns entries from all three, and every entry reports a
  tier and a store name. No entry is returned without both.

- [x] **Search results report tier and store**
  A search whose query matches entries in more than one store returns hits
  from each, and every hit reports the tier and store name it came from
  alongside the information it reports today.

- [x] **A search hit can be read back exactly**
  Taking any single hit from that search and issuing a read using only the
  identifying information that hit carries returns the same entry the hit
  excerpted, in every case, including when an entry of the same name exists
  in another store.

- [x] **Choosing a tier selects only that tier**
  A search restricted to the repository tier returns hits only from
  repositories' own stores; the same search restricted to the project tier
  returns hits only from shared stores; and the same search across both
  returns hits from all of them.

- [x] **Narrowing to chosen names excludes the rest**
  A search narrowed to one store name returns hits only from that store,
  and the same search with no narrowing returns hits from every store its
  tier covers. Listing and the loading of always-applied knowledge behave
  the same way under the same narrowing.

- [x] **No store is implicitly included or excluded**
  Loading always-applied knowledge narrowed to one repository returns that
  repository's entries and no others, including none from any shared store.
  Loading it across both tiers with the same narrowing plus a shared store's
  name returns exactly those two stores' entries.

- [x] **An outdated configuration is rejected with an actionable error**
  A project whose configuration, or whose registered repository's
  configuration, declares knowledge in the superseded form fails on any
  knowledge operation, and the error names the configuration file, what was
  found, and what is required. Running the operation again without editing
  the file produces the same failure, and the file's contents are unchanged
  afterwards.

- [x] **Correcting the configuration resolves the failure**
  After editing that configuration into the required form, the same
  knowledge operation succeeds against that store.

- [x] **A scaffolded repository needs no correction**
  Scaffolding a repository into a project produces a configuration that is
  accepted as-is, and writing to and reading from that repository's own
  store succeeds with no further edit.

- [x] **An entry recorded on the user's behalf is approved with its destination**
  When an entry is recorded on the user's behalf, the destination presented
  for approval beforehand states a tier and a store name as well as a
  location, and the entry is subsequently readable at exactly that
  destination.

- [x] **The published interface advertises the addressing fields**
  The machine-readable interface published by each knowledge operation
  includes the tier, store name, and narrowing fields it accepts or returns,
  and a caller relying only on that published interface can issue a fully
  addressed write successfully.

- [x] **The documentation covers the addressing model**
  The published documentation explains the two tiers and what belongs in
  each, how stores are named, how a write states its destination, and how
  retrieval is narrowed. The documentation site builds and typechecks
  without errors.

- [x] **The supplied layout entries are readable from the documentation repository**
  The two supplied knowledge entries are recorded using the delivered
  behaviour rather than by editing files directly. Reading each back
  addressed to the documentation repository's own store returns its content,
  reading either from any other store finds nothing, and searching for them
  reports that store as their source.

## Technical Approach

- The addressing vocabulary the team settled on is a tier (`project`, `repo`
  or `all`), a `name` identifying one store within a tier, and a flat
  `filter` list of those same names for narrowing retrieval. `name` is the
  singular, exact form used where exactly one store must be hit; `filter` is
  the plural, narrowing form used across many. Both draw on the same
  namespace, so a name means the same thing wherever it appears.

- Keeping `filter` a flat list is deliberate. Tier, store and category are
  independent axes, so a further axis would slot in as a sibling field
  rather than a key inside a nested filter object, and a flat list keeps the
  command-line form as plain repeatable flags rather than an encoded object.

- A repository's knowledge declaration is expected to collapse to a single
  provider block, matching the shape its changelog declaration already uses
  in the same file, which removes the vestigial per-source label rather than
  leaving a one-element list that must always say the same thing.

- Shared stores keep a list shape in the project's own configuration, with
  each entry carrying the name that identifies it.

- Resolution from an address to a store is the single point the current
  defect lives at, and concentrating the change there is expected to close
  it for every caller that goes through it at once.

- The knowledge skill is the agent's route to this behaviour and is best
  revised alongside the commands, so its propose-then-confirm flow shows the
  destination it will write to. Having it enumerate the configured stores
  before proposing is the straightforward way to get there, since it already
  runs that enumeration for other reasons.

- Documentation for this fits on the existing knowledge base reference page
  of the documentation site, extending the section that already explains
  where knowledge sources are declared, rather than on a new page.

## Success Metrics

- **No knowledge entry lands in the wrong store.** Across every write
  performed after delivery, the store holding the resulting entry is the one
  the author named. A write that does not fully state its destination fails
  visibly instead of succeeding somewhere else, so the count of silently
  misplaced entries is zero rather than merely low.

- **The dogfood write succeeds first time.** Recording the two supplied
  layout entries needs no second attempt to correct where they landed, and
  no fallback to editing files directly.

- **Planning loads less irrelevant knowledge.** For planning work scoped to
  one repository, the always-applied knowledge loaded is limited to the
  stores the plan names, measured as a reduction in loaded entries against
  the same task run without narrowing.

- **A caller can predict what a request covers.** Given a tier and a set of
  names, the stores a request considers can be worked out without consulting
  configuration or knowing which store is declared where, because no store
  is implicitly included or excluded.

## Non-Goals

- **Any movement, re-attribution, deletion, or migration of existing
  entries.** Entries already written keep their current location and
  content. Nothing relocates an entry from one store to another, including
  entries misplaced by the defect this work fixes, nothing re-attributes
  existing entries, and nothing adds a way to delete one. Correcting a
  misplaced entry is a manual matter.

- **Filtering retrieval by category.** Narrowing by the kind of knowledge, a
  gotcha or a decision, is a useful and independent axis, but search results
  are ranked and excerpted rather than loaded in full and already carry a
  category label, so the volume problem this work addresses does not require
  it. It can be added later as one more optional field without a breaking
  change.

- **Tier-aware addressing for specs, plans, or changelog records.** Those
  artifacts have their own stores and their own relationship to a project's
  repositories, and none of them exhibited the collision this work
  addresses. Extending the idea to them is a separate question.

- **Deduplicating identical knowledge held in more than one store.** The
  knowledge lookup flow already collapses byte-identical entries when
  consolidating an answer. Making the stores themselves aware of duplication
  is not part of this work.
