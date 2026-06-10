package schemagen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type coverageGVKKey struct {
	Release    string
	APIVersion string
	Kind       string
}

type coverageReportDocument struct {
	FormatVersion int                     `json:"formatVersion"`
	Releases      []coverageReleaseReport `json:"releases"`
}

type coverageReleaseReport struct {
	Tag    string               `json:"tag"`
	Totals coverageTotalsReport `json:"totals"`
	GVKs   []coverageGVKReport  `json:"gvks"`
}

type coverageTotalsReport struct {
	UpstreamGVKs          int `json:"upstreamGVKs"`
	GeneratedGVKs         int `json:"generatedGVKs"`
	TargetFields          int `json:"targetFields"`
	CoveredUpstreamFields int `json:"coveredUpstreamFields"`
	KnownGaps             int `json:"knownGaps"`
}

type coverageGVKReport struct {
	APIVersion   string                `json:"apiVersion"`
	Kind         string                `json:"kind"`
	SourcePath   string                `json:"sourcePath"`
	SourceSHA256 string                `json:"sourceSHA256"`
	Buckets      map[string]int        `json:"buckets"`
	Fields       []coverageFieldReport `json:"fields"`
}

type coverageFieldReport struct {
	Path     string                 `json:"path"`
	Bucket   string                 `json:"bucket"`
	Metadata coverageMetadataReport `json:"metadata"`
}

type coverageMetadataReport struct {
	Description metadataCoverageStatus `json:"description,omitempty"`
	Type        metadataCoverageStatus `json:"type,omitempty"`
	Required    metadataCoverageStatus `json:"required,omitempty"`
	Enum        metadataCoverageStatus `json:"enum,omitempty"`
	Default     metadataCoverageStatus `json:"default,omitempty"`
	Deprecated  metadataCoverageStatus `json:"deprecated,omitempty"`
}

func computeCoverageState(targets []coverageTarget, actual map[actualCoverageKey]actualCoverageField, baseline coverageBaseline) coverageState {
	state := coverageState{}
	gvks := map[coverageGVKKey]*coverageGVKState{}
	targetKeys := map[actualCoverageKey]struct{}{}

	for _, target := range targets {
		key := actualCoverageKey{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
		}
		targetKeys[key] = struct{}{}

		actualField, hasActual := actual[key]
		fieldState := coverageFieldState{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
		}

		var fieldGaps []observedGap
		switch {
		case target.UnsupportedReason != "":
			fieldState.Bucket = bucketUnsupportedShape
			gap := observedGap{
				Release:    target.Release,
				APIVersion: target.APIVersion,
				Kind:       target.Kind,
				Path:       target.Path,
				Category:   gapUnsupportedOpenAPIShape,
				Reason:     target.UnsupportedReason,
			}
			fieldState.Gap = &gap
			fieldGaps = append(fieldGaps, gap)
			markBaselineClassifiedFieldExcluded(&fieldState, baseline)
		case hasActual:
			if actualField.CompatOverride {
				fieldState.Bucket = bucketCoveredWithCompatOverride
			} else {
				fieldState.Bucket = bucketCoveredUpstream
			}
			fieldState.Metadata = compareMetadata(target, actualField)
			fieldGaps = append(fieldGaps, metadataGaps(target, actualField, fieldState.Metadata)...)
			if fieldState.Gap == nil && len(fieldGaps) > 0 {
				gap := fieldGaps[0]
				fieldState.Gap = &gap
			}
		default:
			fieldState.Bucket = bucketMissing
			gap := observedGap{
				Release:    target.Release,
				APIVersion: target.APIVersion,
				Kind:       target.Kind,
				Path:       target.Path,
				Category:   gapMissingField,
				Reason:     "generated schema is missing upstream field",
			}
			fieldState.Gap = &gap
			fieldGaps = append(fieldGaps, gap)
			markBaselineClassifiedFieldExcluded(&fieldState, baseline)
		}

		addCoverageField(gvks, fieldState, target.SourcePath, target.SourceSHA256)
		state.Gaps = append(state.Gaps, fieldGaps...)
	}

	for _, key := range sortedActualKeys(actual) {
		if _, ok := targetKeys[key]; ok {
			continue
		}
		actualField := actual[key]
		bucket := bucketCompatAddedField
		category := gapCompatAddedField
		reason := "generated compatibility field has no upstream target"
		if actualField.CompatAdded {
			bucket = bucketCompatOnlySchema
			category = gapCompatOnlySchema
			reason = "generated compatibility-only schema field has no upstream target"
		}
		gap := observedGap{
			Release:    key.Release,
			APIVersion: key.APIVersion,
			Kind:       key.Kind,
			Path:       key.Path,
			Category:   category,
			Reason:     reason,
		}
		fieldState := coverageFieldState{
			Release:    key.Release,
			APIVersion: key.APIVersion,
			Kind:       key.Kind,
			Path:       key.Path,
			Bucket:     bucket,
			Gap:        &gap,
		}
		if category == gapCompatOnlySchema {
			markBaselineClassifiedFieldExcluded(&fieldState, baseline)
		}
		addCoverageField(gvks, fieldState, "", "")
		state.Gaps = append(state.Gaps, gap)
	}

	state.GVKs = sortedCoverageGVKs(gvks)
	sortCoverageGaps(state.Gaps)
	return state
}

