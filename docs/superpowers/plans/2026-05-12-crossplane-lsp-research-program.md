# Crossplane LSP Research Program Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Execute the approved research program so the next brainstorming session for `vibe-xpls` starts from evidence, runnable spikes, and reviewed decisions.

**Architecture:** The work is documentation-first but evidence-driven. Research lanes create structured notes under `docs/research/`, runnable spikes live under `spikes/`, and final decisions are recorded under `docs/research/decisions/`. The implementation deliberately keeps product code separate from spike code until the later real `vibe-xpls` design is approved.

**Tech Stack:** Markdown research artifacts, Go spike modules, local Crossplane CLI checks, Zed extension validation against `<zed-up-xpls-repo>`, GitHub/official documentation research, optional Docker for Crossplane render validation.

---

## File Structure

- Create: `docs/research/README.md` - research program index and execution rules.
- Create: `docs/research/lanes/01-product-boundary.md` - product-boundary research.
- Create: `docs/research/lanes/02-human-editor-ux.md` - editor workflow research.
- Create: `docs/research/lanes/03-agent-semantic-api.md` - agent workflow and API research.
- Create: `docs/research/lanes/04-lsp-framework.md` - LSP framework decision research.
- Create: `docs/research/lanes/05-yaml-template-parsing.md` - parser and source mapping research.
- Create: `docs/research/lanes/06-crossplane-semantics.md` - Crossplane semantic model research.
- Create: `docs/research/lanes/07-schema-workspace-indexing.md` - schema and indexing research.
- Create: `docs/research/lanes/08-kubernetes-language-intelligence.md` - Kubernetes tooling research.
- Create: `docs/research/lanes/09-existing-tooling.md` - Upbound, Zed, YAML LS, Helm LS, CUE, KCL, and Terraform LS audit.
- Create: `docs/research/lanes/10-release-phase-gates.md` - release and phase-gate research.
- Create: `docs/research/lanes/11-security-reliability.md` - security and reliability research.
- Create: `docs/research/spikes/01-lsp-harness.md` - LSP harness spike report.
- Create: `docs/research/spikes/02-zed-replacement.md` - Zed replacement spike report.
- Create: `docs/research/spikes/03-yaml-template-mapping.md` - YAML/template mapping spike report.
- Create: `docs/research/spikes/04-schema-index.md` - schema index spike report.
- Create: `docs/research/spikes/05-render-validate.md` - render/validate spike report.
- Create: `docs/research/spikes/06-agent-api.md` - agent API spike report.
- Create: `docs/research/spikes/07-kubernetes-tooling.md` - Kubernetes tooling spike report.
- Create: `docs/research/spikes/08-release.md` - release spike report.
- Create: `docs/research/decisions/gate-01-product-boundary.md` - product boundary decision.
- Create: `docs/research/decisions/gate-02-architecture-direction.md` - architecture direction decision.
- Create: `docs/research/decisions/gate-03-reuse-vs-build.md` - reuse/build decision.
- Create: `docs/research/decisions/gate-04-zed-readiness.md` - Zed replacement decision.
- Create: `docs/research/decisions/gate-05-agent-surface.md` - agent surface decision.
- Create: `docs/research/decisions/gate-06-release-discipline.md` - release discipline decision.
- Create: `docs/research/decisions/gate-07-go-no-go.md` - final go/no-go decision.
- Create: `docs/research/fixtures.md` - fixture inventory and coverage matrix.
- Create: `docs/research/release-policy.md` - pre-1.0 release policy.
- Create: `docs/research/crossplane-lsp-research-synthesis.md` - final synthesis.
- Create: `spikes/lsp-harness/` - minimal Go LSP server spike.
- Create: `spikes/yaml-template-mapping/` - source-mapping spike.
- Create: `spikes/schema-index/` - schema lookup spike.
- Create: `spikes/agent-api/` - structured JSON agent API spike.
- Create: `spikes/release/` - release tooling dry-run spike.

## Task 0: Baseline and Rules

**Files:**
- Read: `docs/superpowers/specs/2026-05-12-crossplane-lsp-research-program-design.md`
- Verify: repository state only.

- [ ] **Step 1: Confirm repository state**

Run:

```bash
git status --short --branch
```

Expected: output shows `## main` and no modified or untracked files.

- [ ] **Step 2: Confirm local author identity**

Run:

```bash
git config --local --get user.name
git config --local --get user.email
```

Expected output:

```text
Tim Kersten
tim@io41.com
```

- [ ] **Step 3: Read the approved design**

Run:

```bash
sed -n '1,520p' docs/superpowers/specs/2026-05-12-crossplane-lsp-research-program-design.md
```

Expected: the output includes the research lanes, required runnable spikes, decision gates, and deliverables.

