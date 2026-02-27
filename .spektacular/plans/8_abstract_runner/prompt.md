Additional project knowledge, architectural context, and past learnings can be found in `.spektacular/knowledge/`. Use your available tools to explore this directory as needed.

---

# Specification to Plan

# Feature: 8 Abstract Runner

## Overview
Spektacular should support multiple tools like Claude, Bob, OpenAI, etc. To
acheive this we need to define a common interface for all the tools to implement,
and abstract the current claude implementation into a it's own runner.

## Requirements
- [ ] Define a common interface for all runners to implement
- [ ] Abstract the current claude implementation into a separate runner that implements the common interface

## Constraints

## Acceptance Criteria
- [ ] A common interface for runners is defined and documented
- [ ] A claude runner is implemented that adheres to the common interface and can be used in place of the current implementation
- [ ] All tests pass

## Technical Approach

## Success Metrics

## Non-Goals
