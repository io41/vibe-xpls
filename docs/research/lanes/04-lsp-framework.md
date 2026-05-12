# LSP Framework Research

## Summary

As of 2026-05-12, `vibe-xpls` should not anchor on a heavy LSP framework first. The approved analyzer-first, protocol-first direction is better served by a thin LSP adapter around an internal analyzer core, with the protocol layer kept boring and replaceable.

Among the Go options, `go-lsp` is the strongest full framework if we want a batteries-included server and test harness, but `go.lsp.dev/protocol` + JSON-RPC is the better architectural fit for this project because it keeps semantics out of the transport layer. `glsp` is behind the current protocol surface, and the legacy Sourcegraph stack is archived or generic enough that it is better treated as reference material than as a base. Non-Go stacks are credible, but they add runtime and dependency cost without improving the fit for a Go codebase.

## Sources

- Language Server Protocol 3.17 specification: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- go-lsp docs: https://www.owenrumney.co.uk/go-lsp/
- go-lsp capabilities: https://www.owenrumney.co.uk/go-lsp/capabilities.html
- go-lsp testing: https://www.owenrumney.co.uk/go-lsp/testing.html
- go-lsp package page: https://pkg.go.dev/github.com/owenrumney/go-lsp
- glsp package page: https://pkg.go.dev/github.com/tliron/glsp
- go.lsp.dev/protocol package page: https://pkg.go.dev/go.lsp.dev/protocol
- go.lsp.dev/jsonrpc2 package page: https://pkg.go.dev/go.lsp.dev/jsonrpc2
- go-language-server/protocol repo: https://github.com/go-language-server/protocol
- sourcegraph/go-lsp archive note: https://github.com/sourcegraph/sourcegraph-go
- sourcegraph/jsonrpc2 package page: https://pkg.go.dev/github.com/sourcegraph/jsonrpc2
- VS Code LSP node stack: https://github.com/microsoft/vscode-languageserver-node
- tower-lsp repo: https://github.com/ebkalderon/tower-lsp

## Decision Matrix

| Option | LSP 3.17 coverage | Document sync | Testing support | Maturity / cadence | License | Editor compatibility | Performance risk | Dependency risk | Fit for `vibe-xpls` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `go-lsp` (`owenrumney/go-lsp`) | Strong; docs target 3.17 | Strong; `document.Store` handles full and incremental sync | Strong; `servertest`, debug UI, fuzz/examples | Recent but still pre-1.0; pkg.go.dev shows v0.2.2 published 2026-05-09 | MIT | Broad LSP editor compatibility over stdio/TCP-style transports | Low | Moderate | Best full Go framework, but more opinionated than the analyzer-first path |
| `glsp` (`tliron/glsp`) | Partial; package docs still use `protocol_3_16` and README says some features are not fully implemented | Basic; server and protocol layer are present, but state management is mostly up to the user | Weak/unclear in the docs | Older; pkg.go.dev shows v0.2.2 published 2024-03-09 | Apache-2.0 | Good transport breadth: stdio, TCP, WebSockets, Node.js IPC | Low | High | Not enough current protocol coverage for the first choice |
| `go.lsp.dev/protocol` + `go.lsp.dev/jsonrpc2` | Partial; protocol docs show `Version = "3.15.3"` and many 3.16-era features, but not a full 3.17 surface | Mixed; the types cover sync requests, but the document state machine is ours to build | Weak; no bundled LSP test harness, so integration tests must be custom | Stale-ish in release terms; protocol published 2022, jsonrpc2 published 2022 with newer package activity in 2025 | BSD-3-Clause + MIT | Excellent for any LSP editor because it stays on standard JSON-RPC framing | Low | Moderate | Best fit for analyzer-first/protocol-first because it keeps the adapter thin |
| Legacy Sourcegraph stack (`sourcegraph/go-lsp` + `sourcegraph/jsonrpc2`) | Weak as a framework choice; `sourcegraph/go-lsp` is archived and moved, while `jsonrpc2` is generic JSON-RPC | Weak; transport is generic and there is no modern LSP server harness in the stack | Weak; useful primitives, but no integrated LSP test story | High risk; `sourcegraph/go-lsp` is archived, and this is legacy code | MIT | Compatible in principle, but it is a legacy implementation path | Low | High | Reference only, not a recommended base |
| `microsoft/vscode-languageserver-node` | Strong; current line is 3.17.5 protocol, 9.0.1 client/server | Strong; mature document sync handling in the canonical JS/TS stack | Good; mature ecosystem, but not Go-native | Very mature and actively versioned | MIT | Best VS Code/editor breadth; effectively the reference JS/TS stack | Moderate | High | Credible alternative, but it adds a Node runtime and leaves the Go codebase |
| `tower-lsp` (Rust) | Strong; repo explicitly supports proposed 3.18 features | Strong; standard LSP server/client split | Good; idiomatic Rust testing story | Mature Rust repo; 591 commits and 30 tags on the current GitHub page | MIT + Apache-2.0 | Good stdio/TCP LSP compatibility | Low | High | Technically strong, but cross-language cost is too high for this repo |

## Prototype Criteria

- Prove the analyzer can stay independent of the transport by implementing one thin LSP adapter layer only.
- Cover `initialize`, `didOpen`, `didChange`, `didSave`, `hover`, `definition`, `references`, `rename`, `semanticTokens/full`, and `workspace/executeCommand` before adding anything editor-specific.
- Verify document synchronization in both full and incremental modes against fixture-backed tests.
- Add a harness that can exercise server handlers without a live editor, or prove that the chosen framework already gives us that with acceptable ergonomics.
- Measure cold start, incremental update latency, and memory growth on a realistic workspace so we can spot any framework-induced overhead.
- Confirm compatibility with at least one mainstream editor client and one agent-oriented client before widening scope.

## Recommendation

Use `go.lsp.dev/protocol` plus `go.lsp.dev/jsonrpc2` as the initial server substrate, with a small custom adapter around the analyzer core.

That choice best matches the approved direction: the analyzer owns the semantics, the protocol layer only translates requests and responses, and we avoid baking product behavior into a framework. Keep `go-lsp` as the fallback if the prototype shows we need a richer Go-native framework with built-in test tooling and debug UI. Do not start from `glsp` or the legacy Sourcegraph packages unless the current plan fails on missing protocol coverage.

## Confidence

Medium-high.

- High confidence that `go-lsp` is the strongest Go framework option if we decide we want an opinionated server harness.
- High confidence that `glsp` and the legacy Sourcegraph stack are weaker fits for the current work.
- Medium confidence that the thin `go.lsp.dev/protocol` + JSON-RPC path will remain the best long-term tradeoff, because the final answer depends on prototype friction and the exact editor behaviors we need.

## Evidence That Would Change This Recommendation

- A prototype shows that the thin protocol layer creates too much boilerplate or too many repeated adapter bugs.
- We need built-in LSP test harnesses and debug tooling immediately enough that the extra framework abstraction pays for itself.
- `go.lsp.dev/protocol` or `go.lsp.dev/jsonrpc2` proves too stale for the required 3.17 features or maintenance expectations.
- The editor mix turns out to be overwhelmingly Node- or Rust-based, making a non-Go stack operationally cheaper than keeping the server in Go.
- `go-lsp` demonstrates sustained 3.17+ maintenance and becomes the clear ergonomic winner for implementation speed.
