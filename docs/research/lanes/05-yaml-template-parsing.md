# YAML and Go-Template Parsing Research

## Summary

As of 2026-05-12, `vibe-xpls` should not treat YAML parsing and Go-template parsing as a single library choice. The shape we need is a mixed-source analyzer: parse YAML structure, parse template regions, and preserve a raw-to-masked-to-rendered span map so diagnostics still work while edits are incomplete.

`goccy/go-yaml` is the strongest primary candidate for the YAML side of that analyzer because it exposes parser, lexer, AST, comment preservation, source-aware errors, and source annotation helpers. `go.yaml.in/yaml/v4` is the better-maintained general-purpose YAML alternative and has useful `Node` positions, comments, stream-node options, unique-key checks, and strict field loading, but it is still just a YAML parser and is not template-aware. `gopkg.in/yaml.v3` should be treated as legacy compatibility only; the upstream repository is archived and the YAML org now points new work at `go.yaml.in/yaml/v4`.

For Kubernetes-compatible ingestion, `sigs.k8s.io/yaml` and `k8s.io/apimachinery/pkg/util/yaml` are useful adapters, not semantic engines. They are good for conversion, framing, and multi-document intake, but they do not preserve the source fidelity needed for source-span mapping across embedded templates.

For template handling, `text/template` is the execution model and `text/template/parse` is the closest thing to a syntactic tree. Sprig and the Crossplane `function-go-templating` helper set define the identifier surface we need to recognize. For analyzer-first work, those helpers should be cataloged and validated, not executed, until a later runnable spike proves the render path.

The practical recommendation is a dual-view architecture:

- raw file text for editing and incremental change tracking
- masked YAML/template structure for source mapping and tolerant diagnostics
- rendered view only for later runnable spikes and fixture validation

That keeps the analyzer useful during partial edits, avoids hard dependence on a live runtime context, and leaves room for a later executable render path.

## Sources

- `goccy/go-yaml` repository: https://github.com/goccy/go-yaml
- `goccy/go-yaml` package docs: https://pkg.go.dev/github.com/goccy/go-yaml
- `goccy/go-yaml` token and position docs: https://pkg.go.dev/github.com/goccy/go-yaml/token
- `goccy/go-yaml` parser docs: https://context7.com/goccy/go-yaml/llms.txt
- YAML org maintained fork for v4: https://github.com/yaml/go-yaml
- `go.yaml.in/yaml/v4` package docs: https://pkg.go.dev/go.yaml.in/yaml/v4
- `gopkg.in/yaml.v3` archive: https://github.com/go-yaml/yaml
- `gopkg.in/yaml.v3` package docs: https://pkg.go.dev/gopkg.in/yaml.v3
- Kubernetes YAML utilities: https://pkg.go.dev/k8s.io/apimachinery/pkg/util/yaml
- Kubernetes YAML wrapper: https://pkg.go.dev/sigs.k8s.io/yaml
- Go template package docs: https://pkg.go.dev/text/template
- Go template parse package docs: https://pkg.go.dev/text/template/parse
- Sprig package docs: https://pkg.go.dev/github.com/Masterminds/sprig/v3
- Sprig repository: https://github.com/Masterminds/sprig
- Crossplane compositions and functions: https://docs.crossplane.io/latest/composition/compositions/
- `function-go-templating` repository: https://github.com/crossplane-contrib/function-go-templating

## Parser Matrix

