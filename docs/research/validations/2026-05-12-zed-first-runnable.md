# Zed First Runnable Milestone Validation

## Binary

- Binary path: `<vibe-xpls-binary>`
- Canonical path: `<vibe-xpls-binary>`
- Version output: `vibe-xpls v0.0.1`
- Built from worktree: `<vibe-xpls-worktree>`
- Build command: `go build -o <vibe-xpls-binary> ./cmd/vibe-xpls`
- Path check command: `realpath <vibe-xpls-binary>`

## Zed Extension

- Extension repository: `<zed-up-xpls-repo>`
- Extension commit: `ac1d8cb5f6bd6c16f08af1db8fb8c94cc42c0e6d` (`ac1d8cb`)
- Launch variable: `VIBE_XPLS_BIN=<vibe-xpls-binary>`
- Launch wiring source evidence: `<zed-up-xpls-repo>/src/lib.rs:108` reads `worktree.shell_env()`, `<zed-up-xpls-repo>/src/lib.rs:110` checks `VIBE_XPLS_BIN`, and `<zed-up-xpls-repo>/src/lib.rs:111` returns a `zed::Command` using that value before the `up xpls serve` fallback.
- Unit test source evidence: `<zed-up-xpls-repo>/src/lib.rs:228` defines `reads_vibe_xpls_override_from_shell_env`.
- Focused test evidence: `CARGO_TARGET_DIR=<tmp-dir>/zed-up-xpls-target cargo test reads_vibe_xpls_override_from_shell_env --manifest-path <zed-up-xpls-repo>/Cargo.toml` passed with `1 passed; 0 failed; 10 filtered out`.

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
- `git -C <zed-up-xpls-repo> rev-parse HEAD`
  - Result: PASS. Output was `ac1d8cb5f6bd6c16f08af1db8fb8c94cc42c0e6d`, preserving the Task 13 extension evidence.
- `rg -n "VIBE_XPLS_BIN|shell_env|zed::Command|reads_vibe_xpls_override_from_shell_env" <zed-up-xpls-repo>/src/lib.rs`
  - Result: PASS. Current source still shows `worktree.shell_env()` at line 108, `VIBE_XPLS_BIN` lookup at line 110, `zed::Command` construction at line 111, and the focused unit test at line 229.

## Required Checks

The following are manual Zed validation checks. Leave a checkbox unchecked until the check has actually been performed.

- [ ] Zed launches `<vibe-xpls-binary>`.
- [ ] Missing-binary behavior is understandable when `VIBE_XPLS_BIN` points to `<tmp-dir>/missing-vibe-xpls`.
- [ ] Root package attaches.
- [ ] Nested package attaches.
- [ ] Multi-package workspace attaches without schema cross-contamination.
- [ ] No-root workspace stays quiet.
- [ ] `.yaml` attach behavior was checked without user `file_types` mapping.
- [ ] `.yaml` attach behavior was checked with the documented Crossplane `file_types` mapping.
- [ ] Diagnostics appear.
- [ ] Diagnostics clear after valid edits.
- [ ] Diagnostics clear after document close.
- [ ] Hover works visibly.
- [ ] Completion works visibly.

## Evidence Notes

Record Zed log excerpts, fixture paths, and screenshots or manual observations here during validation.

Do not record environment variables, kubeconfig content, registry credentials, tokens, passwords, private key material, or secret-bearing file contents.

### Manual Validation Status

Full manual Zed validation remains human-pending. The automated and log-only checks above do not prove the visual editor behaviors, so the required UI checkboxes remain unchecked.

Local observations on 2026-05-13:

- `zed --version` returned `Zed 1.1.8 - <zed-app>`.
- Launch attempt for an ordinary `.yaml` file: `VIBE_XPLS_BIN=<vibe-xpls-binary> zed <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api/composition.yaml` exited 0.
- Zed log evidence after that launch showed the fixture opening and the stock YAML language server starting. This is expected without a user `file_types` mapping because `api/composition.yaml` is not one of the extension's built-in `Crossplane YAML` suffixes:
  - `2026-05-13T06:29:59+02:00 INFO  [worktree] inserting parent git repo for this worktree: "internal/analyzer/testdata/workspaces/root/api/composition.yaml"`
  - `2026-05-13T06:30:00+02:00 INFO  [lsp] starting language server process. binary path: "<homebrew-bin-dir>/node", working directory: "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/api", args: ["<zed-data-dir>/languages/yaml-language-server/node_modules/yaml-language-server/bin/yaml-language-server", "--stdio"]`
- Launch attempt for the built-in Crossplane suffix with the package directory as the workspace: `VIBE_XPLS_BIN=<vibe-xpls-binary> zed --new <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root <vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root/crossplane.yaml` exited 0.
- Zed log evidence after that launch reached the `up-xpls` language server path but stopped before launching because this fixture worktree was not trusted in the local Zed UI:
  - `2026-05-13T06:33:35+02:00 INFO  [project::trusted_worktrees] Worktree "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root" is not trusted`
  - `2026-05-13T06:33:35+02:00 INFO  [project::lsp_store] Waiting for worktree "<vibe-xpls-worktree>/internal/analyzer/testdata/workspaces/root" to be trusted, before starting language server up-xpls`
- The Zed log did not show `<vibe-xpls-binary>` starting during these attempts. Therefore the `Zed launches <vibe-xpls-binary>` checkbox is not marked complete.
- No diagnostics, diagnostic clearing, hover, completion, root/nested/multi-package attach, no-root quietness, or `.yaml` file-type mapping behavior was visually observed.
