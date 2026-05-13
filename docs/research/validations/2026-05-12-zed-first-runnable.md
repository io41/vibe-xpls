# Zed First Runnable Milestone Validation

## Binary

- Binary path: `<vibe-xpls-binary>`
- Canonical path: `<vibe-xpls-binary>`
- Version output: `vibe-xpls v0.0.1`
- Built from worktree: `<vibe-xpls-worktree>`
- Build command: `go build -o <vibe-xpls-binary> ./cmd/vibe-xpls`
- Path check command: `realpath <vibe-xpls-binary>`

## Zed Extension

- Extension repository: `<zed-xpls-vibe-repo>`
- Extension commit: `3138ae6106d567edbf056609a5d3ccb0674d5123` (`3138ae6`)
- Extension id: `zed-xpls-vibe`
- Extension name: `Zed xpls Vibe`
- Launch command: `<vibe-xpls-binary> serve`
- Launch wiring source evidence: `<zed-xpls-vibe-repo>/src/lib.rs` defines `MILESTONE_XPLS_BIN` as `<vibe-xpls-binary>` and returns a `zed::Command` with `args: ["serve"]`.
- Agent instruction evidence: `<zed-xpls-vibe-repo>/AGENTS.md` records that validation depends on rebuilding `<vibe-xpls-binary>` from this milestone worktree and installing `zed-xpls-vibe`, not the original `up-xpls` extension.
- Focused test evidence: `cargo test` in `<zed-xpls-vibe-repo>` passed with `9 passed; 0 failed`.
- Zed build evidence: `PATH="<rustup-bin-dir>:$PATH" cargo build --target wasm32-wasip2` in `<zed-xpls-vibe-repo>` passed.
- Installed extension evidence: Zed extension index contains `id = zed-xpls-vibe`, `name = Zed xpls Vibe`, and `repository = https://github.com/io41/zed-xpls-vibe`.

## Final Automated Verification

Run from `<vibe-xpls-worktree>` on 2026-05-13.

- `git status --short --branch`
  - Result: PASS. Output was `## research/crossplane-lsp-research-program` with no modified or untracked files before this validation document update.
- `go version`
  - Result: PASS. Output was `go version go1.26.3 darwin/arm64`.
- `go test ./...`
  - Result: PASS. Packages reported `ok` or `[no test files]`: `cmd/vibe-xpls`, `internal/analyzer`, `internal/app`, `internal/debugcli`, `internal/lsp`, `internal/source`, and `internal/testkit`.
- `go run ./cmd/vibe-xpls debug diagnostics --workspace internal/analyzer/testdata/workspaces/root --uri file:///composition.yaml --text 'apiVersion: apiextensions.crossplane.io/v1
kind: Composition
'`
  - Result: PASS. JSON output was `{"ok":true,"contract":"internal-debug","command":"diagnostics","data":[]}`, which includes `"contract":"internal-debug"` and `"ok":true`.
- LSP initialize smoke with generated `Content-Length`
  - Command:

    ```bash
    zsh -lc 'root="file://$PWD/internal/analyzer/testdata/workspaces/root"; body=$(printf "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"rootUri\":\"%s\",\"capabilities\":{}}}" "$root"); printf "Content-Length: %d\r\n\r\n%s" ${#body} "$body" | go run ./cmd/vibe-xpls serve'
    ```

  - Result: PASS. Output started with `Content-Length: 233` and response JSON included `"serverInfo":{"name":"vibe-xpls","version":"v0.0.1"}`.
- `rg -n "crossplane render|crossplane beta validate|docker|kubectl|kubeconfig|http.Get|net/http|os/exec" cmd internal`
  - Result: PASS. No matches. `rg` exited 1 because no external execution, Docker, cluster, kubeconfig, or network-read patterns were found under `cmd` or `internal`.
- `<vibe-xpls-binary> --version`
  - Result: PASS. Output was `vibe-xpls v0.0.1`.
- `git -C <zed-xpls-vibe-repo> rev-parse HEAD`
  - Result: PASS. Output was `3138ae6106d567edbf056609a5d3ccb0674d5123`.
- `rg -n "zed-xpls-vibe|Zed xpls Vibe|<vibe-xpls-binary>|serve" <zed-xpls-vibe-repo>/extension.toml <zed-xpls-vibe-repo>/src <zed-xpls-vibe-repo>/AGENTS.md`
  - Result: PASS. Current source and metadata show the `zed-xpls-vibe` extension id, `<vibe-xpls-binary>` binary path, and `serve` argument.

## Required Checks

The following are manual Zed validation checks. Leave a checkbox unchecked until the check has actually been performed.

- [x] Zed launches `<vibe-xpls-binary>`.
- [x] Missing-binary behavior is understandable when `<vibe-xpls-binary>` is missing.
- [x] Root package attaches.
- [x] Nested package attaches.
- [ ] Multi-package workspace attaches without schema cross-contamination.
- [x] No-root workspace stays quiet.
- [x] `.yaml` attach behavior was checked without user `file_types` mapping.
- [x] `.yaml` attach behavior was checked with the documented Crossplane `file_types` mapping.
- [x] Diagnostics appear.
- [x] Diagnostics clear after valid edits.
- [x] Diagnostics clear after document close.
- [x] Hover works visibly.
- [ ] Completion works visibly.

