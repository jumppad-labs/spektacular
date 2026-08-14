---
step: new
next: interview
---

The spec scaffold has been created at `{{spec_path}}`.

Before proceeding to the overview step, write the current conversation context to `.spektacular/context.md` if meaningful context exists. Capture the full discussion in detail:

- **What problem was identified and why it needs solving** — the user's motivation, the gap or pain point that prompted this spec
- **All requirements and constraints discussed** — functional requirements, non-functional requirements, technical constraints, business constraints
- **Alternatives considered and why rejected** — other approaches discussed, trade-offs evaluated, reasons for choosing the current direction
- **The user's exact phrasing for key requirements** — preserve the user's language for critical requirements to avoid misinterpretation

If no meaningful context exists (e.g., the user simply said "create a spec for X" without elaboration), leave context.md empty.

Once context is written (or confirmed empty), proceed to the interview step:

```
{{command}} spec goto --data '{"step":"interview"}'
```
