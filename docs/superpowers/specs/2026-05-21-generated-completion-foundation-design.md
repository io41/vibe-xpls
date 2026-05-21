# Generated Completion Foundation Design

## Scope And Goals

This slice creates the foundation for generated Crossplane YAML key completions and docs. It replaces hand-written built-in Crossplane field maps with generated schema data from pinned upstream Crossplane releases.

The first slice is deliberately narrow:

- Generate built-in Crossplane core schemas from pinned upstream release artifacts.
- Support one pinned `v1.20.x` release and one latest pinned `v2.x` release.
- Provide YAML key completions only.
- Provide normalized documentation from upstream OpenAPI descriptions and schema metadata.
- Keep runtime offline and local-first.
- Do not scrape Crossplane docs.
- Do not fetch registries, packages, clusters, or remote schemas at runtime.

The feature should improve completions for core Crossplane resources such as `Composition`, `CompositeResourceDefinition`, `Configuration`, `Provider`, `Function`, and other Crossplane core CRDs present in the pinned bundle.

Success means:

- Completion labels come from generated schema data, not manual maps.
- Completion docs come from upstream OpenAPI descriptions and metadata.
- Version-specific differences between pinned v1 and v2 releases are represented.
- The internal model is ready for later workspace XRD and provider CRD sources.

## Schema Source Model

The design introduces a source-neutral schema model while keeping slice-1 runtime sources narrow.

Core concepts:

- `CrossplaneRelease`: exact built-in bundle identity, using pinned upstream tags such as `v1.20.0` and `v2.2.0`. The major track is derived from the tag.
- `SchemaDocument`: one exact `group/version/kind` schema for one `CrossplaneRelease`, with provenance.
- `SchemaField`: path, description, and metadata consumed by normalized docs in this slice: type, required, default, enum, and deprecation status/text.
- `SchemaProvenance`: source enum, upstream release tag, upstream schema path, upstream content hash, and deterministic generation metadata derived from input identities. Committed artifacts and manifests must not include wall-clock fields.
- `SchemaSource`: a single source enum. Slice 1 only populates generated built-in Crossplane bundles. Future values include workspace XRD, workspace CRD, user schema directory, and opt-in remote cache.

Schema index lookup uses `(CrossplaneRelease, group/version/kind)`. Completion and existing schema-doc lookup paths go through release resolution before schema lookup.

Release resolution is package-scoped:

1. Find the package containing the document.
2. Gather package constraints from future explicit override, package metadata `spec.crossplane.version` SemVer constraints, and v1/v2-only GVK signals in that package.
3. Filter candidate releases by package constraints.
4. Intersect with candidate releases that contain the document's exact `group/version/kind`.
5. If one release remains, use it.
6. If several releases remain, pick the latest pinned release by SemVer. The slice-1 scope of one pinned v1 release and one pinned v2 release makes this deterministic.
7. If no pinned release remains, return no schema completions for that request.

A v1/v2-only GVK signal means an exact GVK observed in a package document that exists in only one pinned release bundle.

When there is no package context, use the pinned release whose bundle contains the document's exact GVK. If multiple pinned releases contain it, pick the latest pinned release by SemVer. If no pinned release contains it, return no schema completions and emit `no-schema-for-release`.

Existing duplicate diagnostics must not regress. Built-in/workspace and workspace/workspace conflicts still behave as today. Cross-release same-GVK built-ins are not conflicts. Future workspace schema conflicts are package-scoped and compared against the package's resolved release.

Resolution results should be cached per package and invalidated when package metadata, package roots, schema inputs, or workspace generation changes.

## Completion Ordering And Value Roadmap

Slice 1 ships generated built-in Crossplane core schemas only. It covers key completions and normalized docs for core Crossplane resources generated from pinned releases, including:

