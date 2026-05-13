package analyzer

import (
	"os"
	"strings"
	"testing"

	"github.com/io41/vibe-xpls/internal/testkit"
)

func TestMaskTemplateActionsPreservesLength(t *testing.T) {
	text := "metadata:\n  name: {{ .Name }}\n"

	mixed := ParseMixedDocument(text)

	if len(mixed.MaskedText) != len(text) {
		t.Fatalf("masked length = %d, want %d", len(mixed.MaskedText), len(text))
	}
	if strings.Contains(mixed.MaskedText, "{{") {
		t.Fatalf("masked text still contains template delimiter: %q", mixed.MaskedText)
	}
	if len(mixed.TemplateDiagnostics) != 0 {
		t.Fatalf("unexpected template diagnostics: %#v", mixed.TemplateDiagnostics)
	}
}

func TestMaskTemplateActionsPreservesCRLF(t *testing.T) {
	text := "{{ if .Enabled }}\r\nkind: Composition\r\n{{ end }}\r\n"

	mixed := ParseMixedDocument(text)

	if len(mixed.MaskedText) != len(text) {
		t.Fatalf("masked length = %d, want %d", len(mixed.MaskedText), len(text))
	}
	for i := range text {
		if (text[i] == '\r' || text[i] == '\n') && mixed.MaskedText[i] != text[i] {
			t.Fatalf("line break at offset %d = %q, want %q", i, mixed.MaskedText[i], text[i])
		}
	}
	if strings.Contains(mixed.MaskedText, "{{") || strings.Contains(mixed.MaskedText, "}}") {
		t.Fatalf("masked text still contains template delimiter: %q", mixed.MaskedText)
	}
}

func TestUnterminatedTemplateDiagnostic(t *testing.T) {
	mixed := ParseMixedDocument("metadata:\n  name: {{ .Name\n")

	if len(mixed.TemplateDiagnostics) != 1 {
		t.Fatalf("template diagnostics = %d, want 1", len(mixed.TemplateDiagnostics))
	}
	if len(mixed.Actions) != 1 {
		t.Fatalf("template actions = %d, want 1", len(mixed.Actions))
	}
	if mixed.Actions[0].Span != mixed.TemplateDiagnostics[0].Span {
		t.Fatalf("action span = %#v, diagnostic span = %#v", mixed.Actions[0].Span, mixed.TemplateDiagnostics[0].Span)
	}
	if !strings.Contains(mixed.TemplateDiagnostics[0].Message, "missing closing delimiter") {
		t.Fatalf("diagnostic = %#v", mixed.TemplateDiagnostics[0])
	}
}

func TestQuotedTemplateDelimiterInsideString(t *testing.T) {
	text := "metadata:\n  name: {{ printf \"}}\" }}\n"

	mixed := ParseMixedDocument(text)

	if len(mixed.TemplateDiagnostics) != 0 {
		t.Fatalf("unexpected template diagnostics: %#v", mixed.TemplateDiagnostics)
	}
	if len(mixed.Actions) != 1 {
		t.Fatalf("template actions = %d, want 1", len(mixed.Actions))
	}
	if mixed.Actions[0].Text != "{{ printf \"}}\" }}" {
		t.Fatalf("action text = %q", mixed.Actions[0].Text)
	}
	if strings.Contains(mixed.MaskedText, "}}") {
		t.Fatalf("masked text still contains closing delimiter: %q", mixed.MaskedText)
	}
}

