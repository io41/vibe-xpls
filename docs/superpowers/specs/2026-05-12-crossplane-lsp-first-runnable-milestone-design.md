# Crossplane LSP First Runnable Milestone Design

**Status:** Approved design
**Date:** 2026-05-12
**Repository:** `<vibe-xpls-repo>`
**Related repository:** `<zed-up-xpls-repo>`
**Research input:** `docs/research/crossplane-lsp-research-synthesis.md`

## Goal

Build the first runnable `vibe-xpls` product milestone: a Zed-first Crossplane authoring loop backed by a shared analyzer and a thin LSP adapter.

The milestone proves that a local `vibe-xpls` binary can be launched from the existing Zed extension path and can provide useful Crossplane editor intelligence in realistic repository shapes. It does not start by defining a public agent API, executing Crossplane commands, or building a render/validate system.

## Product Boundary

The first runnable milestone is a Zed-first editor milestone. It is accepted when Zed can launch the local `vibe-xpls` binary and receive useful language-server behavior for Crossplane workspaces.

Required editor features:

- Diagnostics.
- Hover.
- Completion.
- Stale diagnostic clearing.
- Package and workspace detection across root, nested, multi-package, and no-root repository shapes.

Required product shape:

- A shared analyzer core owns Crossplane semantics.
- A thin LSP adapter exposes diagnostics, hover, and completion.
- A debug-only CLI may inspect analyzer behavior for fixtures, CI, and local debugging.
- The existing `VIBE_XPLS_BIN` path in `<zed-up-xpls-repo>` is the first Zed launch path.

Out of scope for this milestone:

- Public JSON CLI contract for agents.
- MCP or JSON-RPC agent server.
- `crossplane render`.
- `crossplane beta validate`.
- Docker execution.
- Package or schema downloads.
- Live-cluster discovery.
- Kubeconfig-reading commands.
- Workspace writes.
- Code actions.
- Go-to-definition.
- Rendered virtual documents.
- Schema-aware intelligence inside generated template output.

## Architecture

Use an analyzer-first architecture with thin adapters.

The analyzer owns:

- Workspace and package-root detection.
- Document parsing.
- Raw-to-derived source mapping.
- Built-in and workspace schema indexing.
- Diagnostics.
- Hover facts.
- Completion candidates.
- Document and workspace generation tracking.

The LSP server handles transport concerns only:

- JSON-RPC framing.
- LSP lifecycle.
- Document synchronization.
- UTF-8, UTF-16, byte, and rune position conversion.
- Formatting analyzer results into LSP diagnostics, hover responses, and completion items.

The debug CLI is another adapter over the same analyzer. Its output may be JSON so tests can assert on it, but it is internal and non-contractual. It should help inspect package detection, diagnostics, schema lookup, hover, and completion against fixture paths without launching Zed.

The Zed extension remains thin. File classification, launcher configuration, and worktree integration stay in `zed-up-xpls`; Crossplane semantics live in `vibe-xpls`.

## Data Flow

On startup, the LSP server initializes an analyzer workspace for the Zed worktree root. The analyzer classifies the repository shape as one of:

- Root package.
- Nested package.
- Multi-package workspace.
- No package root.

For each opened document, the LSP server sends text changes into the analyzer with a monotonic document generation. The analyzer keeps:

- Raw document text.
- Parsed YAML view.
- Template parse view when the file contains Go template actions.
- Same-length masked YAML view for mixed YAML/template files.
- Schema context for the nearest package or workspace scope.
- Diagnostics tied to the document generation.

Async parse, index, and analysis results must be fenced by document and workspace generation. Stale results are dropped instead of overwriting newer diagnostics or hover/completion state.

Schema lookup is local and deterministic. The first milestone supports:

- Built-in Crossplane schemas shipped with the binary.
- Workspace XRDs.
- Workspace Compositions.
- Workspace provider CRDs.
- Workspace package metadata.

Workspace facts may augment built-ins. Within a package scope, a workspace schema for a matching group, version, and kind takes precedence over the shipped built-in schema so checked-in project state can model the user's actual Crossplane version. Missing provider CRDs or package metadata degrade gracefully by producing fewer completions or lower-confidence diagnostics. Conflicts between workspace schemas should be visible through bounded diagnostics or debug CLI output, but they must not block editor startup.

## Mixed YAML And Template Handling

Mixed YAML and Go-template files use source-mapped basics.

The analyzer should:

- Detect template spans.
- Produce a same-length masked YAML view.
- Parse template syntax enough to report real template diagnostics.
- Avoid misleading YAML diagnostics inside template actions.
- Preserve original source positions for diagnostics.
- Provide hover and completion in stable YAML structure outside template actions.

