# Schema Coverage Ratchet Design

## Goal

Create a schema completeness contract for the committed pinned Crossplane CRDs.
The contract should drive development toward 100% generated built-in schema
support, prevent regressions in CI, and make upstream documentation and schema
changes visible when the target universe changes.

This is not a passive report. It is a ratchet:

- normal PR CI checks the committed pinned CRDs offline and deterministically;
- current gaps are explicit, classified, and checked in;
- coverage decreases fail CI;
- new unclassified gaps fail CI;
- fixed gaps must be removed from the baseline;
- scheduled CI separately checks whether Crossplane upstream has newer or
  changed release artifacts.

The baseline records known debt. It is not the goal. The long-term goal remains
100% coverage for the current committed pins, and then 100% again after each
upstream pin update.

## Scope

In scope:

- Coverage measurement for committed upstream Crossplane CRDs under
  `internal/analyzer/schemadata/upstream`.
- A checked-in baseline that classifies known schema extraction gaps.
- Generated machine-readable and human-readable coverage artifacts.
- CI enforcement for stale coverage artifacts, coverage regressions, new
  unclassified gaps, and obsolete baseline entries.
- Integration with `./scripts/update-generated.sh`.
- A separate networked upstream drift check suitable for scheduled CI.

Out of scope for the first implementation:

- Runtime LSP behavior changes.
- Editor UI for schema coverage.
- Normal PR CI network access.
- Auto-updating pinned Crossplane releases.
- Provider registry, package registry, cluster, or user workspace schema coverage.
- Relationship-aware or value completion coverage.

## Developer Workflow

The primary local workflow remains:

```sh
./scripts/update-generated.sh
```

As part of this design, that command should:

- regenerate built-in schemas;
- regenerate schema coverage artifacts;
- verify generated schemas are byte-identical to committed output;
- verify generated coverage artifacts are byte-identical to committed output;
- enforce the coverage ratchet;
- run the existing tests and local builds.

For faster iteration, the schema generator should expose a narrower coverage
operation behind the update script. The documented command shape should be:

```sh
vibe-xpls-schema-gen generate --config <path> --out <dir>
vibe-xpls-schema-gen coverage generate --config <path> --out <dir>
vibe-xpls-schema-gen coverage check --config <path> --out <dir>
vibe-xpls-schema-gen drift check --config <path>
```

`generate` writes schema artifacts and manifest output. `coverage generate`
writes `coverage.json` and `coverage.md` without rewriting `baseline.json`.
`coverage check` regenerates coverage data in memory or a temporary directory,
compares it to committed coverage artifacts, validates the baseline, and exits
non-zero on stale artifacts, unclassified gaps, obsolete baseline entries, or
coverage regressions. `drift check` is networked and is not called by normal PR
CI.

The implementation may keep a compatibility alias for the current no-subcommand
generator invocation, but scripts and documentation should use the explicit
subcommands once this design lands.

Developer loop:

1. Change generator code or committed pinned upstream inputs.
2. Run the update command.
3. If coverage improves, commit regenerated artifacts and remove fixed known
   gaps from the baseline.
4. If a new gap appears, either fix the generator or classify the gap in the
   baseline with a reason.
5. If the baseline changes, rerun the update command so generated coverage
   artifacts reflect the updated classification state.
6. CI verifies the committed artifacts and baseline match the current extraction
   behavior.

## Coverage Model

Coverage is measured at three levels.

### Resource Coverage

For each pinned Crossplane release:

- upstream CRDs discovered;
- generated schemas emitted;
- skipped CRDs;
- skipped reasons;
- compatibility or synthetic schemas separated from upstream CRD-derived
  schemas.

Target: 100% of intended upstream Crossplane CRDs produce a schema unless the
resource is explicitly excluded with a classified reason.

### Field Coverage

For each generated GVK:

- discoverable OpenAPI field paths;
- generated field paths;
- missing field paths;
- extra generated compatibility fields;
- array and object paths handled consistently.

Target: 100% of discoverable OpenAPI field paths are represented in the
generated schema model unless the path is excluded or blocked by a classified
unsupported shape.

The target-field extractor should be deliberately separate from the generator's
emission path where practical. It may share OpenAPI parsing types, but it should
not simply trust the generated schema as the source of truth. The point is to
detect silent loss between upstream CRD input and generated completion data.

### Coverage Buckets

Coverage data must distinguish how each emitted schema fact was produced:

- `covered-upstream`: emitted from the upstream CRD OpenAPI source without
  compatibility changes.
