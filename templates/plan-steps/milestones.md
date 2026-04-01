## Step {{step}}: {{title}}

Define 2-4 milestones for: **{{overview}}**

Each milestone must have:
- **User-facing goal**: What the user can do or see when this milestone is complete
- **Testable outcomes**: Specific, verifiable results (commands to run, behaviours to observe)
- **Validation point**: How to confirm the milestone is done before moving on

Rules:
- Each milestone should be independently deliverable
- Milestones should build on each other in order
- NO open questions — resolve any uncertainties now by asking the user
- Keep milestones focused on outcomes, not implementation details

Present the milestones to the user for review. Once agreed, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
