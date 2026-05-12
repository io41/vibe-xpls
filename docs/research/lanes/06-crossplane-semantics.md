# Crossplane Semantics Research

## Summary
As of 2026-05-12, `vibe-xpls` should treat Crossplane semantics as a mostly static graph model with a small number of runtime-execution escape hatches. The fast path is to analyze Compositions, function inputs, request/response proto shapes, annotations, and template conventions locally; the optional proof path is to run Crossplane's own render, validate, and trace commands when the question depends on actual function execution, schema enforcement, or live cluster state.

Treat Upbound `xpls` as a reference implementation only. Its current transport and packaging shape should inform the analyzer, but not define the product contract.

## Sources
- Crossplane Compositions, latest docs: https://docs.crossplane.io/latest/composition/compositions/
- Function Patch and Transform, latest docs: https://docs.crossplane.io/latest/guides/function-patch-and-transform/
- Crossplane CLI command reference, latest docs: https://docs.crossplane.io/latest/cli/command-reference/
- Crossplane v1.19 Compositions, legacy `mode: Resources` behavior: https://docs.crossplane.io/v1.19/concepts/compositions/
- Upgrade to Crossplane v2 guide, legacy composition migration context: https://docs.crossplane.io/master/guides/upgrade-to-crossplane-v2/
- `function-go-templating` README: https://github.com/crossplane-contrib/function-go-templating
- `function-sdk-go` request helpers: https://pkg.go.dev/github.com/crossplane/function-sdk-go/request
- `function-sdk-go` proto/v1 API: https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1

## Semantic Model
- Pipeline-mode Compositions are ordered function pipelines. Crossplane sends each function the same fresh observed snapshot, plus the accumulated desired state and any context from earlier steps. Functions may add or mutate desired composed resources, update XR status, and pass context forward; the last function's context is discarded. Source: https://docs.crossplane.io/latest/composition/compositions/ and https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1
- Legacy `mode: Resources` is deprecated backward-compat behavior. Current docs say to migrate to pipeline mode; the legacy model is the patch-and-transform era Composition shape with static `resources`, `patches`, and readiness checks. Source: https://docs.crossplane.io/v1.19/concepts/compositions/ and https://docs.crossplane.io/master/guides/upgrade-to-crossplane-v2/
- Patch-and-transform is the modern pipeline replacement for legacy resources mode. It is declarative, template-like, and intentionally does not support loops or conditionals, which makes most of its semantics statically inspectable. Source: https://docs.crossplane.io/latest/guides/function-patch-and-transform/
- `function-go-templating` turns a `RunFunctionRequest` into rendered manifests. Templates can read `.observed`, `.desired`, `.context`, and `.extraResources`, and the function supports built-in, Sprig, and helper functions such as `setResourceNameAnnotation`. Source: https://github.com/crossplane-contrib/function-go-templating
- `RunFunctionRequest` carries `meta`, `observed`, `desired`, optional `input`, optional `context`, credentials, required resources, and legacy `extraResources`. The request docs explicitly mark `extraResources` as deprecated in favor of `requiredResources`. Source: https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1 and https://pkg.go.dev/github.com/crossplane/function-sdk-go/request
- `RunFunctionResponse` carries `meta`, partial `desired`, `results`, optional `context`, `requirements`, XR/claim conditions, and optional `output`. The desired state is partial by design; omitting fields later deletes them. Source: https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1
- ExtraResources and context writes are pipeline data-flow features, not composed-resource semantics. Current docs treat them as `requiredResources` in v2, with the legacy `extraResources` field still present for compatibility. Function-go-templating also stores extra resources under the `apiextensions.crossplane.io/extra-resources` context key and merges them with prior context. Source: https://docs.crossplane.io/latest/composition/compositions/ and https://github.com/crossplane-contrib/function-go-templating
- Special resources and annotations are part of the templating contract. `gotemplating.fn.crossplane.io/composition-resource-name` names a composed resource, `gotemplating.fn.crossplane.io/ready` marks readiness, `CompositeConnectionDetails` is supported only for legacy v1 XRs, and v2 XRs should instead compose an explicit `Secret`. Source: https://github.com/crossplane-contrib/function-go-templating
- Readiness is partly declarative and partly runtime-derived. Patch-and-transform supports `readinessChecks` such as string, integer, non-empty, condition, boolean, and `None`, while function-go-templating can set readiness directly with the `ready` annotation. Source: https://docs.crossplane.io/latest/guides/function-patch-and-transform/ and https://github.com/crossplane-contrib/function-go-templating
- The Crossplane CLI's `render`, `beta validate`, and `beta trace` commands are the authoritative runtime/external checks. `render` executes functions via Docker by default, `beta validate` is offline schema/CEL validation, and `beta trace` shows live object relationships from a kubeconfig-backed cluster view. Source: https://docs.crossplane.io/latest/cli/command-reference/

