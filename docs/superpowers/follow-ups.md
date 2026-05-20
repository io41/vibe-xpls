# Follow-Up Issues

## 2026-05-20 Zed Completion Validation

### Completion row text is truncated in Zed

Observed in Zed while completing `apiVersion` in `root/crossplane.yaml`.
The completion row shows the `detail` value as truncated text (`Cro...`) and
also shows the analyzer documentation in the same row.

Current server behavior is intentional for this slice:

- `detail`: `Crossplane YAML field`
- `documentation`: analyzer-provided plain string documentation

Initial assessment: this looks like Zed completion UI presentation rather than
an LSP server correctness bug. There may not be a user setting for this. If the
UX is not acceptable, investigate whether the language server should use a
shorter `detail`, omit `detail`, or rely on a different LSP documentation shape
for Zed.

### Root-level completion accepted under `spec` inserts at the document root

Status: investigated on 2026-05-20.

Reproduction:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
spec:
  a
```

Accepting the `apiVersion` completion under `spec` produced:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
spec:
apiVersion:
```

Expected behavior: a completion accepted under `spec` should either insert a
valid field for the `spec` object at the correct indentation or not be offered.
It should not insert a root field at document-root indentation from a nested
position.

Investigation evidence: the language server response itself offers the root
field while the cursor is under `spec`, so this is not a Zed insertion bug. For
the `a` prefix in the reproduction above, the server returns:

```json
{
  "label": "apiVersion",
  "kind": 10,
  "detail": "Crossplane YAML field",
  "documentation": "API version of the Configuration metadata resource.",
  "textEdit": {
    "range": {
      "start": { "line": 5, "character": 0 },
      "end": { "line": 5, "character": 3 }
    },
    "newText": "apiVersion:"
  },
  "insertTextMode": 1
}
```

Root cause: `CompletionAtOffset` detects the cursor as a nested child of
`spec`, but `completionParentPaths` deliberately falls back from `spec` to the
document root when no `spec.<prefix>` completion matches. Because existing
siblings are not filtered, root fields that already exist are still offered as
fallback candidates. Under `spec`, current observed prefixes behave as follows:

- blank prefix: `dependsOn` with nested indentation
- `d`: `dependsOn` with nested indentation
- `a`: existing root `apiVersion` with root indentation
- `k`: existing root `kind` with root indentation
- `m`: existing root `metadata` with root indentation
- `s`: existing root `spec` with root indentation

Likely fix direction: keep the useful root-key dedent path for accidentally
indented missing root keys, but prevent fallback root completions from offering
root keys that already exist in the current YAML document. Add regression
coverage for the `spec:\n  a` case before changing completion fallback logic.

### Zed breadcrumb or outline duplicates YAML path parts

Observed Zed breadcrumb from the same file:

```text
root/crossplane.yaml > spec a > spec a
```

Expected behavior: breadcrumbs should not duplicate the same YAML path segment.

Investigation boundary: determine whether this comes from Zed's tree-sitter
outline for the `Crossplane YAML` language, from the extension language
configuration, or from language-server document symbols if those are added in a
future slice. The current `vibe-xpls` LSP server does not intentionally expose
YAML outline symbols.
