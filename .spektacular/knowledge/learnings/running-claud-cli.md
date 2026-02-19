# Loading the agent

```bash
claude -p "Load and understand the planner agent specification from src/spektacular/defaults/agents/planner.md" --output-format stream-json --verbose
```

Outputs json which contains a session id: c102b04d-1a16-473e-ae3f-e8a5b5b8d87e

# Executing a plan
```bash
claude -p "Now use the planner agent workflow to process .spektacular/specs/1_plan_mode.md and create implementation plans" --resume <session-id> --output-format stream-json --verbose --allowedTools "Bash,Read,Write,Edit" --dangerously-skip-permissions
```