- `covered-with-compat-override`: upstream had the field, but the emitted
  description or metadata was changed by compatibility docs.
- `compat-added-field`: compatibility logic added a field that was not present
  in the upstream CRD OpenAPI source.
- `compat-only-schema`: the entire generated schema is synthetic and has no
  upstream CRD source.
- `missing`: upstream target field or metadata was not emitted.
- `excluded`: upstream target is intentionally excluded by baseline policy.
- `unsupported-shape`: upstream target is blocked by a classified unsupported
  OpenAPI construct.

The headline 100% target is based on upstream CRD-derived targets. Compatibility
fields and compatibility-only schemas are useful product coverage, but they must
not inflate upstream extraction coverage.

### Target Extraction Rules

The first implementation must begin by inventorying all OpenAPI constructs used
by the committed pinned CRDs. At minimum, the rules must cover constructs that
already appear in those CRDs, including:

- `properties`
- `items`
- local `$ref`
- `additionalProperties`
- `x-kubernetes-preserve-unknown-fields`
- `x-kubernetes-embedded-resource`
- `x-kubernetes-int-or-string`
- `oneOf`
- `anyOf`
- `allOf`
- `patternProperties`, if present in committed pinned inputs

The coverage target extractor must define a construct-to-target rule for every
observed construct. Initial rules should be conservative:

- Object `properties` produce one target field per property and recurse into
  child schemas.
- Array `items` produce the array item path using `[]` and recurse into fixed
  item schemas.
- Local `$ref` resolves within the same OpenAPI document; unresolved or cyclic
  refs produce classified unsupported-shape gaps.
- `additionalProperties` produces the containing map field as a target. Fixed
  child keys under an open map are not required unless also declared through
  `properties`.
- `x-kubernetes-preserve-unknown-fields` produces the containing field as a
  target but does not require unknown children.
- `x-kubernetes-embedded-resource` produces a target for the embedded resource
  field and requires an explicit rule for whether Kubernetes resource metadata
  enrichment applies.
- `x-kubernetes-int-or-string` produces one scalar field target whose type
  coverage is satisfied only when the generated metadata preserves that it is an
  int-or-string value or a deliberate fallback is classified.
- `oneOf`, `anyOf`, and `allOf` require explicit per-case handling. If the
  generator cannot deterministically flatten them into completion-safe fields,
  each affected target must be classified as an unsupported-shape gap.
- `patternProperties` represents dynamic keys. It should produce the containing
  map field as a target and should not require arbitrary dynamic child-key
  completions in the first implementation.

These rules are part of the contract. If a new upstream construct appears after
a pin update, coverage check fails until the construct is assigned a rule or a
classified baseline gap.

### Metadata And Documentation Coverage

For each discoverable field:

- description is preserved when present upstream;
- type is captured when available;
- required status is captured;
- enum values are captured;
- default values are captured;
- deprecation metadata is captured when available.

Target: no silent loss of user-facing completion or hover value. Documentation
changes matter because completion and hover quality depend on descriptions and
schema metadata, not only on field paths.

Metadata percentages must use explicit denominators:

- description coverage: fields with an upstream description that also emit the
  same normalized description, divided by fields with an upstream description;
- type coverage: fields with an upstream type or type-equivalent construct that
  emit type metadata, divided by fields with an upstream type or type-equivalent
  construct;
- enum coverage: fields with upstream enum values that emit the same enum set,
  divided by fields with upstream enum values;
- default coverage: fields with upstream defaults that emit the same normalized
  default, divided by fields with upstream defaults;
- deprecation coverage: fields with upstream deprecation metadata that emit
  deprecation metadata, divided by fields with upstream deprecation metadata.

Requiredness is a relation between a parent schema and one of its immediate
children. Required coverage should be tracked by `(release, apiVersion, kind,
parentPath, childName)`, not only by child path.

Fields with no upstream description do not count as missing-description gaps.
They may appear in a separate "upstream-undocumented" count so documentation
quality is visible without treating upstream absence as generator loss.

## Known Gap Baseline

The baseline is a checked-in policy file, not generated output. It classifies
known gaps that remain after current generator support is applied.

Each baseline entry should include:

- release;
- apiVersion and kind;
- path or CRD-level scope;
- category;
- reason;
- short note.

Suggested categories:

- `excluded-resource`
- `unsupported-openapi-shape`
- `missing-field`
- `missing-description`
- `missing-type`
- `missing-required`
- `missing-enum`
- `missing-default`
- `missing-deprecation`
- `compat-added-field`
- `compat-only-schema`