func renderCoverageJSON(state coverageState) ([]byte, error) {
	doc := coverageReportDocument{
		FormatVersion: coverageFormatVersion,
		Releases:      coverageReportReleases(state),
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal coverage report: %w", err)
	}
	return append(raw, '\n'), nil
}

func renderCoverageMarkdown(state coverageState) string {
	var b strings.Builder
	b.WriteString("# Schema Coverage\n\n")
	for _, release := range sortedCoverageReleaseTags(state) {
		gvks := coverageGVKsForRelease(state.GVKs, release)
		gaps := coverageGapsForRelease(state.Gaps, release)
		totals := coverageCounts(gvks, gaps)
		b.WriteString(fmt.Sprintf("## Release %s\n\n", release))
		b.WriteString(fmt.Sprintf(
			"Upstream field coverage: %d/%d (%.2f%%)\n",
			totals.CoveredUpstreamFields,
			totals.TargetFields,
			percent(totals.CoveredUpstreamFields, totals.TargetFields),
		))
		b.WriteString(fmt.Sprintf("Known gaps: %d\n\n", totals.KnownGaps))
		b.WriteString("### Metadata Coverage\n\n")
		b.WriteString("| Metadata | Covered | Overrides | Missing | Target | Coverage | No Upstream Fact |\n")
		b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		for _, row := range metadataCoverageRows(gvks) {
			b.WriteString(fmt.Sprintf(
				"| %s | %d | %d | %d | %d | %.2f%% | %d |\n",
				row.Name,
				row.Covered,
				row.Overrides,
				row.Missing,
				row.Target,
				percent(row.Covered+row.Overrides, row.Target),
				row.NoUpstreamFact,
			))
		}
		b.WriteString("\n")
		b.WriteString("### Known Gaps By Category\n\n")
		if len(gaps) == 0 {
			b.WriteString("No known gaps.\n\n")
		} else {
			b.WriteString("| Category | Count |\n")
			b.WriteString("| --- | ---: |\n")
			for _, row := range gapCategoryRows(gaps) {
				b.WriteString(fmt.Sprintf("| %s | %d |\n", row.Category, row.Count))
			}
			b.WriteString("\n")
		}
		b.WriteString("### Metadata Gap Hotspots\n\n")
		hotspots := metadataGapHotspots(gaps, 10)
		if len(hotspots) == 0 {
			b.WriteString("No metadata gaps.\n\n")
		} else {
			b.WriteString("| API Version | Kind | Category | Count | Examples |\n")
			b.WriteString("| --- | --- | --- | ---: | --- |\n")
			for _, row := range hotspots {
				b.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %s |\n", row.APIVersion, row.Kind, row.Category, row.Count, strings.Join(row.Examples, ", ")))
			}
			b.WriteString("\n")
		}
		b.WriteString("### Worst-Covered GVKs\n\n")
		b.WriteString("| API Version | Kind | Coverage | Known Gaps |\n")
		b.WriteString("| --- | --- | ---: | ---: |\n")
		for _, row := range worstCoveredGVKs(gvks, gaps, 10) {
			b.WriteString(fmt.Sprintf("| %s | %s | %.2f%% | %d |\n", row.APIVersion, row.Kind, row.Percent, row.KnownGaps))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

type metadataCoverageRow struct {
	Name           string
	Covered        int
	Overrides      int
	Missing        int
	Target         int
	NoUpstreamFact int
}

func metadataCoverageRows(gvks []coverageGVKState) []metadataCoverageRow {
	names := []string{"description", "type", "required", "enum", "default", "deprecated"}
	rows := make([]metadataCoverageRow, 0, len(names))
	for _, name := range names {
		row := metadataCoverageRow{Name: name}
		for _, gvk := range gvks {
			for _, field := range gvk.Fields {
				if !coverageFieldIsTarget(field) {
					continue
				}
				row.add(metadataStatusByName(field.Metadata, name))
			}
		}
		row.Target = row.Covered + row.Overrides + row.Missing
		rows = append(rows, row)
	}
	return rows
}

func (row *metadataCoverageRow) add(status metadataCoverageStatus) {
	switch status {
	case metadataCovered:
		row.Covered++
	case metadataCompatOverride:
		row.Overrides++
	case metadataMissing:
		row.Missing++
	case metadataNotPresentUpstream, metadataNotRequired:
		row.NoUpstreamFact++
	}
}

func metadataStatusByName(metadata coverageMetadataState, name string) metadataCoverageStatus {
	switch name {
	case "description":
		return metadata.Description
	case "type":
		return metadata.Type
	case "required":
		return metadata.Required
	case "enum":
		return metadata.Enum
	case "default":
		return metadata.Default
	case "deprecated":
		return metadata.Deprecated
	default:
		return ""
	}
}

type gapCategoryRow struct {
	Category gapCategory
	Count    int
}

func gapCategoryRows(gaps []observedGap) []gapCategoryRow {
	counts := map[gapCategory]int{}
	for _, gap := range gaps {
		counts[gap.Category]++
	}
	rows := make([]gapCategoryRow, 0, len(counts))
	for category, count := range counts {
		rows = append(rows, gapCategoryRow{Category: category, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Category < rows[j].Category
	})
	return rows
}

type metadataGapHotspot struct {
	APIVersion string
	Kind       string
	Category   gapCategory
	Count      int
	Examples   []string
}

func metadataGapHotspots(gaps []observedGap, limit int) []metadataGapHotspot {
	type hotspotKey struct {
		APIVersion string
		Kind       string
		Category   gapCategory
	}
	byKey := map[hotspotKey]*metadataGapHotspot{}
	for _, gap := range gaps {
		if !gapCategoryIsMetadata(gap.Category) {
			continue
		}
		key := hotspotKey{APIVersion: gap.APIVersion, Kind: gap.Kind, Category: gap.Category}
		row, ok := byKey[key]
		if !ok {
			row = &metadataGapHotspot{
				APIVersion: gap.APIVersion,
				Kind:       gap.Kind,
				Category:   gap.Category,
			}
			byKey[key] = row
		}
		row.Count++
		if len(row.Examples) < 3 {
			row.Examples = append(row.Examples, gap.Path)
		}
	}
	rows := make([]metadataGapHotspot, 0, len(byKey))
	for _, row := range byKey {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		if rows[i].APIVersion != rows[j].APIVersion {
			return rows[i].APIVersion < rows[j].APIVersion
		}
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Category < rows[j].Category
	})
	if len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func gapCategoryIsMetadata(category gapCategory) bool {
	switch category {
	case gapMissingDescription, gapMissingType, gapMissingRequired, gapMissingEnum, gapMissingDefault, gapMissingDeprecation:
		return true
	default:
		return false
	}
}

func coverageReportReleases(state coverageState) []coverageReleaseReport {
	tags := sortedCoverageReleaseTags(state)
	releases := make([]coverageReleaseReport, 0, len(tags))
	for _, tag := range tags {
		gvks := coverageGVKsForRelease(state.GVKs, tag)
		gaps := coverageGapsForRelease(state.Gaps, tag)
		releases = append(releases, coverageReleaseReport{
			Tag:    tag,
			Totals: coverageCounts(gvks, gaps),
			GVKs:   coverageGVKReports(gvks),
		})
	}
	return releases
}

func sortedCoverageReleaseTags(state coverageState) []string {
	seen := map[string]struct{}{}
	for _, gvk := range state.GVKs {
		seen[gvk.Release] = struct{}{}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func coverageGVKsForRelease(gvks []coverageGVKState, release string) []coverageGVKState {
	out := make([]coverageGVKState, 0, len(gvks))
	for _, gvk := range gvks {
		if gvk.Release == release {
			out = append(out, gvk)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].APIVersion != out[j].APIVersion {
			return out[i].APIVersion < out[j].APIVersion
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func coverageGapsForRelease(gaps []observedGap, release string) []observedGap {
	out := make([]observedGap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.Release == release {
			out = append(out, gap)
		}
	}
	sortCoverageGaps(out)
	return out
}

func coverageGapsForGVK(gaps []observedGap, release, apiVersion, kind string) []observedGap {
	out := make([]observedGap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.Release == release && gap.APIVersion == apiVersion && gap.Kind == kind {
			out = append(out, gap)
		}
	}
	sortCoverageGaps(out)
	return out
}

func coverageGVKReports(gvks []coverageGVKState) []coverageGVKReport {
	reports := make([]coverageGVKReport, 0, len(gvks))
	for _, gvk := range gvks {
		reports = append(reports, coverageGVKReport{
			APIVersion:   gvk.APIVersion,
			Kind:         gvk.Kind,
			SourcePath:   gvk.SourcePath,
			SourceSHA256: gvk.SourceSHA256,
			Buckets:      sortedBucketMap(gvk.Buckets),
			Fields:       coverageFieldReports(gvk.Fields),
		})
	}
	return reports
}

func coverageFieldReports(fields []coverageFieldState) []coverageFieldReport {
	sorted := append([]coverageFieldState(nil), fields...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		return sorted[i].Bucket < sorted[j].Bucket
	})

	reports := make([]coverageFieldReport, 0, len(sorted))
	for _, field := range sorted {
		reports = append(reports, coverageFieldReport{
			Path:   field.Path,
			Bucket: string(field.Bucket),
			Metadata: coverageMetadataReport{
				Description: field.Metadata.Description,
				Type:        field.Metadata.Type,
				Required:    field.Metadata.Required,
				Enum:        field.Metadata.Enum,
				Default:     field.Metadata.Default,
				Deprecated:  field.Metadata.Deprecated,
			},
		})
	}
	return reports
}

func sortedBucketMap(buckets map[coverageBucket]int) map[string]int {
	keys := make([]string, 0, len(buckets))
	for bucket := range buckets {
		keys = append(keys, string(bucket))
	}
	sort.Strings(keys)

	out := make(map[string]int, len(keys))
	for _, key := range keys {
		out[key] = buckets[coverageBucket(key)]
	}
	return out
}

func coverageCounts(gvks []coverageGVKState, gaps []observedGap) coverageTotalsReport {
	totals := coverageTotalsReport{KnownGaps: len(gaps)}
	for _, gvk := range gvks {
		var hasUpstreamField bool
		var hasGeneratedField bool
		for _, field := range gvk.Fields {
			if coverageFieldIsTarget(field) {
				hasUpstreamField = true
				totals.TargetFields++
			}
			if coverageBucketIsCovered(field.Bucket) {
				totals.CoveredUpstreamFields++
			}
			if coverageFieldIsGenerated(field) {
				hasGeneratedField = true
			}
		}
		if hasUpstreamField {
			totals.UpstreamGVKs++
		}
		if hasGeneratedField {
			totals.GeneratedGVKs++
		}
	}
	return totals
}

func coverageFieldIsTarget(field coverageFieldState) bool {
	switch field.Bucket {
	case bucketCompatAddedField, bucketCompatOnlySchema:
		return false
	default:
		return !coverageFieldIsExcludedCompatOnlySchema(field)
	}
}

func coverageBucketIsCovered(bucket coverageBucket) bool {
	return bucket == bucketCoveredUpstream || bucket == bucketCoveredWithCompatOverride
}

func coverageFieldIsGenerated(field coverageFieldState) bool {
	if coverageFieldIsExcludedCompatOnlySchema(field) {
		return true
	}
	switch field.Bucket {
	case bucketCoveredUpstream, bucketCoveredWithCompatOverride, bucketCompatAddedField, bucketCompatOnlySchema:
		return true
	default:
		return false
	}
}

func coverageFieldIsExcludedCompatOnlySchema(field coverageFieldState) bool {
	return field.Bucket == bucketExcluded && field.Gap != nil && field.Gap.Category == gapCompatOnlySchema
}

func percent(numerator, denominator int) float64 {
	if denominator == 0 {
		return 100
	}
	return float64(numerator) / float64(denominator) * 100
}

type worstCoveredGVK struct {
	APIVersion string
	Kind       string
	Percent    float64
	KnownGaps  int
}

func worstCoveredGVKs(gvks []coverageGVKState, gaps []observedGap, limit int) []worstCoveredGVK {
	rows := make([]worstCoveredGVK, 0, len(gvks))
	for _, gvk := range gvks {
		counts := coverageCounts(
			[]coverageGVKState{gvk},
			coverageGapsForGVK(gaps, gvk.Release, gvk.APIVersion, gvk.Kind),
		)
		if counts.TargetFields == 0 {
			continue
		}
		rows = append(rows, worstCoveredGVK{
			APIVersion: gvk.APIVersion,
			Kind:       gvk.Kind,
			Percent:    percent(counts.CoveredUpstreamFields, counts.TargetFields),
			KnownGaps:  counts.KnownGaps,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Percent != rows[j].Percent {
			return rows[i].Percent < rows[j].Percent
		}
		if rows[i].APIVersion != rows[j].APIVersion {
			return rows[i].APIVersion < rows[j].APIVersion
		}
		return rows[i].Kind < rows[j].Kind
	})
	if len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func markBaselineClassifiedFieldExcluded(field *coverageFieldState, baseline coverageBaseline) {
	if field.Gap == nil {
		return
	}
	if _, ok := baseline.match(*field.Gap); ok {
		field.Bucket = bucketExcluded
	}
}

func addCoverageField(gvks map[coverageGVKKey]*coverageGVKState, field coverageFieldState, sourcePath, sourceSHA256 string) {
	key := coverageGVKKey{
		Release:    field.Release,
		APIVersion: field.APIVersion,
		Kind:       field.Kind,
	}
	gvk, ok := gvks[key]
	if !ok {
		gvk = &coverageGVKState{
			Release:    field.Release,
			APIVersion: field.APIVersion,
			Kind:       field.Kind,
			Buckets:    map[coverageBucket]int{},
		}
		gvks[key] = gvk
	}
	if gvk.SourcePath == "" {
		gvk.SourcePath = sourcePath
	}
	if gvk.SourceSHA256 == "" {
		gvk.SourceSHA256 = sourceSHA256
	}
	gvk.Fields = append(gvk.Fields, field)
	gvk.Buckets[field.Bucket]++
}

func compareMetadata(target coverageTarget, actual actualCoverageField) coverageMetadataState {
	return coverageMetadataState{
		Description: compareStringMetadata(target.Description, actual.Description, actual.CompatOverrideDescription),
		Type:        compareStringMetadata(target.Type, actual.Type, actual.CompatOverrideType),
		Required:    compareRequiredMetadata(target.Required, actual.Required),
		Enum:        compareSliceMetadata(target.Enum, actual.Enum),
		Default:     compareRawJSONMetadata(target.Default, actual.Default),
		Deprecated:  compareDeprecatedMetadata(target.Deprecated, actual.Deprecated),
	}
}

func compareStringMetadata(target, actual string, compatOverride bool) metadataCoverageStatus {
	if target == "" {
		return metadataNotPresentUpstream
	}
	if target == actual {
		return metadataCovered
	}
	if compatOverride && strings.TrimSpace(actual) != "" {
		return metadataCompatOverride
	}
	return metadataMissing
}

func compareRequiredMetadata(target, actual bool) metadataCoverageStatus {
	if !target {
		return metadataNotRequired
	}
	if actual {
		return metadataCovered
	}
	return metadataMissing
}

func compareSliceMetadata(target, actual []string) metadataCoverageStatus {
	if len(target) == 0 {
		return metadataNotPresentUpstream
	}
	if equalStringSet(target, actual) {
		return metadataCovered
	}
	return metadataMissing
}

func compareRawJSONMetadata(target, actual *json.RawMessage) metadataCoverageStatus {
	if target == nil {
		return metadataNotPresentUpstream
	}
	if actual == nil {
		return metadataMissing
	}
	if equalRawJSON(target, actual) {
		return metadataCovered
	}
	return metadataMissing
}

func compareDeprecatedMetadata(target bool, actual string) metadataCoverageStatus {
	if !target {
		return metadataNotPresentUpstream
	}
	if strings.TrimSpace(actual) != "" {
		return metadataCovered
	}
	return metadataMissing
}

func metadataGaps(target coverageTarget, actual actualCoverageField, metadata coverageMetadataState) []observedGap {
	var gaps []observedGap
	if metadata.Description == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingDescription,
			Reason:     stringMetadataGapReason("description", target.Description, actual.Description),
		})
	}
	if metadata.Type == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingType,
			Reason:     stringMetadataGapReason("type", target.Type, actual.Type),
		})
	}
	if metadata.Required == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingRequired,
			Reason:     "upstream marks field required but generated schema does not",
		})
	}
	if metadata.Enum == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingEnum,
			Reason:     fmt.Sprintf("generated schema enum %s differs from upstream enum %s", stringSliceReason(actual.Enum), stringSliceReason(target.Enum)),
		})
	}
	if metadata.Default == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingDefault,
			Reason:     fmt.Sprintf("generated schema default %s differs from upstream default %s", rawJSONReason(actual.Default), rawJSONReason(target.Default)),
		})
	}
	if metadata.Deprecated == metadataMissing {
		gaps = append(gaps, observedGap{
			Release:    target.Release,
			APIVersion: target.APIVersion,
			Kind:       target.Kind,
			Path:       target.Path,
			Category:   gapMissingDeprecation,
			Reason:     "upstream marks field deprecated but generated schema does not",
		})
	}
	return gaps
}

