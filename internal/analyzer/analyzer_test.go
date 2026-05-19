package analyzer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestAnalyzerDiagnosticsHoverAndCompletion(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	a.OpenDocument(uri, text)

	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
	hover, ok := a.Hover(uri, "spec.compositeTypeRef.kind")
	if !ok || !strings.Contains(hover.Markdown, "Composite kind") {
		t.Fatalf("hover = %#v ok=%v", hover, ok)
	}
	completion := a.Completion(uri, "spec.compositeTypeRef")
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("completion missing kind: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionUsesSchemaParentThatDoesNotExistYet(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition-in-progress.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.compositeTypeRef")
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("completion missing kind for absent schema parent: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionPathBasedItemsHaveNilTextEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "path-completion-label-only.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.compositeTypeRef")
	item, ok := completionItemByLabel(completion.Items, "kind")
	if !ok {
		t.Fatalf("completion missing kind: %#v", completion.Items)
	}
	if item.TextEdit != nil {
		t.Fatalf("path-based completion text edit = %#v, want nil", item.TextEdit)
	}
}

func TestAnalyzerCompletionUsesSameRootContextAcrossDocuments(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "multi-composition-in-progress.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n---\napiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.compositeTypeRef")
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("completion missing kind for shared multi-doc root context: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetUsesMappingKeyContext(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-context.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    "
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("blank child-key completion missing kind: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetFiltersPartialMappingKey(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-prefix.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    k"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "kind") {
		t.Fatalf("partial child-key completion missing kind: %#v", completion.Items)
	}
	if containsCompletion(completion.Items, "apiVersion") {
		t.Fatalf("partial child-key completion was not prefix-filtered: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetSuppressesTextEditBeforeExistingColon(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-existing-colon.yaml")

	tests := []struct {
		name   string
		text   string
		needle string
	}{
		{
			name:   "cursor before colon",
			text:   "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    k: CompositeBucket\n",
			needle: "k:",
		},
		{
			name:   "cursor before key tail and colon",
			text:   "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    ki: CompositeBucket\n",
			needle: "ki:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a.ChangeDocument(uri, tt.text)
			cursor := strings.Index(tt.text, tt.needle)
			if cursor < 0 {
				t.Fatalf("test setup: %q not found", tt.needle)
			}
			cursor += len("k")
			if completion := a.CompletionAtOffset(uri, cursor); len(completion.Items) != 0 {
				t.Fatalf("completion before existing colon = %#v, want none", completion.Items)
			}
		})
	}
}

func TestAnalyzerCompletionAtOffsetIncludesRootKeyEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-root-edit.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: root-composition\ns"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "spec")
	if !ok {
		t.Fatalf("completion missing spec: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("spec completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "spec:" {
		t.Fatalf("new text = %q, want spec:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "\n") + 1, End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetCorrectsIndentedRootKeyEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "completion-package-root-edit.yaml")
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: root-package\n  s"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "spec")
	if !ok {
		t.Fatalf("completion missing spec: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("spec completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "spec:" {
		t.Fatalf("new text = %q, want spec:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "\n") + 1, End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetIncludesNestedKeyEdit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-nested-edit.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    k"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "kind")
	if !ok {
		t.Fatalf("completion missing kind: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("kind completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "    kind:" {
		t.Fatalf("new text = %q, want indented kind key", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "\n") + 1, End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

func TestAnalyzerCompletionAtOffsetDoesNotCompleteScalarValues(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-value.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	a.OpenDocument(uri, text)

	valueOffset := strings.Index(text, "CompositeBucket") + len("CompositeBucket")
	if completion := a.CompletionAtOffset(uri, valueOffset); len(completion.Items) != 0 {
		t.Fatalf("scalar value completion = %#v, want none", completion.Items)
	}
	apiVersionOffset := strings.Index(text, "crossplane.io") + len("crossplane")
	if completion := a.CompletionAtOffset(uri, apiVersionOffset); len(completion.Items) != 0 {
		t.Fatalf("apiVersion value completion = %#v, want none", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetDoesNotCompleteBlockScalarValues(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-block-scalar.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: |\n    "
	a.OpenDocument(uri, text)

	if completion := a.CompletionAtOffset(uri, len(text)); len(completion.Items) != 0 {
		t.Fatalf("block scalar completion = %#v, want none", completion.Items)
	}

	a.ChangeDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: |\n")
	if completion := a.CompletionAtOffset(uri, len("apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: |\n")); len(completion.Items) != 0 {
		t.Fatalf("blank block scalar completion = %#v, want none", completion.Items)
	}

	text = "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: |\n    ---\n    "
	a.ChangeDocument(uri, text)
	if completion := a.CompletionAtOffset(uri, len(text)); len(completion.Items) != 0 {
		t.Fatalf("block scalar separator text completion = %#v, want none", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetDoesNotCrossDocumentBeforeRootContext(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-new-document.yaml")
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\n---\n"
	a.OpenDocument(uri, text)

	if completion := a.CompletionAtOffset(uri, len(text)); len(completion.Items) != 0 {
		t.Fatalf("new document without root context completion = %#v, want none", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetDoesNotCompleteAfterParentColon(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "completion-parent-colon.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:"
	a.OpenDocument(uri, text)

	if completion := a.CompletionAtOffset(uri, len(text)); len(completion.Items) != 0 {
		t.Fatalf("parent-colon completion = %#v, want none", completion.Items)
	}
}

func TestAnalyzerUnknownProviderDoesNotInventFields(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "bucket.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.forProvider")
	if len(completion.Items) != 0 {
		t.Fatalf("unknown provider schema should not invent completions: %#v", completion.Items)
	}
}

func TestAnalyzerPathOnlyHoverIsAmbiguousAcrossRootContexts(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "multi-doc.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: composition-demo\n---\napiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: configuration-demo\n"
	a.OpenDocument(uri, text)

	if hover, ok := a.Hover(uri, "metadata.name"); ok {
		t.Fatalf("path-only hover should be ambiguous, got %#v", hover)
	}
	if completion := a.Completion(uri, "metadata"); len(completion.Items) != 0 {
		t.Fatalf("path-only completion should be ambiguous, got %#v", completion.Items)
	}

	compositionOffset := strings.Index(text, "composition-demo")
	if compositionOffset < 0 {
		t.Fatal("test setup: composition name not found")
	}
	hover, ok := a.HoverAtOffset(uri, compositionOffset)
	if !ok || !strings.Contains(hover.Markdown, "Composition") {
		t.Fatalf("composition hover = %#v ok=%v, want Composition-specific hover", hover, ok)
	}
	configurationOffset := strings.Index(text, "configuration-demo")
	if configurationOffset < 0 {
		t.Fatal("test setup: configuration name not found")
	}
	hover, ok = a.HoverAtOffset(uri, configurationOffset)
	if !ok || !strings.Contains(hover.Markdown, "Configuration") {
		t.Fatalf("configuration hover = %#v ok=%v, want Configuration-specific hover", hover, ok)
	}
}

func TestAnalyzerPathOnlyRootContextRejectsTemplateDerivedDuplicateRoot(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "duplicate-root-template.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\napiVersion: {{ .APIVersion }}\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	a.OpenDocument(uri, text)

	if hover, ok := a.Hover(uri, "spec.compositeTypeRef.kind"); ok {
		t.Fatalf("path-only hover should reject unstable duplicate root context, got %#v", hover)
	}
	if completion := a.Completion(uri, "spec.compositeTypeRef"); len(completion.Items) != 0 {
		t.Fatalf("path-only completion should reject unstable duplicate root context, got %#v", completion.Items)
	}
}

func TestNoRootActivationTogglesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	malformed := "spec: [unterminated\n"
	a.OpenDocument(uri, "apiVersion: v1\nkind: ConfigMap\n"+malformed)
	if got := len(a.Diagnostics(uri)); got != 0 {
		t.Fatalf("ordinary no-root yaml should stay quiet, got %d diagnostics", got)
	}
	a.ChangeDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"+malformed)
	if got := len(a.Diagnostics(uri)); got == 0 {
		t.Fatal("Crossplane no-root document should activate diagnostics")
	}
	a.ChangeDocument(uri, "apiVersion: v1\nkind: ConfigMap\n"+malformed)
	if got := len(a.Diagnostics(uri)); got != 0 {
		t.Fatalf("removing activation should clear diagnostics, got %d", got)
	}
}

func TestNoRootOrdinaryCustomResourceDefinitionStaysQuiet(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("ordinary no-root CRD yaml should stay quiet, got %#v", diagnostics)
	}
}

func TestNoRootCompositionKindWithoutShapeStaysQuiet(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("ordinary no-root Composition-shaped name without shape should stay quiet, got %#v", diagnostics)
	}
}

func TestNoRootCompositionKindWithShapeActivatesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) == 0 {
		t.Fatal("Composition kind with stable Composition shape should activate diagnostics")
	}
}

func TestNoRootCompositionShapeLineDiagnosticActivatesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n  compositeTypeRef: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) == 0 {
		t.Fatal("Composition kind with malformed shape line should activate diagnostics")
	}
}

func TestNoRootBlockScalarShapeTextDoesNotActivateDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\ndata: |\n  spec:\n    compositeTypeRef: not real YAML shape\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("block scalar shape text should not activate no-root diagnostics, got %#v", diagnostics)
	}
}

func TestNoRootSequenceBlockScalarShapeTextDoesNotActivateDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n- |\n  compositeTypeRef: not real YAML shape\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("sequence block scalar shape text should not activate no-root diagnostics, got %#v", diagnostics)
	}
}

func TestNoRootSequenceMappingShapeDoesNotActivateDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n- name: item\n  compositeTypeRef:\n    kind: CompositeBucket\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("sequence mapping shape should not activate no-root diagnostics, got %#v", diagnostics)
	}
}

func TestNoRootDashOnlySequenceShapeDoesNotActivateDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n-\n  compositeTypeRef:\n    kind: CompositeBucket\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("dash-only sequence shape should not activate no-root diagnostics, got %#v", diagnostics)
	}
}

func TestNoRootDocumentSeparatorCommentResetsBoundedSniffState(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\n--- # second document\nspec:\n  compositeTypeRef:\nbroken: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("document separator with comment should prevent cross-document kind/shape activation, got %#v", diagnostics)
	}
}

func TestNoRootXRDShapeLineDiagnosticActivatesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: CompositeResourceDefinition\nspec:\n  group: [unterminated\n"
	a.OpenDocument(uri, text)

	if diagnostics := a.Diagnostics(uri); len(diagnostics) == 0 {
		t.Fatal("XRD kind with malformed shape line should activate diagnostics")
	}
}

func TestHugeDocumentDowngradesAnalysis(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "large.yaml")
	a.OpenDocument(uri, strings.Repeat("a", 32))
	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "size limit") {
		t.Fatalf("expected size limit diagnostic, got %#v", diagnostics)
	}
}

func TestHugeNoRootCompositionKindWithoutShapeStaysQuiet(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	a.OpenDocument(uri, "apiVersion: example.io/v1\nkind: Composition\n"+strings.Repeat("a", 64))

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("oversized no-root Composition kind without shape should stay quiet, got %#v", diagnostics)
	}
}

