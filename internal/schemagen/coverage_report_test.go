package schemagen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestCoverageStateJSONUsesReportFieldNames(t *testing.T) {
	state := coverageState{
		GVKs: []coverageGVKState{{
			Release:      "v1.20.7",
			APIVersion:   "example.org/v1",
			Kind:         "Widget",
			SourcePath:   "crds/widgets.yaml",
			SourceSHA256: "abc123",
			Fields: []coverageFieldState{{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.name",
				Bucket:     bucketCoveredUpstream,
				Metadata: coverageMetadataState{
					Description: metadataCovered,
				},
				Gap: &observedGap{
					Release:    "v1.20.7",
					APIVersion: "example.org/v1",
					Kind:       "Widget",
					Path:       "spec.name",
					Category:   gapMissingDescription,
					Reason:     "description differs",
				},
			}},
			Buckets: map[coverageBucket]int{bucketCoveredUpstream: 1},
		}},
		Gaps: []observedGap{{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.name",
			Category:   gapMissingDescription,
			Reason:     "description differs",
		}},
	}

	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal coverage state: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal coverage state: %v", err)
	}

	assertHasJSONKey(t, doc, "gvks")
	assertHasJSONKey(t, doc, "gaps")
	assertMissingJSONKeys(t, doc, "GVKs", "Gaps")

	gvk := objectFromJSONValue(t, arrayFromJSONValue(t, doc["gvks"])[0])
	assertHasJSONKey(t, gvk, "release")
	assertHasJSONKey(t, gvk, "apiVersion")
	assertHasJSONKey(t, gvk, "kind")
	assertHasJSONKey(t, gvk, "sourcePath")
	assertHasJSONKey(t, gvk, "sourceSHA256")
	assertHasJSONKey(t, gvk, "fields")
	assertHasJSONKey(t, gvk, "buckets")
	assertMissingJSONKeys(t, gvk, "Release", "APIVersion", "Kind", "SourcePath", "SourceSHA256", "Fields", "Buckets")

	field := objectFromJSONValue(t, arrayFromJSONValue(t, gvk["fields"])[0])
	assertHasJSONKey(t, field, "release")
	assertHasJSONKey(t, field, "apiVersion")
	assertHasJSONKey(t, field, "kind")
	assertHasJSONKey(t, field, "path")
	assertHasJSONKey(t, field, "bucket")
	assertHasJSONKey(t, field, "metadata")
	assertHasJSONKey(t, field, "gap")
	assertMissingJSONKeys(t, field, "Release", "APIVersion", "Kind", "Path", "Bucket", "Metadata", "Gap")

	metadata := objectFromJSONValue(t, field["metadata"])
	assertHasJSONKey(t, metadata, "description")
	assertMissingJSONKeys(t, metadata, "Description")

	fieldGap := objectFromJSONValue(t, field["gap"])
	assertHasJSONKey(t, fieldGap, "release")
	assertHasJSONKey(t, fieldGap, "apiVersion")
	assertHasJSONKey(t, fieldGap, "kind")
	assertHasJSONKey(t, fieldGap, "path")
	assertHasJSONKey(t, fieldGap, "category")
	assertHasJSONKey(t, fieldGap, "reason")
	assertMissingJSONKeys(t, fieldGap, "Release", "APIVersion", "Kind", "Path", "Category", "Reason")

	gap := objectFromJSONValue(t, arrayFromJSONValue(t, doc["gaps"])[0])
	assertHasJSONKey(t, gap, "release")
	assertHasJSONKey(t, gap, "apiVersion")
	assertHasJSONKey(t, gap, "kind")
	assertHasJSONKey(t, gap, "path")
	assertHasJSONKey(t, gap, "category")
	assertHasJSONKey(t, gap, "reason")
	assertMissingJSONKeys(t, gap, "Release", "APIVersion", "Kind", "Path", "Category", "Reason")
}

