# Release Spike

## Summary

This spike compares lightweight changelog and release automation options for the early `vibe-xpls` Go repository and adds a small version guard under `spikes/release/check-version.sh`.

The recommendation is to start with `git-cliff` for changelog generation because it is local, easy to run in dry-run style, and does not require GitHub API state. GoReleaser and release-please remain useful later, once release packaging, binaries, GitHub releases, and release PR automation are ready.

All releases should stay on `v0.X.X` until the project has months of real usage and an explicit pre-1.0 exit decision.

## Tooling Compared

- `git-cliff`: The docs describe generating changelog files from Git history using Conventional Commits and custom parsers, with quickstart commands such as `git-cliff --init` and `git-cliff -o CHANGELOG.md`: https://git-cliff.org/docs/
- GoReleaser changelog: The changelog backend can use `git`, `github`, or `github-native`, and grouping, sorting, and filtering support depends on the selected backend: https://goreleaser.com/customization/publish/changelog/
- release-please: The repository describes automation for changelog generation, GitHub releases, and version bumps by parsing Conventional Commits and creating release PRs. It also says publication to package managers is out of scope: https://github.com/googleapis/release-please
- Conventional Commits: The specification defines a lightweight commit-message convention that supports automated tooling and maps `fix` and `feat` style changes to SemVer patch and minor releases: https://www.conventionalcommits.org/en/v1.0.0/

## Commands Run

```text
spikes/release/check-version.sh v0.0.1
spikes/release/check-version.sh v0.1.2-rc.1
```

Result: both commands exited 0 with no output.

```text
sh -c 'spikes/release/check-version.sh v1.0.0; test $? -ne 0'
sh -c 'spikes/release/check-version.sh v0.1.2.3; test $? -ne 0'
sh -c 'spikes/release/check-version.sh v0.1beta.2; test $? -ne 0'
sh -c 'spikes/release/check-version.sh v0.1.2foo; test $? -ne 0'
```

Result: exit code 0 from each wrapper, with the guard printing:

```text
release version must stay on v0.X.X before explicit pre-1.0 exit approval
```

## v0 Guard Result

The guard accepts strict `v0.<minor>.<patch>` versions and optional prerelease-style suffixes, implemented as:

```text
^v0\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$
```

The guard rejects non-`v0` releases and malformed version strings such as extra numeric segments, embedded text in the minor position, and trailing text after the patch. This proves the spike can enforce the current release policy before any tag or publication step. The script must still be integrated into release automation before the first public release, for example as a required check before changelog generation, tag creation, GoReleaser execution, or release-please merge handling.

## Changelog Recommendation

Use `git-cliff` as the first changelog generator.

It matches the current project stage because it can run locally over Git history, can be reviewed before committing generated output, and does not depend on GitHub release state, labels, tokens, or release PR lifecycle. Its Conventional Commits and custom parser support also leaves room to tighten history conventions without adopting full release orchestration yet.

Defer GoReleaser changelog ownership until binary packaging, archives, checksums, and publish targets are in scope. Defer release-please until the project wants GitHub release PRs and automated version bump workflow as a managed process.

## Release Dry-Run Recommendation

For the first release lane, use a local dry-run sequence:

```text
spikes/release/check-version.sh v0.0.1
git-cliff -o CHANGELOG.md
git diff -- CHANGELOG.md
```

When packaging becomes necessary, add GoReleaser in snapshot or dry-run mode after the version guard. When GitHub release automation becomes necessary, evaluate release-please with the same guard enforced before merging any release PR that would create a non-`v0` tag.

## Decision Impact

The release system should remain deliberately small for now: enforce `v0.X.X`, generate a reviewable changelog locally, and avoid coupling early releases to GitHub API state or package publication.

The next implementation step is to wire `spikes/release/check-version.sh` into the actual release path before any public release. The policy decision is that `v1.0.0` and later tags are blocked until an explicit exit decision after sustained usage.
