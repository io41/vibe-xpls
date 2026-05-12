package schemaindex

import "testing"

func TestLoadDirIndexesKindsFromAllFixtureSources(t *testing.T) {
	idx := loadTestIndex(t)

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		source     SourceType
	}{
		{
			name:       "xrd declared composite resource",
			apiVersion: "platform.example.org/v1alpha1",
			kind:       "CompositeBucket",
			source:     SourceXRD,
		},
		{
			name:       "composition manifest",
			apiVersion: "apiextensions.crossplane.io/v1",
			kind:       "Composition",
			source:     SourceComposition,
		},
		{
			name:       "provider crd declared managed resource",
			apiVersion: "s3.aws.upbound.io/v1beta1",
			kind:       "Bucket",
			source:     SourceProviderCRD,
		},
		{
			name:       "package metadata manifest",
			apiVersion: "meta.pkg.crossplane.io/v1",
			kind:       "Configuration",
			source:     SourcePackageMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, ok := idx.LookupKind(tt.apiVersion, tt.kind)
			if !ok {
				t.Fatalf("expected %s %s to be indexed", tt.apiVersion, tt.kind)
			}
			if resource.Source.Type != tt.source {
				t.Fatalf("source type = %q, want %q", resource.Source.Type, tt.source)
			}
		})
	}
}

func TestCompositionLookupRecordsCompositeTypeRef(t *testing.T) {
	idx := loadTestIndex(t)

	composition, ok := idx.LookupKind("apiextensions.crossplane.io/v1", "Composition")
	if !ok {
		t.Fatal("expected composition fixture to be indexed")
	}
	if composition.CompositeRef == nil {
		t.Fatal("expected composition compositeTypeRef to be recorded")
	}
	if got, want := composition.CompositeRef.APIVersion, "platform.example.org/v1alpha1"; got != want {
		t.Fatalf("compositeTypeRef apiVersion = %q, want %q", got, want)
	}
	if got, want := composition.CompositeRef.Kind, "CompositeBucket"; got != want {
		t.Fatalf("compositeTypeRef kind = %q, want %q", got, want)
	}
}

func TestFieldDocumentationLookup(t *testing.T) {
	idx := loadTestIndex(t)

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		fieldPath  string
		want       string
		source     SourceType
	}{
		{
			name:       "xrd schema field",
			apiVersion: "platform.example.org/v1alpha1",
			kind:       "CompositeBucket",
			fieldPath:  "spec.parameters.region",
			want:       "AWS region where the backing bucket should be created.",
			source:     SourceXRD,
		},
		{
			name:       "provider crd schema field",
			apiVersion: "s3.aws.upbound.io/v1beta1",
			kind:       "Bucket",
			fieldPath:  "spec.forProvider.bucketName",
			want:       "Name of the S3 bucket to create.",
			source:     SourceProviderCRD,
		},
		{
			name:       "composition directive field",
			apiVersion: "apiextensions.crossplane.io/v1",
			kind:       "Composition",
			fieldPath:  "spec.compositeTypeRef.apiVersion",
			want:       "Composite API version selected by this Composition.",
			source:     SourceComposition,
		},
		{
			name:       "package metadata directive field",
			apiVersion: "meta.pkg.crossplane.io/v1",
			kind:       "Configuration",
			fieldPath:  "spec.dependsOn.provider",
			want:       "Provider package dependency declared by this configuration.",
			source:     SourcePackageMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, ok := idx.FieldDocumentation(tt.apiVersion, tt.kind, tt.fieldPath)
			if !ok {
				t.Fatalf("expected field doc for %s %s %s", tt.apiVersion, tt.kind, tt.fieldPath)
			}
			if doc.Description != tt.want {
				t.Fatalf("description = %q, want %q", doc.Description, tt.want)
			}
			if doc.Source.Type != tt.source {
				t.Fatalf("source type = %q, want %q", doc.Source.Type, tt.source)
			}
		})
	}
}

func TestPackageMetadataDependencyLookup(t *testing.T) {
	idx := loadTestIndex(t)

	meta, ok := idx.PackageMetadata("platform-buckets")
	if !ok {
		t.Fatal("expected package metadata to be indexed")
	}
	if got := len(meta.Dependencies); got != 1 {
		t.Fatalf("dependency count = %d, want 1", got)
	}
	dependency := meta.Dependencies[0]
	if got, want := dependency.Provider, "xpkg.upbound.io/upbound/provider-aws-s3"; got != want {
		t.Fatalf("provider = %q, want %q", got, want)
	}
	if got, want := dependency.Version, "v1.15.0"; got != want {
		t.Fatalf("version = %q, want %q", got, want)
	}
}

func TestMissingLookupsReturnFalse(t *testing.T) {
	idx := loadTestIndex(t)

	if _, ok := idx.LookupKind("example.org/v1", "Missing"); ok {
		t.Fatal("expected missing apiVersion/kind lookup to return false")
	}
	if _, ok := idx.FieldDocumentation("platform.example.org/v1alpha1", "CompositeBucket", "spec.missing"); ok {
		t.Fatal("expected missing field documentation lookup to return false")
	}
}

func loadTestIndex(t *testing.T) *Index {
	t.Helper()

	idx, err := LoadDir("testdata")
	if err != nil {
		t.Fatalf("load test index: %v", err)
	}
	return idx
}
