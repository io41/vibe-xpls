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
- Existing Zed extension fixtures under `<crossplane-yaml-repo>/fixtures/`.
- Official Crossplane documentation examples when license and attribution permit.
