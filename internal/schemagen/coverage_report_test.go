package schemagen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/io41/vibe-xpls/internal/analyzer"
)

func TestCoverageBucketsSeparateUpstreamCompatAndMissingFields(t *testing.T) {
	out := t.TempDir()
	releaseDir := filepath.Join(out, "schemas", "v1.20.7")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	schema := schemaDocumentJSON{
		Release:    "v1.20.7",
		APIVersion: "apiextensions.crossplane.io/v1",
		Kind:       "Composition",
		Fields: []fieldDocJSON{
			{Path: "spec.present", Description: "upstream", Type: "string"},
			{Path: "spec.compositeTypeRef.kind", Description: "Kind of the composite resource type this Composition renders.", Type: "string"},
			{Path: "spec.compatOnly", Description: "compat field", Type: "string"},
		},
		Provenance: schemaProvenanceJSON{
			Source:             string(analyzer.SchemaSourceGeneratedBuiltIn),
			UpstreamReleaseTag: "v1.20.7",
			UpstreamSourcePath: "crds/widgets.yaml",
		},
	}
	if err := writeJSON(filepath.Join(releaseDir, "apiextensions.crossplane.io_v1_Composition.json"), schema); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	targets := []coverageTarget{
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.present", Description: "upstream", Type: "string"},
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.compositeTypeRef.kind", Description: "Kind of the type.", Type: "string"},
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.missing", Description: "lost", Type: "string"},
	}

	actual, err := collectActualCoverageFields(out)
	if err != nil {
		t.Fatalf("collectActualCoverageFields: %v", err)
	}
	state := computeCoverageState(targets, actual, coverageBaseline{FormatVersion: coverageFormatVersion})

	assertCoverageFieldBucket(t, state, "spec.present", bucketCoveredUpstream)
	assertCoverageFieldBucket(t, state, "spec.compositeTypeRef.kind", bucketCoveredWithCompatOverride)
	assertCoverageFieldBucket(t, state, "spec.compatOnly", bucketCompatAddedField)
	assertCoverageFieldBucket(t, state, "spec.missing", bucketMissing)
}

func assertCoverageFieldBucket(t *testing.T, state coverageState, path string, bucket coverageBucket) {
	t.Helper()
	for _, gvk := range state.GVKs {
		for _, field := range gvk.Fields {
			if field.Path == path {
				if field.Bucket != bucket {
					t.Fatalf("%s bucket = %s, want %s", path, field.Bucket, bucket)
				}
				return
			}
		}
	}
	t.Fatalf("missing coverage field %s", path)
}
