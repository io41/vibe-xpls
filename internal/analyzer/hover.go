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
	if !ok || !a.documentActive(uri, parsed) || !parsed.IsStablePath(fieldPath) {
		return Hover{}, false
	}
	apiVersion, kind, ok := rootGVKForPath(parsed, fieldPath)
	if !ok {
		return Hover{}, false
	}
	field, ok := a.schemas.FieldDocumentation(apiVersion, kind, fieldPath)
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
	field, ok := a.schemas.FieldDocumentation(apiVersion, kind, occurrence.Path)
	if !ok {
		return Hover{}, false
	}
	return hoverFromField(field), true
}

func hoverFromField(field FieldDoc) Hover {
	return Hover{Markdown: fmt.Sprintf("**%s**\n\n%s", hoverTitle(field.Path), field.Description)}
}

func rootGVKForPath(parsed YAMLDocument, fieldPath string) (string, string, bool) {
	for _, occurrence := range parsed.occurrences {
		if occurrence.Path != fieldPath || !occurrence.Stable {
			continue
		}
		apiVersion, apiOK := parsed.RootValueForOccurrence(occurrence, "apiVersion")
		kind, kindOK := parsed.RootValueForOccurrence(occurrence, "kind")
		if apiOK && kindOK {
			return apiVersion, kind, true
		}
	}
	apiVersion, apiOK := parsed.Values["apiVersion"]
	kind, kindOK := parsed.Values["kind"]
	return apiVersion, kind, apiOK && kindOK
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
