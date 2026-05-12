# Release And Phase-Gate Research

## Summary

`vibe-xpls` should start at `v0.0.1`, stay on `v0.X.X`, and use release tooling that separates changelog quality from binary publishing. The best initial stack is Conventional Commits plus `git-cliff` for changelog generation and a small explicit version guard. Add GoReleaser once runnable product binaries exist.

Release Please is attractive when the project wants automated release PRs and GitHub releases from commit history. Changie is attractive when maintainers want explicit change fragments before release. For this research-heavy early phase, `git-cliff` is the lowest-friction changelog generator because it can work from the existing Git history without forcing release PR automation or fragment files on every commit.

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
| `git-cliff` | Generates `CHANGELOG.md` from Git history with configurable parsers/templates | Changelog quality depends on commit messages | Best initial changelog generator |
| Changie | Change fragments make release notes explicit and reviewable | More process overhead for every user-facing change | Consider after public contributors appear |
| Release Please | Automates changelog generation, release PRs, GitHub releases, and version bumps | Can over-automate version decisions before API stability | Consider after release workflow stabilizes |
| GoReleaser | Builds/publishes Go artifacts and can generate release changelogs | Not useful until there are real binaries; publishing config adds supply-chain risk | Add with first runnable CLI/LSP binary |
| Commit linting | Enforces changelog-friendly history before merge | Friction for local exploratory commits | Enforce in CI after branch strategy is set |

## v0 Policy

- The first public release tag is `v0.0.1`.
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

Use Conventional Commits and `git-cliff` as the first changelog workflow, plus a repository-local version guard script that rejects non-`v0.X.X` tags. Add GoReleaser when the project has a real CLI or LSP binary to distribute. Revisit Release Please after the first few manual `v0` releases prove the desired release cadence.

The release spike should create the v0 guard first and document a `git-cliff` or equivalent changelog dry-run path. Product implementation phases should not be considered complete unless they leave a runnable command or binary.

## Confidence

High that v0 guardrails are necessary and simple to automate.

High that GoReleaser should wait for a binary.

Medium that `git-cliff` is better than Changie for the first public releases; this depends on whether maintainers prefer commit-derived or fragment-derived changelogs.

## Evidence That Would Change This Recommendation

- Maintainers prefer reviewed release-note fragments over commit-derived changelog entries.
- Release Please proves it can enforce the custom v0 policy cleanly without accidental `v1.0.0` suggestions.
- GoReleaser becomes necessary earlier because the LSP harness or CLI spike turns into the first installable artifact.
- Public contributors produce commit history that is too inconsistent for `git-cliff` without heavy cleanup.
