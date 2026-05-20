# Crossplane LSP Research Program Design

**Status:** Approved design  
**Date:** 2026-05-12  
**Repository:** `<vibe-xpls-repo>`  
**Related repository:** `<crossplane-yaml-repo>`

## Goal

Design a research program that produces the evidence needed for a later, fresh brainstorming session for `vibe-xpls`, a Crossplane language-server project.

This spec is not the design for the language server itself. It defines what must be researched, what runnable spikes must prove, how decisions will be reviewed, and what artifacts must exist before the real Crossplane LSP brainstorming begins.

## Context

The `vibe-xpls` repository is a new Go-oriented repository with no product code yet. It has a public GitHub repository at `https://github.com/io41/vibe-xpls`.

The existing Zed extension at `<crossplane-yaml-repo>` currently starts `vibe-xpls serve` for Crossplane package worktrees. It defines a `Crossplane YAML` language, pins the `gotmpl` Tree-sitter grammar, keeps ordinary YAML untouched, and documents known limitations around mixed YAML plus Go-template highlighting. The future `vibe-xpls` project should be able to replace Upbound `xpls` in this extension, but existing Upbound `xpls` behavior is a reference and migration input, not a compatibility contract.

The research program must give equal priority to:

- Human Crossplane authors using editor features.
- AI coding agents that need structured Crossplane semantic operations.

The editor validation baseline is:

- Protocol-first LSP validation using test harnesses.
- Zed integration validation through `crossplane-yaml`.

## Non-Goals

- Do not implement `vibe-xpls` during this phase.
- Do not choose the final language-server framework during this phase.
- Do not commit to Go, Node, or any other final implementation stack before the research gates complete.
- Do not treat Upbound `xpls` as a compatibility contract.
- Do not design a complete public API for an agent semantic server in this phase.

## Research Program Shape

Use a dual-track evidence program.

The first track validates product shape: target users, workflows, editor needs, agent needs, and whether the correct first product is a general Crossplane LSP, a Zed-centered replacement, a validation companion, a function-specific tool, or a layered analyzer with multiple transports.

The second track validates technical risk: LSP framework choices, YAML/template parsing, schema indexing, Kubernetes tooling reuse, Crossplane render/validate integration, release automation, security, and Zed packaging.

The research program should avoid architecture-first reasoning. A Go LSP remains a plausible direction, but it must earn that position through evidence.

## Research Lanes

### Product Boundary

Determine whether the first real product should be:

- A general Crossplane language server.
- A Zed-centered integration through `crossplane-yaml`.
- A reusable analyzer library with LSP, CLI, and optional MCP/JSON-RPC transports.
- A validation companion focused on render, validate, trace, and diagnostics.
- A narrower function-specific tool, such as `function-go-templating` intelligence.

Output: a product-boundary recommendation with alternatives rejected and evidence cited.

### Human Editor UX

Define expected editor behavior for:

- Diagnostics.
- Completion.
- Hover.
- Go-to-definition.
- References.
- Code actions.
- Commands.
- Virtual rendered documents.

Validate protocol behavior and Zed behavior separately. Diagnostics, completion/hover, and navigation should be treated as the first editor workflows to evaluate. Render previews are Crossplane-specific and should be evaluated as a differentiator, not assumed into the first scope.

Output: editor workflow matrix with protocol requirements, Zed requirements, and fixture coverage.

### Agent Semantic API

Define structured operations for AI coding agents over the same analyzer used by the LSP. Candidate operations include:

- `list-compositions`.
- `explain-template`.
- `find-schema`.
- `render`.
- `validate-workspace`.
- `list-generated-resources`.
- `suggest-fix`.

The first concrete interface should be read-only structured JSON over CLI commands. MCP and JSON-RPC should be evaluated after the analyzer and CLI contracts are stable.

Output: agent workflow matrix, command-shape proposal, and boundaries between LSP and agent APIs.

### LSP Framework

Compare Go LSP framework choices and credible alternatives:

- `github.com/owenrumney/go-lsp`.
- `github.com/tliron/glsp`.
- `go.lsp.dev/protocol` with JSON-RPC.
- Legacy `sourcegraph` packages as negative or fallback references.
- Non-Go or editor-specific alternatives if evidence supports them.

Evaluate protocol coverage, document synchronization, testing support, maturity, maintenance cadence, licensing, editor compatibility, performance, and dependency risk.

Output: framework decision matrix plus a runnable harness spike report.

### YAML and Template Parsing

Evaluate how to handle:

- Plain YAML.
- Plain Go templates.
- YAML documents containing Go template actions.
- `function-go-templating` inline templates, filesystem templates, and environment templates.

Research and spike:

