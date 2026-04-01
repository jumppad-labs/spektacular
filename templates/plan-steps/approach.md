## Step {{step}}: {{title}}

Based on the research findings for: **{{overview}}**

Present 2-3 design options with:
- **Option name** and brief description
- **Pros**: Advantages, with file:line references from research
- **Cons**: Disadvantages, risks, complexity
- **Effort estimate**: Relative complexity (Low / Medium / High)

For each option, reference specific code from the discovery phase to ground the analysis.

Get the user's agreement on:
1. **Chosen direction** — Which option to pursue
2. **Out-of-scope items** — What we're explicitly NOT doing in this plan

Once the user agrees on an approach, advance:

{{config.command}} plan goto --data '{"step":"{{next_step}}"}'