func TestStablePathEligibility(t *testing.T) {
	data, err := os.ReadFile(testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root", "api", "mixed-template.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	doc := ParseYAMLDocument(string(data))

	if !doc.IsStablePath("spec.compositeTypeRef.kind") {
		t.Fatal("expected plain key path to be stable")
	}
	if doc.IsStablePath("spec.xxxxxxxxxxxxxxxxxx") {
		t.Fatal("expected templated key path to be ineligible")
	}
	offset := strings.Index(string(data), "kind: CompositeBucket")
	path, ok := doc.PathAtOffset(offset)
	if !ok || path != "spec.compositeTypeRef.kind" {
		t.Fatalf("path at offset = %q ok=%v, want spec.compositeTypeRef.kind", path, ok)
	}
}

func TestStandaloneTemplateControlActionsDoNotProduceYAMLDiagnostics(t *testing.T) {
	text := "{{ if .Enabled }}\napiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n{{ end }}\n"

	doc := ParseYAMLDocument(text)

	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Source == "yaml" {
			t.Fatalf("unexpected yaml diagnostic: %#v", diagnostic)
		}
	}
	for _, path := range []string{"apiVersion", "kind"} {
		if !doc.IsStablePath(path) {
			t.Fatalf("expected %s to be stable", path)
		}
		offset := strings.Index(text, path+":")
		got, ok := doc.PathAtOffset(offset)
		if !ok || got != path {
			t.Fatalf("path at %s = %q ok=%v, want %s", path, got, ok, path)
		}
	}
}

func TestSequencePathTraversal(t *testing.T) {
	text := `apiVersion: apiextensions.crossplane.io/v1
kind: Composition
spec:
  pipeline:
  - step: render
    functionRef:
      name: function-go-templating
    input:
      apiVersion: gotemplating.fn.crossplane.io/v1beta1
      kind: GoTemplate
`

	doc := ParseYAMLDocument(text)

	for _, path := range []string{
		"spec.pipeline[0].step",
		"spec.pipeline[0].functionRef.name",
		"spec.pipeline[0].input",
	} {
		if !doc.IsStablePath(path) {
			t.Fatalf("expected %s to be stable", path)
		}
	}

	for _, tc := range []struct {
		needle string
		want   string
	}{
		{needle: "step: render", want: "spec.pipeline[0].step"},
		{needle: "render", want: "spec.pipeline[0].step"},
		{needle: "name: function-go-templating", want: "spec.pipeline[0].functionRef.name"},
		{needle: "function-go-templating", want: "spec.pipeline[0].functionRef.name"},
		{needle: "input:", want: "spec.pipeline[0].input"},
		{needle: "apiVersion: gotemplating.fn.crossplane.io/v1beta1", want: "spec.pipeline[0].input.apiVersion"},
	} {
		offset := strings.Index(text, tc.needle)
		if offset < 0 {
			t.Fatalf("test setup: %q not found", tc.needle)
		}
		path, ok := doc.PathAtOffset(offset)
		if !ok || path != tc.want {
			t.Fatalf("path at %q = %q ok=%v, want %s", tc.needle, path, ok, tc.want)
		}
	}
}

func TestMultiDocumentPathAtOffsetUsesOccurrences(t *testing.T) {
	text := "apiVersion: first.example/v1\nkind: First\n---\napiVersion: second.example/v1\nkind: Second\n"

	doc := ParseYAMLDocument(text)

	first := strings.Index(text, "apiVersion")
	second := strings.LastIndex(text, "apiVersion")
	if first < 0 || second <= first {
		t.Fatal("test setup: expected two apiVersion keys")
	}
	for _, offset := range []int{first, second} {
		path, ok := doc.PathAtOffset(offset)
		if !ok || path != "apiVersion" {
			t.Fatalf("path at offset %d = %q ok=%v, want apiVersion", offset, path, ok)
		}
	}
}

func TestDuplicateKeyPathAtOffsetUsesOccurrences(t *testing.T) {
	text := "metadata:\n  name: first\nmetadata:\n  name: second\n"

	doc := ParseYAMLDocument(text)

	first := strings.Index(text, "name:")
	second := strings.LastIndex(text, "name:")
	if first < 0 || second <= first {
		t.Fatal("test setup: expected two name keys")
	}
	for _, offset := range []int{first, second} {
		path, ok := doc.PathAtOffset(offset)
		if !ok || path != "metadata.name" {
			t.Fatalf("path at offset %d = %q ok=%v, want metadata.name", offset, path, ok)
		}
	}
}

