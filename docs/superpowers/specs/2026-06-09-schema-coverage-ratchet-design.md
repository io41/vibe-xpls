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
operation behind the update script. The implementation plan should choose the
exact CLI shape, but the command must support both generating artifacts and
checking them without rewriting files.

Developer loop:

1. Change generator code or committed pinned upstream inputs.
2. Run the update command.
3. If coverage improves, commit regenerated artifacts and remove fixed known
   gaps from the baseline.
4. If a new gap appears, either fix the generator or classify the gap in the
   baseline with a reason.
5. CI verifies the committed artifacts and baseline match the current extraction
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

`coverage.md` is generated human-readable review output. It should summarize:

- release totals;
- overall distance to 100%;
- GVK coverage;
- field coverage;
- metadata/documentation coverage;
- worst-covered GVKs;
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
- optionally, whether upstream CRD file contents for a pinned tag differ from
  the committed copies.

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
- optional later CI workflow for scheduled upstream drift detection.

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
- new unclassified gaps fail;
- obsolete baseline entries fail;
- fixed gaps require baseline cleanup;
- documentation-only changes are detected;
- unsupported OpenAPI shapes are classified;
- normal tests do not access the network.

Existing stale-generation tests should be extended or paired with coverage
staleness tests so `./scripts/update-generated.sh` remains the single local
command for schema generator changes.

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
