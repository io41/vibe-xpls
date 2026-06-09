# Function Input Schema Dispatch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide key completions under Composition `spec.pipeline[].input` from the input object's known local schema.

**Architecture:** Keep the existing generated/workspace schema model unchanged. At completion time, detect when the cursor is inside a concrete pipeline step's `input` object, read that same input object's stable `apiVersion` and `kind`, resolve its schema from package-scoped workspace schemas, then run the existing field-completion logic against the input schema-relative parent path. Unknown, unstable, or schema-less input GVKs continue to produce no input child completions.

**Tech Stack:** Go, existing analyzer YAML occurrence model, existing `Schema`/`FieldDoc` model, existing LSP completion adapter, committed local tests.

---

## Scope

Implement only key completions for known input schemas under:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-go-templating
      input:
        apiVersion: gotemplating.fn.crossplane.io/v1beta1
        kind: GoTemplate
        spec:
          # complete fields from gotemplating.fn.crossplane.io/v1beta1 GoTemplate here
```

The first implementation uses the selected pipeline step by path, for example `spec.pipeline[1].input`, so different pipeline steps can dispatch to different input GVKs. It does not use `functionRef.name` to fetch or infer schemas. It only uses schemas already known to the analyzer through workspace CRDs/XRDs.

Completing at the root of a known input object is in scope: when `apiVersion`
and `kind` are already present, completing with parent path
`spec.pipeline[0].input` may offer root fields from the input schema such as
`spec`. The implementation remains gated by the outer Composition schema
resolution path, so a disabled or corrupt built-in bundle still suppresses the
outer Composition completion before input dispatch runs.

Non-goals:

- No registry, package image, cluster, or network discovery.
- No value completions.
- No diagnostics for mismatched function/input pairs.
- No hover dispatch under input fields in this slice.
- No support for `ops.crossplane.io` operation pipelines in this slice.

## Files

- Modify: `internal/analyzer/yaml.go`
  - Add a small occurrence lookup helper for exact path values in a specific YAML document.
- Modify: `internal/analyzer/parse_test.go`
  - Cover exact occurrence lookup for multiple pipeline input objects.
- Modify: `internal/analyzer/completion.go`
  - Detect Composition function input contexts and dispatch to the input schema.
- Modify: `internal/analyzer/analyzer_test.go`
  - Add analyzer tests for local input schema completion, per-step dispatch, and unknown input schemas.
- Modify: `internal/lsp/server_test.go`
  - Add one wire-level regression test that input-schema completions still serialize as property completion items with Markdown docs.
- Modify: `PROJECT_ROADMAP.md`
  - Move function input schema dispatch from `Next` to `Done` after implementation and verification.

---

### Task 1: Exact YAML Value Lookup For Concrete Input Paths

**Files:**
- Modify: `internal/analyzer/yaml.go`
- Modify: `internal/analyzer/parse_test.go`

- [ ] **Step 1: Add a failing parser test for concrete input path values**

Add this test near `TestSequencePathTraversal` in `internal/analyzer/parse_test.go`:

```go
func TestValueForDocumentPathUsesConcretePipelineInputPath(t *testing.T) {
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-a
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: InputA
    - functionRef:
        name: function-b
      input:
        apiVersion: fn.example.org/v1beta1
        kind: InputB
`

	doc := ParseYAMLDocument(text)

	got, ok := doc.ValueForDocumentPath(0, "spec.pipeline[0].input.apiVersion")
	if !ok || got != "fn.example.org/v1alpha1" {
		t.Fatalf("step 0 input apiVersion = %q ok=%v, want fn.example.org/v1alpha1", got, ok)
	}
	got, ok = doc.ValueForDocumentPath(0, "spec.pipeline[0].input.kind")
	if !ok || got != "InputA" {
		t.Fatalf("step 0 input kind = %q ok=%v, want InputA", got, ok)
	}
	got, ok = doc.ValueForDocumentPath(0, "spec.pipeline[1].input.apiVersion")
	if !ok || got != "fn.example.org/v1beta1" {
		t.Fatalf("step 1 input apiVersion = %q ok=%v, want fn.example.org/v1beta1", got, ok)
	}
	got, ok = doc.ValueForDocumentPath(0, "spec.pipeline[1].input.kind")
	if !ok || got != "InputB" {
		t.Fatalf("step 1 input kind = %q ok=%v, want InputB", got, ok)
	}
}
```

- [ ] **Step 2: Run the parser test and confirm it fails**

Run:

```sh
go test ./internal/analyzer -run TestValueForDocumentPathUsesConcretePipelineInputPath -count=1
```

Expected: fail because `ValueForDocumentPath` does not exist.

- [ ] **Step 3: Add the exact value lookup helper**

Add this method in `internal/analyzer/yaml.go` after `RootValueForOccurrence`:

```go
func (d YAMLDocument) ValueForDocumentPath(documentIndex int, path string) (string, bool) {
	var best PathOccurrence
	bestOK := false
	for _, candidate := range d.occurrences {
		if candidate.DocumentIndex != documentIndex || candidate.Path != path {
			continue
		}
		if !bestOK || candidate.PathSpan.Start > best.PathSpan.Start {
			best = candidate
			bestOK = true
		}
	}
	if !bestOK || !best.Stable || !best.ValueOK {
		return "", false
	}
	return best.Value, true
}
```

- [ ] **Step 4: Verify the parser helper**

Run:

```sh
go test ./internal/analyzer -run TestValueForDocumentPathUsesConcretePipelineInputPath -count=1
```

Expected: pass.

---

### Task 2: Analyzer Dispatch For Composition Pipeline Input Completion

**Files:**
- Modify: `internal/analyzer/completion.go`
- Modify: `internal/analyzer/analyzer_test.go`

- [ ] **Step 1: Add workspace function input schema test fixtures**

Add these helper functions near the existing workspace CRD helpers in `internal/analyzer/analyzer_test.go`:

```go
func workspaceFunctionInputCRD(group, version, kind, fieldName, description string) string {
	return `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: test.` + group + `
spec:
  group: ` + group + `
  names:
    kind: ` + kind + `
    plural: tests
  scope: Namespaced
  versions:
    - name: ` + version + `
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            apiVersion:
              type: string
            kind:
              type: string
            spec:
              type: object
              properties:
                ` + fieldName + `:
                  type: string
                  description: ` + description + `
`
}

func writeFunctionInputCompletionPackage(t *testing.T) (string, *Analyzer) {
	t.Helper()
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	analyzerWriteFile(t, filepath.Join(root, "api", "function-input-crd.yaml"), workspaceFunctionInputCRD("fn.example.org", "v1alpha1", "TemplateInput", "inline", "Inline template source."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	return root, a
}
```

- [ ] **Step 2: Add a failing analyzer test for known input schema completion**

Add this test in `internal/analyzer/analyzer_test.go` near the existing pipeline completion tests:

```go
func TestAnalyzerCompletionAtOffsetDispatchesFunctionInputSchema(t *testing.T) {
	root, a := writeFunctionInputCompletionPackage(t)
	uri := "file://" + filepath.Join(root, "api", "composition-function-input.yaml")
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-go-templating
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: TemplateInput
        spec:
          i`
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "inline")
	if !ok {
		t.Fatalf("completion missing inline input field: %#v", completion.Items)
	}
	if !strings.Contains(item.Documentation, "Inline template source.") {
		t.Fatalf("inline documentation = %q, want input schema docs", item.Documentation)
	}
	if item.TextEdit == nil || item.TextEdit.NewText != "          inline:" {
		t.Fatalf("inline text edit = %#v, want indented inline key", item.TextEdit)
	}
}
```

- [ ] **Step 3: Add a failing analyzer test for per-step dispatch**

Add this test in `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionAtOffsetUsesCurrentPipelineInputGVK(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	analyzerWriteFile(t, filepath.Join(root, "api", "input-a-crd.yaml"), workspaceFunctionInputCRD("fn.example.org", "v1alpha1", "InputA", "alphaField", "Alpha input field."))
	analyzerWriteFile(t, filepath.Join(root, "api", "input-b-crd.yaml"), workspaceFunctionInputCRD("fn.example.org", "v1alpha1", "InputB", "betaField", "Beta input field."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition-two-inputs.yaml")
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-a
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: InputA
        spec:
          alphaField: one
    - functionRef:
        name: function-b
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: InputB
        spec:
          b`
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "betaField") {
		t.Fatalf("completion missing betaField for second input: %#v", completion.Items)
	}
	if containsCompletion(completion.Items, "alphaField") {
		t.Fatalf("completion leaked first input field into second input: %#v", completion.Items)
	}
}
```

- [ ] **Step 4: Add a no-schema regression test**

Add this test in `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionAtOffsetSkipsUnknownFunctionInputSchema(t *testing.T) {
	root, a := writeFunctionInputCompletionPackage(t)
	uri := "file://" + filepath.Join(root, "api", "composition-unknown-function-input.yaml")
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-go-templating
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: MissingInput
        spec:
          i`
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if containsCompletion(completion.Items, "inline") {
		t.Fatalf("unknown input schema completion = %#v, want no TemplateInput fields", completion.Items)
	}
}
```

- [ ] **Step 5: Run the new analyzer tests and confirm they fail**

Run:

```sh
go test ./internal/analyzer -run 'TestAnalyzerCompletionAtOffsetDispatchesFunctionInputSchema|TestAnalyzerCompletionAtOffsetUsesCurrentPipelineInputGVK|TestAnalyzerCompletionAtOffsetSkipsUnknownFunctionInputSchema' -count=1
```

Expected: the first two tests fail because completion still uses the Composition schema under `spec.pipeline[].input`; the unknown-schema test should pass or return no items.

- [ ] **Step 6: Add function input path helpers**

`internal/analyzer/completion.go` already imports `fmt`, `regexp`, `sort`,
and `strings`; no new imports are needed for this task.

Add this package-level regexp near `schemaArrayIndexPattern`:

```go
var compositionFunctionInputPathPattern = regexp.MustCompile(`^spec\.pipeline\[(\d+)\]\.input(?:\.(.*))?$`)
```

Add these helpers near `schemaPathFromParsedPath`:

```go
type functionInputCompletionTarget struct {
	fields         []FieldDoc
	inputChildPath string
}

func compositionFunctionInputPath(path string) (inputPath, inputChildPath string, ok bool) {
	matches := compositionFunctionInputPathPattern.FindStringSubmatch(path)
	if matches == nil {
		return "", "", false
	}
	inputPath = "spec.pipeline[" + matches[1] + "].input"
	return inputPath, matches[2], true
}

func (a *Analyzer) functionInputCompletionTarget(uri string, parsed YAMLDocument, context completionContext) (functionInputCompletionTarget, bool) {
	inputPath, inputChildPath, ok := compositionFunctionInputPath(context.parentPath)
	if !ok {
		inputPath, inputChildPath, ok = compositionFunctionInputPath(context.schemaParentPath)
	}
	if !ok {
		return functionInputCompletionTarget{}, false
	}
	apiVersion, apiOK := parsed.ValueForDocumentPath(context.rootOccurrence.DocumentIndex, inputPath+".apiVersion")
	kind, kindOK := parsed.ValueForDocumentPath(context.rootOccurrence.DocumentIndex, inputPath+".kind")
	if !apiOK || !kindOK || apiVersion == "" || kind == "" {
		return functionInputCompletionTarget{}, false
	}
	fields, ok := a.fieldsForInputGVK(uri, SourceGVK{APIVersion: apiVersion, Kind: kind})
	if !ok {
		return functionInputCompletionTarget{}, false
	}
	return functionInputCompletionTarget{fields: fields, inputChildPath: inputChildPath}, true
}

func (a *Analyzer) fieldsForInputGVK(uri string, gvk SourceGVK) ([]FieldDoc, bool) {
	if a.schemas.HasWorkspaceSchema(gvk) {
		fields := a.schemas.Fields(gvk.APIVersion, gvk.Kind)
		return fields, len(fields) != 0
	}
	if schema, ok := a.workspaceSchemaForURI(uri, gvk); ok {
		fields := fieldsFromSchema(schema)
		return fields, len(fields) != 0
	}
	return nil, false
}

func completionFromFunctionInputFields(context completionContext, fields []FieldDoc, inputChildPath string) Completion {
	if completionContextIsScalarDescendant(context) {
		return Completion{}
	}
	// Input dispatch does not use parent fallback; the edit belongs at the
	// cursor's current indentation.
	completion := filterCompletion(completionFromFields(fields, inputChildPath), context.prefix)
	for i := range completion.Items {
		newText := context.indent + completion.Items[i].Label + ":"
		if context.useNewTextPrefix {
			newText = context.newTextPrefix + completion.Items[i].Label + ":"
		}
		completion.Items[i].TextEdit = &CompletionTextEdit{
			Replace: context.replace,
			NewText: newText,
		}
	}
	return completion
}
```

- [ ] **Step 7: Wire the dispatch into `CompletionAtOffset`**

In `internal/analyzer/completion.go`, find this block in `CompletionAtOffset`:

```go
	if completionContextIsScalarDescendant(context) {
		return Completion{}
	}
	completion := Completion{}
```

Replace it with:

```go
	if apiVersion == "apiextensions.crossplane.io/v1" && kind == "Composition" {
		if target, ok := a.functionInputCompletionTarget(uri, parsed, context); ok {
			return completionFromFunctionInputFields(context, target.fields, target.inputChildPath)
		}
	}
	if completionContextIsScalarDescendant(context) {
		return Completion{}
	}
	completion := Completion{}
```

- [ ] **Step 8: Run analyzer tests**

Run:

```sh
go test ./internal/analyzer -run 'TestValueForDocumentPathUsesConcretePipelineInputPath|TestAnalyzerCompletionAtOffsetDispatchesFunctionInputSchema|TestAnalyzerCompletionAtOffsetUsesCurrentPipelineInputGVK|TestAnalyzerCompletionAtOffsetSkipsUnknownFunctionInputSchema' -count=1
```

Expected: pass.

---

### Task 3: Direct Analyzer API Coverage

**Files:**
- Modify: `internal/analyzer/completion.go`
- Modify: `internal/analyzer/analyzer_test.go`

- [ ] **Step 1: Add a failing direct `Completion` API test**

Add this test in `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionDispatchesFunctionInputSchemaForParentPath(t *testing.T) {
	root, a := writeFunctionInputCompletionPackage(t)
	uri := "file://" + filepath.Join(root, "api", "composition-function-input-parent.yaml")
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-go-templating
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: TemplateInput
        spec:
`
	a.OpenDocument(uri, text)
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !parsed.IsStablePath("spec.pipeline[0].input.spec") {
		t.Fatalf("test setup: input spec path not stable")
	}

	completion := a.Completion(uri, "spec.pipeline[0].input.spec")
	item, ok := completionItemByLabel(completion.Items, "inline")
	if !ok {
		t.Fatalf("completion missing inline input field: %#v", completion.Items)
	}
	if !strings.Contains(item.Documentation, "Inline template source.") {
		t.Fatalf("inline documentation = %q, want input schema docs", item.Documentation)
	}
}
```

- [ ] **Step 2: Run the direct API test and confirm it fails**

Run:

```sh
go test ./internal/analyzer -run TestAnalyzerCompletionDispatchesFunctionInputSchemaForParentPath -count=1
```

Expected: fail because `Completion(uri, parentPath)` does not dispatch input schemas.

- [ ] **Step 3: Add direct parent-path dispatch**

In `internal/analyzer/completion.go`, add this helper:

```go
func (a *Analyzer) functionInputCompletionForParentPath(uri string, parsed YAMLDocument, parentPath string) (Completion, bool) {
	inputPath, inputChildPath, ok := compositionFunctionInputPath(parentPath)
	if !ok {
		return Completion{}, false
	}
	var occurrence PathOccurrence
	occurrenceOK := false
	for _, candidate := range parsed.occurrences {
		if candidate.Path == inputPath && candidate.Stable {
			occurrence = candidate
			occurrenceOK = true
			break
		}
	}
	if !occurrenceOK {
		return Completion{}, false
	}
	apiVersion, apiOK := parsed.ValueForDocumentPath(occurrence.DocumentIndex, inputPath+".apiVersion")
	kind, kindOK := parsed.ValueForDocumentPath(occurrence.DocumentIndex, inputPath+".kind")
	if !apiOK || !kindOK || apiVersion == "" || kind == "" {
		return Completion{}, false
	}
	fields, ok := a.fieldsForInputGVK(uri, SourceGVK{APIVersion: apiVersion, Kind: kind})
	if !ok {
		return Completion{}, false
	}
	return completionFromFields(fields, inputChildPath), true
}
```

Then update `Completion(uri, parentPath)` after `rootContextForCompletionParent`
succeeds and before resolving the root Composition schema. Keep the dispatch
limited to the outer Composition resource:

```go
	if root.apiVersion == "apiextensions.crossplane.io/v1" && root.kind == "Composition" {
		if completion, ok := a.functionInputCompletionForParentPath(uri, parsed, parentPath); ok {
			return completion
		}
	}
```

Do not use this broader, ungated form:

```go
	if completion, ok := a.functionInputCompletionForParentPath(uri, parsed, parentPath); ok {
		return completion
	}
```

- [ ] **Step 4: Run direct and offset analyzer tests**

Run:

```sh
go test ./internal/analyzer -run 'TestAnalyzerCompletionDispatchesFunctionInputSchemaForParentPath|TestAnalyzerCompletionAtOffsetDispatchesFunctionInputSchema|TestAnalyzerCompletionAtOffsetUsesCurrentPipelineInputGVK' -count=1
```

Expected: pass.

---

### Task 4: LSP Wire Regression

**Files:**
- Modify: `internal/lsp/server_test.go`

- [ ] **Step 1: Add a failing LSP test for function input completions**

Add this test near the existing completion tests in `internal/lsp/server_test.go`:

```go
func TestCompletionItemsDispatchFunctionInputSchema(t *testing.T) {
	root := t.TempDir()
	writeLSPTestFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	writeLSPTestFile(t, filepath.Join(root, "api", "function-input-crd.yaml"), lspFunctionInputCRD())
	uri := fileURI(filepath.Join(root, "api", "composition-function-input.yaml"))
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
    - functionRef:
        name: function-go-templating
      input:
        apiVersion: fn.example.org/v1alpha1
        kind: TemplateInput
        spec:
          i`

	messages := runServerFrames(t,
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(root), "capabilities": zedCompletionCapabilities()}),
		notificationFrame(t, "textDocument/didOpen", map[string]any{
			"textDocument": map[string]any{"uri": uri, "text": text},
		}),
		requestFrame(t, 2, "textDocument/completion", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     positionAtOffset(t, text, len(text), source.EncodingUTF16),
		}),
	)

	completion := resultMap(t, responseForID(t, messages, 2))
	item := completionItemByLabelForTest(t, asSlice(t, completion["items"]), "inline")
	if item["kind"] != float64(10) {
		t.Fatalf("kind = %#v, want property", item["kind"])
	}
	documentation := asMap(t, item["documentation"])
	if documentation["kind"] != "markdown" || documentation["value"] != "Inline template source.\n\n_Type: string_" {
		t.Fatalf("documentation = %#v, want markdown input schema docs", item["documentation"])
	}
	edit := asMap(t, item["textEdit"])
	if edit["newText"] != "          inline:" {
		t.Fatalf("newText = %#v, want indented input field", edit["newText"])
	}
}
```

Add this helper near `lspProviderCRD`:

```go
func lspFunctionInputCRD() string {
	return `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: fn.example.org
  names:
    kind: TemplateInput
    plural: templateinputs
  scope: Namespaced
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            apiVersion:
              type: string
            kind:
              type: string
            spec:
              type: object
              properties:
                inline:
                  type: string
                  description: Inline template source.
`
}
```

- [ ] **Step 2: Run the LSP test and confirm it fails before dispatch, then passes after Task 2**

Run:

```sh
go test ./internal/lsp -run TestCompletionItemsDispatchFunctionInputSchema -count=1
```

Expected after Task 2: pass.

---

### Task 5: Full Verification And Roadmap Closeout

**Files:**
- Modify: `PROJECT_ROADMAP.md`

- [ ] **Step 1: Run focused regression tests**

Run:

```sh
go test ./internal/analyzer -run 'TestValueForDocumentPathUsesConcretePipelineInputPath|TestAnalyzerCompletionAtOffsetDispatchesFunctionInputSchema|TestAnalyzerCompletionAtOffsetUsesCurrentPipelineInputGVK|TestAnalyzerCompletionAtOffsetSkipsUnknownFunctionInputSchema|TestAnalyzerCompletionDispatchesFunctionInputSchemaForParentPath' -count=1
go test ./internal/lsp -run TestCompletionItemsDispatchFunctionInputSchema -count=1
```

Expected: both commands pass.

- [ ] **Step 2: Run full Go tests**

Run:

```sh
go test ./...
```

Expected: pass.

- [ ] **Step 3: Update the roadmap**

In `PROJECT_ROADMAP.md`, move this `Next` item:

```md
1. Function input schema dispatch.
   Use the selected pipeline function and known input GVK to provide completions
   under `spec.pipeline[].input` when the input object's schema is known.
