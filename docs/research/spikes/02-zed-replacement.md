# Zed Replacement Spike

## Summary

This spike validates the Zed extension code path for selecting a local `vibe-xpls` binary without treating Upbound `xpls` as a compatibility contract. It does not validate the full Zed UI path yet. The useful contract is the Zed extension launcher shape: identify Crossplane package worktrees, attach the `Crossplane YAML` language server, and start a stdio server.

The temporary Zed branch adds a `VIBE_XPLS_BIN` override so the extension can launch a local `vibe-xpls` binary when the variable is present, while keeping the existing `up xpls serve --verbose` fallback unchanged.

## Current Extension Contract

External repository: `<zed-up-xpls-repo>`

Initial state before the spike:

```text
## main...origin/main
```

The extension currently:

- Defines language server id `up-xpls`.
- Defines a `Crossplane YAML` language.
- Starts `up xpls serve --verbose`.
- Requires the `up` CLI on `PATH` when no override is configured.
- Detects package roots with `crossplane.yaml` or `upbound.yaml`.
- Keeps mixed YAML/template highlighting in the Zed grammar layer.

Upbound `xpls` remains reference-only. The replacement target is the Zed extension command contract, not Upbound server compatibility.

## Temporary Branch Or Diff

Temporary branch:

```text
vibe-xpls-spike
```

External commit:

```text
ac1d8cb feat: allow vibe xpls binary override
```

Final external repository state after this spike:

```text
## vibe-xpls-spike
```

The external repository is intentionally left on `vibe-xpls-spike` because the override was committed there and the next manual Zed run should use that branch. The branch was clean when recorded.

Diff summary:

```text
src/lib.rs | 50 +++++++++++++++++++++++++++++++++++++++++++++++---
```

Behavior added:

- Reads `VIBE_XPLS_BIN` from `worktree.shell_env()`.
- Trims whitespace and ignores empty override values.
- When set, launches the override binary with no args.
- When unset, keeps the existing `up` CLI lookup and `xpls serve --verbose` args.
- Adds unit coverage for override normalization and shell-env lookup.

## Commands Run

In `<zed-up-xpls-repo>`:

```text
git switch -c vibe-xpls-spike
cargo fmt --check
cargo test
PATH="<rustup-bin-dir>:$PATH" cargo build --target wasm32-wasip2
git commit --no-gpg-sign -m "feat: allow vibe xpls binary override"
```

Successful test output:

```text
running 11 tests
...
test result: ok. 11 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out
```

Successful WASM build output:

```text
Finished `dev` profile [unoptimized + debuginfo] target(s) in 7.16s
```

In `vibe-xpls`:

```text
cd spikes/lsp-harness && go build -o <vibe-xpls-lsp-harness> .
test -x <vibe-xpls-lsp-harness>
```

## Manual Zed Result

Manual Zed UI launch was not run in this headless execution. The spike still produced a Zed-loadable WASM extension build and a local harness binary at:

```text
<vibe-xpls-lsp-harness>
```

The next manual check should launch Zed from a shell that exports:

```text
VIBE_XPLS_BIN=<vibe-xpls-lsp-harness>
```

Then open a worktree with a valid root `crossplane.yaml` or `upbound.yaml` and confirm startup logs, diagnostics, hover, and completion. Also test attach behavior for a nested package, a multi-package workspace, a workspace without root manifests, and a `.yaml` file before and after any documented `file_types` mapping.

Unvalidated items:

- Zed startup log line for the override command.
- Actual diagnostics visible in the Zed UI.
- Hover and completion through the Zed client.
- Missing-binary behavior for `VIBE_XPLS_BIN`.
- Worktree shell-environment propagation in a manually launched Zed session.
- Trust identity for override binaries: canonical path, symlink target, and content hash/digest are not yet checked or shown to users.
- Root detection for nested and multi-package repositories.
- Default attachment behavior when a repository lacks root `crossplane.yaml` or `upbound.yaml`.
- Whether first release relies on documented manual `file_types` mappings for broad `.yaml` coverage or requires extension changes before release.

## Compatibility Findings

- The extension can keep its existing `Crossplane YAML` language, grammar, and root detection for the proven code path.
- The fallback `up xpls serve --verbose` path remains intact for current users.
- A `vibe-xpls` binary can be introduced as an environment-selected replacement without renaming the language server id or changing file classification.
- The LSP harness still does not prove Zed UI behavior. It proves only the local stdio protocol loop needed for the next manual Zed run.
- Real readiness also depends on attach coverage: root manifest detection, nested package detection, multi-package workspaces, and documented or implemented `file_types` behavior for Crossplane YAML files.
- A future trust model should not treat any prior `VIBE_XPLS_BIN` approval as valid after the executable path, symlink target, mtime+size, content hash, or workspace realpath changes.

## Decision Impact

The Zed replacement path is promising but not fully proven. The code-path evidence is enough to keep Zed as the first editor gate, but the gate is still open until a manual Zed run confirms startup logs, diagnostics, hover, completion, and worktree environment propagation with `VIBE_XPLS_BIN`.

`vibe-xpls` should expose a stdio LSP binary that the extension can launch directly. The extension should remain a thin launcher/highlighting layer, while Crossplane semantics live in the analyzer and LSP server. The next Zed decision must state whether first release accepts manual `file_types` setup or requires extension changes for automatic attachment in common repository layouts.
