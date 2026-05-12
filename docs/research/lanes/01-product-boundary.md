# Product Boundary Research
## Summary
As of May 12, 2026, the best first-product boundary for `vibe-xpls` is a shared analyzer core with thin LSP, CLI, and MCP transports. That shape serves human editor workflows and AI agent workflows equally, while keeping Upbound `xpls` as a reference implementation rather than a contract.

The main reason is that Crossplane already exposes offline preview and validation flows, Zed already has strong editor and agent integration, and MCP already standardizes local and remote tool transports. The boundary should therefore be the semantic analyzer, not a single editor, a single validation command, or a single function niche.

## Sources
- Crossplane CLI command reference: https://docs.crossplane.io/latest/cli/command-reference/
- Crossplane compositions: https://docs.crossplane.io/latest/composition/compositions/
- Crossplane functions: https://docs.crossplane.io/latest/packages/functions/
- What is Crossplane?: https://docs.crossplane.io/latest/whats-crossplane/
- Zed AI overview: https://zed.dev/docs/ai/overview
- Zed configuring languages: https://zed.dev/docs/configuring-languages
- Zed MCP docs: https://zed.dev/docs/ai/mcp
- Zed MCP server extensions: https://zed.dev/docs/extensions/mcp-extensions
- Language Server Protocol official page: https://microsoft.github.io/language-server-protocol/
- MCP transports spec: https://modelcontextprotocol.io/specification/2025-06-18/basic/transports
- Upbound `xpls` package docs: https://pkg.go.dev/github.com/upbound/up@v0.34.2/internal/xpls

## Findings
- Evidence: `crossplane beta validate` validates compositions against provider or XRD schemas, validates `crossplane render` output, and runs offline without a Crossplane cluster.
- Inference: validation is important, but a standalone validation product would mostly duplicate an existing Crossplane CLI capability. It belongs in the core analyzer and CLI, not as the first product boundary.
- Evidence: `crossplane render` previews composition output locally, uses Docker by default, and supports a `Development` runtime for locally running functions.
- Inference: a function-specific tool is useful, but it is narrower than the actual problem space. The first product needs to understand XRDs, compositions, providers, rendered resources, and function pipelines together.
- Evidence: Crossplane composition functions are gRPC servers, and Crossplane calls them with `RunFunctionRequest` and `RunFunctionResponse`.
- Inference: function-aware behavior should be one analyzer capability, not the product boundary itself.
- Evidence: Zed's Agent Panel can read and write code, Zed can be extended through MCP servers, and Zed also supports external agents.
- Evidence: Zed supports configuring language servers, launching binaries from settings, and enabling or disabling language server support per language.
- Inference: a Zed-centered product would be attractive for Zed users, but it would overfit one editor and leave VS Code, CLI, and non-Zed MCP clients behind.
- Evidence: LSP standardizes editor-to-language-server communication for features like completion, go to definition, references, and hover; MCP standardizes stdio and Streamable HTTP transports for tools.
- Inference: the cleanest boundary is a transport-agnostic analyzer core with editor, CLI, and agent adapters on top.
- Evidence: Upbound `xpls` exposes a stdio JSON-RPC transport (`StdRWC`) and related server, dispatcher, and handler packages.
- Inference: treat `xpls` as a reference and migration input, not as the product boundary. It is a useful implementation clue, not the shape of the first product.

## Alternatives
- General Crossplane LSP: good fit for editor workflows, but too narrow if it excludes agent and CLI use cases, and too broad if it tries to absorb every analysis concern into the protocol layer.
- Zed-centered replacement: strong for Zed-native human and agent workflows, but it hard-codes the first market to one editor and weakens portability.
- Analyzer library with LSP/CLI/MCP transports: best fit for a shared semantic core, best match for both humans and agents, and easiest to reuse across future clients.
- Validation companion: quickest to ship, but it overlaps heavily with `crossplane beta validate` and does not materially expand the product boundary.
- Function-specific tool: valuable for composition-function authors, but too narrow to be the first product line.

## Recommendation
Build the analyzer library with LSP, CLI, and MCP transports as the first product boundary. Keep Zed integration as a consumer and showcase, not the core product. Keep validation as a first-class CLI and MCP capability. Keep function-specific helpers as one analyzer domain, not a separate product.

## Confidence
Medium-high. The source material is current and directly relevant, but the final product boundary still depends on user interviews and workflow telemetry that are not yet available.

## Evidence That Would Change This Recommendation
- If interviews or telemetry show that almost all real usage will be Zed-only, shift the boundary toward a Zed-centered product.
- If the primary demand is offline validation with little need for navigation or agent tooling, narrow the first product to a validation companion.
- If function authors prove to be the dominant user segment, move function-specific tooling earlier in the roadmap.
- If upstream Crossplane grows a stable semantic API that already covers most analysis needs, shrink the custom analyzer scope and focus on adapters.
