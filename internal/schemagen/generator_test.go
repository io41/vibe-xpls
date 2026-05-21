package schemagen

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestGenerateRejectsReleaseTagPathEscape(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "out")
	cfg := fixtureConfig()
	cfg.Releases[0].Tag = "../../escape"

	err := Generate(cfg, out)
	if err == nil {
		t.Fatal("Generate succeeded with path traversal release tag")
	}
	if _, statErr := os.Stat(filepath.Join(base, "escape")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("outside output path stat err = %v, want not exist", statErr)
	}
}

func TestGenerateRejectsCRDDerivedFilenamePathEscape(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "out")
	crdDir := writeCRDDir(t, `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.io
  names:
    kind: ../../../../escape
  scope: Namespaced
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	err := Generate(cfg, out)
	if err == nil {
		t.Fatal("Generate succeeded with path traversal CRD kind")
	}
	if _, statErr := os.Stat(filepath.Join(base, "escape.json")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("outside output path stat err = %v, want not exist", statErr)
	}
}

func TestLoadConfigFileRejectsCWDRelativeFallback(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "bundleFormatVersion": 1,
  "releases": [
    {
      "tag": "v1.20.7",
      "commit": "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
      "rawCRDDir": "internal/schemagen/testdata",
      "crossplaneGoMod": "internal/schemagen/testdata/go.mod"
    }
  ]
}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := LoadConfigFile(configPath); err == nil {
		t.Fatal("LoadConfigFile succeeded with missing config-relative paths")
	}
}

func TestLoadConfigFileResolvesFixturePathsRelativeToConfig(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	configPath := filepath.Join("internal", "schemagen", "testdata", "config.json")
	cfg, err := LoadConfigFile(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.Releases) != 1 {
		t.Fatalf("release count = %d, want 1", len(cfg.Releases))
	}
	fixtureDir := filepath.Clean(filepath.Join("internal", "schemagen", "testdata"))
	rawCRDDir := filepath.Clean(cfg.Releases[0].RawCRDDir)
	crossplaneGoMod := filepath.Clean(cfg.Releases[0].CrossplaneGoMod)
	if rel, err := filepath.Rel(fixtureDir, rawCRDDir); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("RawCRDDir = %q, want under %q", cfg.Releases[0].RawCRDDir, fixtureDir)
	}
	if rel, err := filepath.Rel(fixtureDir, crossplaneGoMod); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("CrossplaneGoMod = %q, want under %q", cfg.Releases[0].CrossplaneGoMod, fixtureDir)
	}
	if err := Generate(cfg, t.TempDir()); err != nil {
		t.Fatalf("generate from loaded config: %v", err)
	}
}

func TestGenerateFailsForMissingLocalRef(t *testing.T) {
	out := t.TempDir()
	crdDir := writeCRDDir(t, `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.io
  names:
    kind: Widget
  scope: Namespaced
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                broken:
                  $ref: "#/definitions/DoesNotExist"
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	if err := Generate(cfg, out); err == nil {
		t.Fatal("Generate succeeded with missing local ref")
	}
}

func TestGenerateFailsForCyclicLocalRef(t *testing.T) {
	out := t.TempDir()
	crdDir := writeCRDDir(t, `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.io
  names:
    kind: Widget
  scope: Namespaced
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          definitions:
            A:
              $ref: "#/definitions/B"
            B:
              $ref: "#/definitions/A"
          properties:
            spec:
              type: object
              properties:
                loop:
                  $ref: "#/definitions/A"
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	if err := Generate(cfg, out); err == nil {
		t.Fatal("Generate succeeded with cyclic local ref")
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	cfg := fixtureConfig()
	out1 := t.TempDir()
	out2 := t.TempDir()
	if err := Generate(cfg, out1); err != nil {
		t.Fatalf("generate first output: %v", err)
	}
	if err := Generate(cfg, out2); err != nil {
		t.Fatalf("generate second output: %v", err)
	}

	files1 := readTree(t, out1)
	files2 := readTree(t, out2)
	if len(files1) != len(files2) {
		t.Fatalf("file count = %d, want %d", len(files2), len(files1))
	}
	for path, want := range files1 {
		if got, ok := files2[path]; !ok {
			t.Fatalf("second output missing %s", path)
		} else if got != want {
			t.Fatalf("content mismatch for %s", path)
		}
	}
}

func TestGenerateSkipsNonCRDMultiDocumentYAML(t *testing.T) {
	out := t.TempDir()
	crdDir := writeCRDDir(t, `apiVersion: v1
kind: ConfigMap
metadata:
  name: ignored
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.io
  names:
    kind: Widget
  scope: Namespaced
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	if err := Generate(cfg, out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "schemas", "v1.20.7", "example.io_v1_Widget.json")); err != nil {
		t.Fatalf("generated CRD schema stat: %v", err)
	}
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

func fixtureConfig() Config {
	return Config{
		BundleFormatVersion: 1,
		Releases: []ReleaseConfig{{
			Tag:             "v1.20.7",
			Commit:          "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
			RawCRDDir:       filepath.Join("testdata"),
			CrossplaneGoMod: filepath.Join("testdata", "go.mod"),
		}},
	}
}

func writeCRDDir(t *testing.T, yamlDoc string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "crd.yaml"), []byte(yamlDoc), 0o644); err != nil {
		t.Fatalf("write CRD: %v", err)
	}
	return dir
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = strings.TrimSpace(string(raw))
		return nil
	}); err != nil {
		t.Fatalf("read output tree: %v", err)
	}
	return files
}
