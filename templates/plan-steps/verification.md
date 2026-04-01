## Step {{step}}: {{title}}

Time to compile the complete plan document for: **{{overview}}**

### Step 1: Gather Metadata

Use the `gather-project-metadata` skill to collect: ISO timestamp, git commit, branch, and repository info.
For skill details: `{{config.command}} skill gather-project-metadata`

### Step 2: Determine Feature Slug

Use the `determine-feature-slug` skill to determine the plan file namespace and number.
For skill details: `{{config.command}} skill determine-feature-slug`

### Step 3: Fill in the Plan Scaffold

Here is the plan scaffold template. Fill in ALL sections — no placeholders, no open questions:

```markdown
{{plan_template}}
```

### Step 4: Review

Verify the completed plan for:
- **Completeness** — all sections are filled with real content
- **Specificity** — file:line references where applicable
- **Success criteria split** — every phase has both automated and manual criteria
- **No open questions** — everything is resolved

### Step 5: Submit

Once you are confident the plan is complete and correct, pipe the filled plan back:

cat <<'EOF' | {{config.command}} plan goto --data '{"step":"{{next_step}}"}' --stdin plan_template
<complete filled plan here>
EOF
