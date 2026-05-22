# Completion UX Cleanup Design

## Summary

Manual Zed validation of the generated completion foundation found two blocking
completion UX issues:

- Completion documentation is Markdown content, but the LSP adapter sends it as
  a plain string, so Zed renders Markdown markers such as `_Required_` literally
  in completion popups.
- Completion works inside existing array-item objects, such as
  `spec.pipeline[].functionRef.name`, but it does not complete the first mapping
  key after a YAML sequence marker, such as `- f` under `spec.pipeline`.

This cleanup is a blocking `Current` roadmap item. It must be completed before
moving on to function input schema dispatch or graph-aware completions.

## Goals

- Render completion documentation as Markdown in Zed.
- Preserve compact completion rows: property kind icon, no generic `detail`, and
  stable `sortText`.
- Complete first keys in schema-backed YAML array items.
- Keep accepted completion edits single-line and indentation-preserving.
- Cover the behavior with analyzer and LSP protocol tests before repeating
  manual Zed validation.

## Non-Goals

- No snippets.
- No multi-line completion insertion.
- No automatic child templates.
- No value completions.
- No function input schema dispatch under `spec.pipeline[].input`.
- No graph-aware relationship completions.
- No new schema bundle data.

## Completion Documentation

The analyzer continues to store normalized documentation as a Go string. The
string content is Markdown; it is not plain prose. The LSP adapter is
responsible for choosing the wire shape that lets clients render that Markdown.

For completion items with non-empty documentation, the LSP adapter must send:

```json
"documentation": {
  "kind": "markdown",
  "value": "..."
}
```

Completion items with empty documentation must continue to omit the
`documentation` field.

Hover already sends Markdown `MarkupContent`; this change aligns completion
documentation with hover semantics.

## Array-Item Key Completion

The analyzer should treat a YAML sequence item that is starting a mapping key as
a valid key-completion context.

Example:

```yaml
spec:
  pipeline:
    - f
```

At the cursor after `f`, completion should use the schema parent
`spec.pipeline[]`, filter candidates by prefix `f`, and offer `functionRef`. The
text edit should replace only the key prefix after the sequence marker and
produce valid YAML:

```yaml
spec:
  pipeline:
    - functionRef:
```

Blank sequence-item key starts should also complete:

```yaml
spec:
  pipeline:
    -
```

and:

```yaml
spec:
  pipeline:
    -
```

The same behavior applies when the cursor is after a sequence marker followed by
a space. At those positions, completion should offer immediate children of
`spec.pipeline[]`, including `functionRef`, `step`, `input`, `credentials`, and
release-specific children such as v2 `requirements`.

This behavior only applies when the sequence item is in a mapping-key position.
The analyzer must still avoid completions in scalar values, block scalars,
template actions, malformed YAML contexts, and lines that already contain a
colon after the cursor.

## Data Flow

1. Zed requests `textDocument/completion` at a cursor position.
2. The LSP server converts the position to an analyzer byte offset.
3. The analyzer identifies a mapping-key completion context.
4. If the current line starts a sequence item, the analyzer separates:
   - the line indentation before `-`,
   - the sequence marker and following space,
   - the partial key prefix after the marker.
5. The analyzer resolves the schema parent to the array item path, such as
   `spec.pipeline[]`.
6. Existing schema lookup, release resolution, prefix filtering, required-key
   ordering, and text-edit generation continue to apply.
7. The LSP server serializes completion documentation as Markdown
   `MarkupContent`.

## Error Handling

If the sequence-item context cannot be mapped to a stable schema parent, the
server should keep the existing conservative behavior and return no completion
items. It should not invent schema paths.

Existing suppression reasons continue to apply:

- `malformed-yaml-context`
- `unstable-template-path`
- `missing-root-gvk`
- `unknown-gvk`
- `no-schema-for-release`
- `unsupported-schema-shape`
- `bundle-disabled`

This cleanup does not add document diagnostics for completion suppression or
documentation wire-shape issues.

## Testing

Analyzer tests should cover:

- `spec.pipeline:\n  - f` completes `functionRef`.
- `spec.pipeline:\n  - ` offers immediate children of `spec.pipeline[]`.
- The text edit for `- f` replaces only `f` and produces `- functionRef:`.
- Existing nested completion still works for
  `spec.pipeline[].functionRef.name`.
- Existing scalar-value, block-scalar, template-action, malformed-context, and
  existing-root-key filtering behavior does not regress.

LSP protocol tests should cover:

- Completion documentation is encoded as
  `{"kind":"markdown","value":"..."}`.
- Completion items still emit `kind == 10`.
- Completion items still omit generic `detail`.
- Completion items with no docs still omit `documentation`.
- Zed-style text edits for array-item key completion have the expected range and
  `newText`.

Manual Zed validation should confirm:

- Completion popups render `_Required_` and `_Type: ..._` as Markdown formatting,
  not literal underscores.
- Completion docs remain readable in Zed's completion popup.
- Completing `- f` under `spec.pipeline` inserts `functionRef:` at the correct
  indentation.
- Completing `name` under an existing `functionRef:` still works.
- v1.20 and v2 release-specific completion sets remain unchanged.

## Acceptance Criteria

- Automated analyzer and LSP tests pass.
- `go test ./...` passes.
- Manual Zed validation passes for Markdown completion docs and array-item key
  completion.
- The roadmap no longer lists these two findings as blocking Current cleanup
  once validation evidence is recorded.
