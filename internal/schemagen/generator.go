package schemagen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/io41/vibe-xpls/internal/analyzer"
	"go.yaml.in/yaml/v4"
)

const generatorVersion = "fixture"

type crdDocument struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       struct {
		Group string `yaml:"group"`
		Scope string `yaml:"scope"`
		Names struct {
			Kind string `yaml:"kind"`
		} `yaml:"names"`
		Versions []struct {
			Name   string `yaml:"name"`
			Served bool   `yaml:"served"`
			Schema struct {
				OpenAPIV3Schema openAPISchema `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

type openAPISchema struct {
	Ref                        string                   `yaml:"$ref"`
	Type                       string                   `yaml:"type"`
	Description                string                   `yaml:"description"`
	Properties                 map[string]openAPISchema `yaml:"properties"`
	Required                   []string                 `yaml:"required"`
	Items                      *openAPISchema           `yaml:"items"`
	Default                    any                      `yaml:"default"`
	Enum                       []any                    `yaml:"enum"`
	AdditionalProperties       any                      `yaml:"additionalProperties"`
	Definitions                map[string]openAPISchema `yaml:"definitions"`
	Defs                       map[string]openAPISchema `yaml:"$defs"`
	XKubernetesPreserveUnknown bool                     `yaml:"x-kubernetes-preserve-unknown-fields"`
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
	Release    string               `json:"release"`
	APIVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Fields     []fieldDocJSON       `json:"fields"`
	Provenance schemaProvenanceJSON `json:"provenance"`
}

type fieldDocJSON struct {
	Path        string           `json:"path"`
	Description string           `json:"description,omitempty"`
	Type        string           `json:"type,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Default     *json.RawMessage `json:"default,omitempty"`
	Enum        []string         `json:"enum,omitempty"`
	Deprecated  string           `json:"deprecated,omitempty"`
}

type schemaProvenanceJSON struct {
	Path               string `json:"path,omitempty"`
	Owner              string `json:"owner,omitempty"`
	Source             string `json:"source,omitempty"`
	UpstreamReleaseTag string `json:"upstreamReleaseTag,omitempty"`
	UpstreamSourcePath string `json:"upstreamSourcePath,omitempty"`
	UpstreamSHA256     string `json:"upstreamSHA256,omitempty"`
}

func Generate(cfg Config, outDir string) error {
	manifest := schemaBundleManifest{
		BundleFormatVersion: cfg.BundleFormatVersion,
		GeneratorVersion:    generatorVersion,
		Releases:            make([]schemaBundleRelease, 0, len(cfg.Releases)),
	}
	for _, release := range cfg.Releases {
		schemas, err := generateRelease(release, outDir)
		if err != nil {
			return err
		}
		sort.Strings(schemas)
		manifest.Releases = append(manifest.Releases, schemaBundleRelease{
			Tag:     release.Tag,
			Commit:  release.Commit,
			Schemas: schemas,
		})
	}
	sort.SliceStable(manifest.Releases, func(i, j int) bool {
		return manifest.Releases[i].Tag < manifest.Releases[j].Tag
	})
	return writeJSONUnder(outDir, "manifest.json", manifest)
}

func generateRelease(release ReleaseConfig, outDir string) ([]string, error) {
	safeReleaseTag, err := sanitizePathComponent(release.Tag)
	if err != nil {
		return nil, fmt.Errorf("release tag %q: %w", release.Tag, err)
	}
	crdFiles, err := yamlFiles(release.RawCRDDir)
	if err != nil {
		return nil, err
	}
	schemaPaths := []string{}
	for _, path := range crdFiles {
		docs, sha, err := readCRDDocuments(path)
		if err != nil {
			return nil, err
		}
		for _, doc := range docs {
			if doc.APIVersion != "apiextensions.k8s.io/v1" || doc.Kind != "CustomResourceDefinition" {
				continue
			}
			for _, version := range doc.Spec.Versions {
				if !version.Served || version.Schema.OpenAPIV3Schema.isZero() {
					continue
				}
				apiVersion := doc.Spec.Group + "/" + version.Name
				fields, err := collectFields(version.Schema.OpenAPIV3Schema, doc.Spec.Scope)
				if err != nil {
					return nil, fmt.Errorf("generate %s %s from %s: %w", apiVersion, doc.Spec.Names.Kind, path, err)
				}
				relSourcePath := relativeSourcePath(release.RawCRDDir, path)
				schema := schemaDocumentJSON{
					Release:    release.Tag,
					APIVersion: apiVersion,
					Kind:       doc.Spec.Names.Kind,
					Fields:     toFieldDocJSON(fields),
					Provenance: schemaProvenanceJSON{
						Owner:              string(analyzer.SchemaOwnerCore),
						Source:             string(analyzer.SchemaSourceGeneratedBuiltIn),
						UpstreamReleaseTag: release.Tag,
						UpstreamSourcePath: relSourcePath,
						UpstreamSHA256:     sha,
					},
				}
				applyCompatibilityFieldDocs(&schema)
				filename, err := schemaFilename(apiVersion, doc.Spec.Names.Kind)
				if err != nil {
					return nil, fmt.Errorf("schema filename for %s %s: %w", apiVersion, doc.Spec.Names.Kind, err)
				}
				relSchemaPath := filepath.ToSlash(filepath.Join("schemas", safeReleaseTag, filename))
				if err := writeJSONUnder(outDir, relSchemaPath, schema); err != nil {
					return nil, err
				}
				schemaPaths = append(schemaPaths, relSchemaPath)
			}
		}
	}
	compatibilitySchemas, err := generatedCompatibilitySchemas(release)
	if err != nil {
		return nil, err
	}
	for _, schema := range compatibilitySchemas {
		filename, err := schemaFilename(schema.APIVersion, schema.Kind)
		if err != nil {
			return nil, fmt.Errorf("compatibility schema filename for %s %s: %w", schema.APIVersion, schema.Kind, err)
		}
		relSchemaPath := filepath.ToSlash(filepath.Join("schemas", safeReleaseTag, filename))
		if err := writeJSONUnder(outDir, relSchemaPath, schema); err != nil {
			return nil, err
		}
		schemaPaths = append(schemaPaths, relSchemaPath)
	}
	return schemaPaths, nil
}

func applyCompatibilityFieldDocs(schema *schemaDocumentJSON) {
	overrides := compatibilityFieldDocs(schema.APIVersion, schema.Kind)
	if len(overrides) == 0 {
		return
	}
	seen := map[string]struct{}{}
	for i := range schema.Fields {
		seen[schema.Fields[i].Path] = struct{}{}
		override, ok := overrides[schema.Fields[i].Path]
		if !ok {
			continue
		}
		schema.Fields[i].Description = override.Description
		if override.Type != "" {
			schema.Fields[i].Type = override.Type
		}
		if override.Required {
			schema.Fields[i].Required = true
		}
	}
	for path, override := range overrides {
		if _, ok := seen[path]; ok {
			continue
		}
		schema.Fields = append(schema.Fields, fieldDocJSON{
			Path:        override.Path,
			Description: override.Description,
			Type:        override.Type,
			Required:    override.Required,
			Default:     override.Default,
			Enum:        override.Enum,
			Deprecated:  override.Deprecated,
		})
	}
	sort.Slice(schema.Fields, func(i, j int) bool {
		return schema.Fields[i].Path < schema.Fields[j].Path
	})
}

func generatedCompatibilitySchemas(release ReleaseConfig) ([]schemaDocumentJSON, error) {
	fields := []analyzer.FieldDoc{
		{Path: "apiVersion", Description: "API version of the Configuration metadata resource.", Type: "string"},
		{Path: "kind", Description: "Resource kind, normally Configuration.", Type: "string"},
		{Path: "metadata.name", Description: "Name of the Configuration package.", Type: "string"},
		{Path: "spec.dependsOn.provider", Description: "Provider package dependency required by this Configuration."},
	}
	return []schemaDocumentJSON{{
		Release:    release.Tag,
		APIVersion: "meta.pkg.crossplane.io/v1",
		Kind:       "Configuration",
		Fields:     toFieldDocJSON(fields),
		Provenance: schemaProvenanceJSON{
			Owner:              string(analyzer.SchemaOwnerCore),
			Source:             string(analyzer.SchemaSourceGeneratedBuiltIn),
			UpstreamReleaseTag: release.Tag,
			UpstreamSourcePath: "generated/compatibility/meta.pkg.crossplane.io_v1_Configuration.json",
		},
	}}, nil
}

func compatibilityFieldDocs(apiVersion, kind string) map[string]analyzer.FieldDoc {
	if apiVersion != "apiextensions.crossplane.io/v1" || kind != "Composition" {
		return nil
	}
	return map[string]analyzer.FieldDoc{
		"apiVersion":                       {Path: "apiVersion", Description: "API version of the Composition resource.", Type: "string"},
		"kind":                             {Path: "kind", Description: "Resource kind, normally Composition.", Type: "string"},
		"metadata.name":                    {Path: "metadata.name", Description: "Name of the Composition.", Type: "string"},
		"spec.compositeTypeRef.apiVersion": {Path: "spec.compositeTypeRef.apiVersion", Description: "API version of the composite resource type this Composition renders.", Type: "string", Required: true},
		"spec.compositeTypeRef.kind":       {Path: "spec.compositeTypeRef.kind", Description: "Kind of the composite resource type this Composition renders.", Type: "string", Required: true},
	}
}

func yamlFiles(root string) ([]string, error) {
	paths := []string{}
	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk CRD dir %s: %w", root, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func readCRDDocuments(path string) ([]crdDocument, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read CRD %s: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	docs := []crdDocument{}
	for {
		var doc crdDocument
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			return nil, "", fmt.Errorf("parse CRD %s: %w", path, err)
		}
		if doc.APIVersion == "" && doc.Kind == "" {
			continue
		}
		docs = append(docs, doc)
	}
	return docs, hex.EncodeToString(sum[:]), nil
}

func collectFields(root openAPISchema, scope string) ([]analyzer.FieldDoc, error) {
	fields := map[string]analyzer.FieldDoc{}
	addMetadataFields(fields, scope)
	if err := walkProperties(fields, root, root, "", nil); err != nil {
		return nil, err
	}
	out := make([]analyzer.FieldDoc, 0, len(fields))
	for _, field := range fields {
		out = append(out, field)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func walkProperties(fields map[string]analyzer.FieldDoc, root, schema openAPISchema, prefix string, required []string) error {
	schema, err := resolveSchema(root, schema, nil)
	if err != nil {
		return err
	}
	if schema.Type == "array" && schema.Items != nil {
		arrayPath := prefix + "[]"
		putField(fields, arrayPath, schema, isRequired(required, lastPathSegment(prefix)))
		item, err := resolveSchema(root, *schema.Items, nil)
		if err != nil {
			return err
		}
		return walkProperties(fields, root, item, arrayPath, item.Required)
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child, err := resolveSchema(root, schema.Properties[name], nil)
		if err != nil {
			return err
		}
		path := joinPath(prefix, name)
		childRequired := isRequired(schema.Required, name)
		if child.Type == "array" && child.Items != nil {
			arrayPath := path + "[]"
			putField(fields, arrayPath, child, childRequired)
			item, err := resolveSchema(root, *child.Items, nil)
			if err != nil {
				return err
			}
			if err := walkProperties(fields, root, item, arrayPath, item.Required); err != nil {
				return err
			}
			continue
		}
		putField(fields, path, child, childRequired)
		if err := walkProperties(fields, root, child, path, child.Required); err != nil {
			return err
		}
	}
	return nil
}

func addMetadataFields(fields map[string]analyzer.FieldDoc, scope string) {
	putSyntheticField(fields, "metadata.name", "string", "Object name.")
	putSyntheticField(fields, "metadata.labels", "object", "Object labels.")
	putSyntheticField(fields, "metadata.annotations", "object", "Object annotations.")
	if scope == "Namespaced" {
		putSyntheticField(fields, "metadata.namespace", "string", "Object namespace.")
	}
}

func putSyntheticField(fields map[string]analyzer.FieldDoc, path, typ, description string) {
	if _, ok := fields[path]; ok {
		return
	}
	fields[path] = analyzer.FieldDoc{
		Path:        path,
		Description: description,
		Type:        typ,
	}
}

func putField(fields map[string]analyzer.FieldDoc, path string, schema openAPISchema, required bool) {
	if path == "" {
		return
	}
	fields[path] = analyzer.FieldDoc{
		Path:        path,
		Description: schema.Description,
		Type:        schema.Type,
		Required:    required,
		Default:     rawDefault(schema.Default),
		Enum:        enumStrings(schema.Enum),
	}
}

func resolveSchema(root, schema openAPISchema, seen map[string]struct{}) (openAPISchema, error) {
	if schema.Ref == "" {
		return schema, nil
	}
	if seen == nil {
		seen = map[string]struct{}{}
	}
	if _, ok := seen[schema.Ref]; ok {
		return openAPISchema{}, fmt.Errorf("cyclic local ref %s", schema.Ref)
	}
	seen[schema.Ref] = struct{}{}
	resolved, ok := resolveLocalRef(root, schema.Ref)
	if !ok {
		return openAPISchema{}, fmt.Errorf("unresolved local ref %s", schema.Ref)
	}
	resolved, err := resolveSchema(root, resolved, seen)
	if err != nil {
		return openAPISchema{}, err
	}
	return mergeSchemaOverride(resolved, schema), nil
}

func resolveLocalRef(root openAPISchema, ref string) (openAPISchema, bool) {
	if !strings.HasPrefix(ref, "#/") {
		return openAPISchema{}, false
	}
	current := root
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	for i := 0; i < len(parts); i++ {
		part := unescapeJSONPointer(parts[i])
		switch part {
		case "properties":
			i++
			if i >= len(parts) {
				return openAPISchema{}, false
			}
			name := unescapeJSONPointer(parts[i])
			next, ok := current.Properties[name]
			if !ok {
				return openAPISchema{}, false
			}
			current = next
		case "items":
			if current.Items == nil {
				return openAPISchema{}, false
			}
			current = *current.Items
		case "definitions":
			i++
			if i >= len(parts) {
				return openAPISchema{}, false
			}
			name := unescapeJSONPointer(parts[i])
			next, ok := current.Definitions[name]
			if !ok {
				return openAPISchema{}, false
			}
			current = next
		case "$defs":
			i++
			if i >= len(parts) {
				return openAPISchema{}, false
			}
			name := unescapeJSONPointer(parts[i])
			next, ok := current.Defs[name]
			if !ok {
				return openAPISchema{}, false
			}
			current = next
		default:
			return openAPISchema{}, false
		}
	}
	return current, true
}

func mergeSchemaOverride(base, override openAPISchema) openAPISchema {
	base.Ref = ""
	if override.Type != "" {
		base.Type = override.Type
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.Properties != nil {
		base.Properties = override.Properties
	}
	if override.Required != nil {
		base.Required = override.Required
	}
	if override.Items != nil {
		base.Items = override.Items
	}
	if override.Default != nil {
		base.Default = override.Default
	}
	if override.Enum != nil {
		base.Enum = override.Enum
	}
	if override.AdditionalProperties != nil {
		base.AdditionalProperties = override.AdditionalProperties
	}
	if override.Definitions != nil {
		base.Definitions = override.Definitions
	}
	if override.Defs != nil {
		base.Defs = override.Defs
	}
	if override.XKubernetesPreserveUnknown {
		base.XKubernetesPreserveUnknown = override.XKubernetesPreserveUnknown
	}
	return base
}

func rawDefault(value any) *json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	msg := json.RawMessage(raw)
	return &msg
}

func enumStrings(values []any) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok {
			out = append(out, s)
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			out = append(out, fmt.Sprint(value))
			continue
		}
		out = append(out, string(raw))
	}
	return out
}

func toFieldDocJSON(fields []analyzer.FieldDoc) []fieldDocJSON {
	out := make([]fieldDocJSON, 0, len(fields))
	for _, field := range fields {
		out = append(out, fieldDocJSON{
			Path:        field.Path,
			Description: field.Description,
			Type:        field.Type,
			Required:    field.Required,
			Default:     field.Default,
			Enum:        field.Enum,
			Deprecated:  field.Deprecated,
		})
	}
	return out
}

func writeJSONUnder(outDir, relPath string, value any) error {
	path, err := safeOutputPath(outDir, relPath)
	if err != nil {
		return err
	}
	return writeJSON(path, value)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", filepath.Dir(path), err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func safeOutputPath(outDir, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("output path %s is absolute", relPath)
	}
	target := filepath.Join(outDir, filepath.FromSlash(relPath))
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir %s: %w", outDir, err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolve output path %s: %w", target, err)
	}
	rel, err := filepath.Rel(absOut, absTarget)
	if err != nil {
		return "", fmt.Errorf("check output containment for %s: %w", target, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("output path %s escapes output dir %s", target, outDir)
	}
	return target, nil
}

