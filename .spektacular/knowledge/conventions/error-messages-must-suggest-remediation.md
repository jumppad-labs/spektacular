# Error messages must describe the problem and suggest remediation

Every error this CLI returns must do two things: state what went wrong, and tell the driving agent exactly what to do about it. A message that only names the problem ("--data is required") without a concrete next step leaves the agent guessing, and a guessing agent tends to improvise a workaround instead of retrying correctly.

Concretely: build errors via `output.NewError(code, message).WithNextAction(<concrete corrective command>)`, never a bare `fmt.Errorf` string with no next_action. `message` names the problem; `next_action` gives an exact, runnable next step — the correct `--data` shape, which `file list` subcommand discovers valid names, which command resumes vs. force-restarts, etc. This applies to every error path in the CLI, not just a specific command family.

Reason: a bare, generic "--data is required" error — and, separately, an unrecognized subcommand silently falling through to cobra's plain-text help with exit 0 instead of a proper error — once caused a driving agent (Bob) to misread these as valid/ambiguous output and bypass the CLI with raw filesystem tools instead of retrying with the correct command.
