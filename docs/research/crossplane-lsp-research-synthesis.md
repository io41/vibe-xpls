# Crossplane LSP Research Synthesis

## Executive Summary

The research program supports a clear next step: proceed to the real Crossplane LSP brainstorming and product-design phase, but do not start production implementation yet.

The recommended shape is `vibe-xpls` as a Go-native Crossplane semantic analyzer with thin adapters for LSP and a structured JSON CLI. Zed is the first editor proof target through a thin local extension path; current first-runnable validation uses `<zed-xpls-vibe-repo>`. AI agents are first-class users through analyzer-backed operations, not cursor-oriented LSP scraping; first scope covers file-backed terminal and CI agents, while editor-embedded agents still need an unsaved-overlay model. Kubernetes and YAML tooling should be reused as schema, validation, and UX precedent, while Crossplane-specific semantics stay in the analyzer.

The strongest remaining blockers before production implementation are manual Zed UI validation, production parser selection, real workspace indexing performance, concrete trust UX, and broader user validation.

Proof level matters: the LSP harness, YAML/template mapping, schema index, and agent API are narrow runnable spikes, not production-proven implementations. They validate direction and risk, but the next design document must keep fixture-backed evidence separate from evidence gathered in real workspaces and editor sessions.

## Recommended Product Boundary

Build a shared analyzer library first, with:

- An LSP adapter for editor workflows.
- A structured JSON CLI adapter for agent and automation workflows.
- Optional MCP later, after analyzer contracts, CLI shapes, and trust gates are stable.

This follows `docs/research/decisions/gate-01-product-boundary.md`. Upbound `xpls` remains a reference and migration input, not a compatibility contract. Crossplane render and validate are proof paths, not the core product boundary.

## Recommended First Implementation Scope

The first runnable product phase should produce a small but real command and LSP loop around the analyzer:

- Workspace/package detection from Crossplane metadata.
- Local parsing of XRDs, Compositions, provider CRDs, package metadata, and mixed YAML/template files.
- Source-mapped diagnostics for syntax, template spans, schema references, function pipeline shape, and known Crossplane annotations.
- Completion and hover from local XRD/CRD OpenAPI schemas and Crossplane helper catalogs.
- Structured JSON CLI commands for disk-backed terminal/CI agents: `list-compositions`, `find-schema`, `validate-workspace`, and fixture-backed or trust-gated `render`.
- An explicit design choice for editor-agent unsaved overlays: LSP document state, JSON-RPC session state, or CLI overlay inputs.
- Zed launch through `zed-xpls-vibe` using `<vibe-xpls-binary> serve`, with manual validation of startup, file classification, root/nested/multi-package/no-root analyzer behavior, diagnostics, hover, completion, and stale diagnostic clearing.
- A Kubernetes/OpenAPI embeddable-library spike before committing to the production schema-index implementation.
- Release guard and changelog dry-run starting at `v0.0.1`.

Do not include live-cluster discovery, default Docker render, default package downloads, write-producing agent tools, or full rendered virtual documents in the first implementation scope.

## Evidence Table

| Topic | Decision | Primary Evidence |
| --- | --- | --- |
| Product boundary | Analyzer core with LSP and JSON CLI first; MCP later | `docs/research/decisions/gate-01-product-boundary.md`, `docs/research/lanes/01-product-boundary.md`, `docs/research/spikes/01-lsp-harness.md` |
| Architecture | Analyzer-first Go core, thin LSP adapter, explicit render/validate proof paths | `docs/research/decisions/gate-02-architecture-direction.md`, `docs/research/lanes/04-lsp-framework.md`, `docs/research/spikes/03-yaml-template-mapping.md`, `docs/research/spikes/05-render-validate.md` |
| Reuse vs build | Provisionally reuse Kubernetes/YAML concepts and optional validators; build Crossplane semantic core; still compare embeddable libraries | `docs/research/decisions/gate-03-reuse-vs-build.md`, `docs/research/lanes/08-kubernetes-language-intelligence.md`, `docs/research/spikes/07-kubernetes-tooling.md` |
| Zed | Code-path replacement is viable; manual UI and attach coverage remain gates | `docs/research/decisions/gate-04-zed-readiness.md`, `docs/research/spikes/02-zed-replacement.md` |
| Agents | File-backed JSON CLI first for terminal/CI agents; editor-agent overlays unresolved; MCP after contracts and trust gates stabilize | `docs/research/decisions/gate-05-agent-surface.md`, `docs/research/lanes/03-agent-semantic-api.md`, `docs/research/spikes/06-agent-api.md` |
| Release discipline | Start at `v0.0.1`, stay `v0.X.X`, use `git-cliff` first | `docs/research/decisions/gate-06-release-discipline.md`, `docs/research/lanes/10-release-phase-gates.md`, `docs/research/spikes/08-release.md` |
| Go/no-go | GO for next brainstorming/design; NO-GO for product implementation | `docs/research/decisions/gate-07-go-no-go.md` |

