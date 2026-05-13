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
