## Decision.

Start public releases at `v0.0.1` and stay on `v0.X.X` until an explicit pre-1.0 exit decision after several months of real usage. Use Conventional Commits with `git-cliff` as the first changelog generator. Guard release versions with `spikes/release/check-version.sh` so non-`v0` and malformed tags fail before changelog generation, tagging, or publishing.

GoReleaser and release-please are deferred. Add GoReleaser when there is a real CLI or LSP binary to package, checksum, and publish. Revisit release-please only after several manual `v0` releases prove the desired cadence and after the custom v0 policy can be enforced around release PR automation.

## Evidence.

- `docs/research/lanes/10-release-phase-gates.md` recommends `v0.0.1` as the first public tag, blocks non-`v0.X.X` releases until a maintainer-approved exit decision, and identifies `git-cliff` as the lowest-friction first changelog generator.
- `docs/research/spikes/08-release.md` added and exercised `spikes/release/check-version.sh`; it accepts `v0.0.1` and `v0.1.2-rc.1`, rejects `v1.0.0` and malformed versions, and prints a policy error for invalid release versions.
- `docs/research/spikes/08-release.md` recommends the first dry-run path as `spikes/release/check-version.sh v0.0.1`, `git-cliff -o CHANGELOG.md`, and review of the resulting `CHANGELOG.md` diff.
- `docs/research/lanes/10-release-phase-gates.md` defers GoReleaser until runnable product binaries exist and warns that release automation can over-automate version decisions before API stability.
- `docs/research/lanes/11-security-reliability.md` supports a conservative release posture because future product features may cross Docker, network, cluster, cache, and agent-tool trust boundaries that need explicit review gates.

## Alternatives Considered.

- Adopt release-please immediately. Rejected for now because release PR automation and automated version bumps add policy surface before the project has proven a manual release cadence or an enforceable pre-1.0 exit gate.
- Adopt Changie immediately. Deferred because reviewed change fragments add process overhead before the repository has public contributors or enough user-facing change volume to justify per-change fragments.
- Let GoReleaser own the first changelog. Deferred because the current research stage has no product binary to package; GoReleaser becomes valuable once binary archives, checksums, and publish targets are real.
- Allow normal SemVer progression to `v1.0.0`. Rejected until sustained usage, documented public CLI/LSP/agent APIs, and at least one release cycle without breaking changes justify an explicit exit decision.

## Risks.

- `git-cliff` output quality depends on commit history quality; inconsistent commits may require squash cleanup or later commit linting.
- A local guard script only helps once it is wired into the real release path; manual tag creation can bypass it unless release procedure and CI enforce it.
- Staying on `v0.X.X` can make users expect instability even after parts of the project mature.
- Deferring GoReleaser leaves packaging decisions unresolved until the first real binary exists.

## What Would Change This Decision.

- Maintainers decide they want reviewed release-note fragments more than commit-derived changelogs.
- Release-please proves it can enforce the custom v0 policy without suggesting or creating non-`v0` tags.
- A CLI or LSP binary becomes the first public artifact and needs archives, checksums, and publish automation immediately.
- Public commit history becomes too inconsistent for `git-cliff` to produce reviewable changelogs without unacceptable cleanup.
