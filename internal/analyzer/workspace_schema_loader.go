package analyzer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

type workspaceCRDDocument struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       struct {
		Group string `yaml:"group"`
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Name   string `yaml:"name"`
			Served bool   `yaml:"served"`
			Schema struct {
				OpenAPIV3Schema workspaceOpenAPISchema `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type workspaceOpenAPISchema struct {
	Type        string                            `yaml:"type"`
	Description string                            `yaml:"description"`
	Properties  map[string]workspaceOpenAPISchema `yaml:"properties"`
	Required    []string                          `yaml:"required"`
	Enum        []any                             `yaml:"enum"`
	Items       *workspaceOpenAPISchema           `yaml:"items"`
}

func (a *Analyzer) loadWorkspaceSchemas() {
	for _, pkg := range a.workspace.PackageRoots {
		for _, schema := range workspaceSchemasForPackage(pkg, a.limits) {
			a.schemas.AddWorkspaceSchema(schema)
		}
	}
}

func workspaceSchemasForPackage(pkg PackageRoot, limits Limits) []Schema {
	if pkg.Root == "" {
		return nil
	}
	var schemas []Schema
	for _, path := range workspaceYAMLFiles(pkg.Root, limits) {
		schemas = append(schemas, workspaceSchemasFromFile(path)...)
	}
	return schemas
}

func workspaceYAMLFiles(root string, limits Limits) []string {
	var files []string
	var totalBytes int64
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".worktrees", "node_modules", "vendor":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		if limits.MaxYAMLFiles > 0 && len(files) >= limits.MaxYAMLFiles {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if limits.MaxYAMLBytes > 0 && totalBytes+info.Size() > limits.MaxYAMLBytes {
			return nil
		}
		totalBytes += info.Size()
		files = append(files, path)
		return nil
	})
	sort.Strings(files)
	return files
}

func workspaceSchemasFromFile(path string) []Schema {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	hash := sha256.Sum256(raw)
	provenance := SchemaProvenance{
		Path:           path,
		Owner:          SchemaOwnerProvider,
		Source:         SchemaSourceWorkspace,
		UpstreamSHA256: hex.EncodeToString(hash[:]),
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var schemas []Schema
	for {
		var doc workspaceCRDDocument
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return schemas
		}
		if doc.APIVersion != "apiextensions.k8s.io/v1" || doc.Kind != "CustomResourceDefinition" {
			continue
		}
		if doc.Spec.Group == "" || doc.Spec.Names.Kind == "" {
			continue
		}
		for _, version := range doc.Spec.Versions {
			if !version.Served || version.Name == "" {
				continue
			}
			fields := collectWorkspaceFields(version.Schema.OpenAPIV3Schema)
			schemas = append(schemas, Schema{
				GVK: SourceGVK{
					APIVersion: doc.Spec.Group + "/" + version.Name,
					Kind:       doc.Spec.Names.Kind,
				},
				Fields:     fields,
				Provenance: provenance,
			})
		}
	}
	return schemas
}

func collectWorkspaceFields(schema workspaceOpenAPISchema) map[string]FieldDoc {
	fields := map[string]FieldDoc{}
	collectWorkspaceObjectFields(fields, "", schema)
	return fields
}

func collectWorkspaceObjectFields(fields map[string]FieldDoc, parentPath string, schema workspaceOpenAPISchema) {
	required := workspaceRequiredSet(schema.Required)
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child := schema.Properties[name]
		path := workspaceJoinPath(parentPath, name)
		collectWorkspaceField(fields, path, child, required[name])
	}
}

func collectWorkspaceField(fields map[string]FieldDoc, path string, schema workspaceOpenAPISchema, required bool) {
	if path == "" {
		return
	}
	fieldPath := path
	if schema.Type == "array" {
		fieldPath += "[]"
	}
	fields[fieldPath] = FieldDoc{
		Path:        fieldPath,
		Description: schema.Description,
		Type:        schema.Type,
		Required:    required,
		Enum:        workspaceEnumValues(schema.Enum),
	}
	if schema.Type == "array" && schema.Items != nil {
		collectWorkspaceObjectFields(fields, fieldPath, *schema.Items)
		return
	}
	collectWorkspaceObjectFields(fields, path, schema)
}

func workspaceRequiredSet(required []string) map[string]bool {
	set := make(map[string]bool, len(required))
	for _, name := range required {
		set[name] = true
	}
	return set
}

func workspaceJoinPath(parentPath, name string) string {
	if parentPath == "" {
		return name
	}
	return parentPath + "." + name
}

func workspaceEnumValues(values []any) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, fmt.Sprint(value))
	}
	return result
}
