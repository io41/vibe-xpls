# vibe-xpls

> **Project status:** `vibe-xpls` is a very early-stage experimental language server. Expect rough edges, incomplete Crossplane coverage, and breaking changes while the core editor experience is still being shaped.

`vibe-xpls` is an experimental Crossplane language server. It currently focuses on local Crossplane package detection, YAML-aware diagnostics, hover, and completion for Crossplane package authoring.

## Built-In Crossplane Schemas

`vibe-xpls` ships an offline generated schema bundle for Crossplane core resources. The bundle is generated from pinned Crossplane release artifacts and is used for YAML key completions and completion documentation.

Current built-in release lines:

- Crossplane `v1.20.7`
- Crossplane `v2.2.1`

Runtime does not download schemas, read registries, or connect to clusters.

## Install

Install the latest released version with Go:

```sh
go install github.com/io41/vibe-xpls/cmd/vibe-xpls@latest
```

Run it as an LSP server:

```sh
vibe-xpls serve
```

Check the installed version:

```sh
vibe-xpls --version
```

For Zed users, the `crossplane-yaml` extension starts `vibe-xpls serve` and manages the pinned `vibe-xpls` installation by default.

For local `vibe-xpls` development, point Zed at a local build with `lsp.crossplane-yaml.binary.path` or place a compatible `vibe-xpls` binary on `PATH`.

## Development

Run the test suite:

```sh
go test ./...
```

Run the CLI from source:

```sh
go run ./cmd/vibe-xpls --version
```

## Releases

Releases use SemVer and stay on the `v0.X.X` line until maintainers explicitly approve a pre-1.0 exit. `v1.0.0` and later are blocked by policy until the criteria in [docs/research/release-policy.md](docs/research/release-policy.md) are met.

Release Please maintains [CHANGELOG.md](CHANGELOG.md) from Conventional Commits and opens release pull requests on merges to `main`. When a release is created, GoReleaser builds GitHub Release artifacts for Linux, macOS, and Windows.

The release workflow uses the built-in `GITHUB_TOKEN` by default. If branch protection requires CI to run on Release Please pull requests, configure a `RELEASE_PLEASE_TOKEN` repository secret backed by a token that is allowed to trigger workflows.

## License

MIT. See [LICENSE](LICENSE).
