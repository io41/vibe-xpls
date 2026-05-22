# Completion UX Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the two blocking Zed validation findings for generated completions: Markdown completion docs must render as Markdown, and schema-backed YAML array items must complete their first mapping key.

**Architecture:** Keep analyzer documentation as normalized Markdown strings and change only the LSP adapter wire shape to Markdown `MarkupContent`. Extend analyzer completion-context detection so a sequence item that is starting a mapping key maps to the array item schema path while preserving conservative suppression in scalar, template, malformed, and unsupported contexts.

**Tech Stack:** Go 1.26.3, existing analyzer package, existing LSP JSON-RPC server, LSP `CompletionItem.documentation` as `MarkupContent`.

---

## File Structure

- Modify: `internal/lsp/server.go`
  Change completion item documentation from plain string output to Markdown `MarkupContent` when non-empty.
- Modify: `internal/lsp/server_test.go`
  Update completion presentation assertions and add LSP text-edit coverage for array-item first-key completion.
- Modify: `internal/analyzer/completion.go`
  Extend completion context parsing for YAML sequence-item mapping keys and keep normal mapping-key behavior unchanged.
- Modify: `internal/analyzer/analyzer_test.go`
  Add analyzer-level tests for `- f`, `- `, and `-` array-item key completion plus edit ranges.
- Modify: `PROJECT_ROADMAP.md`
  After implementation and manual Zed validation, remove the explicit blocking cleanup findings from `Current`.

---

### Task 1: Markdown Completion Documentation Wire Shape

**Files:**
- Modify: `internal/lsp/server.go`
- Modify: `internal/lsp/server_test.go`

- [ ] **Step 1: Update the failing presentation test expectation**

In `internal/lsp/server_test.go`, update the final `documentation` assertion inside `TestCompletionItemsIncludePresentationMetadata` from the current plain-string check to this Markdown `MarkupContent` check:

```go
	item := completionItemByLabelForTest(t, items, "apiVersion")
	wantDocumentation := "API version of the Composition resource.\n\n_Type: string_"
	documentation := asMap(t, item["documentation"])
	if documentation["kind"] != "markdown" || documentation["value"] != wantDocumentation {
		t.Fatalf("apiVersion documentation = %#v, want markdown %q", item["documentation"], wantDocumentation)
	}
	metadata := completionItemByLabelForTest(t, items, "metadata")
	if _, ok := metadata["documentation"]; ok {
		t.Fatalf("metadata documentation = %#v, want omitted for empty docs", metadata["documentation"])
	}
```

- [ ] **Step 2: Run the focused LSP test and confirm it fails**

Run:

```sh
go test ./internal/lsp -run TestCompletionItemsIncludePresentationMetadata
```

Expected: fail because `documentation` is currently encoded as a JSON string, not an object with `kind` and `value`.

- [ ] **Step 3: Change the LSP completion item documentation type**

In `internal/lsp/server.go`, replace the `completionItem` definition with:

```go
type completionItem struct {
	Label          string         `json:"label"`
	Kind           int            `json:"kind"`
	Documentation  *markupContent `json:"documentation,omitempty"`
	SortText       string         `json:"sortText,omitempty"`
	TextEdit       *textEdit      `json:"textEdit,omitempty"`
	InsertTextMode int            `json:"insertTextMode,omitempty"`
}
```

- [ ] **Step 4: Populate completion docs as Markdown only when non-empty**

In `handleCompletion`, change the item mapping loop so `Documentation` is assigned after constructing `out`:

```go
	for _, item := range completion.Items {
		out := completionItem{
			Label:    item.Label,
			Kind:     completionItemKindProperty,
			SortText: item.SortText,
		}
		if item.Documentation != "" {
			out.Documentation = &markupContent{Kind: "markdown", Value: item.Documentation}
		}
		if item.TextEdit != nil {
			out.TextEdit = &textEdit{
				Range:   s.rangeFromTextEditSpan(snapshot.Text, item.TextEdit.Replace),
				NewText: item.TextEdit.NewText,
			}
			if s.completionInsertTextModeAsIsOK {
				out.InsertTextMode = insertTextModeAsIs
			}
		}
		items = append(items, out)
	}
```

- [ ] **Step 5: Run the focused LSP test and confirm it passes**

Run:

```sh
go test ./internal/lsp -run TestCompletionItemsIncludePresentationMetadata
```

Expected: pass.

- [ ] **Step 6: Commit the documentation wire-shape change**

Run:

```sh
git add internal/lsp/server.go internal/lsp/server_test.go
git commit -m "fix: render completion docs as markdown"
```

---

### Task 2: Analyzer Array-Item First-Key Completion

**Files:**
- Modify: `internal/analyzer/analyzer_test.go`
- Modify: `internal/analyzer/completion.go`

