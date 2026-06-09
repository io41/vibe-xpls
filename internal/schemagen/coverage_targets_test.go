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
