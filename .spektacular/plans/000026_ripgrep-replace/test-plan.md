# Test Plan: 000026_ripgrep-replace

All success metrics are covered by automated behavioural tests; no manual test plan is required.

Both spec success metrics are verified automatically:

1. *Zero external runtime dependencies for search* — behavioural via the in-process unit
   and CLI test suite, plus structural: `internal/store` no longer imports `os/exec`
   (`grep -rn "os/exec" internal/` returns nothing).
2. *No regression in search results* — behavioural via the unchanged backend-independent
   tests; the two retired rg-conditional tests' guarantees are re-homed in
   `TestSearch_CaseInsensitiveAndExcludesConventions` (case-insensitive matching,
   conventions exclusion, empty-query semantics) and `TestSearch_ScopeAndLocatorRoundTrip`
   (no-match semantics), as traced in plan.md § Changelog.
