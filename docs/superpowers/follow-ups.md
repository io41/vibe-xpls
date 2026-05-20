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

Investigation boundary: start in the LSP completion request path and the
analyzer path detection for the cursor position. Confirm the server response
`textEdit.range` and `newText` before assuming this is a Zed client issue.

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
