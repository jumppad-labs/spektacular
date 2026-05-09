## spec-id-prefix-method

Spec creation now uses CLI-owned identifier prefixes. By default, `spec new` creates timestamp-prefixed spec names, accepts an optional `id` for external systems, and can be configured with `spec.id_method: timestamp`, `counter`, or `external`. Callers should use the returned `spec_name` and `spec_path` because requested names and ids are normalized before workflow state and files are created.

## 19_codex_integration

Spektacular now ships first-class support for OpenAI Codex alongside Claude Code and Bob. `spektacular init codex` installs the three workflow entry points — spec, plan, and implement — as native SKILL.md files, so Codex users can invoke `$spek-new`, `$spek-plan`, and `$spek-implement` out of the box. As part of this release the workflow entry points become the canonical artefact type across every supported agent: slash commands for Claude and Bob are now thin wrappers that simply invoke the matching skill, and workflow instructions live in one place rather than being copied per-agent. Existing Claude and Bob users should re-run `spektacular init <agent>` once to pick up the new layout.

## 18_release_workflow

Spektacular now has a complete automated release pipeline powered by Dagger and GitHub Actions. Every push to any branch builds cross-platform binaries (darwin/linux on amd64/arm64) with automatic macOS code signing and notarization. Merging to `main` with a `release/patch|minor|major` PR label automatically publishes a new release to GitHub Releases, updates the Homebrew formula in `jumppad-labs/homebrew-repo`, and pushes packages to GemFury. Users can now install spektacular via `brew install jumppad-labs/repo/spektacular` or download signed binaries directly from GitHub Releases, eliminating the need to clone and build from source.
