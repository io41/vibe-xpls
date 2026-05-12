# LSP Harness Spike

## Summary

This spike proves that `vibe-xpls` can run a small, dependency-free Go LSP harness over `Content-Length` framed JSON-RPC messages. It is intentionally not production architecture. The goal is to validate the minimum protocol loop needed before evaluating larger LSP framework choices.

The harness stores opened documents, publishes deterministic diagnostics when a document contains `xpls-spike-error`, answers hover and completion requests, and shuts down cleanly.

## Commands Run

- `cd spikes/lsp-harness && go test ./...`

Initial red-run output after creating the tests and first implementation:

```text
--- FAIL: TestInitializeReturnsCapabilities (0.00s)
...
FAIL
FAIL	github.com/io41/vibe-xpls/spikes/lsp-harness	0.357s
FAIL
```

The failure was in the test helper that decoded output frames. It used `bufio.Reader.Buffered()` before reading, so it never consumed the server output. The helper now reads framed messages until EOF.

Successful output after the fix:

```text
ok  	github.com/io41/vibe-xpls/spikes/lsp-harness	0.437s
```

## Protocol Features Proven

- `initialize` returns server capabilities including document sync, hover, and completion.
- `textDocument/didOpen` stores document text and publishes diagnostics.
- `textDocument/didChange` replaces document text and republishes diagnostics.
- `textDocument/didClose` removes document state and clears diagnostics.
- `textDocument/publishDiagnostics` is emitted as an LSP notification.
- `textDocument/hover` returns markdown hover content.
- `textDocument/completion` returns deterministic completion items.
- `shutdown` returns a JSON-RPC response.
- `exit` terminates the server loop.

## Limitations

- The server is a spike and has no incremental text edit support.
- It uses hand-written JSON-RPC framing rather than a selected framework.
- Diagnostics, hover, and completion are deterministic fixtures, not Crossplane semantics.
- There is no real editor integration in this spike; Zed replacement is covered by the next spike.
- There is no external command execution, rendering, schema lookup, or workspace indexing.

## Evidence for Later Decisions

The spike supports the analyzer-first direction because LSP handlers can remain thin and deterministic while delegating future semantics elsewhere. It also gives the Zed replacement spike a local binary target that does not depend on Upbound `xpls`.

The framework decision should still be made after comparing this hand-written harness with `go-lsp` and the thin `go.lsp.dev/protocol` candidate. This spike proves the protocol loop is small enough to test directly, not that handwritten transport is the final choice.
