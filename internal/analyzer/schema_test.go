package analyzer

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestBuiltInCrossplaneSchemas(t *testing.T) {
	idx := NewSchemaIndex()
	idx.LoadBuiltIns()

	doc, ok := idx.FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected built-in Composition field documentation")
	}
	if doc.Description == "" {
		t.Fatal("expected non-empty field documentation")
	}
}

func TestEmbeddedSchemaBundleLoadsFixture(t *testing.T) {
	idx := NewSchemaIndex()
	status := idx.LoadGeneratedBuiltIns()
	if !status.OK {
		t.Fatalf("bundle status = %#v", status)
	}
	doc, ok := idx.FieldDocumentationForRelease(CrossplaneRelease{Tag: "v1.20.7"}, "apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected generated fixture field")
	}
	if doc.Type != "string" || !doc.Required {
		t.Fatalf("field = %#v, want string required field", doc)
	}
}

func TestGeneratedSchemaBundleIsCurrent(t *testing.T) {
	tmp := t.TempDir()
	cmd := exec.Command("go", "run", "../../cmd/vibe-xpls-schema-gen", "--config", "schemadata/config.json", "--out", tmp)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("regenerate bundle: %v\n%s", err, output)
	}
	assertDirectoriesEqual(t, "schemadata/manifest.json", filepath.Join(tmp, "manifest.json"))
	assertDirectoriesEqual(t, "schemadata/schemas", filepath.Join(tmp, "schemas"))
}

func TestInvalidBundleFormatDisablesGeneratedBuiltIns(t *testing.T) {
	fsys := fstest.MapFS{
		"schemadata/manifest.json": {Data: []byte(`{"bundleFormatVersion":99}`)},
	}
	idx := NewSchemaIndex()
	status := idx.loadGeneratedBuiltInsFromFS(fsys)
	if status.OK {
		t.Fatalf("status = %#v, want failed status", status)
	}
	if len(idx.FieldsForRelease(CrossplaneRelease{Tag: "v1.20.7"}, "apiextensions.crossplane.io/v1", "Composition")) != 0 {
		t.Fatal("invalid bundle should not load generated fields")
	}
}

func TestInvalidSchemaDocumentDisablesGeneratedBuiltInsAtomically(t *testing.T) {
	fsys := fstest.MapFS{
		"schemadata/manifest.json": {Data: []byte(`{
			"bundleFormatVersion": 1,
			"generatorVersion": "fixture",
			"releases": [{
				"tag": "v1.20.7",
				"commit": "fixture",
				"schemas": [
					"schemas/v1.20.7/valid.json",
					"schemas/v1.20.7/corrupt.json"
				]
			}]
		}`)},
		"schemadata/schemas/v1.20.7/valid.json": {Data: []byte(`{
			"apiVersion": "apiextensions.crossplane.io/v1",
			"fields": [
				{"description": "Kind of the composite resource type this Composition renders.", "path": "spec.compositeTypeRef.kind", "required": true, "type": "string"}
			],
			"kind": "Composition",
			"provenance": {"owner": "core", "source": "generated-built-in"},
			"release": "v1.20.7"
		}`)},
		"schemadata/schemas/v1.20.7/corrupt.json": {Data: []byte(`{`)},
	}
	idx := NewSchemaIndex()
	status := idx.loadGeneratedBuiltInsFromFS(fsys)
	if status.OK {
		t.Fatalf("status = %#v, want failed status", status)
	}
	if len(idx.FieldsForRelease(CrossplaneRelease{Tag: "v1.20.7"}, "apiextensions.crossplane.io/v1", "Composition")) != 0 {
		t.Fatal("corrupt bundle should not leak release fields")
	}
	if len(idx.Fields("apiextensions.crossplane.io/v1", "Composition")) != 0 {
		t.Fatal("corrupt bundle should not leak legacy fields")
	}
}

