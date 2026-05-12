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
