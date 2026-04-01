# Spawn Planning Agents

Orchestrate parallel research agents to investigate the codebase for planning purposes.

## Instructions

Use your agent orchestration capability to run multiple research tasks in parallel. Launch the following agents concurrently:

### Agent 1: File Discovery
Find all files related to the feature being planned. Organize results by category:
- **Implementation files**: Source code that will be modified
- **Test files**: Existing tests for the affected code
- **Config files**: Configuration that may need changes
- **Documentation**: Existing docs about the relevant systems

### Agent 2: Prior Research
Search for existing research and plans related to this feature:
- Check `thoughts/` directory for related notes, plans, or research docs
- Check `.spektacular/plans/` for related plans
- Check `.spektacular/specs/` for related specs
- Look for relevant issues, tickets, or TODOs in the codebase

### Agent 3: Similar Implementations
Find code examples that are similar to what needs to be built:
- Search for analogous patterns in the codebase
- Find existing implementations that can be modelled after
- Note file:line references for all findings

### Agent 4: Architecture Analysis
Understand how the relevant components fit together:
- Identify the dependency graph for affected modules
- Map integration points between components
- Note any shared state or cross-cutting concerns

## Output

Each agent should return structured findings. Combine all results and use them to inform your planning decisions.
