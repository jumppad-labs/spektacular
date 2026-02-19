

Great choice. Textual gives us the best combination of streaming markdown rendering, scrollable views, and interactive widgets -- all in one package.

The plan is already generated with Textual as the recommended approach. All three deliverables are ready in `.spektacular/plans/2_rich_tui/`:

- **`plan.md`** - 4-phase implementation plan with full code examples
- **`research.md`** - Library comparison, architecture analysis, design decisions
- **`context.md`** - Quick reference for key files and integration points

The plan uses Textual's `@work(thread=True)` to run the blocking Claude subprocess without freezing the UI, custom `Message` classes (`AgentOutput`, `AgentQuestion`, `AgentComplete`) for thread-safe communication, a `VerticalScroll` container with `Markdown` widget for streaming output, and `QuestionPanel` widgets with clickable buttons for interactive Q&A.

Would you like me to proceed with implementation, or do you want to review/adjust the plan first?