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

func TestCoverageBucketsBaselineClassifiedFieldGapsBecomeExcluded(t *testing.T) {
	targets := []coverageTarget{
		{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.missing",
		},
		{
			Release:           "v1.20.7",
			APIVersion:        "example.org/v1",
			Kind:              "Widget",
			Path:              "spec.unsupported",
			UnsupportedReason: "unsupported fixture shape",
		},
		{
			Release:     "v1.20.7",
			APIVersion:  "example.org/v1",
			Kind:        "Widget",
			Path:        "spec.metadata",
			Description: "upstream description",
			Type:        "string",
		},
	}
	actual := map[actualCoverageKey]actualCoverageField{
		{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.compatOnly",
		}: {
			Path:        "spec.compatOnly",
			CompatAdded: true,
		},
		{
			Release:    "v1.20.7",
			APIVersion: "example.org/v1",
			Kind:       "Widget",
			Path:       "spec.metadata",
		}: {
			Path:        "spec.metadata",
			Description: "generated description",
			Type:        "string",
		},
	}
	baseline := coverageBaseline{
		FormatVersion: coverageFormatVersion,
		Entries: []coverageBaselineEntry{
			{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.missing",
				Category:   gapMissingField,
				Reason:     "current generator omits this fixture field",
				Note:       "test fixture",
			},
			{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.unsupported",
				Category:   gapUnsupportedOpenAPIShape,
				Reason:     "current generator cannot model this shape",
				Note:       "test fixture",
			},
			{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.compatOnly",
				Category:   gapCompatOnlySchema,
				Reason:     "compatibility-only field remains generated",
				Note:       "test fixture",
			},
			{
				Release:    "v1.20.7",
				APIVersion: "example.org/v1",
				Kind:       "Widget",
				Path:       "spec.metadata",
				Category:   gapMissingDescription,
				Reason:     "current generator uses compatibility metadata",
				Note:       "test fixture",
			},
		},
	}

	state := computeCoverageState(targets, actual, baseline)

	assertCoverageFieldBucket(t, state, "spec.missing", bucketExcluded)
	assertCoverageFieldBucket(t, state, "spec.unsupported", bucketExcluded)
	assertCoverageFieldBucket(t, state, "spec.compatOnly", bucketExcluded)
	assertCoverageFieldBucket(t, state, "spec.metadata", bucketCoveredUpstream)
	assertCoverageGap(t, state, "spec.metadata", gapMissingDescription)
}

func assertCoverageGap(t *testing.T, state coverageState, path string, category gapCategory) {
	t.Helper()
	for _, gap := range state.Gaps {
		if gap.Path == path && gap.Category == category {
			return
		}
	}
	t.Fatalf("missing coverage gap path=%s category=%s", path, category)
}
