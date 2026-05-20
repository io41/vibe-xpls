# Completion Presentation Design

## Scope

Improve how existing completion items are presented to LSP clients. Do not add, remove, or rename completion candidates.

This slice is limited to LSP completion item metadata for fields the analyzer already returns today:

- Keep `label` unchanged.
- Keep `textEdit` and `insertTextMode` behavior unchanged.
- Add `kind: Property` for YAML mapping-key completions.
- Add a concise `detail` value for compact completion-list display.
- Preserve the existing full `documentation` value.

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

- `completionItem.kind` uses the LSP `CompletionItemKind.Property` value.
- `completionItem.detail` uses short stable text that does not pretend to be schema truth.
- `completionItem.documentation` continues to carry the analyzer-provided documentation.

If the implementation needs a helper, it should derive detail text from existing analyzer metadata such as `Path`, not from new hand-written Crossplane schema facts.

## Data Flow

1. Analyzer computes completion candidates exactly as it does today.
2. LSP completion handling maps each analyzer item to an LSP completion item.
3. The mapped item includes `label`, `kind`, `detail`, `documentation`, `textEdit`, and `insertTextMode` when supported.
4. The client remains responsible for visual truncation, but the server gives it separate short and long fields so compact rows do not need to carry full documentation text.

## Error Handling

If a completion item lacks documentation, the LSP adapter should omit `documentation` as it does today.

If a completion item lacks enough metadata to derive a specific detail string, it should use a generic detail such as `Crossplane YAML field`.

## Testing

Add focused LSP tests that prove existing completion candidates now include presentation metadata:

- A known field completion has `kind` set to `Property`.
- A known field completion has non-empty `detail`.
- Existing `documentation` is still present for a documented field.
- Existing `textEdit` and `insertTextMode` assertions continue to pass.

No tests should assert new Crossplane fields or new field descriptions in this slice.

## Acceptance Criteria

- Current completion candidates are unchanged.
- LSP completion items expose `kind: Property`.
- LSP completion items expose concise `detail`.
- Existing documentation remains available as documentation, not overloaded into detail.
- The slice has no manual Crossplane field catalog additions.
- `go test ./...` passes.
