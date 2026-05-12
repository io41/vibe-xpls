# Existing Tooling Research

## Summary

As of 2026-05-12, existing tooling should shape `vibe-xpls` but not constrain it as a compatibility clone. Upbound `xpls` and the Upbound VS Code extension prove there is already demand for Crossplane package diagnostics. The local Zed extension proves a concrete replacement path: launch a language server for `Crossplane YAML` worktrees and keep editor-side highlighting separate from server-side semantics.

The strongest reuse lessons come from YAML Language Server, Helm LS, Terraform LS, CUE, and KCL: schema resolution, provider/version awareness, graceful degraded parsing, and domain-specific semantic layers matter more than a generic YAML parser alone.

## Sources

- Upbound `xpls` package: https://pkg.go.dev/github.com/upbound/up/cmd/up/xpls
- Upbound VS Code extension: https://marketplace.visualstudio.com/items?itemName=Upboundio.upbound
- Local Zed extension: `<zed-up-xpls-repo>`
- Local Zed README: `<zed-up-xpls-repo>/README.md`
- Local Zed extension code: `<zed-up-xpls-repo>/src/lib.rs`
- Local Zed extension manifest: `<zed-up-xpls-repo>/extension.toml`
- YAML Language Server: https://github.com/redhat-developer/yaml-language-server
- Helm LS: https://github.com/mrjosh/helm-ls
- Terraform LS: https://github.com/hashicorp/terraform-ls
- VS Code Terraform extension: https://github.com/hashicorp/vscode-terraform
- CUE LSP getting started: https://github.com/cue-lang/cue/wiki/LSP%3A-Getting-started
- KCL VS Code extension: https://marketplace.visualstudio.com/items?itemName=kcl.kcl-vscode-extension
- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/

## Local Zed Extension Findings

The local extension at `<zed-up-xpls-repo>` currently:

- Starts the language server with `up xpls serve --verbose`.
- Requires the Upbound `up` CLI on `PATH`.
- Detects Crossplane package worktrees by root `crossplane.yaml`.
- Detects Upbound project worktrees by root `upbound.yaml`.
- Defines a separate `Crossplane YAML` language.
- Attaches the `up-xpls` language server to `Crossplane YAML`.
- Uses the pinned `gotmpl` Tree-sitter grammar from `ngalaiko/tree-sitter-go-template`.
- Injects YAML highlighting into plain template text.
- Documents that ordinary YAML should remain on Zed's native YAML support.
- Documents that mixed YAML/template highlighting is best effort.
- Documents that stale diagnostics can remain when `up xpls` exits before publishing a clearing diagnostic set.

The current extension command contract is the most important replacement constraint. `vibe-xpls` should be able to expose an equivalent stdio command that the extension can launch without redesigning file classification or highlighting.

## Tool Matrix

| Tool | Strength | Limit for `vibe-xpls` |
| --- | --- | --- |
| Upbound `xpls` | Existing Crossplane LSP reference used by Upbound editor tooling | Reference only; current behavior is not a compatibility contract |
| Upbound VS Code extension | Proves thin editor-client model and Crossplane package diagnostics | VS Code-specific and sparse public capability detail |
| Local `zed-up-xpls` | Concrete Zed replacement target and Crossplane YAML highlighting layer | Delegates all semantics to `up xpls` today |
| YAML Language Server | Mature schema validation, completion, hover, schema associations, Kubernetes schema support | Does not understand Crossplane pipelines, XRD-derived context, or Go templates by itself |
| Helm LS | Useful precedent for template-aware Kubernetes/YAML workflows and delegation to YAML LS | Helm semantics differ from Crossplane `function-go-templating` |
| Terraform LS | Strong example of provider/version-aware infrastructure authoring | Terraform model does not map directly to Kubernetes resources or Crossplane functions |
| CUE tooling | Official LSP for CUE authoring and static evaluation | Should be integrated with, not reimplemented |
| KCL tooling | Existing language server/editor path for KCL authoring | Should be integrated with, not reimplemented |

## Reuse Opportunities

- Reuse YAML LS and Helm LS patterns for schema association, degraded parsing, and template-aware workflows.
- Reuse Terraform LS product lessons around provider/version-aware editing.
- Reuse CUE and KCL tooling by delegating language-specific intelligence where those functions are used.
- Reuse the local Zed extension's command-launch shape and language/highlighting split.
- Reuse Crossplane CLI behavior for authoritative render and validation when latency and environment constraints allow.

## Divergence Points

- Do not clone Upbound `xpls` behavior as a strict contract.
- Do not make the server Zed-only.
- Do not make the editor extension own Crossplane semantics.
- Do not assume a generic YAML LSP can model Crossplane pipeline function behavior.
- Do not reimplement CUE or KCL semantics inside `vibe-xpls`.

## Recommendation

Treat existing tooling as reference architecture and falsification input. The most practical near-term replacement target is the Zed command contract, while the most important design lesson is to keep Crossplane semantics in a shared analyzer that can power LSP, CLI, and later agent-facing adapters.

`vibe-xpls` should interoperate with Kubernetes/YAML tooling where it solves generic schema and YAML problems, then add Crossplane-specific semantic intelligence for XRDs, Compositions, function pipelines, templates, schemas, and render/validate workflows.

## Confidence

High that the local Zed extension provides a concrete replacement path.

High that YAML LS and Helm LS provide useful patterns but cannot fully solve Crossplane semantics.

Medium on Upbound `xpls` capability details because public documentation is limited; it still needs a capability audit.

## Evidence That Would Change This Recommendation

- Upbound `xpls` exposes a stable, reusable library or protocol surface that solves most target workflows.
- YAML LS plus a thin extension covers the top Crossplane workflows with less effort.
- Zed's extension API changes enough that the current command-launch model is no longer the right integration path.
- User research shows CUE, KCL, or another function-specific workflow dominates the expected audience.
