package schemagen

import (
	"sort"
)

func collectCoverageTargets(cfg Config) ([]coverageTarget, error) {
	targets := []coverageTarget{}
	for _, release := range cfg.Releases {
		crdFiles, err := yamlFiles(release.RawCRDDir)
		if err != nil {
			return nil, err
		}
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
					base := coverageTarget{
						Release:      release.Tag,
						APIVersion:   apiVersion,
						Kind:         doc.Spec.Names.Kind,
						SourcePath:   relativeSourcePath(release.RawCRDDir, path),
						SourceSHA256: sha,
					}
					versionTargets := map[string]coverageTarget{}
					addCoverageMetadataTargets(versionTargets, base, doc.Spec.Scope)
					walkCoverageSchema(versionTargets, base, version.Schema.OpenAPIV3Schema, version.Schema.OpenAPIV3Schema, "", false)
					for _, target := range versionTargets {
						targets = append(targets, target)
					}
				}
			}
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].Release != targets[j].Release {
			return targets[i].Release < targets[j].Release
		}
		if targets[i].APIVersion != targets[j].APIVersion {
			return targets[i].APIVersion < targets[j].APIVersion
		}
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		if targets[i].Path != targets[j].Path {
			return targets[i].Path < targets[j].Path
		}
		if targets[i].SourcePath != targets[j].SourcePath {
			return targets[i].SourcePath < targets[j].SourcePath
		}
		return targets[i].SourceSHA256 < targets[j].SourceSHA256
	})
	return targets, nil
}

func addCoverageMetadataTargets(targets map[string]coverageTarget, base coverageTarget, scope string) {
	putCoverageSyntheticTarget(targets, base, "metadata.name", "string", "Object name.")
	putCoverageSyntheticTarget(targets, base, "metadata.labels", "object", "Object labels.")
	putCoverageSyntheticTarget(targets, base, "metadata.annotations", "object", "Object annotations.")
	if scope == "Namespaced" {
		putCoverageSyntheticTarget(targets, base, "metadata.namespace", "string", "Object namespace.")
	}
}

func putCoverageSyntheticTarget(targets map[string]coverageTarget, base coverageTarget, path, typ, description string) {
	if _, ok := targets[path]; ok {
		return
	}
	target := base
	target.Path = path
	target.Description = description
	target.Type = typ
	targets[path] = target
}

func walkCoverageSchema(targets map[string]coverageTarget, base coverageTarget, root, schema openAPISchema, prefix string, required bool) {
	schema, err := resolveSchema(root, schema, nil)
	if err != nil {
		putCoverageUnsupportedTarget(targets, base, schema, prefix, required, err.Error())
		return
	}
	if schema.Type == "array" && schema.Items != nil {
		arrayPath := prefix + "[]"
		putCoverageTarget(targets, base, schema, arrayPath, required, coverageUnsupportedReason(schema))
		if coverageShouldStopAtContainingTarget(schema) || coverageUnsupportedReason(schema) != "" {
			return
		}
		item, err := resolveSchema(root, *schema.Items, nil)
		if err != nil {
			putCoverageUnsupportedTarget(targets, base, *schema.Items, arrayPath, false, err.Error())
			return
		}
		walkCoverageSchemaContents(targets, base, root, item, arrayPath)
		return
	}
	unsupportedReason := coverageUnsupportedReason(schema)
	if prefix != "" {
		putCoverageTarget(targets, base, schema, prefix, required, unsupportedReason)
	}
	if coverageShouldStopAtContainingTarget(schema) || unsupportedReason != "" {
		return
	}
	walkCoverageSchemaContents(targets, base, root, schema, prefix)
}

func walkCoverageSchemaContents(targets map[string]coverageTarget, base coverageTarget, root, schema openAPISchema, prefix string) {
	for _, child := range schema.AllOf {
		walkCoverageAllOfChild(targets, base, root, child, prefix)
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		walkCoverageSchema(targets, base, root, schema.Properties[name], joinPath(prefix, name), isRequired(schema.Required, name))
	}
}

func walkCoverageAllOfChild(targets map[string]coverageTarget, base coverageTarget, root, schema openAPISchema, prefix string) {
	schema, err := resolveSchema(root, schema, nil)
	if err != nil {
		putCoverageUnsupportedTarget(targets, base, schema, prefix, false, err.Error())
		return
	}
	unsupportedReason := coverageUnsupportedReason(schema)
	if unsupportedReason != "" {
		putCoverageUnsupportedTarget(targets, base, schema, prefix, false, unsupportedReason)
		return
	}
	if coverageShouldStopAtContainingTarget(schema) {
		return
	}
	if schema.Type == "array" && schema.Items != nil {
		walkCoverageSchema(targets, base, root, schema, prefix, false)
		return
	}
	walkCoverageSchemaContents(targets, base, root, schema, prefix)
}

func putCoverageTarget(targets map[string]coverageTarget, base coverageTarget, schema openAPISchema, path string, required bool, unsupportedReason string) {
	if path == "" {
		return
	}
	target := base
	target.Path = path
	target.Description = schema.Description
	target.Type = coverageTargetType(schema)
	target.Required = required
	target.Default = rawDefault(schema.Default)
	target.Enum = enumStrings(schema.Enum)
	target.Deprecated = schema.Deprecated != nil && *schema.Deprecated
	target.UnsupportedReason = unsupportedReason
	targets[path] = target
}

func putCoverageUnsupportedTarget(targets map[string]coverageTarget, base coverageTarget, schema openAPISchema, path string, required bool, reason string) {
	if path == "" {
		return
	}
	if target, ok := targets[path]; ok {
		target.UnsupportedReason = reason
		targets[path] = target
		return
	}
	putCoverageTarget(targets, base, schema, path, required, reason)
}

func coverageTargetType(schema openAPISchema) string {
	if schema.XKubernetesIntOrString {
		return "int-or-string"
	}
	return schema.Type
}

func coverageUnsupportedReason(schema openAPISchema) string {
	if len(schema.OneOf) > 0 {
		return "oneOf is unsupported"
	}
	if len(schema.AnyOf) > 0 && !schema.XKubernetesIntOrString {
		return "anyOf is unsupported"
	}
	return ""
}

func coverageShouldStopAtContainingTarget(schema openAPISchema) bool {
	return schema.AdditionalProperties != nil ||
		schema.XKubernetesPreserveUnknown ||
		schema.XKubernetesEmbeddedResource ||
		schema.XKubernetesIntOrString ||
		len(schema.PatternProperties) > 0
}
