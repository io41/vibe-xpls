# Release Automation

## Summary

This note records the active release automation shape for the `vibe-xpls` Go repository and the version guard under `spikes/release/check-version.sh`.

Release Please owns changelog and release PR automation. GoReleaser owns release artifacts. The version guard keeps releases on the `v0.X.X` line until a maintainer-approved pre-1.0 exit decision exists.

All releases should stay on `v0.X.X` until the project has months of real usage and an explicit pre-1.0 exit decision.

## Tooling Compared

- `git-cliff`: The docs describe generating changelog files from Git history using Conventional Commits and custom parsers, with quickstart commands such as `git-cliff --init` and `git-cliff -o CHANGELOG.md`: https://git-cliff.org/docs/
- GoReleaser changelog: The changelog backend can use `git`, `github`, or `github-native`, and grouping, sorting, and filtering support depends on the selected backend: https://goreleaser.com/customization/publish/changelog/
- release-please: The repository describes automation for changelog generation, GitHub releases, and version bumps by parsing Conventional Commits and creating release PRs. It also says publication to package managers is out of scope: https://github.com/googleapis/release-please
- Conventional Commits: The specification defines a lightweight commit-message convention that supports automated tooling and maps `fix` and `feat` style changes to SemVer patch and minor releases: https://www.conventionalcommits.org/en/v1.0.0/

## Commands Run

```text
spikes/release/check-version.sh v0.1.2
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

The guard rejects non-`v0` releases and malformed version strings such as extra numeric segments, embedded text in the minor position, and trailing text after the patch. The release workflow runs this guard before GoReleaser publishes artifacts.

## Changelog Recommendation

Use Release Please as the active changelog generator.

It matches the current project stage because it opens reviewable release PRs from Conventional Commits and keeps version updates, release notes, and GitHub releases on one path.

Keep GoReleaser focused on binary packaging, archives, checksums, and publish targets.

## Release Dry-Run Recommendation

For local release checks, use:

```text
spikes/release/check-version.sh v0.1.2
goreleaser release --snapshot --clean
```

GitHub release automation runs the same guard before publishing any release artifacts.

## Decision Impact

The release system should remain deliberately small: enforce `v0.X.X`, generate reviewable changelog PRs, and publish artifacts only through the guarded release workflow.

The policy decision is that `v1.0.0` and later tags are blocked until an explicit exit decision after sustained usage.
