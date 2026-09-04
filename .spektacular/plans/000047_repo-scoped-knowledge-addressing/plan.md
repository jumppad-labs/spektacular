---
created_date: "2026-09-03"
status: completed
closed_date: "2026-09-03"
---

# Plan: 000047_repo-scoped-knowledge-addressing

<!-- Metadata -->
<!-- Created: 2026-09-03T10:39:14Z -->
<!-- Commit: 0c0cb4f -->
<!-- Branch: f-project-repos -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

This work gives Spektacular's knowledge base a way to say which knowledge you mean. A project can bring several repositories together, and knowledge accumulates in two places: each repository has its own, about its own code, and the project has shared stores that span them. Today every store is labelled the same way and there is no way to distinguish them, so an entry written for one repository silently lands in another, results never say where they came from, and anything reading knowledge receives everything merged together.

Every request now states a tier, the project's shared knowledge or the repositories' own or both, and names one store within it. Writing an entry says where it belongs and lands there; a write that leaves either unstated is refused and told which stores are available; every result reports the tier and store it came from; and retrieval can be narrowed to chosen stores, so planning work on one repository is not loaded with the standing rules of repositories it is not touching.

Anyone recording knowledge in a multi-repository project benefits, as does anyone whose planning quality or cost suffers from unrelated knowledge crowding in. Configuration written in the previous form is rejected with an actionable error rather than migrated, so this is a breaking change existing projects correct by hand once.

## Conventions

- **Error messages must describe the problem and suggest remediation** (`spektacular` repo) — this feature's central new behaviour is a refusal: a write missing its tier or store name must fail. Every refusal is built with `output.NewError(code, message).WithNextAction(...)` and the next action lists the store names available in the tier concerned, which is also what the spec's "a refused write explains how to retry" requires. It applies equally to the superseded-configuration rejection, which must name the file, the key found, and the block required. It also fixes an existing violation the research surfaced: today a failed knowledge read surfaces as `internal_error` with an empty `next_action`.
- **Passing tests are required before calling work done** (`spektacular` repo) — `go test ./...` is green on this branch today, and this change touches `internal/config`, `internal/knowledge`, `internal/store`, `internal/project`, `internal/repo`, `cmd` and `templates`, every one of which has tests that assert the current `scope` shape. No phase is complete while any of them fails, and `cmd/knowledge_test.go:637` must be inverted rather than deleted to keep the suite honest.
- **Plans must sketch content structure, not just summarize it** (`docs` repo) — this plan requires a substantial rewrite of `src/pages/knowledge-base.mdx`'s Configuration section and corrections across four more pages, so the phases covering them carry a `**Content outline**` or `**Content example**` block with headings in order and an illustrative excerpt, with the config keys and field names fixed by what the research verified rather than left for the implementer to invent.
- **No em dashes** (`docs` repo) — binds every piece of prose authored into the `docs` repo, which is all five pages this work touches, and the spec restates it as a constraint. Scoped to that repo: the CLI repo's own `README.md` and `CHANGELOG.md` use em dashes throughout and are not being restyled by this change.
- **MDX authoring conventions** (`docs` repo) — the four rules govern every edit to `src/pages/*.mdx`: no `<div>`, `<section>` or `class=` in a page body (verified by `grep -nE "<div|<section|class=" src/pages/*.mdx` returning nothing), content in slots rather than `body:` string props, a blank line either side of slot content, and code samples as fenced markdown blocks so the `knowledge:` YAML and `--data` examples pick up the shared `astro-expressive-code` chrome.
- **Alternate section background shading** (`docs` repo) — `knowledge-base.mdx` currently alternates `surface` false/true/false/true/false across its five `Section`s. If the addressing model warrants a new top-level section rather than an expansion of the existing Configuration section, its `surface` must be set explicitly to the opposite of the section before it.
- **Label before filename in file-scoped reference headings** (`docs` repo) — `configuration.mdx` documents both `config.yaml` and `repo.yaml`, and their `knowledge` keys diverge under this change. Any new or reworked heading keeps the plain-language label first ("Repository configuration: repo.yaml"), and each file's overview, example and key reference stay grouped as consecutive sections rather than interleaved.

Deliberately not applied: the `glossary` entries in both stores are category README stubs defining what a glossary is, with no project terms recorded yet, so there is no shared vocabulary to honour beyond the tier/name/filter words this spec itself introduces.


## Architecture & Design Decisions

Three directions were weighed. **Resolve addressing in the CLI layer only** (Low effort)
would pre-compute a unique packed label such as `repo:docs` in
`aggregateKnowledgeSources` and leave `knowledge.Set` scope-based, so `byScope` keeps
first-matching but now over labels that never collide. It fixes the misdirected write for
one caller's spelling and nothing else: results would report a packed string rather than a
tier and a store, callers would have to unpack it to satisfy "every result reports its
tier and store", and the resolution rule that caused the defect survives untouched one
layer down. **Push tiering into the store layer** (High effort) would give
`store.NewSourceStore` and `store.Hit` real tier and name fields so a store tags its own
hits. But `NewSourceStore` also backs the two changelog stores
(`cmd/storefile.go:95,133`), which have no knowledge tier and would be forced to invent
values. The chosen direction is **an addressing vocabulary owned by the knowledge layer**
(Medium effort): the concept lives exactly where the defect lives, and the store below and
the commands above stay as generic as they are today.

One naming problem has to be settled first. The knowledge package already uses `Tier` for a
category's **retrieval** tier, `always-applied` or `looked-up`, established when the category
registry was introduced. The spec fixes `tier` as the name of the addressing field, so the two
meanings must coexist. They sit on different objects and so never share a JSON envelope, which
means neither published field has to be renamed; in Go the existing type becomes `CategoryTier`,
giving the unqualified name to the addressing concept, and in documentation the older meaning is
always written as "retrieval tier" wherever the two appear near each other.

Concretely, `internal/knowledge` gains two small value types and `byScope`
(`internal/knowledge/set.go:288`) is deleted. `Address{Tier, Name}` names exactly one
store and backs `Read` and `Write`; `Selector{Tier, Filter}` names a set and backs
`Search`, `List`, `AlwaysAppliedEntries` and `Conventions`. `scopedStore` swaps its
`scope` and `repo` fields for `tier` and `name`, which `aggregateKnowledgeSources`
(`cmd/knowledge.go:180`) stamps during aggregation: a repo-declared block becomes tier
`repo` named after its registry entry, a project-declared source becomes tier `project`
named by its own `name` key. `Set.resolve(Address)` refuses an empty tier, an empty name,
a tier of `all`, and an unknown name, each through
`output.NewError(...).WithNextAction(...)` carrying the names available in the tier
concerned, so a refusal is directly actionable (**error messages must suggest
remediation**). Because resolution is one function on the read and write path, the
collision closes for every caller at once. `Selector` matching is deliberately uniform:
no store is exempt from a narrowing that does not name it, which means removing the
current exemption at `internal/knowledge/set.go:255` that lets project-owned sources load
regardless of the repo filter.

Every result shape reports where it came from. `store.Hit`, `knowledge.Entry`,
`knowledge.Convention`, `knowledge.AlwaysAppliedEntry` and `knowledge.SourceInfo` each
drop `Scope` and gain `Tier` plus `Name`. `Hit` is the one that spans layers, and it
follows the seam the code already uses for `Category`: the store leaves the fields empty
and `Set.Search` stamps them after merging (`internal/knowledge/set.go:136`), so the
changelog stores built through the same constructor are untouched. On the command surface,
`read` and `write` take `tier`, `name` and `path` in `--data`; `search`, `list`,
`always-applied` and `conventions` take a `--tier` flag defaulting to `all` and a
repeatable `--filter`, replacing the narrower `--repo` flag outright. So that narrowing is
still discoverable from the published interface, `commandSchema` (`cmd/spec.go:43`) gains
an `omitempty` `flags` block; every other command family's `--schema` output is byte-identical.

The configuration change is a shape change with a hard rejection rather than a migration.
A repo's `knowledge:` key collapses from a one-element `sources:` list to a single
provider block, matching the `changelog:` key already sitting beside it in the same file,
which removes the per-source label a repo must no longer be able to set. The project's
`knowledge.sources[]` keeps its list and renames `scope:` to `name:`. Both superseded
forms are rejected on load, following the pattern `rejectLegacyRepoAddress`
(`internal/config/config.go:275`) established for the removed `address` key: re-parse the
raw document into a loose shape, then return a `config_invalid` error naming the file, the
key found, and the exact block required. Nothing is aliased and nothing is rewritten in
place, so a stale configuration fails identically on every run until a human edits it.
This is a breaking change for existing projects, including this one: both live `repo.yaml`
files are in the superseded form and every knowledge command fails until they are
corrected by hand. The existing config migrator does not collide, because it is a
file-presence sniff that never parses a `knowledge:` block and synthesises a fresh default
from `NewDefaultRepoConfig` (`cmd/version.go:151,242`).

The work spans both registered repos. In **`spektacular`**
(`/home/nicj/code/github.com/jumppad-labs/spektacular`) it lands in `internal/config`
(both config shapes and both rejections), `internal/knowledge` (the vocabulary and
resolution), `internal/store` (the two `Hit` fields), `cmd/knowledge.go` (flags, `--data`
shape, schemas), `internal/project/init.go` and `internal/repo/footprint.go` (scaffolding
collapsed onto the single block), the `spek-knowledge` skill template with its two
generated copies, six step and agent template lines, and `README.md` plus
`docs/knowledge-base.md`. In **`docs`**
(`/home/nicj/code/github.com/jumppad-labs/spektacular-website`) it lands on
`src/pages/knowledge-base.mdx`, which gains the explanation of the two tiers, and on
`configuration.mdx`, `projects.mdx`, `getting-started.mdx`, `index.mdx` and
`extending.mdx`, whose existing examples and field listings become wrong the moment the
shapes change. The two supplied layout entries are then recorded into the `docs` repo's
own store through the delivered CLI, which is both the last requirement and the proof the
addressing works. Rejected options and the evidence behind the chosen one are in
`research.md#alternatives-considered-and-rejected`.


## Component Breakdown

**Knowledge address and selector (new, `spektacular`).** Two small value types owned by the
knowledge layer that carry the whole addressing vocabulary. The address names exactly one
store by tier and name and is what reads and writes travel on. The selector names a set of
stores by tier plus an optional flat list of names and is what searching, listing, and the
always-applied load travel on. Both are validated where they are constructed rather than at
each use site, so every operation shares one definition of what a valid tier is, what an
empty filter means, and which tiers may name a single store. They depend on nothing; every
other component below depends on them.

**Knowledge store resolver (replaces the existing first-match lookup, `spektacular`).**
Turns an address into exactly one configured store, or into a refusal. It is the single
place the current defect lives, so it is the single place the fix lands. It refuses an
absent tier, an unrecognised tier, a tier that names a set rather than one store, an absent
name, and a name no store in that tier carries; every refusal reports the names available in
the tier concerned so the caller can reissue without inspecting configuration. Its selector
counterpart decides membership for the fan-out operations and applies the same rule to every
store without exception, so a narrowing that does not name a store excludes it and an absent
narrowing includes everything the tier covers.

**Knowledge set (changed, `spektacular`).** The ordered collection of configured stores. Its
members stop carrying a scope label plus a separate repo attribution and instead carry a tier
and a name, which is the same information stated once. Read and write route through the
resolver; search, list, conventions, and the always-applied load route through the selector.
It keeps its existing responsibilities unchanged: ranking merged search results
deterministically, deriving an entry's category from its path, excluding always-applied
categories from search, and tolerating a store that lacks a category directory.

**Knowledge result envelopes (changed, `spektacular`).** The search hit, the list entry, the
convention, the always-applied entry, and the source descriptor. Each drops its single scope
label and gains a tier and a store name, so no result is returned without saying where it
came from and any single result carries enough to be read back exactly. The search hit is the
one shared with the generic store layer; it follows the seam already used for an entry's
category, where the store leaves the fields empty and the knowledge set fills them in after
merging, which keeps the changelog stores built on the same constructor untouched.

**Repo knowledge configuration (changed, `spektacular`).** A repository's declaration of its
own store. It collapses from a list of labelled sources to a single provider block, matching
the changelog declaration already beside it in the same file, so a repository can no longer
declare several stores or name the one it has. It owns the rejection of its own superseded
form.