- `goccy/go-yaml`.
- `yaml.v3` or `yaml.v4`.
- Kubernetes YAML ingestion packages.
- Go `text/template` and `text/template/parse`.
- Sprig functions.
- Crossplane `function-go-templating` helpers.
- Masked YAML views.
- Rendered virtual YAML views.
- Offset and diagnostic mapping back to original files.
- Graceful degradation while users type incomplete YAML or templates.

Output: parser strategy recommendation with fixture-backed mapping results.

### Crossplane Semantics

Research and model:

- Pipeline mode Compositions.
- Legacy resources mode and migration relevance.
- Patch-and-transform.
- `function-go-templating`.
- `RunFunctionRequest` and `RunFunctionResponse`.
- `ExtraResources`.
- Context writes and reads.
- Special resources and annotations.
- Readiness behavior.
- Crossplane CLI `render`, `beta validate`, and `trace`.

Classify what belongs in fast local analysis and what belongs in optional authoritative validation.

Output: semantic capability map and fixture inventory.

### Schema and Workspace Indexing

Evaluate schema sources:

- Built-in Crossplane APIs.
- Workspace XRDs.
- Workspace Compositions.
- Provider CRDs.
- Package dependencies.
- Lock and revision metadata.
- User schema directories.
- Optional live-cluster discovery.
- Upbound Marketplace and model surfaces.

Use CRD OpenAPI/JSON Schema as the canonical source for YAML manifest intelligence. Treat Go model docs as useful for function authoring only after their source and freshness are verified.

Output: schema-source decision matrix and stale-schema handling recommendation.

### Kubernetes Language Intelligence

Research existing Kubernetes-focused language intelligence before duplicating it.

Include:

- `yaml-language-server` Kubernetes mode and schema association behavior.
- Kubernetes CRD schema catalogs.
- kubeconform or kubeval-style validation.
- VS Code Kubernetes tooling.
- Helm LS behavior where templates and Kubernetes schemas interact.
- Maintained Kubernetes-specific language servers or schema tools.

Answer:

- Which Kubernetes YAML capabilities are already solved well?
- How do existing tools handle CRDs, schema versioning, custom resources, and noisy diagnostics?
- Can Crossplane-specific intelligence layer on top of Kubernetes/YAML tooling?
- What remains unsolved once templates, XRD-derived APIs, provider CRDs, and `crossplane render` are involved?

Output: reuse/delegation recommendation and limits.

### Existing Tooling

Audit tools as references and falsification inputs:

- Upbound `xpls`.
- Upbound VS Code extension.
- Local `crossplane-yaml`.
- Red Hat YAML Language Server.
- Helm LS.
- Terraform LS.
- CUE tooling.
- KCL tooling.

The audit should capture current behavior, useful patterns, limitations, and what evidence would make a standalone `vibe-xpls` less attractive.

Output: existing-tooling matrix with replacement, reuse, and divergence notes.

### Release and Phase Gates

Research release tooling for a Go CLI/LSP project:

- `release-please`.
- `Changie`.
- `git-cliff`.
- GoReleaser.
- Conventional Commits and commit linting.

Public releases must stay on the `v0.X.X` line until maintainers explicitly approve leaving pre-1.0 after several months of real-world usage.

Every later implementation phase must end with runnable functional code. Documentation-only milestones are not valid implementation phase gates.

Output: release policy and dry-run recommendation.

### Security and Reliability

Review:

- Docker execution for `crossplane render`.
- Package and schema downloads.
- Optional cluster reads.
- Cache poisoning.
- Untrusted templates.
- Path traversal.
- Workspace trust.
- Agent tool permissions.
- Failure handling when external commands exit or panic.

Output: security and reliability risk register with required mitigations.

## Evidence Standards

Every research lane must produce a concise research note with:

- Sources, preferably primary sources.
- Confidence levels.
- Open questions.
- Recommendation.
- What evidence would change the recommendation.

Claims about current tools require one of:

- Primary source documentation.
- Repository/package source.
- A runnable local spike.
- A clearly labeled inference from sources.

The final synthesis must distinguish evidence from inference.

## Required Runnable Spikes

### LSP Harness Spike

Build a tiny language-server binary that supports:

- `initialize`.
- `textDocument/didOpen`.
- `textDocument/didChange`.
- `textDocument/didClose`.
- Diagnostics.
- Hover.
- Completion.
- `shutdown`.

It must run through protocol tests and be launchable as a local binary.

### Zed Integration Review

Verify that `crossplane-yaml` launches `vibe-xpls serve` without changing the extension's core model.

The spike must validate:

- Stdio launch.
- Worktree environment behavior.
- Crossplane package worktree handling.
- Zed logs on success and failure.
- No regression to current syntax highlighting and language classification assumptions.

### YAML and Template Mapping Spike

Parse mixed Crossplane YAML plus Go-template fixtures and prove:

- Source spans survive template masking or rendering.
- Template diagnostics map back to original positions.
- YAML diagnostics map back to original positions where possible.
- Structural template cases degrade predictably.

