package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type actualCoverageKey struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
}

func collectActualCoverageFields(outDir string) (map[actualCoverageKey]actualCoverageField, error) {
	root := filepath.Join(outDir, "schemas")
	actual := map[actualCoverageKey]actualCoverageField{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read schema %s: %w", path, err)
		}
		var doc schemaDocumentJSON
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("parse schema %s: %w", path, err)
		}
		overrides := compatibilityFieldDocs(doc.APIVersion, doc.Kind)
		for _, field := range doc.Fields {
			override, hasOverride := overrides[field.Path]
			actual[actualCoverageKey{Release: doc.Release, APIVersion: doc.APIVersion, Kind: doc.Kind, Path: field.Path}] = actualCoverageField{
				Path:           field.Path,
				Description:    field.Description,
				Type:           field.Type,
				Required:       field.Required,
				Default:        field.Default,
				Enum:           append([]string(nil), field.Enum...),
				Deprecated:     field.Deprecated,
				CompatOverride: hasOverride && override.Description != "" && override.Description == field.Description,
			}
		}
		if strings.Contains(doc.Provenance.UpstreamSourcePath, "generated/compatibility/") {
			for _, field := range doc.Fields {
				key := actualCoverageKey{Release: doc.Release, APIVersion: doc.APIVersion, Kind: doc.Kind, Path: field.Path}
				value := actual[key]
				value.CompatAdded = true
				actual[key] = value
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return actual, nil
}

func sortedActualKeys(actual map[actualCoverageKey]actualCoverageField) []actualCoverageKey {
	keys := make([]actualCoverageKey, 0, len(actual))
	for key := range actual {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.Join([]string{keys[i].Release, keys[i].APIVersion, keys[i].Kind, keys[i].Path}, "\x00") <
			strings.Join([]string{keys[j].Release, keys[j].APIVersion, keys[j].Kind, keys[j].Path}, "\x00")
	})
	return keys
}