**Project knowledge configuration (changed, `spektacular`).** The project's declaration of
its shared stores. It keeps its list shape, and each entry is identified by a name instead of
a scope label. Names are validated unique within the tier rather than globally. It owns the
rejection of its own superseded form.

**Superseded-configuration rejector (new, `spektacular`).** A load-time guard, one per
configuration file, that re-reads the raw document into a loose shape and fails when it finds
the removed key. It exists as its own component because the typed configuration no longer has
a field the old key could land in, so a silent no-op is the default outcome without it. Each
refusal names the file, the key it found, and the block now required, and nothing is rewritten
on disk, so the same file fails identically on every subsequent run. It reuses the pattern
already established for the removed repo address key rather than introducing a new one.

**Knowledge source aggregator (changed, `spektacular`).** Builds the effective store list for
a project from every registered repository's own declaration followed by the project's shared
declarations. It already stamps each repo-declared source with its repository's registry name;
it now also stamps the tier, which is the point where a repository's registry name becomes its
store name and a shared store's declared name becomes its own. Store-name uniqueness within a
tier is enforced here, where both tiers are visible at once.

**Knowledge command surface (changed, `spektacular`).** The eight knowledge subcommands.
Reading and writing take the tier and store name alongside the path in their existing JSON
input. Searching, listing, conventions, and the always-applied load take a tier option
defaulting to both tiers and a repeatable narrowing option, which replaces the narrower
repo-only option that exists today. It owns translating a refusal from the resolver into the
CLI's standard error envelope.

**Command schema descriptor (changed, `spektacular`).** The shared type every command family
uses to publish its machine-readable interface. It gains an optional block describing the
options a command accepts as flags rather than as JSON input, so the tier and narrowing
options are discoverable without reading documentation or source. The block is omitted when
empty, so every other command family's published interface is unchanged.

**Repository scaffolder (changed, `spektacular`).** The shared footprint writer that project
initialisation and repository registration both funnel through, together with the default
repository configuration it writes. It stops selecting a repository's knowledge store by
matching a scope label against a constant and reads the single declared block instead. A
freshly scaffolded repository must be accepted with no correction, which makes this the
component that keeps the new configuration shape and the scaffolder from drifting apart.

**Knowledge skill (changed, `spektacular`).** The agent's route to this behaviour. Its
lookup, contribute, and update flows all state a tier and a store name where they previously
stated a scope, and its propose-then-confirm checkpoint shows the tier and store an entry
will be written to alongside its location before asking for approval. It continues to
enumerate the configured stores before proposing, which is now also what supplies the names
it must choose between. Its two generated per-agent copies are regenerated so they do not
ship stale.

**Workflow step and agent-rules templates (changed, `spektacular`).** The planning and
implement steps that invoke knowledge commands, and the two managed agent-rules sections that
delegate to the skill. The planning discovery step's repo-scoped always-applied load moves to
the new tier and narrowing options; the remaining references update the vocabulary they use to
describe what the skill owns. No new managed agent-rules section is introduced.

**Product documentation (changed, both repos).** The documentation site's knowledge-base
reference page owns the explanation of the two tiers, what belongs in each, how a store is
named, how a write states its destination, and how retrieval is narrowed. The site's
configuration reference, multi-repository guide, getting-started tutorial, landing page, and
extending guide each own a smaller correction where they currently show the superseded
configuration shape or the superseded result fields. The command repository's own readme and
in-repo knowledge-base document carry a second copy of the same model and are corrected in
step.

**Supplied knowledge entries (new content, `docs`).** Two entries recording the documentation
site's layout conventions and the reasoning behind its text width. They are content rather
than code, and they are the delivery's own proof: they are recorded into the documentation
repository's store through the finished command surface rather than by editing files.


## Data Structures & Interfaces

### The addressing vocabulary

Three tier values, one address, one selector. `Tier` is a named string type so an invalid
value cannot be constructed by accident from an arbitrary string.

The name `Tier` is currently taken inside the knowledge package by a category's **retrieval**
tier, `always-applied` or `looked-up`. That existing type is renamed `CategoryTier` (with
`CategoryTierAlwaysApplied` and `CategoryTierLookedUp`) so the unqualified name goes to the
addressing concept this spec centres on. Only Go identifiers change: the `Category` struct keeps
its `Tier` field and its `tier` JSON key, so the interface `knowledge categories` publishes is
byte-identical. The two `tier` fields never appear on the same object.

```go
type Tier string

const (
    TierProject Tier = "project" // the project's shared stores
    TierRepo    Tier = "repo"    // the registered repositories' own stores
    TierAll     Tier = "all"     // both; valid for fan-out only
)

// Address names exactly one store. Read and Write travel on it.
type Address struct {
    Tier Tier   `json:"tier"`
    Name string `json:"name"`
}

// Selector names a set of stores. Search, List, Conventions and
// AlwaysAppliedEntries travel on it. An empty Filter covers every store the
// tier reaches; a Filter naming a store the tier does not reach is an error,
// not a silent empty result.
type Selector struct {
    Tier   Tier     `json:"tier"`
    Filter []string `json:"filter"`
}
```

`Address.Validate()` rejects an empty tier, an unrecognised tier, `TierAll`, and an empty
name. `Selector.Validate()` rejects an empty or unrecognised tier and treats `TierAll` as
valid. Both return the CLI's structured error carrying a next action, so the same refusal
text reaches a caller whether it came from the command surface or from a direct call.

### The knowledge set's surface

Every entry point changes shape, and the change is uniform: singular operations take an
`Address`, fan-out operations take a `Selector`.

```go
func (s *Set) Read(addr Address, path string) ([]byte, error)
func (s *Set) Write(addr Address, path string, content []byte) error

func (s *Set) Search(query string, sel Selector) ([]store.Hit, error)
func (s *Set) List(sel Selector) ([]Entry, error)
func (s *Set) Conventions(sel Selector) ([]Convention, error)
func (s *Set) AlwaysAppliedEntries(sel Selector) ([]AlwaysAppliedEntry, error)

func (s *Set) Sources() []SourceInfo
```

`AlwaysAppliedEntries` replaces its variadic `repos ...string` parameter, which expressed
one axis of narrowing and exempted stores it could not attribute. `Conventions` gains a
selector it did not have, so the narrowing rule is honoured by every retrieval path with
no exceptions.

### Result envelopes

Each of these drops `Scope` and gains the same two fields, so a caller reads them
identically wherever a result appears. `store.Hit` is the one shared with the generic store
layer; the store leaves `Tier` and `Name` empty and the knowledge set stamps them, exactly
as it already does for `Category`.

```go
type Hit struct {                     // internal/store
    Tier     string   `json:"tier"`   // stamped by the knowledge layer
    Name     string   `json:"name"`   // stamped by the knowledge layer
    Path     string   `json:"path"`
    Title    string   `json:"title"`
    Excerpts []string `json:"excerpts"`
    Score    float64  `json:"score"`
    Category string   `json:"category"`
    Checksum string   `json:"checksum"`
}

type Entry struct { Tier, Name, Path string }
type Convention struct { Tier, Name, Path, Content string }
type AlwaysAppliedEntry struct { Tier, Name, Path, Content, Category string }
type SourceInfo struct { Tier, Name, Provider, Location string }
```

`SourceInfo` loses its separate `Repo` field: a repo-tier store's name *is* its repository's
registry name, so carrying both would be two spellings of one fact. The pair `{tier, name,
path}` on a hit or a list entry is exactly the input a read requires, which is what lets any
single result be retrieved without further disambiguation.

### Configuration shapes

A repository declares one block, matching the `changelog:` block already beside it in the
same file. The project declares a list, each entry named.

```yaml
# repo.yaml — a repository's own store: no list, no label
knowledge:
  provider: file
  config:
    location: knowledge

# config.yaml — the project's shared stores: a list, each named
knowledge:
  sources:
    - name: team
      provider: file
      config:
        location: ~/work/team-knowledge
```

```go
// RepoKnowledgeConfig is a repo's single knowledge store declaration.
type RepoKnowledgeConfig struct {
    Provider string              `yaml:"provider"`
    Config   FileKnowledgeConfig `yaml:"config"`
}

// SourceConfig is one project-declared shared store. Tier and Repo are
// stamped during aggregation and never serialized.
type SourceConfig struct {
    Name     string              `yaml:"name"`
    Provider string              `yaml:"provider"`
    Config   FileKnowledgeConfig `yaml:"config"`
    Tier     Tier                `yaml:"-"`
    Repo     string              `yaml:"-"`
}
```

### Command input and published interface

`read` and `write` carry the address in their existing JSON input; the tier and name are
required and a request missing either is refused before any store is touched.

```jsonc
// knowledge read/write --data
{ "tier": "repo", "name": "docs", "path": "conventions/site-layout.md" }
```

`search`, `list`, `conventions` and `always-applied` carry narrowing on flags:
`--tier <project|repo|all>` defaulting to `all`, and a repeatable `--filter <name>`. To keep
that discoverable from the published interface, the schema envelope every command family
shares gains one optional member:

```go
type commandSchema struct {
    Input  *schemaObj             `json:"input"`
    Output *schemaObj             `json:"output"`
    Flags  map[string]*schemaProp `json:"flags,omitempty"` // new
}
```

`Flags` is omitted when empty, so every command family other than knowledge publishes a
byte-identical interface to the one it publishes today.


## Implementation Detail

**One resolution point, deliberately narrowed.** The knowledge layer today offers a single
lookup that scans its stores for the first whose label matches and returns it, and both
reading and writing go through it. That function disappears. In its place sit two functions
with opposite contracts: one takes an address and must return exactly one store or an error,
the other takes a selector and returns the subset a fan-out should cover. Nothing else in
the package is allowed to reach into the store list directly. A developer reading the
package will find that every question about *which store* is answered in one place, and that
the singular and plural paths cannot be confused because they take different parameter types.

**Refusal as a first-class outcome.** Under the current design an under-specified request
succeeds against whichever store happened to sort first, which is why the bug was silent.
Under the new design an under-specified request is a refusal that carries the names available
in the tier concerned. This follows the codebase's existing structured-error pattern rather
than introducing a new one: a code, a message naming the problem, and a next action giving a
runnable correction. The change of note is that these errors originate inside the knowledge
package rather than only at the command surface, because the resolver is where the
information about available names lives; the package therefore takes on the same dependency
on the shared error type that the configuration package already carries.

**Uniform narrowing, with the exemption removed.** The always-applied reader currently
filters by repository name and deliberately exempts any store it cannot attribute to a
repository. That exemption is deleted. Membership is decided by one predicate applied to
every store without special cases: the store's tier must fall under the requested tier, and
if a narrowing list is present the store's name must appear in it. The visible consequence is
that a project-owned store no longer loads regardless of what was asked for, which is the
behaviour change the spec's predictability requirement demands. The same predicate is applied
to searching, listing and the conventions view, so no retrieval path is a special case.

**Attribution moves from a repair to a construction step.** Today the aggregator stamps a
repository's registry name onto each source it contributes, after the fact, alongside a
separately-declared scope label that may or may not agree with it. Now the aggregator is the
only place where a store's identity is established at all: a repository's contribution is the
repo tier under the repository's registry name, and the project's contributions are the
project tier under their declared names. Uniqueness within a tier is checked here, where both
tiers are visible together, rather than inside either configuration file's own validation
where only one is.

**Two superseded configuration shapes, rejected the same way.** Because the typed
configuration no longer has a field the old keys could land in, a stale file would otherwise
parse cleanly and behave as though nothing were declared. Each configuration file therefore
gains a load-time guard that re-reads the raw document into a loose shape and fails when the
removed key is present. This is not a new pattern: it is the one already used for the removed
repository address key, applied twice more. Nothing is rewritten on disk, so a stale file
fails identically on every run until a human edits it, and the failure names the file, the key
found, and the block now required.

**A published interface that covers flags.** Narrowing is expressed as command-line flags,
which the schema envelope shared by every command family cannot currently describe: it
publishes only the JSON input shape. The envelope gains an optional flags member, omitted
when empty. This is a small widening of a type used across the whole CLI, and the reason it is
worth doing here rather than moving narrowing into JSON input is that the search command takes
its query positionally and is invoked that way by the skill and by four workflow step
templates; changing that invocation to satisfy a schema-shape constraint would be the tail
wagging the dog.