- `Composition`, especially `spec.compositeTypeRef`, `spec.mode`, `spec.pipeline[]`, `spec.pipeline[].functionRef`, and v1-era `spec.resources[]` where present.
- `CompositeResourceDefinition`, including v1/v2-specific fields and schema-definition paths.
- Package metadata resources such as `Configuration`, `Provider`, and `Function`.
- Other Crossplane core CRDs present in the pinned built-in bundle.

This slice is narrower than the older first-runnable milestone scope. Workspace XRDs, workspace provider CRDs, and graph-aware relationship completions remain planned but are not required to prove the generated core schema pipeline.

For LSP item ordering:

- Emit stable `sortText` for every completion item.
- At a completion point, offer immediate child keys only.
- Root authoring keys sort as `apiVersion`, `kind`, `metadata`, `spec`, then remaining keys.
- Within a schema object, required keys sort before optional keys, then lexical by label.
- Parent/object keys and leaf keys are schema-derived. Do not hand-rank individual fields beyond the general ordering rules.
- Omit `documentation` when the upstream schema description is empty. Do not synthesize filler docs.
- Do not offer top-level `status` when the completion parent path is the document root. Do not suppress `status` at non-root schema-definition paths, such as XRD OpenAPI schema properties.

Path notation uses dot paths with `[]` for array items:

- `spec.dependsOn[].provider`
- `spec.pipeline[].functionRef.name`
- `spec.resources[].base`

Array item key completions are in scope when the schema path is clear. Completing fields inside `spec.pipeline[].input` is not in scope unless the input object's own GVK schema is known through a later function-schema dispatch slice.

Product value roadmap, highest to lowest:

1. Built-in Crossplane core resource keys and docs, especially Composition and package metadata authoring.
2. Core array-item key completions for `spec.pipeline[]`, `spec.dependsOn[]`, and v1 `spec.resources[]`.
3. Workspace provider CRD managed-resource keys from local/provider CRDs.
4. Workspace XRD-derived XR keys from local XRD OpenAPI schemas.
5. Function input schema dispatch for `spec.pipeline[].input`.
6. Relationship-aware completions from the Composition/package graph.
7. Safe value completions from schema enums, defaults, and in-workspace YAML/package graph facts.

Completion presentation follows the existing presentation decision:

- `label` is the concise field name.
- `kind` is `CompletionItemKind.Property`, emitted as LSP wire value `10`.
- `documentation` carries explanatory schema docs when available.
- `detail` is generally avoided. It is omitted for current field completions and must not contain generic category text.

## Schema Generation Workstream

The generated schema bundle is a workstream inside this slice. "Hybrid bundle" means one shipped built-in schema bundle containing multiple pinned Crossplane releases, indexed per release.

Pinned inputs:

- A repo config file lists explicit Crossplane inputs: `{tag, commit_sha}` for one `v1.20.x` release and one latest `v2.x` release. "Latest" is resolved when the config is updated, never at runtime.
- The Go toolchain and generator dependencies are pinned through normal Go module/toolchain metadata.
- Generation is reproducible: same config, same generator code, same toolchain/dependencies, and same upstream artifact bytes produce byte-identical committed artifacts.

Source artifacts:

- Vendor or commit the raw upstream CRD YAML artifacts used as generator inputs so staleness tests do not require network access.
- Prefer `cluster/crds/*.yaml` from the pinned Crossplane repo tag when the required core schema exists there.
- If a required core schema is not present as CRD YAML, fallback generation from pinned source must be explicitly configured for that release/GVK and documented.
- Generate Kubernetes metadata child schema from a pinned Kubernetes source matching the Crossplane release's Kubernetes dependency when upstream CRD artifacts expose `metadata` as an object without child properties. Record this metadata source in provenance.
- The generator fails loudly if a configured source path is absent.
- Record upstream release tag, commit SHA, source path, source license, and SHA-256 of the raw upstream artifact bytes for every schema document.

Generated outputs:

- Commit canonical JSON schema artifacts under one generated schema directory.
- Use LF line endings, sorted object keys, sorted schema paths, and stable encoding.
- Store release tag and source metadata in each schema document.
- Store generator version and bundle format version in a manifest.
- Build-time bundle validation fails on unknown or incompatible bundle format versions. Runtime loading of an unknown or incompatible bundle format disables schema completions per the runtime bundle loading rules.
- Load artifacts with `go:embed`; runtime does not need network access, upstream repos, or generator tools.
- The manifest exposes `(release tag, exact group/version/kind)` tuples for release-scoped lookup.

Runtime contract:

- Generated data replaces the current hand-written built-in field maps.
- Generated artifacts may store rich metadata: path, description, type, required flag, default, enum, and deprecation metadata.
- Slice-1 runtime may project that metadata into normalized Markdown documentation strings instead of exposing every metadata field through analyzer structs immediately.
- Empty upstream descriptions remain empty. Completion and existing schema-doc lookups must not synthesize filler documentation.

Normalization:

- Object `properties` become dot-path fields.
- Array item schemas use `[]`, for example `spec.pipeline[].functionRef.name`.
- Parent object/list entries are preserved so immediate child key completions work.
- When a parent object lists key `K` in its OpenAPI `required` array, field `parent.K` is marked required. Array item schemas do not inherit parent requiredness.
- `$ref` is resolved only when it points inside the same OpenAPI document. Cross-document and remote `$ref` are dropped and counted in generator diagnostics.
- `additionalProperties` and `x-kubernetes-preserve-unknown-fields` are treated as open maps; complete known fixed keys only. Entries inside maps such as `metadata.labels`, `metadata.annotations`, and provider extension maps are not key-completed in this slice.
- `oneOf`, `anyOf`, and `allOf` are dropped with generator diagnostics unless a later implementation defines deterministic flattening.
- When a resource `metadata` object lacks detailed child properties in the upstream CRD schema, merge generated Kubernetes `ObjectMeta` child fields for safe authoring keys: `name`, `labels`, and `annotations`. Include `namespace` only for namespace-scoped GVKs.
- Unsupported constructs fail generation when they would hide expected core child fields; otherwise they are counted and reported in generator diagnostics. Expected core child fields are the release/GVK/path label fixtures used by this spec's generator and completion acceptance tests. Those fixtures live in generator test/config data, not runtime product code.

Documentation rendering:

- Trim upstream descriptions and preserve paragraph order.
- Completion documentation starts with the normalized upstream description when present.
- Append schema fact lines after the description in this order when data exists: `_Type: <type>_`, `_Required_`, `_Default: <value>_`, `_Allowed: <comma-separated enum>_`, `_Deprecated: <text>_`.
- Omit absent facts and omit the documentation field entirely when both description and facts are empty.
- Hover documentation uses the same body after the existing hover title header.

Regeneration:

- Add one documented generator command.
- Add tests that rerun generation hermetically and fail if committed artifacts or manifest are stale.
- Staleness tests catch repo drift after config/generator changes. They do not detect upstream changes; tag bumps are manual PRs.
- Schema updates happen through normal PR review, not at build or runtime.
- Add attribution/NOTICE coverage for copied or normalized Apache-2.0 Crossplane schema material.

## Error Handling And Degradation

The language server should prefer fewer completions over misleading completions. In this design, "misleading" means a field that does not exist in the schema bundle selected for the document's exact GVK.

Generation-time behavior:

- Missing configured upstream source path fails generation.
- Unsupported OpenAPI constructs fail generation when they would hide expected core child keys.
- Unsupported constructs that do not affect emitted child keys are counted in generator diagnostics.
- Generator diagnostics are developer-facing and emitted in generator/test output.
- Stale means canonical regeneration from vendored pinned inputs produces byte-different artifacts or manifest. Stale generated artifacts fail the canonical regeneration test, not runtime startup.

Cutover behavior:

- Generated bundle data replaces the current hand-written built-in field maps.
- The server must not silently fall back to stale hand-written schemas after cutover.
- Acceptance test: running with generated bundle loading deliberately failed through a test seam produces no built-in field completions.

