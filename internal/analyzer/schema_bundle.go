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
	staged := NewSchemaIndex()
	loaded := []releaseGVK{}
	for _, release := range manifest.Releases {
		for _, relPath := range release.Schemas {
			schema, err := schemaFromDocumentJSON(fsys, filepath.ToSlash(filepath.Join("schemadata", relPath)))
			if err != nil {
				return SchemaBundleStatus{Message: err.Error()}
			}
			staged.AddGeneratedBuiltIn(schema)
			loaded = append(loaded, releaseGVK{
				Release:    schema.Release,
				APIVersion: schema.GVK.APIVersion,
				Kind:       schema.GVK.Kind,
			})
		}
	}
	for _, key := range loaded {
		idx.AddGeneratedBuiltIn(staged.releaseSchemas[key])
	}
	return SchemaBundleStatus{OK: true}
}

func (idx *SchemaIndex) loadSchemaDocumentJSON(fsys fs.FS, path string) error {
	schema, err := schemaFromDocumentJSON(fsys, path)
	if err != nil {
		return err
	}
	idx.AddGeneratedBuiltIn(schema)
	return nil
}

func schemaFromDocumentJSON(fsys fs.FS, path string) (Schema, error) {
	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return Schema{}, fmt.Errorf("read schema document %s: %w", path, err)
	}
	var doc schemaDocumentJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Schema{}, fmt.Errorf("parse schema document %s: %w", path, err)
	}
	fields := make(map[string]FieldDoc, len(doc.Fields))
	for _, field := range doc.Fields {
		fields[field.Path] = field
	}
	return Schema{
		Release:    CrossplaneRelease{Tag: doc.Release},
		GVK:        SourceGVK{APIVersion: doc.APIVersion, Kind: doc.Kind},
		Fields:     fields,
		Provenance: doc.Provenance,
	}, nil
}
