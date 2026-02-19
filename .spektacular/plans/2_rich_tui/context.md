# Rich TUI - Context

## Quick Summary
Replace the plain `click.echo()` output in plan mode with a full-screen Textual TUI that renders streaming markdown, supports scrollable navigation, and provides interactive question/answer widgets.

## Key Files & Locations
- **New TUI Module**: `src/spektacular/tui.py` (to be created)
- **Primary Integration**: `src/spektacular/plan.py:64-113` (run_plan function)
- **Event Source**: `src/spektacular/runner.py:94-146` (run_claude generator)
- **CLI Entry Point**: `src/spektacular/cli.py:62-77` (plan command)
- **Configuration**: `src/spektacular/config.py:47-61` (AgentConfig)
- **Tests**: `tests/test_tui.py` (to be created)

## Dependencies
- **Code Dependencies**: `runner.py` (ClaudeEvent, run_claude, detect_questions), `config.py` (SpektacularConfig)
- **New External Dependencies**: `textual>=1.0.0` (includes Rich as transitive dependency)
- **Database Changes**: None
- **Existing Dependencies to Keep**: click (still used for CLI routing, not for TUI output)

## Environment Requirements
- **Configuration Variables**: None new
- **Migration Scripts**: None
- **Feature Flags**: None (TUI replaces plain output entirely in plan command)

## Integration Points
- **CLI Command**: `plan` command in `cli.py` launches TUI app instead of calling `run_plan()` directly
- **Event Stream**: `run_claude()` generator feeds events to TUI via Textual workers
- **Question Detection**: `detect_questions()` from `runner.py` used to trigger interactive Q&A widgets
- **Plan Output**: `write_plan_output()` from `plan.py` writes final plan.md (unchanged)

## Architecture Notes
- Textual app runs as a full-screen terminal application
- Claude subprocess runs in a Textual worker thread (non-blocking)
- Events are posted as Textual messages from worker to UI thread
- Markdown widget uses `append()` for efficient incremental streaming
- Question widgets overlay or replace the input area when questions are detected
- The app exits cleanly when the plan generation completes or user quits
