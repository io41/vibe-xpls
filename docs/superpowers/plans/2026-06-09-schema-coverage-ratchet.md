# Schema Coverage Ratchet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an offline schema coverage ratchet for committed pinned Crossplane CRDs, with checked-in coverage artifacts, a human-maintained known-gap baseline, CLI enforcement, and scheduled upstream drift detection.

**Architecture:** Keep runtime analyzer loading separate from coverage enforcement. Build the ratchet inside `internal/schemagen`: extract upstream OpenAPI targets, read generated schema facts, classify coverage buckets, apply the checked-in baseline, render deterministic artifacts, and expose explicit generator CLI subcommands. Normal update and CI paths stay offline; the upstream drift command is a separate networked path for scheduled CI.

**Tech Stack:** Go 1.26.3, `encoding/json`, `crypto/sha256`, `net/http`, `go.yaml.in/yaml/v4`, existing schema generator package, Bash update script, GitHub Actions.

---

## Scope Notes

The design spec is [2026-06-09-schema-coverage-ratchet-design.md](../specs/2026-06-09-schema-coverage-ratchet-design.md). The roadmap frames this as part of the deterministic generated-schema update flow, so this plan keeps `./scripts/update-generated.sh` as the single local command.

Runtime analyzer behavior is intentionally unchanged. `internal/analyzer/schemadata/manifest.json` remains the runtime bundle manifest, and `internal/analyzer/schema_bundle.go` continues embedding only `manifest.json` and `schemas/*/*.json`.

CLI compatibility strategy:

- `vibe-xpls-schema-gen generate --config <path> --out <dir>` is the canonical schema generation command.
- `vibe-xpls-schema-gen --config <path> --out <dir>` remains a compatibility alias for `generate`.
- `vibe-xpls-schema-gen coverage generate --config <path> --out <dir>` writes generated coverage artifacts only.
- `vibe-xpls-schema-gen coverage check --config <path> --out <dir>` regenerates into a temporary directory, compares schema and coverage artifacts with committed output, validates the baseline, and exits non-zero on stale artifacts or ratchet failures.
- `vibe-xpls-schema-gen drift check --config <path>` is networked and never called by normal PR CI.

Baseline population is a deliberate implementation step. The first pass should make `coverage check` print exact unclassified observed gaps. The implementer then adds reviewed `baseline.json` entries for the current committed pins, reruns `coverage generate`, and verifies `coverage check` passes. The command must never overwrite `baseline.json`.

## File Structure

- Create: `internal/schemagen/coverage_types.go`
  - Coverage report, bucket, metadata status, target, actual, gap, and problem types shared by coverage generation, checking, and tests.
- Create: `internal/schemagen/coverage_targets.go`
  - Upstream CRD target extraction, construct inventory, local `$ref` resolution use, target metadata normalization, and unsupported-shape recording.
- Create: `internal/schemagen/coverage_actual.go`
  - Reads generated schema JSON files and classifies actual schema facts, compatibility overrides, compatibility-added fields, and compatibility-only schemas.
- Create: `internal/schemagen/coverage_baseline.go`
  - Loads `coverage/baseline.json`, validates duplicate entries, matches observed gaps, validates wildcard release entries, and reports obsolete baseline entries.
- Create: `internal/schemagen/coverage_report.go`
  - Renders deterministic `coverage/coverage.json` and `coverage/coverage.md`.
- Create: `internal/schemagen/coverage_check.go`
  - Implements `GenerateCoverage` and `CheckCoverage`, including temporary regeneration and artifact comparison.
- Create: `internal/schemagen/coverage_targets_test.go`
  - Fixture matrix for every target extraction rule in the design.
- Create: `internal/schemagen/coverage_baseline_test.go`
  - Baseline matching, duplicate rejection, wildcard, unclassified, and obsolete-entry tests.
- Create: `internal/schemagen/coverage_report_test.go`
  - Deterministic JSON and Markdown rendering tests.
- Create: `internal/schemagen/coverage_check_test.go`
  - Stale artifact and end-to-end coverage check tests.
- Create: `internal/schemagen/drift.go`
  - GitHub-backed release/tag/content drift check with injectable HTTP client and base URL.
- Create: `internal/schemagen/drift_test.go`
  - Network-free drift tests using `httptest.Server`.
- Modify: `internal/schemagen/generator.go`
  - Extend `openAPISchema` with observed OpenAPI constructs and keep schema generation behavior stable.
- Modify: `internal/schemagen/generator_test.go`
  - Keep generator unit coverage green after shared OpenAPI type changes.
- Modify: `cmd/vibe-xpls-schema-gen/main.go`
  - Add explicit subcommand parsing and compatibility alias.
- Create: `cmd/vibe-xpls-schema-gen/main_test.go`
  - CLI command-shape tests for aliases, coverage commands, and drift command validation.
- Modify: `internal/analyzer/schema_test.go`
  - Update stale-generation test to use the explicit `generate` subcommand and compare coverage artifacts.
- Modify: `scripts/update-generated.sh`
  - Run `generate`, `coverage generate`, `coverage check`, tests, and builds.
- Create: `internal/analyzer/schemadata/coverage/baseline.json`
  - Human-maintained known-gap baseline for the current committed pins.
- Create: `internal/analyzer/schemadata/coverage/coverage.json`
  - Generated deterministic machine-readable coverage report.
- Create: `internal/analyzer/schemadata/coverage/coverage.md`
  - Generated deterministic human-readable coverage report.
- Modify: `docs/generated-schemas.md`
  - Document the new subcommands, baseline workflow, and drift command.
- Modify: `PROJECT_ROADMAP.md`
  - Link the schema coverage design and implementation plan from the generated-schema roadmap area.
- Create: `.github/workflows/schema-drift.yml`
  - Scheduled networked upstream drift check.

---

### Task 1: Coverage Types And Baseline File Contract

**Files:**
- Create: `internal/schemagen/coverage_types.go`
- Create: `internal/schemagen/coverage_baseline.go`
- Create: `internal/schemagen/coverage_baseline_test.go`
- Create: `internal/analyzer/schemadata/coverage/baseline.json`

- [ ] **Step 1: Add failing baseline parsing tests**

Create `internal/schemagen/coverage_baseline_test.go`:

```go
package schemagen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCoverageBaselineRejectsDuplicateEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	raw := `{
  "formatVersion": 1,
  "entries": [
    {
      "release": "v1.20.7",
      "apiVersion": "example.org/v1",
      "kind": "Widget",
      "path": "spec.mode",
      "category": "missing-enum",
      "reason": "fixture",
      "note": "first"
    },
    {
      "release": "v1.20.7",
      "apiVersion": "example.org/v1",
      "kind": "Widget",
      "path": "spec.mode",
      "category": "missing-enum",
      "reason": "fixture",
      "note": "second"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	_, err := loadCoverageBaseline(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate baseline entry") {
		t.Fatalf("loadCoverageBaseline error = %v, want duplicate baseline entry", err)
	}
}

func TestCoverageBaselineMatchesExactAndWildcardRelease(t *testing.T) {
	baseline := coverageBaseline{
		FormatVersion: coverageFormatVersion,
		Entries: []coverageBaselineEntry{
			{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.mode",
				Category:   gapMissingEnum,
				Reason:     "current generator does not emit this enum",
				Note:       "exact release",
			},
			{
				Release:    "*",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.size",
				Category:   gapMissingDefault,
				Reason:     "current generator does not emit this default",
				Note:       "all pinned releases",
			},
		},
	}

	exact := observedGap{Release: "v1.20.7", APIVersion: "example.org/v1", Kind: "Widget", Path: "spec.mode", Category: gapMissingEnum}
	if _, ok := baseline.match(exact); !ok {
		t.Fatal("expected exact baseline match")
	}
	wildcard := observedGap{Release: "v2.2.1", APIVersion: "example.org/v1", Kind: "Widget", Path: "spec.size", Category: gapMissingDefault}
	if _, ok := baseline.match(wildcard); !ok {
		t.Fatal("expected wildcard baseline match")
	}
	miss := observedGap{Release: "v2.2.1", APIVersion: "example.org/v1", Kind: "Widget", Path: "spec.mode", Category: gapMissingEnum}
	if _, ok := baseline.match(miss); ok {
		t.Fatal("did not expect exact release entry to match another release")
	}
}

func TestCoverageBaselineWildcardMustMatchAtLeastOneObservedGap(t *testing.T) {
	baseline := coverageBaseline{
		FormatVersion: coverageFormatVersion,
		Entries: []coverageBaselineEntry{{
			Release:    "*",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.removed",
			Category:   gapMissingField,
			Reason:     "fixture",
			Note:       "stale wildcard",
		}},
	}

	problems := validateCoverageBaselineUse(baseline, nil)
	if len(problems) != 1 {
		t.Fatalf("problem count = %d, want 1", len(problems))
	}
	if !strings.Contains(problems[0].Message, "obsolete baseline entry") {
		t.Fatalf("problem = %#v, want obsolete baseline entry", problems[0])
	}
}
```

- [ ] **Step 2: Run the failing baseline tests**

Run:

```sh
go test ./internal/schemagen -run 'TestLoadCoverageBaseline|TestCoverageBaseline' -count=1
```

Expected: fail because coverage baseline types and functions do not exist.

- [ ] **Step 3: Add shared coverage types**

Create `internal/schemagen/coverage_types.go`:

```go
package schemagen

import "encoding/json"

const coverageFormatVersion = 1

type coverageBucket string

const (
	bucketCoveredUpstream          coverageBucket = "covered-upstream"
	bucketCoveredWithCompatOverride coverageBucket = "covered-with-compat-override"
	bucketCompatAddedField         coverageBucket = "compat-added-field"
	bucketCompatOnlySchema         coverageBucket = "compat-only-schema"
	bucketMissing                  coverageBucket = "missing"
	bucketExcluded                 coverageBucket = "excluded"
	bucketUnsupportedShape         coverageBucket = "unsupported-shape"
)

type metadataCoverageStatus string

const (
	metadataCovered            metadataCoverageStatus = "covered"
	metadataMissing            metadataCoverageStatus = "missing"
	metadataNotPresentUpstream metadataCoverageStatus = "not-present-upstream"
	metadataNotRequired        metadataCoverageStatus = "not-required"
)

type gapCategory string

const (
	gapExcludedResource       gapCategory = "excluded-resource"
	gapUnsupportedOpenAPIShape gapCategory = "unsupported-openapi-shape"
	gapMissingField           gapCategory = "missing-field"
	gapMissingDescription     gapCategory = "missing-description"
	gapMissingType            gapCategory = "missing-type"
	gapMissingRequired        gapCategory = "missing-required"
	gapMissingEnum            gapCategory = "missing-enum"
	gapMissingDefault         gapCategory = "missing-default"
	gapMissingDeprecation     gapCategory = "missing-deprecation"
	gapCompatAddedField       gapCategory = "compat-added-field"
	gapCompatOnlySchema       gapCategory = "compat-only-schema"
)

type coverageProblem struct {
	Message string
}

type observedGap struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
	Category   gapCategory
	Reason     string
}

type coverageTarget struct {
	Release    string
	APIVersion string
	Kind       string
	SourcePath string
	SourceSHA256 string
	Path       string
	Description string
	Type       string
	Required   bool
	Default    *json.RawMessage
	Enum       []string
	Deprecated bool
	UnsupportedReason string
}

type actualCoverageField struct {
	Path        string
	Description string
	Type        string
	Required    bool
	Default     *json.RawMessage
	Enum        []string
	Deprecated  string
	CompatOverride bool
	CompatAdded    bool
}
```

- [ ] **Step 4: Add the baseline loader and matcher**

Create `internal/schemagen/coverage_baseline.go`:

```go
package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
)

