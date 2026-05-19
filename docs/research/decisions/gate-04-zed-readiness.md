## Decision.

Treat the Zed replacement path as viable for code-path proof, but keep manual Zed UI validation as a later gate. The Zed extension should remain a thin launcher and syntax/highlighting integration layer; `vibe-xpls` should own Crossplane semantics in the language server and shared analyzer. Ordinary YAML should continue to use Zed's native YAML support.

2026-05-18 update: first-runnable validation now uses the local `<zed-xpls-vibe-repo>` fork. That fork launches `<vibe-xpls-binary> serve` directly for files classified as `Crossplane YAML` and leaves package-root, multi-package, and no-root behavior to the `vibe-xpls` analyzer.

## Evidence.

- Historical spike evidence: `docs/research/spikes/02-zed-replacement.md` records that `<zed-up-xpls-repo>` was on branch `vibe-xpls-spike` at commit `ac1d8cb feat: allow vibe xpls binary override`.
- That historical spike added `VIBE_XPLS_BIN` support so the original extension could launch a local `vibe-xpls` binary when set, while preserving the existing `up xpls serve --verbose` fallback.
- The current validation fork `<zed-xpls-vibe-repo>` supersedes the temporary `VIBE_XPLS_BIN` path for this milestone and hardcodes `<vibe-xpls-binary> serve`.
- The Zed extension build and tests passed in the external repo: `cargo fmt --check`, `cargo test`, and `cargo build --target wasm32-wasip2`.
- The spike built a local LSP harness binary and proved only the stdio code path. It explicitly did not run manual Zed UI validation for startup logs, diagnostics, hover, completion, missing-binary behavior, or worktree shell environment propagation.
- `docs/research/lanes/02-human-editor-ux.md` says the first editor goal should be protocol-first LSP, proven in Zed, with diagnostics, completion, hover, and navigation as the core loop.
- `docs/research/lanes/09-existing-tooling.md` says the local Zed extension already owns Crossplane YAML language selection and mixed template highlighting, while delegating semantics to the language server. In the current validation fork, package detection is also delegated to the language server.
- `docs/research/lanes/02-human-editor-ux.md` records attach constraints: broad `.yaml` matching may require user `file_types`, and package-root detection must be tested for root manifests, nested packages, multi-package workspaces, and repositories without root manifests.

## Alternatives Considered.

- Make the extension own Crossplane semantics. Rejected because both `docs/research/lanes/02-human-editor-ux.md` and `docs/research/lanes/09-existing-tooling.md` point to a thin editor client and shared server-side semantics.
- Treat the Zed path as fully validated now. Rejected because `docs/research/spikes/02-zed-replacement.md` did not run the manual UI path.
- Replace ordinary YAML handling with the Crossplane language. Rejected because the existing Zed extension deliberately keeps ordinary YAML on native Zed YAML support.
- Clone Upbound `xpls` behavior as the contract. Rejected because `docs/research/lanes/09-existing-tooling.md` treats Upbound `xpls` as reference input, not a compatibility contract.

## Risks.

- Diagnostics, hover, and completion may behave differently through Zed than through the local stdio harness.
- Missing-binary handling and stale diagnostic clearing remain unvalidated user-visible paths.
- The extension may not attach in real repositories if users have not configured `file_types` for Crossplane YAML files.
- The analyzer may still mishandle root, nested, multi-package, or no-root repository shapes after Zed launches it.
- A future approval or trust model for configurable executables could be unsafe if it trusts only a command string instead of canonical executable path, symlink target, and content identity.
- Mixed YAML/template highlighting is known to be best effort and should not be coupled to semantic correctness.

## What Would Change This Decision.

- Manual Zed validation fails to launch `<vibe-xpls-binary> serve` through `zed-xpls-vibe` or cannot surface diagnostics, hover, and completion.
- Manual Zed validation shows the server does not attach reliably for root packages, nested packages, multi-package workspaces, or documented `file_types` configurations.
- Zed extension APIs change enough that the current command-launch model is no longer the right integration path.
- User research shows another editor or CLI workflow is a stronger first integration gate than Zed.
- The extension must own semantic state to meet core UX requirements, contradicting the current thin-extension model.