The evidence table names decisions, not production readiness. `docs/research/decisions/gate-07-go-no-go.md` is the controlling interpretation: use the evidence for the next brainstorming/design phase, then require stronger parser, Zed, indexing, trust, and user-validation evidence before product implementation.

## Alternatives Rejected

- Zed-only product: too narrow for CLI, future editors, CI, and agents.
- Generic YAML LSP clone: duplicates mature YAML tooling and still misses Crossplane graph semantics.
- Kubernetes validator as semantic core: useful for structural validation, but cannot model function pipelines, XRD-to-Composition links, or template context.
- Render-first analyzer: authoritative for function behavior, but Docker, cache, registry, kubeconfig, latency, and source-map limits make it unsuitable for hot paths.
- MCP-first surface: promising for agents, but premature before analyzer data contracts and execution trust gates are proven.
- Immediate production implementation: the spikes are deliberately narrow and do not yet settle production parser, Zed UI, indexing, performance, or trust decisions.

## Human Editor UX Findings

The first editor loop should prioritize diagnostics, completion, hover, and navigation. Render previews, virtual rendered documents, and code actions are valuable but should wait until source mapping and schema indexing are reliable.

Zed is the first real editor acceptance gate. The extension should remain thin: file classification, highlighting, and launcher configuration belong there; package/workspace detection and Crossplane semantics belong in the server and analyzer. The Zed gate must test attach coverage, not only process launch: root packages, nested packages, multi-package workspaces, repositories without root manifests, and behavior before and after documented `file_types` mappings.

## Agent Workflow Findings

Terminal and CI agents need repository-level operations over the Crossplane graph:

- List Compositions and pipeline steps.
- Resolve schemas and field documentation.
- Validate a workspace with structured diagnostics.
- Explain template context and helper functions.
- Run or simulate render with explicit trust and authority metadata.

The first agent surface should be a read-only JSON CLI with a stable envelope for disk-backed workflows. Editor-embedded agents need an overlay-aware model before full support can be claimed. The next design must decide whether unsaved state comes from LSP document state, a persistent JSON-RPC session, or explicit CLI overlay inputs, and it must preserve stable object IDs across edits.

## Zed Replacement Findings

The local Zed extension replacement path is viable as a code-path proof. The current validation fork is `<zed-xpls-vibe-repo>`; it uses extension id `zed-xpls-vibe`, launches `<vibe-xpls-binary> serve`, and leaves package/no-root detection to the `vibe-xpls` analyzer.

Manual UI validation is still required before product implementation can claim Zed readiness. Required checks include launcher behavior, missing-binary behavior, file classification, root and nested package detection, multi-package workspaces, documented `file_types` behavior, diagnostics, hover, completion, and stale diagnostic clearing. Any future executable trust approval must be tied to canonical path and executable identity rather than only a configurable command string.

## Kubernetes Reuse Findings

Kubernetes and YAML tools already solve important infrastructure problems: schema association, CRD/OpenAPI validation, field documentation patterns, strict validation modes, and CI-friendly outputs. `vibe-xpls` should reuse these ideas and optional tools, not rebuild them wholesale.

The Crossplane-specific value is the semantic graph: XRDs, Compositions, function pipelines, function-go-templating source maps, provider package discovery, render truth, special annotations/resources, and agent-safe operations. The exact schema implementation path is provisional until the custom fixture index is compared against embeddable Kubernetes/OpenAPI libraries such as CRD structural schema and OpenAPI validation packages.

## Technical Architecture Findings

The analyzer should maintain local, source-aware views:

- Raw text for document changes.
- Masked YAML/template structure for tolerant parsing and diagnostics.
- Schema graph from built-in Crossplane APIs, workspace XRDs, Compositions, provider CRDs, package metadata, and optional user schema directories.
- Rendered or externally validated views only for explicit commands.
- Monotonic document and workspace generations for async parse, index, render, and validation tasks so stale results cannot overwrite newer diagnostics.

Use static analysis for hot-path editor feedback. Use `crossplane render` and `crossplane beta validate` as explicit proof commands with timeouts and trust gates. Keep the LSP adapter replaceable.

## Release and Phase-Gate Findings

Start public releases at `v0.0.1` and keep every release on `v0.X.X` until maintainers approve a pre-1.0 exit after months of real-world usage.

