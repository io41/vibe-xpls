# Kubernetes Language Intelligence Research

## Summary

`vibe-xpls` should reuse Kubernetes schema and validation work where it is already strong, but it should not delegate the Crossplane semantic graph to a generic Kubernetes or YAML language server. Kubernetes tooling already solves core YAML-schema lookup, CRD catalog validation, server-like validation, Helm template precedents, and cluster-oriented commands. Crossplane adds XRD-to-Composition relationships, function pipelines, template context, render results, package metadata, and agent-safe semantic operations.

The best direction is interop and selective reuse: learn from YAML Language Server, Kubeconform, `kubectl-validate`, Helm LS, and VS Code Kubernetes Tools; validate generic Kubernetes resources through schema-derived mechanisms; keep Crossplane semantics in the `vibe-xpls` analyzer.

## Sources

- YAML Language Server: https://github.com/redhat-developer/yaml-language-server
- Datree CRDs catalog: https://github.com/datreeio/CRDs-catalog
- Kubeconform: https://github.com/yannh/kubeconform
- kubectl-validate: https://github.com/kubernetes-sigs/kubectl-validate
- VS Code Kubernetes Tools: https://github.com/vscode-kubernetes-tools/vscode-kubernetes-tools
- Helm LS: https://github.com/mrjosh/helm-ls
- Helm chart format: https://helm.sh/docs/topics/charts/
- Kubernetes CRDs: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/

## Capability Matrix

| Tool | Capabilities To Reuse | Crossplane Limit |
| --- | --- | --- |
| YAML Language Server | YAML syntax, schema association, completion, hover, diagnostics, schema store, Kubernetes CRD store | Does not model XRD/Composition/function pipeline relationships |
| Datree CRDs catalog | Public CRD JSON schemas for shift-left validation and editor schema association | Catalog coverage may miss private providers and workspace-local XRDs |
| Kubeconform | Fast CLI validation, local and remote schema locations, CRD support, CI-friendly outputs | Go module API is documented as work in progress and not stable |
| `kubectl-validate` | Local validation with Kubernetes apiserver validation code and stronger Kubernetes parity | Project focus is Kubernetes objects, not Crossplane package/function semantics |
| VS Code Kubernetes Tools | Cluster explorer, `kubectl explain`, apply/diff/logs, Helm authoring UX, command design | VS Code-specific and cluster-oriented; not a portable analyzer layer |
| Helm LS | Template-aware language server that delegates YAML intelligence to YAML LS, values/schema indexing, noisy diagnostic controls | Helm templates and values differ from Crossplane `function-go-templating` request context |

## Reuse Options

- Embed schema-routing ideas from YAML LS: choose a schema from document content, not only filename globs.
- Allow user-configured schema catalogs and directories, including Datree CRDs catalog or internal mirrors.
- Use Kubeconform or `kubectl-validate` as optional external validators in CI or explicit editor commands, not per-keystroke diagnostics.
- Use Helm LS as a precedent for mixed template/YAML degradation: template intelligence remains domain-specific while YAML diagnostics can be limited when they get noisy.
- Provide `kubectl explain`-style hovers where schema descriptions are available, but map them through Crossplane-specific references.
- Keep ordinary Kubernetes YAML behavior out of the Crossplane-specific language mode unless a package root or file classification says it belongs to `vibe-xpls`.

## Limits For Crossplane

- A Kubernetes schema validator can say whether a rendered managed resource is structurally valid; it cannot infer whether a Composition step produced the right desired resource.
- CRD catalogs do not know a workspace's private XRDs, private providers, package dependencies, or version pins unless the project supplies them.
- Server-like validation may need cluster version, feature gates, admission behavior, or CEL support that a static LSP cannot fully reproduce.
- Helm tooling is useful as a design precedent, but Helm `.Values` and Crossplane `RunFunctionRequest` have different data models.
- Cluster-integrated commands are powerful but must stay behind explicit user action and workspace trust.

## Recommendation

Build `vibe-xpls` as a Crossplane analyzer that can reuse Kubernetes schema assets and optionally call Kubernetes validation tools. Do not build on top of a Kubernetes LSP as the semantic core.

For first scope, implement local schema lookup from built-in Crossplane schemas, workspace XRDs, provider CRDs, and optional user schema directories. Add explicit commands or CI-friendly hooks for Kubeconform or `kubectl-validate` after the source-mapping and trust model are proven.

## Confidence

High that Kubernetes schema and validation tooling should be reused as assets and precedents.

High that generic Kubernetes tooling cannot replace Crossplane-specific semantic analysis.

Medium on whether `kubectl-validate` should be integrated before Kubeconform; its parity goal is attractive, but integration maturity and output mapping need a runnable spike.

## Evidence That Would Change This Recommendation

- A maintained Kubernetes LSP exposes a stable library/API for CRD schema routing, validation, and source-mapped diagnostics that can be extended cleanly with Crossplane semantics.
- The Kubernetes tooling spike shows Kubeconform or `kubectl-validate` can provide fast, source-mapped diagnostics suitable for editor save-time use.
- User research shows `vibe-xpls` users primarily need generic Kubernetes schema validation and rarely use Crossplane-specific navigation or templates.
- Zed integration proves that composing with YAML LS is simpler and more reliable than implementing schema handling inside `vibe-xpls`.
