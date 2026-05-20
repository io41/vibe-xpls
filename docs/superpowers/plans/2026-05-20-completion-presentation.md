# Completion Presentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add LSP presentation metadata to existing completion items without changing completion candidates.

**Architecture:** Keep completion candidate selection in `internal/analyzer` unchanged. Add presentation-only fields in `internal/lsp/server.go` when mapping analyzer completions to LSP completion items. Prove the wire contract through focused LSP tests in `internal/lsp/server_test.go`.

**Tech Stack:** Go, stdlib JSON encoding, LSP 3.17 completion item wire shape, existing `go test` test suite.

---

## File Structure

- Modify `internal/lsp/server.go`: add completion item `kind` and `detail` fields, constants for the LSP wire value and detail text, and populate those fields during completion mapping.
- Modify `internal/lsp/server_test.go`: add a focused test for completion presentation metadata and a small helper for extracting completion labels.

No analyzer files should change. No schema files or built-in field catalogs should change.

## Task 1: Add Failing LSP Completion Presentation Test

**Files:**
- Modify: `internal/lsp/server_test.go`

- [x] **Step 1: Add `reflect` to the test imports**

Change the import block in `internal/lsp/server_test.go` from:

```go
import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
```

to:

```go
import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
```

- [x] **Step 2: Add the failing presentation metadata test**

Insert this test after `TestHoverAndCompletionUseAnalyzer` in `internal/lsp/server_test.go`:

```go
func TestCompletionItemsIncludePresentationMetadata(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-presentation.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n\n"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": map[string]any{}}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	items := asSlice(t, completion["items"])
	if got, want := completionLabelsForTest(t, items), []string{"apiVersion", "kind", "metadata", "spec"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("completion labels = %#v, want %#v", got, want)
	}
	for _, raw := range items {
		item := asMap(t, raw)
		if item["kind"] != float64(10) {
			t.Fatalf("completion item %#v kind = %#v, want LSP Property 10", item["label"], item["kind"])
		}
		if item["detail"] != "Crossplane YAML field" {
			t.Fatalf("completion item %#v detail = %#v, want Crossplane YAML field", item["label"], item["detail"])
		}
	}
	item := completionItemByLabelForTest(t, items, "apiVersion")
	if item["documentation"] != "API version of the Composition resource." {
		t.Fatalf("apiVersion documentation = %#v, want existing analyzer documentation", item["documentation"])
	}
	if item["detail"] == item["documentation"] {
		t.Fatalf("detail was copied from documentation: %#v", item)
	}
}
```

- [x] **Step 3: Add the completion label helper**

Insert this helper near `itemsContainLabel` and `completionItemByLabelForTest` in `internal/lsp/server_test.go`:

```go
func completionLabelsForTest(t *testing.T, items []any) []string {
	t.Helper()
	labels := make([]string, 0, len(items))
	for _, raw := range items {
		item := asMap(t, raw)
		label, ok := item["label"].(string)
		if !ok {
			t.Fatalf("completion item label = %#v, want string", item["label"])
		}
		labels = append(labels, label)
	}
	return labels
}
```

- [x] **Step 4: Run the focused test and verify it fails**

Run:

```bash
go test ./internal/lsp -run TestCompletionItemsIncludePresentationMetadata -count=1
```

Expected: FAIL because current completion items do not include `kind` or `detail`.

- [x] **Step 5: Commit the failing test**

Run:

```bash
git add internal/lsp/server_test.go
git commit --no-gpg-sign -m "test: add completion presentation metadata coverage"
```

Expected: commit succeeds.

## Task 2: Add Completion Presentation Metadata

**Files:**
- Modify: `internal/lsp/server.go`

- [x] **Step 1: Add completion presentation constants**

Change the constants near the top of `internal/lsp/server.go` from:

```go
const (
	methodNotFound = -32601
	invalidParams  = -32602

	// LSP InsertTextMode.asIs.
	insertTextModeAsIs = 1
)
```

to:

```go
const (
	methodNotFound = -32601
	invalidParams  = -32602

	// LSP InsertTextMode.asIs.
	insertTextModeAsIs = 1

	// LSP CompletionItemKind.Property.
	completionItemKindProperty = 10

	completionItemDetailCrossplaneYAMLField = "Crossplane YAML field"
)
```