## Task 1: Research Artifact Scaffold

**Files:**
- Create: `docs/research/README.md`
- Create: `docs/research/fixtures.md`
- Create: `docs/research/release-policy.md`
- Create directories: `docs/research/lanes/`, `docs/research/spikes/`, `docs/research/decisions/`

- [ ] **Step 1: Create directories**

Run:

```bash
mkdir -p docs/research/lanes docs/research/spikes docs/research/decisions
```

Expected: command exits successfully.

- [ ] **Step 2: Create the research index**

Write `docs/research/README.md`:

```markdown
# Crossplane LSP Research Program

This directory contains the research notes, runnable spike reports, decision records, fixture inventory, release policy, and final synthesis required before the real `vibe-xpls` brainstorming session.

The research program gives equal weight to human Crossplane authors and AI coding agents. It validates protocol-level LSP behavior and Zed integration through `<zed-up-xpls-repo>`.

## Execution Rules

- Treat Upbound `xpls` as a reference and migration input, not as a compatibility contract.
- Distinguish evidence from inference in every research note.
- Prefer primary sources and runnable spikes.
- Record confidence levels and what evidence would change each recommendation.
- Keep runnable spike code under `spikes/`, separate from future product code.
- Keep public releases on `v0.X.X` until maintainers approve leaving pre-1.0.

## Lane Notes

- `lanes/01-product-boundary.md`
- `lanes/02-human-editor-ux.md`
- `lanes/03-agent-semantic-api.md`
- `lanes/04-lsp-framework.md`
- `lanes/05-yaml-template-parsing.md`
- `lanes/06-crossplane-semantics.md`
- `lanes/07-schema-workspace-indexing.md`
- `lanes/08-kubernetes-language-intelligence.md`
- `lanes/09-existing-tooling.md`
- `lanes/10-release-phase-gates.md`
- `lanes/11-security-reliability.md`

## Spike Reports

- `spikes/01-lsp-harness.md`
- `spikes/02-zed-replacement.md`
- `spikes/03-yaml-template-mapping.md`
- `spikes/04-schema-index.md`
- `spikes/05-render-validate.md`
- `spikes/06-agent-api.md`
- `spikes/07-kubernetes-tooling.md`
- `spikes/08-release.md`

## Decision Records

- `decisions/gate-01-product-boundary.md`
- `decisions/gate-02-architecture-direction.md`
- `decisions/gate-03-reuse-vs-build.md`
- `decisions/gate-04-zed-readiness.md`
- `decisions/gate-05-agent-surface.md`
- `decisions/gate-06-release-discipline.md`
- `decisions/gate-07-go-no-go.md`
```

- [ ] **Step 3: Create fixture inventory seed**

Write `docs/research/fixtures.md`:

```markdown
# Fixture Inventory

This file tracks the fixtures used by research lanes and runnable spikes.

## Required Fixture Coverage

| Fixture | Purpose | Required By |
| --- | --- | --- |
| Minimal XRD | XRD schema parsing, hover, completion, and validation | schema index, editor UX |
| Invalid XRD | Diagnostic quality and source mapping | editor UX, Crossplane semantics |
| Pipeline Composition | Function pipeline parsing and step navigation | Crossplane semantics, LSP harness |
| `function-go-templating` inline template | Mixed YAML/template parsing and source mapping | YAML/template mapping |
| `function-go-templating` filesystem template | Template path resolution and go-to-definition | YAML/template mapping, editor UX |
| Render input XR | `crossplane render` validation | render/validate |
| Provider CRD | Provider resource schema indexing | schema index, Kubernetes tooling |
| Package metadata | Crossplane package detection and dependency graph | schema index, Zed replacement |
| Ordinary Kubernetes YAML | Regression guard for non-Crossplane YAML behavior | Kubernetes tooling, Zed replacement |

## Fixture Sources

- Local `vibe-xpls` spike fixtures under `spikes/**/testdata/`.
- Existing Zed extension fixtures under `<zed-up-xpls-repo>/fixtures/`.
- Official Crossplane documentation examples when license and attribution permit.
```

- [ ] **Step 4: Create release policy seed**

Write `docs/research/release-policy.md`:

```markdown
# Release Policy

`vibe-xpls` starts at `v0.0.1`.

Public releases must remain on the `v0.X.X` line until maintainers explicitly approve leaving pre-1.0 after several months of real-world usage.

## Pre-1.0 Exit Criteria

`v1.0.0` is blocked until all of these are true:

- At least 90 days of real-world use.
- A documented public API and CLI surface.
- One release cycle with no breaking CLI, LSP, or agent API changes.
- A maintainer-approved migration note.
- An explicit decision record under `docs/research/decisions/`.

## Release Guard

Release automation must reject tags that do not match:

```text
^v0\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$
```

The only exception is a maintainer-approved exit-from-v0 decision.
```

