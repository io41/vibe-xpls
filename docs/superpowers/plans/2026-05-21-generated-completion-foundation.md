# Generated Completion Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hand-written built-in Crossplane completion fields with a generated, release-aware schema bundle that supplies key completions and normalized docs for pinned Crossplane core releases.

**Architecture:** Add a generated schema pipeline with three separate boundaries: generator, embedded bundle loader, and analyzer runtime lookup. The analyzer uses a release-aware schema index keyed by `(CrossplaneRelease, exact GVK)`, resolves a release per request, and then reuses the existing completion, hover, text edit, and workspace activation flows.

**Tech Stack:** Go 1.26.3, `embed`, `encoding/json`, `io/fs`, `go.yaml.in/yaml/v4`, existing analyzer/LSP packages, committed upstream Crossplane CRD YAML artifacts.

---

## Scope Notes

The design spec is [2026-05-21-generated-completion-foundation-design.md](../specs/2026-05-21-generated-completion-foundation-design.md). This plan implements the whole slice but keeps it in reviewable commits.

Current upstream pins checked with `git ls-remote` on 2026-05-21:

- Crossplane `v1.20.7`, peeled commit `5fae6c1ab967e57b1dc792b5c52c97bceda12953`.
- Crossplane `v2.2.1`, peeled commit `713541df7fc5cf0946b6573837831086465a2dbe`.

Do not use the `v2.3.0-rc.*` or `v2.4.0-rc.*` tags for this slice. "Latest v2" means latest stable v2 release.

## File Structure

- `internal/analyzer/schema.go`: keep public schema index surface, expand it to release-aware lookup, schema metadata, and generated built-in loading.
- `internal/analyzer/schema_docs.go`: render normalized completion and hover Markdown from schema field metadata.
- `internal/analyzer/schema_bundle.go`: load and validate embedded generated schema bundle artifacts.
- `internal/analyzer/schema_resolver.go`: resolve package/document release selection.
- `internal/analyzer/schema_test.go`: release-aware schema index, docs rendering, bundle load, and duplicate-diagnostic tests.
- `internal/analyzer/completion.go`: use release-aware schema lookup, path normalization, `sortText`, status suppression, and suppression reasons.
- `internal/analyzer/hover.go`: use release-aware schema lookup and normalized schema docs.
- `internal/analyzer/analyzer.go`: initialize generated built-ins and expose bundle health to LSP.
- `internal/lsp/server.go`: emit `sortText`, preserve existing completion presentation, log suppression reasons, and show one fatal bundle warning at initialize.
- `internal/lsp/server_test.go`: wire-level completion metadata, warning, and degradation tests.
- `cmd/vibe-xpls-schema-gen/main.go`: documented generator entry point.
- `internal/schemagen/config.go`: generator config model.
- `internal/schemagen/generator.go`: CRD OpenAPI normalization to canonical schema documents.
- `internal/schemagen/generator_test.go`: fixture-based generator tests.
- `internal/schemagen/testdata/`: small CRD fixtures for generator unit tests.
- `internal/analyzer/schemadata/config.json`: pinned release/generator config.
- `internal/analyzer/schemadata/upstream/`: committed upstream Crossplane CRD YAML and `go.mod` source artifacts.
- `internal/analyzer/schemadata/manifest.json`: generated bundle manifest.
- `internal/analyzer/schemadata/schemas/<release>/*.json`: generated canonical schema documents.
- `docs/generated-schemas.md`: maintainer command for regenerating schema artifacts and updating pinned releases.

---

### Task 1: Release-Aware Schema Model And Documentation Renderer

**Files:**
- Modify: `internal/analyzer/schema.go`
- Create: `internal/analyzer/schema_docs.go`
- Modify: `internal/analyzer/schema_test.go`
- Modify: `internal/analyzer/hover.go`

- [ ] **Step 1: Add failing tests for release-aware schema lookup and docs rendering**

Add these tests to `internal/analyzer/schema_test.go`:

```go
func TestSchemaIndexLooksUpByReleaseAndGVK(t *testing.T) {
	idx := NewSchemaIndex()
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: v1,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.resources[]": {Path: "spec.resources[]", Description: "v1 resources mode"},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})
	idx.AddGeneratedBuiltIn(Schema{
		Release: v2,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"},
		Fields: map[string]FieldDoc{
			"spec.pipeline[]": {Path: "spec.pipeline[]", Description: "v2 pipeline mode"},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})

	if _, ok := idx.FieldDocumentationForRelease(v1, "apiextensions.crossplane.io/v1", "Composition", "spec.resources[]"); !ok {
		t.Fatal("expected v1 resources field")
	}
	if _, ok := idx.FieldDocumentationForRelease(v1, "apiextensions.crossplane.io/v1", "Composition", "spec.pipeline[]"); ok {
		t.Fatal("v1 lookup returned v2-only field")
	}
	if _, ok := idx.FieldDocumentationForRelease(v2, "apiextensions.crossplane.io/v1", "Composition", "spec.pipeline[]"); !ok {
		t.Fatal("expected v2 pipeline field")
	}
}

func TestFieldDocumentationMarkdown(t *testing.T) {
	def := json.RawMessage("5")
	field := FieldDoc{
		Path:        "spec.revisionHistoryLimit",
		Description: "Number of inactive revisions to retain.\n\nUsed by package managers.",
		Type:        "integer",
		Required:    true,
		Default:     &def,
		Enum:        []string{"1", "5"},
		Deprecated:  "Use spec.revisionActivationPolicy instead.",
	}

	got := fieldCompletionDocumentation(field)
	want := "Number of inactive revisions to retain.\n\nUsed by package managers.\n\n_Type: integer_\n_Required_\n_Default: 5_\n_Allowed: 1, 5_\n_Deprecated: Use spec.revisionActivationPolicy instead._"
	if got != want {
		t.Fatalf("documentation = %q, want %q", got, want)
	}
}
```

Add `encoding/json` to the test imports.

- [ ] **Step 2: Run the focused tests and confirm they fail**

Run:

```bash
go test ./internal/analyzer -run 'TestSchemaIndexLooksUpByReleaseAndGVK|TestFieldDocumentationMarkdown'
```

Expected: fail because `CrossplaneRelease`, release-aware lookup, and doc rendering do not exist yet.

- [ ] **Step 3: Expand schema model types**

Update `internal/analyzer/schema.go` with these concrete type additions while keeping existing methods compiling:

```go
type CrossplaneRelease struct {
	Tag string
}

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
	Release   CrossplaneRelease
	GVK       SourceGVK
	Fields    map[string]FieldDoc
	Provenance SchemaProvenance
}
```

