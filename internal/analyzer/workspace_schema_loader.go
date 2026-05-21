package analyzer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

type workspaceSchemaKey struct {
	PackageRoot string
	GVK         SourceGVK
}

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
	a.workspaceSchemas = map[workspaceSchemaKey]Schema{}
	a.workspaceSchemaDiagnostics = map[string][]Diagnostic{}
	for _, pkg := range a.workspace.PackageRoots {
		a.refreshWorkspaceSchemasForPackage(pkg)
	}
}

func (a *Analyzer) refreshWorkspaceSchemasForURI(uri string) {
	path, ok := filePathFromURI(uri)
	if !ok || !isWorkspaceYAMLFile(path) {
		return
	}
	pkg, ok := a.workspace.PackageForFile(path)
	if !ok {
		return
	}
	a.refreshWorkspaceSchemasForPackage(pkg)
}

func (a *Analyzer) refreshWorkspaceSchemasForPackage(pkg PackageRoot) {
	if a.workspaceSchemas == nil {
		a.workspaceSchemas = map[workspaceSchemaKey]Schema{}
	}
	if a.workspaceSchemaDiagnostics == nil {
		a.workspaceSchemaDiagnostics = map[string][]Diagnostic{}
	}
	for key := range a.workspaceSchemas {
		if key.PackageRoot == pkg.Root {
			delete(a.workspaceSchemas, key)
		}
	}
	delete(a.workspaceSchemaDiagnostics, pkg.Root)
	for _, path := range a.workspaceSchemaSourcePaths(pkg) {
		raw, ok := a.workspaceSchemaSource(path)
		if !ok {
			continue
		}
		schemas, diagnostics := workspaceSchemasFromRaw(path, raw)
		a.workspaceSchemaDiagnostics[pkg.Root] = append(a.workspaceSchemaDiagnostics[pkg.Root], diagnostics...)
		for _, schema := range schemas {
			a.addWorkspaceSchemaForPackage(pkg, schema)
		}
	}
}

func (a *Analyzer) workspaceSchemaSourcePaths(pkg PackageRoot) []string {
	paths := map[string]struct{}{}
	for _, path := range workspaceYAMLFiles(pkg.Root, a.limits) {
		if a.packageOwnsSchemaSourcePath(pkg, path) {
			paths[path] = struct{}{}
		}
	}
	for _, path := range a.docs.FilePaths() {
		if !isWorkspaceYAMLFile(path) {
			continue
		}
		if a.packageOwnsSchemaSourcePath(pkg, path) {
			paths[path] = struct{}{}
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func (a *Analyzer) packageOwnsSchemaSourcePath(pkg PackageRoot, path string) bool {
	owner, ok := a.workspace.PackageForFile(path)
	return ok && owner.Root == pkg.Root && workspaceSchemaSourcePathAllowed(pkg.Root, path)
}

func (a *Analyzer) workspaceSchemaSource(path string) ([]byte, bool) {
	if doc, ok := a.docs.GetByFilePath(path); ok {
		return []byte(doc.Text), true
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return raw, true
}

func (a *Analyzer) addWorkspaceSchemaForPackage(pkg PackageRoot, schema Schema) {
	if _, ok := a.schemas.builtIns[schema.GVK]; ok {
		a.workspaceSchemaDiagnostics[pkg.Root] = append(a.workspaceSchemaDiagnostics[pkg.Root], Diagnostic{
			URI:      fileURIFromPath(schema.Provenance.Path),
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema duplicates built-in Crossplane core schema",
		})
		return
	}
	key := workspaceSchemaKey{PackageRoot: pkg.Root, GVK: schema.GVK}
	if _, ok := a.workspaceSchemas[key]; ok {
		a.workspaceSchemaDiagnostics[pkg.Root] = append(a.workspaceSchemaDiagnostics[pkg.Root], Diagnostic{
			URI:      fileURIFromPath(schema.Provenance.Path),
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema conflicts with another workspace schema",
		})
	}
	a.workspaceSchemas[key] = copySchema(schema)
}

func (a *Analyzer) workspaceSchemaForURI(uri string, gvk SourceGVK) (Schema, bool) {
	path, ok := filePathFromURI(uri)
	if !ok {
		return Schema{}, false
	}
	pkg, ok := a.workspace.PackageForFile(path)
	if !ok {
		return Schema{}, false
	}
	schema, ok := a.workspaceSchemas[workspaceSchemaKey{PackageRoot: pkg.Root, GVK: gvk}]
	if !ok {
		return Schema{}, false
	}
	return copySchema(schema), true
}

func (a *Analyzer) workspaceSchemaDiagnosticsForURI(uri string) []Diagnostic {
	path, ok := filePathFromURI(uri)
	if !ok {
		return nil
	}
	pkg, ok := a.workspace.PackageForFile(path)
	if !ok {
		return nil
	}
	diagnostics := a.workspaceSchemaDiagnostics[pkg.Root]
	if len(diagnostics) == 0 {
		return nil
	}
	result := make([]Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnosticPath, ok := filePathFromURI(diagnostic.URI)
		if !ok || filepath.Clean(diagnosticPath) != filepath.Clean(path) {
			continue
		}
		result = append(result, diagnostic)
	}
	return result
}

func fieldsFromSchema(schema Schema) []FieldDoc {
	fields := make([]FieldDoc, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		fields = append(fields, copyFieldDoc(field))
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})
	return fields
}

func workspaceYAMLFiles(root string, limits Limits) []string {
	var files []string
	var totalBytes int64
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && workspaceDirHasPackageMarker(path) {
				return filepath.SkipDir
			}
			switch entry.Name() {
			case ".git", ".worktrees", "node_modules", "vendor":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !isWorkspaceYAMLFile(path) {
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

func workspaceSchemaSourcePathAllowed(root, path string) bool {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	current := cleanRoot
	for _, part := range parts[:len(parts)-1] {
		if workspaceIgnoredSchemaDir(part) {
			return false
		}
		current = filepath.Join(current, part)
		if workspaceDirHasPackageMarker(current) {
			return false
		}
	}
	return true
}

func workspaceIgnoredSchemaDir(name string) bool {
	switch name {
	case ".git", ".worktrees", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func workspaceDirHasPackageMarker(path string) bool {
	for marker := range markerPriority {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return false
}

func isWorkspaceYAMLFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

func workspaceSchemasFromFile(path string) []Schema {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	schemas, _ := workspaceSchemasFromRaw(path, raw)
	return schemas
}

func workspaceSchemasFromRaw(path string, raw []byte) ([]Schema, []Diagnostic) {
	hash := sha256.Sum256(raw)
	provenance := SchemaProvenance{
		Path:           path,
		Owner:          SchemaOwnerProvider,
		Source:         SchemaSourceWorkspace,
		UpstreamSHA256: hex.EncodeToString(hash[:]),
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var schemas []Schema
	var diagnostics []Diagnostic
	for {
		var doc workspaceCRDDocument
		err := decoder.Decode(&doc)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				URI:      fileURIFromPath(path),
				Source:   "schema",
				Severity: "error",
				Message:  fmt.Sprintf("parse workspace schema source %s: %v", path, err),
			})
			return schemas, diagnostics
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
	return schemas, diagnostics
}

func fileURIFromPath(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Clean(path))}).String()
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
