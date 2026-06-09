package schemagen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCoverageWritesArtifactsWithoutChangingBaseline(t *testing.T) {
	cfg := fixtureConfig()
	out := t.TempDir()
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	baselinePath := writeEmptyCoverageBaseline(t, out)
	before, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline before coverage generation: %v", err)
	}

	if err := GenerateCoverage(cfg, out); err != nil {
		t.Fatalf("GenerateCoverage: %v", err)
	}

	after, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline after coverage generation: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("baseline changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if raw, err := os.ReadFile(filepath.Join(out, "coverage", "coverage.json")); err != nil {
		t.Fatalf("read coverage JSON: %v", err)
	} else if !strings.Contains(string(raw), `"formatVersion": 1`) {
		t.Fatalf("coverage JSON missing format version:\n%s", raw)
	}
	if raw, err := os.ReadFile(filepath.Join(out, "coverage", "coverage.md")); err != nil {
		t.Fatalf("read coverage markdown: %v", err)
	} else if !strings.Contains(string(raw), "# Schema Coverage") {
		t.Fatalf("coverage markdown missing title:\n%s", raw)
	}
}

func TestCheckCoverageFailsForStaleCoverageArtifact(t *testing.T) {
	cfg := fixtureConfig()
	out := t.TempDir()
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	writeEmptyCoverageBaseline(t, out)
	if err := GenerateCoverage(cfg, out); err != nil {
		t.Fatalf("GenerateCoverage: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "coverage", "coverage.md"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale coverage markdown: %v", err)
	}

	err := CheckCoverage(cfg, out)
	if err == nil {
		t.Fatal("CheckCoverage succeeded with stale coverage artifact")
	}
	if got, want := err.Error(), "coverage/coverage.md is stale"; got != want {
		t.Fatalf("CheckCoverage error = %q, want %q", got, want)
	}
}

func TestCompareGeneratedPathReportsMissingSingleFileWithLabel(t *testing.T) {
	root := t.TempDir()
	got := filepath.Join(root, "tmp", "coverage", "coverage.json")
	writeTestFile(t, got, []byte("{}\n"))

	err := compareGeneratedPath(filepath.Join(root, "committed", "coverage", "coverage.json"), got, "coverage/coverage.json")
	if err == nil {
		t.Fatal("compareGeneratedPath succeeded with missing committed single file")
	}
	if got, want := err.Error(), "coverage/coverage.json is missing"; got != want {
		t.Fatalf("compareGeneratedPath error = %q, want %q", got, want)
	}
}

func TestCompareGeneratedPathReportsMissingDirectoryWithLabel(t *testing.T) {
	root := t.TempDir()
	got := filepath.Join(root, "tmp", "schemas")
	writeTestFile(t, filepath.Join(got, "schema.json"), []byte("{}\n"))

	err := compareGeneratedPath(filepath.Join(root, "committed", "schemas"), got, "schemas")
	if err == nil {
		t.Fatal("compareGeneratedPath succeeded with missing committed directory")
	}
	if got, want := err.Error(), "schemas is missing"; got != want {
		t.Fatalf("compareGeneratedPath error = %q, want %q", got, want)
	}
}

func TestCompareGeneratedPathReportsExtraGeneratedFileWithLabel(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "committed", "schemas")
	got := filepath.Join(root, "tmp", "schemas")
	writeTestFile(t, filepath.Join(want, "schema.json"), []byte("{}\n"))
	writeTestFile(t, filepath.Join(got, "schema.json"), []byte("{}\n"))
	writeTestFile(t, filepath.Join(got, "extra.json"), []byte("{}\n"))

	err := compareGeneratedPath(want, got, "schemas")
	if err == nil {
		t.Fatal("compareGeneratedPath succeeded with extra generated file")
	}
	if got, want := err.Error(), "schemas has extra generated file extra.json"; got != want {
		t.Fatalf("compareGeneratedPath error = %q, want %q", got, want)
	}
}

func writeEmptyCoverageBaseline(t *testing.T, outDir string) string {
	t.Helper()
	path := filepath.Join(outDir, "coverage", "baseline.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create coverage dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\n  \"formatVersion\": 1,\n  \"entries\": []\n}\n"), 0o644); err != nil {
		t.Fatalf("write empty coverage baseline: %v", err)
	}
	return path
}

func writeTestFile(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test file dir: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