**Scaffolding stops matching on a magic label.** Both the project initialiser and the shared
repository footprint writer currently locate "the repository's own store" by comparing a
declared scope label against a constant, then looping over what is nominally a list. With one
declared block per repository, both collapse to reading that block. This removes a class of
drift where the scaffolder and the configuration shape could disagree, and it is what makes
the spec's requirement that a freshly scaffolded repository needs no correction hold by
construction rather than by coincidence.

**Documentation and agent guidance are treated as part of the interface, not as follow-up.**
The knowledge skill is the agent's only sanctioned route to these commands, and its
propose-then-confirm checkpoint is where the spec's requirement to show a destination before
writing is satisfied. It is edited in its source template, and its per-agent generated copies
are regenerated in the same change so a stale copy cannot ship. The same reasoning applies to
the workflow step templates that invoke these commands directly and to the documentation
pages whose examples would otherwise instruct readers to write a configuration the tool now
rejects.


## Dependencies

### Internal packages (`spektacular`)

- **`internal/config`** — owns both configuration shapes and both load-time rejections.
  **Changes.** The repository knowledge declaration collapses to a single provider block,
  the project's shared sources rename their identifying key, and two raw-document guards
  are added. It already imports the shared error type, so the rejections need no new
  dependency.
- **`internal/knowledge`** — owns the addressing vocabulary, resolution, and every result
  envelope. **Changes.** Largest single change in the plan: two new value types, the
  first-match lookup deleted, every public method's signature reshaped, and the filter
  exemption removed. It takes on a new import of the shared error package so refusals can
  carry a next action.
- **`internal/store`** — provides the generic read/write/search interface and the search
  hit. **Changes.** Two fields on the hit, left empty by the store and stamped by the
  knowledge layer. The store constructor's label parameter and the changelog stores built
  through it are deliberately untouched.
- **`internal/output`** — provides the structured error envelope with resource and next
  action. **No changes.** It is used as-is, and it is what makes a refusal actionable.
- **`internal/repo`** — provides the registry: which repositories exist, their names, and
  where each one's code and footprint resolve to. It is the authority supplying every
  repo-tier store name. **Changes** limited to the footprint writer, which stops selecting
  a repository's store by matching a label against a constant.
- **`internal/project`** — owns project initialisation and the knowledge directory
  scaffolding. **Changes** limited to the same label-matching collapse.
- **`internal/agent`** — renders skill templates for each supported agent, substituting the
  configured command. **No changes**; it is the mechanism by which the edited skill template
  reaches its per-agent copies, and it explains why those copies must be regenerated rather
  than hand-edited.
- **`cmd`** — the command surface, the source aggregator where tier and name are stamped,
  and the schema envelope shared by every command family. **Changes** to the knowledge
  commands, the aggregator, and one optional member added to the shared envelope.
- **`templates`** — the embedded workflow steps, skills, and agent-rules sections, with an
  existing suite of content-assertion tests over that corpus. **Changes** to the knowledge
  skill and the step and agent-rules lines that describe knowledge addressing.

### External libraries

- **`gopkg.in/yaml.v3`** — already a direct dependency; supplies both the typed decode and
  the loose second pass the superseded-form guards need. **No version change.**
- **`github.com/spf13/cobra`** — already a direct dependency; supplies the repeatable string
  array flag the narrowing option uses, which the always-applied command already relies on
  today. **No version change.**
- **`github.com/stretchr/testify`** — already the test assertion library throughout.
  **No version change.**
- **No new module dependency is introduced in either repository.** The documentation site
  adds no tooling, integration, or package: its existing Astro and Tailwind toolchain builds
  and typechecks the changed pages unchanged, which the spec states as a constraint.

### Upstream specs and prior plans

- **Spec `000047_repo-scoped-knowledge-addressing`** — the source of truth for this plan.
  Already written and closed; nothing about it needs to land first.
- **`go run . plan file list` returns no plans**, so there is no prior plan document this
  one depends on or supersedes.
- **`000039_project-level-capabilities`** (shipped) — introduced multi-repository projects
  and the aggregation of each repository's knowledge into one set. It is what created the
  ambiguity this work removes, and its aggregation code is the seam being changed. No
  further change to it is required.
- **`000042_repo-self-describing-metadata`** (shipped) — moved repository metadata into each
  repository's own configuration and removed the registry's address key, establishing the
  reject-a-removed-key pattern this plan reuses twice. Its rejection helper is the model, not
  a dependency to modify.
- **`000046_relocatable-repo-footprint`** (shipped, on this branch) — established that a
  repository's Spektacular files may sit outside its code and that relative knowledge
  locations resolve against each repository's own root. This plan depends on that resolution
  being correct and does not change it.
- **`000045_config-file-migration`** (shipped) — the legacy single-file to split-file
  migration. Verified **not** to conflict: it is a file-presence check that never parses a
  knowledge block, and it writes a fresh default repository configuration, so it produces the
  new shape automatically once the default changes. No coordination is needed, but its tests
  assert the generated file's contents and will need updating alongside.

### Environment and cross-repository ordering

- **The `docs` repository must be present on disk** for the final requirement to be met, since
  the two supplied entries are written into its own store. It is registered and materialised
  at its recorded root today.
- **Both live `repo.yaml` files are in the superseded form.** They are not an external
  dependency but they are a hard ordering constraint: every knowledge command in this project
  fails from the moment the rejection lands until both files are corrected, so that correction
  must accompany the configuration change rather than follow it. The dogfood write depends on
  it.
- **The documentation site's `node_modules` must be installed** to run its build and
  typecheck. Present locally; CI installs it from the lockfile.


## Testing Approach

### Shape of the suite

The work is tested at three existing levels, all inside the command repository's Go test
suite, following the conventions already in place there: table-free tests using the
project's assertion library, temporary project trees built per test, and expectations
written out by hand rather than derived from the code under test.

- **Unit tests on the knowledge layer** carry the most coverage, because the addressing
  vocabulary and the resolution rule are where the defect lives and where every caller's
  behaviour is decided. They build small multi-store sets directly and assert resolution,
  refusal, and narrowing without going near the command line.
- **Command-surface tests** exercise the real command tree end to end against a temporary
  project on disk, so the JSON envelope a caller actually receives is what is asserted,
  including the shape of a refusal. These are where the acceptance criteria about writes
  landing in the right store, results reporting their origin, and a search hit reading back
  exactly are pinned.
- **Configuration tests** cover the two new shapes and the two rejections, including that a
  rejected file is left byte-identical on disk and fails the same way on a second run.
- **Template content tests** extend an existing suite that asserts what the embedded
  workflow steps and skills say, pinning that the knowledge skill and the planning steps
  describe the new addressing rather than the old vocabulary.
- **Documentation tests** already assert the command repository's readme and changelog
  content; they gain assertions for the addressing model and the breaking-change note.

### Load-bearing assertions, in plain language

- A write that names neither a tier nor a store, or only one of them, records nothing, and a
  read for that location afterwards finds nothing. This holds in a project with several
  stores **and** in a project with exactly one, which is the case most likely to tempt a
  convenience fallback.
- A refusal names the stores available in the tier concerned, and those names are the same
  ones the store-enumeration command reports.
- A write addressed to a repository's own store is readable back at that address and is not
  readable under any other name or in the other tier. Two repositories holding entries at
  the same location stay distinct in both directions.
- A repository whose configuration declares more than one store, or names the one it
  declares, is rejected, and no knowledge operation against that project succeeds until it
  is corrected.
- Every listing entry, every search hit, and every read result reports a tier and a store
  name. Taking any single hit and issuing a read using only what that hit carries returns
  the entry the hit excerpted, including when an identically-named entry exists elsewhere.
- Choosing a tier returns results only from that tier; narrowing to a name returns results
  only from that store; omitting the narrowing covers every store the tier reaches. The
  always-applied load obeys the same rule, with no store implicitly included.
- A stale configuration fails on every knowledge operation with an error naming the file,
  what was found, and what is required; correcting the file resolves it; and a repository
  scaffolded by the tool is accepted with no correction at all.
- A caller reading only the published machine-readable interface can issue a fully addressed
  write successfully, and that interface advertises the tier, name, and narrowing fields.

### Regression the change must invert

One existing command-surface test asserts today's defective behaviour directly: that a read
against an ambiguous scope resolves to the first registered repository's copy, and that a
write lands in that repository's store rather than the intended one. It is rewritten to
assert the opposite, that the same under-specified request is now refused and records
nothing, rather than deleted, so the suite keeps a test standing at the exact point the
defect lived.

### Success metrics, and how each is verified

- **No knowledge entry lands in the wrong store.** *Behavioural test.* The guarantee is
  two-sided and both sides are asserted: a fully addressed write is readable at that address
  and at no other, and an under-specified write fails without recording anything anywhere.
  The second half is what makes the count of silently misplaced entries zero rather than
  merely low, so it is asserted in the single-store case as well as the multi-store case.
- **The dogfood write succeeds first time.** *Manual, captured in the implementation test
  plan.* Whether recording the two supplied entries needed a second attempt, or a fallback
  to editing files directly, is an observation about the act of delivery rather than a
  property of the code. The durable half of it is covered behaviourally by the assertion
  above; that the two entries are readable from the documentation repository's store, absent
  from every other, and reported with that store as their source is asserted as an
  acceptance check.
- **Planning loads less irrelevant knowledge.** *Behavioural test.* In a fixture with more
  than one store holding always-applied entries, the load narrowed to a single store returns
  strictly fewer entries than the same load with no narrowing, and returns exactly the
  narrowed store's entries. That is the reduction the metric describes, measured the way it
  describes it.
- **A caller can predict what a request covers.** *Behavioural test.* A matrix over the
  three tiers crossed with no narrowing, a narrowing naming one store, and a narrowing
  naming several, asserting the exact set of stores covered in each cell against a
  hand-written expectation. The point of the matrix is that no cell is a special case; a
  narrowing that does not name a store excludes it, in every tier, for every retrieval path.

### Deliberate gaps

- **The documentation site gets no new automated tests.** Its correctness gates are its
  existing build and typecheck, plus the manual markup guard its authoring conventions
  define. The spec constrains this work to introduce no new site tooling, and asserting page
  prose in a test suite would be a new mechanism for no additional confidence.
- **No test derives an expected value from the production types.** Result envelopes are
  decoded into independently-declared mirror structures and compared against hand-written
  expectations, which is the existing convention at the command surface and is what keeps a
  renamed field from silently passing.
- **The store layer gains no new tests of its own.** Its two new fields are left empty by the
  store and populated by the knowledge layer, so the knowledge layer's assertions already
  cover the only behaviour that exists; a store-level test would assert that an empty field
  is empty.
- **No test asserts the same guarantee twice through different mechanisms.** Where a
  command-surface test already pins a behaviour end to end, the corresponding unit test
  covers the resolution rule and its error text rather than repeating the outcome.


## Milestones & Phases

### Milestone 1: Knowledge says where it lives, and a vague write is refused

**What changes.** Every knowledge request states which knowledge it means: the project's
shared knowledge, the repositories' own, or both, and which store in particular. Writing an
entry now requires naming both, and a write that leaves either unstated is refused outright
instead of quietly landing somewhere plausible; the refusal lists the stores available so the
write can be reissued immediately. Every result from listing, searching, and reading says
which tier and which store it came from, so a search hit can be read back exactly without a
second lookup and without any risk of retrieving a same-named entry from elsewhere. Searching,
listing, and the always-applied load can all be narrowed to chosen stores, and the narrowing
is honoured with no exceptions: a store that is not named is not included, and a store is
never included merely because it was hard to attribute. In a project with two repositories,
knowledge written for one stops becoming unreachable behind the other. Configuration is
untouched at this point, so existing projects keep working: a repository's store is addressed
by its registered name and a shared store by the name it is already declared under.

**Validation point.** In a project registering two repositories, a write naming a tier and a
store is readable back at that address and nowhere else; the same write with the tier or the
name omitted fails and records nothing, both there and in a single-repository project;
listing and searching report a tier and a store for every result; a hit read back using only
what it carries returns the same entry; and the always-applied load narrowed to one store
returns that store's entries and no others. The full Go test suite passes.

#### - [x] Phase 1.1: Addressing vocabulary and single-store resolution

**Repo:** `spektacular`