func TestHugeNoRootCompositionKindWithShapeReportsSizeLimit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	text := "apiVersion: example.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n" + strings.Repeat("a", 64)
	a.OpenDocument(uri, text)

	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "size limit") {
		t.Fatalf("expected size limit diagnostic for oversized Composition kind with shape, got %#v", diagnostics)
	}
}

func TestHugeNoRootCrossplaneRootSignalReportsSizeLimit(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	a.OpenDocument(uri, "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"+strings.Repeat("a", 64))

	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "size limit") {
		t.Fatalf("expected size limit diagnostic for oversized active no-root doc, got %#v", diagnostics)
	}
}

func TestHugeNoRootOrdinaryDocumentStaysQuiet(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "plain.yaml")
	a.OpenDocument(uri, "apiVersion: v1\nkind: ConfigMap\n"+strings.Repeat("a", 32))

	if diagnostics := a.Diagnostics(uri); len(diagnostics) != 0 {
		t.Fatalf("ordinary oversized no-root yaml should stay quiet, got %#v", diagnostics)
	}
}

func TestAnalyzerDiagnosticsRespectMaxDiagnosticsPerDoc(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDiagnosticsPerDoc: 1}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "capped-diagnostics.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nbad: @value\nmetadata:\n  name: {{ .Name\n"
	if got := len(ParseYAMLDocument(text).Diagnostics); got < 2 {
		t.Fatalf("test setup expected at least 2 diagnostics before cap, got %d", got)
	}
	a.OpenDocument(uri, text)

	diagnostics := a.Diagnostics(uri)
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly 1 due to cap", diagnostics)
	}
}

