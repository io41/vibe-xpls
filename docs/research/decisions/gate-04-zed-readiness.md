## Decision.

Treat the Zed integration path as viable, but keep manual Zed UI validation as a release gate. The `crossplane-yaml` extension should remain a thin launcher and syntax/highlighting integration layer; `vibe-xpls` should own Crossplane semantics in the language server and shared analyzer. Ordinary YAML should continue to use Zed's native YAML support.

## Evidence.

- The `crossplane-yaml` extension launches `vibe-xpls serve` for files classified as `Crossplane YAML`.
- The extension owns language classification and highlighting; package-root, multi-package, and no-root behavior belongs to the `vibe-xpls` analyzer.
- The Zed extension build and tests passed in the external repo: `cargo fmt --check`, `cargo test`, and `cargo build --target wasm32-wasip2`.
- `docs/research/lanes/02-human-editor-ux.md` says the first editor goal should be protocol-first LSP, proven in Zed, with diagnostics, completion, hover, and navigation as the core loop.
- `docs/research/lanes/09-existing-tooling.md` says the local Zed extension owns Crossplane YAML language selection and mixed template highlighting, while delegating semantics and package detection to the language server.
- `docs/research/lanes/02-human-editor-ux.md` records attach constraints: broad `.yaml` matching may require user `file_types`, and package-root detection must be tested for root manifests, nested packages, multi-package workspaces, and repositories without root manifests.

## Alternatives Considered.

- Make the extension own Crossplane semantics. Rejected because both `docs/research/lanes/02-human-editor-ux.md` and `docs/research/lanes/09-existing-tooling.md` point to a thin editor client and shared server-side semantics.
- Treat the Zed path as fully validated without manual coverage. Rejected because user-visible attach, launch, diagnostics, hover, completion, and stale-diagnostic behavior still need explicit validation.
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

- Manual Zed validation fails to launch `vibe-xpls serve` through `crossplane-yaml` or cannot surface diagnostics, hover, and completion.
- Manual Zed validation shows the server does not attach reliably for root packages, nested packages, multi-package workspaces, or documented `file_types` configurations.
- Zed extension APIs change enough that the current command-launch model is no longer the right integration path.
- User research shows another editor or CLI workflow is a stronger first integration gate than Zed.
- The extension must own semantic state to meet core UX requirements, contradicting the current thin-extension model.