Introduce the words the whole feature is built from: a tier naming the project's shared
knowledge, the repositories' own, or both; a name identifying one store within a tier; and a
flat list of those names for narrowing. One piece of groundwork comes first: the word "tier"
already means something else inside the knowledge feature, namely whether a category of entries
is loaded on every task or only looked up when searched, so that older meaning is renamed in the
code to keep the two apart while leaving everything it publishes exactly as it is. Reading and
writing are then reshaped to travel on a single-store address, and the first-match lookup that
silently picked a winner is deleted outright rather than made stricter. A request that does not identify exactly one store is
refused, and the refusal reports the names available in the tier concerned so the caller can
reissue without going and reading configuration.

*Technical detail:* [context.md#phase-1.1](./context.md#phase-11-addressing-vocabulary-and-single-store-resolution)

**Acceptance criteria**:

- [x] A write that omits the tier, or omits the store name, or omits both, fails and records
      nothing, and a subsequent read for that location finds nothing.
- [x] The same omission is refused in a project containing exactly one store, not only where
      several exist.
- [x] A refusal names the stores available in the tier concerned, and those names match the
      ones the store-enumeration command reports.
- [x] A write addressed to a named store is readable back at that address, and reading the
      same location under any other name, or in the other tier, finds nothing.
- [x] Two repositories holding entries at the same location stay distinct: each name returns
      its own repository's content and never the other's.
- [x] A tier value that is neither of the two tiers nor "both" is rejected, and "both" is
      rejected for reading and writing because they address one store.
- [x] The command that lists the knowledge categories and their retrieval tiers publishes exactly
      what it publishes today, with no renamed or reordered fields.

#### - [x] Phase 1.2: Narrowing honoured uniformly across every retrieval path

**Repo:** `spektacular`

Give searching, listing, the conventions view, and the always-applied load one shared way to
say which stores a request covers, and apply it with no exceptions. Today the always-applied
load exempts any store it cannot attribute to a repository, so a shared store loads whatever
was asked for; that exemption is removed. Omitting the narrowing covers every store the chosen
tier reaches, and naming a store that the chosen tier does not reach is an error rather than a
plausible-looking empty result. This is the change that lets planning scoped to one repository
stop loading the standing rules of repositories it is not touching.

*Technical detail:* [context.md#phase-1.2](./context.md#phase-12-narrowing-honoured-uniformly-across-every-retrieval-path)

**Acceptance criteria**:

- [x] Restricting a search to the repository tier returns hits only from repositories' own
      stores; restricting it to the project tier returns hits only from shared stores; asking
      for both returns hits from all of them.
- [x] A search narrowed to one store name returns hits only from that store, and the same
      search with no narrowing returns hits from every store its tier reaches.
- [x] Listing and the always-applied load behave identically under the same narrowing.
- [x] Loading always-applied knowledge narrowed to one repository returns that repository's
      entries and no others, including none from any shared store.
- [x] Loading it across both tiers, narrowed to one repository plus one shared store, returns
      exactly those two stores' entries.
- [x] Narrowing to a name the chosen tier does not contain fails with a message naming the
      valid names, rather than returning nothing.

#### - [x] Phase 1.3: Every result reports the tier and store it came from

**Repo:** `spektacular`

Make every listing entry, search hit, read result, convention, and always-applied entry carry
the tier and store name it originated in, so no result is returned without saying where it
came from. The store name a repository's knowledge is addressed by is established during
aggregation, where a repository's registry name becomes its store name and a shared store's
declared name becomes its own, and where names are checked unique within their tier. The
practical outcome is that a single search hit carries everything a read needs, so a result can
be retrieved exactly without any further lookup and with no risk of fetching a same-named
entry from a different store.

*Technical detail:* [context.md#phase-1.3](./context.md#phase-13-every-result-reports-the-tier-and-store-it-came-from)

**Acceptance criteria**:

- [x] Listing knowledge in a project with two registered repositories and a shared store
      returns entries from all three, and every entry reports a tier and a store name.
- [x] A search matching entries in more than one store returns hits from each, and every hit
      reports its tier and store name alongside the information it reports today.
- [x] Taking any single hit and issuing a read using only the identifying information that hit
      carries returns the same entry the hit excerpted, including when an entry of the same
      name exists in another store.
- [x] The store-enumeration command reports a tier and a name for every configured store, and
      no longer reports a separate repository attribution alongside the name.
- [x] Two stores in the same tier declared under the same name are rejected when the store list
      is built; the same name used once in each tier is accepted.

#### - [x] Phase 1.4: Command surface and published interface

**Repo:** `spektacular`

Expose the addressing on the command line and in the machine-readable interface each knowledge
command publishes. Reading and writing take the tier and store name alongside the path in their
existing JSON input; searching, listing, conventions, and the always-applied load take a tier
option defaulting to both tiers and a repeatable narrowing option, which replaces the
repository-only option that exists today. Because narrowing is expressed as flags rather than
JSON input, the published interface gains a way to describe flags, so a caller can discover the
addressing without reading documentation or source.

*Technical detail:* [context.md#phase-1.4](./context.md#phase-14-command-surface-and-published-interface)

**Acceptance criteria**:

- [x] The published interface for each knowledge command advertises the tier, store name, and
      narrowing fields it accepts or returns.
- [x] A caller relying only on that published interface can issue a fully addressed write
      successfully.
- [x] Every command family other than knowledge publishes exactly the interface it publishes
      today, with no added or reordered fields.
- [x] A refusal reaches the caller in the CLI's standard error envelope, carrying a code, a
      message naming the problem, and a next action giving a runnable correction.
- [x] The existing behaviour where an ambiguous request resolved to the first registered
      repository is gone, and the test that asserted it now asserts the refusal instead.

### Milestone 2: A repository declares one knowledge store, and stale configuration fails loudly

**What changes.** A repository's configuration now declares its single knowledge store as one
provider block, the same shape its changelog declaration already uses, so it can no longer
declare several stores or put a label on the one it has. Knowledge that belongs to no single
repository is declared by the project instead, as many stores as wanted, each under its own
name. Configuration written in the old form is rejected rather than quietly upgraded: any
knowledge operation against it fails with an error naming the file, what was found in it, and
what is required instead, and the file is left exactly as it was so the same failure repeats
until someone edits it. This is a breaking change for existing projects, this one included,
and correcting the two configuration files in this project is part of the milestone. A
repository scaffolded by the tool needs no correction at all.

**Validation point.** A repository configuration declaring more than one store, or naming the
one it declares, is rejected and no knowledge operation against that project succeeds until it
is corrected; editing it into the required form makes the same operation succeed; a freshly
scaffolded repository is accepted as-is and can be written to and read from immediately; and
this project's own two configuration files have been corrected so its knowledge commands work
again. The full Go test suite passes.

#### - [x] Phase 2.1: A repository declares exactly one knowledge store

**Repo:** `spektacular`

Collapse a repository's knowledge declaration from a list of labelled sources to a single
provider block, matching the shape its changelog declaration already uses in the same file. A
repository can then no longer declare several stores, nor put a name or label on the one it
has, because the file has nowhere to say either. The scaffolding that creates a repository's
knowledge directories stops hunting through a list for a store whose label matches a constant
and simply reads the one block, which removes a way for the scaffolder and the configuration
shape to drift apart.

*Technical detail:* [context.md#phase-2.1](./context.md#phase-21-a-repository-declares-exactly-one-knowledge-store)

**Acceptance criteria**:

- [x] A repository's configuration declares its knowledge store as a single provider block with
      no name, label, or list.
- [x] A repository that declares no knowledge store at all still resolves to its default store
      in the expected place.
- [x] Scaffolding a repository into a project produces a configuration that is accepted as-is,
      and writing to and reading from that repository's own store succeeds with no further
      edit.
- [x] The knowledge category directories and their explanatory files are still created for a
      newly scaffolded repository, exactly as before.

#### - [x] Phase 2.2: Shared stores are declared by the project, each under its own name

**Repo:** `spektacular`

Keep the project's own list of shared knowledge stores and identify each one by a name rather
than by a scope label, so the same word means the same thing wherever it appears: in a
declaration, in a write, and in a narrowing. Names are required to be unique within the project
tier, which is what makes a name unambiguous without also constraining what a repository may be
called.

*Technical detail:* [context.md#phase-2.2](./context.md#phase-22-shared-stores-are-declared-by-the-project-each-under-its-own-name)

**Acceptance criteria**:

- [x] A project may declare any number of shared knowledge stores, each identified by its own
      name.
- [x] A write naming the project tier and a declared shared store's name succeeds, and reading
      it back with the same tier and name returns the content supplied.
- [x] Two shared stores declared under the same name are rejected with an error naming the
      duplicate.
- [x] A project declaring no shared stores is valid, and requests restricted to the project tier
      simply return nothing rather than failing.

#### - [x] Phase 2.3: Configuration in the superseded form is rejected with an actionable error

**Repo:** `spektacular`

Fail loudly on any configuration still written the old way, rather than upgrading it silently or
accepting both forms indefinitely. Because the configuration types no longer have a field the
removed keys could land in, a stale file would otherwise parse cleanly and behave as though it
declared nothing, so each file gains a load-time guard that reads the raw document and refuses
when it finds the old shape. The error names the file, what was found in it, and what is
required instead, and nothing on disk is touched, so the same failure repeats until a person
edits the file.

*Technical detail:* [context.md#phase-2.3](./context.md#phase-23-configuration-in-the-superseded-form-is-rejected-with-an-actionable-error)

**Acceptance criteria**:

- [x] A project whose configuration, or whose registered repository's configuration, declares
      knowledge in the superseded form fails on any knowledge operation.
- [x] The error names the configuration file, what was found, and what is now required.
- [x] Running the operation again without editing the file produces the same failure, and the
      file's contents are unchanged afterwards.
- [x] A repository whose configuration declares more than one knowledge store, or names or
      labels the store it declares, is rejected, and no knowledge operation against that project
      succeeds until it is corrected.
- [x] After editing the configuration into the required form, the same knowledge operation
      succeeds against that store.
- [x] Upgrading a project from the older single-file configuration still produces a repository
      configuration that is accepted without further correction.

#### - [x] Phase 2.4: This project's own configuration is corrected

**Repo:** `spektacular`

Bring this project's two repository configuration files into the required form. Both currently
declare knowledge the old way, so from the moment the rejection lands every knowledge command in
this repository fails until they are edited. Correcting them is what restores the tool's ability
to read its own knowledge base, and it is a prerequisite for the entries recorded in the final
milestone.

*Technical detail:* [context.md#phase-2.4](./context.md#phase-24-this-projects-own-configuration-is-corrected)

**Acceptance criteria**:

- [x] Both registered repositories' configuration files declare their knowledge store in the
      required single-block form.
- [x] Every knowledge command runs successfully in this project again.
- [x] Enumerating the configured stores reports one store per registered repository, each under
      its repository's registered name, in the repository tier.
- [x] Reading an entry that exists only in the documentation repository's store succeeds when
      addressed to that repository, which it did not before this work.

### Milestone 3: Agents and documentation describe the two tiers

**What changes.** The knowledge skill, which is how an agent reaches this behaviour, now works
in tiers and store names throughout, and before recording an entry on a user's behalf it shows
the tier and store it will be written to alongside the location, so approval is given for a
destination rather than for a filename. The planning workflow's knowledge load moves onto the
new narrowing, so planning work scoped to one repository stops pulling in the standing rules of
repositories it is not touching. The published machine-readable interface each command exposes
now advertises the tier, store name, and narrowing it accepts, so a caller can discover the
addressing without reading documentation or source. The product documentation explains the two
tiers and what belongs in each, how a store is named, how a write states its destination, and
how retrieval is narrowed, and every existing example that showed the old configuration shape
is corrected so nothing in the documentation instructs a reader to write a file the tool now
rejects.

**Validation point.** A caller relying only on the published interface can issue a fully
addressed write successfully; the skill's proposal shows a tier and a store name before
writing; the documentation site builds and typechecks without errors and its authored prose
follows that site's conventions; and no page, readme, or in-repo document still shows the
superseded configuration shape. The full Go test suite passes.

#### - [x] Phase 3.1: The knowledge skill works in tiers and shows where an entry will land

**Repo:** `spektacular`

Rewrite the knowledge skill, which is how an agent reaches this behaviour, so its lookup,
contribute, and update flows all state a tier and a store name where they previously stated a
scope. Its propose-then-confirm checkpoint becomes the place the spec's approval requirement is
satisfied: before recording an entry on a user's behalf the skill shows the tier and store it
will be written to alongside the location, and waits for approval of that destination. The
layered precedence chain the skill currently describes is removed rather than restated, because
nothing in the tool has ever implemented it.

*Technical detail:* [context.md#phase-3.1](./context.md#phase-31-the-knowledge-skill-works-in-tiers-and-shows-where-an-entry-will-land)

**Acceptance criteria**:

- [x] Before an entry is recorded on the user's behalf, the destination presented for approval
      states a tier and a store name as well as a location.
- [x] The entry is subsequently readable at exactly that destination.
- [x] The skill enumerates the configured stores before proposing, and chooses only from the
      names that enumeration returns.
- [x] The skill no longer describes a precedence order between stores.
- [x] The per-agent copies of the skill match the source it is generated from, so no agent
      receives instructions in the old vocabulary.

#### - [x] Phase 3.2: Workflow steps and agent guidance use the new vocabulary

**Repo:** `spektacular`

Update the workflow steps that invoke knowledge commands directly and the two managed
agent-guidance sections that delegate to the skill. The planning workflow's always-applied load
moves onto the new tier and narrowing options, which is what delivers the reduction in
irrelevant knowledge loaded during planning, and its instruction stops promising that shared
stores load regardless of the narrowing, which is no longer true. No new managed guidance
section is introduced for the knowledge model, because the detail belongs in the skill.

*Technical detail:* [context.md#phase-3.2](./context.md#phase-32-workflow-steps-and-agent-guidance-use-the-new-vocabulary)

**Acceptance criteria**:

- [x] Planning work scoped to one repository loads only that repository's always-applied
      knowledge, measurably fewer entries than the same load with no narrowing.
- [x] No workflow step or agent-guidance section instructs an agent to address knowledge by a
      scope label.
- [x] No workflow step still claims that some stores load regardless of the narrowing given.
- [x] No new managed agent-guidance section describing the knowledge model has been added.

#### - [x] Phase 3.3: The documentation site explains the two tiers

**Repo:** `docs`

Rewrite the knowledge base reference page's configuration and lifecycle material so it explains
the addressing model: what the two tiers are and what belongs in each, how a store gets its
name, how a write states its destination, and how retrieval is narrowed. This extends the
section that already explains where knowledge sources are declared rather than adding a new
page, and it replaces the layered precedence description, which the tool never implemented.

*Technical detail:* [context.md#phase-3.3](./context.md#phase-33-the-documentation-site-explains-the-two-tiers)

**Content outline** for the reworked `Configuration` section, keeping its existing position and
shaded background, and its existing subtitle slot:

1. *Two tiers.* Opening paragraph naming them and drawing the line: a repository's own store
   holds knowledge about that repository's codebase and travels with it; the project's shared
   stores hold knowledge that belongs to no single repository. Illustrative prose: "Knowledge
   lives in one of two tiers. Every registered repository contributes exactly one store, holding
   what is true of that repository's own code. The project declares any number of shared stores
   for knowledge that spans repositories, a team handbook say, or a personal one you carry
   between projects."
2. *A repository declares one store.* Prose plus a fenced `yaml` block, exactly:
   ```yaml
   # repo.yaml
   knowledge:
     provider: file
     config:
       location: knowledge
   ```
   Note that the store's name is the name the project registered the repository under, and that
   the repository does not choose it.
3. *The project declares its shared stores.* Prose plus a fenced `yaml` block, exactly:
   ```yaml
   # .spektacular/config.yaml
   knowledge:
     sources:
       - name: team
         provider: file
         config:
           location: ~/work/team-knowledge
   ```
   Note that names must be unique within the project tier, and that relative locations resolve
   against the project root.
4. *Addressing a store.* A fenced `bash` block showing a fully addressed write, exactly:
   ```bash
   spektacular knowledge write \
     --data '{"tier": "repo", "name": "docs", "path": "conventions/site-layout.md"}' \
     --file ./site-layout.md
   ```
   followed by prose stating that both the tier and the name are required, that a write missing
   either is refused, and that the refusal lists the names available in that tier.
5. *Narrowing what a request covers.* A fenced `bash` block showing two forms, exactly:
   ```bash
   spektacular knowledge search "database timeout" --tier repo
   spektacular knowledge search "database timeout" --tier repo --filter docs
   ```
   followed by prose: omitting the narrowing covers every store the tier reaches, naming stores
   covers exactly those, and no store is ever included or excluded implicitly.

The `Creating` paragraph of the `The lifecycle of an entry` section is updated to the same
fully addressed write shown above, and the `Sources are layered, most-specific first` bullet in
`Why it works this way` is replaced by a bullet explaining that the two tiers answer different
questions rather than overriding one another.

**Acceptance criteria**:

- [x] The page explains the two tiers and what belongs in each, how a store is named, how a
      write states its destination, and how retrieval is narrowed.
- [x] No example on the page shows the superseded configuration shape or an unaddressed write.
- [x] The page no longer claims a precedence order between stores.
- [x] Where the page discusses when a category's entries are loaded, it says "retrieval tier" in
      full, so a reader cannot confuse it with the tier a store belongs to, and the distinction
      between the two is stated once explicitly.
- [x] The site builds and typechecks without errors, and the page body contains no layout markup.
- [x] The authored prose follows the site's conventions, including its prohibition on em dashes.

#### - [x] Phase 3.4: The rest of the documentation site is corrected

**Repo:** `docs`

Correct every other page whose examples or field listings become wrong once the configuration and
result shapes change: the configuration reference and its per-key descriptions, the multi-repository
guide, the getting-started tutorial's multi-source section, the landing page's one-line description
of how results are tagged, and the extending guide's listing of a search result's fields. These are
factual corrections rather than new explanation, and they include resolving an existing
contradiction where the tutorial shows a repository-tier store declared in the project's own file.

*Technical detail:* [context.md#phase-3.4](./context.md#phase-34-the-rest-of-the-documentation-site-is-corrected)

**Content example** for the configuration reference's knowledge key descriptions, replacing the
current per-field bullets. The project-side key, with its `defaultValue` becoming "none":

> Ordered list of the shared knowledge stores the project itself owns, such as a team handbook.
> Each repository declares its own store in its `repo.yaml`, not here.
>
> - `knowledge.sources[].name`: the name this store is addressed by. Must be unique among the
>   project's own stores.
> - `knowledge.sources[].provider`: storage backend; only `file` ships today.
> - `knowledge.sources[].config.location`: where the file provider reads knowledge from.
>   Relative paths resolve against the project root.

and the repository-side key, with its `defaultValue` becoming "a file store at `knowledge`":

> This repository's single knowledge store, holding what is true of its own code. The store is
> addressed by the name the project registered this repository under, so there is no name to set
> here and no more than one store to declare.
>
> - `knowledge.provider`: storage backend; only `file` ships today.
> - `knowledge.config.location`: where the file provider reads this repository's knowledge from.
>   Relative paths resolve against the folder holding `repo.yaml`.

**Acceptance criteria**:

- [x] No page on the site shows a knowledge declaration in the superseded form.
- [x] The configuration reference describes the repository key as a single store with no name and
      the project key as a named list.
- [x] The tutorial no longer shows a repository's own store declared in the project's own
      configuration file.
- [x] The extending guide's description of a search result lists the tier and store fields.
- [x] The site builds and typechecks without errors, and no page body contains layout markup.
- [x] The authored prose follows the site's conventions, including its prohibition on em dashes.

#### - [x] Phase 3.5: The command repository's own documentation is corrected

**Repo:** `spektacular`

Update the two documents in the command repository that carry a second copy of the knowledge model:
the readme's knowledge section, its two configuration examples, and its command list; and the in-repo
knowledge base document, including its command reference table. Add the changelog entry recording
this as a breaking change, since existing projects must edit both their project and repository
configuration by hand.

*Technical detail:* [context.md#phase-3.5](./context.md#phase-35-the-command-repositorys-own-documentation-is-corrected)

**Content example** for the readme section currently titled `Scopes, search, and de-duplication`,
retitled and rewritten to open:

> ### Tiers, search, and de-duplication
>
> Knowledge lives in one of two tiers. Every registered repo contributes exactly one store,
> addressed by the name the project registered it under, holding knowledge about that repo's own
> code. The project declares any number of shared stores under `knowledge.sources`, each with its
> own `name`, for knowledge that belongs to no single repo. Every read, search, and always-applied
> load states a tier and, optionally, the store names to narrow to; every result reports the tier
> and store it came from.

with the `Hit` field listing beneath it replacing its `scope` line with `tier` and `name` lines, and
the changelog entry opening:

> **Breaking change**: knowledge is now addressed by tier and store name. A repo declares its single
> knowledge store as one provider block in `repo.yaml`, and the project names each of its shared
> stores under `knowledge.sources`. Configuration in the previous form is rejected with an error
> naming the file and the required shape; it is not migrated automatically.

**Acceptance criteria**:

- [x] The readme explains the two tiers, how a store is named, how a write states its destination,
      and how retrieval is narrowed.
- [x] Neither the readme nor the in-repo knowledge base document shows the superseded configuration
      shape or an unaddressed command example.
- [x] Neither document still claims a precedence order between stores.
- [x] The changelog records the change as breaking and says what an existing project must edit.

### Milestone 4: The site's layout knowledge is recorded where it belongs

**What changes.** Two knowledge entries about the documentation site, one recording its layout
conventions and one recording the reasoning behind its choice of text width, are recorded into
the documentation repository's own knowledge store. They are written through the delivered
behaviour rather than by editing files by hand, which is what makes them a genuine test of it:
this is exactly the write that could not be expressed before, because there was no way to say
which repository's store was meant. Once recorded, they travel with the documentation
repository into any project that registers it, and searching for them reports that store as
their source.

**Validation point.** Both entries read back from the documentation repository's own store with
the content supplied; reading either from any other store finds nothing; searching for them
reports that store as their source; and neither needed a second attempt or a fallback to
editing files directly.

#### - [x] Phase 4.1: The site's layout knowledge is recorded through the delivered behaviour

**Repo:** `docs`

Record the two supplied knowledge entries, one describing the documentation site's layout
conventions and one recording the reasoning behind its choice of text width, into the documentation
repository's own knowledge store. They are written through the knowledge skill and the delivered
commands rather than by editing files directly, which is the point: this is precisely the write that
could not be expressed before, because there was no way to say which repository's store was meant.

*Technical detail:* [context.md#phase-4.1](./context.md#phase-41-the-sites-layout-knowledge-is-recorded-through-the-delivered-behaviour)

**Acceptance criteria**:

- [x] Both entries are recorded using the delivered behaviour rather than by editing files directly.
- [x] Reading each back addressed to the documentation repository's own store returns its content.
- [x] Reading either from any other store finds nothing.
- [x] Searching for them reports that store as their source.
- [x] Neither write needed a second attempt to correct where it landed.

## Open Questions

Three uncertainties genuinely cannot be settled before implementation begins. Everything else
raised during planning was resolved and recorded as an assumption.

**The bodies of the two supplied knowledge entries.** Milestone 4 records two entries into the
documentation repository's store: one covering the site's layout conventions, one recording the
reasoning behind its choice of text width. Both were drafted in the session that surfaced this
defect, but only their substance was carried forward, not their text, and neither is staged on
disk anywhere in either repository. *Depends on:* whether the user still holds the original
drafts. *When the implementer reaches Phase 4.1:* STOP and ask the user for the drafted bodies.
If the user no longer has them, offer to write both from the summary recorded in the plan's
research and present each for approval before recording it, since the propose-then-confirm
checkpoint applies either way. Do not record an entry whose body the user has not seen.

**Whether the documentation site currently builds and typechecks cleanly.** Two phases carry an
acceptance criterion that the site builds and typechecks without errors, and the site's own
conventions require zero errors and zero warnings before merge. Nothing was run against it during
planning, and its continuous integration runs only the build, never the typecheck, so a
pre-existing typecheck failure would not have been caught. *Depends on:* the state of the site's
existing pages, not on anything this change does. *When the implementer reaches Phase 3.3:* run
the build and the typecheck before editing anything, to establish a baseline. If either already
fails on untouched pages, STOP and tell the user what fails and that it predates this work,
rather than silently absorbing the failures into this change or treating the criterion as
unmeetable.

**What else regenerating the per-agent skill copies rewrites.** Phase 3.1 requires the two
generated copies of the knowledge skill to be brought back in step with their template, and the
sanctioned way to do that is to re-run initialisation for each agent rather than hand-editing
generated files. Whether that run also rewrites other tracked files in this working tree, other
skills, managed agent-guidance sections, or the recorded version, can only be seen by doing it in
a tree whose state is what it will be at that moment. *Depends on:* what else has drifted from
its template by the time the phase runs. *When the implementer reaches Phase 3.1:* inspect the
resulting diff before staging anything. If it touches files outside the knowledge skill's two
copies, report exactly which files changed and ask the user whether to include them, rather than
committing unrelated regeneration alongside this work or reverting changes that may be
legitimate.


## Out of Scope

### From the spec's non-goals

- **Moving, re-attributing, deleting, or migrating any existing knowledge entry.** Entries
  already written keep their current location and content. Nothing relocates an entry from one
  store to another, including entries misplaced by the defect this work fixes; nothing
  re-attributes an existing entry; and no way to delete one is added. Correcting a misplaced
  entry stays a manual matter. Worth noting concretely: this project's own knowledge base has
  not been audited for entries that landed in the wrong store, and this plan does not audit it.
- **Narrowing retrieval by category.** Filtering by the kind of knowledge, a gotcha or a
  decision, is a useful and independent axis, but search results are ranked and excerpted
  rather than loaded in full and already carry a category label, so the volume problem this
  work addresses does not need it. The flat narrowing list is deliberately shaped so a category
  axis can be added later as one more optional field without a breaking change.
- **Tier-aware addressing for specs, plans, or changelog records.** Those artifacts have their
  own stores and their own relationship to a project's repositories, and none of them exhibited
  the collision this work fixes. The changelog already routes to a named repository through its
  own option, which stays as it is.
- **Deduplicating identical knowledge held in more than one store.** The knowledge lookup flow
  already collapses byte-identical entries when consolidating an answer, using the checksum each
  result carries. Making the stores themselves aware of duplication is a separate question.

### Deliberately left out by the chosen design

- **Any migration, alias, or grace period for configuration in the superseded form.** The spec
  forbids it, so a stale file is rejected and corrected by hand. There is no tool command that
  rewrites it, and none is planned here.
- **Reinterpreting the layered precedence between stores.** The documentation and the knowledge
  skill currently describe a most-specific-wins ordering across scopes. Nothing in the tool has
  ever implemented it, so it is removed from the prose rather than redefined in tier terms. If a
  precedence rule is wanted later it needs its own spec, because it would be new behaviour
  rather than a restatement.
- **A managed agent-guidance section describing the knowledge model.** The spec rules it out.
  The existing managed guidance already routes an agent to the knowledge skill, and the detail
  lives in that skill.
- **New tooling, integrations, or automated tests on the documentation site.** The site is
  verified by its existing build, its existing typecheck, and the manual markup guard its
  conventions define. The spec constrains this work to introduce no new site tooling or
  publishing mechanism.
- **Restyling the command repository's own prose.** The documentation repository prohibits em
  dashes and that rule is honoured for everything authored into that repository. The command
  repository's readme and changelog use them throughout and are not being restyled; only their
  factual content about knowledge addressing changes.
- **Auditing every store's contents after the change.** The work guarantees where new writes
  land and what each result reports. It does not inspect what is already in either repository's
  store or judge whether it belongs there.

### Not raised, and deliberately not invented

- Nothing was deferred at the user's request during planning, because no question reached them:
  every design decision had a defensible default and is recorded with its rationale and the
  alternatives rejected. The three items that genuinely cannot be settled until implementation
  begins are in Open Questions rather than here.


## Changelog

### 2026-09-03 — Phase 1.1: Addressing vocabulary and single-store resolution

**What was done**: Introduced the addressing vocabulary the rest of the feature is built from — a
tier (`project`, `repo`, `all`), a store name within a tier, and the `Address` and `Selector`
value types that carry them — and rebuilt single-store resolution on top of it. The first-match
`byScope` lookup that silently picked a winner was deleted outright and replaced by `resolve`,
which returns exactly one store or a refusal naming the stores available in the tier concerned.
`Set.Read` and `Set.Write` now travel on an `Address`, and `knowledge read` / `knowledge write`
take `{"tier","name","path"}` in `--data`. The pre-existing `Tier` type, which means a category's
retrieval tier, was renamed `CategoryTier` to free the unqualified name; its `Category.Tier` field
and `tier` JSON key are untouched, so `knowledge categories` publishes exactly what it did before.

**Deviations**: Two changes the plan files under later phases had to move forward into this one,
because Go will not compile a half-changed package and two of this phase's own acceptance criteria
are untestable without them:

- `SourceInfo` drops `Scope`/`Repo` for `Tier`/`Name` now, rather than in Phase 1.3. The criterion
  "a refusal names the stores available in the tier concerned, and those names match the ones the
  store-enumeration command reports" cannot be asserted while `knowledge sources` still reports a
  scope label.
- `cmd/knowledge.go`'s read and write moved onto the three-part `--data` address now, rather than
  in Phase 1.4, because `Set.Read`/`Set.Write` no longer accept a scope string and there is no
  honest interim mapping from one. Still left for 1.4: the `--tier` / `--filter` flags, the `Flags`
  block on `commandSchema`, the remaining six schemas, and inverting the defect-asserting test.

`NewSet` derives each store's tier and name from the existing `Repo`/`Scope` fields for now
(`addressOf`); Phase 1.3 replaces that with fields the aggregator stamps, as planned.
`TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo` is `t.Skip`ped with its body
intact rather than rewritten, because a bare scope is no longer expressible; Phase 1.4 owns
inverting it in place, as the plan states.

**Files changed**:
- `spektacular: internal/knowledge/address.go` (new)
- `spektacular: internal/knowledge/address_test.go` (new)
- `spektacular: internal/knowledge/category.go`
- `spektacular: internal/knowledge/category_test.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`
- `spektacular: cmd/knowledge_ignore_test.go`

**Discoveries**:

- **An address must never be validated above the knowledge set.** The first cut validated the
  `--data` address in `cmd` before building the `Set`, which passed every test but produced a
  refusal with no store names in it, because only the set knows which stores are configured.
  Verification caught it; the fix was to make the set the sole validator, with the command surface
  checking only the entry path. Later phases must not reintroduce a pre-set address check.
- `runKnowledgeWrite` now builds the set before reading the entry body, so a refusal costs nothing
  and nothing is consumed from stdin on the failure path.
- Interim inconsistency this phase leaves behind, deliberately pinned by tests so Phase 1.3's
  change to it reads as intended rather than as drift: `store.Hit.Scope` carries the packed store
  label `"<tier>:<name>"` (what `NewSet` passes to `store.NewSourceStore`), while `Entry.Scope`,
  `Convention.Scope` and `AlwaysAppliedEntry.Scope` carry the bare store name. Phase 1.3 removes
  `Hit.Scope` and splits all four into `Tier` + `Name`.
- The two `tier` meanings are the single most likely thing to trip over in later phases. In Go the
  addressing one is `Tier` and the retrieval one is `CategoryTier`; in prose the retrieval one is
  always written "retrieval tier".

### 2026-09-03 — Phase 1.2: Narrowing honoured uniformly across every retrieval path

**What was done**: Gave searching, listing, the conventions view and the always-applied load one
shared way to say which stores a request covers, and applied it with no exceptions. `Selector.covers`
is the single membership predicate — the store's tier must fall under the selector's tier, and a
non-empty filter must name the store — and the exemption that let a store with no repo attribution
load regardless of the narrowing is deleted. `Search` and `List` skip a non-covered store before
querying it, so an excluded store is never opened at all. A filter naming a store the chosen tier
does not reach is refused with the reachable names rather than returning a plausible-looking empty
result.

**Deviations**: None to the phase's own scope. The command surface keeps its existing `--repo` flag
for now, mapped through a new `knowledgeAlwaysAppliedSelector()` helper (repos named become
`{Tier: repo, Filter: repos}`, none named becomes `{Tier: all}`), so the behaviour change lands
through today's flag surface. Phase 1.4 replaces `--repo` outright with `--tier` and repeatable
`--filter` across all four fan-out commands and deletes the helper, as planned.

**Files changed**:
- `spektacular: internal/knowledge/address.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- `validateSelector` lives on the `Set` rather than on `Selector`, for the same reason Phase 1.1's
  address validation does: refusing an unreachable filter name usefully means naming the reachable
  ones, and only the set knows them. `Selector.Validate` keeps just the tier check, which is all a
  bare value can decide.
- The behaviour change is visible in this project immediately: `knowledge always-applied --repo docs`
  now returns 7 entries, all from the `docs` store, against 11 unnarrowed. Before this phase the
  narrowed load also carried every project-tier store's entries.
- The coverage matrix test is the one to preserve if these paths are ever refactored. It asserts nine
  cells (three tiers crossed with no narrowing, one name, several names) against hand-written
  expectations, and every cell asserts all four retrieval paths against the same expectation, which
  is what makes "no path is a special case" a property the suite actually holds rather than a claim.
  It was mutation-checked during authoring: re-adding the project-tier exemption fails eight tests.

### 2026-09-03 — Phase 1.3: Every result reports the tier and store it came from

**What was done**: Every listing entry, search hit, read result, convention and always-applied entry
now carries the tier and store name it originated in, so no result is returned without saying where
it came from and any single result carries exactly what a read needs. A store's identity is now
established once, during aggregation, where a repo's contribution becomes the repo tier under that
repo's registry name and the project's contributions become the project tier under their declared
names, and where names are checked unique within a tier. The generic store layer stays generic: it
leaves the two attribution fields empty and the knowledge layer stamps them after merging, the same
seam an entry's category already used, so the two changelog stores built through the same
constructor are untouched.

**Deviations**: None to the phase's scope. Two adjustments worth recording. `addressOf` keeps a
fallback to the project tier under a source's declared scope for a source that reaches `NewSet`
unstamped, so `internal/knowledge`'s own unit tests do not have to route through the `cmd`
aggregator to build a usable set. And `config.SourceConfig.Tier` is a plain `string` rather than
`knowledge.Tier`, because `knowledge` imports `config` and the reverse would be an import cycle.

**Files changed**:
- `spektacular: internal/store/store.go`
- `spektacular: internal/store/search.go`
- `spektacular: internal/store/ignore.go`
- `spektacular: internal/store/store_test.go`
- `spektacular: internal/store/search_test.go`
- `spektacular: internal/config/config.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`
- `spektacular: cmd/knowledge_ignore_test.go`

**Discoveries**:

- **A published schema is not checked against what its command actually returns.** The search
  output schema still declared a `scope` property after the wire format had dropped it, and every
  test passed, because each schema test asserts the schema against its own literal rather than
  against a real invocation. Phase 1.4 touches all eight schemas and should close this by asserting
  each against a live response. This is the same shape as the Phase 1.1 finding: a check that never
  meets the thing it describes.
- Four schemas — `list`, `sources`, `conventions`, `always-applied` — declare only
  `{"type":"array"}` with no `items`, so they are silent about item shape rather than wrong about
  it. Pre-existing and byte-identical to before this work, but Phase 1.4's criterion is not met
  until they describe the tier and name their commands emit.
- `knowledge search --schema` exits 1 without a positional query, because the command declares
  `cobra.ExactArgs(1)`. The one command whose narrowing matters most is the one whose interface is
  hardest to discover. Worth fixing in 1.4 while that file is open.
- Criterion 5's end-to-end route is not the one the plan assumed: `config.KnowledgeConfig.Validate`
  rejects a duplicate `scope:` in the project config during `loadConfig`, before the aggregator
  runs, so two project sources sharing a name never reach `requireUniqueStoreNames`. The guard is
  proved wired in through a repo declaring two knowledge sources instead, both stamped with that
  repo's single registry name. **That path disappears in Phase 2.1**, when a repo may declare only
  one store, so the test needs re-pointing then.

### 2026-09-03 — Phase 1.4: Command surface and published interface

**What was done**: Exposed the addressing on the command line and in the machine-readable interface
each knowledge command publishes, closing Milestone 1. Reading and writing take the tier and store
name alongside the path in their JSON input; searching, listing, conventions and the always-applied
load take `--tier` (defaulting to both tiers) and a repeatable `--filter`, which replace the
repo-only `--repo` option outright. Because narrowing is expressed as flags, the schema envelope
every command family shares gained an optional `flags` block, omitted when empty, so every other
family publishes a byte-identical interface. The four output schemas that previously declared a
bare array now describe their item shapes, built from one shared address declaration so no schema
can describe the address differently from its neighbours.

Two pre-existing defects in scope of this phase's criteria were fixed with it. A correctly addressed
read of a missing entry surfaced as `internal_error` with an empty `next_action`, violating the
repo's own error convention; it now returns `knowledge_entry_not_found` naming the path and the
store, with a next action giving the exact `knowledge list --tier X --filter Y` that shows what the
store does hold. And `knowledge search --schema` required a positional query, so the one command
whose narrowing matters most had an interface a caller could not discover without already knowing
how to invoke it; the argument count moved into the run function, which returns
`knowledge_query_required` with a runnable example.

**Deviations**: None. The two phases that had moved work forward into 1.1 (the `SourceInfo` fields
and the read/write `--data` shape) left this phase exactly the flags, the schemas and the test
inversion the plan assigned to it, all of which landed here.

**Files changed**:
- `spektacular: cmd/spec.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`
- `spektacular: cmd/root_test.go`
- `spektacular: internal/knowledge/address.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`

**Discoveries**:

- **The defect test is inverted and standing, and the suite now has zero skips.**
  `TestKnowledgeReadWrite_DuplicateScopeResolvesToFirstRegistryRepo` became
  `TestKnowledgeReadWrite_AmbiguousRequestIsRefusedNotResolvedToFirstRepo` on the same fixture. It
  asserts the refusal *and* that nothing was recorded, then that both addressed forms reach their
  own repo's store, including the member repo's copy that first-match resolution could never reach.
- The answer to Phase 1.3's "a schema is never checked against what its command returns" is the
  criterion-2 test: it takes the write payload's field names from the decoded `write --schema`
  output and the tier value from that schema's enum, so a schema that drifts from its command fails
  it. Worth keeping that shape if more schemas are added.
- Milestone 1's validation point was checked by hand against a purpose-built two-repo scratch
  project, not only through the suite. Every clause holds, including the filesystem check that a
  refused write records nothing anywhere, in both the two-repo and the single-store case.
- Rough edge left deliberately: a caller still passing the removed `--repo` gets cobra's own
  unknown-flag error wrapped as `internal_error` with no next action. That is cobra's behaviour for
  every command in this CLI, and fixing it properly means a CLI-wide unknown-flag handler, which is
  a different change. Worth a follow-up spec if migration friction shows up.
- Pre-existing and unrelated: `cmd` holds order-dependent tests that fail under `-shuffle` or
  `-count=2` (`TestImplementNew_*`, `TestImplementGoto_RequiresActiveWorkflow`,
  `TestGotoStepRequired_*`, `TestWrapper_FailureIsPrintedExactlyOnce`, `TestUnknownSubcommand_*`).
  They fail in isolation with no knowledge test in the run. Every test this feature added passes
  under both flags.

### 2026-09-03 — Phase 2.1: A repository declares exactly one knowledge store

**What was done**: Collapsed a repo's knowledge declaration from a list of labelled sources to a
single provider block, deliberately shaped like the `changelog:` block sitting beside it in the same
file. A repo can no longer declare several stores, nor put a name or label on the one it has,
because the file has nowhere to say either. Both scaffolders — project initialisation and the shared
repo footprint writer — stopped hunting through a list for a source whose scope label matched a
constant and simply read the one block, which removes the class of drift where the scaffolder and
the configuration shape could disagree.

**Deviations**: None.

**Files changed**:
- `spektacular: internal/config/config.go`
- `spektacular: internal/config/repo.go`
- `spektacular: internal/config/repo_test.go`
- `spektacular: internal/project/init.go`
- `spektacular: internal/project/init_test.go`
- `spektacular: internal/repo/footprint.go`
- `spektacular: internal/repo/footprint_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`
- `spektacular: cmd/repo_test.go`

**Discoveries**:

- **An old-form `repo.yaml` is currently worse than ignored, and Phase 2.3 must catch it.**
  `RepoConfigFromYAMLFile` seeds from `NewDefaultRepoConfig()` before unmarshalling, so the now-
  unknown `sources` key is dropped and the seeded default survives. Because `Provider` is then
  already `file`, `WithDefaults` sees a populated block and fills nothing — so a repo that declared
  a custom knowledge location silently starts reading and writing the default `<repoRoot>/knowledge`
  instead, an empty store, with no error at all. `RepoSourceConfig.UnmarshalYAML` is the exact
  precedent for the fix: a custom unmarshaller that detects the superseded shape and returns an
  error printing the replacement block.
- Two tests were deleted rather than converted, because what they asserted is now unconstructible:
  `TestInit_CreatesProjectKnowledgeSourceOnly` (a repo has one unscoped store, so there is no second
  source to skip) and the "two stores from one repo claim that repo's name" subtest that Phase 1.3
  had used to prove the uniqueness guard was wired in. The guard is still covered directly against
  `requireUniqueStoreNames`, and the cross-tier acceptance case is untouched.
- `cmd/version_test.go` needed no change, contrary to the plan's expectation: neither migration test
  asserts `repo.yaml` contents. The literal generated-YAML assertion now lives in
  `internal/config/repo_test.go` against `NewDefaultRepoConfig()`, which is what the migrator builds
  on, so migration output stays pinned.
- Pre-existing and untouched by this phase, but worth knowing: `internal/project/init.go` resolves a
  *relative* declared knowledge location against the project path, while the runtime resolves a
  repo's relative location against the folder holding its `repo.yaml`. It is never hit for a default
  config, since `WithDefaults` yields an absolute path, but a hand-written relative `location:` would
  make `init` scaffold somewhere the reader does not look.

### 2026-09-03 — Phase 2.2: Shared stores are declared by the project, each under its own name

**What was done**: The project keeps its list of shared knowledge stores, and each is identified by
a name rather than a scope label, so the same word means the same thing wherever it appears: in a
declaration, in a write, and in a narrowing. Names are required unique within the project tier,
which makes a name unambiguous without constraining what a repo may be called — a shared store and
a repo may share a name, because a store's identity is its tier and its name together.
`KnowledgeConfig.WithDefaults` was deleted rather than adapted: it existed to synthesise a default
project-owned store, which is now a repo-tier concern, and keeping it would silently manufacture an
unnamed shared store. The two constants it was the last consumer of went with it.

**Deviations**: One addition beyond the phase's letter. `KnowledgeConfig.Validate`'s two refusals
were bare `fmt.Errorf` strings, so a duplicate name reached the caller as `internal_error` with an
empty next action — a violation of this repo's own error convention, on this phase's own criterion
path. Both are now `output.NewError("config_invalid", ...)` carrying a runnable correction.

**Files changed**:
- `spektacular: internal/config/config.go`
- `spektacular: internal/config/config_test.go`
- `spektacular: internal/config/repo_test.go`
- `spektacular: internal/knowledge/set.go`
- `spektacular: internal/knowledge/set_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`

**Discoveries**:

- **Two harbor e2e fixtures will break the moment Phase 2.3's rejector lands, and no phase names
  them**: `tests/harbor/implement-workflow/environment/repo.yaml` and `.../docs-repo.yaml` both
  still declare the superseded `knowledge.sources` list. They are checked in, are not exercised by
  `go test ./...`, and currently work only by accident — the superseded key parses into an empty
  block, so `WithDefaults` synthesises a default store that happens to land on the same path.
  **Phase 2.4 is extended to cover them.** The `tests/harbor/jobs/<date>__*/` copies are recorded
  run artifacts and must be left alone.
- Three tests were deleted rather than converted because their subject no longer exists: the two
  `TestKnowledgeConfig_WithDefaults*` tests, and `TestKnowledgeConfig_ValidateRejectsDuplicateScope`,
  which was replaced rather than renamed because a substring check on an error string is the wrong
  shape for a structured refusal.
- The round-trip hand check deliberately wrote to the **middle** of three declared shared stores, so
  a regression to declaration-order resolution would fail it rather than pass by luck.

### 2026-09-03 — Phase 2.3: Configuration in the superseded form is rejected with an actionable error

**What was done**: Both configuration files gained a load-time guard that re-reads the raw document
and refuses when it finds the removed shape, because the typed configuration no longer has a field
those keys could land in and a stale file would otherwise parse cleanly and behave as though it
declared nothing. Each refusal names the file, the key it found, and prints the block now required.
Nothing is rewritten on disk, so the same file fails identically on every run until a person edits
it. The pattern is the one already established for the removed repo `address` key, applied twice
more rather than invented anew.

**Deviations**: Phase 2.4's live corrections were made here rather than in the following phase. The
plan's own dependency section states the constraint: from the moment the rejection lands every
knowledge command in this working tree fails, so the correction has to accompany the change rather
than follow it. Corrected to the single-block form: this repo's `.spektacular/repo.yaml`, the `docs`
repo's `.spektacular/repo.yaml`, and the two live harbor fixtures. Phase 2.4 is now a verification
pass over work already done.

**Files changed**:
- `spektacular: internal/config/config.go`
- `spektacular: internal/config/config_test.go`
- `spektacular: internal/config/repo.go`
- `spektacular: internal/config/repo_test.go`
- `spektacular: cmd/knowledge.go`
- `spektacular: cmd/knowledge_test.go`
- `spektacular: cmd/version_test.go`
- `spektacular: .spektacular/repo.yaml`
- `spektacular: tests/harbor/implement-workflow/environment/repo.yaml`
- `spektacular: tests/harbor/implement-workflow/environment/docs-repo.yaml`
- `docs: .spektacular/repo.yaml`

**Discoveries**:

- **The refusal was being swallowed one layer up, and the fix is the mirror of Phase 1.1's.**
  `aggregateKnowledgeSources` wrapped every footprint failure in `repo_footprint` with the next
  action "run `repo add` to repair the repo's footprint" — advice that cannot fix a superseded
  knowledge block and sends the caller somewhere useless. It now passes a typed
  `*output.ErrorResponse` through untouched and wraps only genuinely unreadable footprints. Phase
  1.1's lesson was that remediation cannot be built above the layer holding the facts; this is the
  same failure from the other end, where remediation built at the right layer was discarded above
  it. Both are now pinned by tests.
- That pin is load-bearing and fragile in an instructive way: `repo.FootprintError` implements
  `Unwrap()`, so `errors.As` matches **both** branches in the aggregator. Only the ordering — the
  typed refusal checked first — keeps the code `config_invalid`. Reordering those two branches
  silently restores the old misleading error, and the CLI tests are what catch it.
- Criterion 6 is asserted as acceptance rather than existence: the migration test loads the
  generated `repo.yaml` back through `RepoConfigFromYAMLFile` and requires no error, so migration
  output must actually pass the new guard rather than merely be written.
- A repo-wide sweep confirms nothing checked in still trips either guard, and no template emits a
  `knowledge:` block, so nothing generated can regress into the old shape.

### 2026-09-03 — Phase 2.4: This project's own configuration is corrected

**What was done**: This project's two repository configuration files were brought into the required
single-block form, restoring the tool's ability to read its own knowledge base. The two live harbor
end-to-end fixtures were corrected alongside them.

**Deviations**: The edits were made during Phase 2.3 rather than here, because the plan's dependency
section makes the ordering a hard constraint — every knowledge command in this working tree fails
from the moment the rejection lands. This phase was therefore a verification pass over work already
done. Beyond the plan's two files, two harbor fixtures had to be corrected as well; they were not
named in any phase, are not exercised by `go test ./...`, and worked only by accident, since the
superseded key parsed into an empty block whose synthesised default happened to land on the same
path.

**Files changed**: none in this phase; see Phase 2.3's list.

**Discoveries**:

- **The spec's original defect is now confirmed fixed against the real project, not just in tests.**
  Reading `conventions/mdx-authoring.md` addressed to `docs` returns the entry; the same path
  addressed to `spektacular` returns `knowledge_entry_not_found`. The reproduction recorded in the
  plan's research had the first form returning "not found", because the `docs` store was unreachable
  behind `spektacular` in a first-match scan.
- `knowledge sources` reports exactly one store per registered repo, each under its registry name in
  the repo tier, and this project declares no shared stores, so its project tier is legitimately
  empty.

### 2026-09-03 — Phase 3.1: The knowledge skill works in tiers and shows where an entry will land

**What was done**: The knowledge skill, which is the agent's only sanctioned route to these
commands, now works in tiers and store names throughout. Its propose-then-confirm checkpoint is
where the spec's approval requirement is satisfied: before recording an entry it shows the tier, the
store name and the path, so approval is given for a destination rather than for a filename. It
enumerates the configured stores first and is told to choose only from the names that enumeration
returns, never an inferred one. The layered precedence chain was removed rather than restated in
tier terms, because nothing in the tool has ever implemented it, and replaced with an explicit
statement that a repo's store and the project's shared stores answer different questions with no
precedence between them. Both per-agent copies were regenerated so neither ships stale.

**Deviations**: Two, both from the plan's own open question about what else regenerating the copies
rewrites, and both put to the user:

- Regenerating both agents' copies also refreshed `.bob/skills/spek-implement/SKILL.md` and
  `.bob/skills/spek-plan/SKILL.md`, which were stale because the `.claude` copies were regenerated
  when the repo-sources work landed at `0c0cb4f` and the `.bob` ones were missed. The user chose to
  keep the refresh rather than revert it.
- That catch-up propagated a stray-period typo present in the source template. The user chose to fix
  it, so `templates/skills/workflows/spek-implement/SKILL.md` was corrected and everything
  regenerated.

**Files changed**:
- `spektacular: templates/skills/workflows/spek-knowledge/SKILL.md`
- `spektacular: templates/skills/workflows/spek-implement/SKILL.md`
- `spektacular: templates/knowledge_addressing_test.go` (new)
- `spektacular: .claude/skills/spek-knowledge/SKILL.md`
- `spektacular: .bob/skills/spek-knowledge/SKILL.md`
- `spektacular: .bob/skills/spek-implement/SKILL.md`
- `spektacular: .bob/skills/spek-plan/SKILL.md`

**Discoveries**:

- **Deleting a rule is not the same as deleting the idea behind it.** The explicit
  `project → team → global` chain was removed, but the lookup intro still said "duplicates removed
  and the most specific source winning" — a precedence claim contradicting the new no-precedence
  sentence ten lines below it. The corpus-wide test caught it. When removing a documented model,
  sweep for its summary restatements, not just its definition.
- **A generated copy could be hand-edited undetectably until now.** Nothing compared the checked-in
  `.claude/` and `.bob/` trees against their templates; the existing agent tests render into a temp
  dir and assert on that. The new `TestGeneratedSkillCopiesMatchTheirTemplates` closes it for all
  four skills across both agents.
- **Regenerating more than one agent's copies rewrites `agent:` in `config.yaml`** to whichever ran
  last. It has to be restored afterwards. Anyone doing this again needs to know.
- Phase 3.2 is now forced by a self-cleaning test: the scope-vocabulary sweep carries an exemption
  list naming exactly the six templates 3.2 must rewrite, and **fails if an exempted file stops
  offending**, so the list cannot be left behind once the work is done.

### 2026-09-03 — Phase 3.2: Workflow steps and agent guidance use the new vocabulary

**What was done**: The planning workflow's always-applied load moved onto the new narrowing,
`knowledge always-applied --tier repo --filter <name>`, which is what delivers the reduction in
irrelevant knowledge loaded during planning. Its promise that "project-owned sources always load
regardless of `--repo`" was deleted rather than reworded, because it is no longer true. The step
also gained guidance that a plan needing shared knowledge asks for `--tier all` and names the stores
it wants. The remaining step and agent-guidance references moved from scope selection to tier and
store selection. No new managed agent-guidance section was introduced, per the spec's constraint;
`AGENTS.md` changed by exactly two lines, both vocabulary.

**Deviations**: None.

**Files changed**:
- `spektacular: templates/steps/plan/02-discovery.md`
- `spektacular: templates/steps/plan/03-architecture.md`
- `spektacular: templates/steps/plan/18-walkthrough.md`
- `spektacular: templates/steps/implement/07-update_changelog.md`
- `spektacular: templates/skills/skill_spawn-planning-agents.md`
- `spektacular: templates/agents/memory-context.md`
- `spektacular: templates/agents/knowledge-trigger.md`
- `spektacular: templates/knowledge_addressing_test.go`
- `spektacular: AGENTS.md`, `.claude/`, `.bob/` (regenerated)

**Discoveries**:

- **The scope-vocabulary sweep's exemption list is gone, as designed.** Phase 3.1 left it naming
  exactly the six templates this phase owned, with a test that fails both when an unexempted file
  offends *and* when an exempted one stops offending. That second half is what made the list
  self-cleaning: it could not be quietly left behind once the work was done.
- **Nothing enumerated the managed agent-guidance sections until now.** The existing agent tests
  spot-check individual headings, so a seventh managed section could have appeared unnoticed. The
  new test pins the six by hand and asserts `AGENTS.md` carries exactly those and no others, which
  is what turns the spec's "no new managed section" constraint into something the suite holds.
- Two sweeps had to be made precise rather than blunt, and the reasoning is written into the test
  file: `--repo` is still legitimate on the changelog commands, so only `knowledge` invocations are
  checked for it; and the bare word "regardless" is not banned, only its use on a line that also
  mentions narrowing.

### 2026-09-03 — Phase 3.3: The documentation site explains the two tiers

**What was done**: The knowledge base reference page's Configuration section was rewritten to the
plan's content outline: the two tiers and what belongs in each, a repository declaring one store,
the project declaring its shared stores by name, addressing a store on a write, and narrowing what a
request covers. The lifecycle section's `Creating` example became the fully addressed write, its
searching paragraph now says every hit carries what a read needs plus how to narrow, and its update
line takes tier, store name and path. The `Sources are layered, most-specific first` bullet was
replaced by one explaining that the two tiers answer different questions with no precedence between
them.

**Deviations**: None. The section keeps its position, its `surface` prop and its subtitle slot, so
the page's alternating background shading is unchanged.

**Files changed**:
- `docs: src/pages/knowledge-base.mdx`

**Discoveries**:

- **The plan's open question about the site's build resolves clean.** Run before editing anything:
  `npm run build` exits 0 and `npx astro check` reports 0 errors and 0 warnings, with 3 pre-existing
  `ts(6387)` hints about `document.execCommand` in files this work does not touch. After the edit the
  numbers are identical, so nothing pre-existing was absorbed and nothing new was introduced.
- The retrieval-tier ambiguity needed less work than the plan expected: `The six categories` already
  said "**retrieval tier**" in the checked-in page, so only the new Configuration prose had to state
  the distinction between a category's retrieval tier and a store's tier, which it now does
  explicitly.
- **The `docs` repo's working tree was already dirty before this work began**, with about twenty
  modified files from an unrelated in-progress component refactor. Anyone committing this feature
  must stage its specific paths rather than everything.

### 2026-09-03 — Phase 3.4: The rest of the documentation site is corrected

**What was done**: Every other page whose examples or field listings became wrong once the
configuration and result shapes changed was corrected: the configuration reference and both of its
per-key descriptions, the multi-repository guide, the getting-started tutorial's multi-source
section, the landing page's one-line description of how results are tagged, and the extending
guide's listing of a search result's fields.

**Deviations**: None. `debugging.mdx` was confirmed rather than edited, as the plan anticipated.

**Files changed**:
- `docs: src/pages/configuration.mdx`
- `docs: src/pages/projects.mdx`
- `docs: src/pages/index.mdx`
- `docs: src/pages/extending.mdx`
- `docs: src/content/tutorials/getting-started.mdx`

**Discoveries**:

- **The tutorial's contradiction was real and is resolved.** Its multi-source example declared a
  `project`-scoped store inside the project's own `config.yaml`, which the configuration reference
  says belongs in `repo.yaml`. The example is now split in two, shared stores in `config.yaml` under
  names and the repository's own store in `repo.yaml`, and the section retitled so it describes what
  it actually shows.
- The extending guide needed more than a field rename: its `Store` contract said hits "carry the
  store's own scope so callers can attribute them", which is now the opposite of how the seam works.
  It states that a store leaves the attribution fields empty and the knowledge layer stamps them,
  which is what keeps the changelog stores on the same constructor untouched.
- Site conventions verified after the edits rather than assumed: zero em dashes across all six pages
  this milestone touched, zero layout markup in any page body, build exits 0, and typecheck reports
  0 errors and 0 warnings, identical to the baseline taken before any editing.

### 2026-09-04 — Phase 3.5: The command repository's own documentation is corrected

**What was done**: The two documents carrying a second copy of the knowledge model were corrected.
The readme's knowledge section was retitled and rewritten to explain the two tiers, how a store is
named, how a write states its destination, and how retrieval is narrowed, with the `Hit` listing and
both configuration examples updated and the command list describing the new options. The in-repo
knowledge base document lost its `Layered source precedence` section entirely, replaced by one that
ends by stating plainly that there is no precedence between stores, and every row of its command
reference table now shows the addressed `--data` shape or the narrowing options. The changelog
records the change as breaking and names the two one-time edits an existing project must make.

**Deviations**: None. The readme's own em dashes were left as they are, per the plan: that rule
binds prose authored into the documentation repository, not this one.

**Files changed**:
- `spektacular: README.md`
- `spektacular: docs/knowledge-base.md`
- `spektacular: CHANGELOG.md`
- `spektacular: cmd/docs_test.go`

**Discoveries**:

- **Deleting a section is not enough; its inbound references have to go too.** The rewrite removed
  `## Layered source precedence`, but a line further up still said a refinement is "resolved by
  layered precedence (below)" — a claim that survived the deletion and pointed at a section that no
  longer existed. The test caught it. This is the third time in this feature that removing a model
  left a summary or cross-reference restating it, after the skill's "most specific source winning"
  line and the discovery step's "always load regardless" sentence. Worth treating as the default
  expectation rather than a surprise.
- `knowledge.sources` could not be banned outright in the documentation tests, because the string is
  still correct for the project's own config. The test isolates the repo-configuration section and
  checks only its fenced YAML, and symmetrically asserts the project example still carries
  `sources:` and `- name:`, so an over-eager future removal of the legitimate shape fails too.
- The readme now qualifies its categories heading as "two **retrieval** tiers" and states the two
  axes explicitly, since it is the one document where a category's retrieval tier and a store's
  addressing tier appear within a few paragraphs of each other.

### 2026-09-04 — Phase 4.1: The site's layout knowledge is recorded through the delivered behaviour

**What was done**: Two knowledge entries about the documentation site, one recording its layout
conventions and one recording the reasoning behind its choice of text width, were recorded into the
documentation repository's own knowledge store through the delivered command surface rather than by
editing files. This is exactly the write that could not be expressed before this work, because there
was no way to say which repository's store was meant.

**Deviations**: The plan's open question anticipated this. The user no longer held the original
drafts, so both bodies were written from scratch, presented in full as text, and approved before
either write. Rather than working from the plan's one-line summary, they were grounded in the site's
actual layout system: `src/styles/global.css`, whose own comments already carried the frame and flow
rationale and the `--spek-centered-measure: 52ch` cap, `Section.astro`'s single `max-w-[1100px]`
frame, and the heading scale `clamp(1.625rem, 3vw, 2.25rem)` repeated across `.spek-prose h2`,
`SectionHeader.astro` and `CtaBanner.astro`. The user asked for column widths and heading styles
specifically, so the conventions entry names those values rather than describing them.

**Files changed**:
- `docs: .spektacular/knowledge/conventions/site-layout.md` (new, written via the CLI)
- `docs: .spektacular/knowledge/decisions/frame-width-flow.md` (new, written via the CLI)

**Discoveries**:

- **The dogfood write succeeded first time, for both entries, with no fallback to editing files.**
  That was the success metric the plan classified as manual, and it is the strongest evidence the
  addressing model works in the hand rather than only in tests: the destination was stated once,
  approved once, and landed where it was addressed.
- Verified two-sided, not just positively: both entries read back at
  `{"tier":"repo","name":"docs",...}`, both return `knowledge_entry_not_found` in the `spektacular`
  store, and a search for a distinctive phrase reports tier `repo` and name `docs` as their source.
- The rationale for the layout decision already existed as a comment block in `global.css`. Moving it
  into the knowledge base makes it reachable by the agent at plan time rather than only by whoever
  happens to open the stylesheet, which is the point of recording it.