- [x] **Step 2: Add `kind` and `detail` to the LSP completion item type**

Change `completionItem` in `internal/lsp/server.go` from:

```go
type completionItem struct {
	Label          string    `json:"label"`
	Documentation  string    `json:"documentation,omitempty"`
	TextEdit       *textEdit `json:"textEdit,omitempty"`
	InsertTextMode int       `json:"insertTextMode,omitempty"`
}
```

to:

```go
type completionItem struct {
	Label          string    `json:"label"`
	Kind           int       `json:"kind"`
	Detail         string    `json:"detail"`
	Documentation  string    `json:"documentation,omitempty"`
	TextEdit       *textEdit `json:"textEdit,omitempty"`
	InsertTextMode int       `json:"insertTextMode,omitempty"`
}
```

- [x] **Step 3: Populate presentation metadata during completion mapping**

Change the start of the mapping loop in `handleCompletion` from:

```go
	items := make([]completionItem, 0, len(completion.Items))
	for _, item := range completion.Items {
		out := completionItem{Label: item.Label, Documentation: item.Documentation}
		if item.TextEdit != nil {
```

to:

```go
	items := make([]completionItem, 0, len(completion.Items))
	for _, item := range completion.Items {
		out := completionItem{
			Label:         item.Label,
			Kind:          completionItemKindProperty,
			Detail:        completionItemDetailCrossplaneYAMLField,
			Documentation: item.Documentation,
		}
		if item.TextEdit != nil {
```

- [x] **Step 4: Format the changed Go files**

Run:

```bash
gofmt -w internal/lsp/server.go internal/lsp/server_test.go
```

Expected: command exits 0 and only formatting changes are applied.

- [x] **Step 5: Run the focused test and verify it passes**

Run:

```bash
go test ./internal/lsp -run TestCompletionItemsIncludePresentationMetadata -count=1
```

Expected: PASS.

- [x] **Step 6: Run existing completion edit regression tests**

Run:

```bash
go test ./internal/lsp -run 'TestCompletionItemsIncludePlainTextEdits|TestCompletionTextEditCorrectsIndentedRootKey|TestCompletionTextEditPreservesZeroWidthInsertionRange' -count=1
```

Expected: PASS, proving `textEdit` and `insertTextMode` behavior did not regress.

- [x] **Step 7: Commit the implementation**

Run:

```bash
git add internal/lsp/server.go internal/lsp/server_test.go
git commit --no-gpg-sign -m "fix: add completion presentation metadata"
```

Expected: commit succeeds.

## Task 3: Full Verification

**Files:**
- Verify: `internal/lsp/server.go`
- Verify: `internal/lsp/server_test.go`

- [x] **Step 1: Run the full Go test suite**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [x] **Step 2: Confirm no analyzer or schema catalog changes**

Run:

```bash
git diff --name-only HEAD~2..HEAD
```

Expected output contains only:

```text
internal/lsp/server.go
internal/lsp/server_test.go
```

- [x] **Step 3: Inspect the implementation diff**

Run:

```bash
git diff --check HEAD~2..HEAD
```

Expected: command exits 0 with no whitespace errors.

- [x] **Step 4: Confirm release impact**

Run:

```bash
git log --oneline -2
```

Expected: the implementation commit uses `fix: add completion presentation metadata`, so Release Please will prepare a patch release.

## Self-Review Checklist

- Spec requirement "current completion candidates are unchanged" is covered by the exact-label assertion in Task 1 and the no-analyzer-change check in Task 3.
- Spec requirement "kind: 10 for every emitted item" is covered by the loop assertion in Task 1.
- Spec requirement "detail: Crossplane YAML field for every emitted item" is covered by the loop assertion in Task 1.
- Spec requirement "documentation remains documentation and is not overloaded into detail" is covered by the `apiVersion` documentation assertion and detail/documentation inequality assertion in Task 1.
- Spec requirement "no manual Crossplane field catalog additions" is covered by the file-change check in Task 3.
- Spec requirement "`go test ./...` passes" is covered by Task 3.
