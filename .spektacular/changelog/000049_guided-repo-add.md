---
created_date: "2026-09-07"
status: completed
closed_date: "2026-09-07"
---

# Guided repo add

## What was built

Adding a repository to a project is now a guided conversation rather than a form to fill in.

You are asked one thing, which repository you want to add, and nothing else is asked cold.
Spektacular reads what that repository says about itself, its README, whatever manifest its
language uses, its top-level layout and the languages present, and then asks about its name, its
description, its role and its tags one at a time, each question already carrying a concrete
proposal drawn from what it read. Agreeing is enough to record a value. A user who would rather
not review the proposals individually is offered, once, the chance to hand the whole set over,
and is then asked nothing further about them.

Where Spektacular keeps its own files for the repository is settled quietly, inside the
repository being added, and is raised only when that repository cannot take them or the user has
said it should hold nothing but code, in which case the files live in a folder under the project
and the repository is left untouched. Before anything is written, the flow states in plain terms
which repository is being registered, which folder will be created and where its code lives, and
waits for explicit agreement. Nothing at all is written before that point, so an add abandoned
partway through leaves nothing to clean up.

The flow is a workflow the CLI owns, with progress recorded separately from the one that
specifications, plans and implementations share. An interrupted add resumes exactly where it
stopped with every answer already given still in hand, and an add can be started and finished
while a specification or plan is midway through, with neither disturbing or blocking the other.

Registering a repository in a single command, for a caller that already knows every detail,
works exactly as it did before.

## Why it matters

It fixes two faults in what came before. Users were interrogated for information the tool could
simply have read from the repository sitting in front of it. And Spektacular's internal
vocabulary, the words it uses for its own data model, the files it writes and the commands it
runs, kept surfacing in what it said to them. A user adding their repository never asked to learn
any of it.

The second fault was the harder one, because it was never a set of bad sentences to delete. The
guidance the agent read taught it a vocabulary, and the agent then reproduced that vocabulary in
its own words. The fix is structural: the conversation now lives in instructions the CLI renders
one at a time, every one of which carries a standing rule about what belongs to the tool and what
belongs to the user, and that rule is enforced by tests at two levels rather than trusted.

## Deviations from the plan

- **Nine step instructions were written, not the ten the plan listed.** The plan asked for an
  opening instruction for the internal first step while also specifying that step asks the user
  nothing. Both cannot hold, since a step that renders an instruction stops there and the user
  sees it. The planning workflow had already settled this precedent, and the guided add follows
  it, so starting an add renders the first real question rather than a preamble.
- **The shared registration function needed a parameter the plan's contract omitted**, without
  which the existing tests could not substitute their stand-in for fetching remote repositories
  and would have attempted real clones.
- **A built-in list of dependency and build directories was added to the examination.** The plan
  said to honour a repository's own exclusion file so that vendored directories do not dominate
  the languages reported. Tested against a real repository carrying no such file, they dominated
  anyway, which fails the criterion outright, so a fixed set of well-known directory names is now
  skipped as well.
- **A parent-level dry-run option had to be added** to the repository command group, which did
  not have one; without it a dry run would silently have written real state.
- **The end-to-end suite has not been run against a live agent.** It is written, and its rules are
  proven to fail on transcripts that break them, but producing a real recorded exchange means
  running a coding agent inside a container against real credentials and budget. Five of that
  phase's acceptance criteria are deliberately left unchecked rather than claimed on the strength
  of a suite that has never executed.

## Known gaps

- A registration that fails after the project configuration has been written leaves an entry for
  a repository whose folder does not exist. This is the direct command's existing behaviour,
  confirmed unchanged by this work rather than introduced by it, and is left for a separate fix
  because correcting it would alter behaviour this feature was required to keep identical.
- The debug session log does not record a guided add, because it reads only the state the other
  three workflows share. Nothing in the specification asked for it.
