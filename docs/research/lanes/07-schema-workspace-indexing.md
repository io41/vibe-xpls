# Schema And Workspace Indexing Research

## Summary

`vibe-xpls` should use Kubernetes CRD OpenAPI schemas as the canonical source for YAML validation, completion, and hover, then layer Crossplane workspace semantics on top. The first index should be local-first: built-in Crossplane APIs, workspace XRDs, Compositions, provider CRDs checked into the repository, and package metadata from `crossplane.yaml`.

Package registries, Upbound Marketplace data, `.up/go` model surfaces, and live-cluster discovery are useful later sources, but they introduce freshness, trust, auth, and cache-policy problems. They should be optional inputs, not required for baseline editor or agent behavior.

## Sources

- Crossplane Providers: https://docs.crossplane.io/latest/packages/providers/
- Crossplane Configurations: https://docs.crossplane.io/latest/packages/configurations/
- Crossplane XRDs: https://docs.crossplane.io/latest/composition/composite-resource-definitions/
- Crossplane Compositions: https://docs.crossplane.io/latest/composition/compositions/
- Crossplane CLI command reference: https://docs.crossplane.io/master/cli/command-reference/
- Kubernetes CRDs: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/
- YAML Language Server: https://github.com/redhat-developer/yaml-language-server
- Datree CRDs catalog: https://github.com/datreeio/CRDs-catalog
- Kubeconform: https://github.com/yannh/kubeconform
- Upbound official providers: https://docs.upbound.io/manuals/packages/providers/

## Schema Source Matrix

| Source | Use | Freshness | Risk |
| --- | --- | --- | --- |
| Built-in Crossplane API schemas | Composition, XRD, package, function, and provider package fields | Versioned with `vibe-xpls` | Must track Crossplane releases |
| Workspace XRDs | XR field completion, hover, validation, and Composition `compositeTypeRef` resolution | Immediate file-watch freshness | XRD schema changes may not reflect a running cluster until Crossplane restarts |
| Workspace Compositions | Function pipeline graph, template paths, resource names, and schema references | Immediate file-watch freshness | Composition can reference schemas not present locally |
| Workspace provider CRDs | Managed resource `apiVersion`/`kind` and field docs | Immediate file-watch freshness | Large CRD sets can be expensive to index |
| Package metadata in `crossplane.yaml` | Dependency graph and package-root detection | Immediate file-watch freshness | Metadata declares intent, not installed schemas |
| Package revisions or lock-like metadata | Exact installed or resolved versions when available | Depends on source | May be cluster-only or stale |
| User schema directories | Air-gapped and enterprise schema overrides | User-controlled | Requires clear precedence and conflict reporting |
| Upbound Marketplace/model surfaces | Rich provider docs and generated language models | Remote freshness | Optional network/auth/licensing and API stability concerns |
| Live cluster CRD discovery | Highest fidelity for a selected cluster | Cluster-current | Requires kubeconfig access and workspace trust |

## Freshness Strategy

- Index local files first and update incrementally from file-watch events.
- Treat local XRDs and CRDs as authoritative for the current workspace unless the user explicitly selects another schema source.
- Store each schema with provenance: source file, URL or cluster identity, package/version when known, content hash, and timestamp.
- Prefer digest-pinned package metadata when available. Crossplane packages support image tags and digests; digests are better for reproducible indexing.
- Keep remote downloads off by default for the first implementation. When enabled, use an explicit cache directory, TTL, provenance metadata, and a clear refresh command.
- Keep live-cluster discovery off by default. When enabled, expose it as a workspace-trusted command with read-only Kubernetes access and sanitized errors.
- Detect conflicts by `apiVersion`/`kind` and report which source wins rather than silently merging unrelated schemas.

## Completion/Hover Strategy

- Resolve `apiVersion` and `kind` from known schemas before offering field completion.
- For Crossplane XRs, derive `spec` and allowed status fields from the XRD `openAPIV3Schema`.
- For Compositions, use `compositeTypeRef` to link back to the XRD version marked `referenceable: true`.
- For managed resources, prefer CRD OpenAPI descriptions for hover and validation. If Upbound model docs are later available, use them as supplemental docs, not as the schema authority.
- For templates, expose schema-backed context only where the analyzer can connect the template to an XR, Composition step, and function input source.
- For unknown schemas, degrade to syntax and structural diagnostics instead of inventing fields.

## Recommendation

Build a local schema index around CRD and XRD OpenAPI documents, with explicit provenance and conflict handling. Support these first: built-in Crossplane APIs, workspace XRDs, workspace Compositions, workspace provider CRDs, package metadata, and user-provided schema directories.

Defer live-cluster discovery and remote Marketplace/model indexing until the cache, trust, and auth model are designed. Treat Upbound multi-language resource schemas as a promising enrichment path, not the canonical YAML source.

## Confidence

High that CRD OpenAPI and XRD schemas are the right canonical source for YAML intelligence.

Medium on package dependency indexing because local projects may not contain enough resolved version data without registry or cluster access.

Low on `.up/go` or Marketplace model indexing as a first-scope dependency because the public docs show resource schemas exist, but not a stable local API contract for this project to depend on.

## Evidence That Would Change This Recommendation

- A stable Upbound or Crossplane schema API appears that is easier and safer than CRD OpenAPI indexing.
- Real fixtures show most projects do not check in provider CRDs or package metadata, making remote or cluster discovery necessary earlier.
- Performance tests show provider CRD indexing is too expensive without prebuilt schema bundles.
- Kubernetes validation tools provide a stable Go library API that can replace custom schema lookup without losing Crossplane-specific graph data.
