package analyzer

import "strings"

const (
	rootSignalSniffBytes = 4096
	rootSignalSniffLines = 64
)

func (a *Analyzer) Diagnostics(uri string) []Diagnostic {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return nil
	}
	if a.documentExceedsLimit(doc) {
		if !a.documentActiveByCheapSignal(uri) && !hasBoundedCrossplaneRootSignal(doc.Text) {
			return nil
		}
		return []Diagnostic{{
			URI:      uri,
			Source:   "analyzer",
			Severity: "warning",
			Message:  "document exceeds analyzer size limit; full analysis skipped",
			Span:     Span{Start: 0, End: 0},
		}}
	}
	parsed := ParseYAMLDocument(doc.Text)
	if !a.documentActive(uri, parsed) {
		return nil
	}
	diagnostics := make([]Diagnostic, 0, len(parsed.Diagnostics))
	for _, diagnostic := range parsed.Diagnostics {
		diagnostic.URI = uri
		diagnostics = append(diagnostics, diagnostic)
	}
	if max := a.limits.MaxDiagnosticsPerDoc; max > 0 && len(diagnostics) > max {
		diagnostics = diagnostics[:max]
	}
	return diagnostics
}

func (a *Analyzer) documentActive(uri string, parsed YAMLDocument) bool {
	if a.documentActiveByCheapSignal(uri) {
		return true
	}
	return hasCrossplaneRootSignal(parsed)
}

func (a *Analyzer) documentActiveByCheapSignal(uri string) bool {
	return a.hasPackageContext(uri) || isCrossplaneClassifiedFilename(uri)
}

func (a *Analyzer) hasPackageContext(uri string) bool {
	path, ok := filePathFromURI(uri)
	if !ok {
		return false
	}
	_, ok = a.workspace.PackageForFile(path)
	return ok
}

func isCrossplaneClassifiedFilename(uri string) bool {
	name := strings.ToLower(baseNameFromURI(uri))
	return name == "crossplane.yaml" ||
		name == "crossplane.yml" ||
		strings.HasSuffix(name, ".crossplane.yaml") ||
		strings.HasSuffix(name, ".crossplane.yml")
}

func hasCrossplaneRootSignal(parsed YAMLDocument) bool {
	for _, occurrence := range parsed.occurrences {
		if !occurrence.Stable || !occurrence.ValueOK {
			continue
		}
		switch occurrence.Path {
		case "apiVersion":
			if isCrossplaneCoreAPIVersion(occurrence.Value) {
				return true
			}
		case "kind":
			if isCrossplaneDocumentKind(occurrence.Value) {
				return true
			}
		}
	}
	return false
}

func isCrossplaneCoreAPIVersion(apiVersion string) bool {
	group, _, ok := strings.Cut(apiVersion, "/")
	return ok && (group == "crossplane.io" || strings.HasSuffix(group, ".crossplane.io"))
}

func isCrossplaneDocumentKind(kind string) bool {
	switch kind {
	case "Composition", "CompositeResourceDefinition", "Configuration":
		return true
	default:
		return false
	}
}

func hasBoundedCrossplaneRootSignal(text string) bool {
	if len(text) > rootSignalSniffBytes {
		text = text[:rootSignalSniffBytes]
	}
	for lineStart, lines := 0, 0; lineStart < len(text) && lines < rootSignalSniffLines; lines++ {
		lineEnd := lineStart
		for lineEnd < len(text) && text[lineEnd] != '\n' {
			lineEnd++
		}
		line := strings.TrimSuffix(text[lineStart:lineEnd], "\r")
		if value, ok := rootLevelScalarLineValue(line, "apiVersion"); ok && isCrossplaneCoreAPIVersion(value) {
			return true
		}
		if value, ok := rootLevelScalarLineValue(line, "kind"); ok && isCrossplaneDocumentKind(value) {
			return true
		}
		if lineEnd == len(text) {
			break
		}
		lineStart = lineEnd + 1
	}
	return false
}

func rootLevelScalarLineValue(line, key string) (string, bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' {
		return "", false
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	value, ok := strings.CutPrefix(line, key+":")
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(stripInlineComment(value))
	if value == "" || strings.Contains(value, "{{") || strings.Contains(value, "}}") {
		return "", false
	}
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return value, value != ""
}

func stripInlineComment(value string) string {
	for i := 0; i < len(value); i++ {
		if value[i] != '#' {
			continue
		}
		if i == 0 || value[i-1] == ' ' || value[i-1] == '\t' {
			return value[:i]
		}
	}
	return value
}