The first changelog path should use Conventional Commits plus `git-cliff`. The version guard in `spikes/release/check-version.sh` rejects non-`v0` and malformed versions. GoReleaser belongs after there is a real binary to package, and release-please belongs after the manual release cadence and v0 policy are proven.

Every future implementation phase should leave runnable code and fresh verification evidence.

## Security and Reliability Findings

Default behavior must be local, read-only, non-executing, and deterministic. The analyzer must not implicitly invoke Docker, download packages, read a Kubernetes cluster, write workspace files, or expose raw environment data.

Privileged operations need explicit trust gates:

- Docker-backed render.
- Function development runtimes.
- Package or schema downloads.
- Live-cluster discovery.
- Agent-triggered execution or writes.
- Kubeconfig-touching CLI calls, because even apparent read-only probes can trigger `exec` auth plugins or ambient credential use.

Reliability and security acceptance criteria for the next design phase:

- Normalize all workspace and filesystem-template paths; reject traversal outside the workspace or package root and define symlink behavior.
- Resolve workspace and target paths through symlink evaluation, reject escapes after resolution, prefer no-follow or fd-relative opens where possible, and re-check target identity after opening.
- Store schema and package caches with provenance, immutable content identity, retrieval time, trust level, and an explicit refresh policy. Do not promote tag-only downloads to trusted cross-workspace cache entries.
- Sanitize diagnostics and JSON outputs so they do not expose raw environment variables, kubeconfig data, registry credentials, or secret-bearing file content.
- Apply timeouts and cancellation to parsing, indexing, external commands, downloads, and cluster calls.
- Bind trust grants to immutable subjects: workspace realpath, operation class, canonical executable path or image reference, content hash or digest, and configuration source. Invalidate grants when those subjects change.
- Add regression fixtures for malformed YAML, unterminated templates, template path traversal, symlink escapes, huge documents, stale diagnostics, async generation fencing, and external command timeouts.

Diagnostics should degrade when evidence is unavailable instead of silently crossing trust boundaries.

## Open Risks

- Manual Zed UI behavior may differ from subprocess harness behavior.
- Zed may still reveal attach or analyzer-context issues in common repository shapes until root detection, nested packages, multi-package workspaces, no-root workspaces, and `file_types` expectations are validated.
- Production parser selection may change source-map quality, comment preservation, duplicate-key handling, and LSP position conversion.
- Large provider CRD sets may expose indexing latency, memory, or cancellation problems.
- Local-first schema lookup may be weak when repositories do not check in provider CRDs or resolved package metadata.
- The schema-index implementation path may change after evaluating embeddable Kubernetes/OpenAPI libraries.
- Trust gates need concrete UX across CLI, Zed, and future MCP clients.
- Trust grants can become stale if they are not tied to canonical executable/image/config identity and invalidated when those subjects change.
- Fixture-backed render may mislead agents unless authority metadata stays visible and stable.
- Current parser, schema index, LSP, Zed, and agent API conclusions are spike-level evidence; production design must not present them as real-workspace proof until validated beyond fixtures.
- File-backed agent APIs can produce stale answers for editor-side agents unless unsaved overlays and multi-file draft state are modeled explicitly.
- User validation is still needed to confirm which Crossplane workflows matter most.

## Inputs for the Next Brainstorming Session

Use this research as the starting point for the real `vibe-xpls` brainstorming session:

- Define the first runnable product phase around the analyzer plus LSP and JSON CLI adapters.
- Decide the production parser candidate and required source-map tests.
- Define the minimum Zed manual validation script for `zed-xpls-vibe`, including attach coverage for root manifests, nested and multi-package repos, no-root-manifest repos, and `file_types` mappings.
- Specify schema source precedence, conflict reporting, cache policy, and the embeddable Kubernetes/OpenAPI library comparison.
- Define trust gates and user-visible status for render, validate, downloads, cluster discovery, kubeconfig `exec` auth, and executable/image identity.
- Preserve proof levels in the design document: label fixture-backed spikes, code-path proofs, manual editor evidence, real-workspace evidence, and production-ready conclusions separately.
- Add acceptance criteria for path normalization, workspace escape rejection, symlink policy, no-follow/open identity checks, cache provenance, immutable cache identity, cache refresh, output sanitization, timeout/cancellation, malformed-input resilience, async generation fencing, and stale-diagnostic clearing.
- Design the first agent API contract from the spike envelope, including whether editor-agent overlays use LSP state, JSON-RPC session state, or CLI overlay inputs.
- Decide the first `v0.0.1` changelog and release dry-run path.
- Convert open risks into acceptance criteria before product implementation begins.
