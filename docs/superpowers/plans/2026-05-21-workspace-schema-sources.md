# Workspace Schema Sources Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Load workspace CRD and XRD OpenAPI schemas into the existing schema model so local provider managed resources and workspace composite resources can provide key completions and hover docs.

**Architecture:** Reuse the current `SchemaIndex`, `Schema`, `FieldDoc`, and completion/hover paths. Add a workspace schema loader that scans package-local YAML files, extracts CRD and XRD OpenAPI schemas, converts them with the same path normalization used by generated built-ins, and registers them as `SchemaSourceWorkspace` schemas.

**Tech Stack:** Go 1.26.3, existing analyzer package, `go.yaml.in/yaml/v4`, existing YAML parser and schema index types.

---

## File Structure

- Create: `internal/analyzer/workspace_schema_loader.go`
  Workspace-local schema discovery and conversion from CRD/XRD YAML to `Schema`.
- Modify: `internal/analyzer/analyzer.go`
  Load workspace schemas during analyzer construction after built-ins are loaded.
- Modify: `internal/analyzer/schema.go`
  Add helper surface only if needed for loader integration; keep existing lookup behavior unchanged.
- Modify: `internal/analyzer/analyzer_test.go`
  End-to-end workspace schema completion and hover tests.
- Add: `internal/analyzer/testdata/workspaces/root/api/provider-crd.yaml`
  Minimal provider CRD fixture.
- Add: `internal/analyzer/testdata/workspaces/root/api/xrd.yaml`
  Minimal XRD fixture with a defined XR schema.
- Modify: `PROJECT_ROADMAP.md`
  Move this item from `Next` to `Current` or `Done` when implemented.

## Task 1: Provider CRD Workspace Schema

**Files:**
- Add: `internal/analyzer/testdata/workspaces/root/api/provider-crd.yaml`
- Modify: `internal/analyzer/analyzer_test.go`
- Create: `internal/analyzer/workspace_schema_loader.go`
- Modify: `internal/analyzer/analyzer.go`

- [ ] **Step 1: Add a provider CRD fixture**

Create `internal/analyzer/testdata/workspaces/root/api/provider-crd.yaml`:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: buckets.s3.aws.upbound.io
spec:
  group: s3.aws.upbound.io
  names:
    kind: Bucket
    plural: buckets
  scope: Namespaced
  versions:
    - name: v1beta1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                forProvider:
                  type: object
                  properties:
                    bucketName:
                      type: string
                      description: BucketName is the provider bucket name.
                    acl:
                      type: string
                      enum:
                        - private
                        - public-read
              required:
                - forProvider
```

- [ ] **Step 2: Add the failing provider CRD completion test**

Add this test to `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerLoadsWorkspaceProviderCRDSchema(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "bucket-instance.yaml")
	text := "apiVersion: s3.aws.upbound.io/v1beta1\nkind: Bucket\nspec:\n  forProvider:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec.forProvider")
	if !containsCompletion(completion.Items, "bucketName") {
		t.Fatalf("workspace provider CRD completion missing bucketName: %#v", completion.Items)
	}
	item, ok := completionItemByLabel(completion.Items, "bucketName")
	if !ok || !strings.Contains(item.Documentation, "BucketName is the provider bucket name.") {
		t.Fatalf("bucketName completion = %#v, want provider CRD documentation", item)
	}
	hover, ok := a.Hover(uri, "spec.forProvider.bucketName")
	if !ok || !strings.Contains(hover.Markdown, "BucketName is the provider bucket name.") {
		t.Fatalf("hover = %#v ok=%v, want provider CRD documentation", hover, ok)
	}
}
```

- [ ] **Step 3: Run the focused test and confirm it fails**

Run:

```sh
go test ./internal/analyzer -run TestAnalyzerLoadsWorkspaceProviderCRDSchema
```

Expected: fail because workspace CRD files are not loaded into `SchemaIndex`.

- [ ] **Step 4: Implement the workspace schema loader skeleton**

Create `internal/analyzer/workspace_schema_loader.go`:

```go
package analyzer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
		Scope string `yaml:"scope"`
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
	Items       *workspaceOpenAPISchema           `yaml:"items"`
	Enum        []any                             `yaml:"enum"`
}

