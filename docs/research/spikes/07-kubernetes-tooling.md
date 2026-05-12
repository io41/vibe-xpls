# Kubernetes Tooling Spike

## Summary

This spike compares generic Kubernetes/YAML tooling with the local Go-native schema index proven in `spikes/schema-index`. The result is clear: `vibe-xpls` should not rebuild generic Kubernetes and YAML capabilities wholesale, but it also should not depend on those tools as the Crossplane semantic core.

Existing tooling already solves YAML parsing, schema association, Kubernetes manifest validation, CRD schema validation paths, and server-derived field documentation patterns. Crossplane still needs a semantic layer for XRD-to-Composition relationships, function pipelines, mixed template/YAML source maps, render truth, provider package discovery, special Crossplane annotations and resources, and agent-facing APIs.

Recommended direction: build a Go-native Crossplane analyzer that reuses or adapts schema sources and validation ideas, with optional interop or delegation only where it does not compromise analyzer portability or Zed integration.

## Tools Evaluated

| Tool | Source | Useful Capability | Limit For `vibe-xpls` |
| --- | --- | --- | --- |
| Zed YAML support | https://zed.dev/docs/languages/yaml | Native YAML support through tree-sitter YAML and `redhat-developer/yaml-language-server`; configurable YAML LS settings, schemas, SchemaStore behavior, and `customTags` | Strong editor integration path, but generic YAML configuration does not model Crossplane relationships |
| YAML Language Server | https://github.com/redhat-developer/yaml-language-server | Schema-oriented YAML LSP capabilities for diagnostics, completion, hover, schema association, and custom configuration | Does not understand XRD/Composition/function pipeline semantics by itself |
| Kubeconform | https://kubeconform.mandragor.org/docs/overview/ | High-performance Kubernetes manifest validation with configurable remote/local schema locations, CRD/offline validation capability, and CI-friendly use | OpenAPI-only validation limits mean it cannot validate all Kubernetes admission behavior or Crossplane render semantics |
| Kubeconform CLI options | https://kubeconform.mandragor.org/docs/usage/ | JSON output, strict mode, Kubernetes version selection, schema locations, and concurrency controls | Useful as an optional validator, not as the LSP semantic graph |
| Kubeconform CRD support | https://kubeconform.mandragor.org/docs/crd-support/ | Custom schema locations and CRD-to-JSON-Schema workflows | Requires schema conversion/catalog discipline and still does not derive Crossplane composition intent |
| `kubectl explain` | https://kubernetes.io/docs/reference/kubectl/generated/kubectl_explain/ | Field documentation from server OpenAPI using JSONPath-like field identifiers | Requires a cluster/server OpenAPI source and remains object-schema oriented |
| Local schema index | `spikes/schema-index` | Fixture-backed Go index for XRD, Composition, provider CRD, and Crossplane package metadata with passing `go test -count=1 ./...` | Narrow research implementation; needs a production YAML parser, watchers, precedence, and package discovery |

## Commands Run

Local command evidence for generic tooling:

```text
command -v kubeconform
```

Result: not found, exit 1.

```text
command -v yaml-language-server
```

Result: not found, exit 1.

```text
npx yaml-language-server --help
```

Result: failed with npm `ENOTFOUND` for `https://registry.npmjs.org/yaml-language-server`; npm logs could not be written under `<npm-log-dir>`.

```text
npx kubeconform -h
```

Result: failed with npm `ENOTFOUND` for `https://registry.npmjs.org/kubeconform`; npm logs could not be written under `<npm-log-dir>`.

Local comparison point from Task 6:

```text
cd spikes/schema-index
go test -count=1 ./...
```

Result: passing. The spike indexes local XRD, Composition, provider CRD, and Crossplane package metadata fixtures without network access.

## Capabilities Already Solved

- YAML syntax and structural editing are already covered well by Zed's native YAML support through tree-sitter YAML and YAML Language Server integration.
- Schema association, schema-backed completion, hover, diagnostics, SchemaStore configuration, custom schemas, and custom YAML tags are already established YAML LS concepts.
- Kubernetes manifest validation is already covered by Kubeconform's schema-driven validator, including JSON output, strict mode, Kubernetes version selection, concurrency, and configurable schema locations.
- CRD validation is already a known path through local or remote JSON Schema locations and CRD-to-JSON-Schema conversion workflows.
- `kubectl explain` proves a useful documentation model: field-level help derived from OpenAPI with path-like field identifiers.
- The Task 6 Go-native index proves `vibe-xpls` can resolve local Crossplane schema facts from fixtures without installing or invoking external YAML/Kubernetes tools.

## Crossplane Gaps

Generic Kubernetes/YAML tooling does not solve the Crossplane semantic layer:

- Function pipelines: understanding pipeline steps, function inputs, function output, and desired resource flow across a Composition.
- XRD-to-Composition relationships: connecting composite resource definitions, claims, compositions, and selected composition revisions.
- `function-go-templating` mixed source maps: mapping diagnostics and navigation across Go template source, YAML fragments, request context, and generated resources.
- Render truth: distinguishing static YAML validity from authoritative `crossplane render` output and function results.
- Provider package discovery: deriving provider CRDs from package metadata, dependency constraints, installed package state, local mirrors, or lock-like workspace metadata.
- Special annotations and resources: handling Crossplane-specific annotations, connection details, composition functions, environment/config resources, readiness behavior, and package resources.
- Agent APIs: exposing repository-level operations such as list compositions, find schema, validate workspace, and render through structured, trust-aware commands rather than cursor-only LSP methods.

## Reuse Recommendation

Do not rebuild generic Kubernetes/YAML capabilities wholesale. Build a Crossplane semantic layer that can reuse or adapt existing ideas and schema sources:

- Reuse Zed's YAML and YAML LS integration for ordinary YAML behavior where possible.
- Reuse schema association concepts from YAML LS, especially explicit user schemas, schema directories, SchemaStore controls, and custom tags.
- Reuse Kubeconform-style configurable schema locations, strict/offline validation ideas, JSON output shape, and concurrency model for optional validation commands.
- Reuse `kubectl explain` as a field-documentation precedent when OpenAPI descriptions are available from XRDs, provider CRDs, or built-in Crossplane schemas.
- Keep the core analyzer Go-native and fixture/testable so it remains portable across Zed, CLI, agent APIs, and future editor adapters.
- Treat external validators as optional interop/delegation points for explicit commands or CI checks, not as the source of truth for per-keystroke Crossplane semantics.

## Decision Impact

The implementation direction should prioritize a Go-native analyzer with Crossplane-specific indexing, source mapping, render integration, and agent-facing structured APIs. Generic YAML/Kubernetes tooling should be treated as infrastructure to interoperate with, learn from, or optionally call, not as the product boundary.

This keeps Zed integration practical: `vibe-xpls` can coexist with Zed's YAML support while adding Crossplane-aware navigation, diagnostics, hovers, and commands that YAML LS cannot infer. It also keeps the analyzer usable outside Zed, which matters for CLI checks and AI agent workflows.

Evidence confidence is high for the reuse decision because the evaluated tools explicitly focus on YAML schemas, Kubernetes OpenAPI validation, or server OpenAPI documentation, while the local Task 6 spike already shows a Go-native path for Crossplane-specific schema facts. Evidence confidence is medium for exact delegation boundaries because Kubeconform and YAML LS could not be installed locally in this restricted environment; future work should test source-mapped outputs and latency once network access or vendored binaries are available.