Runtime bundle loading:

- Generated manifest and artifact paths are fixed enough that missing required embedded files fail build or bundle validation.
- Empty, malformed, incompatible, or corrupt embedded artifacts disable schema completions and keep the server running when possible.
- Any fatal bundle load failure, including unknown or incompatible bundle format, empty artifacts, malformed artifacts, or corrupt artifacts, disables schema completions and reports a developer-visible log plus one `window/showMessage` warning per server initialize.
- First implementation is all-or-nothing for bundle load. Per-release isolation is out of scope for this slice.
- Runtime bundle health should not be reported as per-document diagnostics.

Completion behavior:

- Unknown GVK, no schema for the selected release, malformed YAML context, unstable template-derived path, unsupported schema shape, or failed bundle load returns the existing empty completion response shape: `{ "isIncomplete": false, "items": [] }`.
- Missing documentation omits docs; it does not block the completion item.
- Suppressed completions should emit throttled developer-facing reason codes in logs/debug output, including `missing-root-gvk`, `unknown-gvk`, `no-schema-for-release`, `malformed-yaml-context`, `unstable-template-path`, `unsupported-schema-shape`, and `bundle-disabled`.

Diagnostics:

- This slice should not add noisy document diagnostics for generator limitations or bundle health.
- Runtime user-facing diagnostics stay focused on documents being edited and conflicts that affect actual workspace behavior.
- Bundle health and completion suppression reasons are developer/debug signals.

## Testing And Acceptance

Testing proves the generated bundle, release selection, completion behavior, documentation normalization, and degradation paths.

Generator tests:

- Running the generator twice in a clean checkout produces byte-identical artifacts and manifest.
- No committed schema artifact or manifest field is derived from wall-clock time; regeneration is byte-identical across time zones and `SOURCE_DATE_EPOCH` values.
- Every configured `{release, GVK}` source exists and records tag, commit SHA, source path, source license, and raw artifact SHA-256.
- Generated paths use dot notation plus `[]` for array item schemas.
- Required metadata is propagated from parent OpenAPI `required` arrays to immediate child fields.
- Generated Kubernetes metadata enrichment provides `metadata.name`, `metadata.labels`, and `metadata.annotations` when upstream CRD schema omits metadata child properties; `metadata.namespace` is generated only for namespace-scoped GVKs.
- Unsupported constructs follow the generation rules: fail if they would hide expected core child keys, otherwise report diagnostics.
- NOTICE/attribution coverage exists for copied or normalized Apache-2.0 Crossplane schema material.

Schema index tests:

- Manifest data loads into an index keyed by `(CrossplaneRelease, exact group/version/kind)`.
- Same-GVK v1/v2 schemas are isolated; lookup returns release-specific data and proves at least one field or doc differs where upstream differs.
- Stored generated descriptions and metadata match golden fixtures.
- Normalized completion/hover documentation matches expected Markdown.
- Generated built-ins replace hand-written built-ins.
- Test-injected bundle load failure yields no built-in completions or schema docs.
- Corrupting one schema artifact disables the whole bundle for this slice.

Release resolution tests:

- A document with no package context resolves to the pinned release whose bundle contains the document's exact GVK.
- A document with no package context and a GVK present in both pinned bundles resolves to the latest pinned release by SemVer.
- A document with no package context and a v1-only GVK resolves to the pinned v1.20.x release.
- `spec.crossplane.version` filters candidate releases.
- A package with no `spec.crossplane.version` but a GVK that exists in only one pinned bundle resolves to that bundle.
- Exact GVK availability is intersected before tie-breaking.
- If multiple releases remain, latest pinned release by SemVer wins.
- If no pinned release remains, completions return the empty response shape and emit `no-schema-for-release`.

Completion tests:

- Generated fixtures assert `(release, GVK, parent path) -> expected immediate child labels`.
- v1.20 and latest v2 fixtures prove version-specific field sets differ where upstream schemas differ.
- Root completions sort `apiVersion`, `kind`, `metadata`, `spec` first.
- Required keys sort before optional keys within the same schema object.
- `sortText` is non-empty and stable for the same `(release, GVK, parent path, label)` across regeneration and server restarts.
- Every LSP completion item emits `kind == 10`.
- Field completions omit `detail`.
- Root-level `status` is not offered when the completion parent path is the document root. Schema-definition paths may still expose status-related fields.
- Empty upstream descriptions produce no `documentation` field.
- Array item paths with `[]` produce immediate child key completions.
- Unknown or unsupported contexts return `{ "isIncomplete": false, "items": [] }`.

Degradation tests:

- Invalid bundle format disables schema completions, keeps the server running when possible, logs a clear status, and emits one initialize-time `window/showMessage` warning.
- Corrupt embedded artifact disables schema completions and emits one initialize-time `window/showMessage` warning.
- Unknown GVK returns `{ "isIncomplete": false, "items": [] }` and emits `unknown-gvk`.
- Test-injected bundle load failure returns `{ "isIncomplete": false, "items": [] }` and emits `bundle-disabled`.
- Missing root GVK returns `{ "isIncomplete": false, "items": [] }` and emits `missing-root-gvk`.
- Malformed YAML around the cursor returns `{ "isIncomplete": false, "items": [] }` and emits `malformed-yaml-context`.
- Unsupported template-derived completion context returns `{ "isIncomplete": false, "items": [] }` and emits `unstable-template-path`.
- Unsupported schema shape at the completion parent returns `{ "isIncomplete": false, "items": [] }` and emits `unsupported-schema-shape`.
- No schema for the selected release returns `{ "isIncomplete": false, "items": [] }` and emits `no-schema-for-release`.
- `bundle-disabled` is logged at most once per server lifetime. Other suppression reason logs are throttled to at most one log per `(URI, reason code, document generation)`.

Manual Zed smoke tests:

- Existing root, nested, and multi-package completion cases still work with generated labels.
- A v1.20-constrained package shows v1.20-specific labels and docs.
- A latest-v2 package shows v2-specific labels and docs.
- Completion rows show property icons and no generic `detail`.
- Completion documentation renders normalized schema Markdown.
- Completion acceptance preserves the indentation behavior fixed in earlier slices.
- A document outside a package context still offers built-in completions when its exact GVK exists in a pinned bundle.
- Fatal bundle failure warns exactly once and does not create noisy document diagnostics.

## Boundaries And Follow-Ups

This slice stays focused on generated built-in Crossplane core schemas for key completions and docs.

Non-goals for this slice:

- No value completions.
- No docs scraping.
- No registry, package, or cluster discovery.
- No workspace XRD/provider CRD indexing yet.
- No function input schema dispatch.
- No graph-aware relationship completions.
- No runtime network updates.
- No partial runtime recovery if one release slice of the committed bundle is corrupt; first implementation treats bundle loading as all-or-nothing.
- No editor behavior fixes beyond what the language server controls.

Likely follow-ups, in order:

1. Workspace CRD/XRD schema sources using the same source-neutral model.
2. Function input schema dispatch for `spec.pipeline[].input`.
3. Relationship-aware completions from the package/composition graph.
4. Safe value completions from enums, defaults, and in-workspace YAML/package graph facts, without registry or cluster discovery.
5. Optional developer/debug command explaining bundle health, package release resolution, and schema provenance.

Main risks:

- Crossplane OpenAPI shapes may expose unsupported constructs; generator diagnostics and golden tests should make that visible early.
- Version resolution can become confusing if package constraints are loose; keep deterministic package-scoped resolution across the pinned `v1.20.x` and latest pinned `v2.x` bundles, using latest pinned SemVer only after package constraints and exact GVK availability are applied.
- Generated docs can become noisy if over-formatted; docs should stay normalized, concise, and sourced from schema metadata.
