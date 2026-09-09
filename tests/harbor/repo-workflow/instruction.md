# Add a repository to a Spektacular project

You are testing the `spektacular` CLI tool by adding a repository to a project.
The binary is already installed at `/usr/local/bin/spektacular`.

## Setup

First initialize the project:

```bash
spektacular init {{agent}}
```

## Task

There is a repository on this machine at `/srv/harbour`. Add it to this project
by using the `spek-manage-repos` skill that was installed during init.

Run the skill:

```
{{skill_invocation}}
```

The skill will start a guided add and hand you one instruction at a time.
Follow each instruction exactly as it is written. Do not improvise your own
version of the flow, and do not run `spektacular repo add` directly: the point
of this exercise is the guided conversation, not the registration.

## You are running non-interactively

There is no human at the other end of this session. You are playing both roles:
you drive the workflow **and** you answer as the user would.

Keep the two roles clearly separated in your messages. When you are speaking as
the workflow, write the message you would send to a user. When you are answering
as the user, answer as a person would: briefly, and without knowing anything
about how Spektacular works internally.

**Never end your turn waiting for a reply.** Nothing will answer, and the run
will stall. Only stop once the workflow itself reports it has finished.

{{scenario}}

## After completion

Copy the `.spektacular` directory to `/logs/artifacts/` so results are collected:

```bash
cp -r /app/.spektacular /logs/artifacts/spektacular
```

### Success criteria

- The workflow reaches the `finished` state, with every step in `completed_steps`
- `spektacular repo list` reports the new repository with a non-empty description, role and tags, and no missing-metadata warning
- The repository was registered through the guided flow, not through `spektacular repo add`
