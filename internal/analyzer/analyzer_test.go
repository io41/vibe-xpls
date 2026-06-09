package analyzer

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

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

func TestAnalyzerHoverIgnoresCommentsAndCoversCompositeRefValue(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  # This Composition implements the XServiceBusNamespace composite behind IstaServiceBusNamespace.
  name: xservice
spec:
  # IstaServiceBusNamespace claims create XServiceBusNamespace composites. XTopic references these
  # XServiceBusNamespace objects through spec.serviceBusNamespaceRef.
  compositeTypeRef:
    apiVersion: asb.platform.ista.com/v1alpha1
    kind: XServiceBusNamespace
`
	a.OpenDocument(uri, text)

	for _, needle := range []string{
		"This Composition implements",
		"IstaServiceBusNamespace claims",
		"XServiceBusNamespace objects",
	} {
		offset := strings.Index(text, needle)
		if offset < 0 {
			t.Fatalf("test setup: %q not found", needle)
		}
		if hover, ok := a.HoverAtOffset(uri, offset); ok {
			t.Fatalf("comment hover at %q = %#v, want none", needle, hover)
		}
	}

	value := "XServiceBusNamespace"
	start := strings.LastIndex(text, value)
	if start < 0 {
		t.Fatalf("test setup: %q value not found", value)
	}
	for i := 0; i <= len(value); i++ {
		hover, ok := a.HoverAtOffset(uri, start+i)
		if !ok || !strings.Contains(hover.Markdown, "Composite kind") {
			t.Fatalf("hover at value offset %d = %#v ok=%v", i, hover, ok)
		}
	}

	rootKind := "Composition"
	rootKindStart := strings.Index(text, "kind: "+rootKind)
	if rootKindStart < 0 {
		t.Fatalf("test setup: root kind value not found")
	}
	rootKindStart += len("kind: ")
	for i := 0; i <= len(rootKind); i++ {
		hover, ok := a.HoverAtOffset(uri, rootKindStart+i)
		if !ok || !strings.Contains(hover.Markdown, "Resource kind, normally Composition.") {
			t.Fatalf("root kind hover at value offset %d = %#v ok=%v", i, hover, ok)
		}
	}
}

func TestAnalyzerStartupServesCompositeTypeRefAPIVersionDocs(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    apiVersion: example.org/v1\n"
	a.OpenDocument(uri, text)

	hover, ok := a.Hover(uri, "spec.compositeTypeRef.apiVersion")
	if !ok || !strings.Contains(hover.Markdown, "API version of the composite resource type this Composition renders.") {
		t.Fatalf("hover = %#v ok=%v", hover, ok)
	}
	completion := a.Completion(uri, "spec.compositeTypeRef")
	item, ok := completionItemByLabel(completion.Items, "apiVersion")
	if !ok {
		t.Fatalf("completion missing apiVersion: %#v", completion.Items)
	}
	if !strings.Contains(item.Documentation, "API version of the composite resource type this Composition renders.") {
		t.Fatalf("apiVersion completion documentation = %q", item.Documentation)
	}
}

func TestAnalyzerCompletionSortsRootAndRequiredKeys(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	rootCompletion := a.Completion(uri, "")
	if len(rootCompletion.Items) < 4 {
		t.Fatalf("root completion has %d items, want at least 4: %#v", len(rootCompletion.Items), rootCompletion.Items)
	}
	if got := completionLabels(rootCompletion.Items[:4]); !reflect.DeepEqual(got, []string{"apiVersion", "kind", "metadata", "spec"}) {
		t.Fatalf("root labels = %#v", got)
	}
	specCompletion := a.Completion(uri, "spec.compositeTypeRef")
	got := completionLabels(specCompletion.Items)
	if len(got) < 2 || got[0] != "apiVersion" || got[1] != "kind" {
		t.Fatalf("required compositeTypeRef labels should sort first, got %#v", got)
	}
}

func TestAnalyzerCompletionUsesArrayItemSchemaPath(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition-pipeline.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - functionRef:\n        n"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	item, ok := completionItemByLabel(completion.Items, "name")
	if !ok {
		t.Fatalf("array item completion missing name: %#v", completion.Items)
	}
	if item.TextEdit == nil {
		t.Fatalf("name completion missing text edit: %#v", item)
	}
	if item.TextEdit.NewText != "        name:" {
		t.Fatalf("new text = %q, want eight-space indented name:", item.TextEdit.NewText)
	}
	if got, want := item.TextEdit.Replace, (Span{Start: strings.LastIndex(text, "        n"), End: len(text)}); got != want {
		t.Fatalf("replace span = %#v, want %#v", got, want)
	}
}

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

func TestAnalyzerCompletionAtOffsetDoesNotFallbackFromPipelineArrayToSpecObject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crossplane.yaml"), []byte("apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \">=v2.0.0\"\n"), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "composition.yaml")
	prefix := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: root-composition\n"
	tests := []struct {
		name      string
		spec      string
		wantWrite bool
	}{
		{
			name: "spec child",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"  w",
			wantWrite: true,
		},
		{
			name: "array value position",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"    w",
		},
		{
			name: "array item sibling",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"      w",
		},
		{
			name: "nested object child",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"        w",
		},
		{
			name: "scalar descendant",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"          w",
		},
		{
			name: "array item key prefix without match",
			spec: "spec:\n" +
				"  compositeTypeRef: foo\n" +
				"  mode: Pipline\n" +
				"  pipeline:\n" +
				"    - functionRef:\n" +
				"        name: bar\n" +
				"    - w",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := prefix + tt.spec
			a.OpenDocument(uri, text)
			completion := a.CompletionAtOffset(uri, len(text))
			gotWrite := containsCompletion(completion.Items, "writeConnectionSecretsToNamespace")
			if gotWrite != tt.wantWrite {
				t.Fatalf("writeConnectionSecretsToNamespace offered = %v, want %v: %#v", gotWrite, tt.wantWrite, completion.Items)
			}
		})
	}
}

func TestAnalyzerCompletionSuppressesRootStatusOnly(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	if containsCompletion(a.Completion(uri, "").Items, "status") {
		t.Fatal("root status should be suppressed")
	}
}

func TestCompletionDoesNotSuppressNestedStatusSchemaPath(t *testing.T) {
	idx := NewSchemaIndex()
	release := CrossplaneRelease{Tag: "v2.2.1"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: release,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "CompositeResourceDefinition"},
		Fields: map[string]FieldDoc{
			"spec.versions[].schema.openAPIV3Schema.properties.status": {
				Path:        "spec.versions[].schema.openAPIV3Schema.properties.status",
				Description: "Status schema property.",
			},
		},
	})

	completion := completionFromSchema(idx, release, "apiextensions.crossplane.io/v1", "CompositeResourceDefinition", "spec.versions[].schema.openAPIV3Schema.properties")
	if !containsCompletion(completion.Items, "status") {
		t.Fatalf("nested schema status should not be suppressed: %#v", completion.Items)
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

func TestAnalyzerCompletionReportsMissingRootGVKReason(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "missing-gvk.yaml")
	a.OpenDocument(uri, "spec:\n  ")

	completion := a.CompletionAtOffset(uri, len("spec:\n  "))
	if completion.Reason != SuppressionMissingRootGVK {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionMissingRootGVK)
	}
}

func TestAnalyzerCompletionReportsMalformedYAMLReason(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "malformed.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: [unterminated\n  "
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if completion.Reason != SuppressionMalformedYAMLContext {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionMalformedYAMLContext)
	}
}

func TestAnalyzerCompletionReportsUnstableTemplatePathReason(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "template-key.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    {{ .Key }}\n"
	a.OpenDocument(uri, text)
	offset := strings.Index(text, "{{ .Key }}") + len("{{ ")
	if offset < len("{{ ") {
		t.Fatal("test setup: template action not found")
	}

	completion := a.CompletionAtOffset(uri, offset)
	if completion.Reason != SuppressionUnstableTemplatePath {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionUnstableTemplatePath)
	}
}

func TestAnalyzerCompletionSuppressionReasonsRequireKeyContext(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}

	malformedURI := "file://" + filepath.Join(root, "api", "malformed-value.yaml")
	malformedText := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: [unterminated\nmetadata:\n  name: demo\n"
	a.OpenDocument(malformedURI, malformedText)
	malformedOffset := strings.Index(malformedText, "demo")
	if malformedOffset < 0 {
		t.Fatal("test setup: demo not found")
	}
	if completion := a.CompletionAtOffset(malformedURI, malformedOffset); completion.Reason != "" {
		t.Fatalf("malformed value-context reason = %q, want empty", completion.Reason)
	}

	templateURI := "file://" + filepath.Join(root, "api", "template-value.yaml")
	templateText := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: {{ .Kind }}\n"
	a.OpenDocument(templateURI, templateText)
	templateOffset := strings.Index(templateText, "{{ .Kind }}") + len("{{ ")
	if templateOffset < len("{{ ") {
		t.Fatal("test setup: template action not found")
	}
	if completion := a.CompletionAtOffset(templateURI, templateOffset); completion.Reason != "" {
		t.Fatalf("template value-context reason = %q, want empty", completion.Reason)
	}

	disabled, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits(), SchemaBundleFS: unsupportedSchemaBundleFS()})
	if err != nil {
		t.Fatalf("new analyzer with disabled bundle: %v", err)
	}
	bundleURI := "file://" + filepath.Join(root, "api", "bundle-value.yaml")
	bundleText := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nmetadata:\n  name: demo\n"
	disabled.OpenDocument(bundleURI, bundleText)
	bundleOffset := strings.Index(bundleText, "demo")
	if bundleOffset < 0 {
		t.Fatal("test setup: demo not found")
	}
	if completion := disabled.CompletionAtOffset(bundleURI, bundleOffset); completion.Reason != "" {
		t.Fatalf("bundle-disabled value-context reason = %q, want empty", completion.Reason)
	}
}

func TestAnalyzerCompletionMalformedEarlierDocumentDoesNotClassifyLaterDocument(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "malformed-before-valid.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: [unterminated\n---\napiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    "
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if completion.Reason == SuppressionMalformedYAMLContext {
		t.Fatalf("later document reason = %q, want non-malformed", completion.Reason)
	}
	if len(completion.Items) != 0 && !containsCompletion(completion.Items, "kind") {
		t.Fatalf("later document completion = %#v, want kind when items are available", completion.Items)
	}
}

func TestAnalyzerNewAllowsDisabledSchemaBundle(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits(), SchemaBundleFS: unsupportedSchemaBundleFS()})
	if err != nil {
		t.Fatalf("new analyzer with disabled bundle: %v", err)
	}
	status := a.SchemaBundleStatus()
	if status.OK || !strings.Contains(status.Message, "unsupported schema bundle format 99") {
		t.Fatalf("bundle status = %#v, want disabled unsupported-format status", status)
	}

	uri := "file://" + filepath.Join(root, "api", "disabled-bundle.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    "
	a.OpenDocument(uri, text)
	completion := a.CompletionAtOffset(uri, len(text))
	if completion.Reason != SuppressionBundleDisabled {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionBundleDisabled)
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

func TestAnalyzerCompletionAtOffsetDoesNotOfferExistingRootKeysFromNestedFallback(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "completion-spec-fallback.yaml")
	base := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: root-package\nspec:\n  "
	tests := []struct {
		prefix string
		label  string
	}{
		{prefix: "a", label: "apiVersion"},
		{prefix: "k", label: "kind"},
		{prefix: "m", label: "metadata"},
		{prefix: "s", label: "spec"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			text := base + tt.prefix
			a.ChangeDocument(uri, text)
			completion := a.CompletionAtOffset(uri, len(text))
			if containsCompletion(completion.Items, tt.label) {
				t.Fatalf("nested fallback offered existing root key %q for prefix %q: %#v", tt.label, tt.prefix, completion.Items)
			}
		})
	}
}

func TestAnalyzerCompletionAtOffsetStillCompletesSpecChildren(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "completion-spec-child.yaml")
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: root-package\nspec:\n  d"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "dependsOn") {
		t.Fatalf("expected dependsOn completion under spec, got %#v", completion.Items)
	}
	item, ok := completionItemByLabel(completion.Items, "dependsOn")
	if !ok || !strings.Contains(item.Documentation, "Package dependencies required by this Configuration.") {
		t.Fatalf("dependsOn documentation = %q ok=%v, want compatibility parent docs", item.Documentation, ok)
	}
	rootCompletion := a.Completion(uri, "")
	item, ok = completionItemByLabel(rootCompletion.Items, "spec")
	if !ok || !strings.Contains(item.Documentation, "Configuration package specification.") {
		t.Fatalf("spec documentation = %q ok=%v, want compatibility parent docs", item.Documentation, ok)
	}
}

func TestAnalyzerCompletionAtOffsetFallbackScopedToCurrentDocument(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "completion-multi-doc-fallback.yaml")
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: first\n---\napiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nm"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "metadata") {
		t.Fatalf("metadata absent from doc 2 should still be offered despite presence in doc 1: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionAtOffsetSkipsDedentWhenRootKeyExists(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "completion-no-dedent-when-present.yaml")
	text := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: root-package\nspec:\n  something: x\n  s"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if _, ok := completionItemByLabel(completion.Items, "spec"); ok {
		t.Fatalf("dedent to existing root key spec should be suppressed: %#v", completion.Items)
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

func TestAnalyzerCompletionUsesPackageCrossplaneVersionConstraint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crossplane.yaml"), []byte("apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \">=v1.20.0 <v2.0.0\"\n"), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  r"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "resources") {
		t.Fatalf("v1 constrained package should offer resources when present in v1 schema: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionUsesUpperBoundPackageCrossplaneVersionConstraint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crossplane.yaml"), []byte("apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \"< v2.0.0\"\n"), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pa"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "patchSets") {
		t.Fatalf("upper-bound v1 package should offer patchSets when present in v1 schema: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionUsesOpenPackageMarkerVersion(t *testing.T) {
	root := t.TempDir()
	markerPath := filepath.Join(root, "crossplane.yaml")
	if err := os.WriteFile(markerPath, []byte("apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \">=v2.0.0\"\n"), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	a.OpenDocument("file://localhost"+filepath.ToSlash(markerPath), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \">=v1.20.0 <v2.0.0\"\n")
	uri := "file://" + filepath.Join(root, "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  r"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "resources") {
		t.Fatalf("open package metadata should override disk marker version: %#v", completion.Items)
	}
}

func TestResolveSchemaReleaseSupportsLowerBoundRange(t *testing.T) {
	gvk := SourceGVK{APIVersion: "example.io/v1", Kind: "Example"}
	before := CrossplaneRelease{Tag: "v1.12.0"}
	after := CrossplaneRelease{Tag: "v1.12.1"}
	a := analyzerWithPackageMarkerAndGeneratedSchemas(t, ">=v1.12.1-0", []Schema{
		{Release: before, GVK: gvk, Fields: map[string]FieldDoc{"spec.before": {Path: "spec.before"}}},
		{Release: after, GVK: gvk, Fields: map[string]FieldDoc{"spec.after": {Path: "spec.after"}}},
	})

	got := a.resolveSchemaRelease("file://"+filepath.Join(a.workspace.Root, "resource.yaml"), gvk)
	if !got.OK || got.Release != after {
		t.Fatalf("release = %#v, want %#v", got, after)
	}
}

func TestResolveSchemaReleaseSupportsBoundedRange(t *testing.T) {
	gvk := SourceGVK{APIVersion: "example.io/v1", Kind: "Example"}
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	a := analyzerWithPackageMarkerAndGeneratedSchemas(t, ">=v1.20.0 <v2.0.0", []Schema{
		{Release: v1, GVK: gvk, Fields: map[string]FieldDoc{"spec.v1": {Path: "spec.v1"}}},
		{Release: v2, GVK: gvk, Fields: map[string]FieldDoc{"spec.v2": {Path: "spec.v2"}}},
	})

	got := a.resolveSchemaRelease("file://"+filepath.Join(a.workspace.Root, "resource.yaml"), gvk)
	if !got.OK || got.Release != v1 {
		t.Fatalf("release = %#v, want %#v", got, v1)
	}
}

func TestResolveSchemaReleaseSupportsUpperBoundRange(t *testing.T) {
	gvk := SourceGVK{APIVersion: "example.io/v1", Kind: "Example"}
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	a := analyzerWithPackageMarkerAndGeneratedSchemas(t, "< v2.0.0", []Schema{
		{Release: v1, GVK: gvk, Fields: map[string]FieldDoc{"spec.v1": {Path: "spec.v1"}}},
		{Release: v2, GVK: gvk, Fields: map[string]FieldDoc{"spec.v2": {Path: "spec.v2"}}},
	})

	got := a.resolveSchemaRelease("file://"+filepath.Join(a.workspace.Root, "resource.yaml"), gvk)
	if !got.OK || got.Release != v1 {
		t.Fatalf("release = %#v, want %#v", got, v1)
	}
}

func TestResolveSchemaReleaseUnsupportedRangeFallsBackToLatestGVK(t *testing.T) {
	gvk := SourceGVK{APIVersion: "example.io/v1", Kind: "Example"}
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	a := analyzerWithPackageMarkerAndGeneratedSchemas(t, "^1.20.0", []Schema{
		{Release: v1, GVK: gvk, Fields: map[string]FieldDoc{"spec.v1": {Path: "spec.v1"}}},
		{Release: v2, GVK: gvk, Fields: map[string]FieldDoc{"spec.v2": {Path: "spec.v2"}}},
	})

	got := a.resolveSchemaRelease("file://"+filepath.Join(a.workspace.Root, "resource.yaml"), gvk)
	if !got.OK || got.Release != v2 {
		t.Fatalf("release = %#v, want latest exact-GVK release %#v", got, v2)
	}
}

func TestAnalyzerWorkspaceSchemaCompletionAndHover(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	a.schemas.AddWorkspaceSchema(Schema{
		GVK: SourceGVK{APIVersion: "s3.aws.upbound.io/v1beta1", Kind: "Bucket"},
		Fields: map[string]FieldDoc{
			"spec.forProvider.bucketName": {Path: "spec.forProvider.bucketName", Description: "Workspace bucket name."},
		},
		Provenance: SchemaProvenance{Path: "provider-crd.yaml", Owner: SchemaOwnerProvider, Source: SchemaSourceWorkspace},
	})
	uri := "file://" + filepath.Join(root, "api", "bucket.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n    bucketName: example\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.forProvider")
	if !containsCompletion(completion.Items, "bucketName") {
		t.Fatalf("workspace schema completion missing bucketName: %#v", completion.Items)
	}
	hover, ok := a.Hover(uri, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(hover.Markdown, "Workspace bucket name.") {
		t.Fatalf("workspace schema hover = %#v ok=%v", hover, ok)
	}
}

func TestAnalyzerLoadsWorkspaceProviderCRDSchema(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.forProvider")
	if !containsCompletion(completion.Items, "bucketName") {
		t.Fatalf("workspace provider CRD completion missing bucketName: %#v", completion.Items)
	}
	item, ok := completionItemByLabel(completion.Items, "bucketName")
	if !ok || !strings.Contains(item.Documentation, "BucketName is the provider bucket name.") {
		t.Fatalf("bucketName completion = %#v, want provider CRD documentation", item)
	}
	hover, ok := a.Hover(uri, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(hover.Markdown, "BucketName is the provider bucket name.") {
		t.Fatalf("hover = %#v ok=%v, want provider CRD documentation", hover, ok)
	}
}

func TestAnalyzerLoadsWorkspaceXRDSchema(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "xdb.yaml")
	text := "apiVersion: platform.example.org/v1alpha1\nkind: XDatabase\nspec:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec")
	item, ok := completionItemByLabel(completion.Items, "size")
	if !ok {
		t.Fatalf("workspace XRD completion missing size: %#v", completion.Items)
	}
	if !strings.Contains(item.Documentation, "Size selects the database capacity class.") {
		t.Fatalf("size completion = %#v, want XRD documentation", item)
	}
	if !strings.Contains(item.Documentation, "_Allowed: small, large_") {
		t.Fatalf("size completion = %#v, want enum documentation", item)
	}
}

func TestAnalyzerScopesWorkspaceProviderCRDSchemaByPackage(t *testing.T) {
	root := t.TempDir()
	pkgA := filepath.Join(root, "packages", "a")
	pkgB := filepath.Join(root, "packages", "b")
	analyzerWriteFile(t, filepath.Join(pkgA, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package-a\n")
	analyzerWriteFile(t, filepath.Join(pkgB, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package-b\n")
	analyzerWriteFile(t, filepath.Join(pkgA, "api", "provider-crd.yaml"), workspaceProviderCRD("bucketName", "Package A bucket name."))
	analyzerWriteFile(t, filepath.Join(pkgB, "api", "provider-crd.yaml"), workspaceProviderCRD("bucketName", "Package B bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}

	uriA := "file://" + filepath.Join(pkgA, "api", "bucket-instance.yaml")
	uriB := "file://" + filepath.Join(pkgB, "api", "bucket-instance.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(uriA, text)
	a.OpenDocument(uriB, text)

	completionA := a.Completion(uriA, "spec.forProvider")
	itemA, ok := completionItemByLabel(completionA.Items, "bucketName")
	if !ok || !strings.Contains(itemA.Documentation, "Package A bucket name.") || strings.Contains(itemA.Documentation, "Package B bucket name.") {
		t.Fatalf("package A bucketName completion = %#v", itemA)
	}
	hoverA, ok := a.Hover(uriA, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(hoverA.Markdown, "Package A bucket name.") || strings.Contains(hoverA.Markdown, "Package B bucket name.") {
		t.Fatalf("package A hover = %#v ok=%v", hoverA, ok)
	}

	completionB := a.Completion(uriB, "spec.forProvider")
	itemB, ok := completionItemByLabel(completionB.Items, "bucketName")
	if !ok || !strings.Contains(itemB.Documentation, "Package B bucket name.") || strings.Contains(itemB.Documentation, "Package A bucket name.") {
		t.Fatalf("package B bucketName completion = %#v", itemB)
	}
	hoverB, ok := a.Hover(uriB, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(hoverB.Markdown, "Package B bucket name.") || strings.Contains(hoverB.Markdown, "Package A bucket name.") {
		t.Fatalf("package B hover = %#v ok=%v", hoverB, ok)
	}
}

func TestAnalyzerSkipsNestedPackageWorkspaceProviderCRDSchema(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "packages", "child")
	analyzerWriteFile(t, filepath.Join(parent, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: parent\n")
	analyzerWriteFile(t, filepath.Join(child, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: child\n")
	analyzerWriteFile(t, filepath.Join(parent, "api", "provider-crd.yaml"), workspaceProviderCRD("bucketName", "Parent bucket name."))
	analyzerWriteFile(t, filepath.Join(child, "api", "provider-crd.yaml"), workspaceProviderCRD("childOnlyName", "Child bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	parentURI := "file://" + filepath.Join(parent, "api", "bucket-instance.yaml")
	childURI := "file://" + filepath.Join(child, "api", "bucket-instance.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(parentURI, text)
	a.OpenDocument(childURI, text)

	parentHover, ok := a.Hover(parentURI, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(parentHover.Markdown, "Parent bucket name.") || strings.Contains(parentHover.Markdown, "Child bucket name.") {
		t.Fatalf("parent hover = %#v ok=%v", parentHover, ok)
	}
	parentCompletion := a.Completion(parentURI, "spec.forProvider")
	if containsCompletion(parentCompletion.Items, "childOnlyName") {
		t.Fatalf("parent completion contains nested child CRD field: %#v", parentCompletion.Items)
	}
	assertBucketCompletionDoc(t, parentCompletion, "bucketName", "Parent bucket name.")
	parentSchema, ok := a.workspaceSchemas[workspaceSchemaKey{
		PackageRoot: parent,
		GVK:         SourceGVK{APIVersion: "s3.aws.upbound.io/v1beta1", Kind: "Bucket"},
	}]
	if !ok {
		t.Fatal("parent workspace schema missing")
	}
	if _, ok := parentSchema.Fields["spec.forProvider.childOnlyName"]; ok {
		t.Fatalf("parent workspace schema ingested nested child CRD fields: %#v", parentSchema.Fields)
	}

	childHover, ok := a.Hover(childURI, "spec.forProvider.childOnlyName")
	if !ok || !strings.Contains(childHover.Markdown, "Child bucket name.") || strings.Contains(childHover.Markdown, "Parent bucket name.") {
		t.Fatalf("child hover = %#v ok=%v", childHover, ok)
	}
	childCompletion := a.Completion(childURI, "spec.forProvider")
	if containsCompletion(childCompletion.Items, "bucketName") {
		t.Fatalf("child completion contains parent CRD field: %#v", childCompletion.Items)
	}
	assertBucketCompletionDoc(t, childCompletion, "childOnlyName", "Child bucket name.")
	if diagnostics := a.Diagnostics(parentURI); containsDiagnosticMessage(diagnostics, "workspace schema conflicts with another workspace schema") {
		t.Fatalf("parent diagnostics = %#v, want no nested child conflict", diagnostics)
	}
	if diagnostics := a.workspaceSchemaDiagnostics[parent]; containsDiagnosticMessage(diagnostics, "workspace schema conflicts with another workspace schema") {
		t.Fatalf("parent package schema diagnostics = %#v, want no nested child conflict", diagnostics)
	}
}

func TestAnalyzerWorkspaceProviderCRDArrayItemHover(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	analyzerWriteFile(t, filepath.Join(root, "api", "provider-crd.yaml"), workspaceProviderCRDWithArray("Rule name is the provider rule name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n    rules:\n      - name: primary\n"
	a.OpenDocument(uri, text)

	hover, ok := a.Hover(uri, "spec.forProvider.rules[0].name")
	if !ok || !strings.Contains(hover.Markdown, "Rule name is the provider rule name.") {
		t.Fatalf("array direct hover = %#v ok=%v", hover, ok)
	}
	offset := strings.Index(text, "primary")
	if offset < 0 {
		t.Fatal("test setup: primary value not found")
	}
	hover, ok = a.HoverAtOffset(uri, offset)
	if !ok || !strings.Contains(hover.Markdown, "Rule name is the provider rule name.") {
		t.Fatalf("array offset hover = %#v ok=%v", hover, ok)
	}
}

func TestAnalyzerRefreshesWorkspaceProviderCRDSchemaFromOpenDocument(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	crdPath := filepath.Join(root, "api", "provider-crd.yaml")
	analyzerWriteFile(t, crdPath, workspaceProviderCRD("diskName", "Disk bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	resourceURI := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	resourceText := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(resourceURI, resourceText)

	assertBucketCompletionDoc(t, a.Completion(resourceURI, "spec.forProvider"), "diskName", "Disk bucket name.")

	crdURI := "file://" + crdPath
	a.OpenDocument(crdURI, workspaceProviderCRD("openName", "Open document bucket name."))
	completion := a.Completion(resourceURI, "spec.forProvider")
	assertBucketCompletionDoc(t, completion, "openName", "Open document bucket name.")
	if containsCompletion(completion.Items, "diskName") {
		t.Fatalf("completion still contains diskName after open document refresh: %#v", completion.Items)
	}

	a.ChangeDocument(crdURI, workspaceProviderCRD("changedName", "Changed document bucket name."))
	completion = a.Completion(resourceURI, "spec.forProvider")
	assertBucketCompletionDoc(t, completion, "changedName", "Changed document bucket name.")
	if containsCompletion(completion.Items, "openName") {
		t.Fatalf("completion still contains openName after change document refresh: %#v", completion.Items)
	}

	a.CloseDocument(crdURI)
	completion = a.Completion(resourceURI, "spec.forProvider")
	assertBucketCompletionDoc(t, completion, "diskName", "Disk bucket name.")
	if containsCompletion(completion.Items, "changedName") {
		t.Fatalf("completion still contains changedName after close document refresh: %#v", completion.Items)
	}
}

func TestAnalyzerSurfacesPackageWorkspaceSchemaDuplicateDiagnostics(t *testing.T) {
	root := t.TempDir()
	pkg := filepath.Join(root, "packages", "a")
	analyzerWriteFile(t, filepath.Join(pkg, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package-a\n")
	firstCRD := filepath.Join(pkg, "api", "provider-crd-a.yaml")
	secondCRD := filepath.Join(pkg, "api", "provider-crd-b.yaml")
	analyzerWriteFile(t, firstCRD, workspaceProviderCRD("bucketName", "First bucket name."))
	analyzerWriteFile(t, secondCRD, workspaceProviderCRD("bucketName", "Second bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	bucketURI := "file://" + filepath.Join(pkg, "api", "bucket-instance.yaml")
	a.OpenDocument(bucketURI, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\n")
	crdURI := "file://" + secondCRD
	a.OpenDocument(crdURI, workspaceProviderCRD("bucketName", "Second bucket name."))

	if diagnostics := a.Diagnostics(bucketURI); containsDiagnosticMessage(diagnostics, "workspace schema conflicts with another workspace schema") {
		t.Fatalf("bucket diagnostics = %#v, want no source CRD warning", diagnostics)
	}
	diagnostics := a.Diagnostics(crdURI)
	if !containsDiagnosticMessage(diagnostics, "workspace schema conflicts with another workspace schema") {
		t.Fatalf("diagnostics = %#v, want workspace schema conflict warning", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == "workspace schema conflicts with another workspace schema" && diagnostic.URI != crdURI {
			t.Fatalf("duplicate diagnostic URI = %q, want %q", diagnostic.URI, crdURI)
		}
	}
}

func TestAnalyzerKeepsGeneratedBuiltInSchemaWhenWorkspaceCRDDuplicatesCoreGVK(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	crdPath := filepath.Join(root, "api", "composition-crd.yaml")
	analyzerWriteFile(t, crdPath, workspaceCompositionDuplicateCRD())
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}

	compositionURI := "file://" + filepath.Join(root, "api", "composition.yaml")
	compositionText := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  compositeTypeRef:\n    kind: CompositeBucket\n"
	a.OpenDocument(compositionURI, compositionText)

	hover, ok := a.Hover(compositionURI, "spec.compositeTypeRef.kind")
	if !ok || !strings.Contains(hover.Markdown, "Kind of the composite resource type this Composition renders.") {
		t.Fatalf("hover = %#v ok=%v, want generated built-in documentation", hover, ok)
	}
	if strings.Contains(hover.Markdown, "Workspace duplicate kind documentation.") {
		t.Fatalf("hover = %#v, want generated built-in documentation instead of workspace duplicate", hover)
	}
	completion := a.Completion(compositionURI, "spec")
	if containsCompletion(completion.Items, "workspaceOnly") {
		t.Fatalf("completion contains workspace duplicate-only field: %#v", completion.Items)
	}
	item, ok := completionItemByLabel(a.Completion(compositionURI, "spec.compositeTypeRef").Items, "kind")
	if !ok || !strings.Contains(item.Documentation, "Kind of the composite resource type this Composition renders.") {
		t.Fatalf("kind completion = %#v, want generated built-in documentation", item)
	}

	crdURI := "file://" + crdPath
	a.OpenDocument(crdURI, workspaceCompositionDuplicateCRD())
	diagnostics := a.Diagnostics(crdURI)
	if !containsDiagnosticMessage(diagnostics, "workspace schema duplicates built-in Crossplane core schema") {
		t.Fatalf("diagnostics = %#v, want duplicate built-in warning", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == "workspace schema duplicates built-in Crossplane core schema" && diagnostic.URI != crdURI {
			t.Fatalf("duplicate built-in diagnostic URI = %q, want %q", diagnostic.URI, crdURI)
		}
	}
}

func TestAnalyzerKeepsSameGVKWorkspaceSchemaDiagnosticsPackageScoped(t *testing.T) {
	root := t.TempDir()
	pkgA := filepath.Join(root, "packages", "a")
	pkgB := filepath.Join(root, "packages", "b")
	analyzerWriteFile(t, filepath.Join(pkgA, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package-a\n")
	analyzerWriteFile(t, filepath.Join(pkgB, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package-b\n")
	analyzerWriteFile(t, filepath.Join(pkgA, "api", "provider-crd.yaml"), workspaceProviderCRD("bucketName", "Package A bucket name."))
	analyzerWriteFile(t, filepath.Join(pkgB, "api", "provider-crd.yaml"), workspaceProviderCRD("bucketName", "Package B bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uriA := "file://" + filepath.Join(pkgA, "api", "bucket-instance.yaml")
	uriB := "file://" + filepath.Join(pkgB, "api", "bucket-instance.yaml")
	a.OpenDocument(uriA, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\n")
	a.OpenDocument(uriB, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\n")

	for _, uri := range []string{uriA, uriB} {
		diagnostics := a.Diagnostics(uri)
		if containsDiagnosticMessage(diagnostics, "workspace schema conflicts with another workspace schema") {
			t.Fatalf("%s diagnostics = %#v, want no cross-package conflict warning", uri, diagnostics)
		}
	}
}

func TestAnalyzerLoadsUnsavedNewWorkspaceProviderCRDSchema(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	resourceURI := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	a.OpenDocument(resourceURI, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n")
	if completion := a.Completion(resourceURI, "spec.forProvider"); containsCompletion(completion.Items, "unsavedName") {
		t.Fatalf("completion unexpectedly contains unsavedName before CRD opens: %#v", completion.Items)
	}

	crdURI := "file://" + filepath.Join(root, "api", "unsaved-provider-crd.yaml")
	a.OpenDocument(crdURI, workspaceProviderCRD("unsavedName", "Unsaved document bucket name."))

	assertBucketCompletionDoc(t, a.Completion(resourceURI, "spec.forProvider"), "unsavedName", "Unsaved document bucket name.")
}

func TestAnalyzerIgnoresOpenWorkspaceProviderCRDInIgnoredDirectory(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	analyzerWriteFile(t, filepath.Join(root, "api", "provider-crd.yaml"), workspaceProviderCRD("diskName", "Disk bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	resourceURI := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	a.OpenDocument(resourceURI, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n")
	assertBucketCompletionDoc(t, a.Completion(resourceURI, "spec.forProvider"), "diskName", "Disk bucket name.")

	ignoredCRDURI := "file://" + filepath.Join(root, "vendor", "provider-crd.yaml")
	a.OpenDocument(ignoredCRDURI, workspaceProviderCRD("ignoredName", "Ignored vendor bucket name."))

	completion := a.Completion(resourceURI, "spec.forProvider")
	assertBucketCompletionDoc(t, completion, "diskName", "Disk bucket name.")
	if containsCompletion(completion.Items, "ignoredName") {
		t.Fatalf("completion contains ignored vendor CRD field: %#v", completion.Items)
	}
	if diagnostics := a.Diagnostics(ignoredCRDURI); containsDiagnosticSourceMessage(diagnostics, "schema", "workspace schema conflicts with another workspace schema") {
		t.Fatalf("ignored CRD diagnostics = %#v, want no workspace schema conflict", diagnostics)
	}
}

func TestAnalyzerDoesNotTreatOrdinaryYAMLAsWorkspaceSchemaSource(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	analyzerWriteFile(t, filepath.Join(root, "api", "provider-crd.yaml"), workspaceProviderCRD("diskName", "Disk bucket name."))
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	resourceURI := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	a.OpenDocument(resourceURI, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n    diskName: [unterminated\n")

	assertBucketCompletionDoc(t, a.Completion(resourceURI, "spec.forProvider"), "diskName", "Disk bucket name.")
	if diagnostics := a.Diagnostics(resourceURI); containsDiagnosticSourceMessage(diagnostics, "schema", "parse workspace schema source") {
		t.Fatalf("ordinary resource diagnostics = %#v, want no workspace schema parse diagnostic", diagnostics)
	}
}

func TestAnalyzerLoadsUnsavedMultiDocumentWorkspaceProviderCRD(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	resourceURI := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	a.OpenDocument(resourceURI, "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n")
	crdURI := "file://" + filepath.Join(root, "api", "multi-doc-provider-crd.yaml")
	a.OpenDocument(crdURI, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: ignored\n---\n"+workspaceProviderCRD("multiDocName", "Multi document bucket name."))

	assertBucketCompletionDoc(t, a.Completion(resourceURI, "spec.forProvider"), "multiDocName", "Multi document bucket name.")
}

func TestAnalyzerSurfacesMalformedWorkspaceProviderCRDDiagnostic(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	crdURI := "file://" + filepath.Join(root, "api", "provider-crd.yaml")
	a.OpenDocument(crdURI, "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: [unterminated\n")

	diagnostics := a.Diagnostics(crdURI)
	if !containsDiagnosticSourceMessage(diagnostics, "schema", "parse workspace schema source") {
		t.Fatalf("diagnostics = %#v, want schema parse diagnostic", diagnostics)
	}

	a.ChangeDocument(crdURI, "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: [still-unterminated\n")
	diagnostics = a.Diagnostics(crdURI)
	if !containsDiagnosticSourceMessage(diagnostics, "schema", "parse workspace schema source") {
		t.Fatalf("diagnostics after change = %#v, want schema parse diagnostic", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Source == "schema" && strings.Contains(diagnostic.Message, "parse workspace schema source") && diagnostic.URI != crdURI {
			t.Fatalf("malformed diagnostic URI = %q, want %q", diagnostic.URI, crdURI)
		}
	}
}

func TestAnalyzerWorkspaceSchemaDiagnosticMatchesEscapedFileURI(t *testing.T) {
	root := filepath.Join(t.TempDir(), "package with space")
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	crdPath := filepath.Join(root, "api", "provider crd.yaml")
	crdURI := (&url.URL{Scheme: "file", Path: crdPath}).String()
	a.OpenDocument(crdURI, "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: [unterminated\n")

	diagnostics := a.Diagnostics(crdURI)
	if !containsDiagnosticSourceMessage(diagnostics, "schema", "parse workspace schema source") {
		t.Fatalf("diagnostics = %#v, want schema parse diagnostic for escaped URI %q", diagnostics, crdURI)
	}
}

func TestAnalyzerWorkspaceSchemaDiagnosticMatchesLocalhostFileURI(t *testing.T) {
	root := t.TempDir()
	analyzerWriteFile(t, filepath.Join(root, "crossplane.yaml"), "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nmetadata:\n  name: package\n")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	crdURI := "file://localhost" + filepath.ToSlash(filepath.Join(root, "api", "provider-crd.yaml"))
	a.OpenDocument(crdURI, "apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nspec:\n  group: [unterminated\n")

	diagnostics := a.Diagnostics(crdURI)
	if !containsDiagnosticSourceMessage(diagnostics, "schema", "parse workspace schema source") {
		t.Fatalf("diagnostics = %#v, want schema parse diagnostic for localhost URI %q", diagnostics, crdURI)
	}
}

func TestAnalyzerUnknownProviderDoesNotInventFields(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "vpc.yaml")
	text := "apiVersion: ec2.aws.upbound.io/v1beta1\nkind: VPC\nspec:\n  forProvider:\n"
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

func analyzerWithPackageMarkerAndGeneratedSchemas(t *testing.T, versionRange string, schemas []Schema) *Analyzer {
	t.Helper()
	root := t.TempDir()
	marker := "apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \"" + versionRange + "\"\n"
	if err := os.WriteFile(filepath.Join(root, "crossplane.yaml"), []byte(marker), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	workspace, err := DetectWorkspace(root)
	if err != nil {
		t.Fatalf("detect workspace: %v", err)
	}
	idx := NewSchemaIndex()
	idx.bundleStatus = SchemaBundleStatus{OK: true}
	for _, schema := range schemas {
		idx.AddGeneratedBuiltIn(schema)
	}
	return &Analyzer{
		workspace: workspace,
		limits:    DefaultLimits(),
		docs:      NewDocumentStore(),
		schemas:   idx,
	}
}

func unsupportedSchemaBundleFS() fstest.MapFS {
	return fstest.MapFS{
		"schemadata/manifest.json": {Data: []byte(`{"bundleFormatVersion":99}`)},
	}
}

func analyzerWriteFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create parent directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func workspaceProviderCRD(fieldName, description string) string {
	return `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: buckets.s3.aws.upbound.io
spec:
  group: s3.aws.upbound.io
  names:
    kind: Bucket
    plural: buckets
  scope: Namespaced
  versions:
    - name: v1beta1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                forProvider:
                  type: object
                  properties:
                    ` + fieldName + `:
                      type: string
                      description: ` + description + `
              required:
                - forProvider
`
}

func workspaceProviderCRDWithArray(description string) string {
	return `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: buckets.s3.aws.upbound.io
spec:
  group: s3.aws.upbound.io
  names:
    kind: Bucket
    plural: buckets
  scope: Namespaced
  versions:
    - name: v1beta1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                forProvider:
                  type: object
                  properties:
                    rules:
                      type: array
                      items:
                        type: object
                        properties:
                          name:
                            type: string
                            description: ` + description + `
`
}

func workspaceCompositionDuplicateCRD() string {
	return `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: compositions.apiextensions.crossplane.io
spec:
  group: apiextensions.crossplane.io
  names:
    kind: Composition
    plural: compositions
  scope: Cluster
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                workspaceOnly:
                  type: string
                  description: Workspace duplicate-only field.
                compositeTypeRef:
                  type: object
                  properties:
                    kind:
                      type: string
                      description: Workspace duplicate kind documentation.
`
}

func assertBucketCompletionDoc(t *testing.T, completion Completion, label, doc string) {
	t.Helper()
	item, ok := completionItemByLabel(completion.Items, label)
	if !ok {
		t.Fatalf("completion missing %s: %#v", label, completion.Items)
	}
	if !strings.Contains(item.Documentation, doc) {
		t.Fatalf("%s completion = %#v, want documentation %q", label, item, doc)
	}
}

func containsDiagnosticMessage(diagnostics []Diagnostic, message string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Message == message {
			return true
		}
	}
	return false
}

func containsDiagnosticSourceMessage(diagnostics []Diagnostic, source, messageSubstring string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Source == source && strings.Contains(diagnostic.Message, messageSubstring) {
			return true
		}
	}
	return false
}

func containsCompletion(items []CompletionItem, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}

func completionLabels(items []CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}

func completionItemByLabel(items []CompletionItem, label string) (CompletionItem, bool) {
	for _, item := range items {
		if item.Label == label {
			return item, true
		}
	}
	return CompletionItem{}, false
}
