---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Plan: 000049_guided-repo-add

<!-- Metadata -->
<!-- Created: 2026-09-07T10:07:46Z -->
<!-- Commit: 6c72adc -->
<!-- Branch: f-project-repos -->
<!-- Repository: git@github.com:jumppad-labs/spektacular.git -->

## Overview

Adding a repository to a project becomes a guided conversation rather than a form to fill in. The user is asked one thing, which repository they want to add, and every remaining detail arrives as a concrete proposal drawn from the repository itself, so agreeing is enough to record it; where Spektacular's own files go is settled quietly by default, nothing is written until the user confirms in their own terms, and nothing Spektacular says to them names its internal vocabulary, the files it writes, or the commands it runs. The flow is a workflow the CLI owns, with state of its own, so an interrupted add resumes where it stopped and an add can run alongside a specification or plan already in progress. People adding a second or third repository stop being interrogated for facts their own code already states, and the single-command form continues to work unchanged for callers that already know every detail.

## Conventions

- **Error messages must describe the problem and suggest remediation** (spektacular) — this feature adds a whole family of new failure paths: `repo new` bouncing off an in-progress guided add, `repo goto` called for a step the FSM will not accept, a `location` that is not a directory, a target folder that cannot be written to, and a registration that fails after confirmation. Every one must be built with `output.NewError(code, message).WithNextAction(...)` naming the exact corrective command, never a bare `fmt.Errorf`. The stakes are higher here than usual, because the driving agent is mid-conversation with a user and a guessing agent will improvise a workaround in front of them.
- **Passing tests are required before calling work done** (spektacular) — `go test ./...` passes in full on the current tree and must pass after every phase. This feature additionally changes what the test suite covers: the automated check over standing instructions works from an explicit list of workflows rather than discovering them, so a phase that adds this flow's instructions without extending that list leaves them silently unchecked instead of failing.
- **Plans must sketch content structure, not just summarize it** (docs) — the documentation requirement rewrites a substantial block of published prose (`src/pages/projects.mdx:159-194`) and introduces a worked dialogue that does not exist yet. The phase covering it must carry a labelled **Content outline** with headings in order and an illustrative example per section, including the exact dialogue shape, not a one-paragraph summary of what the page should convey.
- **MDX authoring conventions** (docs) — the rewritten section is authored inside existing section components with native slot content, blank lines around every slot body, and the guided-flow dialogue as a fenced triple-backtick block rather than a JSX string prop. No `<div>`, `<section>` or `class=` may appear in the page body; if the new copy needs a shape no existing component provides, it becomes a component under `src/components/sections/`.
- **Alternate section background shading** (docs) — the "Registering a repository" section currently resolves to `surface={false}` after a shaded predecessor. If the rewrite splits it or adds a sibling section, each new band must set `surface` explicitly to the opposite of its own immediate predecessor, and must not copy the existing shaded-then-shaded adjacency between bands five and six on that page.
- **No em dashes** (docs) — applies to every prose surface this plan ships as content: the rewritten docs page copy, the README's registry paragraph, the rewritten `spek-manage-repos` skill, and the new step templates. It is scoped to authored deliverables, following the scoping used by plan 000046, not to this plan's own working documents.

Deliberately not carried: **Label before filename in file-scoped reference headings** (docs) is dropped because the documentation change edits a narrative section on the projects page and adds no new file-scoped reference heading; the `repo.yaml` and `config.yaml` headings on the configuration page are untouched by this work.

## Architecture & Design Decisions

**Options weighed for the primary axis: how a guided add gets workflow state without competing for the one the project already has.** *Option A, a second state file*: the guided add becomes a fourth CLI-owned FSM whose state is persisted to `.spektacular/repo-state.json` under the kind `repo`, while spec, plan and implement keep sharing `.spektacular/state.json` exactly as they do today. Pros: the engine needs no change at all, because `workflow.New(steps, statePath, cfg, st, out)` already takes the state path as a parameter (`internal/workflow/workflow.go:73`) and the only place the shared filename is fixed is a single `cmd`-layer helper (`cmd/spec.go:106-108`); the resume machinery needs no change either, because `resumeOrClear(statePath, command, expectedKind, force)` and `guardKind(statePath, command, expectedKind)` already take both the path and the kind as parameters (`cmd/resume.go:133-152,169-181`), so they are reusable verbatim against a different file, with their cross-kind branch simply never firing when only one kind ever writes it; and the spec's "must not compete, must not be blocked, must not disturb" trio is satisfied structurally rather than by careful behaviour. Cons: two state files now exist where contributors expect one, and the debug session log's probe reads only `state.json` (`cmd/root.go:187-215`), so a guided add produces no session record until that is extended. Effort: Low. *Option B, one state file holding several concurrent workflows*: extend `workflow.State` into a map keyed by kind. Pros: one file, one mental model, and the debug probe keeps working for every kind. Cons: the on-disk shape (`internal/workflow/state.go:14-21`) has four independent readers that would all break together, including a private mirror struct in `cmd/root.go`, `internal/sessionlog`, the `writeInProgressState` fixture helper in `cmd/cross_kind_test.go`, and the harbor python oracles which index `state["data"]["name"]` directly (`tests/harbor/spec-workflow/tests/test_spec_workflow.py:53-58`); and it converts a per-project singleton into a keyed collection for the benefit of exactly one new caller. Effort: High, and disproportionate. *Option C, a stateless renderer*: since every answer already arrives as `--data` on each `goto`, persist nothing and let the agent carry the accumulated payload forward turn by turn. Pros: the smallest possible change, no new file, no new kind. Cons: it fails the spec's resume requirement outright, since an interrupted add would have nothing to resume from and the agent's context is the only place the answers ever lived. Effort: Low, but it does not meet the specification. **Option A is chosen.** Rejected options and their evidence are recorded in `research.md#alternatives-considered-and-rejected`.

**The shape of the workflow.** A new `internal/steps/repo` package declares a linear ten-state FSM in the same `Steps()` shape every existing workflow uses (`internal/steps/spec/steps.go:25-39`): `new`, `locate`, `name`, `description`, `role`, `tags`, `placement`, `confirm`, `register`, `finished`, with the terminal state named `finished` because `State.InProgress()` hardcodes that literal (`internal/workflow/state.go:25-27`). `new` takes an optional `location` in its payload and asks the user nothing. `locate` is the only question asked cold, and its instruction branches on a mustache inverted section so that a repository already named in the user's request is recorded and skipped rather than asked for again. Two of the states are conditional in their instruction rather than in the graph, which is how the spec/plan workflows already handle "sometimes this step has nothing to ask" (`templates/steps/spec/00b-interview.md` tells the agent that a short interview, or none at all, is correct): `locate` skips silently when the folder is known, and `placement` records the default silently whenever the target folder is writable and the user has not said the repository should receive nothing but code. The one graph-level conditional is delegation: `placement` declares `Src: ["name", "description", "role", "tags"]`, so when the user answers the first proposal with "just use what you think", the agent advances straight to `placement` carrying all four proposed values in a single `goto` payload, and no further question about them is ever rendered. Multi-source transitions are already an engine capability with a test behind them (`internal/workflow/workflow_test.go`, `TestMultiSourceTransition`), so this costs nothing new.

