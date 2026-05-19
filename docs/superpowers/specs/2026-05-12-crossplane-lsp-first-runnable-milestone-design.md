# Crossplane LSP First Runnable Milestone Design

**Status:** Approved design
**Date:** 2026-05-12
**Repository:** `<vibe-xpls-repo>`
**Related repository:** `<zed-xpls-vibe-repo>`
**Research input:** `docs/research/crossplane-lsp-research-synthesis.md`

## Goal

Build the first runnable `vibe-xpls` product milestone: a Zed-first Crossplane authoring loop backed by a shared analyzer and a thin LSP adapter.

The milestone proves that a local `vibe-xpls` binary can be launched from the local `zed-xpls-vibe` validation extension and can provide useful Crossplane editor intelligence in realistic repository shapes. It does not start by defining a public agent API, executing Crossplane commands, or building a render/validate system.

Runnable means a `vibe-xpls` binary that Zed can launch as `<vibe-xpls-binary> serve` through `zed-xpls-vibe` and that satisfies the acceptance criteria in this document. If this milestone produces a public release, it ships as `v0.0.1` and remains on the `v0.X.X` line per `docs/research/decisions/gate-06-release-discipline.md`.

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
- The local `zed-xpls-vibe` dev extension launches `<vibe-xpls-binary> serve` as the first Zed launch path.

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

This intentionally narrows the final research synthesis, which recommended a first read-only JSON CLI for terminal and CI agents. The public agent CLI is deferred so the first runnable milestone can prove the unresolved Zed editor gate without also committing to agent UX, unsaved-overlay behavior, or a stable command contract. The internal debug CLI exists only to make analyzer behavior inspectable during implementation and tests.

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
- LSP position encoding negotiation, defaulting to UTF-16 and using UTF-8 when the client advertises support through LSP 3.17 `general.positionEncodings`.
- Conversion between protocol positions and analyzer byte offsets in the raw source.
- Formatting analyzer results into LSP diagnostics, hover responses, and completion items.
- Formatting YAML key completions with explicit LSP text edits when label-only insertion would be ambiguous. Completion labels stay concise, such as `spec` or `kind`, but accepted edits insert valid YAML key text at the analyzer-determined indentation, such as `spec:` or `    kind:`.

The analyzer's canonical source positions are byte offsets in raw document text. Rune indexes are an implementation detail of conversion routines, not part of the LSP contract.

Completion text edits are a correctness requirement for this milestone, not a snippet feature. The first milestone should not insert snippet placeholders, automatic child-line templates, or extra newlines. It should only replace the current YAML key prefix and indentation with the intended key text so accepting a completion preserves valid YAML structure in Zed.

The debug CLI is another adapter over the same analyzer. Its output may be JSON so tests can assert on it, but it is internal and non-contractual. It should help inspect package detection, diagnostics, schema lookup, hover, and completion against fixture paths without launching Zed.

The Zed extension remains thin. File classification and launcher configuration stay in `zed-xpls-vibe`; package/workspace detection and Crossplane semantics live in `vibe-xpls`.

## Data Flow

On startup, the LSP server initializes an analyzer workspace for the Zed worktree root. The analyzer classifies the repository shape as one of:

- Root package.
- Nested package.
- Multi-package workspace.
- No package root.

Zed starts the language server for files classified as `Crossplane YAML`. The validation extension does not require a root `crossplane.yaml` or root `upbound.yaml`; after launch, the analyzer scans the workspace for `crossplane.yaml` and `upbound.yaml` package markers. A document belongs to the nearest containing package root. Root and nested package facts are isolated by package scope; multi-package workspaces index all package roots but do not share workspace schema facts across package boundaries unless a later design introduces explicit shared schema directories.

For each opened document, the LSP server sends text changes into the analyzer with a monotonic document generation. The analyzer keeps:

- Raw document text.
- Parsed YAML view.
- Template parse view when the file contains Go template actions.
- Same-length masked YAML view for mixed YAML/template files.
- Schema context for the nearest package or workspace scope.
- Diagnostics tied to the document generation.

Async parse, index, and analysis results must be fenced by document and workspace generation. Document generation advances on each text change for that document. Workspace generation advances when package markers, XRDs, Compositions, provider CRDs, package metadata, or workspace-root detection change.

Diagnostics are push-model results: each publish operation is tied to an internal generation, and the server must not publish diagnostics older than the newest generation already known for that URI. `didClose` sends an empty `publishDiagnostics` notification for the closed URI. Hover and completion are pull-model results: requests run against a snapshot generation and either cancel, return empty, or return an explicit stale-result error if the originating generation is superseded before the response is produced.

