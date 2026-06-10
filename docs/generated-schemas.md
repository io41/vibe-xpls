# Generated Schemas

The built-in Crossplane schema bundle is generated from committed upstream Crossplane release artifacts.

Current pins:

- Crossplane `v1.20.7`, commit `5fae6c1ab967e57b1dc792b5c52c97bceda12953`
- Crossplane `v2.2.1`, commit `713541df7fc5cf0946b6573837831086465a2dbe`

Regenerate and verify after changing `internal/analyzer/schemadata/config.json`,
generator code, committed upstream artifacts, generated-schema documentation,
coverage artifacts, or coverage baseline entries:

```bash
./scripts/update-generated.sh
```

The command regenerates schema artifacts and coverage artifacts, enforces the
coverage ratchet, runs stale-generation checks, runs the full Go test suite,
and builds local CLIs into `dist/local/`.

The generation block is:

```bash
go run ./cmd/vibe-xpls-schema-gen generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
go run ./cmd/vibe-xpls-schema-gen coverage generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
go run ./cmd/vibe-xpls-schema-gen coverage check --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

`internal/analyzer/schemadata/coverage/baseline.json` is human-maintained.
Coverage generation reads it but never rewrites it. New coverage gaps should be
fixed in the generator when practical, or classified with a baseline entry when
the gap is intentional or not yet supported. Remove obsolete baseline entries
when generator support improves; the coverage check reports entries that no
longer match an observed gap.

`internal/analyzer/schemadata/coverage/coverage.md` is the human review report.
It summarizes upstream field coverage, metadata coverage, known gaps by
category, and metadata gap hotspots. `coverage.json` remains the full
machine-readable per-field source of truth.

Generated compatibility documentation may intentionally replace terse or generic
upstream descriptions with Crossplane-specific completion text. These fields are
reported as `covered-with-compat-override` metadata, not as missing upstream
description gaps.

The generator must produce byte-identical output from committed inputs. Runtime never downloads schemas.

## Upstream Drift

Normal PR checks do not access the network. They only regenerate and compare
committed schema inputs and generated artifacts.

Scheduled CI checks upstream drift with:

```bash
go run ./cmd/vibe-xpls-schema-gen drift check --config internal/analyzer/schemadata/config.json
```

The scheduled workflow runs the command with `--require-token` and
`GITHUB_TOKEN` so GitHub rate limits are explicit. The drift check reports
pinned tag commit drift, newer stable Crossplane tags, and CRD content drift. It
does not mutate committed schema inputs.