### Schema Index Spike

Index:

- One XRD.
- One Composition.
- One provider CRD or schema source.
- One package metadata file.

Demonstrate:

- `apiVersion/kind` lookup.
- Field documentation lookup.
- Stale schema invalidation behavior or a documented limit.

### Render and Validate Spike

Run `crossplane render` and, where useful, `crossplane beta validate` on fixture inputs.

Measure:

- Cold runtime.
- Warm runtime.
- Docker dependency behavior.
- Package/schema cache behavior.
- Failure modes.
- Diagnostic mapping limits.

### Agent API Spike

Expose read-only structured JSON operations over CLI first:

- `list-compositions`.
- `find-schema`.
- `validate-workspace`.
- `render`.

MCP and JSON-RPC are evaluated only after the CLI contract is stable.

### Kubernetes Tooling Spike

Compare at least one reuse or delegation path using existing Kubernetes/YAML tooling against a Go-native path.

Measure:

- What Kubernetes YAML behavior is already solved.
- What Crossplane behavior remains unsolved.
- Integration friction.
- Latency and diagnostic quality.

### Release Spike

Set up changelog and release tooling in dry-run mode.

The spike must produce:

- A `v0.X.X` release-note path.
- A release dry-run.
- A guard that rejects `v1.0.0` until the project explicitly exits pre-1.0.

## Decision Gates

### Gate 1: Product Boundary

Choose the real first product boundary. The decision must cite user workflows and spike results.

### Gate 2: Architecture Direction

Choose the likely core shape, such as analyzer-first with LSP, CLI, optional MCP/JSON-RPC, and render adapters. This remains provisional until the next brainstorming session, but must be strong enough to guide the real spec.

### Gate 3: Reuse vs Build

Decide what to reuse, delegate to, or avoid:

- Kubernetes/YAML tooling.
- Crossplane CLI.
- Upbound `xpls`.
- YAML LS.
- Helm LS patterns.
- CUE/KCL tooling.
- Go parser and framework libraries.

### Gate 4: Zed Integration Readiness

Confirm the LSP can be launched by `crossplane-yaml`, works over stdio, handles Crossplane package worktrees, and does not regress current syntax highlighting or language classification assumptions.

### Gate 5: Agent Surface

Decide whether the agent API is in the first implementation scope or a parallel follow-up.

If included, it must be structured JSON over stable analyzer interfaces, not scraped LSP behavior.

### Gate 6: Release Discipline

Confirm:

- Releases stay on `v0.X.X`.
- Changelog generation exists.
- Release dry-runs exist.
- Later development phases each produce runnable functional code.

### Gate 7: Go/No-Go

Proceed to the real Crossplane LSP brainstorming only if the research resolves enough unknowns to avoid designing from assumptions. If not, write a narrower follow-up research plan.

## Deliverables

Primary deliverable:

- `docs/research/crossplane-lsp-research-synthesis.md`

Supporting deliverables:

- `docs/research/lanes/` for one research note per lane.
- `docs/research/spikes/` for spike reports.
- `docs/research/decisions/` for decision matrices and gate records.
- `docs/research/fixtures.md` for fixture inventory.
- `docs/research/release-policy.md` for release and versioning policy.

The synthesis must include:

- Recommended product boundary.
- Recommended first implementation scope.
- Major alternatives considered and why they were rejected.
- Evidence table linking each recommendation to sources, spikes, or user/workflow findings.
- Open risks and what would change the recommendation.
- Inputs for the next brainstorming session.

## Acceptance Criteria

The research program is ready to turn into an implementation plan when:

- This design spec is committed.
- The implementation plan decomposes the research program into runnable, reviewable tasks.
- Each research lane has a named output artifact.
- Each required spike has an executable verification command.
- Zed replacement research explicitly uses `<crossplane-yaml-repo>`.
- Upbound `xpls` is treated as reference-only.
- Kubernetes language intelligence has its own lane.
- Human editor UX and agent semantic workflows are both first-class.

## References

- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/
- Crossplane CLI command reference: https://docs.crossplane.io/latest/cli/command-reference/
- Crossplane XRDs: https://docs.crossplane.io/latest/composition/composite-resource-definitions/
- function-go-templating: https://github.com/crossplane-contrib/function-go-templating
- LSP specification: https://microsoft.github.io/language-server-protocol/
- YAML Language Server: https://github.com/redhat-developer/yaml-language-server
- Helm LS: https://github.com/mrjosh/helm-ls
- Zed language extensions: https://zed.dev/docs/extensions/languages
- Zed extension development: https://zed.dev/docs/extensions/developing-extensions
- Upbound VS Code extension: https://marketplace.visualstudio.com/items?itemName=Upboundio.upbound
- Upbound `xpls` package: https://pkg.go.dev/github.com/upbound/up/cmd/up/xpls
- SemVer 2.0.0: https://semver.org/
