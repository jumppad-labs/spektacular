## Discovery Grounding

> Managed by `{{command}} init` — edit `templates/agents/discovery-grounding.md`
> in the Spektacular source, not this section in place. Hand edits will not
> survive the next init.

When investigating this codebase — answering "how does X work" or "why does
Y look like this" outside of a structured spec/plan/implement workflow —
ground your discovery in the current code and conversation context, not in
specs or plans. Source code is the source of truth for what the system
*does*; a spec or plan can be stale, partially implemented, or superseded,
so treating one as ground truth risks investigating a fiction instead of
the actual system. If a spec or plan is genuinely relevant to a question
the code can't answer (an intentional but non-obvious constraint or
tradeoff), it's fine to read it — but verify what it claims against the
code before relying on it.

The `.spektacular/` directory is not part of the codebase for this purpose.
It holds generated artifacts *about* the codebase — specs, plans, knowledge
entries, context, changelogs — not the system itself. Don't sweep it in
when asked to search or read "the codebase"; a broad grep or file scan
should treat `.spektacular/` as out of scope unless the task explicitly
concerns specs, plans, or knowledge.

This does not apply to the spec/plan/implement workflows themselves, which
read `.spektacular/` deliberately and by design (e.g. `spek-implement`
reading the approved plan, or a discovery step's own prior-research lookup)
— those steps already say when to consult it.