The analyzer should not attempt schema-aware hover or completion inside generated template output in this milestone. `function-go-templating` context, helper intelligence, filesystem templates, emitted resources, and render-aware template semantics are later product slices.

## Error Handling And Trust

Default behavior is local, read-only, non-executing, and deterministic.

The milestone must not implicitly invoke external tools, Docker, network access, cluster reads, or workspace writes. If information is missing, the analyzer should produce fewer results or clear status rather than crossing a trust boundary.

Required error behavior:

- Malformed YAML produces source-mapped diagnostics without crashing the server.
- Unterminated template actions produce template diagnostics without flooding YAML diagnostics.
- No-root workspaces stay quiet unless an opened document declares a Crossplane API group, a Composition, an XRD, a provider CRD, or package metadata.
- Diagnostics are cleared when files close, become valid, or move to a newer generation.
- Large or invalid files are bounded so editor responsiveness is preserved.
- Analyzer panics are contained so one bad file cannot terminate the editor loop.

Path handling must be explicit from the start. Workspace and package paths are resolved under the workspace root. Any feature that reads related files must reject path traversal and symlink escapes after path resolution. Diagnostics and debug output must not leak raw environment variables, credentials, kubeconfig data, registry credentials, or secret-bearing file content.

## Testing And Acceptance

Acceptance has three layers.

### Analyzer Fixture Tests

Analyzer tests cover core behavior without LSP or Zed:

- Package detection for root, nested, multi-package, and no-root repositories.
- Schema discovery from built-ins plus workspace XRDs, Compositions, provider CRDs, and package metadata.
- Diagnostics for valid YAML, invalid YAML, malformed YAML, and mixed YAML/template files.
- Template span masking and original-source diagnostic mapping.
- Hover over stable YAML structure.
- Completion over stable YAML structure.
- Document and workspace generation fencing.
- Stale diagnostic clearing behavior.

### LSP Protocol Tests

Protocol tests cover:

- Initialize and shutdown.
- Open, change, and close document synchronization.
- Publish diagnostics.
- Hover.
- Completion.
- UTF-8 and UTF-16 position conversion.
- Stale diagnostic clearing.

### Manual Zed Validation

Manual Zed validation is required before the milestone counts as runnable. The validation uses `<zed-up-xpls-repo>` and `VIBE_XPLS_BIN`.

Required checks:

- Zed launches the local `vibe-xpls` binary.
- Missing-binary behavior is understandable.
- A root package attaches.
- A nested package attaches.
- A multi-package workspace attaches without schema cross-contamination.
- A no-root workspace stays quiet.
- Diagnostics appear and clear correctly.
- Hover works visibly.
- Completion works visibly.
- Stale diagnostics do not survive valid edits or document close.

Tests alone are not sufficient. The research found Zed attach and UI behavior to be unresolved, so this manual checklist is a release gate for the milestone.

## Proof Levels

The milestone must label evidence by proof level:

- Fixture-backed analyzer evidence.
- Protocol-level LSP evidence.
- Manual Zed editor evidence.
- Real-workspace evidence, if gathered.

Fixture-backed evidence must not be presented as production readiness. Manual Zed evidence is required for the first runnable claim.

## Open Risks

- Zed launch or attach behavior may differ from protocol harness behavior.
- Parser choice may affect comments, anchors, tags, duplicate keys, malformed input, source maps, and position conversion.
- Large provider CRD sets may expose indexing latency or memory issues.
- Local-only schema lookup may be weak when repositories do not check in provider CRDs or package metadata.
- Multi-package workspaces may reveal schema precedence and isolation edge cases.
- Mixed YAML/template source maps may become more complex when later milestones add filesystem templates or render-aware output.
- The internal debug CLI could accidentally become a public agent API unless documentation and naming keep it explicitly non-contractual.

## Acceptance Summary

The first runnable milestone is complete only when all of these are true:

- Analyzer fixture tests pass for package detection, schema lookup, diagnostics, hover, completion, mixed YAML/template basics, and stale generation behavior.
- LSP protocol tests pass for document sync, diagnostics, hover, completion, position conversion, and stale diagnostic clearing.
- Manual Zed validation passes through `VIBE_XPLS_BIN` for root, nested, multi-package, and no-root workspaces.
- No external execution, Docker, downloads, cluster reads, kubeconfig reads, or workspace writes occur during normal editor behavior.
- The debug CLI remains internal and non-contractual.
