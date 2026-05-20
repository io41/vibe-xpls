## Decision.

Stay on `v0.X.X` until an explicit pre-1.0 exit decision after several months of real usage. Use Conventional Commits with Release Please for changelog and release PR automation. Guard release versions with `spikes/release/check-version.sh` so non-`v0` and malformed tags fail before publishing.

GoReleaser publishes release artifacts when Release Please creates a valid `v0.X.X` release.

## Evidence.

- `docs/research/lanes/10-release-phase-gates.md` blocks non-`v0.X.X` releases until a maintainer-approved exit decision and identifies Release Please as the active changelog automation.
- `docs/research/spikes/08-release.md` added and exercised `spikes/release/check-version.sh`; it accepts `v0.1.2` and `v0.1.2-rc.1`, rejects `v1.0.0` and malformed versions, and prints a policy error for invalid release versions.
- `docs/research/lanes/10-release-phase-gates.md` keeps GoReleaser in the release path now that there is a runnable binary and warns that release automation can over-automate version decisions before API stability.
- `docs/research/lanes/11-security-reliability.md` supports a conservative release posture because future product features may cross Docker, network, cluster, cache, and agent-tool trust boundaries that need explicit review gates.

## Alternatives Considered.

- Adopt Changie immediately. Deferred because reviewed change fragments add process overhead before the repository has public contributors or enough user-facing change volume to justify per-change fragments.
- Let GoReleaser own the changelog. Rejected because Release Please owns changelog and version PR automation.
- Allow normal SemVer progression to `v1.0.0`. Rejected until sustained usage, documented public CLI/LSP/agent APIs, and at least one release cycle without breaking changes justify an explicit exit decision.

## Risks.

- Release Please output quality depends on commit history quality; inconsistent commits may require squash cleanup or later commit linting.
- A local guard script only helps once it is wired into the real release path; manual tag creation can bypass it unless release procedure and CI enforce it.
- Staying on `v0.X.X` can make users expect instability even after parts of the project mature.
- Deferring GoReleaser leaves packaging decisions unresolved until the first real binary exists.

## What Would Change This Decision.

- Maintainers decide they want reviewed release-note fragments more than commit-derived changelogs.
- Release Please cannot enforce the custom v0 policy without suggesting or creating non-`v0` tags.
- Public commit history becomes too inconsistent for Release Please to produce reviewable changelogs without unacceptable cleanup.