- [ ] **Step 5: Verify scaffold**

Run:

```bash
find docs/research -maxdepth 3 -type f | sort
```

Expected output includes:

```text
docs/research/README.md
docs/research/fixtures.md
docs/research/release-policy.md
```

- [ ] **Step 6: Commit scaffold**

Run:

```bash
git add docs/research
git commit -m "docs: scaffold research program artifacts"
```

Expected: a commit is created.

## Task 2: Product and Workflow Research Wave

**Files:**
- Create: `docs/research/lanes/01-product-boundary.md`
- Create: `docs/research/lanes/02-human-editor-ux.md`
- Create: `docs/research/lanes/03-agent-semantic-api.md`
- Create: `docs/research/lanes/09-existing-tooling.md`

- [ ] **Step 1: Dispatch product-boundary research**

Use a research subagent with this prompt:

```text
Research product boundaries for `vibe-xpls`, a future Crossplane language-server project. Treat Upbound `xpls` as a reference only. Evaluate these first-product shapes: general Crossplane LSP, Zed-centered replacement, analyzer library with LSP/CLI/MCP transports, validation companion, and function-specific tool. Consider human editor workflows and AI agent workflows equally. Use current primary sources where possible and cite URLs. Write the final note to docs/research/lanes/01-product-boundary.md with sections: Summary, Sources, Findings, Alternatives, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

Expected: `docs/research/lanes/01-product-boundary.md` exists and contains cited findings.

- [ ] **Step 2: Dispatch human-editor UX research**

Use a research subagent with this prompt:

```text
Research human editor UX requirements for `vibe-xpls`. Evaluate diagnostics, completion, hover, go-to-definition, references, code actions, commands, virtual rendered documents, and noisy diagnostic controls. Include protocol-first LSP validation and Zed integration through `<zed-up-xpls-repo>`. Compare YAML LS, Helm LS, Terraform LS, CUE tooling, KCL tooling, Upbound VS Code extension, and current Zed extension behavior. Use primary sources and cite URLs. Write the final note to docs/research/lanes/02-human-editor-ux.md with sections: Summary, Sources, Feature Priority, Zed Requirements, Protocol Requirements, Fixtures, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

Expected: `docs/research/lanes/02-human-editor-ux.md` exists and contains feature priorities plus Zed requirements.

- [ ] **Step 3: Dispatch agent semantic API research**

Use a research subagent with this prompt:

```text
Research agent-facing semantic APIs for `vibe-xpls`. Evaluate why AI agents need graph-shaped Crossplane operations beyond cursor-shaped LSP methods. Compare CLI structured JSON, JSON-RPC, MCP, and internal analyzer library adapters. Candidate operations include list-compositions, explain-template, find-schema, render, validate-workspace, list-generated-resources, and suggest-fix. Use primary sources for LSP, MCP, Crossplane CLI, and comparable code-intelligence tools. Write the final note to docs/research/lanes/03-agent-semantic-api.md with sections: Summary, Sources, Operations, Interface Options, Security Boundaries, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

Expected: `docs/research/lanes/03-agent-semantic-api.md` exists and distinguishes CLI, JSON-RPC, and MCP roles.

- [ ] **Step 4: Dispatch existing-tooling audit**

Use a research subagent with this prompt:

```text
Audit existing tooling relevant to `vibe-xpls`: Upbound `xpls`, Upbound VS Code extension, `<zed-up-xpls-repo>`, Red Hat YAML Language Server, Helm LS, Terraform LS, CUE tooling, and KCL tooling. Treat Upbound `xpls` as a reference only, not a compatibility contract. For local Zed files, inspect README.md, extension.toml, src/lib.rs, docs/superpowers/specs, docs/superpowers/plans, languages/crossplane-yaml, and fixtures. Write the final note to docs/research/lanes/09-existing-tooling.md with sections: Summary, Sources, Local Zed Extension Findings, Tool Matrix, Reuse Opportunities, Divergence Points, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

Expected: `docs/research/lanes/09-existing-tooling.md` exists and includes the current Zed extension command path `up xpls serve --verbose`.

- [ ] **Step 5: Verify wave outputs**

Run:

```bash
for f in docs/research/lanes/01-product-boundary.md docs/research/lanes/02-human-editor-ux.md docs/research/lanes/03-agent-semantic-api.md docs/research/lanes/09-existing-tooling.md; do test -s "$f" || exit 1; done
rg -n "Evidence That Would Change|Recommendation|Confidence" docs/research/lanes/01-product-boundary.md docs/research/lanes/02-human-editor-ux.md docs/research/lanes/03-agent-semantic-api.md docs/research/lanes/09-existing-tooling.md
```

Expected: both commands exit successfully.

