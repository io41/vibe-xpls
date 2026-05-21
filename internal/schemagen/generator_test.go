package schemagen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFixtureCRD(t *testing.T) {
	out := t.TempDir()
	cfg := Config{
		BundleFormatVersion: 1,
		Releases: []ReleaseConfig{{
			Tag:             "v1.20.7",
			Commit:          "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
			RawCRDDir:       filepath.Join("testdata"),
			CrossplaneGoMod: filepath.Join("testdata", "go.mod"),
		}},
	}
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "schemas", "v1.20.7", "apiextensions.crossplane.io_v1_Composition.json"))
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}
	var doc struct {
		Fields []struct {
			Path     string `json:"path"`
			Required bool   `json:"required,omitempty"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse generated schema: %v", err)
	}
	assertGeneratedPath(t, doc.Fields, "metadata.name", false)
	assertGeneratedPath(t, doc.Fields, "metadata.labels", false)
	assertGeneratedPath(t, doc.Fields, "metadata.annotations", false)
	assertGeneratedPath(t, doc.Fields, "spec.compositeTypeRef.apiVersion", true)
	assertGeneratedPath(t, doc.Fields, "spec.compositeTypeRef.kind", true)
}

func assertGeneratedPath(t *testing.T, fields []struct {
	Path     string `json:"path"`
	Required bool   `json:"required,omitempty"`
}, path string, required bool) {
	t.Helper()
	for _, field := range fields {
		if field.Path == path {
			if field.Required != required {
				t.Fatalf("%s required = %v, want %v", path, field.Required, required)
			}
			return
		}
	}
	t.Fatalf("missing generated path %s", path)
}