type coverageBaseline struct {
	FormatVersion int                     `json:"formatVersion"`
	Entries       []coverageBaselineEntry `json:"entries"`
}

type coverageBaselineEntry struct {
	Release    string      `json:"release"`
	APIVersion string      `json:"apiVersion,omitempty"`
	Kind       string      `json:"kind,omitempty"`
	Path       string      `json:"path,omitempty"`
	Category   gapCategory `json:"category"`
	Reason     string      `json:"reason"`
	Note       string      `json:"note"`
}

func loadCoverageBaseline(path string) (coverageBaseline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return coverageBaseline{}, fmt.Errorf("read coverage baseline: %w", err)
	}
	var baseline coverageBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return coverageBaseline{}, fmt.Errorf("parse coverage baseline: %w", err)
	}
	if baseline.FormatVersion != coverageFormatVersion {
		return coverageBaseline{}, fmt.Errorf("unsupported coverage baseline format %d", baseline.FormatVersion)
	}
	seen := map[string]struct{}{}
	for i, entry := range baseline.Entries {
		if entry.Release == "" || entry.Category == "" || entry.Reason == "" || entry.Note == "" {
			return coverageBaseline{}, fmt.Errorf("baseline entry %d is missing release, category, reason, or note", i)
		}
		key := entry.key()
		if _, ok := seen[key]; ok {
			return coverageBaseline{}, fmt.Errorf("duplicate baseline entry %s", key)
		}
		seen[key] = struct{}{}
	}
	return baseline, nil
}

func (entry coverageBaselineEntry) key() string {
	return entry.Release + "\x00" + entry.APIVersion + "\x00" + entry.Kind + "\x00" + entry.Path + "\x00" + string(entry.Category)
}

func (baseline coverageBaseline) match(gap observedGap) (coverageBaselineEntry, bool) {
	for _, entry := range baseline.Entries {
		if entry.Release != "*" && entry.Release != gap.Release {
			continue
		}
		if entry.APIVersion != gap.APIVersion || entry.Kind != gap.Kind || entry.Path != gap.Path || entry.Category != gap.Category {
			continue
		}
		return entry, true
	}
	return coverageBaselineEntry{}, false
}

