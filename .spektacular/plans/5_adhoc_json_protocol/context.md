# Adhoc JSON Protocol - Context

## Quick Summary
Add `spektacular adhoc` command with dual output modes: CLI (human-readable terminal output) and JSON (bidirectional JSONL protocol over stdin/stdout for machine orchestration by OpenClaw and other tools).

## Key Files & Locations

### New Files
- **Protocol Types**: `internal/protocol/types.go` — Message envelope and event payloads
- **Protocol IDs**: `internal/protocol/ids.go` — Run/message ID generation
- **Protocol Writer**: `internal/protocol/writer.go` — JSONL stdout writer
- **Protocol Reader**: `internal/protocol/reader.go` — JSONL stdin reader
- **Adhoc Engine**: `internal/adhoc/adhoc.go` — Execution engine bridging runner to protocol
- **CLI Command**: `cmd/adhoc.go` — Cobra command with `--output` flag

### Modified Files
- **Root Command**: `cmd/root.go:29` — Register `adhocCmd`
- **Config**: `internal/config/config.go:45-48` — Add `AdhocOutput` field

### Key Existing Files
- **Runner**: `internal/runner/runner.go` — Claude subprocess spawner (reused by adhoc engine)
- **Plan Loop**: `internal/plan/plan.go:101-148` — Reference pattern for event processing
- **Config**: `internal/config/config.go` — Configuration types and loading

## Dependencies

### Code Dependencies
- `internal/runner` — Claude subprocess management, `ClaudeEvent`, `DetectQuestions()`
- `internal/config` — Configuration loading and defaults
- `internal/protocol` — New package for JSONL protocol types and I/O

### External Dependencies
- `github.com/spf13/cobra` — CLI framework (existing)
- `encoding/json` — JSON marshal/unmarshal (stdlib)
- `crypto/rand` — ID generation (stdlib)
- No new external dependencies required

### Database Changes
None.

## Environment Requirements

### Configuration Variables
- `output.adhoc_output` in `.spektacular/config.yaml` — Persisted default output mode (`cli` or `json`)

### Migration Scripts
None.

### Feature Flags
None — output mode controlled by `--output` flag or config.

## Integration Points

### Inbound Protocol (stdin)
| Event Type | Purpose |
|------------|---------|
| `run.start` | Initiate adhoc execution with prompt |
| `run.input` | Provide answer to `run.question` |
| `run.cancel` | Request cancellation of current run |

### Outbound Protocol (stdout)
| Event Type | Purpose |
|------------|---------|
| `run.started` | Acknowledge start, report provider/model |
| `run.progress` | Stream text, tool use, status updates |
| `run.question` | Normalized interactive prompt |
| `run.artifact` | Generated file/plan metadata |
| `run.completed` | Terminal: successful completion |
| `run.failed` | Terminal: error with code and message |
| `run.cancelled` | Terminal: clean cancellation |

### External Services
- Claude CLI (`claude` command) — spawned as subprocess by existing runner

### Message Queues
None — direct stdin/stdout communication.
