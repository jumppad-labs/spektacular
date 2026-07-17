# Test Plan: 000036_spec_plan_historical_artifacts

All three of the spec's success metrics are runtime-behavioural observations of how agents cite sources during real user sessions. They cannot be asserted from within the Go test suite, so the plan's Testing Approach classified every one as **Manual — captured in the implementation test plan**. The procedures below are grounded in the actual shipped surface: the new `## Historical Artifacts: Specs and Plans as Archaeology` section in `AGENTS.md` (injected by `spektacular init`), plus the two step-template clarifiers in `templates/steps/plan/02-discovery.md` and `templates/steps/implement/01-read_plan.md`.

## Pre-condition for all three procedures

Before running any of the procedures below, confirm the new rule is live in the target repository:

1. Re-run `go run . init` (or, in a downstream consuming repo, `spektacular init`) inside the repo you will test in.
2. Read `AGENTS.md` at the repo root and confirm it now contains a `## Historical Artifacts: Specs and Plans as Archaeology` section with the archaeology and owning-workflow exceptions stated in prose.
3. Confirm `CLAUDE.md` at the repo root contains `@AGENTS.md` (so a fresh Claude Code session actually loads the rule).

If any of these three checks fails, the manual procedures below cannot honestly pass — fix the setup first.

## Metric 1 — Users stop needing to remind agents mid-session that specs and plans are not current-state documentation

**What to measure**: Whether a fresh agent session already knows the rule without any user prompt.

**How**:

1. Start a **new** Claude Code session in a Spektacular-initialized repo whose `AGENTS.md` carries the new section (per pre-condition above).
2. Ask the agent, verbatim: *"What's the rule in this repo about reading files under `.spektacular/specs/` and `.spektacular/plans/`?"*
3. Read the answer.

**Expected result**: The agent's first response accurately paraphrases the rule — specs and plans are historical archaeology, not current-state docs; general discovery reads are avoided; the archaeology and owning-workflow exceptions are both mentioned. The user does not have to prompt it, restate the rule, or correct it. Pass if the answer covers the historicity framing plus at least one exception; fail if the agent says "I don't know" or claims specs/plans are authoritative descriptions of current behaviour.

**Who / when**: Run by a reviewer once per release that touches instruction surfaces in `AGENTS.md` or the step templates. If this metric fails, tighten the wording in `templates/agents/historical-artifacts.md` and re-run.

## Metric 2 — When agents describe existing features, their citations point to source code, tests, or configuration — not to files under `.spektacular/specs/` or `.spektacular/plans/`

**What to measure**: Whether current-state answers cite code artifacts, not spec/plan documents.

**How**:

1. Start a **new** Claude Code session in a Spektacular-initialized repo.
2. Ask the agent to explain how a real, shipped feature currently works — e.g. *"How does the `Memory & Context` section get installed into `AGENTS.md`?"* or *"How does the plan-workflow's discovery step decide which knowledge to load?"*
3. Inspect the agent's tool-call log (or the citations in its answer).

**Expected result**: Every path the agent cites in its answer points somewhere under source directories (e.g. `internal/agent/`, `templates/steps/`, `templates/agents/`), test directories, or configuration (`.spektacular/config.yaml`, `AGENTS.md`, `CLAUDE.md`). No file under `.spektacular/specs/` or `.spektacular/plans/` is cited, and no `spec file read` / `plan file read` CLI call appears in the tool-call log. Pass if all citations are code/tests/config; fail if any spec or plan file is cited as authoritative for current behaviour.

**Who / when**: Run by a reviewer against the shipped rule at least once per release that touches `AGENTS.md`, `templates/agents/`, or the two clarified step templates. If this metric fails, add examples to `templates/agents/historical-artifacts.md` that explicitly distinguish "cite code" from "cite spec/plan" and re-run.

## Metric 3 — When users ask historical or intent questions, agents correctly reach for the relevant spec or plan and cite it as historical context

**What to measure**: Whether historical/intent questions successfully unlock spec/plan reads, framed as historical context rather than as current behaviour.

**How**:

1. Start a **new** Claude Code session in a Spektacular-initialized repo.
2. Ask the agent an explicitly historical question — e.g. *"Why was the plan-workflow discovery step originally introduced?"* or *"What was the original intent behind the spec at `.spektacular/specs/000034_spec-plan-implement-reconciliation.md`?"*
3. Observe whether the agent reads the relevant spec/plan and how it frames the citation.

**Expected result**: The agent reads the relevant historical document — via `Read` on the path, `spec file read`, or `plan file read` — and its answer frames the content as historical context ("the spec proposed…", "at the time the plan was written the author intended…"). The answer does not present the historical intent as an authoritative description of current behaviour. Pass if the archaeology read happens AND the framing is explicitly historical; fail if the agent refuses to read the document even for a clearly historical question, or reads it but frames the content as current-state description.

**Who / when**: Run by a reviewer at least once per release that touches the archaeology-exception wording in `templates/agents/historical-artifacts.md`. If this metric fails, tighten the exception's phrasing so it more clearly signals that historical intent questions are exactly the class this rule permits reads for.

## Deferred non-metric acceptance check

Spec acceptance criterion #9 — the per-feature changelog entry contains the "process document, not product document" website-documentation follow-up note — is verified by directly reading the changelog artifact produced by the next step (`update_feature_changelog`) rather than as a runtime-behavioural procedure. It is called out here only so a reviewer running through this file does not miss it.