func TestSimplePathSpansUseExactByteOffsets(t *testing.T) {
	text := "spec:\n  kind: Bucket\n"

	doc := ParseYAMLDocument(text)

	keyStart := strings.Index(text, "kind")
	valueStart := strings.Index(text, "Bucket")
	path := "spec.kind"
	if got := doc.KeySpans[path]; got != (Span{Start: keyStart, End: keyStart + len("kind")}) {
		t.Fatalf("key span = %#v, want exact kind span", got)
	}
	if got := doc.ValueSpans[path]; got != (Span{Start: valueStart, End: valueStart + len("Bucket")}) {
		t.Fatalf("value span = %#v, want exact Bucket span", got)
	}
	if got := doc.PathSpans[path]; got != (Span{Start: keyStart, End: valueStart + len("Bucket")}) {
		t.Fatalf("path span = %#v, want key-through-value span", got)
	}
}

func TestQuotedValueSpansUseRawSource(t *testing.T) {
	text := "kind: \"Bucket\"\nescaped: \"B\\\"ucket\"\n"

	doc := ParseYAMLDocument(text)

	quotedStart := strings.Index(text, "\"Bucket\"")
	escapedStart := strings.Index(text, "\"B\\\"ucket\"")
	for _, tc := range []struct {
		path  string
		start int
		raw   string
	}{
		{path: "kind", start: quotedStart, raw: "\"Bucket\""},
		{path: "escaped", start: escapedStart, raw: "\"B\\\"ucket\""},
	} {
		if tc.start < 0 {
			t.Fatalf("test setup: raw value %q not found", tc.raw)
		}
		if got := doc.ValueSpans[tc.path]; got != (Span{Start: tc.start, End: tc.start + len(tc.raw)}) {
			t.Fatalf("%s value span = %#v, want raw source span", tc.path, got)
		}
	}
}

func TestNullAndEmptyValuesRemainStable(t *testing.T) {
	text := "spec:\n  empty:\n  explicitNull: null\n"

	doc := ParseYAMLDocument(text)

	for _, path := range []string{"spec.empty", "spec.explicitNull"} {
		if !doc.IsStablePath(path) {
			t.Fatalf("expected %s to be stable", path)
		}
		offset := strings.Index(text, strings.TrimPrefix(path, "spec.")+":")
		got, ok := doc.PathAtOffset(offset)
		if !ok || got != path {
			t.Fatalf("path at %s = %q ok=%v, want %s", path, got, ok, path)
		}
	}
}

func TestTemplateScalarValueIsNotStable(t *testing.T) {
	text := "apiVersion: {{ .APIVersion }}\nkind: Composition\n"

	doc := ParseYAMLDocument(text)

	if doc.IsStablePath("apiVersion") {
		t.Fatal("expected templated scalar value path to be unstable")
	}
	if value, ok := doc.Values["apiVersion"]; ok && value != "" {
		t.Fatalf("templated scalar value recorded as %q", value)
	}
	offset := strings.Index(text, "{{ .APIVersion }}")
	path, ok := doc.PathAtOffset(offset)
	if ok {
		t.Fatalf("path inside template action = %q, want no path", path)
	}
	if !doc.IsStablePath("kind") {
		t.Fatal("expected plain sibling path to remain stable")
	}
}

func TestUnterminatedInlineTemplateScalarValueIsNotStable(t *testing.T) {
	text := "metadata:\n  name: {{ .Name\n"

	doc := ParseYAMLDocument(text)

	if len(doc.Mixed.TemplateDiagnostics) != 1 {
		t.Fatalf("template diagnostics = %d, want 1", len(doc.Mixed.TemplateDiagnostics))
	}
	if !doc.IsStablePath("metadata") {
		t.Fatal("expected parent metadata path to remain stable")
	}
	if doc.IsStablePath("metadata.name") {
		t.Fatal("expected unterminated template scalar path to be unstable")
	}
	if value, ok := doc.Values["metadata.name"]; ok {
		t.Fatalf("unterminated template scalar value recorded as %q", value)
	}
	offset := strings.Index(text, "{{ .Name")
	path, ok := doc.PathAtOffset(offset)
	if ok {
		t.Fatalf("path inside unterminated template action = %q, want no path", path)
	}
}

