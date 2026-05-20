## Decision.

The first product boundary is a reusable analyzer library with two initial consumers: an LSP server for editor workflows and a structured JSON CLI for automation and agent workflows. Zed is the first showcase and replacement consumer, not the product boundary. MCP remains a later adapter after the analyzer contract, CLI output, and trust model are stable. Upbound `xpls` is a reference implementation and migration input, not a compatibility contract.

## Evidence.

- `docs/research/lanes/01-product-boundary.md` recommends an analyzer library with LSP and structured JSON CLI adapters first, with MCP staged after the CLI contract and execution trust model are proven.
- `docs/research/lanes/01-product-boundary.md` finds that LSP standardizes editor features, MCP standardizes tool transports, and the clean boundary is a transport-agnostic analyzer core rather than one editor or one validation command.
- `docs/research/lanes/09-existing-tooling.md` shows the `crossplane-yaml` extension can launch a language server for `Crossplane YAML` files and keep highlighting separate from server semantics, making Zed a concrete editor target.
- `docs/research/lanes/09-existing-tooling.md` explicitly says not to clone Upbound `xpls` as a strict contract, not to make the server Zed-only, and not to make the editor extension own Crossplane semantics.
- `docs/research/spikes/01-lsp-harness.md` proves a small Go LSP protocol loop can support diagnostics, hover, completion, document sync, and subprocess stdio framing while keeping semantics outside the handlers.
- `docs/research/spikes/05-render-validate.md` shows Crossplane render and validate are useful proof steps but are not suitable as the per-keystroke product boundary because they can depend on Docker, cache state, registry access, kubeconfig, and credential helpers.

## Alternatives Considered.

- Zed-centered product: attractive because the local extension is the clearest near-term showcase, but it overfits one editor and weakens CLI, VS Code, CI, and future agent use.
- Validation companion: faster to explain, but it overlaps with `crossplane beta validate` and does not cover navigation, completion, schema lookup, source mapping, or agent-safe semantic queries.
- Function-specific tool: useful for `function-go-templating` authors, but too narrow for workspaces that need XRDs, Compositions, provider CRDs, package metadata, and rendered resources understood together.
- Upbound `xpls` compatibility clone: useful for migration testing, but public capability detail is limited and the existing behavior should inform rather than define the new contract.

## Risks.

- A shared analyzer contract may take longer than a Zed-only replacement because it must serve editor, CLI, and later agent use cases.
- The structured JSON CLI could become a second protocol surface if it is designed independently from analyzer data types.
- Deferring MCP could delay agent-native adoption if agent workflows become the primary early demand.
- Treating Upbound `xpls` as reference only may miss edge cases current users rely on unless migration fixtures capture them.

## What Would Change This Decision.

- User research or telemetry shows the real first audience is overwhelmingly Zed-only.
- Early users primarily want offline validation and do not value navigation, completion, schema hover, source mapping, or agent-facing queries.
- Upbound exposes a stable reusable `xpls` library or protocol surface that already covers the target analyzer workflows.
- MCP clients become the dominant integration path before the CLI contract and command trust model are stable.
