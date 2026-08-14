# Passing tests are required before calling work done

Before reporting any change to this repo as complete, run `go test ./...`
and confirm it passes in full. A change is not done while any test fails,
including a test that was already failing before the change started —
either fix the test or the code it exercises, or explicitly flag the
failure to the user and get their agreement before treating the work as
finished.

Do not dismiss a failure as "pre-existing" and move on silently: a failing
test left unaddressed becomes the next person's surprise. Confirm whether
the failure is caused by the current change or predates it, but either way
raise it and resolve it (fix, or explicit user sign-off to defer) before
declaring done.
