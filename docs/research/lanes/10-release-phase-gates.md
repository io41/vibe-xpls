# Release And Phase-Gate Research

## Summary

`vibe-xpls` should stay on `v0.X.X` and use release tooling that separates changelog quality from binary publishing. The active stack is Conventional Commits, Release Please, GoReleaser, and a small explicit version guard.

Release Please owns automated release PRs and GitHub releases from commit history. Changie remains an option if maintainers later want explicit change fragments before release.

## Sources

- Release Please GitHub Action: https://github.com/marketplace/actions/release-please-action
- git-cliff documentation: https://git-cliff.org/docs/
- Changie guide: https://changie.dev/guide/
- GoReleaser releases: https://www.goreleaser.com/customization/release/
- GoReleaser release command: https://goreleaser.com/cmd/goreleaser_release/
- Conventional Commits: https://www.conventionalcommits.org/en/v1.0.0/
- Semantic Versioning: https://semver.org/

## Tool Matrix

| Tool | Strength | Risk | Fit |
| --- | --- | --- | --- |
| Conventional Commits | Machine-readable commit intent, supports automated changelog and version decisions | Requires contributor discipline or squash cleanup | Use immediately |
| `git-cliff` | Generates `CHANGELOG.md` from Git history with configurable parsers/templates | Changelog quality depends on commit messages | Reference option, not active automation |
| Changie | Change fragments make release notes explicit and reviewable | More process overhead for every user-facing change | Consider after public contributors appear |
| Release Please | Automates changelog generation, release PRs, GitHub releases, and version bumps | Can over-automate version decisions before API stability | Active automation |
| GoReleaser | Builds/publishes Go artifacts and can generate release changelogs | Publishing config adds supply-chain risk | Active binary publishing |
| Commit linting | Enforces changelog-friendly history before merge | Friction for local exploratory commits | Enforce in CI after branch strategy is set |

## v0 Policy

- All public tags must match `^v0\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$` until a maintainer-approved exit decision exists.
- `v1.0.0` is blocked until there are several months of real-world usage, a documented public CLI/LSP/agent API, no breaking changes across at least one release cycle, and an explicit decision record.
- SemVer allows `0.y.z` for initial development where the public API is not stable. `vibe-xpls` should still document breaking CLI, LSP, and agent API changes in every release.

## Phase Gates

Every later implementation phase should produce runnable code and evidence:

- Phase start: define the command a user can run by the end of the phase.
- During phase: add or update tests before claiming behavior works.
- Before merge: run `go test ./...` for changed modules and any spike-specific commands.
- Before release: run the v0 version guard, changelog generation, GoReleaser snapshot or equivalent dry-run, and smoke-test the built binary.
- Before any non-v0 tag: require a decision record explicitly approving pre-1.0 exit.

## Recommendation

Use Conventional Commits and Release Please for changelog and release PR automation, GoReleaser for binary artifacts, and a repository-local version guard script that rejects non-`v0.X.X` tags. Product implementation phases should not be considered complete unless they leave a runnable command or binary.

## Confidence

High that v0 guardrails are necessary and simple to automate.

High that GoReleaser belongs in the active release path now that there is a runnable binary.

Medium that Release Please remains the right changelog workflow; this depends on whether maintainers prefer commit-derived automation or reviewed change fragments.

## Evidence That Would Change This Recommendation

- Maintainers prefer reviewed release-note fragments over commit-derived changelog entries.
- Release Please cannot enforce the custom v0 policy cleanly without accidental `v1.0.0` suggestions.
- Public contributors produce commit history that is too inconsistent for commit-derived changelogs without heavy cleanup.
