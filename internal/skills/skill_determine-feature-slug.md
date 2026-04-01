# Determine Feature Slug

Determine the feature slug (namespace + number) for a plan document.

## Instructions

1. **Auto-detect the next number**: Look at existing plan files in `.spektacular/plans/` and `thoughts/plans/` to find the highest existing number. Increment by 1.

2. **Format**: `NNNN-description` where:
   - `NNNN` is a zero-padded 4-digit number (e.g., `0001`, `0042`)
   - `description` is a kebab-case slug derived from the plan name

3. **Confirm with user**: Present the proposed slug via `AskUserQuestion` and let them modify it if needed.

## Example

If the highest existing plan number is `0003`:
- Proposed slug: `0004-add-health-endpoint`
- Ask: "The next feature slug would be `0004-add-health-endpoint`. Is this correct, or would you like to change it?"

## Commands

```bash
# Find existing plan numbers
ls .spektacular/plans/ thoughts/plans/ 2>/dev/null | grep -oP '^\d+' | sort -n | tail -1
```
