package analyzer

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
)

//go:embed schemadata/manifest.json schemadata/schemas/*/*.json
var generatedSchemaFS embed.FS

const schemaBundleFormatVersion = 1

type SchemaBundleStatus struct {
	OK      bool
	Message string
}

type schemaBundleManifest struct {
	BundleFormatVersion int                   `json:"bundleFormatVersion"`
	GeneratorVersion    string                `json:"generatorVersion"`
	Releases            []schemaBundleRelease `json:"releases"`
}

type schemaBundleRelease struct {
	Tag     string   `json:"tag"`
	Commit  string   `json:"commit"`
	Schemas []string `json:"schemas"`
}

type schemaDocumentJSON struct {
	Release    string           `json:"release"`
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Fields     []FieldDoc       `json:"fields"`
	Provenance SchemaProvenance `json:"provenance"`
}

func (idx *SchemaIndex) LoadGeneratedBuiltIns() SchemaBundleStatus {
	return idx.loadGeneratedBuiltInsFromFS(generatedSchemaFS)
}

func (idx *SchemaIndex) loadGeneratedBuiltInsFromFS(fsys fs.FS) SchemaBundleStatus {
	raw, err := fs.ReadFile(fsys, "schemadata/manifest.json")
	if err != nil {
		return SchemaBundleStatus{Message: fmt.Sprintf("read schema manifest: %v", err)}
	}
	var manifest schemaBundleManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return SchemaBundleStatus{Message: fmt.Sprintf("parse schema manifest: %v", err)}
	}
	if manifest.BundleFormatVersion != schemaBundleFormatVersion {
		return SchemaBundleStatus{Message: fmt.Sprintf("unsupported schema bundle format %d", manifest.BundleFormatVersion)}
	}
	for _, release := range manifest.Releases {
		for _, relPath := range release.Schemas {
			if err := idx.loadSchemaDocumentJSON(fsys, filepath.ToSlash(filepath.Join("schemadata", relPath))); err != nil {
				return SchemaBundleStatus{Message: err.Error()}
			}
		}
	}
	return SchemaBundleStatus{OK: true}
}

func (idx *SchemaIndex) loadSchemaDocumentJSON(fsys fs.FS, path string) error {
	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return fmt.Errorf("read schema document %s: %w", path, err)
	}
	var doc schemaDocumentJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse schema document %s: %w", path, err)
	}
	fields := make(map[string]FieldDoc, len(doc.Fields))
	for _, field := range doc.Fields {
		fields[field.Path] = field
	}
	idx.AddGeneratedBuiltIn(Schema{
		Release:    CrossplaneRelease{Tag: doc.Release},
		GVK:        SourceGVK{APIVersion: doc.APIVersion, Kind: doc.Kind},
		Fields:     fields,
		Provenance: doc.Provenance,
	})
	return nil
}