func TestProviderSchemaCanBeAdded(t *testing.T) {
	idx := NewSchemaIndex()
	idx.AddWorkspaceSchema(Schema{
		GVK: SourceGVK{APIVersion: "s3.aws.upbound.io/v1beta1", Kind: "Bucket"},
		Fields: map[string]FieldDoc{
			"spec.forProvider.bucketName": {Path: "spec.forProvider.bucketName", Description: "Name of the S3 bucket to create."},
		},
		Provenance: SchemaProvenance{Path: "provider-crd.yaml", Owner: SchemaOwnerProvider},
	})

	doc, ok := idx.FieldDocumentation("s3.aws.upbound.io/v1beta1", "Bucket", "spec.forProvider.bucketName")
	if !ok {
		t.Fatal("expected provider field documentation")
	}
	if doc.Description != "Name of the S3 bucket to create." {
		t.Fatalf("description = %q", doc.Description)
	}
}

func TestCoreDuplicateDoesNotOverrideBuiltIn(t *testing.T) {
	idx := NewSchemaIndex()
	idx.LoadBuiltIns()
	idx.AddWorkspaceSchema(Schema{
		GVK: SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.compositeTypeRef.kind": {Path: "spec.compositeTypeRef.kind", Description: "wrong workspace duplicate"},
		},
		Provenance: SchemaProvenance{Path: "duplicate-composition-crd.yaml", Owner: SchemaOwnerCore},
	})

	doc, ok := idx.FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected built-in field documentation")
	}
	if doc.Description == "wrong workspace duplicate" {
		t.Fatal("workspace duplicate replaced built-in schema")
	}
	if len(idx.Diagnostics()) != 1 {
		t.Fatalf("diagnostics = %d, want 1", len(idx.Diagnostics()))
	}
}

func TestSchemaIndexLooksUpByReleaseAndGVK(t *testing.T) {
	idx := NewSchemaIndex()
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: v1,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.resources[]": {Path: "spec.resources[]", Description: "v1 resources mode"},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})
	idx.AddGeneratedBuiltIn(Schema{
		Release: v2,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.pipeline[]": {Path: "spec.pipeline[]", Description: "v2 pipeline mode"},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})

	if _, ok := idx.FieldDocumentationForRelease(v1, "apiextensions.crossplane.io/v1", "Composition", "spec.resources[]"); !ok {
		t.Fatal("expected v1 resources field")
	}
	if _, ok := idx.FieldDocumentationForRelease(v1, "apiextensions.crossplane.io/v1", "Composition", "spec.pipeline[]"); ok {
		t.Fatal("v1 lookup returned v2-only field")
	}
	if _, ok := idx.FieldDocumentationForRelease(v2, "apiextensions.crossplane.io/v1", "Composition", "spec.pipeline[]"); !ok {
		t.Fatal("expected v2 pipeline field")
	}
}

func TestFieldDocumentationMarkdown(t *testing.T) {
	def := json.RawMessage("5")
	field := FieldDoc{
		Path:        "spec.revisionHistoryLimit",
		Description: "Number of inactive revisions to retain.\n\nUsed by package managers.",
		Type:        "integer",
		Required:    true,
		Default:     &def,
		Enum:        []string{"1", "5"},
		Deprecated:  "Use spec.revisionActivationPolicy instead.",
	}

	got := fieldCompletionDocumentation(field)
	want := "Number of inactive revisions to retain.\n\nUsed by package managers.\n\n_Type: integer_\n_Required_\n_Default: 5_\n_Allowed: 1, 5_\n_Deprecated: Use spec.revisionActivationPolicy instead._"
	if got != want {
		t.Fatalf("documentation = %q, want %q", got, want)
	}
}

func TestSchemaIndexCopiesFieldDocMetadataOnAdd(t *testing.T) {
	idx := NewSchemaIndex()
	release := CrossplaneRelease{Tag: "v2.2.1"}
	def := json.RawMessage(`"original"`)
	enum := []string{"original"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: release,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.mode": {
				Path:    "spec.mode",
				Default: &def,
				Enum:    enum,
			},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})
	enum[0] = "mutated"
	def = json.RawMessage(`"mutated"`)

	doc, ok := idx.FieldDocumentationForRelease(release, "apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected field documentation")
	}
	if doc.Enum[0] != "original" {
		t.Fatalf("enum = %q, want original", doc.Enum[0])
	}
	if got := string(*doc.Default); got != `"original"` {
		t.Fatalf("default = %q, want %q", got, `"original"`)
	}
}

