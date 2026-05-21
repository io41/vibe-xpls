package analyzer

import (
	"encoding/json"
	"testing"
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