- [ ] **Step 6: Commit product and workflow research**

Run:

```bash
git add docs/research/lanes/01-product-boundary.md docs/research/lanes/02-human-editor-ux.md docs/research/lanes/03-agent-semantic-api.md docs/research/lanes/09-existing-tooling.md
git commit -m "docs: add product and workflow research lanes"
```

Expected: a commit is created.

## Task 3: Technical Research Wave

**Files:**
- Create: `docs/research/lanes/04-lsp-framework.md`
- Create: `docs/research/lanes/05-yaml-template-parsing.md`
- Create: `docs/research/lanes/06-crossplane-semantics.md`
- Create: `docs/research/lanes/07-schema-workspace-indexing.md`
- Create: `docs/research/lanes/08-kubernetes-language-intelligence.md`
- Create: `docs/research/lanes/10-release-phase-gates.md`
- Create: `docs/research/lanes/11-security-reliability.md`

- [ ] **Step 1: Dispatch LSP framework research**

Use a research subagent with this prompt:

```text
Research LSP framework choices for `vibe-xpls`. Compare `github.com/owenrumney/go-lsp`, `github.com/tliron/glsp`, `go.lsp.dev/protocol` with JSON-RPC, legacy Sourcegraph packages, and credible non-Go alternatives. Evaluate LSP 3.17 coverage, document synchronization, testing support, maturity, maintenance cadence, licensing, editor compatibility, performance risk, and dependency risk. Use primary sources and cite URLs. Write the final note to docs/research/lanes/04-lsp-framework.md with sections: Summary, Sources, Decision Matrix, Prototype Criteria, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 2: Dispatch YAML/template parsing research**

Use a research subagent with this prompt:

```text
Research YAML and Go-template parsing for `vibe-xpls`. Compare `goccy/go-yaml`, `yaml.v3` or `yaml.v4`, Kubernetes YAML ingestion packages, Go `text/template` and `text/template/parse`, Sprig, and Crossplane `function-go-templating` helpers. Evaluate mixed YAML/template strategies including masking, rendered views, tolerant parsing, source-span mapping, and degraded behavior during incomplete edits. Use primary sources and cite URLs. Write the final note to docs/research/lanes/05-yaml-template-parsing.md with sections: Summary, Sources, Parser Matrix, Mapping Risks, Prototype Criteria, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 3: Dispatch Crossplane semantics research**

Use a research subagent with this prompt:

```text
Research Crossplane semantics for `vibe-xpls`. Cover pipeline-mode Compositions, legacy resources mode, patch-and-transform, function-go-templating, RunFunctionRequest and RunFunctionResponse, ExtraResources, context writes and reads, special resources and annotations, readiness, Crossplane CLI render, beta validate, and trace. Classify fast static analysis versus optional authoritative validation. Use official Crossplane sources where possible and cite URLs. Write the final note to docs/research/lanes/06-crossplane-semantics.md with sections: Summary, Sources, Semantic Model, Static Analysis Candidates, Authoritative Validation Candidates, Fixture Needs, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 4: Dispatch schema/indexing research**

Use a research subagent with this prompt:

```text
Research schema and workspace indexing for `vibe-xpls`. Evaluate built-in Crossplane APIs, workspace XRDs, Compositions, provider CRDs, package dependencies, lock/revision metadata, user schema directories, optional live-cluster discovery, and Upbound Marketplace/model surfaces. Treat CRD OpenAPI/JSON Schema as the likely canonical source for YAML intelligence unless evidence says otherwise. Use primary sources and cite URLs. Write the final note to docs/research/lanes/07-schema-workspace-indexing.md with sections: Summary, Sources, Schema Source Matrix, Freshness Strategy, Completion/Hover Strategy, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 5: Dispatch Kubernetes language-intelligence research**

Use a research subagent with this prompt:

```text
Research Kubernetes language intelligence for `vibe-xpls`. Include YAML Language Server Kubernetes mode, CRD schema catalogs, kubeconform or kubeval-style validation, VS Code Kubernetes tooling, Helm LS interactions, and maintained Kubernetes-specific language servers or schema tools. Identify what Crossplane can reuse, delegate to, interoperate with, or avoid duplicating. Use primary sources and cite URLs. Write the final note to docs/research/lanes/08-kubernetes-language-intelligence.md with sections: Summary, Sources, Capability Matrix, Reuse Options, Limits for Crossplane, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 6: Dispatch release and phase-gate research**

Use a research subagent with this prompt:

```text
Research release and phase-gate tooling for `vibe-xpls`. Compare release-please, Changie, git-cliff, GoReleaser, Conventional Commits, and commit linting. Recommend how to start at v0.0.1, stay on v0.X.X, generate changelogs, run release dry-runs, and enforce runnable functional code at every later implementation phase. Use primary sources and cite URLs. Write the final note to docs/research/lanes/10-release-phase-gates.md with sections: Summary, Sources, Tool Matrix, v0 Policy, Phase Gates, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 7: Dispatch security and reliability research**