Add `encoding/json` to `schema.go`.

Replace `SchemaIndex.schemas map[SourceGVK]Schema` with both maps:

```go
schemas        map[SourceGVK]Schema
releaseSchemas map[releaseGVK]Schema
```

Add:

```go
type releaseGVK struct {
	Release    CrossplaneRelease
	APIVersion string
	Kind       string
}
```

- [ ] **Step 4: Implement release-aware index methods**

Add these methods to `internal/analyzer/schema.go`:

```go
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

func (idx *SchemaIndex) FieldDocumentationForRelease(release CrossplaneRelease, apiVersion, kind, fieldPath string) (FieldDoc, bool) {
	schema, ok := idx.releaseSchemas[releaseGVK{Release: release, APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return FieldDoc{}, false
	}
	doc, ok := schema.Fields[fieldPath]
	return doc, ok
}

func (idx *SchemaIndex) FieldsForRelease(release CrossplaneRelease, apiVersion, kind string) []FieldDoc {
	schema, ok := idx.releaseSchemas[releaseGVK{Release: release, APIVersion: apiVersion, Kind: kind}]
	if !ok {
		return nil
	}
	fields := make([]FieldDoc, 0, len(schema.Fields))
	for _, doc := range schema.Fields {
		fields = append(fields, doc)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Path < fields[j].Path
	})
	return fields
}
```

Keep existing `FieldDocumentation` and `Fields` as compatibility wrappers for workspace and old tests.

- [ ] **Step 5: Implement documentation renderer**

Create `internal/analyzer/schema_docs.go`:

```go
package analyzer

import (
	"bytes"
	"encoding/json"
	"strings"
)

func fieldCompletionDocumentation(field FieldDoc) string {
	var sections []string
	if desc := strings.TrimSpace(field.Description); desc != "" {
		sections = append(sections, desc)
	}
	var facts []string
	if field.Type != "" {
		facts = append(facts, "_Type: "+field.Type+"_")
	}
	if field.Required {
		facts = append(facts, "_Required_")
	}
	if field.Default != nil {
		facts = append(facts, "_Default: "+compactJSON(*field.Default)+"_")
	}
	if len(field.Enum) != 0 {
		facts = append(facts, "_Allowed: "+strings.Join(field.Enum, ", ")+"_")
	}
	if field.Deprecated != "" {
		facts = append(facts, "_Deprecated: "+strings.TrimSpace(field.Deprecated)+"_")
	}
	if len(facts) != 0 {
		sections = append(sections, strings.Join(facts, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func compactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err == nil {
		return buf.String()
	}
	return strings.TrimSpace(string(raw))
}
```

Update `hoverFromField` in `internal/analyzer/hover.go` so it uses the shared body:

```go
func hoverFromField(field FieldDoc) Hover {
	body := fieldCompletionDocumentation(field)
	if body == "" {
		return Hover{Markdown: "**" + hoverTitle(field.Path) + "**"}
	}
	return Hover{Markdown: fmt.Sprintf("**%s**\n\n%s", hoverTitle(field.Path), body)}
}
```

- [ ] **Step 6: Run focused analyzer tests**

Run:

```bash
go test ./internal/analyzer -run 'TestSchemaIndexLooksUpByReleaseAndGVK|TestFieldDocumentationMarkdown|TestBuiltInCrossplaneSchemas'
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/analyzer/schema.go internal/analyzer/schema_docs.go internal/analyzer/schema_test.go internal/analyzer/hover.go
git commit -m "feat: add release-aware schema model"
```

---

### Task 2: Embedded Bundle Loader And Fixture Artifact

**Files:**
- Create: `internal/analyzer/schema_bundle.go`
- Create: `internal/analyzer/schemadata/manifest.json`
- Create: `internal/analyzer/schemadata/schemas/v1.20.7/apiextensions.crossplane.io_v1_Composition.json`
- Create: `internal/analyzer/schemadata/schemas/v1.20.7/meta.pkg.crossplane.io_v1_Configuration.json`
- Modify: `internal/analyzer/schema_test.go`
- Modify: `internal/analyzer/analyzer.go`

- [ ] **Step 1: Add fixture manifest and schema artifact**

Create `internal/analyzer/schemadata/manifest.json`:

```json
{
  "bundleFormatVersion": 1,
  "generatorVersion": "fixture",
  "releases": [
    {
      "tag": "v1.20.7",
      "commit": "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
      "schemas": [
        "schemas/v1.20.7/apiextensions.crossplane.io_v1_Composition.json",
        "schemas/v1.20.7/meta.pkg.crossplane.io_v1_Configuration.json"
      ]
    }
  ]
}
```

Create `internal/analyzer/schemadata/schemas/v1.20.7/apiextensions.crossplane.io_v1_Composition.json`:

```json
{
  "apiVersion": "apiextensions.crossplane.io/v1",
  "fields": [
    {
      "description": "API version of the Composition resource.",
      "path": "apiVersion",
      "type": "string"
    },
    {
      "description": "Resource kind, normally Composition.",
      "path": "kind",
      "type": "string"
    },
    {
      "description": "Name of the Composition.",
      "path": "metadata.name",
      "type": "string"
    },
    {
      "description": "Kind of the composite resource type this Composition renders.",
      "path": "spec.compositeTypeRef.kind",
      "required": true,
      "type": "string"
    }
  ],
  "kind": "Composition",
  "provenance": {
    "owner": "core",
    "source": "generated-built-in",
    "upstreamReleaseTag": "v1.20.7",
    "upstreamSourcePath": "fixture"
  },
  "release": "v1.20.7"
}
```

Create `internal/analyzer/schemadata/schemas/v1.20.7/meta.pkg.crossplane.io_v1_Configuration.json`:

```json
{
  "apiVersion": "meta.pkg.crossplane.io/v1",
  "fields": [
    {
      "description": "API version of the Configuration metadata resource.",
      "path": "apiVersion",
      "type": "string"
    },
    {
      "description": "Resource kind, normally Configuration.",
      "path": "kind",
      "type": "string"
    },
    {
      "description": "Name of the Configuration package.",
      "path": "metadata.name",
      "type": "string"
    },
    {
      "description": "Provider package dependency required by this Configuration.",
      "path": "spec.dependsOn.provider",
      "type": "string"
    }
  ],
  "kind": "Configuration",
  "provenance": {
    "owner": "core",
    "source": "generated-built-in",
    "upstreamReleaseTag": "v1.20.7",
    "upstreamSourcePath": "fixture"
  },
  "release": "v1.20.7"
}
```

