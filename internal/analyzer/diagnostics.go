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
		if len(parsed.Diagnostics) == 0 || !hasBoundedCrossplaneRootSignal(doc.Text) {
			return nil
		}
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
		if occurrence.Path == "apiVersion" && isCrossplaneCoreAPIVersion(occurrence.Value) {
			return true
		}
	}
	for _, occurrence := range parsed.occurrences {
		if !occurrence.Stable || !occurrence.ValueOK || occurrence.Path != "kind" {
			continue
		}
		if hasCrossplaneShapeForKind(parsed, occurrence.DocumentIndex, occurrence.Value) {
			return true
		}
	}
	return false
}

func isCrossplaneCoreAPIVersion(apiVersion string) bool {
	group, _, ok := strings.Cut(apiVersion, "/")
	return ok && (group == "crossplane.io" || strings.HasSuffix(group, ".crossplane.io"))
}

func hasCrossplaneShapeForKind(parsed YAMLDocument, documentIndex int, kind string) bool {
	for _, occurrence := range parsed.occurrences {
		if occurrence.DocumentIndex != documentIndex || !occurrence.Stable {
			continue
		}
		if isCrossplaneShapePathForKind(kind, occurrence.Path) {
			return true
		}
	}
	return false
}

func isCrossplaneShapePathForKind(kind string, path string) bool {
	switch kind {
	case "Composition":
		return pathMatchesOrDescendsFrom(path, "spec.compositeTypeRef")
	case "CompositeResourceDefinition":
		return pathMatchesOrDescendsFrom(path, "spec.group") || pathMatchesOrDescendsFrom(path, "spec.names")
	case "Configuration":
		return pathMatchesOrDescendsFrom(path, "spec.dependsOn")
	default:
		return false
	}
}

func pathMatchesOrDescendsFrom(path, parent string) bool {
	return path == parent || strings.HasPrefix(path, parent+".") || strings.HasPrefix(path, parent+"[")
}

func hasBoundedCrossplaneRootSignal(text string) bool {
	if len(text) > rootSignalSniffBytes {
		text = text[:rootSignalSniffBytes]
	}
	signals := boundedDocumentRootSignals{shapePaths: map[string]struct{}{}}
	var stack []sniffPathEntry
	blockScalarParentIndent := -1
	for lineStart, lines := 0, 0; lineStart < len(text) && lines < rootSignalSniffLines; lines++ {
		lineEnd := lineStart
		for lineEnd < len(text) && text[lineEnd] != '\n' {
			lineEnd++
		}
		line := strings.TrimSuffix(text[lineStart:lineEnd], "\r")
		if blockScalarParentIndent >= 0 {
			if strings.TrimSpace(line) == "" || leadingSpaces(line) > blockScalarParentIndent {
				if lineEnd == len(text) {
					break
				}
				lineStart = lineEnd + 1
				continue
			}
			blockScalarParentIndent = -1
		}
		if isDocumentSeparatorLine(line) {
			if signals.hasKindShapeSignal() {
				return true
			}
			signals = boundedDocumentRootSignals{shapePaths: map[string]struct{}{}}
			stack = nil
			lineStart = lineEnd + 1
			continue
		}
		if value, ok := rootLevelScalarLineValue(line, "apiVersion"); ok && isCrossplaneCoreAPIVersion(value) {
			return true
		}
		if value, ok := rootLevelScalarLineValue(line, "kind"); ok {
			signals.kind = value
		}
		if path, value, indent, ok := sniffMappingPath(line, &stack); ok {
			signals.shapePaths[path] = struct{}{}
			if isBlockScalarHeader(value) {
				blockScalarParentIndent = indent
			}
		}
		if lineEnd == len(text) {
			break
		}
		lineStart = lineEnd + 1
	}
	return signals.hasKindShapeSignal()
}

type boundedDocumentRootSignals struct {
	kind       string
	shapePaths map[string]struct{}
}

func (s boundedDocumentRootSignals) hasKindShapeSignal() bool {
	for path := range s.shapePaths {
		if isCrossplaneShapePathForKind(s.kind, path) {
			return true
		}
	}
	return false
}

func isDocumentSeparatorLine(line string) bool {
	trimmed := strings.TrimSpace(stripInlineComment(line))
	return trimmed == "---" || trimmed == "..."
}

type sniffPathEntry struct {
	indent int
	path   string
}

func sniffMappingPath(line string, stack *[]sniffPathEntry) (string, string, int, bool) {
	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
		return "", "", 0, false
	}
	indent := leadingSpaces(line)
	if indent < len(line) && line[indent] == '\t' {
		return "", "", 0, false
	}
	trimmed := line[indent:]
	if strings.HasPrefix(trimmed, "- ") {
		return "", "", 0, false
	}
	key, value, ok := splitSimpleMappingLine(trimmed)
	if !ok {
		return "", "", 0, false
	}
	for len(*stack) > 0 && indent <= (*stack)[len(*stack)-1].indent {
		*stack = (*stack)[:len(*stack)-1]
	}
	path := key
	if len(*stack) > 0 {
		path = (*stack)[len(*stack)-1].path + "." + key
	}
	if strings.TrimSpace(stripInlineComment(value)) == "" {
		*stack = append(*stack, sniffPathEntry{indent: indent, path: path})
	}
	return path, value, indent, true
}

func leadingSpaces(line string) int {
	for i := 0; i < len(line); i++ {
		if line[i] != ' ' {
			return i
		}
	}
	return len(line)
}

func splitSimpleMappingLine(line string) (string, string, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, "{}[]&*!,|>%'\"") {
		return "", "", false
	}
	return key, value, true
}

func isBlockScalarHeader(value string) bool {
	value = strings.TrimSpace(stripInlineComment(value))
	if value == "" || (value[0] != '|' && value[0] != '>') {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if c != '+' && c != '-' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
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