Use a security-reviewer subagent with this prompt:

```text
Research security and reliability risks for `vibe-xpls`. Review Docker execution for crossplane render, package/schema downloads, optional cluster reads, cache poisoning, untrusted templates, path traversal, workspace trust, agent tool permissions, external command exits, and language-server process crashes. Recommend mitigations and review gates. Use primary sources where possible and cite URLs. Write the final note to docs/research/lanes/11-security-reliability.md with sections: Summary, Sources, Risk Register, Required Mitigations, Review Gates, Recommendation, Confidence, Evidence That Would Change This Recommendation.
```

- [ ] **Step 8: Verify technical wave outputs**

Run:

```bash
for f in docs/research/lanes/04-lsp-framework.md docs/research/lanes/05-yaml-template-parsing.md docs/research/lanes/06-crossplane-semantics.md docs/research/lanes/07-schema-workspace-indexing.md docs/research/lanes/08-kubernetes-language-intelligence.md docs/research/lanes/10-release-phase-gates.md docs/research/lanes/11-security-reliability.md; do test -s "$f" || exit 1; done
rg -n "Recommendation|Confidence|Evidence That Would Change" docs/research/lanes/04-lsp-framework.md docs/research/lanes/05-yaml-template-parsing.md docs/research/lanes/06-crossplane-semantics.md docs/research/lanes/07-schema-workspace-indexing.md docs/research/lanes/08-kubernetes-language-intelligence.md docs/research/lanes/10-release-phase-gates.md docs/research/lanes/11-security-reliability.md
```

Expected: both commands exit successfully.

- [ ] **Step 9: Commit technical research**

Run:

```bash
git add docs/research/lanes
git commit -m "docs: add technical research lanes"
```

Expected: a commit is created.

## Task 4: LSP Harness Spike

**Files:**
- Create: `spikes/lsp-harness/go.mod`
- Create: `spikes/lsp-harness/main.go`
- Create: `spikes/lsp-harness/main_test.go`
- Create: `docs/research/spikes/01-lsp-harness.md`

- [ ] **Step 1: Create the spike directory**

Run:

```bash
mkdir -p spikes/lsp-harness
```

Expected: command exits successfully.

- [ ] **Step 2: Write tests before implementation**

Create `spikes/lsp-harness/main_test.go` with tests that:

- Send `initialize` and expect a response with `capabilities`.
- Send `textDocument/didOpen` and expect a `textDocument/publishDiagnostics` notification.
- Send `textDocument/hover` and expect a hover response.
- Send `textDocument/completion` and expect completion items.
- Send `shutdown` and expect a response.

Run:

```bash
cd spikes/lsp-harness && go test ./...
```

Expected: tests fail because `main.go` is not implemented yet.

- [ ] **Step 3: Implement the minimal LSP harness**

Create `spikes/lsp-harness/go.mod`:

```go
module github.com/io41/vibe-xpls/spikes/lsp-harness

go 1.23
```

Create `spikes/lsp-harness/main.go` with a minimal stdio LSP server that:

- Reads and writes `Content-Length` framed JSON-RPC messages.
- Handles `initialize`.
- Handles `textDocument/didOpen`, `textDocument/didChange`, and `textDocument/didClose`.
- Publishes a deterministic diagnostic for documents containing the string `xpls-spike-error`.
- Handles `textDocument/hover`.
- Handles `textDocument/completion`.
- Handles `shutdown`.
- Handles `exit`.

- [ ] **Step 4: Verify the spike**

Run:

```bash
cd spikes/lsp-harness && go test ./...
```

Expected: all tests pass.

- [ ] **Step 5: Write the spike report**

Write `docs/research/spikes/01-lsp-harness.md` with sections:

- Summary.
- Commands Run.
- Protocol Features Proven.
- Limitations.
- Evidence for Later Decisions.

The report must include the successful `go test ./...` command and the tested LSP methods.

- [ ] **Step 6: Commit LSP harness spike**

Run:

```bash
git add spikes/lsp-harness docs/research/spikes/01-lsp-harness.md
git commit -m "test: add lsp harness spike"
```

Expected: a commit is created.

## Task 5: Zed Replacement Spike

**Files:**
- Create: `docs/research/spikes/02-zed-replacement.md`
- External read/write during spike: `<zed-up-xpls-repo>`

- [ ] **Step 1: Record current Zed extension state**

Run:

```bash
git -C <zed-up-xpls-repo> status --short --branch
sed -n '1,220p' <zed-up-xpls-repo>/README.md
sed -n '1,220p' <zed-up-xpls-repo>/extension.toml
sed -n '1,260p' <zed-up-xpls-repo>/src/lib.rs
```

Expected: the status output is recorded in the spike report, and source output shows the current `up xpls serve --verbose` command path.

- [ ] **Step 2: Add a local override in a temporary Zed branch**

In `<zed-up-xpls-repo>`, create a temporary branch named `vibe-xpls-spike`. Modify `src/lib.rs` so `language_server_command` uses the `VIBE_XPLS_BIN` environment variable when it is present and otherwise keeps the current `up xpls serve --verbose` behavior.

Run:

```bash
git -C <zed-up-xpls-repo> switch -c vibe-xpls-spike
```

Expected: branch switch succeeds.

- [ ] **Step 3: Verify Zed extension tests**

Run:

```bash
cd <zed-up-xpls-repo> && cargo fmt --check
cd <zed-up-xpls-repo> && cargo test
cd <zed-up-xpls-repo> && PATH="<rustup-bin-dir>:$PATH" cargo build --target wasm32-wasip2
```

Expected: all commands exit successfully.

- [ ] **Step 4: Launch Zed manually with the spike binary**

Run:

```bash
cd <zed-up-xpls-repo> && VIBE_XPLS_BIN=<vibe-xpls-repo>/spikes/lsp-harness/lsp-harness zed --foreground .
```

Expected: Zed starts and logs show the language server command path. Manual verification records whether diagnostics, hover, and completion are visible in the fixture worktree.

- [ ] **Step 5: Write the spike report**

Write `docs/research/spikes/02-zed-replacement.md` with sections:

- Summary.
- Current Extension Contract.
- Temporary Branch or Diff.
- Commands Run.
- Manual Zed Result.
- Compatibility Findings.
- Decision Impact.

The report must state that Upbound `xpls` is reference-only and that the replacement target is the Zed extension command contract.

- [ ] **Step 6: Return the external Zed repository to its original branch**

Run:

```bash
git -C <zed-up-xpls-repo> status --short --branch
```

Expected: any uncommitted Zed changes are either committed on the temporary branch or explicitly recorded in the spike report before switching away.

- [ ] **Step 7: Commit Zed spike report in `vibe-xpls`**

Run:

```bash
git add docs/research/spikes/02-zed-replacement.md
git commit -m "docs: add zed replacement spike report"
```

Expected: a commit is created in `vibe-xpls`.

## Task 6: Parsing, Schema, and Agent API Spikes

**Files:**
- Create: `spikes/yaml-template-mapping/`
- Create: `spikes/schema-index/`
- Create: `spikes/agent-api/`
- Create: `docs/research/spikes/03-yaml-template-mapping.md`
- Create: `docs/research/spikes/04-schema-index.md`
- Create: `docs/research/spikes/06-agent-api.md`

- [ ] **Step 1: Build the YAML/template mapping spike**

Create a Go spike under `spikes/yaml-template-mapping` that:

- Contains fixtures with template actions in scalar values, list items, block scalars, keys, and multi-document output.
- Parses template actions and records source spans.
- Produces a masked YAML view.
- Parses the masked YAML view.
- Tests diagnostic mapping back to original source positions.

Run:

```bash
cd spikes/yaml-template-mapping && go test ./...
```

Expected: tests pass and prove at least one mapped diagnostic outside a template span and one template diagnostic inside a template span.

- [ ] **Step 2: Write the YAML/template mapping report**

Write `docs/research/spikes/03-yaml-template-mapping.md` with sections:

- Summary.
- Fixture Cases.
- Commands Run.
- Mapping Results.
- Degradation Rules.
- Decision Impact.

- [ ] **Step 3: Build the schema index spike**

Create a Go spike under `spikes/schema-index` that:

- Indexes one XRD fixture.
- Indexes one Composition fixture.
- Indexes one provider CRD or schema fixture.
- Indexes one package metadata fixture.
- Exposes lookup functions for `apiVersion/kind` and field documentation.

Run:

```bash
cd spikes/schema-index && go test ./...
```

Expected: tests pass and prove `apiVersion/kind` lookup plus field documentation lookup.

- [ ] **Step 4: Write the schema index report**

Write `docs/research/spikes/04-schema-index.md` with sections:

- Summary.
- Indexed Sources.
- Commands Run.
- Lookup Results.
- Freshness Limits.
- Decision Impact.

- [ ] **Step 5: Build the agent API spike**

Create a Go spike under `spikes/agent-api` that exposes read-only JSON commands:

- `list-compositions`.
- `find-schema`.
- `validate-workspace`.
- `render`.

The `render` command may call the LSP harness or return a fixture-backed render result if the Crossplane CLI is unavailable. It must return structured JSON and include an `ok` boolean.