- [ ] **Step 2: Add failing bundle loader tests**

Add to `internal/analyzer/schema_test.go`:

```go
func TestEmbeddedSchemaBundleLoadsFixture(t *testing.T) {
	idx := NewSchemaIndex()
	status := idx.LoadGeneratedBuiltIns()
	if !status.OK {
		t.Fatalf("bundle status = %#v", status)
	}
	doc, ok := idx.FieldDocumentationForRelease(CrossplaneRelease{Tag: "v1.20.7"}, "apiextensions.crossplane.io/v1", "Composition", "spec.compositeTypeRef.kind")
	if !ok {
		t.Fatal("expected generated fixture field")
	}
	if doc.Type != "string" || !doc.Required {
		t.Fatalf("field = %#v, want string required field", doc)
	}
}

func TestInvalidBundleFormatDisablesGeneratedBuiltIns(t *testing.T) {
	fsys := fstest.MapFS{
		"schemadata/manifest.json": {Data: []byte(`{"bundleFormatVersion":99}`)},
	}
	idx := NewSchemaIndex()
	status := idx.loadGeneratedBuiltInsFromFS(fsys)
	if status.OK {
		t.Fatalf("status = %#v, want failed status", status)
	}
	if len(idx.FieldsForRelease(CrossplaneRelease{Tag: "v1.20.7"}, "apiextensions.crossplane.io/v1", "Composition")) != 0 {
		t.Fatal("invalid bundle should not load generated fields")
	}
}
```

Add `testing/fstest` to imports.

- [ ] **Step 3: Run loader tests and confirm they fail**

Run:

```bash
go test ./internal/analyzer -run 'TestEmbeddedSchemaBundleLoadsFixture|TestInvalidBundleFormatDisablesGeneratedBuiltIns'
```

Expected: fail because bundle loading does not exist.

- [ ] **Step 4: Implement bundle loader**

Create `internal/analyzer/schema_bundle.go`:

```go
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
	BundleFormatVersion int                     `json:"bundleFormatVersion"`
	GeneratorVersion    string                  `json:"generatorVersion"`
	Releases            []schemaBundleRelease   `json:"releases"`
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
		Release:   CrossplaneRelease{Tag: doc.Release},
		GVK:       SourceGVK{APIVersion: doc.APIVersion, Kind: doc.Kind},
		Fields:    fields,
		Provenance: doc.Provenance,
	})
	return nil
}
```

- [ ] **Step 5: Switch analyzer startup to generated built-ins**

In `internal/analyzer/analyzer.go`, replace:

```go
schemas := NewSchemaIndex()
schemas.LoadBuiltIns()
```

with:

```go
schemas := NewSchemaIndex()
schemas.bundleStatus = schemas.LoadGeneratedBuiltIns()
```

Add `bundleStatus SchemaBundleStatus` to `SchemaIndex`.

Keep `LoadBuiltIns` in `schema.go` for compatibility until Task 4 removes old hand-written data from runtime tests.

- [ ] **Step 6: Run focused tests**

Run:

```bash
go test ./internal/analyzer
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/analyzer/schema_bundle.go internal/analyzer/schema.go internal/analyzer/analyzer.go internal/analyzer/schema_test.go internal/analyzer/schemadata
git commit -m "feat: load generated schema bundle"
```

---

### Task 3: Schema Generator With Fixture-Driven Normalization

**Files:**
- Create: `cmd/vibe-xpls-schema-gen/main.go`
- Create: `internal/schemagen/config.go`
- Create: `internal/schemagen/generator.go`
- Create: `internal/schemagen/generator_test.go`
- Create: `internal/schemagen/testdata/composition-crd.yaml`
- Create: `internal/schemagen/testdata/config.json`

- [ ] **Step 1: Add fixture CRD and generator config**

Create `internal/schemagen/testdata/composition-crd.yaml` with a minimal CRD:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: compositions.apiextensions.crossplane.io
spec:
  group: apiextensions.crossplane.io
  names:
    kind: Composition
  scope: Cluster
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            apiVersion:
              type: string
              description: API version of the Composition resource.
            kind:
              type: string
              description: Resource kind, normally Composition.
            metadata:
              type: object
            spec:
              type: object
              description: Composition spec.
              required:
                - compositeTypeRef
              properties:
                compositeTypeRef:
                  type: object
                  description: Composite type reference.
                  required:
                    - apiVersion
                    - kind
                  properties:
                    apiVersion:
                      type: string
                      description: API version of the type.
                    kind:
                      type: string
                      description: Kind of the type.
```

Create `internal/schemagen/testdata/config.json`:

```json
{
  "bundleFormatVersion": 1,
  "releases": [
    {
      "tag": "v1.20.7",
      "commit": "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
      "rawCRDDir": "internal/schemagen/testdata",
      "crossplaneGoMod": "internal/schemagen/testdata/go.mod"
    }
  ]
}
```

Create `internal/schemagen/testdata/go.mod`:

```go
module github.com/crossplane/crossplane

