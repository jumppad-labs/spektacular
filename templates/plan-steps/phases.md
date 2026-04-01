## Step {{step}}: {{title}}

For each milestone, define implementation phases for: **{{overview}}**

Each phase must include:

- **Complexity**: Low / Medium / High
- **Token estimate**: ~Nk tokens (rough estimate for agent context usage)
- **Agent strategy**: How to break the work down
  - Low: Single agent, sequential execution
  - Medium: 2-3 parallel agents for independent changes
  - High: Parallel analysis, sequential integration
- **File changes**: Specific file:line changes based on research findings
- **Success criteria**:
  - **Automated**: Commands from `thoughts/notes/commands.md` (e.g., `go build ./...`, `npm test`)
  - **Manual**: Concrete verification steps a human can perform

For guidance on agent orchestration strategy: `{{config.command}} skill spawn-implementation-agents`

Rules:
- Every file change must reference a specific file (and line range where applicable)
- NO open questions — resolve any uncertainties now
- Success criteria must be split into automated and manual
- Automated criteria should use actual project commands

Present the phases to the user for review. Once agreed, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