**Where the examination lives, and why it is split.** Producing a proposal needs two capabilities the project keeps in two different places: reading a repository, which only Go can do reliably and testably, and writing a sentence about it, which only the agent can do at all. There is no language model in the binary and no language-detection or README-parsing dependency in `go.mod:5-12`. So the split is: a pure examiner function in `internal/repo` reads what a repository says about itself, bounded to the README's opening prose, the identity declared by whatever manifest is present (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`), the top-level directory entries, and an extension histogram; the `name` step's callback runs it once, stores the evidence in the workflow data map so it survives resume and is never recomputed, and pushes it into its own instruction through `stepkit.StepRequest.Extra`, which is the existing precedented channel for a callback to put computed facts in front of the agent (`internal/stepkit/stepkit.go:55-94`, used today by the plan workflow's repo roster at `internal/steps/plan/steps.go:88-97`). The `description`, `role` and `tags` steps read that stored evidence back and render it again. This keeps the examination inside the "what a repository states about itself" boundary the spec draws, keeps it fast enough to sit between two questions because it is a handful of bounded file reads, and gives the acceptance criterion "proposals are drawn from the repository, not the folder name" a real oracle: the examiner is a pure function over a directory and is table-testable against synthetic trees, in the `t.TempDir()` idiom this area already uses (`cmd/repo_test.go:70-94`). When a repository offers nothing to read, the examiner returns empty evidence and the templates' inverted sections instruct the agent to ask without proposals, which is exactly what the "must not require a repository to be readable" constraint demands.

**Who writes, and why not the agent.** The registration core of `runRepoAdd` — everything from the config load through the idempotent registry upsert, `EnsureFootprint`, and the write of the repository's own descriptive fields (`cmd/repo.go:113-252`) — is extracted into a single shared function that both `repo add` and the guided workflow's `register` step call. This is the standing preference for a DRY extraction over a copied code path, and it is also the only way "the direct add is unchanged" is guaranteed by construction rather than by parallel maintenance: there is one write path, and `repo add`'s handler becomes a thin argument-parsing shell in front of it. The alternative of having the `register` step's instruction tell the agent to run `repo add --data '<rendered payload>'` was rejected for a reason specific to this feature. The diagnosis behind the whole specification is that the vocabulary leak was never a set of bad sentences but a taught vocabulary and shape; rendering a payload carrying `location`, `source` and `provider` into the agent's context at the final step is that same teaching, delivered at the one moment the agent is also composing the user-facing confirmation. Keeping the payload entirely inside Go removes the temptation rather than warning against it. It also makes "nothing is written before confirmation" a property of the state machine instead of a property of agent obedience: callbacks are registered as `before_<event>` precisely so a failure vetoes the transition before `enter_state` persists anything (`internal/workflow/workflow.go:113-133`, and the recorded gotcha at `.spektacular/knowledge/gotchas/fsm-cancel-only-works-before-transition-commits.md`), so a declined confirmation leaves the workflow on `confirm` with the target repository and the project registry both untouched. This does qualify the recorded rule that step callbacks must not write files directly (`.spektacular/knowledge/architecture/working-with-files-from-steps.md`): that rule protects `.spektacular/` file plumbing from bypassing `store.Store`, and registration is a domain operation writing into a repository outside that root, routed through the same function the direct command uses.

**Placement, and what the two outcomes actually mean.** Nothing in Go chooses between putting Spektacular's files inside the repository being added and putting them in a folder under the project; `runRepoAdd` always writes the footprint into the folder named by `location` (`cmd/repo.go:130-138`), and "separate" is achieved purely by the caller passing a project-side `location` together with a `source` naming the code. Today that choice lives only in the skill, which is why it leaked terminology. The `placement` step makes it a workflow decision with two concrete outcomes: the default records `location` as the target repository's own folder and lets the written `repo.yaml` take the default file source of `..`; declining records `location` as `repos/<name>` under the project root and a `source` pointing at the target repository's absolute path, leaving that repository with no new folder. Both are expressed to the agent as a placement choice with a known consequence, never as a pair of named layouts, and the writability probe that decides whether to ask at all is a real create-then-remove filesystem operation rather than an inspection of permission bits, because permission bits do not answer the question reliably across the platforms in play. Vocabulary containment is enforced at two levels rather than trusted: a template-contract test asserts every `templates/steps/repo/*.md` carries the standing rule that Spektacular's internal terms, file names and command text belong to the agent and never to the user, following the marker-assertion pattern the existing contract tests already use (`templates/context_directive_test.go`, `templates/rejection_repair_directive_test.go`); and a harbor container suite parses the real agent transcript for the banned terms, which is the only mechanism in the project capable of checking a criterion phrased about what the user was shown (`tests/harbor/spec-workflow/tests/test_spec_workflow.py:93-125`). The guided flow is driven by a rewritten `spek-manage-repos` skill that stops being a static playbook and becomes a workflow driver in the shape of `templates/skills/workflows/spek-plan/SKILL.md`, carrying the same loop discipline and resume-report branch; the eight-step playbook currently sitting in that file uncommitted is superseded, its content moving into the step templates where the CLI owns it. In the docs repo, the "Registering a repository" section of `src/pages/projects.mdx:159-194` is rewritten around the guided flow with a worked dialogue in the house voice established by `src/pages/how-it-works.mdx:228-261`, keeping the single-command form documented as the non-interactive path.

## Component Breakdown

**spektacular repo**

- **Guided add workflow (new)** — the ordered step table and callbacks for the guided registration flow: locate the repository, propose and confirm its name, description, role and tags one at a time, decide where Spektacular's files go, confirm in the user's own terms, register. It owns the step graph, the delegation shortcut that lets a user hand the whole set of proposals over in one answer, and the rule that nothing is written before the confirmation step has been passed. It reads and writes only the workflow data store, and calls the shared registration component for the single mutating action at the end. It is a peer of the spec, plan and implement step tables, built from the same step-configuration shape, and is deliberately not a member of the group that shares the project's single workflow slot.

- **Guided add commands (new)** — the two command handlers that start and advance the guided flow. Starting it detects an add already in progress and returns a resume report instead of clobbering it; advancing it accepts the answer gathered for the current step and renders the next instruction. They own the guided add's state location, which is separate from the one spec, plan and implement share, and its workflow kind. They own nothing about the conversation itself; that lives in the step instructions.

- **Repository examiner (new)** — a pure reader that answers "what does this repository say about itself": the opening prose of its README, the identity its language manifest declares, its top-level entries, and which languages are present by weight. It owns the boundary of that examination, deliberately stopping short of anything that would require reading code, and it owns returning an explicitly empty result for a repository that offers nothing readable, so callers branch on evidence rather than on errors. It has no knowledge of the workflow, the conversation, or the registry; the proposal steps call it and decide what to do with what it returns.

- **Repository registration core (extracted from the existing add command)** — the single place a repository is registered: the idempotent registry upsert by name, the footprint creation or repair, and the write of the repository's own descriptive metadata and code location. It becomes a shared component called by two callers, the existing single-command add and the guided workflow's final step, so there is exactly one write path and the single-command form's behaviour cannot drift from the guided one. It owns the missing-metadata notice it already emits.

- **Existing repo commands (changed)** — the single-command add keeps its exact argument surface, validation, output shape and behaviour, and becomes a thin parsing shell in front of the shared registration core. The listing command is unchanged. Both continue to be reachable without touching the guided flow.

- **Guided add step instructions (new)** — the per-step instruction texts the CLI renders and the agent performs. They own the entire user-facing conversation: which question is asked, in what order, how a proposed value is stated inside the question so that agreement alone records it, when a question is skipped because its answer is already known or its default is safe, and the standing rule that Spektacular's internal terms, the files it writes, and the commands it runs belong to the agent and never appear in what the user is shown. They also own the confirmation text that names the repository, the folder to be created, and where the code lives.

- **Guided add skill (changed)** — the agent-facing driver for the flow. It stops being a static playbook that improvises the conversation and becomes a workflow driver: start, read the returned instruction, perform it, advance, repeat until the workflow reports itself finished, with the resume-versus-restart branch handled the way the spec and plan skills already handle it. The conversational content it carries today moves into the step instructions; what remains is loop discipline and resume handling. Registry inspection and footprint repair guidance stay where they are, unchanged.

- **Resume instruction text (changed)** — the shared text shown when an unfinished workflow is found gains a branch for a guided add, which has no per-section working files to read back because its gathered answers live in the workflow's own data.

- **Skill and command installation registry (changed)** — the table mapping skill names to their templates, and the descriptions used to generate command wrappers, so the rewritten skill is installed for every supported agent.

- **Template contract tests (changed)** — the suite that walks the embedded instruction templates and asserts each carries the standing directives its workflow requires. Its coverage set is a fixed list of workflow directories and must be extended to include the guided add's, or the new instructions ship unchecked. It gains an assertion that every guided-add instruction carries the vocabulary-containment rule.

- **Guided add end-to-end suite (new)** — a containerised run of a real coding agent against a seeded project, whose verifier parses the agent's own transcript. It owns the acceptance criteria that are statements about the conversation rather than about the resulting files: that only the repository is asked for cold, that each later question states a proposed value, that the questions arrive one per exchange in a fixed order, that delegation ends the questioning, that the confirmation names repository, folder and code location, and that none of Spektacular's internal vocabulary appears in anything the user was shown. It is the counterpart to the existing spec, plan and implement suites and is run the same way, by hand rather than in continuous integration.

- **Cross-workflow isolation tests (changed)** — the existing suite that proves a workflow of one kind refuses to be operated by another kind's commands and leaves its state byte-for-byte untouched. It gains the cases proving a guided add starts, runs and finishes while a spec is in progress, that neither disturbs the other's state, and that the spec resumes afterwards at the step it was on.

- **In-repo documentation (changed)** — the README's registry paragraph, which currently describes registration only as a single command, gains the guided flow as the way a repository is normally added and keeps the single-command form as the option for a caller that already knows everything.

**docs repo**

- **Multi-repo projects page (changed)** — its registration section is rewritten around the guided flow: a repository is added through a conversation that asks one question at a time and proposes an answer to each, shown as a worked dialogue in the voice the site already uses for the spec interview, with the single-command form retained and labelled as the non-interactive path for callers that already know every detail. The instruction to supply name, description, role and tags as command arguments during an interactive add is removed. The neighbouring explanation of which metadata lives where is left in place and becomes the reference the new copy points at.

## Data Structures & Interfaces

No existing serialised format changes. The project registry entry, the repository's own configuration file, the single-command add's payload and output, and the listing projection all keep their current shapes exactly. What this feature adds is one new workflow's step table, one persisted state file in the shape the engine already writes, two new command payloads, one new value type for what a repository says about itself, and one extracted function signature shared by two callers.

**Guided add step table** — an instance of the existing step-configuration shape, not a new type. The graph is linear apart from one multi-source transition, which is what carries the delegation shortcut: answering the first proposal with "use what you think" advances straight to placement with every proposed value supplied in that one call.

```go
{Name: "new",         Src: []string{"start"},   Dst: "new"}
{Name: "locate",      Src: []string{"new"},     Dst: "locate"}
{Name: "name",        Src: []string{"locate"},  Dst: "name"}
{Name: "description", Src: []string{"name"},    Dst: "description"}
{Name: "role",        Src: []string{"description"}, Dst: "role"}
{Name: "tags",        Src: []string{"role"},    Dst: "tags"}
{Name: "placement",   Src: []string{"name", "description", "role", "tags"}, Dst: "placement"}
{Name: "confirm",     Src: []string{"placement"}, Dst: "confirm"}
{Name: "register",    Src: []string{"confirm"}, Dst: "register"}
{Name: "finished",    Src: []string{"register"}, Dst: "finished"}
```

The terminal step is named `finished` because the engine's in-progress test compares against that literal; a differently named terminal state would make an add that ran to completion look permanently unfinished to the resume check.

**Persisted state** — the engine's existing state shape, written to a file of its own rather than the one spec, plan and implement share, and stamped with a new workflow kind. Nothing about the shape changes; only the path and the kind value are new.

```json
{
  "kind": "repo",
  "current_step": "role",
  "completed_steps": ["new", "locate", "name", "description"],
  "created_at": "...", "updated_at": "...",
  "data": {
    "location": "../../spektacular-website",
    "name": "docs",
    "description": "Marketing and documentation site, built with Astro and Tailwind",
    "role": "documentation",
    "tags": ["docs", "website", "astro"],
    "placement": "inside",
    "evidence": { }
  }
}
```

The data map is the whole of the gathered-answer store: every answer the user agrees to is seeded by the call that advances past its question, and the engine persists it on the transition, so an interrupted add resumes with everything already agreed intact and nothing to re-ask. `evidence` caches the examination so it is performed once, not once per proposal. `placement` is one of two values, meaning Spektacular's files go inside the repository being added, or in a folder under the project with the repository named as the source of the code.

**Command payloads** — both new commands use the established `--data` JSON convention.

```json
// repo new --data   (all fields optional; --force discards an add already in progress)
{"location": "../../spektacular-website"}

// repo goto --data  (step is required; the remaining field is the answer just agreed)
{"step": "description", "description": "Marketing and documentation site, built with Astro and Tailwind"}
{"step": "placement", "name": "docs", "description": "...", "role": "documentation", "tags": ["docs"]}
```

The second form is the delegation shortcut: one call carrying every proposal the user handed over, landing on the step after the last question it answers.

**Repository evidence** — the value the examiner returns, and the only thing the proposal steps know about the repository they are describing. It reports readability rather than returning an error, so a repository that offers nothing to read is an ordinary case the instructions branch on, not a failure that aborts the add.

```go
// internal/repo
type Evidence struct {
    Readable   bool     // false when nothing useful could be read
    Readme     string   // opening prose only, truncated
    Manifest   string   // the manifest file found, e.g. "go.mod"
    Identity   string   // the name that manifest declares
    Summary    string   // the description that manifest declares, when it has one
    TopLevel   []string // top-level entries, directories first
    Languages  []string // languages present, most-weighted first
}

func Examine(dir string) Evidence
```

`Examine` has no error return and no side effects. Every field is optional; `Readable` is true when at least one of the prose, manifest or language fields is populated. The instruction templates render the populated fields and fall back to asking without a proposal when it is false.

**Shared registration** — the contract between the two entry points. Its input carries exactly the fields the single-command add accepts today, and its output carries exactly the three the single-command add reports today, so that command becomes a parsing shell in front of it with no observable change.

```go
// internal/repo
type Registration struct {
    Name, Location, Source string
    Description, Role      string
    Tags, Dependencies     []string
}

type RegistrationResult struct {
    Registered   bool
    Footprint    FootprintStatus // created | repaired | unchanged
    MetadataNote string          // set only when description, role and tags are all empty
}

func Register(cfg *config.Config, projectRoot string, in Registration) (RegistrationResult, error)
```

Because a guided add always supplies a description, a role and tags, `MetadataNote` is provably empty for one, which is the "a guided add always produces descriptive metadata" requirement expressed as a property of this contract rather than as a behaviour to remember.

**Step rendering** — the guided add supplies its own implementation of the existing path-strategy interface, contributing the repository's name and the absolute folder its code lives at as template variables, and naming the folder as the workflow's primary path. Its result struct mirrors the spec and plan ones: the step name, the repository name, the primary path, and the rendered instruction. Per-step computed facts, the cached evidence and the writability probe among them, reach their instruction through the existing per-call extras channel rather than through new result fields.

## Implementation Detail

**One genuinely new pattern: a workflow that is not in the shared slot.** Until now, "a workflow" and "the project's workflow" have been the same thing. Three step tables exist and all three take turns occupying one persisted state, which is why starting one while another is unfinished is a conflict the CLI has to arbitrate. This plan separates those two ideas for the first time: the workflow engine becomes a general facility that any command family can instantiate against a state location of its own, and the shared slot becomes one particular use of it that spec, plan and implement continue to share. A developer reading the changed code will find that the engine itself is untouched, because it always took its state location as a parameter, and that the arbitration machinery is untouched too, because it always took both the location and the expected kind as parameters. What changes is only that a second caller now passes different values for both. The clearest way to read the result is that the guided add is a peer of the three existing workflows in every respect except membership of the group that contends for one state.

**The conversation moves from a playbook to a state machine, and that is where most of the risk lands.** The behaviour this feature specifies already exists in prose, as a numbered flow inside the repository-management skill that the agent reads once and then improvises from. Moving it into a step table changes who owns each beat: the CLI decides which question comes next and refuses to be driven out of order, and the instruction for each step is rendered fresh with the facts that step needs rather than recalled from a document read many turns earlier. The pattern for writing those instructions is established and should be followed closely rather than reinvented, including the standing footer every non-terminal step carries and the explicit statement of what the step's stopping condition is. The one place this plan extends the pattern is a standing vocabulary rule, present in every one of the guided add's instructions, saying that Spektacular's internal terms, the names of the files it writes, and the commands it runs are the agent's business and never appear in what the user is shown. That rule is a peer of the existing standing directives, is asserted the same way the others are, and exists because the failure it prevents is not a wording slip but a vocabulary the agent absorbs and then reproduces in its own words.

**Two conditional behaviours are expressed in prose, one in the graph, and the distinction is deliberate.** Skipping the opening question when the user has already named the repository, and taking the default file placement without asking when the target folder can be written to, are both handled by the step's instruction branching on a fact the callback supplies, with the step still being entered and completed. Delegation is different: when the user hands over the whole set of proposals at once, several questions genuinely do not happen, and that is modelled as a transition the graph permits rather than as instructions the agent is told to skip. A reader tracing the flow can therefore take the step order at face value, and the only place they need to look for a shortcut is the one step whose sources are plural. The cost of this split is that a completed run's record of which steps ran is honest about delegation but not about a silently-skipped question; that is the right trade, because delegation is a decision the user made and a skipped question is an absence of one.

**The examination introduces a module boundary the codebase does not have yet: a reader that characterises a repository.** Everything the repository package does today either resolves where a repository is or creates and repairs the files Spektacular puts in it. Nothing reads a repository to describe it, and nothing in the dependency set helps, so this is written by hand and deliberately kept small. Its shape is a pure function from a directory to a value, with no error return, no configuration, and no awareness of the workflow that calls it. Two consequences matter for anyone extending it later. First, the boundary of what it reads is the feature, not an implementation detail: it stops at what a repository states about itself, and any future temptation to infer more by reading code is a change to the specification rather than an improvement to the function. Second, because it cannot fail, the "nothing readable" case is an ordinary value that the instructions branch on, which is what keeps an unreadable repository addable instead of turning it into an error path a caller might abort on.

**The single mutating action is extracted rather than duplicated, and the existing command becomes a shell.** Registration today is one linear handler that parses a payload and then does the work. The work half becomes a function two callers share, and the parse half stays where it is. A developer reading the existing add command afterwards will find it noticeably shorter and will see immediately that it makes no decisions of its own beyond validating its input. This is what makes the promise that the single-command form is unchanged verifiable rather than aspirational, since there is no second implementation that could drift, and it is also what lets the guided flow's final step be a plain call rather than an instruction telling the agent to run a command. Placing the write behind a step transition rather than in an instruction has a second effect worth naming: because a step's work runs before its transition is committed, a confirmation the user declines leaves the workflow standing where it was with nothing written, and that guarantee comes from the engine rather than from the agent doing as it was told.

**Testing gains a layer the codebase already has but has not applied here.** Roughly half of this feature's acceptance criteria are statements about a conversation rather than about files on disk, and the only mechanism that can observe a conversation is the containerised end-to-end harness that runs a real coding agent and captures its transcript. A new suite of that kind is therefore part of the feature rather than an optional extra, and it is the only place criteria such as "the questions arrive one per exchange" or "none of these words appears in what the user was shown" can be checked at all. Everything else is checked the way this area is already checked: the examiner as a pure function over synthetic directory trees, the registration contract through the existing command-level tests, and the isolation guarantee through the existing suite that proves one workflow's state is byte-for-byte untouched by another's commands. The suite of contract tests over the instruction templates needs one thing noticing: its coverage set is an explicit list of workflow directories rather than a discovery of them, so a new workflow's instructions are not covered until the list includes it, and a phase that adds instructions without extending that list ships them unchecked rather than failing.

## Dependencies

No new external libraries are introduced, and no prior spec or plan must land before this one starts. Every dependency below already exists in the tree today.

**Internal packages (spektacular repo)**

- **The workflow engine** — provides the state machine, the persisted state shape, the data store that carries answers between steps, and the guarantee that a step's work runs before its transition is committed. Needs no change: it already accepts the state location and the workflow kind as construction parameters, which is the whole basis of this plan's chosen direction.
- **The workflow arbitration helpers** — provide the "an unfinished workflow was found" detection, the resume report, and the discard-and-restart path. Needs no change: both helpers already take the state location and the expected kind as parameters, so they are reused as they are, against a different location and a new kind.
- **The shared step-rendering pipeline** — provides template lookup, the standard template variables, the per-call extras channel through which a step callback puts computed facts into its own instruction, and the workflow-specific result shape. Needs no change; the guided add supplies its own path strategy and result builder in the existing shape.
- **The repository package** — provides registry resolution, footprint creation and repair, and source resolution. Changed: it gains the repository examiner as a new reader, and it becomes the home of the extracted registration function. Its existing resolution, cloning, staleness and repair behaviour is untouched, which the spec requires explicitly.
- **The configuration package** — provides the registry entry and the repository's own configuration, their validation, and their YAML round-trip. Needs no change; the guided add writes the same fields through the same path the single-command add already uses.
- **The store abstraction** — provides scoped access to the project's own directory. Used by the guided add only where a step genuinely touches project files; the registration write deliberately goes through the repository package instead, because it writes outside that scope.
- **The output package** — provides the shared error envelope with its remediation field. Needs no change, but every new failure path this feature adds must be constructed through it, per the project's standing convention on error messages.
- **The agent install machinery** — provides the table that maps skill names to their templates and installs them per agent. Changed only by a description update; the skill's name is unchanged, so no new installation target appears.
- **The embedded template filesystem** — provides every instruction and skill template. Needs no change beyond the new files themselves, since it embeds its whole tree.

**External libraries**

- **The finite state machine library** — provides the transition graph and the callback hooks the engine wraps. No version change, no new usage pattern; the one behaviour this plan leans on, that a pre-transition callback's failure prevents both the state change and its persistence, is already relied on and already documented in the project's knowledge base.
- **The mustache template renderer** — provides instruction rendering, including the section and inverted-section tags that give each instruction its "evidence present" and "nothing readable" branches without any branching in Go. No version change.
- **The assertion library used across the test suite** — no version change.
- Nothing is added to the dependency set. The examination is written by hand precisely because no language-detection or manifest-parsing library is present, and adding one would exceed what the specification allows the examination to do.

**Repositories**

- **The documentation repository** — the published multi-repo projects page is a deliverable of this plan, so that repository must be present and buildable when the documentation phase runs. It has no code dependency on the CLI repository; the two changes are independent and can land in either order.

**Tooling**

- **The containerised end-to-end harness** — provides the only mechanism able to observe an agent's transcript, which roughly half of this feature's acceptance criteria are phrased in terms of. Already used by three existing suites and already wired into the build file; this plan adds a fourth suite and its build target. Worth flagging that it is run by hand and not by continuous integration, so the criteria it covers are verified deliberately rather than automatically.

**Planning dependencies**

- None. Two prior plans inform this work heavily, the one that introduced the interview-driven spec workflow and the one that separated a repository's location from its code, but both are complete and shipped, and nothing in this plan waits on them.

## Testing Approach

The feature splits cleanly into two halves that need different kinds of proof. One half is ordinary code with ordinary outputs: what the examiner reads, what gets registered, which state file is written, and what happens when two workflows exist at once. That half is covered by the fast test suite in the conventions this area already uses, with real temporary directories rather than mocked filesystems, real state files read back and asserted against, and the command tree driven end to end rather than its handlers called directly. The other half is a conversation, and roughly half the acceptance criteria are statements about what the user was asked and shown rather than about anything left on disk. Nothing in the fast suite can observe that, so a containerised run of a real coding agent, whose verifier parses the agent's own transcript, is a first-class part of this feature rather than an optional extra.

**Unit coverage concentrates on the examiner**, because it is the one genuinely new pure function and because the quality of every proposal rests on it. Table-driven cases over synthetic repository trees assert that a repository with a README and a language manifest yields evidence containing terms from those files and not merely the folder's name, that the languages reported reflect what is actually present, that a repository offering nothing readable yields an explicitly empty result rather than an error, and that the examination reads only what a repository states about itself. That last assertion is the one guarding the specification's boundary: it is what stops a later change from quietly turning a fast, shallow reader into a code analyser.

**Contract coverage sits over the instruction templates**, extending the suite that already walks the embedded templates and asserts each carries the standing directives its workflow requires. Two things must happen here. The suite's coverage set is an explicit list of workflow directories rather than a discovery of them, so it must be extended to include the guided add's or the new instructions ship unchecked; and it gains an assertion that every one of those instructions carries the vocabulary-containment rule. This is a weaker guarantee than the transcript check, and deliberately so: it proves the rule is present in what the agent is told, on every commit, while the transcript check proves the rule was followed, on demand.

**Command-level coverage** asserts the shapes and the guarantees that do not need an agent. That a guided add started with no repository named renders exactly one question and nothing about name, description, role, tags or placement. That a guided add started against a folder already named opens on the name proposal instead. That the answers agreed so far are persisted and that resuming an interrupted add returns the step it stopped at with those answers intact. That at the moment the confirmation is presented the target repository has no new folder and the project registry is unchanged, and that declining leaves both untouched, which the state machine makes true structurally because a step's work runs before its transition commits. That a guided add of a writable repository is asked no placement question and afterwards has a folder inside that repository, while a declined placement produces a registration whose files live under the project with the target repository unmodified. And that a completed guided add records a non-empty description, role and tags, and provably does not emit the missing-metadata notice.

**Regression coverage protects the single-command form.** The existing tests for it are the specification of "unchanged" and must pass untouched after the registration core is extracted, which is the point of extracting rather than duplicating: there is one implementation, so there is nothing to drift. The declared input and output schemas, the validation errors, the footprint tree, and the registry entry written are all already asserted, and none of those assertions may be edited as part of this work. An edit to one of them during implementation is a signal that the extraction changed behaviour and should stop the phase.

**Isolation coverage** extends the existing suite that proves one workflow's commands refuse to operate another's state and leave it byte-for-byte untouched. The new cases prove the guided add starts and finishes with a spec workflow in progress, reporting no conflict and requiring nothing to be discarded, that the spec's state file is unchanged across the whole add, and that the spec resumes afterwards at the step it was on with its gathered content intact.

**End-to-end coverage** is a new containerised suite alongside the three that exist, seeded with a project and a repository to add, driving a real agent through the skill and parsing its transcript. It owns the criteria that are only observable in the conversation: that the first question names nothing but the repository, that every question after it states a specific proposed value, that name, description, role and tags each appear in a separate exchange in that order with no message asking for more than one, that answering the first proposal with a delegation produces no further questions about the remaining three and registers the proposed values, that the confirmation names the repository, the folder and where the code lives and contains no command text, and that none of the banned internal terms, configuration file names, command names, arguments or flags appears anywhere in what the user was shown. This suite is run deliberately rather than on every commit, which is how the existing three are run.

**Deliberate gaps.** No tests are added for registry resolution, source resolution, cloning, staleness or footprint repair, because the specification puts all of them out of scope and their existing coverage is the guarantee that they did not move. No test asserts the debug session log records a guided add, because it deliberately does not. No new coverage is added for the documentation change beyond the site's own build and type check, since prose is verified by review.

**Success metrics.**

- *In a typical add, the user supplies one value in their own words and answers every other question by agreeing with what was proposed.* **Behavioural test**, in the end-to-end suite. The transcript is asserted to contain exactly one question answered with free-form content the user originated, the repository, with every later answer being agreement with a value the question itself stated.
- *Proposed values are accepted without correction far more often than they are corrected, and a field corrected more often than accepted indicates the examination is looking at the wrong things.* **Manual — captured in the implementation test plan.** This is a rate observed across many real adds by different people against different repositories, not a property of any single run.
- *Adding a repository takes under a minute of the user's attention, from the first question to the confirmation.* **Manual — captured in the implementation test plan.** Elapsed time in an automated run measures the agent and the container, not a person's attention.
- *Users adding a second or third repository do so without consulting documentation or asking what a question means.* **Manual — captured in the implementation test plan.** This is an observation about people, and no assertion can stand in for it.
- *Adding a repository partway through a spec or plan no longer forces a choice between the two.* **Behavioural test**, at two levels: the isolation suite asserts a guided add runs to completion with a spec in progress and leaves the spec's state byte-for-byte unchanged, and the end-to-end suite asserts the transcript contains no conflict report and no discard prompt.

## Milestones & Phases

### Milestone 1: One registration path, and a repository that can describe itself

**What changes:** Nothing a user can see. This is a deliberate internal cleanup, and it earns its own milestone because everything after it depends on both halves being right and because it is the only milestone that can be verified purely by existing tests continuing to pass. Registering a repository is currently one handler that parses a request and then does the work; the work half becomes a function that any caller can use, so that when a second way of adding a repository arrives there is still only one implementation of what adding a repository means, and the promise that the existing single-command form is unchanged is true by construction rather than by parallel maintenance. Alongside it, the system gains the ability to read a repository and say what it appears to be: the opening of its README, the identity its language manifest declares, what sits at its top level, and which languages it is written in. Nothing calls that reader yet. Both pieces are inert on their own, and both are load-bearing for the milestone that follows.

**Validation point:** The full test suite passes with every existing registration test unedited, and the new reader is covered by cases proving it draws on a repository's README and manifest rather than its folder name, reports the languages actually present, and returns an explicitly empty result rather than failing when a repository offers nothing to read.

#### - [x] Phase 1.1: One way to register a repository

**Repo:** spektacular

The work of registering a repository, adding it to the project's list, creating or repairing its Spektacular folder, and recording what it is, currently lives inside the handler for the single command that does it. That work moves into a function of its own, and the command becomes a thin shell that checks its input and calls it. Nothing about the command changes: the same request, the same result, the same files written. This exists so that when a second way of adding a repository arrives there is still only one definition of what adding a repository means.

*Technical detail:* [context.md#phase-11](./context.md#phase-11-one-way-to-register-a-repository)

**Acceptance criteria**:

- [x] Registering a repository with a single command produces exactly the registry entry, folder contents and reported result it produced before this change.
- [x] The declared shape of that command's request and response is unchanged.
- [x] Every existing test covering registration passes without being edited.
- [x] The registration work is reachable from anywhere in the codebase without going through the command layer.

#### - [x] Phase 1.2: A repository that can describe itself

**Repo:** spektacular

Spektacular gains the ability to look at a repository and report what it appears to be: the opening prose of its README, the identity declared by whatever manifest its language uses, what sits at its top level, and which languages it is written in. The examination stops there by design, it never reads code, and it is fast enough to sit between two questions in a live conversation. A repository that offers nothing readable produces an empty report rather than an error, so a caller can ask its questions without suggestions instead of giving up. Nothing calls this yet.

*Technical detail:* [context.md#phase-12](./context.md#phase-12-a-repository-that-can-describe-itself)

**Acceptance criteria**:

- [x] Given a repository with a README and a language manifest, the report contains terms that appear in those files and is not merely a restatement of the folder's name.
- [x] The languages reported reflect what the repository actually contains.
- [x] A repository with no README, no manifest and no recognisable content produces an empty report and no error.
- [x] The examination reads only what a repository states about itself, and reads no source files beyond identifying which languages are present.

### Milestone 2: Adding a repository becomes a guided conversation

**What changes:** This is the feature. Adding a repository stops being a request to fill in, and becomes a conversation that asks for one thing and proposes the rest. The user is asked which repository they want to add, and nothing else is asked cold: the system reads the repository and then asks about its name, then its description, then its role, then its tags, one at a time, each question already carrying a concrete suggestion drawn from what the repository says about itself, so agreeing is enough to record it. A user who would rather not review the suggestions individually can hand the whole set over in one answer and be asked nothing further about them. Where Spektacular's own files go is decided quietly, inside the repository being added, and is only raised as a question when that repository cannot take them or the user has said it should hold nothing but code. Before anything is written the system states, in the user's own terms, which repository is being registered, which folder will be created, and where its code lives, and waits. Nothing Spektacular says names its internal vocabulary, the files it writes, or the commands it runs. An add interrupted partway through picks up exactly where it stopped with every answer already given still in hand, and an add can be started and finished while a specification or plan is midway through, with neither disturbing the other. Registering a repository in a single command, for a caller that already knows every detail, works exactly as it does today.

**Validation point:** A guided add can be started, driven to completion, and the resulting registration carries a non-empty description, role and tags with no missing-metadata warning. An add interrupted after the description is agreed resumes at the next unanswered question without re-asking anything. A guided add runs to completion with a specification in progress, that specification's state is unchanged throughout, and it resumes afterwards at the step it was on. Declining at the confirmation leaves both the target repository and the project registry untouched. Every existing test passes.

#### - [x] Phase 2.1: A guided add that can be started, advanced and resumed

**Repo:** spektacular

The guided flow gets its own state machine and its own two commands, one to start an add and one to advance it. Its progress is recorded separately from the one specifications, plans and implementations share, so starting an add never collides with work already underway and never blocks it. Starting an add that is already in progress reports where it stopped and offers to resume or start over, exactly as the other workflows do. At this point the flow can be walked from beginning to end and interrupted and resumed, but its steps say little; the conversation itself arrives in a later phase.

*Technical detail:* [context.md#phase-21](./context.md#phase-21-a-guided-add-that-can-be-started-advanced-and-resumed)

**Acceptance criteria**:

- [x] An add can be started, advanced through every step in order, and reaches a finished state.
- [x] Advancing to a step the flow is not ready for is refused with a message naming what to do instead.
- [x] An add interrupted partway through can be resumed at the step it stopped on, with every answer already given still recorded.
- [x] An add can be started and completed while a specification is in progress; that specification's recorded progress is unchanged throughout and it resumes afterwards at the step it was on.
- [x] Starting a specification, plan or implementation while an add is in progress is not blocked by it.

#### - [x] Phase 2.2: The flow knows what the repository says about itself

**Repo:** spektacular

The step that asks about the repository's name examines the repository first, once, and keeps what it found for the rest of the conversation, so the steps that follow can each state a suggestion without repeating the work. The step that decides where Spektacular's files go first establishes whether the target repository can actually take them. Both facts are handed to the step's own instruction, so the instruction can either state a suggestion or, when there is nothing to suggest, say so.

*Technical detail:* [context.md#phase-22](./context.md#phase-22-the-flow-knows-what-the-repository-says-about-itself)

**Acceptance criteria**:

- [x] The repository is examined once per add, and the result survives an interruption and resume without being recomputed.
- [x] The steps asking about name, description, role and tags each have the examination's findings available to them.
- [x] The step deciding file placement knows whether the target repository can be written to.
- [x] When a repository offers nothing readable, the steps that would suggest a value instead have nothing to offer, and say so, rather than failing.

#### - [x] Phase 2.3: Nothing is written until the user says so

**Repo:** spektacular

The final step of the flow performs the registration, using the single shared implementation the direct command uses. It runs as part of moving past the confirmation, so a confirmation that is declined leaves the target repository and the project's list of repositories exactly as they were. The placement decision reached earlier is translated here into where the folder is created and where the repository's code is recorded as living, so declining to add a folder to a repository results in one that lives under the project instead, with that repository untouched.

*Technical detail:* [context.md#phase-23](./context.md#phase-23-nothing-is-written-until-the-user-says-so)

**Acceptance criteria**:

- [x] At the point the confirmation is presented, the target repository has no new folder and the project's list of repositories is unchanged.
- [x] Declining at the confirmation leaves both untouched and leaves the add resumable at the confirmation.
- [x] Confirming registers the repository with the name, description, role and tags gathered during the conversation.
- [x] A repository registered this way carries a non-empty description, role and tags, and produces no missing-metadata warning.
- [x] Declining to add a folder to the repository results in a registration whose files live under the project, with that repository left with no new folder.

#### - [x] Phase 2.4: The conversation itself

**Repo:** spektacular

Each step of the flow gets its instruction: what to ask, how to state the suggestion inside the question so that agreeing is enough to record it, when not to ask at all, and what never to say. Only the opening question is asked cold. A repository already named in the request is not asked for again. The four questions about name, description, role and tags come one at a time in that order, and a user who would rather not review them individually can hand the whole set over and be asked nothing further. Where the files go is decided quietly unless the repository cannot take them or the user has said it should hold nothing but code. Before anything is written, the flow states which repository is being registered, which folder will be created and where its code lives, in the user's own terms, and waits. Every instruction carries a standing rule that Spektacular's internal vocabulary, the files it writes and the commands it runs belong to the agent and never appear in what the user is shown.

*Technical detail:* [context.md#phase-24](./context.md#phase-24-the-conversation-itself)

**Acceptance criteria**:

- [x] Starting an add with no repository named produces exactly one question, asking which repository to add, and no question about anything else until it is answered.
- [x] Starting an add from a request that already names the repository's folder produces no question about which repository to add.
- [x] Each question about name, description, role and tags states a specific suggested value, such that agreeing alone records it unchanged.
- [x] Those four questions arrive one per exchange, in that order.
- [x] Answering the first suggestion by handing over the whole set produces no further questions about the remaining three.
- [x] Adding a writable repository produces no question about where Spektacular's files go.
- [x] Every instruction in the flow carries the rule that keeps Spektacular's internal vocabulary out of what the user is shown, and the automated check covering standing instructions now includes this flow.

#### - [x] Phase 2.5: An agent that drives the flow instead of improvising it

**Repo:** spektacular

The skill that handles repositories stops carrying its own account of how to add one and becomes a driver for the flow: start it, read what it says to do, do it, advance, repeat until it reports itself finished, with the resume-or-start-over branch handled the way the specification and plan skills already handle it. What it says about inspecting the list of repositories and repairing a broken registration is unchanged. The text shown when an unfinished workflow is found gains a branch for an add, which has no gathered sections to read back because its answers travel with it.

*Technical detail:* [context.md#phase-25](./context.md#phase-25-an-agent-that-drives-the-flow-instead-of-improvising-it)

**Acceptance criteria**:

- [x] The skill instructs the agent to keep advancing the flow until it reports itself finished, rather than stopping after the first successful call.
- [x] The skill describes what to do when an add is already in progress, including resuming it and discarding it.
- [x] The skill no longer contains its own step-by-step account of the add conversation.
- [x] The skill is installed for every supported agent by initialising a project.
- [x] Resuming an interrupted add is described without reference to gathered section files, which an add does not have.

### Milestone 3: The guided conversation is proved end to end

**What changes:** Nothing a user can see, but the half of this feature's promises that are about the conversation rather than about files stop resting on inspection and start being checked. A real coding agent is run against a real project inside a container and its side of the exchange is examined: that the opening question names nothing but the repository, that every question after it states a specific suggestion, that the four questions arrive one per exchange in a fixed order, that handing the whole set over ends the questioning and records the suggested values, that the confirmation names the repository, the folder and where the code lives without quoting a command, and that none of Spektacular's internal terms, file names, or command text ever appears in what the user was shown. This is the only kind of check capable of observing any of that, and it is worth its own milestone because it is where the feature's central claim either holds or does not.

**Validation point:** The new end-to-end run passes against a real agent, and the checks it applies to the conversation fail when deliberately given a transcript that breaks each rule, so the suite is known to be capable of catching a regression rather than merely passing.

#### - [x] Phase 3.1: Proving the conversation, not just the result

**Repo:** spektacular

A real coding agent is run against a seeded project inside a container, driven through the guided add, and its side of the exchange is examined afterwards. This is the only way to check the promises this feature makes about the conversation rather than about files: that only the repository is asked for cold, that every later question states a suggestion, that the questions arrive one at a time in a fixed order, that handing the set over ends the questioning, that the confirmation names the repository, the folder and where the code lives without quoting a command, and that none of Spektacular's internal vocabulary reaches the user.

*Technical detail:* [context.md#phase-31](./context.md#phase-31-proving-the-conversation-not-just-the-result)

**Acceptance criteria**:

- [ ] A real agent completes a guided add against a seeded project without human intervention.
- [ ] The recorded exchange shows exactly one question answered with content the user originated, and every later answer agreeing with a value the question stated.
- [ ] The recorded exchange shows name, description, role and tags each asked and answered separately, in that order.
- [ ] A separate run in which the user delegates shows no questions about description, role or tags, and registers the suggested values.
- [ ] Nothing shown to the user in the recorded exchange contains Spektacular's internal terms, the name of any file it writes, or any command, argument or flag.
- [x] Each of those checks fails when given an exchange that deliberately breaks the rule it covers.

### Milestone 4: The documentation describes the guided add

**What changes:** The published documentation stops describing registration as a form to be filled in on a command line. The page covering multi-repo projects describes adding a repository as a guided flow that asks one question at a time and proposes an answer to each, shows what that exchange actually looks like, and keeps the single-command form documented as the option for a caller that already knows every detail. The instruction to supply a name, description, role and tags as command arguments during an interactive add is gone. The project's own README is brought into line in the same way, so a reader arriving at either surface finds the same account of how a repository is added.

**Validation point:** The documentation site builds and type-checks cleanly with no layout markup introduced into the page body, the registration section describes the guided flow and retains the single-command form, and no instruction to supply descriptive metadata as command arguments during an interactive add remains on either the site or the README.

#### - [x] Phase 4.1: The published documentation describes the guided add

**Repo:** docs

The page covering multi-repo projects is rewritten where it describes registering a repository. Instead of presenting registration as a single command whose arguments the reader supplies, it describes a guided flow that asks one question at a time and proposes an answer to each, shows what that exchange looks like, and keeps the single-command form as the option for a caller that already knows every detail. The instruction to pass a name, description, role and tags as command arguments during an interactive add is removed.

**Content outline** for the rewritten "Registering a repository" section:

1. *Opening paragraph, replacing the current "in one command" framing.* Registration is a short conversation. Illustrative: "You are asked which repository you want to add. Everything else is proposed for you, drawn from the repository itself, so you confirm or correct a suggestion rather than supplying facts your own code already states."
2. *A worked exchange, as a fenced code block, in the voice used for the specification interview.* Illustrative:

   ```
   Agent: Which repository would you like to add?
   You:   ../../docs

   Agent: I'd call this one "docs". Sound right?
   You:   yes

   Agent: For a description: "Marketing and documentation site,
          built with Astro and Tailwind." Good, or would you
          put it differently?
   You:   that's right

   Agent: Its role looks like documentation. Agreed?
   You:   yes

   Agent: Tags: docs, website, astro. Anything to add or drop?
   You:   that's fine

   Agent: I'll register ../../docs as "docs", add a .spektacular
          folder inside it, and record its code as living there.
          Go ahead?
   You:   yes
   ```

3. *A short paragraph on what the flow decides for you.* Where Spektacular's files go is settled quietly, inside the repository being added, and you are only asked if that repository cannot take them or you have said it should hold nothing but code. Nothing is written until you confirm.
4. *A paragraph retaining the single-command form, relabelled as the non-interactive path,* keeping the existing examples for a checkout on disk and for code that lives elsewhere, introduced as "A caller that already knows every detail can register a repository in one command:" rather than as the primary route.
5. *The existing idempotency and removal notes, unchanged.*

*Technical detail:* [context.md#phase-41](./context.md#phase-41-the-published-documentation-describes-the-guided-add)

**Acceptance criteria**:

- [x] The page states that a repository is added through a guided flow that asks one question at a time and proposes an answer to each.
- [x] The page shows what that exchange looks like.
- [x] The page states that the single-command form remains available for callers that already know every detail.
- [x] The page contains no instruction to supply a name, description, role or tags as command arguments during an interactive add.
- [x] The site builds and type-checks cleanly, and no layout markup has been introduced into the page body.
- [x] Any new or changed section's background shading is the opposite of the section immediately before it.

#### - [x] Phase 4.2: The README tells the same story

**Repo:** spektacular

The project's own README currently describes registering a repository only as a single command with its arguments. It is brought into line with the published documentation, so a reader arriving at either surface finds the same account.

**Content example** for the README's registry paragraph, replacing the sentence beginning "`repo add` is pointed at a repo's code":

> A repository is normally added through a guided flow: you are asked which repository to add, and its name, description, role and tags are each proposed for you from what the repository says about itself, one question at a time, with a plain-language confirmation before anything is written. Spektacular's files go inside the repository being added unless it cannot take them or you say otherwise, in which case they live in a folder under the project and the repository is left with only its code. A caller that already knows every detail can still register a repository in a single command. Where the code lives is declared in the repository's own `repo.yaml` as `source`.

The surrounding sentences about the registry's `name` and `location` keys, the removed `address` key, and the fact that removal is a manual config edit are unchanged.

*Technical detail:* [context.md#phase-42](./context.md#phase-42-the-readme-tells-the-same-story)

**Acceptance criteria**:

- [x] The README describes adding a repository as a guided flow that proposes each value.
- [x] The README states that the single-command form remains available.
- [x] The README no longer presents supplying descriptive metadata as command arguments as the way a repository is added.
- [x] The README's account matches the published documentation's.


## Open Questions

Two uncertainties genuinely cannot be settled before the code exists, and both are about behaviour observed rather than behaviour designed. Everything else raised during planning was resolved and recorded, either in the architecture or in the assumption log.

- **Whether the written instructions actually produce one question per exchange from a real agent.** The requirement that name, description, role and tags each arrive as their own exchange is a property of what the agent does with an instruction, not of the state machine, which permits either. Nothing before the end-to-end run can tell us whether an instruction that says "ask about exactly one thing" is enough, or whether an agent will batch two questions into one message anyway. This depends on the instruction wording and on the model driving it, and it is the single most likely thing about this feature to need iteration. **What the implementer should do:** treat a batched question in the recorded exchange as a wording problem, not a design problem, and iterate on the instruction text; the step graph does not change. Only if repeated wording attempts fail should this come back to the user, because at that point the one-question-per-exchange requirement itself is in question.

- **Whether the bounded examination produces suggestions people actually accept.** The examination is deliberately shallow, and the specification forbids deepening it. Whether that shallowness is enough for the description and tags to be accepted more often than corrected can only be learned by running it against real repositories, which is exactly what the specification says a field corrected more often than accepted is a signal of. **What the implementer should do:** if a field is consistently wrong during the end-to-end runs, adjust which of the already-permitted sources it draws on and in what order, for instance preferring a manifest's declared description over the README's opening prose or the reverse. Do not widen what is read. If no ordering of the permitted sources produces usable suggestions for a field, STOP and ask the user, because the remedy would be to change the specification's boundary on the examination.

## Out of Scope

**From the specification's non-goals**

- **Inspecting the list of repositories and repairing a broken registration are not being made guided.** They keep their current shape. Note in passing that there is no separate repair command to leave alone: repair is what re-running an add or re-initialising the project already does to a footprint, and it stays that way.
- **Removing a repository remains a manual edit of the project's configuration.** No command is being added for it, guided or otherwise.
- **A stale clone continues to produce a warning and nothing more.** No automatic fetching or pulling is introduced, and the staleness check is untouched.
- **Characterising a repository by analysing its code is explicitly excluded.** The examination reads what a repository states about itself, and the boundary of what it reads is part of the specification rather than an implementation detail; widening it later is a change to the specification, not an improvement.
- **Specifications, plans and implementations continue to share one workflow slot among themselves and to collide with one another exactly as they do today.** Only the add is taken out of that contention. Making the other three concurrent is a separate piece of work and is not started here.
- **Other skills keep whatever vocabulary they use today.** The rule that Spektacular's internals belong to the agent rather than the user is applied to the add flow only, not swept across the rest of the skill surface.

**Left to a later plan by this design**

- **The debug session log does not record a guided add.** It reads only the state shared by the other three workflows and recognises only their command names, so an add produces no session record. This is a one-line extension if it turns out to matter, and it is deliberately not taken now because nothing in the specification asks for it.
- **The guided flow gets two commands, not four.** There is no separate command to report where an add has got to or to list its steps, as there is for specifications and plans. Starting an add already in progress reports where it stopped, which is the whole of what the resume requirement needs; the introspection commands can be added later if a need appears.
- **The registry's declared request and response shapes are frozen for the single-command form.** No fields are added to it and none are deprecated, even where the guided flow now makes some of them rarely typed by hand.
- **Nothing is done about the shaded-band irregularity on the multi-repo projects page.** Two adjacent sections there are both shaded, which departs from the site's alternation rule. This plan sets its own section's shading correctly relative to its predecessor and leaves the pre-existing adjacency alone rather than reflowing a page it is only partly editing.
- **The configuration reference page is not restructured.** It mentions adding a repository only as the event that creates a repository's configuration file, which stays accurate. At most it gets a passing touch-up; its key reference is left as it is.

## Changelog

### 2026-09-07 — Phase 1.1: One way to register a repository

**What was done**: The work of registering a repository, which lived inside the handler for the single command that does it, moved into a shared function of its own. The command is now a thin shell that validates its input, calls that function, and reports what it returned. Nothing about the command changed: same request, same result, same files written.

**Deviations**:

- **`Register` takes a `GitRunner` the plan's stated signature omits.** The plan gives `Register(cfg *config.Config, projectRoot string, in Registration)`, but the extracted body calls `repo.New(cfg, root, repoGit)` and then resolves the entry so a git source is materialized at registration time. `repoGit` is a package-level variable in `cmd` that the existing tests substitute with a stub. Without the parameter those tests could not stub git and would attempt real clones, breaking the very rule that defines this phase. Adopted: `Register(cfg *config.Config, projectRoot string, git GitRunner, in Registration) (RegistrationResult, error)`.
- **Four helpers moved, not the one the plan named.** The plan lists only the descriptive-fields-empty check. `reposEqual`, `repoConfigDescriptiveFieldsEqual` and their shared `stringSlicesEqual` are also inside the extracted body and had no callers outside `cmd/repo.go`, so all four moved. Only the empty-check needed exporting, as `DescriptiveFieldsEmpty`, because the listing command is its second caller; the other three are unexported in `internal/repo`.
- **The config path is derived rather than passed.** The old body wrote through a `cmd` helper returning `<cwd>/.spektacular/config.yaml`. Since the project root is also the working directory, `Register` derives the same path from `config.ProjectConfigDir(projectRoot)`, which keeps the parameter list at the plan's shape.

**Files changed**:

- `spektacular: internal/repo/register.go` (new)
- `spektacular: internal/repo/register_test.go` (new)
- `spektacular: cmd/repo.go`

**Discoveries**:

- **The result map stayed in the command layer on purpose.** `Register` returns a struct, and the command maps it onto the `{registered, footprint, metadata_note}` map itself, omitting the note when empty rather than emitting it blank. Moving the map into `Register` would have changed the emitted JSON and broken the frozen output schema. The guided add's register step will need to do the same mapping for itself rather than expecting the shared function to produce a payload.
- **`config.Config.Validate()` requires a slug-safe project name and at least one registered repository.** A test fixture that builds a bare config and calls `Register` against it passes only because the registration itself satisfies the repos rule; the name has to be set explicitly.
- **`EnsureFootprint`'s caller-supplied config has no effect once a `repo.yaml` already exists.** That is why writing the descriptive metadata and source is a separate step after it, and why re-registering an already-footprinted repository still updates its metadata. Anything that assumes the scaffold config is authoritative will be wrong on the second add.

### 2026-09-07 — Phase 1.2: A repository that can describe itself

**What was done**: Spektacular gained the ability to look at a repository and report what it appears to be: the opening prose of its README, the identity and summary declared by whatever manifest its language uses, what sits at its top level, and which languages it is written in. The examination never fails, so a repository that offers nothing readable produces an empty report rather than an error and stays addable. Nothing calls it yet.

**Deviations**:

- **A built-in vendored-directory skip set was added, beyond the exclusion file the plan named.** The plan says to honour the repository's own exclusion file so that vendored and build directories do not dominate the language histogram. Honouring it alone does not achieve that: smoke-tested against the project's real documentation repository, which carries no exclusion file, the dependency and build directories fed straight into the histogram, which is the exact outcome the phase says to avoid and which fails the criterion that reported languages reflect what the repository contains rather than what it depends on. A fixed set of well-known dependency and build directory names is now skipped when weighing languages only; a repository's own exclusion file is still honoured on top of it. Everything else in the examination is unaffected.

**Files changed**:

- `spektacular: internal/repo/examine.go` (new)
- `spektacular: internal/repo/examine_test.go` (new)

**Discoveries**:

- **Every manifest had to be hand-rolled except two.** There is no module-path parser, no TOML library and no manifest-parsing dependency available, and adding one would exceed what the specification allows the examination to do. Only the JavaScript manifest and the Java one have a standard-library parser behind them; the rest are deliberate line scans. A later phase that wants a new manifest format should expect to write another scanner, not reach for a library.
- **The two-level walk is load-bearing for more than speed.** Because the histogram never descends past the second directory level, a repository that keeps its source deeper than that reports fewer languages than it really has. That is the specified bound, not a defect, but any future complaint that "the languages look wrong" should be checked against depth before anything else.
- **The reported languages are a weak signal for a proposal and the manifest and README are strong ones.** In practice both registered repositories produced an accurate identity and description from their manifest and README, while the language list was the least distinctive part of the evidence. The proposal steps should lean on identity and summary first and treat languages as supporting detail, which also matters for the open question about whether proposals get accepted or corrected.

### 2026-09-07 — Phase 2.1: A guided add that can be started, advanced and resumed

**What was done**: Adding a repository is now a workflow the CLI owns, with two commands to start and advance it and progress recorded in a place of its own. It can be walked from beginning to end, refuses to be driven out of order, reports where it stopped when started a second time, and can be discarded and restarted. Because its progress is kept separately from the one specifications, plans and implementations share, an add and a specification can now be in progress at the same time without either disturbing or blocking the other. The steps say little so far; the conversation itself arrives in a later phase.

**Deviations**:

- **Nine step instructions, not the ten the plan lists.** The plan asks for an opening instruction for the internal first step while also specifying that step asks the user nothing. Both cannot hold: a step that renders an instruction stops there and the user sees it. The planning workflow already settled this, its own first step advances silently and has no instruction file, and the guided add now follows that precedent. The practical effect is the one the next phase's first acceptance criterion requires: starting an add renders the first real question rather than a preamble.
- **A parent-level dry-run option had to be added.** The two new commands were modelled on the specification workflow's, which read a dry-run option declared on their parent. The repository command group had no such option, so the copied lookup would have silently reported false and a dry run would have written real state. The option is now declared alongside the existing one.

**Files changed**:

- `spektacular: internal/steps/repo/steps.go` (new)
- `spektacular: internal/steps/repo/strategy.go` (new)
- `spektacular: internal/steps/repo/result.go` (new)
- `spektacular: templates/steps/repo/01-locate.md` through `09-finished.md` (new, nine files)
- `spektacular: cmd/repo.go`
- `spektacular: cmd/repo_workflow_test.go` (new)
- `spektacular: cmd/cross_kind_test.go`
- `spektacular: cmd/repo_test.go` (reset helper only)
- `spektacular: templates/context_directive_test.go`

**Discoveries**:

- **The automated check over standing instructions is an allow-list and was extended in this phase.** Without that one-line addition the nine new instructions would have shipped entirely unchecked while the suite stayed green. Its terminal-file exemption keys off a filename ending in `-finished.md`, so a terminal instruction named any other way would be wrongly required to carry the standing footer. Two stale comments in that file, naming three workflows and counting their instructions, were corrected.
- **The path strategy is built per call here, not once per workflow.** The specification and planning workflows configure theirs from settings known before the workflow starts. This one names the folder the repository's code lives in, which is gathered during the conversation, so it is constructed on every step and reports nothing until the locating step has run.
- **The workflow's own name has to be seeded before it is agreed.** Both the instruction renderer and the resume report read the workflow instance's name from the same gathered value, so an add interrupted before the naming step would otherwise report an unfinished workflow with no name. The locating step now seeds it from the target folder, and the naming step supersedes it.
- **Two failures in the existing command test suite appear only when it is run twice in one process, and predate this work.** They were confirmed to reproduce with this phase's test files removed. They are option-state leaking between runs in the existing suite, and are worth fixing separately.

### 2026-09-07 — Phase 2.2: The flow knows what the repository says about itself

**What was done**: The step that asks about the repository's name now examines the repository first and keeps what it found for the rest of the conversation, so the steps that follow each state a suggestion without repeating the work. The step that decides where the project's files go first establishes whether the target repository can actually take them, by trying rather than by reading permission bits. Both facts reach the step's own instruction, so an instruction either states a suggestion or, when the repository says nothing about itself, says there is nothing to suggest and asks plainly.

**Deviations**: None.

**Files changed**:

- `spektacular: internal/steps/repo/probe.go` (new)
- `spektacular: internal/steps/repo/steps.go`
- `spektacular: internal/steps/repo/steps_test.go` (new)
- `spektacular: templates/steps/repo/02-name.md`, `03-description.md`, `04-role.md`, `05-tags.md`, `06-placement.md`

**Discoveries**:

- **What gets cached must be the rendered variables, not the examination value, and the reason is invisible until someone resumes.** Gathered answers are persisted as JSON and read back as a plain map. Caching the examination's own value would mean an instruction renders from a typed value in the session that gathered it and from a map after a resume, and the two do not address their fields the same way. The failure would be silent rather than loud: a step would quietly claim the repository says nothing about itself even though the findings were sitting in the recorded state. Normalising to a lowercase-keyed map at the moment of caching makes both paths identical. Any future value cached in gathered answers has the same hazard.
- **The strongest available test of "examined once" is to destroy the evidence between two steps.** Deleting the repository's README and manifest after the first proposal and then rendering the next one proves reuse in a way no call counter can, because it also proves the reuse survives a process boundary. This is now covered both as a unit test and end to end through the real command.
- **The writability probe must be guarded in tests against being run as root**, which bypasses permission bits and would make the read-only case silently pass for the wrong reason. The test skips when the effective user id is zero and restores the directory mode afterwards, without which the temporary directory cleanup itself fails.

### 2026-09-07 — Phase 2.3: Nothing is written until the user says so

**What was done**: The final step of the flow performs the registration, through the single shared implementation the direct command uses. It runs as part of moving past the confirmation, so a confirmation that is declined, or a registration that fails, leaves the flow standing where it was. The placement decision reached earlier is translated here into where the folder is created and where the repository's code is recorded as living, so choosing to leave a repository holding nothing but code produces a registration whose files sit under the project instead.

**Deviations**:

- **The registration needs two things a step is not given.** A step callback receives the gathered answers, an output writer, the project store and the workflow settings. It receives neither the project configuration nor the means to fetch a remote repository, both of which registering requires. Resolved without widening the callback signature: the project store knows the project root, so the step loads the configuration from its usual place beside it and constructs the fetcher itself.

**Files changed**:

- `spektacular: internal/steps/repo/registration.go` (new)
- `spektacular: internal/steps/repo/registration_test.go` (new)
- `spektacular: internal/steps/repo/steps.go`
- `spektacular: cmd/repo_workflow_test.go`

**Discoveries**:

- **A pre-existing defect surfaced and was deliberately left alone.** A registration that fails after the project configuration has been written leaves an entry for a repository whose folder does not exist, which listing then reports as a broken location. The order inside the shared registration is: write the configuration, then create the folder, then scaffold it. This was confirmed to be the direct command's existing behaviour rather than anything this work introduced, by pointing that command at a read-only target and seeing the identical half-written result. Reordering or adding a rollback would change behaviour that an earlier phase froze and proved unchanged, so it is recorded here for a separate fix rather than taken now.
- **The two entry points resolve a location from different starting points, and both are correct.** The guided flow resolves what the user typed relative to the working directory, which is what a person means by a relative path, then converts it into the form the registry stores. The direct command stores its argument as given, and it is resolved later relative to the folder holding the project configuration. They arrive at the same place from different origins. This also makes a trap when reproducing failures by hand: a relative location passed to the direct command from the project directory quietly creates a different, writable folder, so the command appears to succeed when it should not.
- **Gathered answers come back from storage in a different shape than they went in.** A list of tags supplied during a session is a list of strings; the same list read back after an interruption is a list of untyped values. Anything reading a list out of gathered answers has to accept both, which is now handled in one place.

### 2026-09-07 — Phase 2.4: The conversation itself

**What was done**: Every step of the flow got its instruction: what to ask, how to state the suggestion inside the question so that agreeing is enough to record it, when not to ask at all, and what never to say. Only the opening question is asked cold. A repository already named is not asked for again. The four questions come one at a time in a fixed order, and a user who would rather not review them individually is offered, once, the chance to hand the whole set over. Where the files go is decided quietly unless the repository cannot take them. Before anything is written the flow states which repository is being registered, which folder will be created and where its code lives, and waits for explicit agreement. Every instruction carries a standing rule keeping Spektacular's vocabulary, the files it writes and the commands it runs out of what the user is shown.

**Deviations**: None.

**Files changed**:

- `spektacular: templates/steps/repo/01-locate.md` through `09-finished.md` (all nine rewritten)
- `spektacular: templates/vocabulary_containment_test.go` (new)
- `spektacular: templates/guided_add_conversation_test.go` (new)
- `spektacular: internal/steps/repo/probe.go`
- `spektacular: internal/steps/repo/steps_test.go`

**Discoveries**:

- **A list in a template repeats everything inside its section, including the label.** An instruction that printed a labelled line for the languages present printed that label once per language, and the top-level listing did the same. Reading the template does not reveal this; rendering it does. The fix was to supply a ready-joined line alongside the list rather than to contort the template, and any future instruction wanting one labelled line from a list needs the same.
- **Contract tests over prose pass vacuously with alarming ease**, so each was mutation-checked: the property was deliberately broken, the matching test watched to fail, and the file restored. This is worth the effort every time a test asserts on wording rather than behaviour.
- **Three assertions from an earlier phase had to move, and the distinction matters.** They pinned the placeholder wording this phase always intended to replace. The branch selection they exist to protect is unchanged and still asserted; only the sentences they match on moved to the real content. That is a deliberate contract change, not a test being softened to hide a regression, and the difference is worth stating explicitly in any review.
- **The standing vocabulary rule belongs on the closing report too.** The refresh-context directive is deliberately absent from terminal steps, which makes it tempting to treat the closing report as exempt from everything. It is not: it is spoken to the user like any other message and leaks just as easily, so the rule is on all nine.

### 2026-09-07 — Phase 2.5: An agent that drives the flow instead of improvising it

**What was done**: The skill that handles repositories stopped carrying its own account of how to add one and became a driver for the flow: start it, read what it says to do, do it, advance, repeat until it reports itself finished, with the resume-or-start-over branch handled the way the specification and plan skills already handle it. What it says about inspecting the registry, materialization and staleness, repairing a broken registration and removal is unchanged. The text shown when an unfinished workflow is found gained a branch for an add, which has no gathered section files because its answers travel with it.

**Deviations**: None.

**Files changed**:

- `spektacular: templates/skills/workflows/spek-manage-repos/SKILL.md`
- `spektacular: templates/steps/resume.md`
- `spektacular: internal/agent/commands.go`
- `spektacular: templates/skill_resume_test.go`
- `spektacular: templates/guided_add_skill_test.go` (new)
- `spektacular: .claude/skills/spek-manage-repos/SKILL.md`, `.bob/skills/spek-manage-repos/SKILL.md`, `.bob/commands/spek-manage-repos.md` (regenerated by init, not hand-edited)

**Discoveries**:

- **Refreshing the installed copies is two commands in a specific order.** Initialising installs only the agent named, so bringing every install into line takes one run per agent. Initialising also writes the project's own agent and command into its configuration, so the last run wins and the project has to end on the agent it actually uses. Getting that order wrong silently changes the project's configuration while appearing to succeed.
- **An assertion covering every workflow's skill had to be rewritten rather than extended.** It required each skill to describe the per-section working files, with one workflow exempted. A guided add has no such files either, so adding it to the list made the suite fail, correctly. The assertion now names both workflows that have none, which is what the contract always was; the list and the exemption had simply been written as though there would never be a third case.
- **The description rendered into one agent's command wrapper escapes apostrophes into HTML entities**, because it is substituted through an escaping placeholder rather than a raw one. This predates the work here and affects another skill's wrapper too, so it was left alone and is worth a separate fix.

### 2026-09-07 — Phase 3.1: Proving the conversation, not just the result

**What was done**: A containerised end-to-end suite was written for the guided add, in two scenarios: one where the user answers each question in turn, and one where they hand the whole set over after the first proposal. The container seeds a repository that states enough about itself for every proposal to be drawn from it, with a declared name and description that deliberately differ from its folder name. The verifier parses the agent's own transcript and checks the promises that are about the conversation rather than the files. Two build targets were added beside the existing three, and the project's own testing documentation now lists the suite.

**Deviations**:

- **The suite is written and its rules are proven, but it has not been run against a live agent.** Five of this phase's six acceptance criteria are statements about a recorded exchange, and producing one means running a real coding agent inside a container against the user's own credentials and budget. That is an outward-facing action with a real cost, and it is the user's call rather than something to spend on their behalf mid-workflow. Those five criteria are deliberately left unchecked rather than marked complete on the strength of a suite that has never executed. The sixth, that each rule provably fails on an exchange that breaks it, is checked, because that one is verifiable locally and was.

**Files changed**:

- `spektacular: tests/harbor/repo-workflow/task.toml`, `environment/Dockerfile`, `instruction.md`, `scenario-one-at-a-time.md`, `scenario-delegated.md`, `solution/solve.sh`, `tests/test.sh` (new)
- `spektacular: tests/harbor/repo-workflow/tests/test_repo_workflow.py` (new)
- `spektacular: tests/harbor/repo-workflow/tests/test_verifier_selfcheck.py` (new)
- `spektacular: Makefile`
- `spektacular: README.md`

**Discoveries**:

- **The rules had to be written as importable functions so they could be tested without a container.** Putting the logic directly inside the container test cases would have made every rule unverifiable until a live run existed, which is exactly how a rule that can never fail survives review. Separating the predicates from the test wrappers let twenty-seven local checks prove each rule fails on a transcript that breaks it and passes on one that honours it, with no container and no agent involved. Any future transcript-shaped suite should be built the same way round.
- **The vocabulary scan has a boundary that must be stated, not assumed.** Only the agent's own prose to the user can be scanned. Its commands necessarily contain the banned words, and so do the instructions it reads back, so scanning tool inputs or tool results would fail every correct run. Two of the banned words were also dropped from the scan because they are ordinary English and would produce false failures rather than catching leaks.
- **The value a question proposes rides on the call that leaves that question, not the one that enters it.** Correlating a proposal with its recorded answer therefore keys off the field in the payload rather than the step name. A payload carrying several fields at once is not a violation of that rule; it is the delegation shortcut, which a different check covers.
- **The likeliest first-run failure is a true one.** An agent narrating the mechanics to itself in prose would trip the vocabulary scan, and that is the leak the whole feature exists to prevent rather than a defect in the check.

### 2026-09-07 — Phase 4.1: The published documentation describes the guided add

**What was done**: The multi-repo projects page no longer presents registration as a command whose arguments the reader supplies. It now opens by saying that adding a repository is a short conversation in which only the first question is asked cold, shows the exchange as a worked dialogue in the voice the site already uses for the specification interview, explains what the flow decides quietly and that nothing is written before confirmation, notes that an add can run alongside a specification or plan already in progress, and keeps the single-command form as the path scripts should take. The instruction to supply descriptive metadata as command arguments during an interactive add is gone.

**Deviations**: None.

**Files changed**:

- `docs: src/pages/projects.mdx`

**Discoveries**:

- **A section component renders exactly one named summary slot, at the top.** The first draft used a second one partway down to introduce the non-interactive path. That is not a compile error and the type check does not catch it; the content is simply hoisted into the header, silently reordering the page. It only showed up by rendering the built output and reading it. Any second summary paragraph has to be ordinary prose.
- **This page carries no in-page anchor links and its headings are given no identifiers**, so a cross-reference written as an anchor would have built cleanly and resolved to nothing. Naming the neighbouring section in prose is the safe form here until headings get identifiers.
- **The type check and the build both pass on markup the page cannot actually render correctly**, which is worth knowing about the whole verification approach for this repository. Both were clean before the slot problem was found. Reading the generated page is the check that catches this class of fault, and it belongs alongside the build rather than instead of it.

### 2026-09-07 — Phase 4.2: The README tells the same story

**What was done**: The project's own README described registering a repository only as a single command with its arguments. It now describes the guided flow first, one question at a time with each value proposed and a plain-language confirmation before anything is written, notes that an add can run alongside a specification or plan already in progress, and keeps the single-command form for a caller that already knows every detail. A reader arriving at the README or at the published documentation now finds the same account.

**Deviations**: None.

**Files changed**:

- `spektacular: README.md`

**Discoveries**:

- **Replacing the paragraph was not enough on its own.** A later sentence in the same section still named the single command as the way to manage the registry, which quietly contradicted the new opening and would have left the page arguing with itself. Prose changes of this kind need the surrounding sentences read for consequences, not just the target paragraph swapped.