func TestTemplateSequenceElementIsNotStable(t *testing.T) {
	text := "items: [{{ .Items }}]\nkind: Composition\n"

	doc := ParseYAMLDocument(text)

	if !doc.IsStablePath("items") {
		t.Fatal("expected parent sequence key to remain stable")
	}
	if doc.IsStablePath("items[0]") {
		t.Fatal("expected templated sequence element to be unstable")
	}
	offset := strings.Index(text, "{{ .Items }}")
	path, ok := doc.PathAtOffset(offset)
	if ok {
		t.Fatalf("path inside sequence template action = %q, want no path", path)
	}
	if !doc.IsStablePath("kind") {
		t.Fatal("expected plain sibling path to remain stable")
	}
}

func TestYAMLDiagnosticUsesParserSpan(t *testing.T) {
	text := "apiVersion: v1\nspec: [unterminated\n"

	doc := ParseYAMLDocument(text)

	var yamlDiagnostic Diagnostic
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Source == "yaml" {
			yamlDiagnostic = diagnostic
			break
		}
	}
	if yamlDiagnostic.Source == "" {
		t.Fatalf("expected yaml diagnostic, got %#v", doc.Diagnostics)
	}
	if yamlDiagnostic.Span.Start == 0 && yamlDiagnostic.Span.End == 0 {
		t.Fatalf("expected parser-backed non-zero span, got %#v", yamlDiagnostic)
	}
	if yamlDiagnostic.Span.End <= yamlDiagnostic.Span.Start {
		t.Fatalf("expected non-empty diagnostic span, got %#v", yamlDiagnostic)
	}
}

func TestMalformedYAMLPreservesEarlierBestEffortPaths(t *testing.T) {
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: [unterminated\n"

	doc := ParseYAMLDocument(text)

	var yamlDiagnostic Diagnostic
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Source == "yaml" {
			yamlDiagnostic = diagnostic
			break
		}
	}
	if yamlDiagnostic.Source == "" {
		t.Fatalf("expected yaml diagnostic, got %#v", doc.Diagnostics)
	}
	if yamlDiagnostic.Span.Start == 0 && yamlDiagnostic.Span.End == 0 {
		t.Fatalf("expected non-zero yaml diagnostic span, got %#v", yamlDiagnostic)
	}
	for _, path := range []string{"apiVersion", "kind"} {
		if !doc.IsStablePath(path) {
			t.Fatalf("expected earlier %s path to remain stable", path)
		}
		offset := strings.Index(text, path+":")
		got, ok := doc.PathAtOffset(offset)
		if !ok || got != path {
			t.Fatalf("path at %s = %q ok=%v, want %s", path, got, ok, path)
		}
	}
}

func TestUnterminatedTemplateStillParsesEarlierYAML(t *testing.T) {
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n{{ if .Enabled\nmetadata:\n"

	doc := ParseYAMLDocument(text)

	if len(doc.Mixed.TemplateDiagnostics) != 1 {
		t.Fatalf("template diagnostics = %d, want 1", len(doc.Mixed.TemplateDiagnostics))
	}
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Source == "yaml" {
			t.Fatalf("unexpected yaml diagnostic from unterminated action: %#v", diagnostic)
		}
	}
	for _, path := range []string{"apiVersion", "kind"} {
		if !doc.IsStablePath(path) {
			t.Fatalf("expected %s to be stable", path)
		}
		offset := strings.Index(text, path+":")
		got, ok := doc.PathAtOffset(offset)
		if !ok || got != path {
			t.Fatalf("path at %s = %q ok=%v, want %s", path, got, ok, path)
		}
	}
}