- [ ] **Step 1: Add failing analyzer tests for sequence-item key starts**

Add these tests near the existing completion-at-offset tests in `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionAtOffsetCompletesFirstArrayItemKey(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-array-item-key.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - f"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "functionRef")
	if !ok {
		t.Fatalf("completion missing functionRef: %#v", completion.Items)
	}
	if containsCompletion(completion.Items, "step") {
		t.Fatalf("prefix-filtered completion included step: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("functionRef completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "functionRef:" {
		t.Fatalf("new text = %q, want functionRef:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "f"), End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetCompletesBlankArrayItemKey(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-array-item-blank-key.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - "
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	for _, label := range []string{"functionRef", "step", "input", "credentials"} {
		if !containsCompletion(completion.Items, label) {
			t.Fatalf("completion missing %s: %#v", label, completion.Items)
		}
	}
	item, ok := completionItemByLabel(completion.Items, "functionRef")
	if !ok {
		t.Fatalf("completion missing functionRef: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("functionRef completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "functionRef:" {
		t.Fatalf("new text = %q, want functionRef:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: len(text), End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetCompletesArrayItemKeyAfterBareDash(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-array-item-dash-key.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    -"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "functionRef")
	if !ok {
		t.Fatalf("completion missing functionRef: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("functionRef completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != " functionRef:" {
		t.Fatalf("new text = %q, want leading space before functionRef", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: len(text), End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetDoesNotFallbackFromArrayItemToParentObject(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-array-item-no-parent-fallback.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - m"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if containsCompletion(completion.Items, "mode") {
		t.Fatalf("array item completion fell back to spec.mode: %#v", completion.Items)
	}
}
```

- [ ] **Step 2: Run the focused analyzer tests and confirm they fail**

Run:

```sh
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffset(Completes(FirstArrayItemKey|BlankArrayItemKey|ArrayItemKeyAfterBareDash)|DoesNotFallbackFromArrayItemToParentObject)'
```

Expected: fail because `completionContextAtOffset` currently rejects lines whose trimmed prefix starts with `-`.

- [ ] **Step 3: Extend `completionContext` with schema and edit-prefix metadata**

In `internal/analyzer/completion.go`, replace `completionContext` with:

```go
type completionContext struct {
	parentPath       string
	schemaParentPath string
	prefix           string
	rootOccurrence   PathOccurrence
	replace          Span
	indent           string
	newTextPrefix    string
	useNewTextPrefix bool
	allowParentPaths bool
}
```

- [ ] **Step 4: Add sequence-item helper types and functions**

Add these helpers below `completionContextAtOffset` in `internal/analyzer/completion.go`:

```go
type sequenceItemKeyContext struct {
	prefix           string
	replace          Span
	newTextPrefix    string
	useNewTextPrefix bool
}

func sequenceItemKeyCompletionContext(text string, indentEnd, offset int) (sequenceItemKeyContext, bool) {
	if indentEnd >= offset || text[indentEnd] != '-' {
		return sequenceItemKeyContext{}, false
	}
	keyStart := indentEnd + 1
	newTextPrefix := " "
	if keyStart < offset {
		switch text[keyStart] {
		case ' ':
			keyStart++
			newTextPrefix = ""
		case '\t':
			return sequenceItemKeyContext{}, false
		default:
			return sequenceItemKeyContext{}, false
		}
	}
	rawPrefix := text[keyStart:offset]
	if strings.TrimSpace(rawPrefix) != rawPrefix {
		return sequenceItemKeyContext{}, false
	}
	prefix := strings.TrimSpace(rawPrefix)
	if !isBareCompletionKeyPrefix(prefix) || !isBareCompletionKeyPrefix(rawPrefix) {
		return sequenceItemKeyContext{}, false
	}
	return sequenceItemKeyContext{
		prefix:           prefix,
		replace:          Span{Start: keyStart, End: offset},
		newTextPrefix:    newTextPrefix,
		useNewTextPrefix: true,
	}, true
}

func arrayItemSchemaParentPath(parentPath string) string {
	if parentPath == "" {
		return ""
	}
	return parentPath + "[0]"
}

func completionTextEditNewText(context completionContext, item CompletionItem) string {
	if context.useNewTextPrefix {
		return context.newTextPrefix + item.Label + ":"
	}
	return completionItemIndent(item) + item.Label + ":"
}
```

- [ ] **Step 5: Update `completionContextAtOffset` to allow sequence-item key contexts**

Inside `completionContextAtOffset`, replace the block from `rawPrefix := text[indentEnd:offset]` through the returned `completionContext` with this structure:

