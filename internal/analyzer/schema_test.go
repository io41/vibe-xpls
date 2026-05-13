package analyzer

import "testing"

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
