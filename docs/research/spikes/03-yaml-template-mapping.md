# YAML Template Mapping Spike

## Summary

This spike validates a dependency-light mapping path for mixed YAML and Go-template files. The Go code records template action spans, builds a same-length masked YAML view by replacing non-newline action bytes with `x`, runs a minimal structural YAML check over the masked view, and maps diagnostics back to original source positions.

The useful result is that byte offset, line, and column mapping can be identity-preserving when the mask keeps the same length and newline layout. This is enough to prove the analyzer can separate template-owned diagnostics from YAML-owned diagnostics before selecting a full YAML parser.

## Fixture Cases

Fixture: `spikes/yaml-template-mapping/fixtures/mixed.yaml.tmpl`

- Scalar values: `apiVersion: {{ .APIVersion }}`, `kind: {{ .Kind }}`, and `name: {{ .Name | quote }}`.
- List items: `- {{ .PrimaryTag }}` and a quoted scalar containing `{{ .Suffix }}`.
- Block scalars: `templated line {{ .BlockAction }}` inside `settings: |`.
- Keys: `{{ .LabelKey }}: "enabled"` and document root key `{{ .DocumentKey }}:`.
- Multi-document output: two `---` separators produce three document sections.
- YAML diagnostic fixture: `invalidLineWithoutColon` is outside any template span.
- Template diagnostic fixture: `badTemplate: {{ if }}` has a closed action span but an invalid control action.

## Commands Run

```text
gofmt -w spikes/yaml-template-mapping/mapping.go spikes/yaml-template-mapping/mapping_test.go
```

```text
cd spikes/yaml-template-mapping && go test ./...
```

The first sandboxed test attempt could not write to the default Go build cache:

```text
open <user-cache-dir>/go-build/...: operation not permitted
```

Rerunning the same test command with normal Go cache access passed:

```text
ok  	github.com/io41/vibe-xpls/spikes/yaml-template-mapping	5.251s
```

## Mapping Results

The fixture produces a masked YAML view with the same byte length as the original source. Example shape:

```yaml
apiVersion: xxxxxxxxxxxxxxxx
kind: xxxxxxxxxxxx
metadata:
  name: xxxxxxxxxxxxxxxxxxx
  labels:
    xxxxxxxxxxxxxx: "enabled"
spec:
  tags:
    - xxxxxxxxxxxxxxxxx
    - "static-xxxxxxxxxxxx"
  settings: |
    literal before
    templated line xxxxxxxxxxxxxxxxxx
    literal after
---
xxxxxxxxxxxxxxxxx:
  enabled: true
invalidLineWithoutColon
---
badTemplate: xxxxxxxx
```

The minimal YAML check reports one diagnostic from the masked view and maps it back to original source line 18, column 1, or zero-based line 17, column 0. The mapped span is exactly `invalidLineWithoutColon`, and it is outside every recorded template action span.

The template action validator reports one diagnostic inside `{{ if }}`. The diagnostic maps to the `if` token at original source line 20, column 17, or zero-based line 19, column 16. The diagnostic span is contained by the recorded `{{ if }}` action span.

Tests prove both cases:

- `TestMaskedViewPreservesCoordinatesAndParsesYAML` verifies same-length masking, masked YAML parsing, and the outside-template YAML diagnostic mapping.
- `TestDiagnosticsMapToOriginalTemplateAndYAMLPositions` verifies the outside-template YAML diagnostic and the inside-template diagnostic.

## Degradation Rules

- The spike uses byte offsets and byte columns. That is acceptable for the ASCII fixture, but production code should define UTF-16, UTF-8 byte, and rune conversions explicitly at LSP boundaries.
- The masked YAML checker is structural only. It recognizes mappings, list items, document markers, and block scalar indentation well enough for this spike. It does not validate full YAML grammar, anchors, tags, flow style, duplicate keys, schema, or Crossplane semantics.
- Template action parsing is delimiter-based. The first `}}` closes an action, and nested delimiter semantics are not modeled.
- Closed template actions can be masked even when their internal template syntax is invalid. That lets YAML analysis continue while template diagnostics remain attached to the original action span.
- If a future YAML parser reports a diagnostic wholly inside a template span, the analyzer should prefer a template diagnostic or suppress the YAML diagnostic as mask noise.
- If a YAML diagnostic overlaps both template and non-template bytes, the analyzer should mark it low confidence and attach it to the nearest non-template boundary instead of claiming an exact semantic location.
- If a template action is unterminated, the analyzer should report a template delimiter diagnostic first and avoid trusting downstream YAML diagnostics after the opening delimiter.

## Decision Impact

The spike supports an analyzer pipeline that records template spans before YAML parsing, masks template bytes into a YAML-safe view, and maps parser diagnostics back to original source positions. This keeps editor diagnostics stable for ordinary YAML errors while avoiding false YAML errors from template expressions in keys, scalar values, list items, block scalars, and multi-document output.

For production, the masking and source map approach is worth carrying forward, but the minimal YAML checker should be replaced by a real YAML parser. The same-length mask is a strong default because it makes diagnostic mapping cheap and predictable; a richer source map is only needed if later masking rewrites lengths or normalizes line endings.