| Candidate | What it does well | Where it breaks for `vibe-xpls` | Fit |
| --- | --- | --- | --- |
| `goccy/go-yaml` | Parser, lexer, AST, comment retention, richer parse errors, source annotation, YAMLPath, anchor/alias support. | Not template-aware; still needs masking and a separate template catalog. Parser behavior still has to be validated against incomplete edits. | Best primary YAML AST candidate for analyzer-first work. |
| `go.yaml.in/yaml/v4` | Maintained YAML-org line, `Node` positions, comments, stream nodes, known-field and unique-key options, multi-document support. | Still a plain YAML parser; no native mixed-template model and no explicit tolerant-edit story. | Best compatibility/reference parser and fallback baseline. |
| `gopkg.in/yaml.v3` | Familiar `Node` API with line/column/comments and broad ecosystem familiarity. | Upstream repo is archived; treat as frozen legacy. Not a good new dependency. | Legacy-only compatibility target. |
| `k8s.io/apimachinery/pkg/util/yaml` | Multi-document framing, YAML-to-JSON decoding, stream decoding, `InputOffset`. | Converts through JSON-oriented helpers and does not preserve the full source structure needed for span mapping or template analysis. | Good ingestion adapter, not the core analyzer. |
| `sigs.k8s.io/yaml` | Easy YAML/JSON marshaling and unmarshaling, broad Kubernetes ecosystem compatibility. | JSON conversion path loses YAML-specific fidelity and is not template-aware. | Useful at boundaries, not for semantic analysis. |
| `text/template` + `text/template/parse` | Defines template syntax, parse trees, positions, delimiters, cloning, and error context. | Parse package is an internal/shared structure, not a general semantic layer; it does not know YAML. | Best template syntax layer, but only around masked or isolated template regions. |
| Sprig | Large helper catalog for templates, familiar to Helm-like templating workflows. | Helper surface is runtime behavior, not structure; some helpers are execution-only and should not be evaluated in the analyzer. | Register helpers as known names and signatures. |
| Crossplane `function-go-templating` helpers | Crossplane-specific helper set: `toYaml`, `fromYaml`, `include`, `getCompositeResource`, `getComposedResource`, `getExtraResources`, `getExtraResourcesFromContext`, `setResourceNameAnnotation`, `randomChoice`. | Custom helpers are part of the semantic contract, but not a substitute for parsing or span mapping. | Must be modeled in the template catalog for Crossplane fixtures. |

## Mapping Risks

- Plain YAML parsers will choke on template delimiters unless template regions are masked or isolated first.
- A rendered-only view loses original source identity, which makes diagnostics harder to trust during partial edits.
- `text/template/parse` positions are template positions, not YAML-node positions, so span maps must bridge two coordinate systems.
- Kubernetes YAML conversion helpers are good at intake and conversion, but they collapse the structure we need for comments, anchors, and source spans.
- Template execution can depend on runtime data, which makes incomplete-edit behavior fragile if the analyzer waits for a full render.
- Helper resolution is semantic, not syntactic; if the analyzer executes helpers too early, it will become brittle and side-effect prone.

## Prototype Criteria

- A masked-file pipeline can parse plain YAML, inline `{{ ... }}` actions, block templates, and multi-document files without requiring a live render context.
- The prototype preserves raw line/column mapping for YAML nodes and template nodes, including diagnostics for invalid helper names and malformed template actions.
- Incomplete edits degrade to best-effort structure and diagnostics instead of hard failure.
- The analyzer can classify Sprig helpers and Crossplane helpers from a fixed registry without executing them.
- Comments, anchors, aliases, and document boundaries remain available in the structural view.
- A later runnable spike can swap the masked view for actual template execution and produce a rendered view for comparison.

## Recommendation

Use `goccy/go-yaml` as the primary YAML AST/parser candidate for the analyzer, but wrap it in a custom mixed-source pipeline. Parse template regions separately with `text/template` / `text/template/parse`, and register Sprig plus Crossplane helper names as a catalog for validation. Keep `go.yaml.in/yaml/v4` as the compatibility baseline and `k8s.io/apimachinery/pkg/util/yaml` / `sigs.k8s.io/yaml` as ingestion adapters only.

Do not build the first design around full template execution. For incomplete edits, prefer a masked YAML skeleton with degraded diagnostics over a render-first pipeline. That fits the analyzer-first architecture and still leaves room for later runnable spikes that compare the masked view against real rendering.

Do not choose `gopkg.in/yaml.v3` for new work unless a compatibility constraint forces it. The upstream repo is archived, while the YAML org-maintained v4 line is the current direction.

## Confidence

Medium-high.

The current docs strongly support the parser capability comparison, but the best mixed YAML/template strategy still needs a runnable spike to prove span mapping and incomplete-edit behavior on real fixtures.

## Evidence That Would Change This Recommendation

- A spike shows `goccy/go-yaml` cannot tolerate partial edits or preserve spans well enough for the analyzer.
- `go.yaml.in/yaml/v4` proves easier to mask and map than `goccy/go-yaml` in real Crossplane fixtures.
- The real file corpus turns out to be render-first with little need for source-span mapping.
- Crossplane helper usage is narrower than expected, making a smaller helper catalog sufficient.
- A future template-aware YAML library appears that preserves structure, comments, and spans better than the current options.
