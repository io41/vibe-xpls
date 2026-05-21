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
	idx.bundleStatus = idx.LoadGeneratedBuiltIns()
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