## Static Analysis Candidates
- High confidence: parse Composition mode, function steps, function refs, function input kinds, and legacy `resources` templates as first-class syntax, because these are declarative and directly exposed in YAML. Source: https://docs.crossplane.io/latest/composition/compositions/ and https://docs.crossplane.io/v1.19/concepts/compositions/
- High confidence: model patch-and-transform patches, transforms, `readinessChecks`, and resource names statically. These are schema-shaped declarative fields and do not require executing the function to identify the intended graph. Source: https://docs.crossplane.io/latest/guides/function-patch-and-transform/
- High confidence: parse `RunFunctionRequest` and `RunFunctionResponse` as structured graph state, including partial desired-state semantics, context flow, results, requirements, and condition targets. Source: https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1
- High confidence: statically inspect `function-go-templating` manifests for the known special annotations and special resource kinds, including `ExtraResources`, `Context`, `CompositeConnectionDetails`, `composition-resource-name`, and `ready`. Source: https://github.com/crossplane-contrib/function-go-templating
- Medium confidence: partially evaluate Go-template expressions for literal resource names, helper calls, and obvious context lookups, but treat any nontrivial template logic as opaque unless a runtime fixture proves it. Source: https://github.com/crossplane-contrib/function-go-templating
- Medium confidence: classify `requiredResources` versus deprecated `extraResources`, and `v1` versus `v1beta1` request/response compatibility, so the analyzer can explain legacy versus current semantics without running Crossplane. Source: https://pkg.go.dev/github.com/crossplane/function-sdk-go/request and https://pkg.go.dev/github.com/crossplane/function-sdk-go/proto/v1

## Authoritative Validation Candidates
- `crossplane render` should be the proof step for function behavior, because it actually executes the function runtime and can expose template rendering, resource accumulation, readiness signaling, and context propagation. Source: https://docs.crossplane.io/latest/cli/command-reference/
- `crossplane render` should also be the proof step for runtime-specific annotations such as `render.crossplane.io/runtime`, `render.crossplane.io/runtime-docker-cleanup`, `render.crossplane.io/runtime-docker-pull-policy`, and `render.crossplane.io/runtime-development-target`. Source: https://docs.crossplane.io/latest/composition/compositions/
- `crossplane beta validate` should be the proof step for schema/CEL correctness and for checking `crossplane render` output against provider or XRD schemas. It is offline, but it is still the authoritative Crossplane validator for those checks. Source: https://docs.crossplane.io/latest/cli/command-reference/
- `crossplane beta trace` should be the proof step for live object topology and installed package graph questions, because it reflects the cluster's actual installed state rather than a hypothetical rendered graph. Source: https://docs.crossplane.io/latest/cli/command-reference/
- Dynamic required-resource requests are runtime behavior. The 5-iteration stability limit and the actual resource-resolution results are not fully provable from static YAML alone, so they belong in executable validation. Source: https://docs.crossplane.io/latest/composition/compositions/

## Fixture Needs
- A minimal pipeline fixture set: `xr.yaml`, `composition.yaml`, `functions.yaml`, and one golden rendered output per function chain. Source basis: https://docs.crossplane.io/latest/cli/command-reference/
- A legacy `mode: Resources` fixture set that mirrors the same intent, so the analyzer can show old-versus-new semantics side by side. Source basis: https://docs.crossplane.io/v1.19/concepts/compositions/
- Function runtime fixtures for `crossplane render`: observed resources, context files, context values, and any cluster-like inputs needed for extra/required resources. Source basis: https://docs.crossplane.io/latest/cli/command-reference/ and https://docs.crossplane.io/latest/composition/compositions/
- Template fixtures for `function-go-templating`: inline, filesystem, and environment-based template sources, plus examples for `composition-resource-name`, `ready`, `ExtraResources`, `Context`, and legacy `CompositeConnectionDetails`. Source basis: https://github.com/crossplane-contrib/function-go-templating
- Schema fixtures for `crossplane beta validate`: CRD/OpenAPI schemas, XRD schemas, and at least one CEL-backed XRD case. Source basis: https://docs.crossplane.io/latest/cli/command-reference/
- Trace fixtures should be live-cluster harnesses rather than pure files, because `beta trace` is about installed graph topology rather than render-time function execution. Source basis: https://docs.crossplane.io/latest/cli/command-reference/

## Recommendation
Implement a static semantic analyzer first, and treat Crossplane execution as an optional validation layer. The analyzer should understand both pipeline-mode compositions and legacy `mode: Resources`, but it should mark the legacy path as deprecated and avoid promoting it as the preferred contract.

Use `crossplane render` only when a fixture or user request needs proof of function execution, use `crossplane beta validate` when a schema or CEL answer must be authoritative, and use `crossplane beta trace` only for live object-topology questions. Keep Upbound `xpls` reference-only and do not let its current transport model narrow the product contract.

## Confidence
Medium-high. The official docs are consistent on the broad shape of the semantics, and the function SDK proto docs make the request/response model explicit. The remaining uncertainty is mostly about how much of the behavior the local corpus will depend on opaque custom functions, legacy v1 compatibility, and live-cluster state.

## Evidence That Would Change This Recommendation
- If Crossplane publishes a stable, machine-readable semantic inspection API that replaces most of `render`, `validate`, and `trace`, shift more weight from the local analyzer to that upstream API.
- If the target repositories depend heavily on opaque custom functions whose behavior cannot be inferred from YAML, move runtime validation earlier and reduce confidence in static-only analysis.
- If the real corpus is still dominated by legacy `mode: Resources` and legacy v1-only connection handling, prioritize compatibility shims and legacy fixtures over pipeline-first abstractions.
- If `crossplane render` or `beta validate` becomes available as a cheap library call rather than a Docker/CLI boundary, fold that proof path deeper into the default analyzer workflow.