func (a *Analyzer) loadWorkspaceSchemas() {
	for _, pkg := range a.workspace.PackageRoots {
		for _, schema := range workspaceSchemasForPackage(pkg) {
			a.schemas.AddWorkspaceSchema(schema)
		}
	}
}

func workspaceSchemasForPackage(pkg PackageRoot) []Schema {
	files := workspaceYAMLFiles(pkg.Root)
	schemas := []Schema{}
	for _, path := range files {
		schemas = append(schemas, workspaceSchemasFromFile(path)...)
	}
	return schemas
}
```

- [ ] **Step 5: Wire the loader into analyzer construction**

Update `New` in `internal/analyzer/analyzer.go` so the analyzer is assigned before loading workspace schemas:

```go
	analyzer := &Analyzer{
		workspace: workspace,
		limits:    defaultLimits(options.Limits),
		docs:      NewDocumentStore(),
		schemas:   schemas,
	}
	analyzer.loadWorkspaceSchemas()
	return analyzer, nil
```

- [ ] **Step 6: Implement provider CRD extraction**

Add these helpers to `internal/analyzer/workspace_schema_loader.go`:

```go
func workspaceYAMLFiles(root string) []string {
	paths := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths
}

func workspaceSchemasFromFile(path string) []Schema {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	sum := sha256.Sum256(raw)
	var doc workspaceCRDDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	if doc.APIVersion != "apiextensions.k8s.io/v1" || doc.Kind != "CustomResourceDefinition" {
		return nil
	}
	schemas := []Schema{}
	for _, version := range doc.Spec.Versions {
		if !version.Served || doc.Spec.Group == "" || version.Name == "" || doc.Spec.Names.Kind == "" {
			continue
		}
		fields := map[string]FieldDoc{}
		collectWorkspaceFields(fields, version.Schema.OpenAPIV3Schema, "", nil)
		schemas = append(schemas, Schema{
			GVK: SourceGVK{APIVersion: doc.Spec.Group + "/" + version.Name, Kind: doc.Spec.Names.Kind},
			Fields: fields,
			Provenance: SchemaProvenance{
				Path:           path,
				Owner:          SchemaOwnerProvider,
				Source:         SchemaSourceWorkspace,
				UpstreamSHA256: hex.EncodeToString(sum[:]),
			},
		})
	}
	return schemas
}

func collectWorkspaceFields(fields map[string]FieldDoc, schema workspaceOpenAPISchema, prefix string, required []string) {
	requiredSet := map[string]struct{}{}
	for _, name := range required {
		requiredSet[name] = struct{}{}
	}
	for name, child := range schema.Properties {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		doc := FieldDoc{Path: path, Description: strings.TrimSpace(child.Description), Type: child.Type}
		if _, ok := requiredSet[name]; ok {
			doc.Required = true
		}
		for _, value := range child.Enum {
			doc.Enum = append(doc.Enum, fmt.Sprint(value))
		}
		fields[path] = doc
		if child.Items != nil {
			itemPath := path + "[]"
			fields[itemPath] = FieldDoc{Path: itemPath, Description: strings.TrimSpace(child.Description), Type: child.Items.Type}
			collectWorkspaceFields(fields, *child.Items, itemPath, child.Items.Required)
		}
		collectWorkspaceFields(fields, child, path, child.Required)
	}
}
```

- [ ] **Step 7: Run the provider CRD test**

Run:

```sh
go test ./internal/analyzer -run TestAnalyzerLoadsWorkspaceProviderCRDSchema
```

Expected: pass.

## Task 2: XRD-Derived Workspace XR Schema

**Files:**
- Add: `internal/analyzer/testdata/workspaces/root/api/xrd.yaml`
- Modify: `internal/analyzer/analyzer_test.go`
- Modify: `internal/analyzer/workspace_schema_loader.go`

- [ ] **Step 1: Add an XRD fixture**

Create `internal/analyzer/testdata/workspaces/root/api/xrd.yaml`:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xdatabases.platform.example.org
spec:
  group: platform.example.org
  names:
    kind: XDatabase
    plural: xdatabases
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                size:
                  type: string
                  description: Size selects the database capacity class.
                  enum:
                    - small
                    - large
```