func TestAnalyzerLimitsDefaultFieldByField(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: Limits{MaxDocumentBytes: 16}})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	defaults := DefaultLimits()
	if a.limits.MaxDocumentBytes != 16 {
		t.Fatalf("MaxDocumentBytes = %d, want caller override", a.limits.MaxDocumentBytes)
	}
	if a.limits.MaxDiagnosticsPerDoc != defaults.MaxDiagnosticsPerDoc {
		t.Fatalf("MaxDiagnosticsPerDoc = %d, want %d", a.limits.MaxDiagnosticsPerDoc, defaults.MaxDiagnosticsPerDoc)
	}
	if a.limits.MaxYAMLFiles != defaults.MaxYAMLFiles {
		t.Fatalf("MaxYAMLFiles = %d, want %d", a.limits.MaxYAMLFiles, defaults.MaxYAMLFiles)
	}
	if a.limits.MaxYAMLBytes != defaults.MaxYAMLBytes {
		t.Fatalf("MaxYAMLBytes = %d, want %d", a.limits.MaxYAMLBytes, defaults.MaxYAMLBytes)
	}
	if a.limits.DocumentSoftDeadline != defaults.DocumentSoftDeadline {
		t.Fatalf("DocumentSoftDeadline = %s, want %s", a.limits.DocumentSoftDeadline, defaults.DocumentSoftDeadline)
	}
}

func TestNoRootCrossplaneFilenameActivatesDiagnostics(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "no-root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "example.crossplane.yaml")
	a.OpenDocument(uri, "apiVersion: v1\nkind: ConfigMap\nspec: [unterminated\n")
	if got := len(a.Diagnostics(uri)); got == 0 {
		t.Fatal("Crossplane-classified filename should activate diagnostics")
	}
}

func containsCompletion(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func completionItemByLabel(items []CompletionItem, label string) (CompletionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return CompletionItem{}, false
}