func validateCoverageBaselineUse(baseline coverageBaseline, gaps []observedGap) []coverageProblem {
	matched := map[string]struct{}{}
	for _, gap := range gaps {
		if entry, ok := baseline.match(gap); ok {
			matched[entry.key()] = struct{}{}
		}
	}
	var problems []coverageProblem
	for _, entry := range baseline.Entries {
		if _, ok := matched[entry.key()]; ok {
			continue
		}
		problems = append(problems, coverageProblem{Message: fmt.Sprintf("obsolete baseline entry %s", entry.key())})
	}
	return problems
}
```

- [ ] **Step 5: Add the initial empty baseline file**

Create `internal/analyzer/schemadata/coverage/baseline.json`:

```json
{
  "formatVersion": 1,
  "entries": []
}
```

This file is human-maintained. Later tasks populate it after the coverage engine reports current observed gaps.

- [ ] **Step 6: Verify baseline tests pass**

Run:

```sh
go test ./internal/schemagen -run 'TestLoadCoverageBaseline|TestCoverageBaseline' -count=1
```

Expected: pass.

- [ ] **Step 7: Commit baseline contract**

Run:

```sh
git add internal/schemagen/coverage_types.go internal/schemagen/coverage_baseline.go internal/schemagen/coverage_baseline_test.go internal/analyzer/schemadata/coverage/baseline.json
git commit -m "feat: add schema coverage baseline contract"
```

---

### Task 2: Upstream Target Extraction And Construct Inventory

**Files:**
- Create: `internal/schemagen/coverage_targets.go`
- Create: `internal/schemagen/coverage_targets_test.go`
- Modify: `internal/schemagen/generator.go`

- [ ] **Step 1: Add failing tests for target extraction rules**

Create `internal/schemagen/coverage_targets_test.go`:

```go
package schemagen

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCoverageTargetsIncludeObservedOpenAPIConstructs(t *testing.T) {
	crdDir := writeCRDDir(t, `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.org
  names:
    kind: Widget
  scope: Namespaced
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          definitions:
            RefObject:
              type: object
              properties:
                fromRef:
                  type: string
                  description: from a local ref
          properties:
            spec:
              type: object
              required:
                - names
              properties:
                names:
                  type: array
                  description: name list
                  items:
                    type: object
                    required:
                      - value
                    properties:
                      value:
                        type: string
                        description: name value
                labels:
                  type: object
                  additionalProperties:
                    type: string
                preserved:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
                embedded:
                  type: object
                  x-kubernetes-embedded-resource: true
                  x-kubernetes-preserve-unknown-fields: true
                ref:
                  $ref: "#/definitions/RefObject"
                quantity:
                  anyOf:
                    - type: integer
                    - type: string
                  x-kubernetes-int-or-string: true
                choice:
                  oneOf:
                    - type: string
                    - type: integer
                merged:
                  allOf:
                    - type: object
                      properties:
                        child:
                          type: string
                patterned:
                  type: object
                  patternProperties:
                    "^[a-z]+$":
                      type: string
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		t.Fatalf("collectCoverageTargets: %v", err)
	}

	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "metadata.name", "string", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "metadata.namespace", "string", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.names[]", "array", true, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.names[].value", "string", true, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.labels", "object", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.preserved", "object", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.embedded", "object", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.ref.fromRef", "string", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.quantity", "int-or-string", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.choice", "", false, "oneOf")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.merged.child", "string", false, "")
	assertCoverageTarget(t, targets, "example.org/v1", "Widget", "spec.patterned", "object", false, "")
}

func TestCoverageTargetsRejectUnresolvedAndCyclicRefsAsUnsupported(t *testing.T) {
	crdDir := writeCRDDir(t, `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
spec:
  group: example.org
  names:
    kind: Broken
  scope: Cluster
  versions:
    - name: v1
      served: true
      schema:
        openAPIV3Schema:
          type: object
          definitions:
            A:
              $ref: "#/definitions/B"
            B:
              $ref: "#/definitions/A"
          properties:
            spec:
              type: object
              properties:
                missingRef:
                  $ref: "#/definitions/DoesNotExist"
                cycle:
                  $ref: "#/definitions/A"
`)
	cfg := fixtureConfig()
	cfg.Releases[0].RawCRDDir = crdDir

	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		t.Fatalf("collectCoverageTargets: %v", err)
	}

	assertCoverageTarget(t, targets, "example.org/v1", "Broken", "spec.missingRef", "", false, "unresolved local ref")
	assertCoverageTarget(t, targets, "example.org/v1", "Broken", "spec.cycle", "", false, "cyclic local ref")
}

func TestCoverageTargetsIncludeSourcePathAndSHA(t *testing.T) {
	cfg := fixtureConfig()
	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		t.Fatalf("collectCoverageTargets: %v", err)
	}
	for _, target := range targets {
		if target.SourcePath == "" || target.SourceSHA256 == "" {
			t.Fatalf("target %#v missing source path or sha", target)
		}
	}
}

func assertCoverageTarget(t *testing.T, targets []coverageTarget, apiVersion, kind, path, typ string, required bool, unsupportedContains string) {
	t.Helper()
	for _, target := range targets {
		if target.APIVersion != apiVersion || target.Kind != kind || target.Path != path {
			continue
		}
		if target.Type != typ || target.Required != required {
			t.Fatalf("%s %s %s = type %q required %v, want type %q required %v", apiVersion, kind, path, target.Type, target.Required, typ, required)
		}
		if unsupportedContains != "" && !strings.Contains(target.UnsupportedReason, unsupportedContains) {
			t.Fatalf("%s unsupported reason = %q, want containing %q", path, target.UnsupportedReason, unsupportedContains)
		}
		if unsupportedContains == "" && target.UnsupportedReason != "" {
			t.Fatalf("%s unsupported reason = %q, want empty", path, target.UnsupportedReason)
		}
		return
	}
	t.Fatalf("missing target %s %s %s", apiVersion, kind, path)
}

func TestCoverageTargetsFixtureConfigUsesConfigRelativePaths(t *testing.T) {
	t.Chdir(filepath.Join("..", ".."))
	cfg, err := LoadConfigFile(filepath.Join("internal", "schemagen", "testdata", "config.json"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := collectCoverageTargets(cfg); err != nil {
		t.Fatalf("collectCoverageTargets: %v", err)
	}
}
```

Add `strings` to the imports in the test file after adding the assertions.

- [ ] **Step 2: Run the failing target tests**

Run:

```sh
go test ./internal/schemagen -run TestCoverageTargets -count=1
```

Expected: fail because target extraction does not exist and `openAPISchema` does not yet model every construct.

- [ ] **Step 3: Extend the OpenAPI schema struct**

Modify `openAPISchema` in `internal/schemagen/generator.go` so it includes the constructs covered by the design:

```go
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
	PatternProperties          map[string]openAPISchema `yaml:"patternProperties"`
	OneOf                      []openAPISchema          `yaml:"oneOf"`
	AnyOf                      []openAPISchema          `yaml:"anyOf"`
	AllOf                      []openAPISchema          `yaml:"allOf"`
	Deprecated                 *bool                    `yaml:"deprecated"`
	XKubernetesPreserveUnknown bool                     `yaml:"x-kubernetes-preserve-unknown-fields"`
	XKubernetesEmbeddedResource bool                    `yaml:"x-kubernetes-embedded-resource"`
	XKubernetesIntOrString     bool                     `yaml:"x-kubernetes-int-or-string"`
}
```

Update `mergeSchemaOverride` to copy the new fields when an override sets them:

```go
if override.PatternProperties != nil {
	base.PatternProperties = override.PatternProperties
}
if override.OneOf != nil {
	base.OneOf = override.OneOf
}
if override.AnyOf != nil {
	base.AnyOf = override.AnyOf
}
if override.AllOf != nil {
	base.AllOf = override.AllOf
}
if override.Deprecated != nil {
	base.Deprecated = override.Deprecated
}
if override.XKubernetesEmbeddedResource {
	base.XKubernetesEmbeddedResource = override.XKubernetesEmbeddedResource
}
if override.XKubernetesIntOrString {
	base.XKubernetesIntOrString = override.XKubernetesIntOrString
}
```

- [ ] **Step 4: Add the target extractor**

Create `internal/schemagen/coverage_targets.go`:

```go
package schemagen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func collectCoverageTargets(cfg Config) ([]coverageTarget, error) {
	var targets []coverageTarget
	for _, release := range cfg.Releases {
		releaseTargets, err := collectReleaseCoverageTargets(release)
		if err != nil {
			return nil, err
		}
		targets = append(targets, releaseTargets...)
	}
	sort.Slice(targets, func(i, j int) bool {
		return coverageTargetSortKey(targets[i]) < coverageTargetSortKey(targets[j])
	})
	return targets, nil
}

func collectReleaseCoverageTargets(release ReleaseConfig) ([]coverageTarget, error) {
	crdFiles, err := yamlFiles(release.RawCRDDir)
	if err != nil {
		return nil, err
	}
	var targets []coverageTarget
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
				ctx := coverageTargetContext{
					release:    release.Tag,
					apiVersion: apiVersion,
					kind:       doc.Spec.Names.Kind,
					sourcePath: relativeSourcePath(release.RawCRDDir, path),
					sourceSHA:  sha,
					root:       version.Schema.OpenAPIV3Schema,
				}
				fields := map[string]coverageTarget{}
				addCoverageMetadataTargets(ctx, fields, doc.Spec.Scope)
				walkCoverageTargets(ctx, fields, version.Schema.OpenAPIV3Schema, "", nil, nil)
				for _, target := range fields {
					targets = append(targets, target)
				}
			}
		}
	}
	return targets, nil
}

type coverageTargetContext struct {
	release    string
	apiVersion string
	kind       string
	sourcePath string
	sourceSHA  string
	root       openAPISchema
}

func walkCoverageTargets(ctx coverageTargetContext, fields map[string]coverageTarget, schema openAPISchema, prefix string, required []string, seenRefs map[string]struct{}) {
	resolved, err := resolveSchema(ctx.root, schema, seenRefs)
	if err != nil {
		putUnsupportedCoverageTarget(ctx, fields, prefix, schema, err.Error())
		return
	}
	schema = resolved
	if schema.XKubernetesIntOrString {
		schema.Type = "int-or-string"
	}
	if len(schema.OneOf) != 0 {
		putUnsupportedCoverageTarget(ctx, fields, prefix, schema, "oneOf")
		return
	}
	if len(schema.AnyOf) != 0 && !schema.XKubernetesIntOrString {
		putUnsupportedCoverageTarget(ctx, fields, prefix, schema, "anyOf")
		return
	}
	if len(schema.AllOf) != 0 {
		for _, item := range schema.AllOf {
			walkCoverageTargets(ctx, fields, mergeSchemaOverride(schema, item), prefix, required, seenRefs)
		}
		return
	}
	if schema.Type == "array" && schema.Items != nil {
		arrayPath := prefix + "[]"
		putCoverageTarget(ctx, fields, arrayPath, schema, isRequired(required, lastPathSegment(prefix)))
		item, err := resolveSchema(ctx.root, *schema.Items, seenRefs)
		if err != nil {
			putUnsupportedCoverageTarget(ctx, fields, arrayPath, *schema.Items, err.Error())
			return
		}
		walkCoverageTargets(ctx, fields, item, arrayPath, item.Required, seenRefs)
		return
	}
	if schema.AdditionalProperties != nil || schema.XKubernetesPreserveUnknown || len(schema.PatternProperties) != 0 || schema.XKubernetesEmbeddedResource {
		putCoverageTarget(ctx, fields, prefix, schema, isRequired(required, lastPathSegment(prefix)))
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child := schema.Properties[name]
		path := joinPath(prefix, name)
		childRequired := isRequired(schema.Required, name)
		putCoverageTarget(ctx, fields, path, child, childRequired)
		walkCoverageTargets(ctx, fields, child, path, child.Required, seenRefs)
	}
}

func putCoverageTarget(ctx coverageTargetContext, fields map[string]coverageTarget, path string, schema openAPISchema, required bool) {
	if path == "" {
		return
	}
	targetType := schema.Type
	if schema.XKubernetesIntOrString {
		targetType = "int-or-string"
	}
	deprecated := false
	if schema.Deprecated != nil {
		deprecated = *schema.Deprecated
	}
	fields[path] = coverageTarget{
		Release:     ctx.release,
		APIVersion:  ctx.apiVersion,
		Kind:        ctx.kind,
		SourcePath:  ctx.sourcePath,
		SourceSHA256: ctx.sourceSHA,
		Path:        path,
		Description: schema.Description,
		Type:        targetType,
		Required:    required,
		Default:     rawDefault(schema.Default),
		Enum:        enumStrings(schema.Enum),
		Deprecated:  deprecated,
	}
}

func putUnsupportedCoverageTarget(ctx coverageTargetContext, fields map[string]coverageTarget, path string, schema openAPISchema, reason string) {
	if path == "" {
		return
	}
	target := coverageTarget{
		Release:     ctx.release,
		APIVersion:  ctx.apiVersion,
		Kind:        ctx.kind,
		SourcePath:  ctx.sourcePath,
		SourceSHA256: ctx.sourceSHA,
		Path:        path,
		Description: schema.Description,
		Type:        schema.Type,
		UnsupportedReason: reason,
	}
	fields[path] = target
}

func addCoverageMetadataTargets(ctx coverageTargetContext, fields map[string]coverageTarget, scope string) {
	putCoverageTarget(ctx, fields, "metadata.name", openAPISchema{Type: "string", Description: "Object name."}, false)
	putCoverageTarget(ctx, fields, "metadata.labels", openAPISchema{Type: "object", Description: "Object labels."}, false)
	putCoverageTarget(ctx, fields, "metadata.annotations", openAPISchema{Type: "object", Description: "Object annotations."}, false)
	if scope == "Namespaced" {
		putCoverageTarget(ctx, fields, "metadata.namespace", openAPISchema{Type: "string", Description: "Object namespace."}, false)
	}
}

func coverageTargetSortKey(target coverageTarget) string {
	return strings.Join([]string{target.Release, target.APIVersion, target.Kind, target.Path}, "\x00")
}

func sourcePathForDebug(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func unsupportedConstruct(path, construct string) error {
	return fmt.Errorf("%s at %s", construct, path)
}
```

After adding the file, remove `sourcePathForDebug` and `unsupportedConstruct` if they are unused by the compiler. Keep only helpers that compile.

- [ ] **Step 5: Fix imports and run target tests**

Run:

```sh
go test ./internal/schemagen -run TestCoverageTargets -count=1
```

Expected: pass.

- [ ] **Step 6: Run generator tests to catch shared OpenAPI regressions**

Run:

```sh
go test ./internal/schemagen -count=1
```

Expected: pass.

- [ ] **Step 7: Commit target extraction**

Run:

```sh
git add internal/schemagen/generator.go internal/schemagen/coverage_targets.go internal/schemagen/coverage_targets_test.go
git commit -m "feat: extract schema coverage targets"
```

---

### Task 3: Actual Schema Fact Collection And Coverage Buckets

**Files:**
- Create: `internal/schemagen/coverage_actual.go`
- Modify: `internal/schemagen/coverage_types.go`
- Create: `internal/schemagen/coverage_report_test.go`

- [ ] **Step 1: Add failing bucket classification tests**

Create `internal/schemagen/coverage_report_test.go` with the first actual-vs-target test:

```go
package schemagen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/io41/vibe-xpls/internal/analyzer"
)

func TestCoverageBucketsSeparateUpstreamCompatAndMissingFields(t *testing.T) {
	out := t.TempDir()
	releaseDir := filepath.Join(out, "schemas", "v1.20.7")
	if err := os.MkdirAll(releaseDir, 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	schema := schemaDocumentJSON{
		Release:    "v1.20.7",
		APIVersion: "apiextensions.crossplane.io/v1",
		Kind:       "Composition",
		Fields: []fieldDocJSON{
			{Path: "spec.present", Description: "upstream", Type: "string"},
			{Path: "spec.compositeTypeRef.kind", Description: "Kind of the composite resource type this Composition renders.", Type: "string"},
			{Path: "spec.compatOnly", Description: "compat field", Type: "string"},
		},
		Provenance: schemaProvenanceJSON{
			Source:             string(analyzer.SchemaSourceGeneratedBuiltIn),
			UpstreamReleaseTag: "v1.20.7",
			UpstreamSourcePath: "crds/widgets.yaml",
		},
	}
	if err := writeJSON(filepath.Join(releaseDir, "apiextensions.crossplane.io_v1_Composition.json"), schema); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	targets := []coverageTarget{
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.present", Description: "upstream", Type: "string"},
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.compositeTypeRef.kind", Description: "Kind of the type.", Type: "string"},
		{Release: "v1.20.7", APIVersion: "apiextensions.crossplane.io/v1", Kind: "Composition", Path: "spec.missing", Description: "lost", Type: "string"},
	}

	actual, err := collectActualCoverageFields(out)
	if err != nil {
		t.Fatalf("collectActualCoverageFields: %v", err)
	}
	state := computeCoverageState(targets, actual, coverageBaseline{FormatVersion: coverageFormatVersion})

	assertCoverageFieldBucket(t, state, "spec.present", bucketCoveredUpstream)
	assertCoverageFieldBucket(t, state, "spec.compositeTypeRef.kind", bucketCoveredWithCompatOverride)
	assertCoverageFieldBucket(t, state, "spec.compatOnly", bucketCompatAddedField)
	assertCoverageFieldBucket(t, state, "spec.missing", bucketMissing)
}
```

- [ ] **Step 2: Run the failing bucket test**

Run:

```sh
go test ./internal/schemagen -run TestCoverageBucketsSeparateUpstreamCompatAndMissingFields -count=1
```

Expected: fail because actual fact collection and coverage state computation do not exist.

- [ ] **Step 3: Add report field state types**

Extend `internal/schemagen/coverage_types.go`:

```go
type coverageFieldState struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
	Bucket     coverageBucket
	Metadata   coverageMetadataState
	Gap        *observedGap
}

type coverageMetadataState struct {
	Description metadataCoverageStatus `json:"description,omitempty"`
	Type        metadataCoverageStatus `json:"type,omitempty"`
	Required    metadataCoverageStatus `json:"required,omitempty"`
	Enum        metadataCoverageStatus `json:"enum,omitempty"`
	Default     metadataCoverageStatus `json:"default,omitempty"`
	Deprecated  metadataCoverageStatus `json:"deprecated,omitempty"`
}

type coverageGVKState struct {
	Release    string
	APIVersion string
	Kind       string
	SourcePath string
	SourceSHA256 string
	Fields     []coverageFieldState
	Buckets    map[coverageBucket]int
}

type coverageState struct {
	GVKs []coverageGVKState
	Gaps []observedGap
}
```

- [ ] **Step 4: Add actual fact collection**

Create `internal/schemagen/coverage_actual.go`:

```go
package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type actualCoverageKey struct {
	Release    string
	APIVersion string
	Kind       string
	Path       string
}

func collectActualCoverageFields(outDir string) (map[actualCoverageKey]actualCoverageField, error) {
	root := filepath.Join(outDir, "schemas")
	actual := map[actualCoverageKey]actualCoverageField{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read schema %s: %w", path, err)
		}
		var doc schemaDocumentJSON
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("parse schema %s: %w", path, err)
		}
		overrides := compatibilityFieldDocs(doc.APIVersion, doc.Kind)
		for _, field := range doc.Fields {
			override, hasOverride := overrides[field.Path]
			actual[actualCoverageKey{Release: doc.Release, APIVersion: doc.APIVersion, Kind: doc.Kind, Path: field.Path}] = actualCoverageField{
				Path:           field.Path,
				Description:    field.Description,
				Type:           field.Type,
				Required:       field.Required,
				Default:        field.Default,
				Enum:           append([]string(nil), field.Enum...),
				Deprecated:     field.Deprecated,
				CompatOverride: hasOverride && override.Description != "" && override.Description == field.Description,
			}
		}
		if strings.Contains(doc.Provenance.UpstreamSourcePath, "generated/compatibility/") {
			for _, field := range doc.Fields {
				key := actualCoverageKey{Release: doc.Release, APIVersion: doc.APIVersion, Kind: doc.Kind, Path: field.Path}
				value := actual[key]
				value.CompatAdded = true
				actual[key] = value
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return actual, nil
}

func sortedActualKeys(actual map[actualCoverageKey]actualCoverageField) []actualCoverageKey {
	keys := make([]actualCoverageKey, 0, len(actual))
	for key := range actual {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.Join([]string{keys[i].Release, keys[i].APIVersion, keys[i].Kind, keys[i].Path}, "\x00") <
			strings.Join([]string{keys[j].Release, keys[j].APIVersion, keys[j].Kind, keys[j].Path}, "\x00")
	})
	return keys
}
```

- [ ] **Step 5: Add coverage state computation**

Create `computeCoverageState` in `internal/schemagen/coverage_report.go`:

```go
package schemagen

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

func computeCoverageState(targets []coverageTarget, actual map[actualCoverageKey]actualCoverageField, baseline coverageBaseline) coverageState {
	gvkMap := map[string]*coverageGVKState{}
	var gaps []observedGap
	seenTargets := map[actualCoverageKey]struct{}{}
	for _, target := range targets {
		key := actualCoverageKey{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, Path: target.Path}
		seenTargets[key] = struct{}{}
		field := coverageFieldState{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, Path: target.Path}
		if target.UnsupportedReason != "" {
			field.Bucket = bucketUnsupportedShape
			gap := observedGap{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, Path: target.Path, Category: gapUnsupportedOpenAPIShape, Reason: target.UnsupportedReason}
			field.Gap = &gap
			gaps = append(gaps, gap)
		} else if got, ok := actual[key]; ok {
			field.Metadata = compareMetadata(target, got)
			if got.CompatOverride {
				field.Bucket = bucketCoveredWithCompatOverride
			} else {
				field.Bucket = bucketCoveredUpstream
			}
			gaps = append(gaps, metadataGaps(target, got)...)
		} else {
			field.Bucket = bucketMissing
			gap := observedGap{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, Path: target.Path, Category: gapMissingField, Reason: "target field is absent from generated schema"}
			field.Gap = &gap
			gaps = append(gaps, gap)
		}
		addCoverageField(gvkMap, target, field)
	}
	for _, key := range sortedActualKeys(actual) {
		if _, ok := seenTargets[key]; ok {
			continue
		}
		got := actual[key]
		target := coverageTarget{Release: key.Release, APIVersion: key.APIVersion, Kind: key.Kind, Path: key.Path}
		bucket := bucketCompatAddedField
		category := gapCompatAddedField
		if got.CompatAdded {
			bucket = bucketCompatOnlySchema
			category = gapCompatOnlySchema
		}
		gap := observedGap{Release: key.Release, APIVersion: key.APIVersion, Kind: key.Kind, Path: key.Path, Category: category, Reason: "generated compatibility field has no upstream CRD target"}
		field := coverageFieldState{Release: key.Release, APIVersion: key.APIVersion, Kind: key.Kind, Path: key.Path, Bucket: bucket, Gap: &gap}
		gaps = append(gaps, gap)
		addCoverageField(gvkMap, target, field)
	}
	out := coverageState{Gaps: gaps}
	for _, gvk := range gvkMap {
		sort.Slice(gvk.Fields, func(i, j int) bool { return gvk.Fields[i].Path < gvk.Fields[j].Path })
		gvk.Buckets = map[coverageBucket]int{}
		for _, field := range gvk.Fields {
			gvk.Buckets[field.Bucket]++
		}
		out.GVKs = append(out.GVKs, *gvk)
	}
	sort.Slice(out.GVKs, func(i, j int) bool {
		left := strings.Join([]string{out.GVKs[i].Release, out.GVKs[i].APIVersion, out.GVKs[i].Kind}, "\x00")
		right := strings.Join([]string{out.GVKs[j].Release, out.GVKs[j].APIVersion, out.GVKs[j].Kind}, "\x00")
		return left < right
	})
	return out
}

func addCoverageField(gvks map[string]*coverageGVKState, target coverageTarget, field coverageFieldState) {
	key := strings.Join([]string{target.Release, target.APIVersion, target.Kind}, "\x00")
	gvk := gvks[key]
	if gvk == nil {
		gvk = &coverageGVKState{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, SourcePath: target.SourcePath, SourceSHA256: target.SourceSHA256}
		gvks[key] = gvk
	}
	gvk.Fields = append(gvk.Fields, field)
}

func compareMetadata(target coverageTarget, actual actualCoverageField) coverageMetadataState {
	return coverageMetadataState{
		Description: compareStringMetadata(target.Description, actual.Description),
		Type:        compareStringMetadata(target.Type, actual.Type),
		Required:    compareRequiredMetadata(target.Required, actual.Required),
		Enum:        compareSliceMetadata(target.Enum, actual.Enum),
		Default:     compareRawJSONMetadata(target.Default, actual.Default),
		Deprecated:  compareDeprecatedMetadata(target.Deprecated, actual.Deprecated),
	}
}

func compareStringMetadata(want, got string) metadataCoverageStatus {
	want = strings.TrimSpace(want)
	got = strings.TrimSpace(got)
	if want == "" {
		return metadataNotPresentUpstream
	}
	if want == got {
		return metadataCovered
	}
	return metadataMissing
}

func compareRequiredMetadata(want, got bool) metadataCoverageStatus {
	if !want {
		return metadataNotRequired
	}
	if got {
		return metadataCovered
	}
	return metadataMissing
}

func compareSliceMetadata(want, got []string) metadataCoverageStatus {
	if len(want) == 0 {
		return metadataNotPresentUpstream
	}
	if reflect.DeepEqual(want, got) {
		return metadataCovered
	}
	return metadataMissing
}

func compareRawJSONMetadata(want, got *json.RawMessage) metadataCoverageStatus {
	if want == nil {
		return metadataNotPresentUpstream
	}
	if got == nil {
		return metadataMissing
	}
	var wantBuf bytes.Buffer
	var gotBuf bytes.Buffer
	if json.Compact(&wantBuf, *want) != nil || json.Compact(&gotBuf, *got) != nil {
		if strings.TrimSpace(string(*want)) == strings.TrimSpace(string(*got)) {
			return metadataCovered
		}
		return metadataMissing
	}
	if wantBuf.String() == gotBuf.String() {
		return metadataCovered
	}
	return metadataMissing
}

func compareDeprecatedMetadata(want bool, got string) metadataCoverageStatus {
	if !want {
		return metadataNotPresentUpstream
	}
	if strings.TrimSpace(got) != "" {
		return metadataCovered
	}
	return metadataMissing
}

func metadataGaps(target coverageTarget, actual actualCoverageField) []observedGap {
	state := compareMetadata(target, actual)
	var gaps []observedGap
	add := func(status metadataCoverageStatus, category gapCategory, reason string) {
		if status == metadataMissing {
			gaps = append(gaps, observedGap{Release: target.Release, APIVersion: target.APIVersion, Kind: target.Kind, Path: target.Path, Category: category, Reason: reason})
		}
	}
	add(state.Description, gapMissingDescription, "upstream description is absent or changed in generated schema")
	add(state.Type, gapMissingType, "upstream type is absent or changed in generated schema")
	add(state.Required, gapMissingRequired, "upstream required marker is absent in generated schema")
	add(state.Enum, gapMissingEnum, "upstream enum set is absent or changed in generated schema")
	add(state.Default, gapMissingDefault, "upstream default is absent or changed in generated schema")
	add(state.Deprecated, gapMissingDeprecation, "upstream deprecation metadata is absent in generated schema")
	return gaps
}
```

- [ ] **Step 6: Add the test assertion helper**

Add this helper to `internal/schemagen/coverage_report_test.go`:

```go
func assertCoverageFieldBucket(t *testing.T, state coverageState, path string, bucket coverageBucket) {
	t.Helper()
	for _, gvk := range state.GVKs {
		for _, field := range gvk.Fields {
			if field.Path == path {
				if field.Bucket != bucket {
					t.Fatalf("%s bucket = %s, want %s", path, field.Bucket, bucket)
				}
				return
			}
		}
	}
	t.Fatalf("missing coverage field %s", path)
}
```

- [ ] **Step 7: Verify bucket classification**

Run:

```sh
go test ./internal/schemagen -run TestCoverageBucketsSeparateUpstreamCompatAndMissingFields -count=1
```

Expected: pass after import fixes.

- [ ] **Step 8: Commit actual coverage facts**

Run:

```sh
git add internal/schemagen/coverage_types.go internal/schemagen/coverage_actual.go internal/schemagen/coverage_report.go internal/schemagen/coverage_report_test.go
git commit -m "feat: classify generated schema coverage"
```

---

### Task 4: Baseline Application And Ratchet Validation

**Files:**
- Modify: `internal/schemagen/coverage_baseline.go`
- Modify: `internal/schemagen/coverage_baseline_test.go`
- Modify: `internal/schemagen/coverage_report.go`

- [ ] **Step 1: Add failing ratchet validation tests**

Append these tests to `internal/schemagen/coverage_baseline_test.go`:

```go
func TestValidateCoverageRatchetFailsForUnclassifiedGap(t *testing.T) {
	state := coverageState{Gaps: []observedGap{{
		Release:    "v1.20.7",
		APIVersion: "example.org/v1",
		Kind:       "Widget",
		Path:       "spec.mode",
		Category:   gapMissingField,
		Reason:     "field absent",
	}}}

	problems := validateCoverageRatchet(state, coverageBaseline{FormatVersion: coverageFormatVersion})
	if len(problems) != 1 {
		t.Fatalf("problem count = %d, want 1", len(problems))
	}
	if !strings.Contains(problems[0].Message, "unclassified coverage gap") {
		t.Fatalf("problem = %#v, want unclassified coverage gap", problems[0])
	}
}

func TestValidateCoverageRatchetPassesForClassifiedGap(t *testing.T) {
	state := coverageState{Gaps: []observedGap{{
		Release:    "v1.20.7",
		APIVersion: "example.org/v1",
		Kind:       "Widget",
		Path:       "spec.mode",
		Category:   gapMissingField,
		Reason:     "field absent",
	}}}
	baseline := coverageBaseline{
		FormatVersion: coverageFormatVersion,
		Entries: []coverageBaselineEntry{{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.mode",
			Category:   gapMissingField,
			Reason:     "current generator omits this fixture field",
			Note:       "test fixture",
		}},
	}

	if problems := validateCoverageRatchet(state, baseline); len(problems) != 0 {
		t.Fatalf("problems = %#v, want none", problems)
	}
}

func TestValidateCoverageRatchetFailsForObsoleteBaselineEntry(t *testing.T) {
	baseline := coverageBaseline{
		FormatVersion: coverageFormatVersion,
		Entries: []coverageBaselineEntry{{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.removed",
			Category:   gapMissingField,
			Reason:     "field used to be absent",
			Note:       "stale fixture",
		}},
	}

	problems := validateCoverageRatchet(coverageState{}, baseline)
	if len(problems) != 1 {
		t.Fatalf("problem count = %d, want 1", len(problems))
	}
	if !strings.Contains(problems[0].Message, "obsolete baseline entry") {
		t.Fatalf("problem = %#v, want obsolete baseline entry", problems[0])
	}
}
```

- [ ] **Step 2: Run the failing ratchet tests**

Run:

```sh
go test ./internal/schemagen -run TestValidateCoverageRatchet -count=1
```

Expected: fail because `validateCoverageRatchet` does not exist.

- [ ] **Step 3: Implement ratchet validation**

Add to `internal/schemagen/coverage_baseline.go`:

```go
func validateCoverageRatchet(state coverageState, baseline coverageBaseline) []coverageProblem {
	var problems []coverageProblem
	for _, gap := range state.Gaps {
		if _, ok := baseline.match(gap); ok {
			continue
		}
		problems = append(problems, coverageProblem{Message: fmt.Sprintf(
			"unclassified coverage gap release=%s apiVersion=%s kind=%s path=%s category=%s reason=%s",
			gap.Release, gap.APIVersion, gap.Kind, gap.Path, gap.Category, gap.Reason,
		)})
	}
	problems = append(problems, validateCoverageBaselineUse(baseline, state.Gaps)...)
	return problems
}
```

- [ ] **Step 4: Mark baseline-classified field states as excluded**

In `computeCoverageState`, after each gap is created and before appending the field, check whether it matches the baseline:

```go
if field.Gap != nil {
	if _, ok := baseline.match(*field.Gap); ok {
		field.Bucket = bucketExcluded
	}
}
```

Apply that logic in the missing-field, unsupported-shape, and compatibility-only branches. Metadata-only gaps remain recorded in `state.Gaps`; the field bucket remains the field-level bucket because the field itself may be present.

- [ ] **Step 5: Verify baseline and bucket tests**

Run:

```sh
go test ./internal/schemagen -run 'TestValidateCoverageRatchet|TestCoverageBuckets' -count=1
```

Expected: pass.

- [ ] **Step 6: Commit ratchet validation**

Run:

```sh
git add internal/schemagen/coverage_baseline.go internal/schemagen/coverage_baseline_test.go internal/schemagen/coverage_report.go
git commit -m "feat: enforce schema coverage baseline"
```

---

### Task 5: Deterministic Coverage Artifacts

**Files:**
- Modify: `internal/schemagen/coverage_report.go`
- Modify: `internal/schemagen/coverage_report_test.go`

- [ ] **Step 1: Add failing deterministic report tests**

Append to `internal/schemagen/coverage_report_test.go`:

```go
func TestRenderCoverageJSONIsDeterministic(t *testing.T) {
	state := coverageState{GVKs: []coverageGVKState{{
		Release:    "v1.20.7",
		APIVersion: "example.org/v1",
		Kind:       "Widget",
		SourcePath: "crds/widgets.yaml",
		SourceSHA256: "abc123",
		Buckets: map[coverageBucket]int{
			bucketMissing:         1,
			bucketCoveredUpstream: 2,
		},
		Fields: []coverageFieldState{
			{Release: "v1.20.7", APIVersion: "example.org/v1", Kind: "Widget", Path: "spec.missing", Bucket: bucketMissing},
			{Release: "v1.20.7", APIVersion: "example.org/v1", Kind: "Widget", Path: "spec.present", Bucket: bucketCoveredUpstream, Metadata: coverageMetadataState{Description: metadataCovered, Type: metadataCovered, Required: metadataNotRequired, Enum: metadataNotPresentUpstream, Default: metadataNotPresentUpstream, Deprecated: metadataNotPresentUpstream}},
		},
	}}}

	got, err := renderCoverageJSON(state)
	if err != nil {
		t.Fatalf("renderCoverageJSON: %v", err)
	}
	want := `{
  "formatVersion": 1,
  "releases": [
    {
      "tag": "v1.20.7",
      "totals": {
        "upstreamGVKs": 1,
        "generatedGVKs": 1,
        "targetFields": 2,
        "coveredUpstreamFields": 1,
        "knownGaps": 1
      },
      "gvks": [
        {
          "apiVersion": "example.org/v1",
          "kind": "Widget",
          "sourcePath": "crds/widgets.yaml",
          "sourceSHA256": "abc123",
          "buckets": {
            "covered-upstream": 1,
            "missing": 1
          },
          "fields": [
            {
              "path": "spec.missing",
              "bucket": "missing",
              "metadata": {}
            },
            {
              "path": "spec.present",
              "bucket": "covered-upstream",
              "metadata": {
                "description": "covered",
                "type": "covered",
                "required": "not-required",
                "enum": "not-present-upstream",
                "default": "not-present-upstream",
                "deprecated": "not-present-upstream"
              }
            }
          ]
        }
      ]
    }
  ]
}
`
	if string(got) != want {
		t.Fatalf("coverage json:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderCoverageMarkdownSummarizesWorstGVKs(t *testing.T) {
	state := coverageState{GVKs: []coverageGVKState{{
		Release:    "v1.20.7",
		APIVersion: "example.org/v1",
		Kind:       "Widget",
		Buckets: map[coverageBucket]int{
			bucketCoveredUpstream: 1,
			bucketMissing:         1,
		},
		Fields: []coverageFieldState{
			{Path: "spec.present", Bucket: bucketCoveredUpstream},
			{Path: "spec.missing", Bucket: bucketMissing},
		},
	}}}

	got := renderCoverageMarkdown(state)
	for _, want := range []string{
		"# Schema Coverage",
		"## Release v1.20.7",
		"Upstream field coverage: 1/2 (50.00%)",
		"| example.org/v1 | Widget | 50.00% | 1 |",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run the failing report tests**

Run:

```sh
go test ./internal/schemagen -run 'TestRenderCoverageJSON|TestRenderCoverageMarkdown' -count=1
```

Expected: fail because renderers do not exist.

- [ ] **Step 3: Add coverage JSON renderer**

Implement `renderCoverageJSON` in `internal/schemagen/coverage_report.go` using explicit DTO structs with stable ordering:

```go
type coverageJSON struct {
	FormatVersion int                   `json:"formatVersion"`
	Releases      []coverageJSONRelease `json:"releases"`
}

type coverageJSONRelease struct {
	Tag    string            `json:"tag"`
	Totals coverageJSONTotals `json:"totals"`
	GVKs   []coverageJSONGVK  `json:"gvks"`
}

type coverageJSONTotals struct {
	UpstreamGVKs          int `json:"upstreamGVKs"`
	GeneratedGVKs         int `json:"generatedGVKs"`
	TargetFields          int `json:"targetFields"`
	CoveredUpstreamFields  int `json:"coveredUpstreamFields"`
	KnownGaps             int `json:"knownGaps"`
}

type coverageJSONGVK struct {
	APIVersion  string                    `json:"apiVersion"`
	Kind        string                    `json:"kind"`
	SourcePath  string                    `json:"sourcePath,omitempty"`
	SourceSHA256 string                   `json:"sourceSHA256,omitempty"`
	Buckets     map[coverageBucket]int    `json:"buckets"`
	Fields      []coverageJSONField       `json:"fields"`
}

type coverageJSONField struct {
	Path     string                `json:"path"`
	Bucket   coverageBucket        `json:"bucket"`
	Metadata coverageMetadataState `json:"metadata"`
}

func renderCoverageJSON(state coverageState) ([]byte, error) {
	doc := coverageJSON{FormatVersion: coverageFormatVersion}
	byRelease := map[string][]coverageGVKState{}
	for _, gvk := range state.GVKs {
		byRelease[gvk.Release] = append(byRelease[gvk.Release], gvk)
	}
	releases := make([]string, 0, len(byRelease))
	for release := range byRelease {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	for _, release := range releases {
		item := coverageJSONRelease{Tag: release}
		gvks := byRelease[release]
		sort.Slice(gvks, func(i, j int) bool {
			return gvks[i].APIVersion+"\x00"+gvks[i].Kind < gvks[j].APIVersion+"\x00"+gvks[j].Kind
		})
		for _, gvk := range gvks {
			jsonGVK := coverageJSONGVK{
				APIVersion:  gvk.APIVersion,
				Kind:        gvk.Kind,
				SourcePath:  gvk.SourcePath,
				SourceSHA256: gvk.SourceSHA256,
				Buckets:     sortedBucketMap(gvk.Buckets),
			}
			item.Totals.UpstreamGVKs++
			item.Totals.GeneratedGVKs++
			for _, field := range gvk.Fields {
				jsonGVK.Fields = append(jsonGVK.Fields, coverageJSONField{Path: field.Path, Bucket: field.Bucket, Metadata: field.Metadata})
				if field.Bucket != bucketCompatAddedField && field.Bucket != bucketCompatOnlySchema {
					item.Totals.TargetFields++
				}
				if field.Bucket == bucketCoveredUpstream || field.Bucket == bucketCoveredWithCompatOverride {
					item.Totals.CoveredUpstreamFields++
				}
				if field.Bucket == bucketMissing || field.Bucket == bucketExcluded || field.Bucket == bucketUnsupportedShape {
					item.Totals.KnownGaps++
				}
			}
			item.GVKs = append(item.GVKs, jsonGVK)
		}
		doc.Releases = append(doc.Releases, item)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func sortedBucketMap(input map[coverageBucket]int) map[coverageBucket]int {
	out := map[coverageBucket]int{}
	for _, bucket := range []coverageBucket{
		bucketCoveredUpstream,
		bucketCoveredWithCompatOverride,
		bucketCompatAddedField,
		bucketCompatOnlySchema,
		bucketMissing,
		bucketExcluded,
		bucketUnsupportedShape,
	} {
		if count := input[bucket]; count != 0 {
			out[bucket] = count
		}
	}
	return out
}
```

- [ ] **Step 4: Add Markdown renderer**

Add to `internal/schemagen/coverage_report.go`:

```go
func renderCoverageMarkdown(state coverageState) string {
	var b strings.Builder
	b.WriteString("# Schema Coverage\n\n")
	byRelease := map[string][]coverageGVKState{}
	for _, gvk := range state.GVKs {
		byRelease[gvk.Release] = append(byRelease[gvk.Release], gvk)
	}
	releases := make([]string, 0, len(byRelease))
	for release := range byRelease {
		releases = append(releases, release)
	}
	sort.Strings(releases)
	for _, release := range releases {
		gvks := byRelease[release]
		covered, total, gaps := coverageCounts(gvks)
		b.WriteString("## Release " + release + "\n\n")
		b.WriteString(fmt.Sprintf("Upstream field coverage: %d/%d (%.2f%%)\n\n", covered, total, percent(covered, total)))
		b.WriteString(fmt.Sprintf("Known gaps: %d\n\n", gaps))
		b.WriteString("### Worst-Covered GVKs\n\n")
		b.WriteString("| API Version | Kind | Coverage | Known Gaps |\n")
		b.WriteString("| --- | --- | ---: | ---: |\n")
		sort.Slice(gvks, func(i, j int) bool {
			leftCovered, leftTotal, _ := coverageCounts([]coverageGVKState{gvks[i]})
			rightCovered, rightTotal, _ := coverageCounts([]coverageGVKState{gvks[j]})
			leftPct := percent(leftCovered, leftTotal)
			rightPct := percent(rightCovered, rightTotal)
			if leftPct == rightPct {
				return gvks[i].APIVersion+"\x00"+gvks[i].Kind < gvks[j].APIVersion+"\x00"+gvks[j].Kind
			}
			return leftPct < rightPct
		})
		limit := len(gvks)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			gvkCovered, gvkTotal, gvkGaps := coverageCounts([]coverageGVKState{gvks[i]})
			b.WriteString(fmt.Sprintf("| %s | %s | %.2f%% | %d |\n", gvks[i].APIVersion, gvks[i].Kind, percent(gvkCovered, gvkTotal), gvkGaps))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func coverageCounts(gvks []coverageGVKState) (covered int, total int, gaps int) {
	for _, gvk := range gvks {
		for _, field := range gvk.Fields {
			switch field.Bucket {
			case bucketCompatAddedField, bucketCompatOnlySchema:
				continue
			case bucketCoveredUpstream, bucketCoveredWithCompatOverride:
				covered++
				total++
			case bucketMissing, bucketExcluded, bucketUnsupportedShape:
				total++
				gaps++
			default:
				total++
			}
		}
	}
	return covered, total, gaps
}

func percent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 100
	}
	return float64(numerator) * 100 / float64(denominator)
}
```

Add `fmt` to the imports for `coverage_report.go`.

- [ ] **Step 5: Verify deterministic report tests**

Run:

```sh
go test ./internal/schemagen -run 'TestRenderCoverageJSON|TestRenderCoverageMarkdown' -count=1
```

Expected: pass after import fixes.

- [ ] **Step 6: Commit report rendering**

Run:

```sh
git add internal/schemagen/coverage_report.go internal/schemagen/coverage_report_test.go
git commit -m "feat: render schema coverage artifacts"
```

---

### Task 6: Coverage Generation And Check Commands

**Files:**
- Create: `internal/schemagen/coverage_check.go`
- Create: `internal/schemagen/coverage_check_test.go`
- Modify: `cmd/vibe-xpls-schema-gen/main.go`
- Create: `cmd/vibe-xpls-schema-gen/main_test.go`
- Modify: `internal/analyzer/schema_test.go`

- [ ] **Step 1: Add failing coverage check tests**

Create `internal/schemagen/coverage_check_test.go`:

```go
package schemagen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCoverageWritesArtifactsWithoutChangingBaseline(t *testing.T) {
	out := t.TempDir()
	cfg := fixtureConfig()
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("generate schemas: %v", err)
	}
	coverageDir := filepath.Join(out, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("mkdir coverage: %v", err)
	}
	baselinePath := filepath.Join(coverageDir, "baseline.json")
	baseline := "{\n  \"formatVersion\": 1,\n  \"entries\": []\n}\n"
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}

	if err := GenerateCoverage(cfg, out); err != nil {
		t.Fatalf("GenerateCoverage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(coverageDir, "coverage.json")); err != nil {
		t.Fatalf("coverage json stat: %v", err)
	}
	if _, err := os.Stat(filepath.Join(coverageDir, "coverage.md")); err != nil {
		t.Fatalf("coverage md stat: %v", err)
	}
	got, err := os.ReadFile(baselinePath)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	if string(got) != baseline {
		t.Fatalf("baseline changed:\n%s", got)
	}
}

func TestCheckCoverageFailsForStaleCoverageArtifact(t *testing.T) {
	out := t.TempDir()
	cfg := fixtureConfig()
	if err := Generate(cfg, out); err != nil {
		t.Fatalf("generate schemas: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(out, "coverage"), 0o755); err != nil {
		t.Fatalf("mkdir coverage: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "coverage", "baseline.json"), []byte("{\n  \"formatVersion\": 1,\n  \"entries\": []\n}\n"), 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := GenerateCoverage(cfg, out); err != nil {
		t.Fatalf("GenerateCoverage: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "coverage", "coverage.md"), []byte("# stale\n"), 0o644); err != nil {
		t.Fatalf("write stale coverage: %v", err)
	}

	err := CheckCoverage(cfg, out)
	if err == nil || !strings.Contains(err.Error(), "coverage/coverage.md is stale") {
		t.Fatalf("CheckCoverage error = %v, want stale coverage.md", err)
	}
}
```

- [ ] **Step 2: Run failing coverage check tests**

Run:

```sh
go test ./internal/schemagen -run 'TestGenerateCoverage|TestCheckCoverage' -count=1
```

Expected: fail because `GenerateCoverage` and `CheckCoverage` do not exist.

- [ ] **Step 3: Implement generation and check orchestration**

Create `internal/schemagen/coverage_check.go`:

```go
package schemagen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateCoverage(cfg Config, outDir string) error {
	baseline, err := loadCoverageBaseline(filepath.Join(outDir, "coverage", "baseline.json"))
	if err != nil {
		return err
	}
	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		return err
	}
	actual, err := collectActualCoverageFields(outDir)
	if err != nil {
		return err
	}
	state := computeCoverageState(targets, actual, baseline)
	jsonRaw, err := renderCoverageJSON(state)
	if err != nil {
		return err
	}
	if err := writeFileUnder(outDir, "coverage/coverage.json", jsonRaw); err != nil {
		return err
	}
	if err := writeFileUnder(outDir, "coverage/coverage.md", []byte(renderCoverageMarkdown(state))); err != nil {
		return err
	}
	return nil
}

func CheckCoverage(cfg Config, outDir string) error {
	tmp, err := os.MkdirTemp("", "vibe-xpls-schema-coverage-*")
	if err != nil {
		return fmt.Errorf("create temp coverage dir: %w", err)
	}
	defer os.RemoveAll(tmp)
	if err := Generate(cfg, tmp); err != nil {
		return fmt.Errorf("regenerate schemas: %w", err)
	}
	if err := copyFile(filepath.Join(outDir, "coverage", "baseline.json"), filepath.Join(tmp, "coverage", "baseline.json")); err != nil {
		return err
	}
	if err := GenerateCoverage(cfg, tmp); err != nil {
		return err
	}
	for _, rel := range []string{"manifest.json", "schemas", "coverage/coverage.json", "coverage/coverage.md"} {
		if err := compareGeneratedPath(filepath.Join(outDir, rel), filepath.Join(tmp, rel), rel); err != nil {
			return err
		}
	}
	baseline, err := loadCoverageBaseline(filepath.Join(outDir, "coverage", "baseline.json"))
	if err != nil {
		return err
	}
	targets, err := collectCoverageTargets(cfg)
	if err != nil {
		return err
	}
	actual, err := collectActualCoverageFields(tmp)
	if err != nil {
		return err
	}
	state := computeCoverageState(targets, actual, baseline)
	if problems := validateCoverageRatchet(state, baseline); len(problems) != 0 {
		return fmt.Errorf(formatCoverageProblems(problems))
	}
	return nil
}

func writeFileUnder(outDir, relPath string, raw []byte) error {
	path, err := safeOutputPath(outDir, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create output dir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	return nil
}

func compareGeneratedPath(wantPath, gotPath, label string) error {
	wantFiles, err := filesByRelativePathForCheck(wantPath)
	if err != nil {
		return err
	}
	gotFiles, err := filesByRelativePathForCheck(gotPath)
	if err != nil {
		return err
	}
	if len(wantFiles) != len(gotFiles) {
		return fmt.Errorf("%s is stale: file count differs", label)
	}
	for path, want := range wantFiles {
		got, ok := gotFiles[path]
		if !ok {
			return fmt.Errorf("%s is stale: missing generated file %s", label, path)
		}
		if !bytes.Equal(want, got) {
			if path == "." {
				return fmt.Errorf("%s is stale", label)
			}
			return fmt.Errorf("%s/%s is stale", label, path)
		}
	}
	for path := range gotFiles {
		if _, ok := wantFiles[path]; !ok {
			return fmt.Errorf("%s is stale: extra generated file %s", label, path)
		}
	}
	return nil
}
```

Add a production copy of `filesByRelativePathForCheck` based on the existing test helper in `internal/analyzer/schema_test.go`.

- [ ] **Step 4: Add formatted ratchet problem output**

Add to `internal/schemagen/coverage_baseline.go`:

```go
func formatCoverageProblems(problems []coverageProblem) string {
	var b strings.Builder
	b.WriteString("schema coverage ratchet failed")
	for _, problem := range problems {
		b.WriteString("\n- ")
		b.WriteString(problem.Message)
	}
	return b.String()
}
```

Add `strings` to the imports.

- [ ] **Step 5: Verify coverage orchestration tests**

Run:

```sh
go test ./internal/schemagen -run 'TestGenerateCoverage|TestCheckCoverage' -count=1
```

Expected: pass after adding missing imports and helpers.

- [ ] **Step 6: Add failing CLI command-shape tests**

Create `cmd/vibe-xpls-schema-gen/main_test.go`:

```go
package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaGenCLIExplicitGenerateAndCompatibilityAlias(t *testing.T) {
	config := filepath.Join("..", "..", "internal", "schemagen", "testdata", "config.json")
	for _, args := range [][]string{
		{"run", ".", "generate", "--config", config, "--out", t.TempDir()},
		{"run", ".", "--config", config, "--out", t.TempDir()},
	} {
		cmd := exec.Command("go", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
}

func TestSchemaGenCLIRejectsUnknownCommand(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "unknown")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("unknown command succeeded")
	}
	if !strings.Contains(string(output), "usage: vibe-xpls-schema-gen") {
		t.Fatalf("output = %s, want usage", output)
	}
}
```

- [ ] **Step 7: Run failing CLI tests**

Run:

```sh
go test ./cmd/vibe-xpls-schema-gen -run TestSchemaGenCLI -count=1
```

Expected: explicit `generate` and unknown-command behavior fail until CLI parsing is implemented.

- [ ] **Step 8: Implement CLI subcommands**

Replace `cmd/vibe-xpls-schema-gen/main.go` with subcommand parsing:

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/io41/vibe-xpls/internal/schemagen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "generate" || args[0] == "--config" || args[0] == "--out" {
		if len(args) != 0 && args[0] == "generate" {
			args = args[1:]
		}
		return runGenerate(args)
	}
	if args[0] == "coverage" {
		return runCoverage(args[1:])
	}
	if args[0] == "drift" {
		return runDrift(args[1:])
	}
	return usageError(fmt.Sprintf("unknown command %q", args[0]))
}

func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	configPath := fs.String("config", "internal/analyzer/schemadata/config.json", "schema generator config")
	outDir := fs.String("out", "internal/analyzer/schemadata", "schema data output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := schemagen.Generate(cfg, *outDir); err != nil {
		return fmt.Errorf("generate schemas: %w", err)
	}
	return nil
}

func runCoverage(args []string) error {
	if len(args) == 0 {
		return usageError("coverage requires generate or check")
	}
	mode := args[0]
	fs := flag.NewFlagSet("coverage "+mode, flag.ContinueOnError)
	configPath := fs.String("config", "internal/analyzer/schemadata/config.json", "schema generator config")
	outDir := fs.String("out", "internal/analyzer/schemadata", "schema data output directory")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	switch mode {
	case "generate":
		return schemagen.GenerateCoverage(cfg, *outDir)
	case "check":
		return schemagen.CheckCoverage(cfg, *outDir)
	default:
		return usageError(fmt.Sprintf("unknown coverage command %q", mode))
	}
}

func runDrift(args []string) error {
	if len(args) == 0 || args[0] != "check" {
		return usageError("drift requires check")
	}
	fs := flag.NewFlagSet("drift check", flag.ContinueOnError)
	configPath := fs.String("config", "internal/analyzer/schemadata/config.json", "schema generator config")
	requireToken := fs.Bool("require-token", false, "fail when GITHUB_TOKEN is not set")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return schemagen.CheckDrift(cfg, schemagen.DriftOptions{Token: os.Getenv("GITHUB_TOKEN"), RequireToken: *requireToken})
}

func usageError(message string) error {
	return fmt.Errorf("%s\nusage: vibe-xpls-schema-gen [generate] --config <path> --out <dir>\n       vibe-xpls-schema-gen coverage generate --config <path> --out <dir>\n       vibe-xpls-schema-gen coverage check --config <path> --out <dir>\n       vibe-xpls-schema-gen drift check --config <path>", message)
}
```

`CheckDrift` is added in Task 8. To keep this task compiling before Task 8, add a temporary exported stub in `internal/schemagen/drift.go` with the final signature and a body that returns `nil`; Task 8 replaces it with the real implementation.

- [ ] **Step 9: Update stale-generation test to use explicit command**

In `internal/analyzer/schema_test.go`, change `TestGeneratedSchemaBundleIsCurrent` command construction to:

```go
cmd := exec.Command("go", "run", "../../cmd/vibe-xpls-schema-gen", "generate", "--config", "schemadata/config.json", "--out", tmp)
```

After coverage artifacts exist, this same test will compare `schemadata/coverage/coverage.json` and `schemadata/coverage/coverage.md` as part of Task 7.

- [ ] **Step 10: Verify CLI and coverage tests**

Run:

```sh
go test ./internal/schemagen ./cmd/vibe-xpls-schema-gen ./internal/analyzer -run 'TestGenerateCoverage|TestCheckCoverage|TestSchemaGenCLI|TestGeneratedSchemaBundleIsCurrent' -count=1
```

Expected: pass.

- [ ] **Step 11: Commit coverage commands**

Run:

```sh
git add internal/schemagen/coverage_check.go cmd/vibe-xpls-schema-gen/main.go cmd/vibe-xpls-schema-gen/main_test.go internal/analyzer/schema_test.go
git commit -m "feat: add schema coverage generator commands"
```

---

### Task 7: Generated Coverage Artifacts, Update Script, And Docs

**Files:**
- Modify: `scripts/update-generated.sh`
- Modify: `internal/analyzer/schema_test.go`
- Create: `internal/analyzer/schemadata/coverage/coverage.json`
- Create: `internal/analyzer/schemadata/coverage/coverage.md`
- Modify: `internal/analyzer/schemadata/coverage/baseline.json`
- Modify: `docs/generated-schemas.md`
- Modify: `PROJECT_ROADMAP.md`

- [ ] **Step 1: Update the single update command**

Modify `scripts/update-generated.sh` so the generator section is:

```sh
echo "==> Regenerating built-in Crossplane schema bundle"
go run ./cmd/vibe-xpls-schema-gen generate \
  --config internal/analyzer/schemadata/config.json \
  --out internal/analyzer/schemadata

echo "==> Regenerating schema coverage artifacts"
go run ./cmd/vibe-xpls-schema-gen coverage generate \
  --config internal/analyzer/schemadata/config.json \
  --out internal/analyzer/schemadata

echo "==> Checking schema coverage ratchet"
go run ./cmd/vibe-xpls-schema-gen coverage check \
  --config internal/analyzer/schemadata/config.json \
  --out internal/analyzer/schemadata
```

Keep the existing schema generator tests, full test suite, and build steps after this block.

- [ ] **Step 2: Generate initial coverage artifacts**

Run:

```sh
go run ./cmd/vibe-xpls-schema-gen generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
go run ./cmd/vibe-xpls-schema-gen coverage generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

Expected: `coverage/coverage.json` and `coverage/coverage.md` are created. `coverage/baseline.json` remains present and is not rewritten by the command.

- [ ] **Step 3: Run coverage check to get current baseline failures**

Run:

```sh
go run ./cmd/vibe-xpls-schema-gen coverage check --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

Expected before baseline population: fail with `schema coverage ratchet failed` and one bullet per unclassified observed gap. Every bullet must include release, apiVersion, kind, path, category, and reason.

- [ ] **Step 4: Populate the human-maintained baseline**

Edit `internal/analyzer/schemadata/coverage/baseline.json` by adding one entry per current observed gap. Each entry must use the exact release, apiVersion, kind, path, and category printed by `coverage check`. Use these reason strings consistently:

```json
{
  "release": "v1.20.7",
  "apiVersion": "example.org/v1",
  "kind": "Widget",
  "path": "spec.path",
  "category": "missing-field",
  "reason": "current generator does not emit this upstream field",
  "note": "initial coverage ratchet baseline"
}
```

Use category-specific reasons:

- `missing-field`: `current generator does not emit this upstream field`
- `missing-description`: `current generator does not preserve this upstream description`
- `missing-type`: `current generator does not preserve this upstream type`
- `missing-required`: `current generator does not preserve this upstream required marker`
- `missing-enum`: `current generator does not preserve this upstream enum set`
- `missing-default`: `current generator does not preserve this upstream default`
- `missing-deprecation`: `current generator does not preserve this upstream deprecation marker`
- `unsupported-openapi-shape`: copy the unsupported construct from the check output into `reason`
- `compat-added-field`: `compatibility docs intentionally add this generated field`
- `compat-only-schema`: `compatibility docs intentionally add this generated schema`

Do not use wildcard release entries in the initial baseline unless the same exact gap is observed in both pinned releases and the reason is identical.

- [ ] **Step 5: Regenerate coverage after baseline population**

Run:

```sh
go run ./cmd/vibe-xpls-schema-gen coverage generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

Expected: `coverage/coverage.json` and `coverage/coverage.md` reflect the populated baseline counts.

- [ ] **Step 6: Verify the ratchet passes**

Run:

```sh
go run ./cmd/vibe-xpls-schema-gen coverage check --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

Expected: pass with exit code 0.

- [ ] **Step 7: Extend stale-generation test to compare coverage artifacts**

In `internal/analyzer/schema_test.go`, add after the schema directory comparison in `TestGeneratedSchemaBundleIsCurrent`:

```go
assertDirectoriesEqual(t, "schemadata/coverage/coverage.json", filepath.Join(tmp, "coverage", "coverage.json"))
assertDirectoriesEqual(t, "schemadata/coverage/coverage.md", filepath.Join(tmp, "coverage", "coverage.md"))
```

Also copy the committed baseline into the temp directory before running the coverage generation command, or change the test command sequence to run:

```go
cmd := exec.Command("go", "run", "../../cmd/vibe-xpls-schema-gen", "generate", "--config", "schemadata/config.json", "--out", tmp)
```

then copy `schemadata/coverage/baseline.json` into `tmp/coverage/baseline.json`, then run:

```go
cmd = exec.Command("go", "run", "../../cmd/vibe-xpls-schema-gen", "coverage", "generate", "--config", "schemadata/config.json", "--out", tmp)
```

Expected: the test regenerates both schema and coverage artifacts from committed inputs.

- [ ] **Step 8: Update generated schema docs**

Modify `docs/generated-schemas.md` so the command section says:

````markdown
Regenerate and verify after changing `internal/analyzer/schemadata/config.json`,
generator code, committed upstream artifacts, coverage baseline entries, or
generated-schema documentation:

```bash
./scripts/update-generated.sh
```

The command runs:

```bash
vibe-xpls-schema-gen generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
vibe-xpls-schema-gen coverage generate --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
vibe-xpls-schema-gen coverage check --config internal/analyzer/schemadata/config.json --out internal/analyzer/schemadata
```

`coverage/baseline.json` is human-maintained. Coverage generation never rewrites
it. When the ratchet reports a new gap, fix the generator or add a reviewed
baseline entry with the exact release, apiVersion, kind, path, category, reason,
and note. When generator support improves, remove obsolete baseline entries and
rerun the update command.
````

- [ ] **Step 9: Update the roadmap references**

Modify `PROJECT_ROADMAP.md`:

- Keep the `Single Update Command` text accurate by adding coverage artifacts and baseline entries to the list of inputs that require `./scripts/update-generated.sh`.
- Add references:

```markdown
- Schema coverage ratchet design:
  `docs/superpowers/specs/2026-06-09-schema-coverage-ratchet-design.md`
- Schema coverage ratchet implementation plan:
  `docs/superpowers/plans/2026-06-09-schema-coverage-ratchet.md`
```

- [ ] **Step 10: Run the single update command**

Run:

```sh
./scripts/update-generated.sh
```

Expected: schema generation, coverage generation, coverage check, focused generator/analyzer tests, full Go test suite, and local CLI builds pass.

- [ ] **Step 11: Commit artifacts and docs**

Run:

```sh
git add scripts/update-generated.sh internal/analyzer/schema_test.go internal/analyzer/schemadata/coverage docs/generated-schemas.md PROJECT_ROADMAP.md
git commit -m "feat: add generated schema coverage artifacts"
```

---

### Task 8: Upstream Drift Check And Scheduled Workflow

**Files:**
- Create: `internal/schemagen/drift.go`
- Create: `internal/schemagen/drift_test.go`
- Modify: `cmd/vibe-xpls-schema-gen/main.go`
- Modify: `cmd/vibe-xpls-schema-gen/main_test.go`
- Create: `.github/workflows/schema-drift.yml`
- Modify: `docs/generated-schemas.md`

- [ ] **Step 1: Add failing drift tests**

Create `internal/schemagen/drift_test.go`:

```go
package schemagen

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckDriftRequiresTokenWhenConfigured(t *testing.T) {
	err := CheckDrift(fixtureConfig(), DriftOptions{RequireToken: true})
	if err == nil || !strings.Contains(err.Error(), "GITHUB_TOKEN is required") {
		t.Fatalf("CheckDrift error = %v, want required token", err)
	}
}

func TestCheckDriftDetectsPinnedCommitMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/crossplane/crossplane/git/ref/tags/v1.20.7":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":{"sha":"different"}}`))
		case "/repos/crossplane/crossplane/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"name":"v2.2.1","commit":{"sha":"713541df7fc5cf0946b6573837831086465a2dbe"}}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := fixtureConfig()
	err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil || !strings.Contains(err.Error(), "pinned release v1.20.7 resolves to different") {
		t.Fatalf("CheckDrift error = %v, want commit mismatch", err)
	}
}

func TestCheckDriftReportsNetworkErrorsSeparately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`unavailable`))
	}))
	defer server.Close()

	err := CheckDrift(fixtureConfig(), DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil || !strings.Contains(err.Error(), "query upstream drift") {
		t.Fatalf("CheckDrift error = %v, want upstream query failure", err)
	}
}
```

- [ ] **Step 2: Run failing drift tests**

Run:

```sh
go test ./internal/schemagen -run TestCheckDrift -count=1
```

Expected: fail until real drift logic exists.

- [ ] **Step 3: Implement drift checking**

Replace the temporary stub with `internal/schemagen/drift.go`:

```go
package schemagen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type DriftOptions struct {
	GitHubBaseURL string
	HTTPClient    *http.Client
	Token         string
	RequireToken  bool
}

