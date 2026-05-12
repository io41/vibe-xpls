package yamltemplatemapping

import (
	"os"
	"strings"
	"testing"
)

func TestFixtureActionsHaveSourceSpans(t *testing.T) {
	source := loadFixture(t)
	analysis := Analyze(source)

	requiredActions := map[string]string{
		"{{ .APIVersion }}":  "scalar value",
		"{{ .PrimaryTag }}":  "list item",
		"{{ .BlockAction }}": "block scalar",
		"{{ .LabelKey }}":    "mapping key",
		"{{ .DocumentKey }}": "document key",
	}
	for actionText, purpose := range requiredActions {
		action, ok := findAction(analysis.Actions, actionText)
		if !ok {
			t.Fatalf("missing %s action %q", purpose, actionText)
		}
		if got := source[action.Span.Start.Offset:action.Span.End.Offset]; got != actionText {
			t.Fatalf("recorded span for %s = %q, want %q", purpose, got, actionText)
		}
		if action.Span.Start.Line < 0 || action.Span.Start.Column < 0 {
			t.Fatalf("invalid position for %s action: %#v", purpose, action.Span.Start)
		}
	}

	if got := strings.Count(source, "\n---"); got != 2 {
		t.Fatalf("fixture should contain two document separators, got %d", got)
	}
}

func TestMaskedViewPreservesCoordinatesAndParsesYAML(t *testing.T) {
	source := loadFixture(t)
	analysis := Analyze(source)

	if len(analysis.Masked) != len(source) {
		t.Fatalf("masked length = %d, want %d", len(analysis.Masked), len(source))
	}
	if strings.Contains(analysis.Masked, "{{") || strings.Contains(analysis.Masked, "}}") {
		t.Fatalf("masked view still contains template delimiters:\n%s", analysis.Masked)
	}

	for _, action := range analysis.Actions {
		maskedAction := analysis.Masked[action.Span.Start.Offset:action.Span.End.Offset]
		for i, char := range maskedAction {
			if char != 'x' && char != '\n' && char != '\r' {
				t.Fatalf("masked action %q has non-mask character %q at relative offset %d", maskedAction, char, i)
			}
		}
	}

	rawYAMLDiagnostics := ParseMaskedYAML(analysis.Masked)
	if len(rawYAMLDiagnostics) != 1 {
		t.Fatalf("masked YAML diagnostics = %d, want 1: %#v", len(rawYAMLDiagnostics), rawYAMLDiagnostics)
	}

	mapped := MapMaskedDiagnostics(source, rawYAMLDiagnostics, analysis.Actions)
	if mapped[0].InsideTemplate {
		t.Fatalf("mapped YAML diagnostic should be outside a template span: %#v", mapped[0])
	}
	if got := source[mapped[0].Span.Start.Offset:mapped[0].Span.End.Offset]; got != "invalidLineWithoutColon" {
		t.Fatalf("mapped YAML diagnostic span = %q, want invalidLineWithoutColon", got)
	}
	if got, want := mapped[0].Span.Start.Line, 17; got != want {
		t.Fatalf("mapped YAML diagnostic line = %d, want %d", got, want)
	}
	if got, want := mapped[0].Span.Start.Column, 0; got != want {
		t.Fatalf("mapped YAML diagnostic column = %d, want %d", got, want)
	}
}

func TestDiagnosticsMapToOriginalTemplateAndYAMLPositions(t *testing.T) {
	source := loadFixture(t)
	analysis := Analyze(source)

	yamlDiagnostic := findDiagnostic(t, analysis.Diagnostics, "yaml", "line is not a mapping")
	if yamlDiagnostic.InsideTemplate {
		t.Fatalf("YAML diagnostic should be outside template span: %#v", yamlDiagnostic)
	}
	if got := source[yamlDiagnostic.Span.Start.Offset:yamlDiagnostic.Span.End.Offset]; got != "invalidLineWithoutColon" {
		t.Fatalf("YAML diagnostic mapped to %q, want invalidLineWithoutColon", got)
	}
	if got, want := yamlDiagnostic.Span.Start.Line, 17; got != want {
		t.Fatalf("YAML diagnostic line = %d, want %d", got, want)
	}
	if got, want := yamlDiagnostic.Span.Start.Column, 0; got != want {
		t.Fatalf("YAML diagnostic column = %d, want %d", got, want)
	}

	templateDiagnostic := findDiagnostic(t, analysis.Diagnostics, "template", "requires an operand")
	if !templateDiagnostic.InsideTemplate {
		t.Fatalf("template diagnostic should be inside template span: %#v", templateDiagnostic)
	}
	if got := source[templateDiagnostic.Span.Start.Offset:templateDiagnostic.Span.End.Offset]; got != "if" {
		t.Fatalf("template diagnostic span = %q, want if", got)
	}
	if got, want := templateDiagnostic.Span.Start.Line, 19; got != want {
		t.Fatalf("template diagnostic line = %d, want %d", got, want)
	}
	if got, want := templateDiagnostic.Span.Start.Column, 16; got != want {
		t.Fatalf("template diagnostic column = %d, want %d", got, want)
	}

	action, ok := containingAction(analysis.Actions, templateDiagnostic.Span)
	if !ok {
		t.Fatalf("template diagnostic did not map into a recorded action span: %#v", templateDiagnostic)
	}
	if action.Text != "{{ if }}" {
		t.Fatalf("template diagnostic action = %q, want {{ if }}", action.Text)
	}
}

func loadFixture(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile("fixtures/mixed.yaml.tmpl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(data)
}

func findAction(actions []TemplateAction, text string) (TemplateAction, bool) {
	for _, action := range actions {
		if action.Text == text {
			return action, true
		}
	}
	return TemplateAction{}, false
}

func findDiagnostic(t *testing.T, diagnostics []Diagnostic, source, messagePart string) Diagnostic {
	t.Helper()

	for _, diagnostic := range diagnostics {
		if diagnostic.Source == source && strings.Contains(diagnostic.Message, messagePart) {
			return diagnostic
		}
	}
	t.Fatalf("missing %s diagnostic containing %q in %#v", source, messagePart, diagnostics)
	return Diagnostic{}
}

func containingAction(actions []TemplateAction, span Span) (TemplateAction, bool) {
	for _, action := range actions {
		if span.Start.Offset >= action.Span.Start.Offset && span.End.Offset <= action.Span.End.Offset {
			return action, true
		}
	}
	return TemplateAction{}, false
}