```

to `Done` with this wording:

```md
- Function input schema dispatch for Composition pipeline inputs. When a
  `spec.pipeline[].input` object has a stable `apiVersion` and `kind` whose
  schema is known locally, completions under that input use the input object's
  schema.
```

Renumber the remaining `Next` list to:

```md
1. Relationship-aware completions.
   Use the local package, XRD, Composition, function, and provider graph to
   suggest safe relationships such as composition type refs and package
   dependency references.

2. Safe value completions.
   Add value completions from schema enums, defaults, and in-workspace facts
   without inventing values or querying remote systems.

3. Developer/debug schema insight command.
   Add a command that explains bundle health, selected release, active package
   root, schema provenance, and completion suppression reasons for a file.
```

- [ ] **Step 4: Final status check**

Run:

```sh
git status --short
```

Expected: only intended implementation files are modified, plus any unrelated
pre-existing fixture edits that were already present before implementation.
Leave unrelated pre-existing fixture edits unstaged and do not revert them
unless the user explicitly asks.

- [ ] **Step 5: Commit**

Run:

```sh
git add internal/analyzer/yaml.go internal/analyzer/parse_test.go internal/analyzer/completion.go internal/analyzer/analyzer_test.go internal/lsp/server_test.go PROJECT_ROADMAP.md
git commit -m "feat: dispatch function input schema completions"
```
