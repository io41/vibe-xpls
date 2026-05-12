# Schema Index Spike

## Summary

This spike proves a local, dependency-light schema index can derive useful Crossplane lookup data from fixture-backed files. The Go package under `spikes/schema-index` indexes one XRD, one Composition, one provider CRD, and one package metadata file, then exposes `LookupKind(apiVersion, kind)` and `FieldDocumentation(apiVersion, kind, path)`.

The implementation is intentionally narrow. It uses a small indentation-based YAML subset parser that supports the fixture shapes needed for this research task, not arbitrary YAML.

## Indexed Sources

| Source | Fixture | Indexed Result |
| --- | --- | --- |
| XRD | `spikes/schema-index/testdata/xrd.yaml` | `platform.example.org/v1alpha1`, kind `CompositeBucket`, with OpenAPI field documentation |
| Composition | `spikes/schema-index/testdata/composition.yaml` | `apiextensions.crossplane.io/v1`, kind `Composition`, with `compositeTypeRef` captured |
| Provider CRD | `spikes/schema-index/testdata/provider-crd.yaml` | `s3.aws.upbound.io/v1beta1`, kind `Bucket`, with OpenAPI field documentation |
| Package metadata | `spikes/schema-index/testdata/crossplane.yaml` | `meta.pkg.crossplane.io/v1`, kind `Configuration`, with one provider dependency |

## Commands Run

- `cd spikes/schema-index && go test ./...`

Successful output:

```text
ok  	github.com/io41/vibe-xpls/spikes/schema-index	0.749s
```

- `git diff --check`

Successful output: no output.

## Lookup Results

- `LookupKind("platform.example.org/v1alpha1", "CompositeBucket")` returns the XRD-declared composite resource.
- `LookupKind("apiextensions.crossplane.io/v1", "Composition")` returns the Composition fixture and records its `platform.example.org/v1alpha1` `CompositeBucket` reference.
- `LookupKind("s3.aws.upbound.io/v1beta1", "Bucket")` returns the provider CRD resource.
- `LookupKind("meta.pkg.crossplane.io/v1", "Configuration")` returns the package metadata fixture.
- `FieldDocumentation("platform.example.org/v1alpha1", "CompositeBucket", "spec.parameters.region")` returns the XRD OpenAPI description.
- `FieldDocumentation("s3.aws.upbound.io/v1beta1", "Bucket", "spec.forProvider.bucketName")` returns the provider CRD OpenAPI description.
- `FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.apiVersion")` returns the Composition fixture directive documentation.
- `FieldDocumentation("meta.pkg.crossplane.io/v1", "Configuration", "spec.dependsOn.provider")` returns the package metadata fixture directive documentation.

## Freshness Limits

- The index reads local files only. There is no file watcher, cache invalidation, remote package lookup, or live-cluster discovery.
- The parser handles only the YAML subset used by the fixtures: maps, lists of maps, quoted scalars, and OpenAPI `properties` trees.
- It does not preserve comments except explicit `# xpls:doc` fixture directives used where the fixture has no embedded OpenAPI schema.
- It does not resolve package tags, digests, transitive dependencies, or installed package revisions.
- Duplicate `apiVersion` and `kind` pairs are reported as errors instead of applying a production precedence policy.

## Decision Impact

The spike supports building the first real schema index around local files and OpenAPI schema fragments. XRD and provider CRD fixtures already provide enough information for `apiVersion`/`kind` lookup and field hover documentation without network access.

The next production design should replace the subset parser with a real YAML parser, add file-watch freshness, define conflict precedence, and separate built-in Crossplane schemas from workspace-discovered schemas. Package metadata should be indexed for dependency context, but it is not enough by itself to provide provider field schemas.
