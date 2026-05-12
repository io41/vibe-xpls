# Crossplane LSP Research Synthesis

## Executive Summary

The research program supports a clear next step: proceed to the real Crossplane LSP brainstorming and product-design phase, but do not start production implementation yet.

The recommended shape is `vibe-xpls` as a Go-native Crossplane semantic analyzer with thin adapters for LSP and a structured JSON CLI. Zed is the first editor proof target through the existing `<zed-up-xpls-repo>` extension, but not the product boundary. AI agents are first-class users through analyzer-backed repository operations, not cursor-oriented LSP scraping. Kubernetes and YAML tooling should be reused as schema, validation, and UX precedent, while Crossplane-specific semantics stay in the analyzer.

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
- Structured JSON CLI commands for `list-compositions`, `find-schema`, `validate-workspace`, and fixture-backed or trust-gated `render`.
- Zed launch through `VIBE_XPLS_BIN` with manual validation of startup, diagnostics, hover, completion, and stale diagnostic clearing.
- Release guard and changelog dry-run starting at `v0.0.1`.

Do not include live-cluster discovery, default Docker render, default package downloads, write-producing agent tools, or full rendered virtual documents in the first implementation scope.

## Evidence Table

| Topic | Decision | Primary Evidence |
| --- | --- | --- |
| Product boundary | Analyzer core with LSP and JSON CLI first; MCP later | `docs/research/decisions/gate-01-product-boundary.md`, `docs/research/lanes/01-product-boundary.md`, `docs/research/spikes/01-lsp-harness.md` |
| Architecture | Analyzer-first Go core, thin LSP adapter, explicit render/validate proof paths | `docs/research/decisions/gate-02-architecture-direction.md`, `docs/research/lanes/04-lsp-framework.md`, `docs/research/spikes/03-yaml-template-mapping.md`, `docs/research/spikes/05-render-validate.md` |
| Reuse vs build | Reuse Kubernetes/YAML concepts and optional validators; build Crossplane semantic core | `docs/research/decisions/gate-03-reuse-vs-build.md`, `docs/research/lanes/08-kubernetes-language-intelligence.md`, `docs/research/spikes/07-kubernetes-tooling.md` |
| Zed | Code-path replacement is viable; manual Zed UI remains a gate | `docs/research/decisions/gate-04-zed-readiness.md`, `docs/research/spikes/02-zed-replacement.md` |
| Agents | Structured JSON CLI first; MCP after contracts and trust gates stabilize | `docs/research/decisions/gate-05-agent-surface.md`, `docs/research/lanes/03-agent-semantic-api.md`, `docs/research/spikes/06-agent-api.md` |
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

Zed is the first real editor acceptance gate. The extension should remain thin: file classification, highlighting, launcher configuration, and worktree detection belong there; Crossplane semantics belong in the server and analyzer.

## Agent Workflow Findings

Agents need repository-level operations over the Crossplane graph:

- List Compositions and pipeline steps.
- Resolve schemas and field documentation.
- Validate a workspace with structured diagnostics.
- Explain template context and helper functions.
- Run or simulate render with explicit trust and authority metadata.

The first agent surface should be a read-only JSON CLI with a stable envelope. LSP remains the editor protocol, and MCP is a later adapter.

## Zed Replacement Findings

The local Zed extension replacement path is viable as a code-path proof. The external repo `<zed-up-xpls-repo>` has branch `vibe-xpls-spike` at commit `ac1d8cb feat: allow vibe xpls binary override`, adding `VIBE_XPLS_BIN` while preserving the Upbound fallback.

Manual UI validation is still required before product implementation can claim Zed readiness. Required checks include launcher environment propagation, missing-binary behavior, diagnostics, hover, completion, and stale diagnostic clearing.

## Kubernetes Reuse Findings

Kubernetes and YAML tools already solve important infrastructure problems: schema association, CRD/OpenAPI validation, field documentation patterns, strict validation modes, and CI-friendly outputs. `vibe-xpls` should reuse these ideas and optional tools, not rebuild them wholesale.

The Crossplane-specific value is the semantic graph: XRDs, Compositions, function pipelines, function-go-templating source maps, provider package discovery, render truth, special annotations/resources, and agent-safe operations.

## Technical Architecture Findings

The analyzer should maintain local, source-aware views:

- Raw text for document changes.
- Masked YAML/template structure for tolerant parsing and diagnostics.
- Schema graph from built-in Crossplane APIs, workspace XRDs, Compositions, provider CRDs, package metadata, and optional user schema directories.
- Rendered or externally validated views only for explicit commands.

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

Reliability and security acceptance criteria for the next design phase:

- Normalize all workspace and filesystem-template paths; reject traversal outside the workspace or package root and define symlink behavior.
- Store schema and package caches with provenance, content identity where available, retrieval time, trust level, and an explicit refresh policy.
- Sanitize diagnostics and JSON outputs so they do not expose raw environment variables, kubeconfig data, registry credentials, or secret-bearing file content.
- Apply timeouts and cancellation to parsing, indexing, external commands, downloads, and cluster calls.
- Add regression fixtures for malformed YAML, unterminated templates, template path traversal, huge documents, stale diagnostics, and external command timeouts.

Diagnostics should degrade when evidence is unavailable instead of silently crossing trust boundaries.

## Open Risks

- Manual Zed UI behavior may differ from subprocess harness behavior.
- Production parser selection may change source-map quality, comment preservation, duplicate-key handling, and LSP position conversion.
- Large provider CRD sets may expose indexing latency, memory, or cancellation problems.
- Local-first schema lookup may be weak when repositories do not check in provider CRDs or resolved package metadata.
- Trust gates need concrete UX across CLI, Zed, and future MCP clients.
- Fixture-backed render may mislead agents unless authority metadata stays visible and stable.
- Current parser, schema index, LSP, Zed, and agent API conclusions are spike-level evidence; production design must not present them as real-workspace proof until validated beyond fixtures.
- User validation is still needed to confirm which Crossplane workflows matter most.

## Inputs for the Next Brainstorming Session

Use this research as the starting point for the real `vibe-xpls` brainstorming session:

- Define the first runnable product phase around the analyzer plus LSP and JSON CLI adapters.
- Decide the production parser candidate and required source-map tests.
- Define the minimum Zed manual validation script for `VIBE_XPLS_BIN`.
- Specify schema source precedence, conflict reporting, and cache policy.
- Define trust gates and user-visible status for render, validate, downloads, and cluster discovery.
- Preserve proof levels in the design document: label fixture-backed spikes, code-path proofs, manual editor evidence, real-workspace evidence, and production-ready conclusions separately.
- Add acceptance criteria for path normalization, workspace escape rejection, symlink policy, cache provenance, cache refresh, output sanitization, timeout/cancellation, malformed-input resilience, and stale-diagnostic clearing.
- Design the first agent API contract from the spike envelope.
- Decide the first `v0.0.1` changelog and release dry-run path.
- Convert open risks into acceptance criteria before product implementation begins.
