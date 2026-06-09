package schemagen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func GenerateCoverage(cfg Config, outDir string) error {
	baseline, err := loadCoverageBaseline(filepath.Join(outDir, "coverage", "baseline.json"))
	if err != nil {
		return err
	}
	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		return fmt.Errorf("collect coverage targets: %w", err)
	}
	actual, err := collectActualCoverageFields(outDir)
	if err != nil {
		return fmt.Errorf("collect actual coverage fields: %w", err)
	}
	state := computeCoverageState(targets, actual, baseline)

	rawJSON, err := renderCoverageJSON(state)
	if err != nil {
		return err
	}
	if err := writeFileUnder(outDir, "coverage/coverage.json", rawJSON); err != nil {
		return err
	}
	if err := writeFileUnder(outDir, "coverage/coverage.md", []byte(renderCoverageMarkdown(state))); err != nil {
		return err
	}
	return nil
}

func CheckCoverage(cfg Config, outDir string) error {
	tmp, err := os.MkdirTemp("", "vibe-xpls-schema-coverage-*")
	if err != nil {
		return fmt.Errorf("create temporary output dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	if err := Generate(cfg, tmp); err != nil {
		return fmt.Errorf("generate schemas for coverage check: %w", err)
	}
	if err := copyFile(
		filepath.Join(outDir, "coverage", "baseline.json"),
		filepath.Join(tmp, "coverage", "baseline.json"),
	); err != nil {
		return err
	}
	if err := GenerateCoverage(cfg, tmp); err != nil {
		return fmt.Errorf("generate coverage for check: %w", err)
	}

	comparisons := []struct {
		relPath string
		label   string
	}{
		{relPath: "manifest.json", label: "manifest.json"},
		{relPath: "schemas", label: "schemas"},
		{relPath: "coverage/coverage.json", label: "coverage/coverage.json"},
		{relPath: "coverage/coverage.md", label: "coverage/coverage.md"},
	}
	for _, comparison := range comparisons {
		relPath := filepath.FromSlash(comparison.relPath)
		if err := compareGeneratedPath(
			filepath.Join(outDir, relPath),
			filepath.Join(tmp, relPath),
			comparison.label,
		); err != nil {
			return err
		}
	}

	baseline, err := loadCoverageBaseline(filepath.Join(outDir, "coverage", "baseline.json"))
	if err != nil {
		return err
	}
	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		return fmt.Errorf("collect coverage targets: %w", err)
	}
	actual, err := collectActualCoverageFields(tmp)
	if err != nil {
		return fmt.Errorf("collect actual coverage fields: %w", err)
	}
	state := computeCoverageState(targets, actual, baseline)
	if problems := validateCoverageRatchet(state, baseline); len(problems) > 0 {
		return errors.New(formatCoverageProblems(problems))
	}
	return nil
}

func writeFileUnder(outDir, relPath string, raw []byte) error {
	path, err := safeOutputPath(outDir, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func compareGeneratedPath(wantPath, gotPath, label string) error {
	if _, err := os.Stat(wantPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s is missing", label)
		}
		return fmt.Errorf("stat path %s: %w", wantPath, err)
	}
	wantFiles, err := filesByRelativePathForCheck(wantPath)
	if err != nil {
		return err
	}
	gotFiles, err := filesByRelativePathForCheck(gotPath)
	if err != nil {
		return err
	}
	for _, relPath := range sortedFileKeys(wantFiles) {
		want := wantFiles[relPath]
		got, ok := gotFiles[relPath]
		if !ok {
			if relPath == "." {
				return fmt.Errorf("%s is missing", label)
			}
			return fmt.Errorf("%s is missing generated file %s", label, relPath)
		}
		if !bytes.Equal(want, got) {
			if relPath == "." {
				return fmt.Errorf("%s is stale", label)
			}
			return fmt.Errorf("%s/%s is stale", label, relPath)
		}
	}
	for _, relPath := range sortedFileKeys(gotFiles) {
		if _, ok := wantFiles[relPath]; ok {
			continue
		}
		if relPath == "." {
			return fmt.Errorf("%s is unexpected", label)
		}
		return fmt.Errorf("%s has extra generated file %s", label, relPath)
	}
	return nil
}

func filesByRelativePathForCheck(root string) (map[string][]byte, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat path %s: %w", root, err)
	}
	if !info.IsDir() {
		raw, err := os.ReadFile(root)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", root, err)
		}
		return map[string][]byte{".": raw}, nil
	}
	files := map[string][]byte{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relPath)] = raw
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk path %s: %w", root, err)
	}
	return files, nil
}

func sortedFileKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
