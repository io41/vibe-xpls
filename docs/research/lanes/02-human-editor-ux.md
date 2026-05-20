# Human Editor UX Research

## Summary

The first human-editor goal for `vibe-xpls` should be a protocol-first LSP experience that is also proven in Zed through the `crossplane-yaml` extension.

The most valuable initial UX is diagnostics, completion, hover, and navigation for Crossplane package authors. Render previews and code actions are important differentiators, but they should be gated by reliable parsing, schema lookup, and source mapping.

## Sources

- Language Server Protocol 3.17: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
- Zed language extensions: https://zed.dev/docs/extensions/languages
- Zed extension development: https://zed.dev/docs/extensions/developing-extensions
- YAML Language Server: https://github.com/redhat-developer/yaml-language-server
- Helm LS: https://github.com/mrjosh/helm-ls
- Terraform LS: https://github.com/hashicorp/terraform-ls
- VS Code Terraform extension: https://github.com/hashicorp/vscode-terraform
- CUE LSP getting started: https://github.com/cue-lang/cue/wiki/LSP%3A-Getting-started
- KCL VS Code extension: https://marketplace.visualstudio.com/items?itemName=kcl.kcl-vscode-extension
- Upbound VS Code extension: https://marketplace.visualstudio.com/items?itemName=Upboundio.upbound
- `crossplane-yaml` Zed extension: `<crossplane-yaml-repo>`

## Feature Priority

1. Diagnostics. Crossplane failures often show up late through package validation, function execution, schema validation, or rendered resources. Fast local diagnostics should cover syntax, obvious semantic mistakes, stale schema references, unresolved package/workspace references, and template parse errors.
2. Completion. The editor should complete `apiVersion`, `kind`, Crossplane package fields, XRD-derived XR fields, function input fields, template root objects, and known template helper functions.
3. Hover. Hover should explain schema fields, Crossplane annotations, function input fields, template helpers, and special resources such as `ExtraResources` and context writes.
4. Go-to-definition and references. These are high value because Crossplane authors move between XRDs, Compositions, function inputs, template files, package metadata, and provider schemas.
5. Commands. Initial commands should focus on `render`, `validate`, schema refresh, and opening rendered output.
6. Virtual rendered documents. Rendered resources are a strong Crossplane-specific feature, but they require reliable source mapping and should not be treated as a prerequisite for baseline editing.
7. Code actions. Code actions should come after diagnostics are trustworthy. Good early candidates are adding missing annotations, creating fixture files, or inserting known helper functions.

## Zed Requirements

The Zed extension must remain a thin launcher and language integration layer. The language server should own Crossplane semantics.

The `crossplane-yaml` extension:

- Defines a `Crossplane YAML` language.
- Uses the pinned `gotmpl` Tree-sitter grammar.
- Injects YAML highlighting into plain template text.
- Keeps ordinary YAML on Zed's native YAML language.
- Starts `vibe-xpls serve`.
- Resolves or manages a pinned `vibe-xpls` installation by default.
- Supports user override through `lsp.crossplane-yaml.binary.path`.
- Leaves Crossplane package detection to the `vibe-xpls` analyzer.

`vibe-xpls` should keep the command path simple without requiring the extension to redesign syntax highlighting or file classification. Zed validation must cover successful startup, missing-binary failure, diagnostics, completion, hover, and stale diagnostic clearing.

Known Zed constraints from the local extension:

- `path_suffixes` works for exact filenames such as `crossplane.yaml`, but broad glob-style matching requires user `file_types`.
- `first_line_pattern` cannot reliably override built-in YAML suffix matching for normal `.yaml` files.
- The language server attach path depends on file classification, while package-root interpretation belongs to the analyzer. Repositories without root `crossplane.yaml` or `upbound.yaml`, nested packages, and multi-package workspaces need explicit validation before the integration can be called production-ready.
- Broad Crossplane `.yaml` coverage may require documented user `file_types` mappings unless the extension grows better classification.
- Mixed YAML/template highlighting is best effort and should not be coupled to semantic validation.

Zed acceptance criteria must therefore include file classification and root detection, not only server process launch. The manual Zed gate should test a normal root package, a nested package, a workspace without root manifests, and behavior before and after any documented `file_types` mapping.

## Protocol Requirements

The LSP spike must prove:

- `initialize` and capability negotiation.
- `textDocument/didOpen`, `textDocument/didChange`, and `textDocument/didClose`.
- `textDocument/publishDiagnostics`.
- `textDocument/completion`.
- `textDocument/hover`.
- At least one navigation method, preferably definition or references, before a real MVP decision.
- Graceful `shutdown` and `exit`.

Protocol tests should run independently of Zed. Zed tests should then prove the same server works through a real editor launcher.

## Fixtures

Required editor fixtures:

- Minimal XRD with valid schema fields.
- Invalid XRD with a clear diagnostic.
- Pipeline Composition with at least one function step.
- `function-go-templating` inline template with Crossplane helper calls.
- Filesystem template directory referenced by a Composition.
- Provider CRD or schema fixture for managed resource completion.
- Package metadata with dependencies.
- Ordinary Kubernetes YAML to ensure non-Crossplane YAML is not degraded.

## Recommendation

Optimize for a protocol-first LSP with Zed as the first real editor integration. Do not build a Zed-only server, but treat the `crossplane-yaml` integration as a required acceptance gate.

Use diagnostics, completion, hover, and navigation as the core authoring loop. Defer rich render previews and code actions until source mapping and schema indexing are proven.

## Confidence

High that diagnostics, completion, hover, and navigation are the right initial editor priorities. This is consistent across LSP, YAML LS, Helm LS, Terraform LS, CUE, and the existing Upbound/Zed tooling.

Medium that virtual rendered documents should be deferred. They are valuable, but source mapping and render latency are still unproven.

## Evidence That Would Change This Recommendation

- Zed cannot reliably launch or communicate with a standalone `vibe-xpls` binary.
- User research shows render/debug loops are more important than diagnostics and schema intelligence.
- Existing Kubernetes/YAML tooling covers most needed editor UX with less implementation cost.
- Source mapping from templates to rendered resources proves easier than expected and makes virtual documents cheap enough for first scope.
