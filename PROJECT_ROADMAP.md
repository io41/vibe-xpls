# Project Roadmap

This file is the project-level source of truth for what is done, what is
currently in progress, and what comes next. Keep detailed design and execution
plans in `docs/superpowers/specs/` and `docs/superpowers/plans/`, but update
this roadmap whenever the project direction, status, priorities, or completion
criteria change.

## Maintenance Rules

- Update this file in the same pull request as any roadmap-changing work.
- Move items between `Done`, `Current`, and `Next` as soon as the status changes.
- Link to the relevant design, implementation plan, validation note, or release
  note when an item becomes concrete.
- Keep roadmap items outcome-oriented. Implementation details belong in linked
  specs and plans.
- Keep the single generation/build command documented here and in
  `docs/generated-schemas.md` whenever the schema or documentation pipeline
  changes.

## Single Update Command

Run this after changing schema generator code, schema bundle config, committed
schema inputs, schema docs, or generated-schema documentation:

```sh
./scripts/update-generated.sh
```

The command regenerates committed schema artifacts, runs stale-generation and
schema tests, runs the full Go test suite, and builds both CLIs into
`dist/local/`.

## Done

- Local Crossplane package detection for root, nested, multi-package, and
  no-package workspaces.
- YAML-aware parsing, path mapping, diagnostics, hover, and key completion for
  stable Crossplane YAML structures.
- Zed-first LSP server path with initialize, document sync, diagnostics, hover,
  completion, shutdown, and UTF-8/UTF-16 position handling.
- Completion presentation policy: field completions use property kind, omit
  generic `detail`, and keep prose in `documentation`.
- Offline generated built-in Crossplane core schema bundle for pinned releases:
  Crossplane `v1.20.7` and `v2.2.1`.
- Release-aware schema lookup keyed by exact `(Crossplane release, apiVersion,
  kind)`.
- Reproducible schema generation from committed upstream Crossplane CRD
  artifacts.
- Stale-generation tests that rerun the generator and compare output with the
  committed manifest and schema JSON files.
- Workspace CRD/XRD schema sources. Local provider CRDs and XRD OpenAPI schemas
  are loaded into the source-neutral schema model for package-scoped key
  completions and docs without registry, cluster, or network access.
- Manual Zed validation passed for the generated completion foundation,
  including Markdown completion documentation, YAML array-item first-key
  completion, and v1/v2 release-specific completion sets.
- Generated compatibility schemas include parent-key documentation for package
  metadata completions.

## Current

- Keeping schema generation and update flow simple, offline, deterministic, and
  reviewable.

## Next

1. Function input schema dispatch.
   Use the selected pipeline function and known input GVK to provide completions
   under `spec.pipeline[].input` when the input object's schema is known.

2. Relationship-aware completions.
   Use the local package, XRD, Composition, function, and provider graph to
   suggest safe relationships such as composition type refs and package
   dependency references.

3. Safe value completions.
   Add value completions from schema enums, defaults, and in-workspace facts
   without inventing values or querying remote systems.

4. Developer/debug schema insight command.
   Add a command that explains bundle health, selected release, active package
   root, schema provenance, and completion suppression reasons for a file.

## References

- Generated schema docs: `docs/generated-schemas.md`
- Generated completion design:
  `docs/superpowers/specs/2026-05-21-generated-completion-foundation-design.md`
- Generated completion implementation plan:
  `docs/superpowers/plans/2026-05-21-generated-completion-foundation.md`
- Completion presentation decision:
  `docs/research/decisions/completion-presentation.md`
