package analyzer

import "strings"

func (a *Analyzer) Diagnostics(uri string) []Diagnostic {
	doc, ok := a.docs.Get(uri)
	if !ok {
		return nil
	}
	if a.documentExceedsLimit(doc) {
		if !a.documentActiveByCheapSignal(uri) {
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
	case "Composition", "CompositeResourceDefinition", "CustomResourceDefinition", "Configuration":
		return true
	default:
		return false
	}
}
