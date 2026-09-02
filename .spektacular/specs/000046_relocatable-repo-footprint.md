---
created_date: "2026-09-02"
status: completed
closed_date: "2026-09-02"
---

# Feature: 000046_relocatable-repo-footprint

## Overview

Spektacular can now keep everything it knows about a repo — its configuration, knowledge base, and changelog — separately from the code it describes, instead of only colocated with it. Teams that cannot, or prefer not to, commit Spektacular files into a code repository can run everything from a project folder that holds one folder per repo and points each at its source wherever it is: a directory on disk or a git repository to clone. That project folder can itself be committed and shared, while the code repositories stay untouched.

## Requirements

- [x] **Users can declare where a repo's code lives**
  A repo's configuration can name its source with a single `source` setting. The source is either a file location (`file://` scheme or a plain path; absolute, relative to the directory holding the repo configuration, or containing environment variable references) or a git location (a git URL such as `git://`, `ssh://`, `https://`, or an scp-style address). A git source is cloned into Spektacular's working folder inside the project the first time the repo is used, and reused thereafter.

- [x] **Registry entries locate only the repo's own folder**
  A project's registry entry carries the repo's `name` and a `location`: the folder holding the repo's configuration. The previous `local` key is still accepted as an alias for `location`. The previous `address` key is no longer accepted; a configuration that still uses it fails with an error that says to declare the git `source` in the repo's own configuration instead.

- [x] **Knowledge and changelog locations come from the repo configuration**
  The repo configuration's knowledge and changelog locations are used regardless of where the source is, so Spektacular's files land where the repo configuration says, not in the source.

- [x] **Only the project and per-repo changelogs**
  A feature produces a project changelog at the location the project configuration defines and one repo changelog per affected repo at the location that repo's configuration defines. No other changelog file is written.

- [x] **Repo list root reflects the source**
  In the repo list output, a repo's root is the resolved source when one is specified (the file location, or the clone of a git location); otherwise the root is calculated as it is today.

- [x] **Templates tell the agent to use repo source locations**
  Workflow steps and skills that direct the agent to a repo's code tell it to use that repo's source location.

- [x] **Repo source locations are provided to the agent at run time**
  The source location of every repo the agent may work in is provided to it when the workflow runs, rather than being assumed or discovered by the agent itself.

- [x] **Documentation is updated**
  The documentation site, the in-repo documentation, and the repo-management skill describe the `source` and `location` settings and show that a repo's Spektacular files can be colocated with its code or kept separately from it, for both file and git sources.

## Constraints

- No constraints beyond the requirements; the detailed design is left to the plan.

## Acceptance Criteria

- [x] **Source is resolved in every accepted form**
  Given a repo configuration whose source is an absolute file path, a path relative to the repo configuration's directory, a path containing an environment variable reference, or a `file://` form of any of these, the repo list output reports the resolved absolute path as that repo's root in every case. Given a git source, using the repo clones it into Spektacular's working folder and the repo list output reports the clone as the root.

- [x] **Registry keys are `name` and `location`**
  A registry entry with `location` resolves to the folder holding the repo configuration; an entry with the older `local` key resolves identically; an entry with `address` fails configuration loading with an error naming the repo and telling the user to set `source` in that repo's configuration.

- [x] **Spektacular files land where the repo configuration says**
  After registering a repo whose source is a separate directory, writing a knowledge entry for it, and running an implement workflow to completion against it, the knowledge entry and the repo's changelog exist at the locations the repo configuration declares, and the source directory contains only the code changes.

- [x] **Only the project and per-repo changelogs are written**
  After an implement workflow completes, a project changelog exists at the project's configured location and one repo changelog exists per affected repo at that repo's configured location, and no other changelog file has been created or modified anywhere, including at the root of any source directory.

- [x] **Repo list root follows source**
  For a repo with a source set, the repo list output reports that source as the root; for a repo without one, the root is identical to what the current release reports.

- [x] **Templates direct the agent to the source**
  Every rendered workflow step and skill that directs the agent to a repo's code tells it to use that repo's source location; none tells it to work in the current or project directory as a stand-in.

- [x] **Sources are provided at run time**
  Running a plan or implement workflow against a project with more than one repo, at least one of which has a separate source, gives the agent every repo's source location during the run, and the code changes land in each repo's source.

- [x] **Documentation covers source and colocation**
  The documentation site and the in-repo documentation document the `source` and `location` settings, and both they and the repo-management skill show a repo colocated with its Spektacular files, a repo kept separate from them with a file source, and a repo kept separate from them with a git source.

## Technical Approach

- Prefer having repo resolution apply the source once, so every consumer that needs the code (git operations, the repo list output) sees the source as the root, while consumers of the repo's own files keep using the repo configuration's directory.
- Prefer registration as the writer of the source setting, in the same way it already writes the other descriptive fields into the repo configuration.
- Consider a test that renders the workflow templates and checks for the current assumptions that the project's own repo is "the directory you are running in" and that a release note goes to a root changelog in the code, rather than relying on a manual pass.

## Success Metrics

- A user can set up a project folder that describes an external code repository, run a plan and an implement workflow end to end, and finish with a clean `git status` in the code repository apart from the intended code changes.
- An agent following the repo-management skill registers a repo with a separate source correctly on the first attempt, without the user explaining the colocated versus separate layouts.

## Non-Goals

- Moving a repo's descriptive metadata, knowledge, or changelog into the project configuration; considered and rejected.
- Cloning a git source anywhere other than Spektacular's own working folder inside the project. The clone location is not user-configurable.
- Migrating existing registry entries that use `address`. Nothing has shipped with it; users move the value into the repo configuration's `source` by hand.
- More than one source per repo, or a file source that is not a git working tree.
- Migrating existing colocated repos to a separate layout. Users who want that move the files themselves and set the source.