func sortedCoverageGVKs(gvks map[coverageGVKKey]*coverageGVKState) []coverageGVKState {
	keys := make([]coverageGVKKey, 0, len(gvks))
	for key := range gvks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGVKKeys(keys[i], keys[j]) < 0
	})

	out := make([]coverageGVKState, 0, len(keys))
	for _, key := range keys {
		gvk := *gvks[key]
		sort.Slice(gvk.Fields, func(i, j int) bool {
			left := gvk.Fields[i]
			right := gvk.Fields[j]
			if left.Path != right.Path {
				return left.Path < right.Path
			}
			return left.Bucket < right.Bucket
		})
		out = append(out, gvk)
	}
	return out
}

func sortCoverageGaps(gaps []observedGap) {
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Release != gaps[j].Release {
			return gaps[i].Release < gaps[j].Release
		}
		if gaps[i].APIVersion != gaps[j].APIVersion {
			return gaps[i].APIVersion < gaps[j].APIVersion
		}
		if gaps[i].Kind != gaps[j].Kind {
			return gaps[i].Kind < gaps[j].Kind
		}
		if gaps[i].Path != gaps[j].Path {
			return gaps[i].Path < gaps[j].Path
		}
		if gaps[i].Category != gaps[j].Category {
			return gaps[i].Category < gaps[j].Category
		}
		return gaps[i].Reason < gaps[j].Reason
	})
}