func CheckDrift(cfg Config, opts DriftOptions) error {
	if opts.RequireToken && opts.Token == "" {
		return fmt.Errorf("GITHUB_TOKEN is required for scheduled schema drift check")
	}
	baseURL := strings.TrimRight(opts.GitHubBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	var problems []string
	for _, release := range cfg.Releases {
		sha, err := githubTagSHA(client, baseURL, opts.Token, release.Tag)
		if err != nil {
			return fmt.Errorf("query upstream drift for %s: %w", release.Tag, err)
		}
		if sha != release.Commit {
			problems = append(problems, fmt.Sprintf("pinned release %s resolves to different commit %s, want %s", release.Tag, sha, release.Commit))
		}
	}
	if len(problems) != 0 {
		return fmt.Errorf("schema upstream drift detected\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func githubTagSHA(client *http.Client, baseURL, token, tag string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/repos/crossplane/crossplane/git/ref/tags/"+tag, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github ref status %s", resp.Status)
	}
	var body struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Object.SHA == "" {
		return "", fmt.Errorf("github ref response omitted object sha")
	}
	return body.Object.SHA, nil
}
```

This first implementation validates tag-to-commit drift. Add release freshness and CRD file content drift in the same file before closing the task by using the same request helper:

- `GET /repos/crossplane/crossplane/tags?per_page=100` to detect a stable semver tag greater than the latest configured release.
- `GET /repos/crossplane/crossplane/contents/<crd path>?ref=<tag>` with `Accept: application/vnd.github.raw` to compare SHA-256 of upstream CRD contents to committed files.

Each problem should append a separate bullet to `schema upstream drift detected`.

- [ ] **Step 4: Add CLI test for scheduled token requirement**

Append to `cmd/vibe-xpls-schema-gen/main_test.go`:

```go
func TestSchemaGenCLIDriftRequiresTokenWhenRequested(t *testing.T) {
	config := filepath.Join("..", "..", "internal", "schemagen", "testdata", "config.json")
	cmd := exec.Command("go", "run", ".", "drift", "check", "--config", config, "--require-token")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("drift check without token succeeded")
	}
	if !strings.Contains(string(output), "GITHUB_TOKEN is required") {
		t.Fatalf("output = %s, want token error", output)
	}
}
```

- [ ] **Step 5: Add scheduled drift workflow**

Create `.github/workflows/schema-drift.yml`:

```yaml
name: Schema Drift

on:
  schedule:
    - cron: "17 5 * * 1"
  workflow_dispatch:

permissions:
  contents: read

jobs:
  drift:
    name: Check upstream schema drift
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true

      - name: Check Crossplane upstream drift
        run: go run ./cmd/vibe-xpls-schema-gen drift check --config internal/analyzer/schemadata/config.json --require-token
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 6: Document drift command**

Append this section to `docs/generated-schemas.md`:

````markdown
## Upstream Drift Check

Normal PR checks do not access the network. Scheduled CI checks upstream drift:

```bash
vibe-xpls-schema-gen drift check --config internal/analyzer/schemadata/config.json
```

The scheduled workflow runs with `--require-token` and `GITHUB_TOKEN` so GitHub
rate limits are explicit. The command reports pinned tag commit drift, newer
stable Crossplane tags, and CRD content drift. It does not mutate committed
schema inputs.
````

- [ ] **Step 7: Verify drift tests**

Run:

```sh
go test ./internal/schemagen ./cmd/vibe-xpls-schema-gen -run 'TestCheckDrift|TestSchemaGenCLIDrift' -count=1
```

Expected: pass without contacting the real network.

- [ ] **Step 8: Commit drift check**

Run:

```sh
git add internal/schemagen/drift.go internal/schemagen/drift_test.go cmd/vibe-xpls-schema-gen/main.go cmd/vibe-xpls-schema-gen/main_test.go .github/workflows/schema-drift.yml docs/generated-schemas.md
git commit -m "feat: add scheduled schema drift check"
```

---

### Task 9: Final Verification And Acceptance Pass

**Files:**
- No new files.
- Modify only files needed to fix verification failures from earlier tasks.

- [ ] **Step 1: Run focused schema generation verification**

Run:

```sh
go test ./internal/schemagen ./internal/analyzer -count=1
```

Expected: pass. This proves generator unit tests, coverage tests, bundle stale tests, and analyzer schema tests agree.

- [ ] **Step 2: Run the single update command**

Run:

```sh
./scripts/update-generated.sh
```

Expected: pass. This proves the documented local workflow regenerates schemas, regenerates coverage artifacts, enforces the ratchet, runs all tests, and builds both CLIs.

- [ ] **Step 3: Confirm normal tests do not require network**

Run:

```sh
go test ./... -count=1
```

Expected: pass without external network access. Drift tests must use `httptest.Server`; no test should call GitHub.

- [ ] **Step 4: Inspect generated coverage artifacts**

Run:

```sh
git diff -- internal/analyzer/schemadata/coverage/coverage.json internal/analyzer/schemadata/coverage/coverage.md internal/analyzer/schemadata/coverage/baseline.json
```

Expected: generated artifacts are deterministic, sorted, and reviewable. Baseline entries are exact release-scoped entries unless an intentional wildcard entry matches at least one observed gap.

- [ ] **Step 5: Check formatting and module tidiness**

Run:

```sh
gofmt -w internal/schemagen cmd/vibe-xpls-schema-gen
go mod tidy -diff
git diff --check
```

Expected: `gofmt` leaves Go files formatted, `go mod tidy -diff` prints no required changes, and `git diff --check` reports no whitespace errors.

- [ ] **Step 6: Review acceptance criteria**

Confirm each statement is true before the final commit:

- `./scripts/update-generated.sh` reports schema coverage failures with actionable reasons.
- `go test ./...` enforces committed pinned CRDs offline.
- `coverage.json` and `coverage.md` are deterministic and checked in.
- `baseline.json` classifies every current observed gap and fails when entries become obsolete.
- Description, type, required, enum, default, and deprecation metadata are included in coverage status.
- `.github/workflows/schema-drift.yml` runs only the networked drift command.
- The bare generator invocation still works as a compatibility alias for `generate`.

- [ ] **Step 7: Commit final fixes**

Run:

```sh
git add .
git commit -m "test: verify schema coverage ratchet"
```

If there are no final fixes after Task 8, skip this commit.

---

## Spec Coverage Review

- Developer workflow: Tasks 6 and 7 add explicit CLI subcommands, compatibility alias, update script integration, and docs.
- Resource and field coverage: Tasks 2, 3, and 5 extract targets, collect actual generated fields, classify buckets, and render totals per release/GVK.
- Coverage buckets: Task 3 defines and tests upstream, compat override, compat added, compat-only, missing, excluded, and unsupported-shape buckets.
- Target extraction rules: Task 2 covers properties, items, local refs, additionalProperties, preserve unknown fields, embedded resources, int-or-string, oneOf, anyOf, allOf, and patternProperties.
- Metadata coverage: Task 3 compares description, type, required, enum, default, and deprecation metadata with explicit status values.
- Known gap baseline: Tasks 1, 4, and 7 load, validate, apply, populate, and enforce the baseline.
- Artifacts: Tasks 5 and 7 create deterministic `coverage.json`, `coverage.md`, and `baseline.json` under `internal/analyzer/schemadata/coverage/`.
- Ratchet semantics: Tasks 4, 6, and 7 fail on stale artifacts, unclassified gaps, obsolete baseline entries, and schema artifact drift.
- Upstream drift detection: Task 8 adds the networked drift command and scheduled GitHub Actions workflow.
- Tests: Tasks 1 through 9 add focused tests plus full update-command verification.