Run:

```bash
cd spikes/agent-api && go test ./...
```

Expected: tests pass and each command returns valid JSON.

- [ ] **Step 6: Write the agent API report**

Write `docs/research/spikes/06-agent-api.md` with sections:

- Summary.
- Commands.
- JSON Contracts.
- Commands Run.
- Security Boundaries.
- Decision Impact.

- [ ] **Step 7: Commit parsing, schema, and agent spikes**

Run:

```bash
git add spikes/yaml-template-mapping spikes/schema-index spikes/agent-api docs/research/spikes/03-yaml-template-mapping.md docs/research/spikes/04-schema-index.md docs/research/spikes/06-agent-api.md
git commit -m "test: add parsing schema and agent api spikes"
```

Expected: a commit is created.

## Task 7: Kubernetes, Render, and Release Spikes

**Files:**
- Create: `docs/research/spikes/05-render-validate.md`
- Create: `docs/research/spikes/07-kubernetes-tooling.md`
- Create: `docs/research/spikes/08-release.md`
- Create: `spikes/release/`

- [ ] **Step 1: Run render and validate measurements**

Check tool availability:

```bash
command -v crossplane
command -v docker
```

If both exist, run a render/validate fixture using `crossplane render` and `crossplane beta validate`. If either command is missing, record the missing command and run a fixture-only dry path that validates the spike report format.

Write `docs/research/spikes/05-render-validate.md` with sections:

- Summary.
- Tool Availability.
- Commands Run.
- Cold Runtime.
- Warm Runtime.
- Docker Behavior.
- Cache Behavior.
- Diagnostic Mapping Limits.
- Decision Impact.

- [ ] **Step 2: Run Kubernetes tooling comparison**

Compare at least one existing Kubernetes/YAML validation path with a Go-native or fixture-backed path.

Candidate commands:

```bash
npx yaml-language-server --help
npx kubeconform -h
```

If a command is unavailable or network access fails, record the failure and rely on official documentation plus any locally available tool.

Write `docs/research/spikes/07-kubernetes-tooling.md` with sections:

- Summary.
- Tools Evaluated.
- Commands Run.
- Capabilities Already Solved.
- Crossplane Gaps.
- Reuse Recommendation.
- Decision Impact.

- [ ] **Step 3: Build release dry-run spike**

Create `spikes/release/check-version.sh`:

```sh
#!/bin/sh
set -eu

version="${1:-}"
case "$version" in
  v0.[0-9]*.[0-9]*|v0.[0-9]*.[0-9]*-*)
    exit 0
    ;;
  *)
    echo "release version must stay on v0.X.X before explicit pre-1.0 exit approval" >&2
    exit 1
    ;;
esac
```

Run:

```bash
chmod +x spikes/release/check-version.sh
spikes/release/check-version.sh v0.0.1
! spikes/release/check-version.sh v1.0.0
```

Expected: `v0.0.1` passes and `v1.0.0` fails.

- [ ] **Step 4: Write release spike report**

Write `docs/research/spikes/08-release.md` with sections:

- Summary.
- Tooling Compared.
- Commands Run.
- v0 Guard Result.
- Changelog Recommendation.
- Release Dry-Run Recommendation.
- Decision Impact.

- [ ] **Step 5: Commit Kubernetes, render, and release spikes**

Run:

```bash
git add docs/research/spikes/05-render-validate.md docs/research/spikes/07-kubernetes-tooling.md docs/research/spikes/08-release.md spikes/release
git commit -m "test: add render kubernetes and release spikes"
```

Expected: a commit is created.

## Task 8: Decision Gates and Synthesis

**Files:**
- Create: `docs/research/decisions/gate-01-product-boundary.md`
- Create: `docs/research/decisions/gate-02-architecture-direction.md`
- Create: `docs/research/decisions/gate-03-reuse-vs-build.md`
- Create: `docs/research/decisions/gate-04-zed-readiness.md`
- Create: `docs/research/decisions/gate-05-agent-surface.md`
- Create: `docs/research/decisions/gate-06-release-discipline.md`
- Create: `docs/research/decisions/gate-07-go-no-go.md`
- Create: `docs/research/crossplane-lsp-research-synthesis.md`

- [ ] **Step 1: Write decision records**

For each gate file, write sections:

- Decision.
- Evidence.
- Alternatives Considered.
- Risks.
- What Would Change This Decision.

Use evidence from lane notes and spike reports.

- [ ] **Step 2: Write final synthesis**

Write `docs/research/crossplane-lsp-research-synthesis.md` with sections:

- Executive Summary.
- Recommended Product Boundary.
- Recommended First Implementation Scope.
- Evidence Table.
- Alternatives Rejected.
- Human Editor UX Findings.
- Agent Workflow Findings.
- Zed Replacement Findings.
- Kubernetes Reuse Findings.
- Technical Architecture Findings.
- Release and Phase-Gate Findings.
- Security and Reliability Findings.
- Open Risks.
- Inputs for the Next Brainstorming Session.

- [ ] **Step 3: Verify synthesis references all gates**

Run:

```bash
for f in docs/research/decisions/gate-01-product-boundary.md docs/research/decisions/gate-02-architecture-direction.md docs/research/decisions/gate-03-reuse-vs-build.md docs/research/decisions/gate-04-zed-readiness.md docs/research/decisions/gate-05-agent-surface.md docs/research/decisions/gate-06-release-discipline.md docs/research/decisions/gate-07-go-no-go.md docs/research/crossplane-lsp-research-synthesis.md; do test -s "$f" || exit 1; done
rg -n "Recommended Product Boundary|Evidence Table|Inputs for the Next Brainstorming Session" docs/research/crossplane-lsp-research-synthesis.md
```

Expected: both commands exit successfully.

- [ ] **Step 4: Commit decision records and synthesis**

Run:

```bash
git add docs/research/decisions docs/research/crossplane-lsp-research-synthesis.md
git commit -m "docs: synthesize crossplane lsp research"
```

Expected: a commit is created.

## Task 9: Independent Review

**Files:**
- Modify: research artifacts if review findings require changes.

- [ ] **Step 1: Request critical review**

Use a critic subagent with this prompt:

```text
Review the completed `docs/research` program for product-boundary mistakes, weak evidence, hidden assumptions, missing Kubernetes reuse options, missing Zed replacement risks, missing agent workflow risks, and unjustified architecture conclusions. Return findings ordered by severity with file references and concrete fixes.
```

- [ ] **Step 2: Request security review**

Use a security-reviewer subagent with this prompt:

```text
Review `docs/research/lanes/11-security-reliability.md`, all spike reports, and the synthesis for security and reliability gaps. Focus on Docker execution, package downloads, cluster reads, untrusted templates, path traversal, workspace trust, cache poisoning, and agent tool permissions. Return findings ordered by severity with file references and concrete fixes.
```

- [ ] **Step 3: Request verification review**

Use a verifier subagent with this prompt:

```text
Verify that the research program satisfies `docs/superpowers/specs/2026-05-12-crossplane-lsp-research-program-design.md`. Check every lane, spike, decision gate, Zed requirement, Kubernetes lane, Upbound reference-only rule, agent workflow requirement, and release v0.X.X rule. Return missing items with file references.
```

- [ ] **Step 4: Apply review fixes**

For each Critical or Important finding, update the relevant research artifact. Minor findings may be fixed or recorded under Open Risks in the synthesis.

- [ ] **Step 5: Commit review fixes**

Run:

```bash
git add docs/research spikes
git commit -m "docs: address research review findings"
```

Expected: a commit is created if review fixes changed files. If no files changed, record the no-change review outcome in the final response.

## Task 10: Final Verification

**Files:**
- Verify all research artifacts and spike code.

- [ ] **Step 1: Run markdown red-flag checks**

Run:

```bash
rg -n "TB""D|TO""DO|FIX""ME|imple""ment later|\\?\\?\\?" docs/research docs/superpowers/plans docs/superpowers/specs
```

Expected: command exits with status 1 because no matches are found.

Run:

```bash
git diff --check
```

Expected: command exits successfully with no output.

- [ ] **Step 2: Run spike tests**

Run:

```bash
cd spikes/lsp-harness && go test ./...
cd ../../spikes/yaml-template-mapping && go test ./...
cd ../../spikes/schema-index && go test ./...
cd ../../spikes/agent-api && go test ./...
cd ../.. && spikes/release/check-version.sh v0.0.1
cd <vibe-xpls-repo> && ! spikes/release/check-version.sh v1.0.0
```

Expected: all commands behave as described in their spike reports.

- [ ] **Step 3: Verify required files exist**

Run:

```bash
for f in docs/research/crossplane-lsp-research-synthesis.md docs/research/fixtures.md docs/research/release-policy.md docs/research/decisions/gate-07-go-no-go.md; do test -s "$f" || exit 1; done
find docs/research/lanes -type f | wc -l
find docs/research/spikes -type f | wc -l
```

Expected: file checks pass, lane count is at least `11`, and spike report count is at least `8`.

- [ ] **Step 4: Verify git state**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected: status is clean and recent commits show scaffold, research lanes, spikes, decisions, and review fixes.

- [ ] **Step 5: Handoff**

Report:

- Research synthesis path.
- Decision gate outcomes.
- Spike commands that passed.
- Review findings fixed or accepted.
- Whether the next step is a fresh brainstorming session for the real `vibe-xpls` design.