func TestRenderCoverageJSONIsDeterministic(t *testing.T) {
	state := coverageRenderTestState()

	first, err := renderCoverageJSON(state)
	if err != nil {
		t.Fatalf("render coverage JSON: %v", err)
	}
	second, err := renderCoverageJSON(state)
	if err != nil {
		t.Fatalf("render coverage JSON again: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("coverage JSON differs between renders:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	want := `{
  "formatVersion": 1,
  "releases": [
    {
      "tag": "v1.20.7",
      "totals": {
        "upstreamGVKs": 1,
        "generatedGVKs": 1,
        "targetFields": 2,
        "coveredUpstreamFields": 1,
        "knownGaps": 1
      },
      "gvks": [
        {
          "apiVersion": "example.org/v1",
          "kind": "Widget",
          "sourcePath": "crds/widgets.yaml",
          "sourceSHA256": "abc123",
          "buckets": {
            "covered-upstream": 1,
            "missing": 1
          },
          "fields": [
            {
              "path": "spec.missing",
              "bucket": "missing",
              "metadata": {}
            },
            {
              "path": "spec.present",
              "bucket": "covered-upstream",
              "metadata": {
                "description": "covered",
                "type": "covered",
                "required": "not-required",
                "enum": "not-present-upstream",
                "default": "not-present-upstream",
                "deprecated": "not-present-upstream"
              }
            }
          ]
        }
      ]
    }
  ]
}
`
	if string(first) != want {
		t.Fatalf("coverage JSON mismatch:\ngot:\n%s\nwant:\n%s", first, want)
	}
}

func TestRenderCoverageMarkdownSummarizesWorstGVKs(t *testing.T) {
	got := renderCoverageMarkdown(coverageRenderTestState())
	for _, want := range []string{
		"# Schema Coverage",
		"## Release v1.20.7",
		"Upstream field coverage: 1/2 (50.00%)",
		"Known gaps: 1",
		"### Worst-Covered GVKs",
		"| example.org/v1 | Widget | 50.00% | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("coverage markdown missing %q:\n%s", want, got)
		}
	}
}

func TestCompatibilityOverrideDetectedFromDescriptionTypeOrRequired(t *testing.T) {
	tests := []struct {
		name     string
		field    fieldDocJSON
		override analyzer.FieldDoc
		want     bool
	}{
		{
			name:     "description",
			field:    fieldDocJSON{Description: "compat description"},
			override: analyzer.FieldDoc{Description: "compat description"},
			want:     true,
		},
		{
			name:     "type",
			field:    fieldDocJSON{Type: "object"},
			override: analyzer.FieldDoc{Type: "object"},
			want:     true,
		},
		{
			name:     "required",
			field:    fieldDocJSON{Required: true},
			override: analyzer.FieldDoc{Required: true},
			want:     true,
		},
		{
			name:     "not reflected",
			field:    fieldDocJSON{Description: "upstream", Type: "string"},
			override: analyzer.FieldDoc{Description: "compat", Type: "object", Required: true},
			want:     false,
		},
		{
			name:     "empty override metadata",
			field:    fieldDocJSON{Description: "upstream", Type: "string", Required: true},
			override: analyzer.FieldDoc{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldHasCompatibilityOverride(tt.field, tt.override)
			if got != tt.want {
				t.Fatalf("fieldHasCompatibilityOverride() = %t, want %t", got, tt.want)
			}
		})
	}
}

func coverageRenderTestState() coverageState {
	return coverageState{
		GVKs: []coverageGVKState{{
			Release:      "v1.20.7",
			APIVersion:   "example.org/v1",
			Kind:         "Widget",
			SourcePath:   "crds/widgets.yaml",
			SourceSHA256: "abc123",
			Fields: []coverageFieldState{
				{
					Release:    "v1.20.7",
					APIVersion: "example.org/v1",
					Kind:       "Widget",
					Path:       "spec.present",
					Bucket:     bucketCoveredUpstream,
					Metadata: coverageMetadataState{
						Description: metadataCovered,
						Type:        metadataCovered,
						Required:    metadataNotRequired,
						Enum:        metadataNotPresentUpstream,
						Default:     metadataNotPresentUpstream,
						Deprecated:  metadataNotPresentUpstream,
					},
				},
				{
					Release:    "v1.20.7",
					APIVersion: "example.org/v1",
					Kind:       "Widget",
					Path:       "spec.missing",
					Bucket:     bucketMissing,
				},
			},
			Buckets: map[coverageBucket]int{
				bucketMissing:         1,
				bucketCoveredUpstream: 1,
			},
		}},
	}
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

func assertHasJSONKey(t *testing.T, object map[string]any, key string) {
	t.Helper()
	if _, ok := object[key]; !ok {
		t.Fatalf("missing JSON key %q in %#v", key, object)
	}
}

func assertMissingJSONKeys(t *testing.T, object map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := object[key]; ok {
			t.Fatalf("unexpected JSON key %q in %#v", key, object)
		}
	}
}

func arrayFromJSONValue(t *testing.T, value any) []any {
	t.Helper()
	array, ok := value.([]any)
	if !ok {
		t.Fatalf("JSON value %#v is %T, want array", value, value)
	}
	if len(array) == 0 {
		t.Fatalf("JSON array is empty")
	}
	return array
}

func objectFromJSONValue(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("JSON value %#v is %T, want object", value, value)
	}
	return object
}