func TestSchemaIndexReturnsCopiedFieldDocMetadata(t *testing.T) {
	idx := NewSchemaIndex()
	release := CrossplaneRelease{Tag: "v2.2.1"}
	def := json.RawMessage(`"original"`)
	idx.AddGeneratedBuiltIn(Schema{
		Release: release,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.mode": {
				Path:    "spec.mode",
				Default: &def,
				Enum:    []string{"original"},
			},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})

	doc, ok := idx.FieldDocumentationForRelease(release, "apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected release field documentation")
	}
	doc.Enum[0] = "mutated"
	*doc.Default = json.RawMessage(`"mutated"`)

	again, ok := idx.FieldDocumentationForRelease(release, "apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected release field documentation")
	}
	if again.Enum[0] != "original" {
		t.Fatalf("release enum = %q, want original", again.Enum[0])
	}
	if got := string(*again.Default); got != `"original"` {
		t.Fatalf("release default = %q, want %q", got, `"original"`)
	}

	fields := idx.FieldsForRelease(release, "apiextensions.crossplane.io/v1", "Composition")
	if len(fields) != 1 {
		t.Fatalf("release fields = %d, want 1", len(fields))
	}
	fields[0].Enum[0] = "mutated again"
	*fields[0].Default = json.RawMessage(`"mutated again"`)

	again, ok = idx.FieldDocumentationForRelease(release, "apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected release field documentation")
	}
	if again.Enum[0] != "original" {
		t.Fatalf("release enum after FieldsForRelease mutation = %q, want original", again.Enum[0])
	}
	if got := string(*again.Default); got != `"original"` {
		t.Fatalf("release default after FieldsForRelease mutation = %q, want %q", got, `"original"`)
	}

	legacyDoc, ok := idx.FieldDocumentation("apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected legacy field documentation")
	}
	legacyDoc.Enum[0] = "legacy mutated"
	*legacyDoc.Default = json.RawMessage(`"legacy mutated"`)

	legacyFields := idx.Fields("apiextensions.crossplane.io/v1", "Composition")
	if len(legacyFields) != 1 {
		t.Fatalf("legacy fields = %d, want 1", len(legacyFields))
	}
	legacyFields[0].Enum[0] = "legacy mutated again"
	*legacyFields[0].Default = json.RawMessage(`"legacy mutated again"`)

	again, ok = idx.FieldDocumentationForRelease(release, "apiextensions.crossplane.io/v1", "Composition", "spec.mode")
	if !ok {
		t.Fatal("expected release field documentation")
	}
	if again.Enum[0] != "original" {
		t.Fatalf("enum after legacy mutation = %q, want original", again.Enum[0])
	}
	if got := string(*again.Default); got != `"original"` {
		t.Fatalf("default after legacy mutation = %q, want %q", got, `"original"`)
	}
}

func TestFieldDocumentationMarkdownOmitsWhitespaceOnlyFacts(t *testing.T) {
	field := FieldDoc{
		Description: "Readable description.",
		Type:        " \t\n",
		Deprecated:  " \n\t",
	}

	got := fieldCompletionDocumentation(field)
	want := "Readable description."
	if got != want {
		t.Fatalf("documentation = %q, want %q", got, want)
	}
}

func assertDirectoriesEqual(t *testing.T, wantPath, gotPath string) {
	t.Helper()
	wantInfo, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("stat want path %s: %v", wantPath, err)
	}
	if !wantInfo.IsDir() {
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("read want file %s: %v", wantPath, err)
		}
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatalf("read got file %s: %v", gotPath, err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("%s is stale", wantPath)
		}
		return
	}
	entries, err := os.ReadDir(wantPath)
	if err != nil {
		t.Fatalf("read want dir %s: %v", wantPath, err)
	}
	for _, entry := range entries {
		assertDirectoriesEqual(t, filepath.Join(wantPath, entry.Name()), filepath.Join(gotPath, entry.Name()))
	}
}
