package analyzer

import (
	"fmt"
	"strings"
	"unicode"
)

type Hover struct {
	Markdown string
}

func (a *Analyzer) Hover(uri, fieldPath string) (Hover, bool) {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Hover{}, false
	}
	root, ok := rootContextForExistingPath(parsed, fieldPath)
	if !ok {
		return Hover{}, false
	}
	gvk := SourceGVK{APIVersion: root.apiVersion, Kind: root.kind}
	if a.schemas.HasWorkspaceSchema(gvk) {
		field, ok := a.schemas.FieldDocumentation(root.apiVersion, root.kind, fieldPath)
		if !ok {
			return Hover{}, false
		}
		return hoverFromField(field), true
	}
	resolution := a.resolveSchemaRelease(uri, gvk)
	if !resolution.OK {
		return Hover{}, false
	}
	field, ok := a.schemas.FieldDocumentationForRelease(resolution.Release, root.apiVersion, root.kind, fieldPath)
	if !ok {
		return Hover{}, false
	}
	return hoverFromField(field), true
}

func (a *Analyzer) HoverAtOffset(uri string, offset int) (Hover, bool) {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Hover{}, false
	}
	occurrence, ok := parsed.PathOccurrenceAtOffset(offset)
	if !ok || !occurrence.Stable || parsed.offsetInTemplateAction(offset) {
		return Hover{}, false
	}
	apiVersion, apiOK := parsed.RootValueForOccurrence(occurrence, "apiVersion")
	kind, kindOK := parsed.RootValueForOccurrence(occurrence, "kind")
	if !apiOK || !kindOK {
		return Hover{}, false
	}
	gvk := SourceGVK{APIVersion: apiVersion, Kind: kind}
	if a.schemas.HasWorkspaceSchema(gvk) {
		field, ok := a.schemas.FieldDocumentation(apiVersion, kind, occurrence.Path)
		if !ok {
			return Hover{}, false
		}
		return hoverFromField(field), true
	}
	resolution := a.resolveSchemaRelease(uri, gvk)
	if !resolution.OK {
		return Hover{}, false
	}
	field, ok := a.schemas.FieldDocumentationForRelease(resolution.Release, apiVersion, kind, occurrence.Path)
	if !ok {
		return Hover{}, false
	}
	return hoverFromField(field), true
}

func hoverFromField(field FieldDoc) Hover {
	body := fieldCompletionDocumentation(field)
	if body == "" {
		return Hover{Markdown: "**" + hoverTitle(field.Path) + "**"}
	}
	return Hover{Markdown: fmt.Sprintf("**%s**\n\n%s", hoverTitle(field.Path), body)}
}

type rootContext struct {
	apiVersion string
	kind       string
}

func rootContextForExistingPath(parsed YAMLDocument, fieldPath string) (rootContext, bool) {
	var root rootContext
	haveRoot := false
	matched := false
	for _, occurrence := range parsed.occurrences {
		if occurrence.Path != fieldPath {
			continue
		}
		matched = true
		if !occurrence.Stable {
			return rootContext{}, false
		}
		next, ok := rootContextForOccurrence(parsed, occurrence)
		if !ok {
			return rootContext{}, false
		}
		if haveRoot && next != root {
			return rootContext{}, false
		}
		root = next
		haveRoot = true
	}
	if !matched || !haveRoot {
		return rootContext{}, false
	}
	return root, true
}

func singleStableRootContext(parsed YAMLDocument) (rootContext, bool) {
	seenDocuments := map[int]struct{}{}
	uniqueRoots := map[rootContext]struct{}{}
	for _, occurrence := range parsed.occurrences {
		if _, seen := seenDocuments[occurrence.DocumentIndex]; seen {
			continue
		}
		seenDocuments[occurrence.DocumentIndex] = struct{}{}
		root, ok := rootContextForOccurrence(parsed, occurrence)
		if !ok {
			return rootContext{}, false
		}
		uniqueRoots[root] = struct{}{}
	}
	if len(uniqueRoots) != 1 {
		return rootContext{}, false
	}
	for root := range uniqueRoots {
		return root, true
	}
	return rootContext{}, false
}

func rootContextForOccurrence(parsed YAMLDocument, occurrence PathOccurrence) (rootContext, bool) {
	apiVersion, apiOK := parsed.RootValueForOccurrence(occurrence, "apiVersion")
	kind, kindOK := parsed.RootValueForOccurrence(occurrence, "kind")
	if !apiOK || !kindOK || apiVersion == "" || kind == "" {
		return rootContext{}, false
	}
	return rootContext{apiVersion: apiVersion, kind: kind}, true
}

func hoverTitle(path string) string {
	switch path {
	case "apiVersion":
		return "API version"
	case "spec.compositeTypeRef.apiVersion":
		return "Composite API version"
	case "spec.compositeTypeRef.kind":
		return "Composite kind"
	default:
		return humanizeFieldName(lastPathSegment(path))
	}
}

func lastPathSegment(path string) string {
	if dot := strings.LastIndex(path, "."); dot >= 0 {
		path = path[dot+1:]
	}
	if bracket := strings.LastIndex(path, "]"); bracket >= 0 && bracket+1 < len(path) {
		path = path[bracket+1:]
	}
	return path
}

func humanizeFieldName(name string) string {
	if name == "" {
		return ""
	}
	var out []rune
	var prev rune
	for i, r := range name {
		if i == 0 {
			out = append(out, unicode.ToUpper(r))
		} else {
			if unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				out = append(out, ' ')
			}
			out = append(out, unicode.ToLower(r))
		}
		prev = r
	}
	return string(out)
}
