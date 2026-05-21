package analyzer

import (
	"encoding/json"
	"sort"
)

type SourceGVK struct {
	APIVersion string
	Kind       string
}

type CrossplaneRelease struct {
	Tag string
}

type SchemaOwner string

const (
	SchemaOwnerCore     SchemaOwner = "core"
	SchemaOwnerProvider SchemaOwner = "provider"
	SchemaOwnerUser     SchemaOwner = "user"
)

type SchemaSource string

const (
	SchemaSourceGeneratedBuiltIn SchemaSource = "generated-built-in"
	SchemaSourceWorkspace        SchemaSource = "workspace"
)

type FieldDoc struct {
	Path        string
	Description string
	Type        string
	Required    bool
	Default     *json.RawMessage
	Enum        []string
	Deprecated  string
}

type SchemaProvenance struct {
	Path               string
	Owner              SchemaOwner
	Source             SchemaSource
	UpstreamReleaseTag string
	UpstreamSourcePath string
	UpstreamSHA256     string
}

type Schema struct {
	Release    CrossplaneRelease
	GVK        SourceGVK
	Fields     map[string]FieldDoc
	Provenance SchemaProvenance
}

type SchemaIndex struct {
	schemas        map[SourceGVK]Schema
	releaseSchemas map[releaseGVK]Schema
	builtIns       map[SourceGVK]struct{}
	diagnostics    []Diagnostic
	bundleStatus   SchemaBundleStatus
}

type releaseGVK struct {
	Release    CrossplaneRelease
	APIVersion string
	Kind       string
}

func NewSchemaIndex() *SchemaIndex {
	return &SchemaIndex{
		schemas:        map[SourceGVK]Schema{},
		releaseSchemas: map[releaseGVK]Schema{},
		builtIns:       map[SourceGVK]struct{}{},
	}
}

func (idx *SchemaIndex) LoadBuiltIns() {
	idx.addBuiltInSchema(Schema{
		GVK: SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"apiVersion":                       {Path: "apiVersion", Description: "API version of the Composition resource."},
			"kind":                             {Path: "kind", Description: "Resource kind, normally Composition."},
			"metadata.name":                    {Path: "metadata.name", Description: "Name of the Composition."},
			"spec.compositeTypeRef.apiVersion": {Path: "spec.compositeTypeRef.apiVersion", Description: "API version of the composite resource type this Composition renders."},
			"spec.compositeTypeRef.kind":       {Path: "spec.compositeTypeRef.kind", Description: "Kind of the composite resource type this Composition renders."},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore},
	})
	idx.addBuiltInSchema(Schema{
		GVK: SourceGVK{APIVersion: "meta.pkg.crossplane.io/v1", Kind: "Configuration"},
		Fields: map[string]FieldDoc{
			"apiVersion":              {Path: "apiVersion", Description: "API version of the Configuration metadata resource."},
			"kind":                    {Path: "kind", Description: "Resource kind, normally Configuration."},
			"metadata.name":           {Path: "metadata.name", Description: "Name of the Configuration package."},
			"spec.dependsOn.provider": {Path: "spec.dependsOn.provider", Description: "Provider package dependency required by this Configuration."},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore},
	})
}

func (idx *SchemaIndex) AddWorkspaceSchema(schema Schema) {
	if _, ok := idx.builtIns[schema.GVK]; ok {
		idx.diagnostics = append(idx.diagnostics, Diagnostic{
			URI:      schema.Provenance.Path,
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema duplicates built-in Crossplane core schema",
		})
		return
	}
	if _, ok := idx.schemas[schema.GVK]; ok {
		idx.diagnostics = append(idx.diagnostics, Diagnostic{
			URI:      schema.Provenance.Path,
			Source:   "schema",
			Severity: "warning",
			Message:  "workspace schema conflicts with another workspace schema",
		})
	}
	idx.schemas[schema.GVK] = copySchema(schema)
}

func (idx *SchemaIndex) AddGeneratedBuiltIn(schema Schema) {
	idx.releaseSchemas[releaseGVK{
		Release:    schema.Release,
		APIVersion: schema.GVK.APIVersion,
		Kind:       schema.GVK.Kind,
	}] = copySchema(schema)
	idx.builtIns[schema.GVK] = struct{}{}
	if _, ok := idx.schemas[schema.GVK]; !ok {
		idx.schemas[schema.GVK] = copySchema(schema)
	}
}

func (idx *SchemaIndex) FieldDocumentation(apiVersion, kind, fieldPath string) (FieldDoc, bool) {
	schema, ok := idx.schemas[SourceGVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return FieldDoc{}, false
	}
	doc, ok := schema.Fields[fieldPath]
	return copyFieldDoc(doc), ok
}

func (idx *SchemaIndex) FieldDocumentationForRelease(release CrossplaneRelease, apiVersion, kind, fieldPath string) (FieldDoc, bool) {
	schema, ok := idx.releaseSchemas[releaseGVK{Release: release, APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return FieldDoc{}, false
	}
	doc, ok := schema.Fields[fieldPath]
	return copyFieldDoc(doc), ok
}

func (idx *SchemaIndex) Fields(apiVersion, kind string) []FieldDoc {
	schema, ok := idx.schemas[SourceGVK{APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return nil
	}
	fields := make([]FieldDoc, 0, len(schema.Fields))
	for _, doc := range schema.Fields {
		fields = append(fields, copyFieldDoc(doc))
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})
	return fields
}

func (idx *SchemaIndex) FieldsForRelease(release CrossplaneRelease, apiVersion, kind string) []FieldDoc {
	schema, ok := idx.releaseSchemas[releaseGVK{Release: release, APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return nil
	}
	fields := make([]FieldDoc, 0, len(schema.Fields))
	for _, doc := range schema.Fields {
		fields = append(fields, copyFieldDoc(doc))
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})
	return fields
}

func (idx *SchemaIndex) Diagnostics() []Diagnostic {
	diagnostics := make([]Diagnostic, len(idx.diagnostics))
	copy(diagnostics, idx.diagnostics)
	return diagnostics
}

func (idx *SchemaIndex) addBuiltInSchema(schema Schema) {
	idx.AddGeneratedBuiltIn(schema)
}

func copySchema(schema Schema) Schema {
	fields := make(map[string]FieldDoc, len(schema.Fields))
	for path, doc := range schema.Fields {
		fields[path] = copyFieldDoc(doc)
	}
	schema.Fields = fields
	return schema
}

func copyFieldDoc(doc FieldDoc) FieldDoc {
	if doc.Default != nil {
		raw := append(json.RawMessage(nil), (*doc.Default)...)
		doc.Default = &raw
	}
	if doc.Enum != nil {
		doc.Enum = append([]string(nil), doc.Enum...)
	}
	return doc
}