CI should fail when:

- an observed gap has no baseline entry;
- a baseline entry no longer corresponds to an observed gap;
- a generated schema disappears for an intended CRD;
- a generated field disappears without a matching classified reason;
- metadata coverage decreases without a matching classified reason;
- coverage artifacts are stale.

The baseline should be reviewable and intentionally a little uncomfortable to
edit. Adding a gap should be a deliberate product decision, not a convenient way
to quiet a report.

Baseline entries are pinned to exact release tags by default. When a Crossplane
pin is upgraded, entries for the old release become obsolete and CI should fail
until the developer regenerates coverage and either removes them or recreates
equivalent entries for the new release. The first implementation may support an
explicit `release: "*"` scope for gaps that are known to apply to every pinned
release, but wildcard entries must still match an observed gap in at least one
current release.

Resource exclusions are keyed by exact `(release, apiVersion, kind)`. A CRD
file with multiple served versions produces one resource-coverage target per
served `(apiVersion, kind)`. Excluding one served version does not exclude the
others.

## Artifacts

Store coverage artifacts close to the existing schema bundle:

```text
internal/analyzer/schemadata/
  config.json
  manifest.json
  schemas/
  coverage/
    coverage.json
    coverage.md
    baseline.json
```

`coverage.json` is generated machine-readable data for CI and automation. It
should be deterministic, sorted, and stable across platforms.

The JSON shape should be explicit and versioned. The first schema should follow
this shape:

```json
{
  "formatVersion": 1,
  "releases": [
    {
      "tag": "v2.2.1",
      "commit": "713541df7fc5cf0946b6573837831086465a2dbe",
      "totals": {
        "upstreamGVKs": 24,
        "generatedGVKs": 24,
        "targetFields": 1200,
        "coveredUpstreamFields": 1180,
        "knownGaps": 20
      },
      "gvks": [
        {
          "apiVersion": "apiextensions.crossplane.io/v1",
          "kind": "Composition",
          "sourcePath": "upstream/crossplane/v2.2.1/cluster/crds/apiextensions.crossplane.io_compositions.yaml",
          "sourceSHA256": "abc123",
          "buckets": {
            "covered-upstream": 100,
            "covered-with-compat-override": 3,
            "compat-added-field": 2,
            "missing": 0,
            "unsupported-shape": 1
          },
          "fields": [
            {
              "path": "spec.pipeline[].input",
              "bucket": "covered-upstream",
              "metadata": {
                "description": "covered",
                "type": "covered",
                "required": "not-required",
                "enum": "not-present-upstream",
                "default": "not-present-upstream",
                "deprecated": "not-present-upstream"
              }
            }
          ]
        }
      ]
    }
  ]
}
```

Releases must sort by release tag, GVKs by `apiVersion` then `kind`, fields by
path, and map-like bucket/category summaries by lexical key.

`coverage.md` is generated human-readable review output. It should summarize:

- release totals;
- overall distance to 100%;
- GVK coverage;
- field coverage;
- metadata/documentation coverage;
- worst-covered GVKs, limited to the 10 lowest upstream field-coverage
  percentages;
- unsupported-shape counts by reason;
- known gaps remaining.

`baseline.json` is human-maintained policy. It should not be overwritten by the
generator.

`manifest.json` should remain focused on runtime schema loading. Coverage data
should not be added to the runtime manifest unless runtime behavior later needs
it.

## Ratchet Semantics

The ratchet should compare the newly computed coverage state against committed
coverage artifacts and the baseline.

The check passes when:

- committed coverage artifacts match regenerated output;
- every observed gap is classified by the baseline;
- every baseline entry still maps to an observed gap;
- per-release and per-GVK coverage did not decrease outside classified gaps;
- no stale schema generation is detected.

The check fails with actionable output. A developer should be able to tell
whether to:

- update generated artifacts;
- fix the generator;
- remove an obsolete baseline entry;
- classify a new gap;
- update pinned upstream inputs.

When a gap is fixed by generator work, the baseline entry becomes stale and CI
fails until the entry is removed. This makes progress toward 100% visible and
prevents dead exceptions from accumulating.

## Upstream Drift Detection

Upstream drift detection is separate from normal PR CI. It is explicitly
networked and intended for scheduled CI, such as a nightly or weekly job.

The drift command should compare `internal/analyzer/schemadata/config.json`
against Crossplane upstream and report:

- whether each pinned release tag still resolves to the configured commit;
- whether newer Crossplane releases exist beyond the latest pinned release;
- whether upstream CRD file contents for a pinned tag differ from the committed
  copies.

The first implementation should use GitHub as the source of truth:

- release freshness comes from Crossplane Git tags or GitHub Releases for
  `crossplane/crossplane`;
- tag-to-commit validation comes from the GitHub Git refs API or `git
  ls-remote`;
- CRD content drift comes from fetching the configured CRD paths at the pinned
  tag and comparing their SHA-256 hashes with committed copies.

Scheduled CI should provide `GITHUB_TOKEN` to avoid unauthenticated rate limits.
The drift command should also run locally without a token when rate limits allow
it, but CI should treat missing `GITHUB_TOKEN` as configuration error for the
scheduled job.

The command should not mutate files by default. First implementation can fail
the scheduled job with a clear stale-pins message. Opening an issue or PR can be
added later.

This separation keeps normal PR CI deterministic while ensuring the project gets
a signal when "100%" has changed because upstream changed.

## Implementation Shape

Most implementation should live in `internal/schemagen`, because coverage is a
generator completeness concern, not runtime analyzer behavior.

Likely components:

- `coverage.go`: computes target and actual coverage facts.
- `coverage_report.go`: renders deterministic JSON and Markdown.
- `coverage_baseline.go`: loads and validates the known-gap baseline.
- schema generator CLI support for coverage generation and checking.
- `scripts/update-generated.sh` integration.
- scheduled CI workflow support for upstream drift detection.

Data flow:

1. Load pinned releases from `schemadata/config.json`.
2. Read committed upstream CRDs.
3. Extract discoverable target fields and metadata from OpenAPI.
4. Generate or load actual schema output.
5. Compare target facts to actual generated schema facts.
6. Apply baseline classifications.
7. Emit deterministic `coverage.json` and `coverage.md`.
8. Fail when artifacts are stale, regressions appear, new gaps are
   unclassified, or baseline entries are obsolete.

Unsupported OpenAPI constructs should not disappear silently. If the generator
cannot map a construct into the schema model, the coverage pass should either
prove no user-facing field or metadata was lost, or require a classified gap.

The implementation should extend the existing stale-generation test path rather
than adding a second full generator invocation where practical. The regenerated
temporary output should include both `schemas/` and `coverage/`, and tests
should compare both directories against committed artifacts.

## Error Handling

Malformed committed upstream CRDs should fail coverage generation.

Unsupported shapes should be reported with enough context to classify or fix:

- release;
- source file;
- apiVersion and kind;
- path;
- construct;
- short reason.

Network errors in the upstream drift command should be reported separately from
stale pins. A scheduled drift job that cannot reach upstream should make the
operational failure clear, not imply that schema support regressed.

Normal PR coverage checks must not require network access.

## Testing

Tests should cover:

- deterministic coverage JSON output;
- deterministic Markdown summary output;
- stale coverage artifacts are detected;
- the handcrafted CRD fixture matrix produces the expected target field and
  metadata set before comparing to generator output;
- new unclassified gaps fail;
- obsolete baseline entries fail;
- fixed gaps require baseline cleanup;
- documentation-only changes are detected;
- unsupported OpenAPI shapes are classified;
- compatibility overrides and compatibility-only schemas land in separate
  coverage buckets;
- release pin upgrades make old release-scoped baseline entries obsolete unless
  they are intentionally recreated or wildcard-scoped;
- normal tests do not access the network.

Existing stale-generation tests should be extended or paired with coverage
staleness tests so `./scripts/update-generated.sh` remains the single local
command for schema generator changes.

The fixture matrix should include at least one handcrafted CRD for each
construct rule in "Target Extraction Rules". Each fixture should assert the
target set independently from whether the generator currently emits that target.

## Acceptance Criteria

- A developer can run the update command locally and see schema coverage
  failures with actionable reasons.
- CI can enforce the committed pinned CRDs offline.
- Coverage artifacts are deterministic and reviewable.
- The baseline makes known gaps explicit and shrinks when generator support
  improves.
- Documentation and metadata changes are treated as user-facing schema coverage.
- Scheduled upstream drift detection can tell maintainers when pinned Crossplane
  releases should be refreshed.
- The initial implementation PR includes generated `coverage.json`,
  `coverage.md`, and a populated `baseline.json` that classifies every observed
  current gap, so normal CI is green at landing.
- The implementation plan defines the exact CLI compatibility strategy for the
  current no-subcommand generator invocation.
