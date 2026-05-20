# Completion Presentation Design

## Scope

Improve how existing completion items are presented to LSP clients. Do not add, remove, or rename completion candidates.

This slice is limited to LSP completion item metadata for fields the analyzer already returns today:

- Keep `label` unchanged.
- Keep `textEdit` and `insertTextMode` behavior unchanged.
- Add `kind: 10`, the LSP wire value for `CompletionItemKind.Property`, for every emitted completion item.
- Omit generic `detail` values.
- Preserve the existing full `documentation` value as the same plain string wire shape used today.

## Non-Goals

- Do not manually add Crossplane fields.
- Do not manually add or rewrite Crossplane field documentation.
- Do not expand built-in schemas.
- Do not implement generated schema ingestion in this slice.
- Do not change completion trigger behavior, indentation behavior, text edit ranges, or candidate filtering.

Completion catalogs and Crossplane field documentation change over time. Future work that expands field coverage must come from generated schema or documentation sources, not hand-maintained field lists.

## Architecture

The analyzer remains the source of completion candidates and schema-derived documentation. The LSP adapter remains responsible for protocol formatting.

Implementation should prefer keeping presentation-only metadata in `internal/lsp`:

- `completionItem.kind` uses the LSP `CompletionItemKind.Property` wire value, `10`.
- `completionItem.detail` is omitted unless a future slice adds short, item-specific metadata.
- `completionItem.documentation` continues to carry the analyzer-provided documentation as a plain string.

The implementation should not derive `detail` from schema paths or field documentation in this slice. Generic category text belongs in `kind`, and explanatory prose belongs in `documentation`.

## Data Flow

1. Analyzer computes completion candidates exactly as it does today.
2. LSP completion handling maps each analyzer item to an LSP completion item.
3. The mapped item includes `label`, `kind`, `documentation`, `textEdit`, and `insertTextMode` when supported.
4. The client remains responsible for visual truncation, but the server avoids adding generic detail text that crowds compact completion rows.

## Error Handling

If a completion item lacks documentation, the LSP adapter should omit `documentation` as it does today.

`detail` is omitted unless future item-specific metadata is deliberately added.

## Testing

Add focused LSP tests that prove existing completion candidates now include presentation metadata:

- Every returned completion item has `kind` set to JSON number `10`.
- Every returned completion item omits generic `detail`.
- Existing `documentation` is still present for a documented field.
- Existing completion labels remain unchanged for the tested request.
- Existing `textEdit` and `insertTextMode` assertions continue to pass.

No tests should assert new Crossplane fields or new field descriptions in this slice.

## Acceptance Criteria

- Current completion candidates are unchanged.
- LSP completion items expose `kind: 10` for every emitted item.
- LSP completion items omit generic `detail`.
- Existing documentation remains available as documentation, not overloaded into detail.
- The slice has no manual Crossplane field catalog additions.
- `go test ./...` passes.