```go
	rawPrefix := text[indentEnd:offset]
	sequenceContext, sequenceOK := sequenceItemKeyCompletionContext(text, indentEnd, offset)
	keyCandidate := rawPrefix
	prefix := strings.TrimSpace(rawPrefix)
	replace := Span{Start: lineStart, End: offset}
	newTextPrefix := ""
	useNewTextPrefix := false
	if sequenceOK {
		keyCandidate = text[sequenceContext.replace.Start:sequenceContext.replace.End]
		prefix = sequenceContext.prefix
		replace = sequenceContext.replace
		newTextPrefix = sequenceContext.newTextPrefix
		useNewTextPrefix = sequenceContext.useNewTextPrefix
	} else if strings.HasPrefix(strings.TrimLeft(rawPrefix, " \t"), "-") {
		return completionContext{}, "", false
	}
	if offsetInTemplateActionForCompletion(parsed, offset) {
		return completionContext{}, SuppressionUnstableTemplatePath, false
	}
	afterCursor := text[offset:lineEnd]
	if colon := strings.Index(afterCursor, ":"); colon >= 0 {
		return completionContext{}, "", false
	} else if strings.TrimSpace(afterCursor) != "" {
		return completionContext{}, "", false
	}
	keyCandidate = strings.TrimSpace(keyCandidate)
	if !isBareCompletionKeyPrefix(prefix) || !isBareCompletionKeyPrefix(keyCandidate) {
		return completionContext{}, "", false
	}

	parentPath, rootOccurrence, ok := parentCompletionContext(parsed, lineStart, indentEnd-lineStart)
	if !ok {
		return completionContext{}, "", false
	}
	schemaParentPath := parentPath
	if sequenceOK {
		schemaParentPath = arrayItemSchemaParentPath(parentPath)
	}
	return completionContext{
		parentPath:       parentPath,
		schemaParentPath: schemaParentPath,
		prefix:           prefix,
		rootOccurrence:   rootOccurrence,
		replace:          replace,
		indent:           text[lineStart:indentEnd],
		newTextPrefix:    newTextPrefix,
		useNewTextPrefix: useNewTextPrefix,
		allowParentPaths: !sequenceOK,
	}, "", true
```

- [ ] **Step 6: Use `schemaParentPath` and context-aware edit text in `CompletionAtOffset`**

In `CompletionAtOffset`, replace the loop header and stability check:

```go
	for i, parentPath := range completionParentPaths(context.parentPath) {
		if parentPath != "" && !parsed.IsStablePath(parentPath) {
			continue
		}
```

with:

```go
	parentPaths := completionParentPaths(context.schemaParentPath)
	if !context.allowParentPaths && len(parentPaths) > 1 {
		parentPaths = parentPaths[:1]
	}
	for i, parentPath := range parentPaths {
		stabilityPath := parentPath
		if i == 0 {
			stabilityPath = context.parentPath
		}
		if stabilityPath != "" && !parsed.IsStablePath(stabilityPath) {
			continue
		}
```

Then replace the text edit assignment:

```go
			NewText: completionItemIndent(completion.Items[i]) + completion.Items[i].Label + ":",
```

with:

```go
			NewText: completionTextEditNewText(context, completion.Items[i]),
```

- [ ] **Step 7: Run the focused analyzer tests and confirm they pass**

Run:

```sh
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffset(Completes(FirstArrayItemKey|BlankArrayItemKey|ArrayItemKeyAfterBareDash)|DoesNotFallbackFromArrayItemToParentObject)|TestAnalyzerCompletionUsesArrayItemSchemaPath|TestAnalyzerCompletionAtOffsetIncludesNestedKeyEdit|TestAnalyzerCompletionAtOffsetDoesNotCompleteScalarValues|TestAnalyzerCompletionAtOffsetDoesNotCompleteBlockScalarValues|TestAnalyzerCompletionAtOffsetDoesNotCompleteAfterParentColon'
```

Expected: pass.

- [ ] **Step 8: Commit the analyzer array-item completion change**

Run:

```sh
git add internal/analyzer/completion.go internal/analyzer/analyzer_test.go
git commit -m "fix: complete first keys in yaml array items"
```

---

### Task 3: LSP Array-Item Text Edit Coverage

**Files:**
- Modify: `internal/lsp/server_test.go`

- [ ] **Step 1: Add a failing LSP text-edit test for `- f`**

Add this test near the existing completion text edit tests in `internal/lsp/server_test.go`:

```go
func TestCompletionTextEditCompletesFirstArrayItemKey(t *testing.T) {
	root := testRoot(t)
	uri := fileURI(filepath.Join(root, "api", "completion-array-item-edit.yaml"))
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - f"

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{
			"rootUri":      fileURI(root),
			"capabilities": zedCompletionCapabilities(),
		}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "functionRef")
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "functionRef:" {
		t.Fatalf("newText = %#v, want functionRef:", edit["newText"])
	}
	if item["insertTextMode"] != float64(1) {
		t.Fatalf("insertTextMode = %#v, want asIs", item["insertTextMode"])
	}
	rng := asMap(t, edit["range"])
	start := asMap(t, rng["start"])
	end := asMap(t, rng["end"])
	if start["line"] != float64(4) || start["character"] != float64(6) || end["line"] != float64(4) || end["character"] != float64(7) {
		t.Fatalf("textEdit range = %#v, want line 4 char 6..7", rng)
	}
}
```

- [ ] **Step 2: Run the focused LSP text-edit test**

Run:

```sh
go test ./internal/lsp -run TestCompletionTextEditCompletesFirstArrayItemKey
```

Expected: pass because Task 2 already added the analyzer text edit that the LSP adapter maps to protocol ranges.

- [ ] **Step 3: Run all focused completion tests**

Run:

```sh
go test ./internal/analyzer ./internal/lsp -run 'Completion'
```

Expected: pass.

- [ ] **Step 4: Commit the LSP array-item coverage**

Run:

```sh
git add internal/lsp/server_test.go
git commit -m "test: cover array item completion edits"
```

---

### Task 4: Verification, Manual Zed Validation, And Roadmap Closeout

**Files:**
- Modify: `PROJECT_ROADMAP.md`

- [ ] **Step 1: Run the full Go test suite**

Run:

```sh
go test ./...
```

Expected: pass.

- [ ] **Step 2: Build local CLIs for Zed validation**

Run:

```sh
./scripts/update-generated.sh
```

Expected: the command regenerates committed schema artifacts without diff, runs stale-generation and schema tests, runs the full Go test suite, and builds `dist/local/vibe-xpls`.

- [ ] **Step 3: Confirm Zed points at the local validation binary**

Check `~/.config/zed/settings.json` and make sure the configured binary path is:

```json
"/Users/tim.kersten/Code/gh/vibe-xpls/dist/local/vibe-xpls"
```

If it is not, update only the `lsp.crossplane-yaml.binary.path` value to that path and restart Zed.

- [ ] **Step 4: Manually validate Markdown completion docs in Zed**

Open:

```text
/Users/tim.kersten/Code/gh/vibe-xpls/internal/analyzer/testdata/workspaces/root/api/composition.yaml
```

Use this document state:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: root-composition
spec:
  
```

Trigger completion under `spec:`. Expected:

- `compositeTypeRef`, `mode`, `pipeline`, and `writeConnectionSecretsToNamespace` appear.
- The completion docs render `_Type: object_` and `_Required_` as Markdown formatting, not literal underscores.
- Completion rows still show property icons and no generic `detail`.

- [ ] **Step 5: Manually validate array-item first-key completion in Zed**

Use this document state:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: root-composition
spec:
  pipeline:
    - f
```

Trigger completion after `f` and accept `functionRef`. Expected result:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: root-composition
spec:
  pipeline:
    - functionRef:
```

Then continue with:

```yaml
        n
```

Trigger completion after `n`. Expected: `name` completes correctly under `functionRef`.

- [ ] **Step 6: Re-check v1/v2 release-specific completion sets**

Temporarily set the package marker to v1.20:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
spec:
  crossplane:
    version: ">=v1.20.0 <v2.0.0"
```

Expected under `Composition.spec`: v1-era fields such as `resources`, `patchSets`, and `publishConnectionDetailsWithStoreConfigRef` appear.

Then temporarily set the package marker to v2:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
spec:
  crossplane:
    version: ">=v2.0.0"
```

Expected under `Composition.spec`: v2 field set appears without v1 `resources`.

Restore `internal/analyzer/testdata/workspaces/root/crossplane.yaml` after this manual check:

```yaml
apiVersion: meta.pkg.crossplane.io/v1
kind: Configuration
metadata:
  name: root-package
```

- [ ] **Step 7: Update the roadmap after validation passes**

In `PROJECT_ROADMAP.md`, replace the Current bullet:

```md
- Tightening completion UX issues found through real editor use, including
  Markdown rendering for completion documentation, first-key completion in YAML
  array items, and parent-key documentation gaps for generated compatibility
  schemas.
```

with:

```md
- Tightening remaining completion UX issues found through real editor use,
  including parent-key documentation gaps for generated compatibility schemas.
```

If parent-key documentation gaps have also been resolved before this step, remove the completion UX bullet entirely.

- [ ] **Step 8: Commit roadmap closeout**

Run:

```sh
git add PROJECT_ROADMAP.md
git commit -m "docs: close completion ux cleanup"
```

- [ ] **Step 9: Final status check**

Run:

```sh
git status --short
```

Expected: no output, unless manual Zed validation intentionally left local-only settings outside the repository.
