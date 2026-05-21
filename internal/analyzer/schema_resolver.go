package analyzer

import (
	"os"
	"path/filepath"
	"strings"
)

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

func (a *Analyzer) resolveSchemaRelease(uri string, gvk SourceGVK) schemaResolution {
	if !a.schemas.bundleStatus.OK {
		return schemaResolution{Reason: SuppressionBundleDisabled}
	}
	candidates := a.schemas.ReleasesForGVK(gvk)
	if len(candidates) == 0 {
		return schemaResolution{Reason: SuppressionUnknownGVK}
	}
	path, ok := filePathFromURI(uri)
	if !ok {
		return latestSchemaRelease(candidates)
	}
	pkg, ok := a.workspace.PackageForFile(path)
	if !ok {
		return latestSchemaRelease(candidates)
	}
	versionRange, ok := a.packageCrossplaneVersionRange(pkg)
	if !ok {
		return latestSchemaRelease(candidates)
	}
	filtered := make([]CrossplaneRelease, 0, len(candidates))
	for _, release := range candidates {
		if versionRange.includes(release) {
			filtered = append(filtered, release)
		}
	}
	if len(filtered) == 0 {
		return schemaResolution{Reason: SuppressionNoSchemaForRelease}
	}
	return latestSchemaRelease(filtered)
}

func latestSchemaRelease(releases []CrossplaneRelease) schemaResolution {
	if len(releases) == 0 {
		return schemaResolution{Reason: SuppressionUnknownGVK}
	}
	sortCrossplaneReleases(releases)
	return schemaResolution{Release: releases[len(releases)-1], OK: true}
}

type crossplaneVersionRange struct {
	min          schemaSemVer
	minInclusive bool
	max          schemaSemVer
	maxExclusive bool
}

func (a *Analyzer) packageCrossplaneVersionRange(pkg PackageRoot) (crossplaneVersionRange, bool) {
	markerPath, ok := packageMarkerPath(pkg)
	if !ok {
		return crossplaneVersionRange{}, false
	}
	if doc, ok := a.docs.GetByFilePath(markerPath); ok {
		return packageCrossplaneVersionRangeFromText(doc.Text)
	}
	raw, err := os.ReadFile(markerPath)
	if err != nil {
		return crossplaneVersionRange{}, false
	}
	return packageCrossplaneVersionRangeFromText(string(raw))
}

func packageMarkerPath(pkg PackageRoot) (string, bool) {
	if _, ok := markerPriority[pkg.Marker]; !ok {
		return "", false
	}
	if pkg.Root == "" {
		return "", false
	}
	return filepath.Join(pkg.Root, pkg.Marker), true
}

func packageCrossplaneVersionRangeFromText(text string) (crossplaneVersionRange, bool) {
	parsed := ParseYAMLDocument(text)
	version := strings.TrimSpace(parsed.Values["spec.crossplane.version"])
	if version == "" {
		return crossplaneVersionRange{}, false
	}
	return parseCrossplaneVersionRange(version)
}

func parseCrossplaneVersionRange(value string) (crossplaneVersionRange, bool) {
	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 2 {
		return crossplaneVersionRange{}, false
	}
	var versionRange crossplaneVersionRange
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, ">="):
			if versionRange.min.ok {
				return crossplaneVersionRange{}, false
			}
			min := parseSchemaSemVer(strings.TrimPrefix(part, ">="))
			if !min.ok {
				return crossplaneVersionRange{}, false
			}
			versionRange.min = min
			versionRange.minInclusive = true
		case strings.HasPrefix(part, "<"):
			if versionRange.max.ok {
				return crossplaneVersionRange{}, false
			}
			max := parseSchemaSemVer(strings.TrimPrefix(part, "<"))
			if !max.ok {
				return crossplaneVersionRange{}, false
			}
			versionRange.max = max
			versionRange.maxExclusive = true
		default:
			return crossplaneVersionRange{}, false
		}
	}
	if !versionRange.min.ok || !versionRange.minInclusive {
		return crossplaneVersionRange{}, false
	}
	if len(parts) == 2 && (!versionRange.max.ok || !versionRange.maxExclusive) {
		return crossplaneVersionRange{}, false
	}
	return versionRange, true
}

func (r crossplaneVersionRange) includes(release CrossplaneRelease) bool {
	version := parseSchemaSemVer(release.Tag)
	if !version.ok {
		return false
	}
	if r.min.ok && compareSchemaSemVer(version, r.min) < 0 {
		return false
	}
	if r.max.ok && compareSchemaSemVer(version, r.max) >= 0 {
		return false
	}
	return true
}
