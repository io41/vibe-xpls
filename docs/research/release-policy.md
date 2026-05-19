# Release Policy

`vibe-xpls` starts at `v0.0.1`.

Public releases must remain on the `v0.X.X` line until maintainers explicitly approve leaving pre-1.0 after several months of real-world usage.

## Pre-1.0 Exit Criteria

`v1.0.0` is blocked until all of these are true:

- At least 90 days of real-world use.
- A documented public API and CLI surface.
- One release cycle with no breaking CLI, LSP, or agent API changes.
- A maintainer-approved migration note.
- An explicit decision record under `docs/research/decisions/`.

## Release Guard

Release automation must reject tags that do not match:

```text
^v0\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$
```

The only exception is a maintainer-approved exit-from-v0 decision.

## Automation

Release Please owns `CHANGELOG.md` and the release pull request on `main`.
Release Please is configured to keep pre-1.0 changes on the `v0.X.X` line.
The first release was bootstrapped at `0.0.1`; later release pull requests
are calculated from Conventional Commits.

GoReleaser publishes binaries when Release Please creates a valid `v0.X.X`
release from a release pull request merge. The workflow runs
`spikes/release/check-version.sh` before publishing.