func schemaFilename(apiVersion, kind string) (string, error) {
	apiVersionComponent, err := sanitizePathComponent(strings.ReplaceAll(apiVersion, "/", "_"))
	if err != nil {
		return "", fmt.Errorf("apiVersion %q: %w", apiVersion, err)
	}
	kindComponent, err := sanitizePathComponent(kind)
	if err != nil {
		return "", fmt.Errorf("kind %q: %w", kind, err)
	}
	return apiVersionComponent + "_" + kindComponent + ".json", nil
}

func sanitizePathComponent(component string) (string, error) {
	if component == "" {
		return "", fmt.Errorf("path component is empty")
	}
	if component == "." || component == ".." {
		return "", fmt.Errorf("path component %q is not allowed", component)
	}
	for _, r := range component {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '-':
		default:
			return "", fmt.Errorf("path component %q contains unsafe character %q", component, r)
		}
	}
	return component, nil
}

func relativeSourcePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func isRequired(required []string, name string) bool {
	for _, item := range required {
		if item == name {
			return true
		}
	}
	return false
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func lastPathSegment(path string) string {
	path = strings.TrimSuffix(path, "[]")
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

func unescapeJSONPointer(value string) string {
	value = strings.ReplaceAll(value, "~1", "/")
	return strings.ReplaceAll(value, "~0", "~")
}

func (schema openAPISchema) isZero() bool {
	return schema.Ref == "" &&
		schema.Type == "" &&
		schema.Description == "" &&
		len(schema.Properties) == 0 &&
		len(schema.Required) == 0 &&
		schema.Items == nil &&
		schema.Default == nil &&
		len(schema.Enum) == 0 &&
		schema.AdditionalProperties == nil &&
		len(schema.Definitions) == 0 &&
		len(schema.Defs) == 0 &&
		!schema.XKubernetesPreserveUnknown
}
