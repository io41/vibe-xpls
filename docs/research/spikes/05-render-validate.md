# Render Validate Spike

## Summary

This spike tested `crossplane render` and `crossplane beta validate` as authoritative Crossplane proof steps for `vibe-xpls`. The tools are useful, but their runtime behavior is not appropriate for the LSP hot path.

Evidence supports using render and validate as explicit, on-save, or manual commands. The analyzer still needs a local static path for per-keystroke diagnostics, schema lookup, and template-aware parsing.

Primary source context: Crossplane documents `crossplane render` as a way to preview composition output without a control plane and says it requires Docker for its default runtime; function annotations can select Docker or Development runtimes and Docker pull policy. Source: https://docs.crossplane.io/latest/composition/compositions/

Crossplane documents `crossplane beta validate` as offline validation that can validate render output and may download and cache provider or configuration schemas. Source: https://docs.crossplane.io/latest/cli/command-reference/

## Tool Availability

- `command -v crossplane` returned `<homebrew-bin-dir>/crossplane`.
- `command -v docker` returned `<homebrew-bin-dir>/docker`.
- `crossplane version` printed `Client Version: v2.2.1`, then failed while trying to use kubeconfig context `minikube`.
- `docker version` succeeded with client version `29.4.3` and server version `28.4.0`.
- `docker image ls xpkg.upbound.io/crossplane-contrib/function-go-templating` found cached image `xpkg.upbound.io/crossplane-contrib/function-go-templating:v0.11.0`, image ID `34f1ca0ff2e1`, disk usage `80.8MB`, and content size `23.5MB`.

The failed `crossplane version` probe is important evidence: even version detection may touch kubeconfig. It should not run in the LSP hot path.

## Commands Run

Fixture path:

```text
<render-fixture-dir>/{xr.yaml,composition.yaml,functions.yaml}
```

The render fixture used `render.crossplane.io/runtime-docker-pull-policy: Never` and package `xpkg.upbound.io/crossplane-contrib/function-go-templating:v0.11.0`.

Commands and outcomes:

```text
command -v crossplane
command -v docker
crossplane version
docker version
docker image ls xpkg.upbound.io/crossplane-contrib/function-go-templating
crossplane render <render-fixture-dir>/xr.yaml <render-fixture-dir>/composition.yaml <render-fixture-dir>/functions.yaml --include-function-results --include-full-xr --timeout=15s
<time-bin> -p crossplane render <render-fixture-dir>/xr.yaml <render-fixture-dir>/composition.yaml <render-fixture-dir>/functions.yaml --include-function-results --include-full-xr --timeout=15s
crossplane beta validate spikes/schema-index/testdata/xrd.yaml spikes/schema-index/testdata/composition.yaml --skip-success-results
<time-bin> -p crossplane beta validate spikes/schema-index/testdata/xrd.yaml spikes/schema-index/testdata/composition.yaml --skip-success-results
crossplane beta validate spikes/schema-index/testdata/xrd.yaml spikes/schema-index/testdata/provider-crd.yaml --skip-success-results
<time-bin> -p crossplane beta validate spikes/schema-index/testdata/xrd.yaml spikes/schema-index/testdata/composition.yaml --skip-success-results --cache-dir <crossplane-cache-dir> --clean-cache
```

## Cold Runtime

The closest clean-cache validate run used `--cache-dir <crossplane-cache-dir> --clean-cache`. It failed in `real 0.18` with:

```text
cannot download package xpkg.crossplane.io/crossplane/crossplane:v2.2.1
One or more parameters passed to the function were not valid. (-50)
```

This is not a successful cold-runtime measurement. It shows that cache misses can cross into registry and macOS credential-helper behavior, which is too environment-dependent for editor diagnostics.

No clean cold render timing was captured. The tested render path used a cached function image and `runtime-docker-pull-policy: Never`.

## Warm Runtime

The successful escalated render command emitted the XR plus an `s3.aws.upbound.io/v1beta1` `Bucket` named `demo-bucket`. No function result objects appeared despite `--include-function-results`, because the function returned no explicit results.

Timed warm render runs:

```text
real 1.41 user 0.03 sys 0.02
real 1.42 user 0.03 sys 0.02
```

Validation of the XRD plus Composition succeeded:

```text
Total 1 resources: 0 missing schemas, 1 success cases, 0 failure cases
```

The timed warm validate run returned:

```text
real 0.20 user 0.11 sys 0.03
```

## Docker Behavior

Unprivileged timed render failed in `0.38s` with Docker socket access denied:

```text
operation not permitted
<docker-socket>
```

Escalated render succeeded with the cached function image. This confirms the Docker boundary is real even for a small local fixture, and that permissions can determine whether render works.

The cached image evidence is relevant but narrow: it proves the fixture can run without pulling when the image is already available and pull policy is `Never`. It does not prove first-run behavior, registry availability, or behavior for other functions.

## Cache Behavior

`crossplane beta validate` can use cached schemas, and Crossplane's command reference says provider and configuration schemas may be downloaded and cached. Source: https://docs.crossplane.io/latest/cli/command-reference/

The warm XRD plus Composition validation succeeded quickly. The clean-cache attempt failed before producing a useful validation result because it tried to download `xpkg.crossplane.io/crossplane/crossplane:v2.2.1` and hit a macOS credential-helper error.

Cache state therefore affects both latency and reliability. The analyzer should not make correctness or responsiveness depend on Crossplane package cache state during hot-path diagnostics.

## Diagnostic Mapping Limits

Render output is authoritative for executed function behavior, but it does not automatically provide source spans back to the original XR, Composition, function input, or template regions. The successful render emitted final resources, not a mapping from generated fields to source ranges.

`--include-function-results` did not produce function result objects for the fixture because the function returned no explicit results. That means diagnostics cannot assume this flag will always provide user-facing explanations.

Validation can report schema success and missing-schema cases, but the observed missing-schema output was coarse:

```text
[!] could not find CRD/XRD for: apiextensions.k8s.io/v1, Kind=CustomResourceDefinition
Total 1 resources: 1 missing schemas, 0 success cases, 0 failure cases
```

For editor diagnostics, `vibe-xpls` still needs local parsing, schema provenance, and source mapping. External render and validate results can augment diagnostics, but they are not a complete mapping layer.

## Decision Impact

Render and validate should be explicit, on-save, or manual operations, not per-keystroke LSP work. Warm validate is plausibly cheap enough for save-time use on small inputs, but render crosses Docker permissions and function runtime boundaries even when images are cached.

The analyzer must parse enough locally to avoid Docker, cache, network, credential-helper, and kubeconfig dependencies in hot paths. Use static analysis for fast diagnostics and reserve Crossplane CLI execution for trusted, bounded proof commands with clear status and timeouts.
