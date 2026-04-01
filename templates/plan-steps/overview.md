## Step {{step}}: {{title}}

{{#overview}}
The following overview has been provided for this plan:

> {{overview}}

Does this accurately describe what we're planning? Are there any clarifications or refinements needed?

If the overview needs changes, discuss with the user and refine it. Once the overview is accurate, advance to the next step:

{{config.command}} plan goto --data '{"step":"{{next_step}}", "overview":"<refined overview here>"}'
{{/overview}}
{{^overview}}
Ask the user to describe what they want to plan in 2-3 sentences:
• What are we building or changing?
• What problem does it solve?
• What's the desired outcome?

Once you have a clear overview, advance to the next step:

{{config.command}} plan goto --data '{"step":"{{next_step}}", "overview":"<overview text here>"}'
{{/overview}}
