## Decision.

Reuse Kubernetes, YAML, schema, and validation tooling concepts where they are strong, and optionally call external validators for explicit commands or CI. Do not build a generic YAML LSP, and do not delegate the Crossplane semantic core to a generic Kubernetes or YAML LSP. The first analyzer should use a local schema index backed by CRD and XRD OpenAPI schemas, then layer Crossplane-specific relationships and template semantics on top.

## Evidence.

- `docs/research/lanes/08-kubernetes-language-intelligence.md` says Kubernetes tooling already solves YAML schema lookup, CRD catalog validation, server-like validation, Helm template precedents, and cluster-oriented commands, but not Crossplane XRD-to-Composition relationships, function pipelines, template context, render results, package metadata, or agent-safe operations.
- `docs/research/lanes/08-kubernetes-language-intelligence.md` recommends interop and selective reuse, not using a Kubernetes LSP as the semantic core.
- `docs/research/lanes/09-existing-tooling.md` identifies YAML Language Server and Helm LS as useful patterns for schema association, degraded parsing, and template-aware workflows, while also saying generic YAML LSPs cannot infer Crossplane pipeline behavior.
- `docs/research/lanes/07-schema-workspace-indexing.md` recommends CRD and XRD OpenAPI schemas as the canonical source for validation, completion, and hover, with local workspace sources first and remote or live-cluster sources optional later.
- `docs/research/spikes/04-schema-index.md` proves local XRD, Composition, provider CRD, and package metadata fixtures can provide `apiVersion`/`kind` lookup and field documentation without external tools or network access.
- `docs/research/spikes/07-kubernetes-tooling.md` concludes that generic YAML/Kubernetes tools should be reused as infrastructure, ideas, or optional calls, while the core remains a Go-native Crossplane analyzer portable across Zed, CLI, agent APIs, and future editor adapters.

## Alternatives Considered.

- Build a generic YAML LSP: duplicates mature YAML tooling and still misses Crossplane-specific graph, pipeline, package, and render semantics.
- Delegate to YAML Language Server plus configuration: reuses schema association and diagnostics, but cannot model XRD-to-Composition links, `function-go-templating` context, or analyzer-level agent commands.
- Delegate to a Kubernetes validator as the core: useful for structural validation, but it cannot infer whether a Composition step produced the intended managed resource or map function output back to source.
- Depend on remote CRD catalogs or live cluster discovery first: improves coverage for some workspaces, but adds freshness, trust, auth, cache, and reproducibility problems.

## Risks.

- Reusing concepts instead of embedding existing tools means the project must implement enough YAML parsing, schema routing, and indexing to be useful.
- Optional validators may produce diagnostics with different severity, schema precedence, or source mapping than the local analyzer.
- Local-first schema lookup can miss provider CRDs or package versions that are not checked into the workspace.
- Avoiding a generic YAML LSP core may require extra coexistence work so ordinary YAML behavior remains familiar in editors.

## What Would Change This Decision.

- A maintained Kubernetes or YAML LSP exposes a stable embeddable API for CRD schema routing, source-mapped diagnostics, and extension points that can cleanly host Crossplane semantics.
- User research shows most users only need generic Kubernetes or YAML validation, with little demand for Crossplane navigation, templates, render proof, package awareness, or agent operations.
- Real-world workspaces rarely contain enough local XRD, CRD, or package metadata for useful local-first indexing.
- Kubeconform, `kubectl-validate`, or another validator proves fast, hermetic, source-mapped, and extensible enough to serve save-time Crossplane workflows without replacing the analyzer.
