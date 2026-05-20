# Completion Presentation Decision

## Decision

Completion rows should stay compact and follow normal LSP editor conventions.

Use LSP completion fields as follows:

- `label`: the concise field name shown to the user and inserted by default
  when no text edit is provided.
- `kind`: the editor-rendered category or icon. Use this for generic category
  semantics such as field/property.
- `documentation`: explanatory prose, schema descriptions, and other field
  documentation.
- `detail`: generally omitted. Use it only when a short, single-line,
  item-specific value meaningfully improves selection beyond `label`, `kind`,
  and `documentation`.

Do not use `detail` for generic category text, prose documentation, source
marketing, emoji, or custom symbols.

## Reasoning

Editors decide how much of `detail` and `documentation` to show in a completion
row. Zed shows both inline, so generic detail text consumes scarce row width and
truncates more useful documentation.

The LSP `kind` field already lets the client choose a property/field icon.
Repeating that category as generic `detail` adds visual noise without improving
selection.

## Current Application

For current Crossplane YAML field completions:

- Emit `kind: 10` (`CompletionItemKind.Property`).
- Preserve analyzer field descriptions in `documentation`.
- Omit `detail`.

Future schema-derived completions may add `detail` only when it is specific and
useful, for example compact type/source/disambiguation metadata. That future use
must be deliberate, tested, and not a default filler value.
