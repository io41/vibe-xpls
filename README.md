# vibe-xpls

`vibe-xpls` is an experimental Crossplane language server. It currently focuses on local Crossplane package detection, YAML-aware diagnostics, hover, and completion for the first Zed validation path.

## Install

Install the latest released version with Go:

```sh
go install github.com/io41/vibe-xpls/cmd/vibe-xpls@v0.0.1
```

Run it as an LSP server:

```sh
vibe-xpls serve
```

Check the installed version:

```sh
vibe-xpls --version
```

The development Zed extension currently launches a fixed local binary path during validation. Until that extension grows installer/configuration support, rebuild and copy the binary to the path expected by the extension when doing local Zed validation.

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