func compareGVKKeys(left, right coverageGVKKey) int {
	leftKey := strings.Join([]string{left.Release, left.APIVersion, left.Kind}, "\x00")
	rightKey := strings.Join([]string{right.Release, right.APIVersion, right.Kind}, "\x00")
	return strings.Compare(leftKey, rightKey)
}

func equalStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func equalRawJSON(left, right *json.RawMessage) bool {
	leftRaw := normalizedRawJSON(left)
	rightRaw := normalizedRawJSON(right)
	return bytes.Equal(leftRaw, rightRaw)
}

func normalizedRawJSON(value *json.RawMessage) []byte {
	if value == nil {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(*value, &decoded); err != nil {
		return bytes.TrimSpace(*value)
	}
	raw, err := json.Marshal(decoded)
	if err != nil {
		return bytes.TrimSpace(*value)
	}
	return raw
}

func stringMetadataGapReason(name, target, actual string) string {
	if actual == "" {
		return fmt.Sprintf("generated schema is missing %s metadata", name)
	}
	return fmt.Sprintf("generated schema %s %q differs from upstream %q", name, actual, target)
}

func stringSliceReason(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	return "[" + strings.Join(values, ", ") + "]"
}

func rawJSONReason(value *json.RawMessage) string {
	if value == nil {
		return "<missing>"
	}
	return string(normalizedRawJSON(value))
}