Schema lookup is local and deterministic. The first milestone supports:

- Built-in Crossplane schemas shipped with the binary.
- Workspace XRDs.
- Workspace Compositions.
- Workspace provider CRDs.
- Workspace package metadata.

Workspace facts may augment built-ins by adding new kinds, package facts, and Crossplane graph relationships. They must not field-merge with an existing schema in this milestone.

Schema precedence is split by ownership:

- Crossplane core API groups shipped with `vibe-xpls`, including `apiextensions.crossplane.io`, `pkg.crossplane.io`, and `meta.pkg.crossplane.io`, use the built-in schema as authority. Workspace duplicates produce a bounded diagnostic on the duplicate schema file and debug CLI output rather than silently replacing the built-in schema.
- Provider and user-defined kinds use the workspace schema as authority because the built-in schema set does not define them.
- Conflicts between workspace schemas with the same group, version, and kind are reported on the conflicting schema files and in debug CLI output. The analyzer chooses a deterministic winner for continued operation, but the conflict remains visible.

Missing provider CRDs degrade to structural YAML and Crossplane graph analysis: the analyzer can still reason about `apiVersion`, `kind`, package context, and Composition shape, but it should not invent field completions or field-level diagnostics for unknown provider resources.

## Mixed YAML And Template Handling

Mixed YAML and Go-template files use source-mapped basics.

The analyzer should:

- Detect template spans.
- Produce a same-length masked YAML view.
- Parse template syntax enough to report real template diagnostics.
- Avoid misleading YAML diagnostics inside template actions.
- Preserve original source positions for diagnostics.
- Provide hover and completion in stable YAML structure outside template actions.

Stable YAML structure means an AST node whose key path from the document root contains no template-span bytes. A value that contains template spans can still receive syntax diagnostics, but it is not eligible for schema-aware hover or completion in this milestone.

The analyzer should not attempt schema-aware hover or completion inside generated template output in this milestone. `function-go-templating` context, helper intelligence, filesystem templates, emitted resources, and render-aware template semantics are later product slices.

## Error Handling And Trust

Default behavior is local, read-only, non-executing, and deterministic.

The milestone must not implicitly invoke external tools, Docker, network access, cluster reads, or workspace writes. If information is missing, the analyzer should produce fewer results or clear status rather than crossing a trust boundary.

Required error behavior:

- Malformed YAML produces source-mapped diagnostics without crashing the server.
- Unterminated template actions produce template diagnostics without flooding YAML diagnostics.
- No-root workspaces stay quiet unless an activation signal is present.
- Diagnostics are cleared when files close, become valid, or move to a newer generation.
- Large or invalid files are bounded so editor responsiveness is preserved.
- Analyzer panics are contained so one bad file cannot terminate the editor loop.

No-root activation is recomputed per edit and uses this signal order:

1. A package marker in the workspace root or an ancestor directory.
2. A filename that the Zed extension classifies as `Crossplane YAML`, including documented user `file_types` mappings.
3. A parseable `apiVersion` in the masked YAML view that belongs to a Crossplane core API group.
4. A parseable `kind` and shape for Composition, XRD, package metadata, or a CRD document that Zed has already classified as Crossplane YAML.

If an edit removes the activation signal, the server clears prior diagnostics for that URI and returns no hover or completion items until the signal returns.

Initial responsiveness limits are part of acceptance:

- Full document analysis is skipped or downgraded for a single document larger than 2 MiB.
- Diagnostics are capped at 100 per document.
- A single document analysis pass has a 500 ms soft deadline before returning partial results.
- Initial workspace indexing stops after 10,000 YAML-like files or 100 MiB of scanned YAML-like content and reports a bounded workspace-limit diagnostic or debug status.

These values are defaults for the first milestone and can be tuned by later evidence, but the implementation must have explicit limits and tests for the downgrade behavior.

Path handling must be explicit from the start. Workspace and package paths are resolved under the workspace root. In this milestone, traversal and symlink-escape rejection applies to workspace scanning, package-marker discovery, schema-file reads, and any package-relative file reads. Filesystem template expansion is out of scope. Diagnostics and debug output must not leak raw environment variables, credentials, kubeconfig data, registry credentials, or secret-bearing file content.

Executable trust UX is deferred because this milestone uses a developer-controlled local binary. Manual validation must record the canonical `<vibe-xpls-binary>` binary path used, but end-user executable approval and content-identity trust gates are a later design topic.

## Testing And Acceptance

Acceptance has three layers.

### Analyzer Fixture Tests

Analyzer tests cover core behavior without LSP or Zed:

- Package detection for root, nested, multi-package, and no-root repositories.
- Schema discovery from built-ins plus workspace XRDs, Compositions, provider CRDs, and package metadata.
- Schema precedence for Crossplane core duplicates, provider CRDs, user-defined XRD schemas, and conflicting workspace schemas.
- Diagnostics for valid YAML, invalid YAML, malformed YAML, unterminated templates, and mixed YAML/template files.
- Template span masking, eligible and ineligible stable YAML positions, and original-source diagnostic mapping.
- No-root activation toggling and diagnostic clearing when activation disappears.
- Hover at a known `apiVersion`, `kind`, or schema path returns indexed OpenAPI documentation when available and a clear absence when no documentation is indexed.
- Completion at known schema paths suggests indexed fields and does not invent field completions for unknown provider schemas.
- Completion context exposes enough source range information for the LSP adapter to replace the current YAML key prefix and indentation when accepting a completion.
- Document and workspace generation fencing.
- Stale diagnostic clearing behavior.
- Huge-document downgrade behavior.
- Workspace scan caps.
- Symlink escape and path traversal rejection during workspace scans and package-relative reads.
- Analyzer panic recovery for malformed fixtures.

External command timeout fixtures are not required in this milestone because normal editor behavior does not invoke external commands.

### LSP Protocol Tests

Protocol tests cover:

- Initialize and shutdown.
- Open, change, and close document synchronization.
- Publish diagnostics.
- Hover.
- Completion.
- Completion text edits for YAML keys, including root-key prefix replacement and nested-key indentation.
- UTF-16 default position handling and UTF-8 handling when negotiated with a supporting client.
- Stale diagnostic clearing.
- Pull-model stale hover and completion behavior.

### Manual Zed Validation

Manual Zed validation is required before the milestone counts as runnable. The validation uses the local `<zed-xpls-vibe-repo>` dev extension, which launches `<vibe-xpls-binary> serve`.

Required checks:

- The local binary is produced from the current worktree for validation, and the canonical `<vibe-xpls-binary>` path is recorded with the validation evidence.
- Zed launches the local `vibe-xpls` binary.
- Missing-binary behavior is understandable.
- A root package attaches.
- A nested package attaches.
- A multi-package workspace attaches without schema cross-contamination.
- A no-root workspace stays quiet.
- `.yaml` attach behavior is verified both without user `file_types` mappings and with the documented Crossplane `file_types` mapping. The mapping used during validation is recorded.
- Diagnostics appear and clear correctly.
- Hover works visibly.
- Completion works visibly.
- Accepting a completion inserts the completed YAML key at the correct indentation.
- Stale diagnostics do not survive valid edits or document close.

Tests alone are not sufficient. The research found Zed attach and UI behavior to be unresolved, so this manual checklist is a release gate for the milestone.

## Proof Levels

The milestone must label evidence by proof level:

- Fixture-backed analyzer evidence.
- Protocol-level LSP evidence.
- Manual Zed editor evidence.
- Real-workspace evidence, which is not required for this milestone but must be labeled separately if gathered.

Fixture-backed evidence must not be presented as production readiness. Manual Zed evidence is required for the first runnable claim.

## Open Risks

- Zed launch or attach behavior may differ from protocol harness behavior.
- Parser choice may affect comments, anchors, tags, duplicate keys, malformed input, source maps, and position conversion.
- Large provider CRD sets may expose indexing latency or memory issues.
- Local-only schema lookup may be weak when repositories do not check in provider CRDs or package metadata.
- Multi-package workspaces may reveal schema precedence and isolation edge cases.
- Mixed YAML/template source maps may become more complex when later milestones add filesystem templates or render-aware output.
- The internal debug CLI could accidentally become a public agent API unless documentation and naming keep it explicitly non-contractual.

Implementation planning must include an explicit parser-selection task before product code is written. `docs/research/lanes/05-yaml-template-parsing.md` makes `goccy/go-yaml` the starting candidate and `go.yaml.in/yaml/v4` a fallback, but this milestone design does not silently choose one.

## Acceptance Summary

The first runnable milestone is complete only when all of these are true:

- Analyzer fixture tests pass for package detection, schema lookup, schema precedence, diagnostics, hover, completion, mixed YAML/template basics, no-root activation, bounded-resource behavior, path safety, and stale generation behavior.
- LSP protocol tests pass for document sync, diagnostics, hover, completion, completion text edits, negotiated position conversion, stale diagnostic clearing, and stale pull-request behavior.
- Manual Zed validation passes through `zed-xpls-vibe` for root, nested, multi-package, and no-root workspaces.
- No external execution, Docker, downloads, cluster reads, kubeconfig reads, or workspace writes occur during normal editor behavior.
- The debug CLI remains internal and non-contractual.
