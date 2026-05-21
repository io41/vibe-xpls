package analyzer

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
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

func (idx *SchemaIndex) ReleasesForGVK(gvk SourceGVK) []CrossplaneRelease {
	seen := map[CrossplaneRelease]struct{}{}
	releases := []CrossplaneRelease{}
	for key := range idx.releaseSchemas {
		if key.APIVersion != gvk.APIVersion || key.Kind != gvk.Kind {
			continue
		}
		if _, ok := seen[key.Release]; ok {
			continue
		}
		seen[key.Release] = struct{}{}
		releases = append(releases, key.Release)
	}
	sortCrossplaneReleases(releases)
	return releases
}

func (idx *SchemaIndex) DefaultReleaseForGVK(gvk SourceGVK) (CrossplaneRelease, bool) {
	releases := idx.ReleasesForGVK(gvk)
	if len(releases) == 0 {
		return CrossplaneRelease{}, false
	}
	return releases[len(releases)-1], true
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

type schemaSemVer struct {
	major int
	minor int
	patch int
	pre   string
	ok    bool
}

func sortCrossplaneReleases(releases []CrossplaneRelease) {
	sort.Slice(releases, func(i, j int) bool {
		return compareCrossplaneReleases(releases[i], releases[j]) < 0
	})
}

func compareCrossplaneReleases(left, right CrossplaneRelease) int {
	leftVersion := parseSchemaSemVer(left.Tag)
	rightVersion := parseSchemaSemVer(right.Tag)
	if leftVersion.ok && rightVersion.ok {
		return compareSchemaSemVer(leftVersion, rightVersion)
	}
	if leftVersion.ok != rightVersion.ok {
		if leftVersion.ok {
			return 1
		}
		return -1
	}
	return strings.Compare(left.Tag, right.Tag)
}

func parseSchemaSemVer(tag string) schemaSemVer {
	tag = strings.TrimSpace(tag)
	tag = strings.TrimPrefix(tag, "v")
	pre := ""
	if split := strings.Index(tag, "-"); split >= 0 {
		pre = tag[split+1:]
		tag = tag[:split]
	}
	parts := strings.Split(tag, ".")
	if len(parts) != 3 {
		return schemaSemVer{}
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return schemaSemVer{}
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return schemaSemVer{}
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return schemaSemVer{}
	}
	return schemaSemVer{major: major, minor: minor, patch: patch, pre: pre, ok: true}
}

func compareSchemaSemVer(left, right schemaSemVer) int {
	if left.major != right.major {
		return compareInts(left.major, right.major)
	}
	if left.minor != right.minor {
		return compareInts(left.minor, right.minor)
	}
	if left.patch != right.patch {
		return compareInts(left.patch, right.patch)
	}
	if left.pre == right.pre {
		return 0
	}
	if left.pre == "" {
		return 1
	}
	if right.pre == "" {
		return -1
	}
	return strings.Compare(left.pre, right.pre)
}

func compareInts(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
