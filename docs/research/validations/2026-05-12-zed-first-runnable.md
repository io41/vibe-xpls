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
