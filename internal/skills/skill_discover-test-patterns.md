# Discover Test Patterns

Discover and document the project's testing conventions, patterns, and infrastructure. Write findings to `thoughts/notes/testing.md`.

## Instructions

1. Find all test files in the project and identify patterns:
   - File naming conventions (e.g., `*_test.go`, `*.test.ts`, `*.spec.js`)
   - Directory structure (co-located, `__tests__/`, `test/`)
   - Test framework(s) in use

2. For each test pattern found, document:
   - **Framework**: The test framework (e.g., Go testing, Jest, pytest)
   - **File pattern**: How test files are named and located
   - **Example**: A representative test showing the pattern
   - **Run command**: How to run these specific tests

3. Look for:
   - Unit test patterns
   - Integration test patterns
   - Test fixtures / helpers / factories
   - Mocking patterns
   - Test configuration files
   - CI test commands

4. Create `thoughts/notes/testing.md` with the following structure:

```markdown
# Testing Patterns

## Framework
<framework name and version>

## File Conventions
- Pattern: `<glob pattern>`
- Location: <where tests live>

## Unit Tests
<example and conventions>

## Integration Tests
<example and conventions>

## Test Helpers
<shared utilities, fixtures, factories>

## Running Tests
- All: `<command>`
- Single: `<command>`
- Coverage: `<command>`
```

5. If `thoughts/notes/` directory doesn't exist, create it.
