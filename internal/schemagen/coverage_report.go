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
		markBaselineClassifiedFieldExcluded(&fieldState, baseline)
		addCoverageField(gvks, fieldState, "", "")
		state.Gaps = append(state.Gaps, gap)
	}

	state.GVKs = sortedCoverageGVKs(gvks)
	sortCoverageGaps(state.Gaps)
	return state
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
		Description: compareStringMetadata(target.Description, actual.Description),
		Type:        compareStringMetadata(target.Type, actual.Type),
		Required:    compareRequiredMetadata(target.Required, actual.Required),
		Enum:        compareSliceMetadata(target.Enum, actual.Enum),
		Default:     compareRawJSONMetadata(target.Default, actual.Default),
		Deprecated:  compareDeprecatedMetadata(target.Deprecated, actual.Deprecated),
	}
}

func compareStringMetadata(target, actual string) metadataCoverageStatus {
	if target == "" {
		return metadataNotPresentUpstream
	}
	if target == actual {
		return metadataCovered
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