- [ ] **Step 2: Add the failing XRD completion test**

Add this test to `internal/analyzer/analyzer_test.go`:

```go
func TestAnalyzerLoadsWorkspaceXRDSchema(t *testing.T) {
	root := testkit.FixturePath(t, "internal", "analyzer", "testdata", "workspaces", "root")
	a, err := New(Options{WorkspaceRoot: root, Limits: DefaultLimits()})
	if err != nil {
		t.Fatalf("new analyzer: %v", err)
	}
	uri := "file://" + filepath.Join(root, "api", "xdb.yaml")
	text := "apiVersion: platform.example.org/v1alpha1\nkind: XDatabase\nspec:\n"
	a.OpenDocument(uri, text)

	completion := a.Completion(uri, "spec")
	item, ok := completionItemByLabel(completion.Items, "size")
	if !ok {
		t.Fatalf("workspace XRD completion missing size: %#v", completion.Items)
	}
	if !strings.Contains(item.Documentation, "Size selects the database capacity class.") {
		t.Fatalf("size completion = %#v, want XRD documentation", item)
	}
	if !strings.Contains(item.Documentation, "_Allowed: small, large_") {
		t.Fatalf("size completion = %#v, want enum documentation", item)
	}
}
```

- [ ] **Step 3: Run the focused XRD test and confirm it fails**

Run:

```sh
go test ./internal/analyzer -run TestAnalyzerLoadsWorkspaceXRDSchema
```

Expected: fail because XRD documents are not yet converted into XR schemas.

- [ ] **Step 4: Add XRD document types and extraction**

Extend `internal/analyzer/workspace_schema_loader.go` with an XRD document type and branch in `workspaceSchemasFromFile`:

```go
type workspaceXRDDocument struct {
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
```

When the generic document header is `apiextensions.crossplane.io/v1` and
`CompositeResourceDefinition`, unmarshal into `workspaceXRDDocument`, produce
`SchemaOwnerUser`, and register GVKs using `spec.group + "/" + version.name`
and `spec.names.kind`.

Use this branch shape inside `workspaceSchemasFromFile`:

```go
var header struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}
if err := yaml.Unmarshal(raw, &header); err != nil {
	return nil
}
switch {
case header.APIVersion == "apiextensions.k8s.io/v1" && header.Kind == "CustomResourceDefinition":
	return workspaceCRDSchemasFromRaw(path, raw, sum)
case header.APIVersion == "apiextensions.crossplane.io/v1" && header.Kind == "CompositeResourceDefinition":
	return workspaceXRDSchemasFromRaw(path, raw, sum)
default:
	return nil
}
```

- [ ] **Step 5: Run the XRD test**

Run:

```sh
go test ./internal/analyzer -run TestAnalyzerLoadsWorkspaceXRDSchema
```

Expected: pass.

## Task 3: Conflict And Package Scope Behavior

**Files:**
- Modify: `internal/analyzer/analyzer_test.go`
- Modify: `internal/analyzer/workspace_schema_loader.go`

- [ ] **Step 1: Add duplicate workspace schema test**

Add this test to `internal/analyzer/analyzer_test.go`:

```go
func TestWorkspaceSchemaLoaderReportsWorkspaceDuplicate(t *testing.T) {
	idx := NewSchemaIndex()
	gvk := SourceGVK{APIVersion: "example.org/v1", Kind: "Widget"}
	idx.AddWorkspaceSchema(Schema{
		GVK:    gvk,
		Fields: map[string]FieldDoc{"spec.first": {Path: "spec.first"}},
		Provenance: SchemaProvenance{
			Path:   "first.yaml",
			Owner:  SchemaOwnerProvider,
			Source: SchemaSourceWorkspace,
		},
	})
	idx.AddWorkspaceSchema(Schema{
		GVK:    gvk,
		Fields: map[string]FieldDoc{"spec.second": {Path: "spec.second"}},
		Provenance: SchemaProvenance{
			Path:   "second.yaml",
			Owner:  SchemaOwnerProvider,
			Source: SchemaSourceWorkspace,
		},
	})
	if got := idx.Diagnostics(); len(got) != 1 || !strings.Contains(got[0].Message, "workspace schema conflicts") {
		t.Fatalf("diagnostics = %#v, want workspace conflict warning", got)
	}
}
```

- [ ] **Step 2: Add built-in duplicate test**

Add this test to `internal/analyzer/analyzer_test.go`:

```go
func TestWorkspaceSchemaLoaderDoesNotReplaceGeneratedBuiltIn(t *testing.T) {
	idx := NewSchemaIndex()
	release := CrossplaneRelease{Tag: "v2.2.1"}
	gvk := SourceGVK{APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition"}
	idx.AddGeneratedBuiltIn(Schema{
		Release: release,
		GVK:     gvk,
		Fields: map[string]FieldDoc{
			"spec.compositeTypeRef.kind": {
				Path:        "spec.compositeTypeRef.kind",
				Description: "generated built-in",
			},
		},
		Provenance: SchemaProvenance{Owner: SchemaOwnerCore, Source: SchemaSourceGeneratedBuiltIn},
	})
	idx.AddWorkspaceSchema(Schema{
		GVK: gvk,
		Fields: map[string]FieldDoc{
			"spec.compositeTypeRef.kind": {
				Path:        "spec.compositeTypeRef.kind",
				Description: "workspace duplicate",
			},
		},
		Provenance: SchemaProvenance{
			Path:   "composition-crd.yaml",
			Owner:  SchemaOwnerProvider,
			Source: SchemaSourceWorkspace,
		},
	})
	doc, ok := idx.FieldDocumentationForRelease(release, gvk.APIVersion, gvk.Kind, "spec.compositeTypeRef.kind")
	if !ok || doc.Description != "generated built-in" {
		t.Fatalf("built-in doc = %#v ok=%v, want generated built-in preserved", doc, ok)
	}
	if got := idx.Diagnostics(); len(got) != 1 || !strings.Contains(got[0].Message, "duplicates built-in") {
		t.Fatalf("diagnostics = %#v, want built-in duplicate warning", got)
	}
}
```

- [ ] **Step 3: Keep package-local scanning bounded**

Add this helper behavior to `workspaceYAMLFiles` so it skips unrelated package
roots when walking from a package root:

```go
func workspaceYAMLFiles(root string) []string {
	paths := []string{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && isPackageMarkerDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	return paths
}

func isPackageMarkerDir(path string) bool {
	for marker := range markerPriority {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run focused tests**

Run:

```sh
go test ./internal/analyzer -run 'Workspace.*Schema|Duplicate'
```

Expected: pass.

## Task 4: Verification And Roadmap Update

**Files:**
- Modify: `PROJECT_ROADMAP.md`

- [ ] **Step 1: Run full verification**

Run:

```sh
./scripts/update-generated.sh
```

Expected: schema generation is current, `go test ./...` passes, and both local
CLIs build into `dist/local/`.

- [ ] **Step 2: Update the roadmap**

Move "Workspace CRD/XRD schema sources" from `Next` to `Done` in
`PROJECT_ROADMAP.md`, and make "Function input schema dispatch" the first
`Next` item.

- [ ] **Step 3: Commit**

Run:

```sh
git add PROJECT_ROADMAP.md internal/analyzer internal/analyzer/testdata
git commit -m "feat: load workspace schema sources"
```