## Evidence Notes

Record Zed log excerpts, fixture paths, and screenshots or manual observations here during validation.

Do not record environment variables, kubeconfig content, registry credentials, tokens, passwords, private key material, or secret-bearing file contents.

### Manual Validation Status

Manual Zed validation is partially complete. Launch, root package attachment, nested package attachment, no-root quietness, diagnostics, diagnostic clearing, missing-binary behavior, hover, and the corrected `composition.yaml` file-type mapping were observed. Completion is visible but buggy, and multi-package schema isolation remains human-pending.

Historical unsuccessful attempts with the original `up-xpls` extension on 2026-05-13:

- `zed --version` returned `Zed 1.1.8 - <zed-app>`.
- Launch attempt for an ordinary `.yaml` file: `VIBE_XPLS_BIN=<vibe-xpls-binary> zed <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api/composition.yaml` exited 0.
- Zed log evidence after that launch showed the fixture opening and the stock YAML language server starting. This is expected without a user `file_types` mapping because `api/composition.yaml` is not one of the extension's built-in `Crossplane YAML` suffixes:
  - `2026-05-13T06:29:59+02:00 INFO  [worktree] inserting parent git repo for this worktree: "internal/analyzer/testdata/workspaces/root/api/composition.yaml"`
  - `2026-05-13T06:30:00+02:00 INFO  [lsp] starting language server process. binary path: "<homebrew-bin-dir>/node", working directory: "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api", args: ["<zed-data-dir>/languages/yaml-language-server/node_modules/yaml-language-server/bin/yaml-language-server", "--stdio"]`
- Launch attempt for the built-in Crossplane suffix with the package directory as the workspace: `VIBE_XPLS_BIN=<vibe-xpls-binary> zed --new <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/crossplane.yaml` exited 0.
- Zed log evidence after that launch reached the `up-xpls` language server path but stopped before launching because this fixture worktree was not trusted in the local Zed UI:
  - `2026-05-13T06:33:35+02:00 INFO  [project::trusted_worktrees] Worktree "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root" is not trusted`
  - `2026-05-13T06:33:35+02:00 INFO  [project::lsp_store] Waiting for worktree "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root" to be trusted, before starting language server up-xpls`
- The Zed log did not show `<vibe-xpls-binary>` starting during these attempts. At this point, the `Zed launches <vibe-xpls-binary>` checkbox was still incomplete.
- No diagnostics, diagnostic clearing, hover, completion, root/nested/multi-package attach, no-root quietness, or `.yaml` file-type mapping behavior was visually observed during these original-extension attempts.

Updated local observations after replacing the validation extension on 2026-05-13:

- The original `up-xpls` extension was uninstalled from Zed.
- A forked dev extension from `<zed-xpls-vibe-repo>` was installed and shown in Zed as `Zed xpls Vibe` v0.0.1.
- Zed extension build logs showed `<zed-xpls-vibe-repo>` compiling and writing `extension.wasm`.
- Zed extension index evidence contains `zed-xpls-vibe` and no longer relies on `VIBE_XPLS_BIN` for this validation path.
- Zed log evidence includes:
  - `2026-05-13T09:39:18+02:00 INFO  [project::lsp_store] Waiting for worktree "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root" to be trusted, before starting language server zed-xpls-vibe`
  - `2026-05-13T09:39:29+02:00 INFO  [lsp] starting language server process. binary path: "<vibe-xpls-binary>", working directory: "<user-home>/Code/ista-se/cas/devops/config/cluster-as-a-service/configurations/ista-azure-service-bus", args: ["serve"]`
- Manual observation from Tim Kersten: after installing the new extension, it was running, and hovering over symbols in `<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api/composition.yaml` showed useful descriptions.
- The hover observation is fixture-backed evidence for root package attachment and visible hover behavior in the real Zed path.

Additional manual observations from Tim Kersten on 2026-05-13:

- Diagnostics appeared, cleared after valid edits, and cleared after document close in Zed.
- Completion did not show suggestions in Crossplane YAML, while a separate Python file did show suggestions. Completion remains an open product issue.
- Moving `<vibe-xpls-binary>` away, restarting the language server, and restoring the binary produced a readable Zed startup error for `zed-xpls-vibe`.
- After restoring the binary, Zed restarted with `<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api/composition.yaml` classified as ordinary `YAML`; manually changing it back to `Crossplane YAML` restored the working diagnostics path.
- Opening `<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/nested/packages/network/` as its own Zed project and opening `crossplane.yaml` showed the `zed-xpls-vibe` language server running.
- Opening the no-root fixture kept `plain.yaml` on ordinary `YAML`; manually forcing `Crossplane YAML` showed the expected readable no-root error from `zed-xpls-vibe`.
- Local Zed settings were corrected to map `**/composition.yaml` and `**/composition.yml` to `Crossplane YAML` in addition to the documented `*-composition.*` and `*-definition.*` patterns.
- After restart/reload, `<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api/composition.yaml` opened as `Crossplane YAML` without manual language switching, and the language server started.
- Hover and diagnostics continued to work after the corrected mapping.
- Zed showed some completion behavior, but completion was buggy: it did not trigger in the correct places and/or triggered in incorrect places. Completion remains unchecked until the trigger/context behavior is corrected and revalidated.