require k8s.io/apimachinery v0.33.0
```

- [ ] **Step 2: Add failing generator tests**

Create `internal/schemagen/generator_test.go`:

```go
package schemagen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateFixtureCRD(t *testing.T) {
	out := t.TempDir()
	cfg := Config{
		BundleFormatVersion: 1,
		Releases: []ReleaseConfig{{
			Tag:             "v1.20.7",
			Commit:          "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
			RawCRDDir:       filepath.Join("testdata"),
			CrossplaneGoMod: filepath.Join("testdata", "go.mod"),
		}},
	}
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("generate: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "schemas", "v1.20.7", "apiextensions.crossplane.io_v1_Composition.json"))
	if err != nil {
		t.Fatalf("read generated schema: %v", err)
	}
	var doc struct {
		Fields []struct {
			Path     string `json:"path"`
			Required bool   `json:"required,omitempty"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse generated schema: %v", err)
	}
	assertGeneratedPath(t, doc.Fields, "metadata.name", false)
	assertGeneratedPath(t, doc.Fields, "metadata.labels", false)
	assertGeneratedPath(t, doc.Fields, "metadata.annotations", false)
	assertGeneratedPath(t, doc.Fields, "spec.compositeTypeRef.apiVersion", true)
	assertGeneratedPath(t, doc.Fields, "spec.compositeTypeRef.kind", true)
}
```

Add helper:

```go
func assertGeneratedPath(t *testing.T, fields []struct {
	Path     string `json:"path"`
	Required bool   `json:"required,omitempty"`
}, path string, required bool) {
	t.Helper()
	for _, field := range fields {
		if field.Path == path {
			if field.Required != required {
				t.Fatalf("%s required = %v, want %v", path, field.Required, required)
			}
			return
		}
	}
	t.Fatalf("missing generated path %s", path)
}
```

- [ ] **Step 3: Run generator tests and confirm they fail**

Run:

```bash
go test ./internal/schemagen
```

Expected: fail because `Config`, `ReleaseConfig`, and `Generate` do not exist.

- [ ] **Step 4: Implement config types**

Create `internal/schemagen/config.go`:

```go
package schemagen

type Config struct {
	BundleFormatVersion int             `json:"bundleFormatVersion"`
	Releases            []ReleaseConfig `json:"releases"`
}

type ReleaseConfig struct {
	Tag             string `json:"tag"`
	Commit          string `json:"commit"`
	RawCRDDir       string `json:"rawCRDDir"`
	CrossplaneGoMod string `json:"crossplaneGoMod"`
}

func LoadConfigFile(path string) (Config, error)
```

- [ ] **Step 5: Implement generator normalization**

Create `internal/schemagen/generator.go` with these exported functions and behavior:

```go
package schemagen

func Generate(cfg Config, outDir string) error
```

Implementation requirements:

- `LoadConfigFile` resolves `rawCRDDir` and `crossplaneGoMod` relative to the directory that contains the config file.
- Walk each resolved `ReleaseConfig.RawCRDDir`.
- Parse files ending in `.yaml` or `.yml`.
- Select documents with `apiVersion: apiextensions.k8s.io/v1` and `kind: CustomResourceDefinition`.
- For each served CRD version with `schema.openAPIV3Schema`, produce one schema JSON document.
- Emit paths by descending `properties`.
- Emit array item paths with `[]` when a schema has `type: array` and `items`.
- Mark `parent.child` required when the parent schema's `required` list contains `child`.
- Add metadata enrichment for `metadata.name`, `metadata.labels`, `metadata.annotations`; add `metadata.namespace` only when CRD `spec.scope` is `Namespaced`.
- Resolve only local `#/...` refs.
- Sort fields by path and marshal JSON with deterministic indentation.
- Write `manifest.json` with sorted schema paths.

Use these internal structs:

```go
type crdDocument struct {
	Spec struct {
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
	Ref                       string                    `yaml:"$ref"`
	Type                      string                    `yaml:"type"`
	Description               string                    `yaml:"description"`
	Properties                map[string]openAPISchema  `yaml:"properties"`
	Required                  []string                  `yaml:"required"`
	Items                     *openAPISchema            `yaml:"items"`
	Default                   any                       `yaml:"default"`
	Enum                      []any                     `yaml:"enum"`
	AdditionalProperties      any                       `yaml:"additionalProperties"`
	XKubernetesPreserveUnknown bool                     `yaml:"x-kubernetes-preserve-unknown-fields"`
}
```

Use `go.yaml.in/yaml/v4` for YAML parsing.

- [ ] **Step 6: Add CLI entry point**

Create `cmd/vibe-xpls-schema-gen/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/io41/vibe-xpls/internal/schemagen"
)

func main() {
	configPath := flag.String("config", "internal/analyzer/schemadata/config.json", "schema generator config")
	outDir := flag.String("out", "internal/analyzer/schemadata", "schema data output directory")
	flag.Parse()

	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if err := schemagen.Generate(cfg, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate schemas: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Run generator tests**

Run:

```bash
go test ./internal/schemagen
```

Expected: pass.

- [ ] **Step 8: Commit**

```bash
git add cmd/vibe-xpls-schema-gen internal/schemagen
git commit -m "feat: add schema generator"
```

---

### Task 4: Real Pinned Source Artifacts And Generated Bundle

**Files:**
- Create: `internal/analyzer/schemadata/config.json`
- Create: `internal/analyzer/schemadata/upstream/crossplane/v1.20.7/go.mod`
- Create: `internal/analyzer/schemadata/upstream/crossplane/v1.20.7/cluster/crds/*.yaml`
- Create: `internal/analyzer/schemadata/upstream/crossplane/v2.2.1/go.mod`
- Create: `internal/analyzer/schemadata/upstream/crossplane/v2.2.1/cluster/crds/*.yaml`
- Replace: `internal/analyzer/schemadata/manifest.json`
- Replace: `internal/analyzer/schemadata/schemas/**/*.json`
- Create: `docs/generated-schemas.md`
- Create: `NOTICE`
- Modify: `internal/analyzer/schema.go`
- Modify: `internal/analyzer/schema_test.go`

- [ ] **Step 1: Add real schema generator config**

Create `internal/analyzer/schemadata/config.json`:

```json
{
  "bundleFormatVersion": 1,
  "releases": [
    {
      "tag": "v1.20.7",
      "commit": "5fae6c1ab967e57b1dc792b5c52c97bceda12953",
      "rawCRDDir": "upstream/crossplane/v1.20.7/cluster/crds",
      "crossplaneGoMod": "upstream/crossplane/v1.20.7/go.mod"
    },
    {
      "tag": "v2.2.1",
      "commit": "713541df7fc5cf0946b6573837831086465a2dbe",
      "rawCRDDir": "upstream/crossplane/v2.2.1/cluster/crds",
      "crossplaneGoMod": "upstream/crossplane/v2.2.1/go.mod"
    }
  ]
}
```

- [ ] **Step 2: Vendor pinned upstream source artifacts**

Run these commands from the repo root:

```bash
mkdir -p /tmp/vibe-xpls-schema-sources
curl -L -o /tmp/vibe-xpls-schema-sources/crossplane-v1.20.7.tar.gz https://github.com/crossplane/crossplane/archive/refs/tags/v1.20.7.tar.gz
curl -L -o /tmp/vibe-xpls-schema-sources/crossplane-v2.2.1.tar.gz https://github.com/crossplane/crossplane/archive/refs/tags/v2.2.1.tar.gz
tar -xzf /tmp/vibe-xpls-schema-sources/crossplane-v1.20.7.tar.gz -C /tmp/vibe-xpls-schema-sources
tar -xzf /tmp/vibe-xpls-schema-sources/crossplane-v2.2.1.tar.gz -C /tmp/vibe-xpls-schema-sources
mkdir -p internal/analyzer/schemadata/upstream/crossplane/v1.20.7/cluster/crds
mkdir -p internal/analyzer/schemadata/upstream/crossplane/v2.2.1/cluster/crds
cp /tmp/vibe-xpls-schema-sources/crossplane-1.20.7/go.mod internal/analyzer/schemadata/upstream/crossplane/v1.20.7/go.mod
cp /tmp/vibe-xpls-schema-sources/crossplane-1.20.7/cluster/crds/*.yaml internal/analyzer/schemadata/upstream/crossplane/v1.20.7/cluster/crds/
cp /tmp/vibe-xpls-schema-sources/crossplane-2.2.1/go.mod internal/analyzer/schemadata/upstream/crossplane/v2.2.1/go.mod
cp /tmp/vibe-xpls-schema-sources/crossplane-2.2.1/cluster/crds/*.yaml internal/analyzer/schemadata/upstream/crossplane/v2.2.1/cluster/crds/
```

Expected: both upstream directories contain `go.mod` and many CRD YAML files.

- [ ] **Step 3: Generate canonical schema bundle**

Run:

```bash
go run ./cmd/vibe-xpls-schema-gen --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

Expected: `internal/analyzer/schemadata/manifest.json` lists both releases and `internal/analyzer/schemadata/schemas/v1.20.7` plus `internal/analyzer/schemadata/schemas/v2.2.1` contain generated JSON files.

- [ ] **Step 4: Add stale-generation test**

Add to `internal/analyzer/schema_test.go`:

```go
func TestGeneratedSchemaBundleIsCurrent(t *testing.T) {
	tmp := t.TempDir()
	cfg, err := schemagen.LoadConfigFile("schemadata/config.json")
	if err != nil {
		t.Fatalf("load generator config: %v", err)
	}
	if err := schemagen.Generate(cfg, tmp); err != nil {
		t.Fatalf("regenerate bundle: %v", err)
	}
	assertDirectoriesEqual(t, "schemadata/manifest.json", filepath.Join(tmp, "manifest.json"))
	assertDirectoriesEqual(t, "schemadata/schemas", filepath.Join(tmp, "schemas"))
}
```

- [ ] **Step 5: Remove hand-written built-in field maps**

In `internal/analyzer/schema.go`, replace the old hard-coded `LoadBuiltIns` body with a generated-bundle wrapper:

```go
func (idx *SchemaIndex) LoadBuiltIns() {
	idx.bundleStatus = idx.LoadGeneratedBuiltIns()
}
```

There must be no hard-coded Crossplane field map in `LoadBuiltIns` after this step.

Add imports to `schema_test.go`:

```go
import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/io41/vibe-xpls/internal/schemagen"
)
```

Add helper functions:

```go
func assertDirectoriesEqual(t *testing.T, wantPath, gotPath string) {
	t.Helper()
	wantInfo, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("stat want path %s: %v", wantPath, err)
	}
	if !wantInfo.IsDir() {
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("read want file %s: %v", wantPath, err)
		}
		got, err := os.ReadFile(gotPath)
		if err != nil {
			t.Fatalf("read got file %s: %v", gotPath, err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("%s is stale", wantPath)
		}
		return
	}
	entries, err := os.ReadDir(wantPath)
	if err != nil {
		t.Fatalf("read want dir %s: %v", wantPath, err)
	}
	for _, entry := range entries {
		assertDirectoriesEqual(t, filepath.Join(wantPath, entry.Name()), filepath.Join(gotPath, entry.Name()))
	}
}
```

- [ ] **Step 6: Add generated schema docs**

Create `docs/generated-schemas.md`:

```markdown
# Generated Schemas

The built-in Crossplane schema bundle is generated from committed upstream Crossplane release artifacts.

Current pins:

- Crossplane `v1.20.7`, commit `5fae6c1ab967e57b1dc792b5c52c97bceda12953`
- Crossplane `v2.2.1`, commit `713541df7fc5cf0946b6573837831086465a2dbe`

Regenerate after changing `internal/analyzer/schemadata/config.json` or generator code:

```bash
go run ./cmd/vibe-xpls-schema-gen --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
go test ./internal/schemagen ./internal/analyzer
```

The generator must produce byte-identical output from committed inputs. Runtime never downloads schemas.
```

- [ ] **Step 7: Add attribution notice**

Create `NOTICE`:

```text
vibe-xpls

This product includes generated schema data derived from Crossplane CustomResourceDefinition artifacts.

Crossplane is licensed under the Apache License, Version 2.0.
Source: https://github.com/crossplane/crossplane
Pinned releases used for generated schema data:
- v1.20.7, commit 5fae6c1ab967e57b1dc792b5c52c97bceda12953
- v2.2.1, commit 713541df7fc5cf0946b6573837831086465a2dbe
```

- [ ] **Step 8: Run stale-generation and bundle tests**

Run:

```bash
go test ./internal/schemagen ./internal/analyzer -run 'TestGeneratedSchemaBundleIsCurrent|TestEmbeddedSchemaBundleLoadsFixture|TestBuiltInCrossplaneSchemas'
```

Expected: pass.

- [ ] **Step 9: Commit**

```bash
git add cmd/vibe-xpls-schema-gen internal/schemagen internal/analyzer/schemadata internal/analyzer/schema.go internal/analyzer/schema_test.go docs/generated-schemas.md NOTICE
git commit -m "feat: generate Crossplane core schemas"
```

---

### Task 5: Package-Scoped Release Resolver

**Files:**
- Create: `internal/analyzer/schema_resolver.go`
- Modify: `internal/analyzer/schema.go`
- Modify: `internal/analyzer/analyzer.go`
- Modify: `internal/analyzer/schema_test.go`
- Modify: `internal/analyzer/analyzer_test.go`

- [ ] **Step 1: Add failing resolver tests**

Add to `internal/analyzer/schema_test.go`:

```go
func TestResolveReleaseNoPackageByExactGVK(t *testing.T) {
	idx := NewSchemaIndex()
	v1 := CrossplaneRelease{Tag: "v1.20.7"}
	v2 := CrossplaneRelease{Tag: "v2.2.1"}
	idx.AddGeneratedBuiltIn(Schema{Release: v1, GVK: SourceGVK{APIVersion: "example.io/v1", Kind: "OnlyV1"}, Fields: map[string]FieldDoc{"spec.v1": {Path: "spec.v1"}}})
	idx.AddGeneratedBuiltIn(Schema{Release: v1, GVK: SourceGVK{APIVersion: "example.io/v1", Kind: "Both"}, Fields: map[string]FieldDoc{"spec.v1": {Path: "spec.v1"}}})
	idx.AddGeneratedBuiltIn(Schema{Release: v2, GVK: SourceGVK{APIVersion: "example.io/v1", Kind: "Both"}, Fields: map[string]FieldDoc{"spec.v2": {Path: "spec.v2"}}})

	got, ok := idx.DefaultReleaseForGVK(SourceGVK{APIVersion: "example.io/v1", Kind: "OnlyV1"})
	if !ok || got != v1 {
		t.Fatalf("OnlyV1 release = %#v ok=%v, want v1", got, ok)
	}
	got, ok = idx.DefaultReleaseForGVK(SourceGVK{APIVersion: "example.io/v1", Kind: "Both"})
	if !ok || got != v2 {
		t.Fatalf("Both release = %#v ok=%v, want v2", got, ok)
	}
}
```

Add analyzer-level tests for a package constraint:

```go
func TestAnalyzerCompletionUsesPackageCrossplaneVersionConstraint(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "crossplane.yaml"), []byte("apiVersion: meta.pkg.crossplane.io/v1\nkind: Configuration\nspec:\n  crossplane:\n    version: \">=v1.20.0 <v2.0.0\"\n"), 0o600); err != nil {
		t.Fatalf("write package metadata: %v", err)
	}
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  r"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "resources") {
		t.Fatalf("v1 constrained package should offer resources when present in v1 schema: %#v", completion.Items)
	}
}
```

- [ ] **Step 2: Run resolver tests and confirm they fail**

Run:

```bash
go test ./internal/analyzer -run 'TestResolveReleaseNoPackageByExactGVK|TestAnalyzerCompletionUsesPackageCrossplaneVersionConstraint'
```

Expected: fail because release resolution is not implemented.

- [ ] **Step 3: Add schema index release helpers**

Add to `internal/analyzer/schema.go`:

```go
func (idx *SchemaIndex) ReleasesForGVK(gvk SourceGVK) []CrossplaneRelease
func (idx *SchemaIndex) DefaultReleaseForGVK(gvk SourceGVK) (CrossplaneRelease, bool)
```

Implementation:

- Iterate `idx.releaseSchemas`.
- Filter by exact `APIVersion` and `Kind`.
- Sort releases by SemVer using a small parser for tags like `v1.20.7`.
- Return the last release for default.

- [ ] **Step 4: Implement package resolver**

Create `internal/analyzer/schema_resolver.go`:

```go
package analyzer

type schemaResolution struct {
	Release CrossplaneRelease
	Reason  SuppressionReason
	OK      bool
}

type SuppressionReason string

const (
	SuppressionMissingRootGVK         SuppressionReason = "missing-root-gvk"
	SuppressionUnknownGVK             SuppressionReason = "unknown-gvk"
	SuppressionNoSchemaForRelease     SuppressionReason = "no-schema-for-release"
	SuppressionMalformedYAMLContext   SuppressionReason = "malformed-yaml-context"
	SuppressionUnstableTemplatePath   SuppressionReason = "unstable-template-path"
	SuppressionUnsupportedSchemaShape SuppressionReason = "unsupported-schema-shape"
	SuppressionBundleDisabled         SuppressionReason = "bundle-disabled"
)
```

Add:

```go
func (a *Analyzer) resolveSchemaRelease(uri string, gvk SourceGVK) schemaResolution
```

Resolution rules:

- If bundle status is not OK, return `SuppressionBundleDisabled`.
- If no bundle release contains the exact GVK, return `SuppressionUnknownGVK`.
- If the file is not inside a package root, use `idx.DefaultReleaseForGVK(gvk)`.
- If inside a package root, parse the package marker file and read `spec.crossplane.version` when present.
- Filter release candidates by the package SemVer range.
- Intersect with candidates containing the exact GVK.
- If multiple remain, return latest SemVer.
- If none remain after package filtering, return `SuppressionNoSchemaForRelease`.

Implement only the range forms needed by Crossplane package metadata now:

```go
>=v1.20.0 <v2.0.0
>=v1.12.1-0
```

For unsupported range syntax, ignore the package constraint and use exact-GVK availability.

- [ ] **Step 5: Wire resolver into completion and hover**

Modify `CompletionAtOffset`, `Completion`, `HoverAtOffset`, and `Hover` so they resolve a release before schema lookup.

Example completion path:

```go
gvk := SourceGVK{APIVersion: apiVersion, Kind: kind}
resolution := a.resolveSchemaRelease(uri, gvk)
if !resolution.OK {
	return Completion{Reason: resolution.Reason}
}
candidate := completionFromSchema(a.schemas, resolution.Release, apiVersion, kind, parentPath)
```

Add `Reason SuppressionReason` to `Completion`.

Add release-aware overload:

```go
func completionFromSchema(schemas *SchemaIndex, release CrossplaneRelease, apiVersion, kind, parentPath string) Completion
```

- [ ] **Step 6: Run resolver tests**

Run:

```bash
go test ./internal/analyzer -run 'TestResolveReleaseNoPackageByExactGVK|TestAnalyzerCompletionUsesPackageCrossplaneVersionConstraint|TestAnalyzerDiagnosticsHoverAndCompletion'
```

Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/analyzer/schema.go internal/analyzer/schema_resolver.go internal/analyzer/completion.go internal/analyzer/hover.go internal/analyzer/schema_test.go internal/analyzer/analyzer_test.go
git commit -m "feat: resolve schema release per package"
```

---

### Task 6: Completion Semantics From Generated Schema

**Files:**
- Modify: `internal/analyzer/completion.go`
- Modify: `internal/analyzer/schema.go`
- Modify: `internal/analyzer/analyzer_test.go`
- Modify: `internal/lsp/server.go`
- Modify: `internal/lsp/server_test.go`

- [ ] **Step 1: Add failing analyzer completion tests**

Add to `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionSortsRootAndRequiredKeys(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	rootCompletion := a.Completion(uri, "")
	if got := completionLabels(rootCompletion.Items[:4]); !reflect.DeepEqual(got, []string{"apiVersion", "kind", "metadata", "spec"}) {
		t.Fatalf("root labels = %#v", got)
	}
	specCompletion := a.Completion(uri, "spec.compositeTypeRef")
	got := completionLabels(specCompletion.Items)
	if len(got) < 2 || got[0] != "apiVersion" || got[1] != "kind" {
		t.Fatalf("required compositeTypeRef labels should sort first, got %#v", got)
	}
}

func TestAnalyzerCompletionUsesArrayItemSchemaPath(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition-pipeline.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec:\n  pipeline:\n    - functionRef:\n        n"
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if !containsCompletion(completion.Items, "name") {
		t.Fatalf("array item completion missing name: %#v", completion.Items)
	}
}

func TestAnalyzerCompletionSuppressesRootStatusOnly(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "composition.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"
	a.OpenDocument(uri, text)

	if containsCompletion(a.Completion(uri, "").Items, "status") {
		t.Fatal("root status should be suppressed")
	}
}

func TestCompletionDoesNotSuppressNestedStatusSchemaPath(t *testing.T) {
	idx := NewSchemaIndex()
	release := CrossplaneRelease{Tag: "v2.2.1"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: release,
		GVK:     SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "CompositeResourceDefinition"},
		Fields: map[string]FieldDoc{
			"spec.versions[].schema.openAPIV3Schema.properties.status": {
				Path:        "spec.versions[].schema.openAPIV3Schema.properties.status",
				Description: "Status schema property.",
			},
		},
	})

	completion := completionFromSchema(idx, release, "apiextensions.crossplane.io/v1", "CompositeResourceDefinition", "spec.versions[].schema.openAPIV3Schema.properties")
	if !containsCompletion(completion.Items, "status") {
		t.Fatalf("nested schema status should not be suppressed: %#v", completion.Items)
	}
}
```

Add helper:

```go
func completionLabels(items []CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}
```

- [ ] **Step 2: Add failing LSP sortText test**

In `internal/lsp/server_test.go`, extend `TestCompletionItemsIncludePresentationMetadata`:

```go
if item["sortText"] == "" {
	t.Fatalf("completion item %#v missing sortText", item["label"])
}
```

Also assert the root sort order is stable:

```go
if item["label"] == "apiVersion" && item["sortText"] != "0000_apiVersion" {
	t.Fatalf("apiVersion sortText = %#v, want 0000_apiVersion", item["sortText"])
}
```

- [ ] **Step 3: Run focused tests and confirm they fail**

Run:

```bash
go test ./internal/analyzer ./internal/lsp -run 'TestAnalyzerCompletionSortsRootAndRequiredKeys|TestAnalyzerCompletionUsesArrayItemSchemaPath|TestAnalyzerCompletionSuppressesRootStatusOnly|TestCompletionItemsIncludePresentationMetadata'
```

Expected: fail because ordering, array path normalization, and `sortText` are not complete.

- [ ] **Step 4: Add completion metadata and schema-path normalization**

Update `CompletionItem` in `internal/analyzer/completion.go`:

```go
type CompletionItem struct {
	Label         string
	Path          string
	Documentation string
	SortText      string
	TextEdit      *CompletionTextEdit
}
```

Add:

```go
func schemaPathFromParsedPath(path string) string {
	re := regexp.MustCompile(`\[\d+\]`)
	return re.ReplaceAllString(path, "[]")
}
```

Use `schemaPathFromParsedPath` before schema lookup for completion parent paths and existing-path filtering.

- [ ] **Step 5: Implement ordering and root status suppression**

Update `completionFromSchema`:

- Use `FieldsForRelease`.
- Build immediate child labels only.
- Omit root-level `status`.
- Preserve parent/object completions.
- Use `fieldCompletionDocumentation(field)` for leaf docs.
- Sort by:
  1. root rank: `apiVersion`, `kind`, `metadata`, `spec`;
  2. required fields before optional;
  3. lexical label.
- Set stable `SortText` with zero-padded rank:

```go
func completionSortText(parentPath string, item completionCandidate, index int) string {
	return fmt.Sprintf("%04d_%s", index, item.label)
}
```

Ensure the same `(release, GVK, parent path, label)` gets the same `sortText` across runs.

- [ ] **Step 6: Emit `sortText` in LSP**

Update `completionItem` in `internal/lsp/server.go`:

```go
SortText string `json:"sortText,omitempty"`
```

Map analyzer item:

```go
SortText: item.SortText,
```

- [ ] **Step 7: Run focused completion tests**

Run:

```bash
go test ./internal/analyzer ./internal/lsp -run 'Completion'
```

Expected: pass.

- [ ] **Step 8: Commit**

```bash
git add internal/analyzer/completion.go internal/analyzer/schema.go internal/analyzer/analyzer_test.go internal/lsp/server.go internal/lsp/server_test.go
git commit -m "feat: complete from generated schema paths"
```

---

### Task 7: Degradation Reasons, Runtime Warning, And Logging

**Files:**
- Modify: `internal/analyzer/analyzer.go`
- Modify: `internal/analyzer/completion.go`
- Modify: `internal/analyzer/schema_bundle.go`
- Modify: `internal/lsp/server.go`
- Modify: `internal/lsp/server_test.go`
- Modify: `internal/analyzer/analyzer_test.go`

- [ ] **Step 1: Add analyzer degradation tests**

Add to `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerCompletionReportsMissingRootGVKReason(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "missing-gvk.yaml")
	a.OpenDocument(uri, "spec:\n  ")

	completion := a.CompletionAtOffset(uri, len("spec:\n  "))
	if completion.Reason != SuppressionMissingRootGVK {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionMissingRootGVK)
	}
}

func TestAnalyzerCompletionReportsMalformedYAMLReason(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "malformed.yaml")
	text := "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\nspec: [unterminated\n  "
	a.OpenDocument(uri, text)

	completion := a.CompletionAtOffset(uri, len(text))
	if completion.Reason != SuppressionMalformedYAMLContext {
		t.Fatalf("reason = %q, want %q", completion.Reason, SuppressionMalformedYAMLContext)
	}
}
```

- [ ] **Step 2: Add LSP warning/logging tests**

Add a server constructor test seam in `internal/lsp/server.go`:

```go
type analyzerFactory func(analyzer.Options) (*analyzer.Analyzer, error)
```

Add field:

```go
newAnalyzer analyzerFactory
loggedSuppression map[string]struct{}
bundleWarningShown bool
```

Initialize `newAnalyzer` in `NewServer` to `analyzer.New`.

Add this failing test in `internal/lsp/server_test.go` after `TestInitializeAdvertisesCapabilitiesAndNegotiatesPositionEncoding`:

```go
func TestInitializeWarnsOnceForBundleFailure(t *testing.T) {
	var stderr bytes.Buffer
	in := bytes.NewBuffer(nil)
	out := bytes.NewBuffer(nil)
	s := NewServer(in, out, &stderr)
	s.newAnalyzer = func(options analyzer.Options) (*analyzer.Analyzer, error) {
		a, err := analyzer.New(options)
		if err != nil {
			return nil, err
		}
		a.SetSchemaBundleStatusForTest(analyzer.SchemaBundleStatus{Message: "unsupported schema bundle format 99"})
		return a, nil
	}

	frames := []string{
		requestFrame(t, 1, "initialize", map[string]any{"rootUri": fileURI(testRoot(t)), "capabilities": map[string]any{}}),
		requestFrame(t, 2, "initialize", map[string]any{"rootUri": fileURI(testRoot(t)), "capabilities": map[string]any{}}),
		notificationFrame(t, "exit", nil),
	}
	for _, frame := range frames {
		if _, err := in.WriteString(frame); err != nil {
			t.Fatalf("write frame: %v", err)
		}
	}
	if code := s.Run(); code != 0 {
		t.Fatalf("server exit = %d", code)
	}
	messages := readMessages(t, out.Bytes())
	warnings := 0
	for _, msg := range messages {
		if msg.Method == "window/showMessage" {
			warnings++
			params := paramsMap(t, msg)
			if params["type"] != float64(2) || !strings.Contains(params["message"].(string), "schema completions are disabled") {
				t.Fatalf("warning params = %#v", params)
			}
		}
	}
	if warnings != 1 {
		t.Fatalf("warnings = %d, want 1", warnings)
	}
}
```

- [ ] **Step 3: Run degradation tests and confirm they fail**

Run:

```bash
go test ./internal/analyzer ./internal/lsp -run 'TestAnalyzerCompletionReports|TestInitializeWarns'
```

Expected: fail until reasons and warning path exist.

- [ ] **Step 4: Add bundle status accessor**

In `internal/analyzer/analyzer.go`:

```go
func (a *Analyzer) SchemaBundleStatus() SchemaBundleStatus {
	return a.schemas.bundleStatus
}
```

Add this analyzer test helper in a non-test file because the LSP package needs it:

```go
func (a *Analyzer) SetSchemaBundleStatusForTest(status SchemaBundleStatus) {
	a.schemas.bundleStatus = status
}
```

- [ ] **Step 5: Populate suppression reasons**

In `CompletionAtOffset`:

- If `ParseYAMLDocument` has diagnostics that make context unsafe, return `SuppressionMalformedYAMLContext`.
- If `completionContextAtOffset` fails because of template action, return `SuppressionUnstableTemplatePath`.
- If root `apiVersion` or `kind` is missing, return `SuppressionMissingRootGVK`.
- If release resolver cannot find exact GVK, return `SuppressionUnknownGVK` or `SuppressionNoSchemaForRelease`.
- If bundle load failed, return `SuppressionBundleDisabled`.

Keep empty completion wire shape unchanged.

- [ ] **Step 6: Implement LSP warning and throttled logs**

After analyzer creation in `handleInitialize`, call:

```go
if status := s.analyzer.SchemaBundleStatus(); !status.OK && !s.bundleWarningShown {
	s.bundleWarningShown = true
	if err := s.notify("window/showMessage", map[string]any{
		"type":    2,
		"message": "Crossplane schema completions are disabled: " + status.Message,
	}); err != nil {
		return err
	}
}
```

When completion returns no items and has a reason, log to `errOut` with throttling:

```go
func (s *Server) logSuppression(uri string, generation analyzer.Generation, reason analyzer.SuppressionReason) {
	if reason == "" {
		return
	}
	key := string(reason)
	if reason != analyzer.SuppressionBundleDisabled {
		key = fmt.Sprintf("%s:%d:%s", uri, generation, reason)
	}
	if _, ok := s.loggedSuppression[key]; ok {
		return
	}
	s.loggedSuppression[key] = struct{}{}
	fmt.Fprintf(s.errOut, "completion suppressed: uri=%s reason=%s\n", uri, reason)
}
```

- [ ] **Step 7: Run degradation tests**

Run:

```bash
go test ./internal/analyzer ./internal/lsp -run 'TestAnalyzerCompletionReports|TestInitializeWarns|TestCompletion'
```

Expected: pass.

- [ ] **Step 8: Commit**

```bash
git add internal/analyzer internal/lsp
git commit -m "feat: report schema completion degradation"
```

---

### Task 8: Docs, Full Verification, And Manual Test Notes

**Files:**
- Modify: `README.md`
- Modify: `docs/generated-schemas.md`
- Modify: `docs/superpowers/plans/2026-05-21-generated-completion-foundation.md` only if implementation details changed during execution

- [ ] **Step 1: Update README with generated schema behavior**

Add a short section to `README.md`:

```markdown
## Built-In Crossplane Schemas

`vibe-xpls` ships an offline generated schema bundle for Crossplane core resources. The bundle is generated from pinned Crossplane release artifacts and is used for YAML key completions and completion documentation.

Current built-in release lines:

- Crossplane `v1.20.7`
- Crossplane `v2.2.1`

Runtime does not download schemas, read registries, or connect to clusters.
```

- [ ] **Step 2: Run full verification**

Run:

```bash
go test ./...
```

Expected: pass.

Run:

```bash
go run ./cmd/vibe-xpls-schema-gen --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
git diff --exit-code -- internal/analyzer/schemadata
```

Expected: no diff.

- [ ] **Step 3: Build binary for manual testing**

Run:

```bash
go build -o /private/tmp/vibe-xpls ./cmd/vibe-xpls
```

Expected: binary exists at `/private/tmp/vibe-xpls`.

- [ ] **Step 4: Manual Zed smoke tests**

Use the extension configured to `/private/tmp/vibe-xpls`.

Test these cases:

- Open `internal/analyzer/testdata/workspaces/root/crossplane.yaml`; at root, `apiVersion`, `kind`, `metadata`, `spec` sort first.
- In a `Composition`, complete under `spec.compositeTypeRef`; `apiVersion` and `kind` appear first and have documentation.
- In a `Composition` with `spec.pipeline: - functionRef:`, complete under `functionRef`; `name` appears.
- In a `Configuration`, complete under `spec.dependsOn:`; package dependency keys appear with documentation and no generic `detail`.
- In a package constrained to Crossplane 1.x, v1-specific Composition fields appear where they exist in the generated schema.
- In a package without a root context, an exact GVK present in the bundle still returns built-in completions.
- Completion acceptance does not reintroduce the root `spec:` indentation regression.

- [ ] **Step 5: Commit docs and verification notes**

```bash
git add README.md docs/generated-schemas.md
git commit -m "docs: document generated schema bundle"
```

- [ ] **Step 6: Final branch check**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected: clean worktree on the implementation branch, with task commits in order.
