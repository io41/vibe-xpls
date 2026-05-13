package analyzer

import "strings"

type Completion struct {
	Items []CompletionItem
}

type CompletionItem struct {
	Label         string
	Documentation string
}

func (a *Analyzer) Completion(uri, parentPath string) Completion {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Completion{}
	}
	if parentPath != "" && !parsed.IsStablePath(parentPath) {
		return Completion{}
	}
	apiVersion, kind, ok := rootGVKForPath(parsed, parentPath)
	if !ok {
		return Completion{}
	}
	return completionFromSchema(a.schemas, apiVersion, kind, parentPath)
}

func (a *Analyzer) CompletionAtOffset(uri string, offset int) Completion {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Completion{}
	}
	occurrence, ok := parsed.PathOccurrenceAtOffset(offset)
	if !ok || !occurrence.Stable || parsed.offsetInTemplateAction(offset) {
		return Completion{}
	}
	apiVersion, apiOK := parsed.RootValueForOccurrence(occurrence, "apiVersion")
	kind, kindOK := parsed.RootValueForOccurrence(occurrence, "kind")
	if !apiOK || !kindOK {
		return Completion{}
	}
	parentPath := occurrence.Path
	if !hasDirectSchemaChildren(a.schemas, apiVersion, kind, parentPath) {
		parentPath = parentYAMLPath(parentPath)
	}
	if parentPath != "" && !parsed.IsStablePath(parentPath) {
		return Completion{}
	}
	return completionFromSchema(a.schemas, apiVersion, kind, parentPath)
}

func completionFromSchema(schemas *SchemaIndex, apiVersion, kind, parentPath string) Completion {
	var items []CompletionItem
	seen := map[string]struct{}{}
	prefix := parentPath
	if prefix != "" {
		prefix += "."
	}
	for _, field := range schemas.Fields(apiVersion, kind) {
		if !strings.HasPrefix(field.Path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(field.Path, prefix)
		if rest == "" {
			continue
		}
		label := rest
		documentation := field.Description
		if split := strings.IndexAny(rest, ".["); split >= 0 {
			label = rest[:split]
			documentation = ""
		}
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		items = append(items, CompletionItem{Label: label, Documentation: documentation})
	}
	return Completion{Items: items}
}

func hasDirectSchemaChildren(schemas *SchemaIndex, apiVersion, kind, parentPath string) bool {
	return len(completionFromSchema(schemas, apiVersion, kind, parentPath).Items) > 0
}

func parentYAMLPath(path string) string {
	if dot := strings.LastIndex(path, "."); dot >= 0 {
		return path[:dot]
	}
	return ""
}
