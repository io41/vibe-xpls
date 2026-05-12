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

func TestUnterminatedTemplateDiagnostic(t *testing.T) {
	mixed := ParseMixedDocument("metadata:\n  name: {{ .Name\n")

	if len(mixed.TemplateDiagnostics) != 1 {
		t.Fatalf("template diagnostics = %d, want 1", len(mixed.TemplateDiagnostics))
	}
	if !strings.Contains(mixed.TemplateDiagnostics[0].Message, "missing closing delimiter") {
		t.Fatalf("diagnostic = %#v", mixed.TemplateDiagnostics[0])
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
