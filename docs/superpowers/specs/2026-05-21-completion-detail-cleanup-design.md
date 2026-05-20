# Completion Detail Cleanup Design

## Scope

Remove the generic `detail` value from LSP completion items.

This slice changes presentation metadata only. It does not add, remove, or
rename completion candidates, and it does not change text edits, indentation, or
trigger behavior.

## Requirements

- Keep `label` unchanged.
- Keep `kind: 10` for emitted field completion items.
- Omit `detail` for current Crossplane YAML field completions.
- Keep analyzer-provided field descriptions in `documentation`.
- Keep `textEdit` and `insertTextMode` behavior unchanged.

## Non-Goals

- Do not add generated schema ingestion.
- Do not add hand-maintained field descriptions.
- Do not add item-specific detail values in this slice.
- Do not change Zed settings or extension behavior.

## Rationale

`detail` should be generally avoided unless it is short, single-line,
item-specific, and meaningfully improves selection. The generic field category
is already represented by `kind`, and long prose belongs in `documentation`.

Zed displays `detail` and `documentation` in the completion row, so a generic
detail value makes the row more crowded without adding useful information.

## Testing

Update the focused LSP completion presentation test to assert:

- labels are unchanged;
- every returned item still has `kind: 10`;
- `detail` is omitted;
- existing documentation remains present for documented fields.

Run `go test ./...`.
