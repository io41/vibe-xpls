package analyzer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var schemaArrayIndexPattern = regexp.MustCompile(`\[\d+\]`)

type Completion struct {
	Items  []CompletionItem
	Reason SuppressionReason
}

type CompletionItem struct {
	Label         string
	Path          string
	Documentation string
	SortText      string
	TextEdit      *CompletionTextEdit
}

type CompletionTextEdit struct {
	Replace Span
	NewText string
}

func (a *Analyzer) Completion(uri, parentPath string) Completion {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Completion{}
	}
	root, ok := rootContextForCompletionParent(parsed, parentPath)
	if !ok {
		return Completion{}
	}
	gvk := SourceGVK{APIVersion: root.apiVersion, Kind: root.kind}
	schemaParentPath := schemaPathFromParsedPath(parentPath)
	if a.schemas.HasWorkspaceSchema(gvk) {
		return completionFromWorkspaceSchema(a.schemas, root.apiVersion, root.kind, schemaParentPath)
	}
	if schema, ok := a.workspaceSchemaForURI(uri, gvk); ok {
		return completionFromFields(fieldsFromSchema(schema), schemaParentPath)
	}
	resolution := a.resolveSchemaRelease(uri, gvk)
	if !resolution.OK {
		return Completion{Reason: resolution.Reason}
	}
	return completionFromSchema(a.schemas, resolution.Release, root.apiVersion, root.kind, schemaParentPath)
}

func (a *Analyzer) CompletionAtOffset(uri string, offset int) Completion {
	_, parsed, ok := a.currentYAMLDocument(uri)
	if !ok || !a.documentActive(uri, parsed) {
		return Completion{}
	}
	context, reason, ok := completionContextAtOffset(parsed, offset)
	if !ok {
		return Completion{Reason: reason}
	}
	if malformedYAMLContextAtOffset(parsed, offset) {
		return Completion{Reason: SuppressionMalformedYAMLContext}
	}
	apiVersion, apiOK := parsed.RootValueForOccurrence(context.rootOccurrence, "apiVersion")
	kind, kindOK := parsed.RootValueForOccurrence(context.rootOccurrence, "kind")
	if !apiOK || !kindOK {
		return Completion{Reason: SuppressionMissingRootGVK}
	}
	gvk := SourceGVK{APIVersion: apiVersion, Kind: kind}
	legacyWorkspaceSchema := a.schemas.HasWorkspaceSchema(gvk)
	schema, scopedWorkspaceSchema := a.workspaceSchemaForURI(uri, gvk)
	var resolution schemaResolution
	if !legacyWorkspaceSchema && !scopedWorkspaceSchema {
		if !a.schemas.bundleStatus.OK {
			return Completion{Reason: SuppressionBundleDisabled}
		}
		resolution = a.resolveSchemaRelease(uri, gvk)
		if !resolution.OK {
			return Completion{Reason: resolution.Reason}
		}
	}
	completion := Completion{}
	parentPaths := completionParentPaths(context.schemaParentPath)
	if !context.allowParentPaths && len(parentPaths) > 1 {
		parentPaths = parentPaths[:1]
	}
	selectedParentPath := ""
	for i, parentPath := range parentPaths {
		stabilityPath := parentPath
		if i == 0 {
			stabilityPath = context.parentPath
		}
		if stabilityPath != "" && !parsed.IsStablePath(stabilityPath) {
			continue
		}
		var candidate Completion
		schemaParentPath := schemaPathFromParsedPath(parentPath)
		if legacyWorkspaceSchema {
			candidate = completionFromWorkspaceSchema(a.schemas, apiVersion, kind, schemaParentPath)
		} else if scopedWorkspaceSchema {
			candidate = completionFromFields(fieldsFromSchema(schema), schemaParentPath)
		} else {
			candidate = completionFromSchema(a.schemas, resolution.Release, apiVersion, kind, schemaParentPath)
		}
		if i > 0 {
			candidate = filterExistingCompletionPaths(candidate, parsed, context.rootOccurrence.DocumentIndex)
		}
		completion = filterCompletion(candidate, context.prefix)
		if len(completion.Items) != 0 {
			selectedParentPath = parentPath
			break
		}
	}
	for i := range completion.Items {
		completion.Items[i].TextEdit = &CompletionTextEdit{
			Replace: context.replace,
			NewText: completionTextEditNewText(context, selectedParentPath, completion.Items[i]),
		}
	}
	return completion
}

func malformedYAMLContextAtOffset(parsed YAMLDocument, offset int) bool {
	currentLineStart := lineStartForOffset(parsed.Mixed.RawText, offset)
	for _, diagnostic := range parsed.Diagnostics {
		if diagnostic.Source != "yaml" || diagnostic.Severity != "error" {
			continue
		}
		diagnosticLineStart := lineStartForOffset(parsed.Mixed.RawText, diagnostic.Span.Start)
		if diagnosticLineStart < currentLineStart && !documentSeparatorBetween(parsed.Mixed.RawText, diagnostic.Span.Start, offset) {
			return true
		}
	}
	return false
}

func filterExistingCompletionPaths(completion Completion, parsed YAMLDocument, documentIndex int) Completion {
	if len(completion.Items) == 0 {
		return completion
	}
	existing := map[string]struct{}{}
	for _, occurrence := range parsed.occurrences {
		if occurrence.DocumentIndex == documentIndex {
			existing[schemaPathFromParsedPath(occurrence.Path)] = struct{}{}
		}
	}
	items := completion.Items[:0]
	for _, item := range completion.Items {
		if _, ok := existing[schemaPathFromParsedPath(item.Path)]; ok {
			continue
		}
		items = append(items, item)
	}
	return Completion{Items: items, Reason: completion.Reason}
}

type completionContext struct {
	parentPath       string
	schemaParentPath string
	prefix           string
	rootOccurrence   PathOccurrence
	replace          Span
	indent           string
	newTextPrefix    string
	useNewTextPrefix bool
	allowParentPaths bool
}

func completionContextAtOffset(parsed YAMLDocument, offset int) (completionContext, SuppressionReason, bool) {
	text := parsed.Mixed.RawText
	if len(text) == 0 {
		return completionContext{}, "", false
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}

	lineStart := lineStartForOffset(text, offset)
	lineEnd := lineContentEndForOffset(text, offset)
	beforeCursor := text[lineStart:offset]
	if colon := strings.LastIndex(beforeCursor, ":"); colon >= 0 {
		return completionContext{}, "", false
	}

	indentEnd := completionLineIndentEnd(text, lineStart, lineEnd)
	if lineIsBlockScalarContent(text, lineStart, indentEnd-lineStart) {
		return completionContext{}, "", false
	}
	rawPrefix := text[indentEnd:offset]
	sequenceContext, sequenceOK := sequenceItemKeyCompletionContext(text, indentEnd, offset)
	keyCandidate := rawPrefix
	prefix := strings.TrimSpace(rawPrefix)
	replace := Span{Start: lineStart, End: offset}
	newTextPrefix := ""
	useNewTextPrefix := false
	if sequenceOK {
		keyCandidate = text[sequenceContext.replace.Start:sequenceContext.replace.End]
		prefix = sequenceContext.prefix
		replace = sequenceContext.replace
		newTextPrefix = sequenceContext.newTextPrefix
		useNewTextPrefix = sequenceContext.useNewTextPrefix
	} else if strings.HasPrefix(strings.TrimLeft(rawPrefix, " \t"), "-") {
		return completionContext{}, "", false
	}
	if offsetInTemplateActionForCompletion(parsed, offset) {
		return completionContext{}, SuppressionUnstableTemplatePath, false
	}
	afterCursor := text[offset:lineEnd]
	if colon := strings.Index(afterCursor, ":"); colon >= 0 {
		return completionContext{}, "", false
	} else if strings.TrimSpace(afterCursor) != "" {
		return completionContext{}, "", false
	}
	keyCandidate = strings.TrimSpace(keyCandidate)
	if !isBareCompletionKeyPrefix(prefix) || !isBareCompletionKeyPrefix(keyCandidate) {
		return completionContext{}, "", false
	}

	parentPath, rootOccurrence, ok := parentCompletionContext(parsed, lineStart, indentEnd-lineStart)
	if !ok {
		return completionContext{}, "", false
	}
	schemaParentPath := parentPath
	if sequenceOK {
		schemaParentPath = arrayItemSchemaParentPath(parentPath)
	}
	return completionContext{
		parentPath:       parentPath,
		schemaParentPath: schemaParentPath,
		prefix:           prefix,
		rootOccurrence:   rootOccurrence,
		replace:          replace,
		indent:           text[lineStart:indentEnd],
		newTextPrefix:    newTextPrefix,
		useNewTextPrefix: useNewTextPrefix,
		allowParentPaths: !sequenceOK,
	}, "", true
}

type sequenceItemKeyContext struct {
	prefix           string
	replace          Span
	newTextPrefix    string
	useNewTextPrefix bool
}

func sequenceItemKeyCompletionContext(text string, indentEnd, offset int) (sequenceItemKeyContext, bool) {
	if indentEnd >= offset || text[indentEnd] != '-' {
		return sequenceItemKeyContext{}, false
	}
	keyStart := indentEnd + 1
	newTextPrefix := " "
	if keyStart < offset {
		switch text[keyStart] {
		case ' ':
			keyStart++
			newTextPrefix = ""
		case '\t':
			return sequenceItemKeyContext{}, false
		default:
			return sequenceItemKeyContext{}, false
		}
	}
	rawPrefix := text[keyStart:offset]
	if strings.TrimSpace(rawPrefix) != rawPrefix {
		return sequenceItemKeyContext{}, false
	}
	prefix := strings.TrimSpace(rawPrefix)
	if !isBareCompletionKeyPrefix(prefix) || !isBareCompletionKeyPrefix(rawPrefix) {
		return sequenceItemKeyContext{}, false
	}
	return sequenceItemKeyContext{
		prefix:           prefix,
		replace:          Span{Start: keyStart, End: offset},
		newTextPrefix:    newTextPrefix,
		useNewTextPrefix: true,
	}, true
}

func arrayItemSchemaParentPath(parentPath string) string {
	if parentPath == "" {
		return ""
	}
	return parentPath + "[0]"
}

func completionTextEditNewText(context completionContext, parentPath string, item CompletionItem) string {
	if context.useNewTextPrefix {
		return context.newTextPrefix + item.Label + ":"
	}
	if parentPath != "" && parentPath == context.schemaParentPath {
		return context.indent + item.Label + ":"
	}
	return completionItemIndent(item) + item.Label + ":"
}

func rootContextForCompletionParent(parsed YAMLDocument, parentPath string) (rootContext, bool) {
	if parentPath == "" {
		return singleStableRootContext(parsed)
	}
	if pathExists(parsed, parentPath) {
		return rootContextForExistingPath(parsed, parentPath)
	}
	return singleStableRootContext(parsed)
}

func pathExists(parsed YAMLDocument, path string) bool {
	for _, occurrence := range parsed.occurrences {
		if occurrence.Path == path {
			return true
		}
	}
	return false
}

func completionFromSchema(schemas *SchemaIndex, release CrossplaneRelease, apiVersion, kind, parentPath string) Completion {
	return completionFromFields(schemas.FieldsForRelease(release, apiVersion, kind), parentPath)
}

func completionFromWorkspaceSchema(schemas *SchemaIndex, apiVersion, kind, parentPath string) Completion {
	return completionFromFields(schemas.Fields(apiVersion, kind), parentPath)
}

type completionCandidate struct {
	label         string
	path          string
	documentation string
	required      bool
}

func completionFromFields(fields []FieldDoc, parentPath string) Completion {
	parentPath = schemaPathFromParsedPath(parentPath)
	candidates := map[string]completionCandidate{}
	prefix := parentPath
	if prefix != "" {
		prefix += "."
	}
	for _, field := range fields {
		if !strings.HasPrefix(field.Path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(field.Path, prefix)
		if rest == "" {
			continue
		}
		label := immediateCompletionLabel(rest)
		if label == "" {
			continue
		}
		if parentPath == "" && label == "status" {
			continue
		}
		path := prefix + label
		candidate := candidates[label]
		if candidate.label == "" {
			candidate = completionCandidate{label: label, path: path}
		}
		if fieldIsImmediateCompletionChild(field.Path, prefix, label) {
			candidate.required = field.Required
			candidate.documentation = fieldCompletionDocumentation(field)
		}
		candidates[label] = candidate
	}
	items := completionItemsFromCandidates(parentPath, candidates)
	return Completion{Items: items}
}

func schemaPathFromParsedPath(path string) string {
	return schemaArrayIndexPattern.ReplaceAllString(path, "[]")
}

func immediateCompletionLabel(rest string) string {
	split := strings.IndexAny(rest, ".[")
	if split < 0 {
		return rest
	}
	return rest[:split]
}

func fieldIsImmediateCompletionChild(fieldPath, prefix, label string) bool {
	rest := strings.TrimPrefix(fieldPath, prefix)
	return rest == label || rest == label+"[]"
}

func completionItemsFromCandidates(parentPath string, candidates map[string]completionCandidate) []CompletionItem {
	sorted := make([]completionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		sorted = append(sorted, candidate)
	}
	sort.Slice(sorted, func(i, j int) bool {
		leftRank := rootCompletionRank(parentPath, sorted[i].label)
		rightRank := rootCompletionRank(parentPath, sorted[j].label)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if sorted[i].required != sorted[j].required {
			return sorted[i].required
		}
		return sorted[i].label < sorted[j].label
	})
	items := make([]CompletionItem, 0, len(sorted))
	for i, candidate := range sorted {
		items = append(items, CompletionItem{
			Label:         candidate.label,
			Path:          candidate.path,
			Documentation: candidate.documentation,
			SortText:      completionSortText(parentPath, candidate, i),
		})
	}
	return items
}

func rootCompletionRank(parentPath, label string) int {
	if parentPath != "" {
		return 100
	}
	switch label {
	case "apiVersion":
		return 0
	case "kind":
		return 1
	case "metadata":
		return 2
	case "spec":
		return 3
	default:
		return 100
	}
}

func completionSortText(parentPath string, item completionCandidate, index int) string {
	return fmt.Sprintf("%04d_%s", index, item.label)
}

func completionParentPaths(parentPath string) []string {
	paths := []string{parentPath}
	for parentPath != "" {
		if split := strings.LastIndex(parentPath, "."); split >= 0 {
			parentPath = parentPath[:split]
		} else {
			parentPath = ""
		}
		paths = append(paths, parentPath)
	}
	return paths
}

func completionItemIndent(item CompletionItem) string {
	if item.Path == "" {
		return ""
	}
	return strings.Repeat("  ", strings.Count(item.Path, "."))
}

func filterCompletion(completion Completion, prefix string) Completion {
	if prefix == "" {
		return completion
	}
	items := completion.Items[:0]
	for _, item := range completion.Items {
		if strings.HasPrefix(item.Label, prefix) {
			items = append(items, item)
		}
	}
	return Completion{Items: items, Reason: completion.Reason}
}

func offsetInTemplateActionForCompletion(parsed YAMLDocument, offset int) bool {
	if offset < len(parsed.Mixed.RawText) && parsed.offsetInTemplateAction(offset) {
		return true
	}
	return offset > 0 && parsed.offsetInTemplateAction(offset-1)
}

func completionLineIndentEnd(text string, lineStart, lineEnd int) int {
	offset := lineStart
	for offset < lineEnd {
		switch text[offset] {
		case ' ', '\t':
			offset++
		default:
			return offset
		}
	}
	return offset
}

func parentCompletionContext(parsed YAMLDocument, beforeOffset, indent int) (string, PathOccurrence, bool) {
	var nearest PathOccurrence
	nearestOK := false

	occurrences := append([]PathOccurrence(nil), parsed.occurrences...)
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].KeySpan.Start < occurrences[j].KeySpan.Start
	})

	type stackEntry struct {
		occurrence PathOccurrence
		indent     int
	}
	var stack []stackEntry
	for _, occurrence := range occurrences {
		if !occurrence.Stable || !occurrence.KeySpanOK || occurrence.KeySpan.Start >= beforeOffset {
			continue
		}
		if !nearestOK || occurrence.KeySpan.Start > nearest.KeySpan.Start {
			nearest = occurrence
			nearestOK = true
		}
		keyIndent := occurrence.KeySpan.Start - lineStartForOffset(parsed.Mixed.RawText, occurrence.KeySpan.Start)
		for len(stack) > 0 && stack[len(stack)-1].indent >= keyIndent {
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, stackEntry{occurrence: occurrence, indent: keyIndent})
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].indent < indent && !documentSeparatorBetween(parsed.Mixed.RawText, stack[i].occurrence.KeySpan.Start, beforeOffset) {
			return stack[i].occurrence.Path, stack[i].occurrence, true
		}
	}
	if nearestOK && !documentSeparatorBetween(parsed.Mixed.RawText, nearest.KeySpan.Start, beforeOffset) {
		return "", nearest, true
	}
	return "", PathOccurrence{}, false
}

func lineIsBlockScalarContent(text string, lineStart, indent int) bool {
	inBlockScalar := false
	blockScalarParentIndent := -1
	for scan := 0; scan < lineStart; {
		lineEnd := lineContentEndForOffset(text, scan)
		indentEnd := completionLineIndentEnd(text, scan, lineEnd)
		lineIndent := indentEnd - scan
		trimmed := strings.TrimSpace(text[indentEnd:lineEnd])
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if inBlockScalar {
				if lineIndent > blockScalarParentIndent {
					next := lineEndIncludingNewline(text, lineEnd)
					if next <= scan {
						break
					}
					scan = next
					continue
				}
				inBlockScalar = false
				blockScalarParentIndent = -1
			}
			if isDocumentSeparatorLine(trimmed) {
				inBlockScalar = false
				blockScalarParentIndent = -1
			} else if mappingLineStartsBlockScalar(trimmed) {
				inBlockScalar = true
				blockScalarParentIndent = lineIndent
			}
		}
		next := lineEndIncludingNewline(text, lineEnd)
		if next <= scan {
			break
		}
		scan = next
	}
	if !inBlockScalar {
		return false
	}
	lineEnd := lineContentEndForOffset(text, lineStart)
	indentEnd := completionLineIndentEnd(text, lineStart, lineEnd)
	trimmed := strings.TrimSpace(text[indentEnd:lineEnd])
	return trimmed == "" || indent > blockScalarParentIndent
}

func mappingLineStartsBlockScalar(trimmed string) bool {
	colon := strings.Index(trimmed, ":")
	if colon < 0 {
		return false
	}
	afterColon := strings.TrimSpace(trimmed[colon+1:])
	return strings.HasPrefix(afterColon, "|") || strings.HasPrefix(afterColon, ">")
}

func documentSeparatorBetween(text string, start, end int) bool {
	if start < 0 {
		start = 0
	}
	if end > len(text) {
		end = len(text)
	}
	for scan := lineStartForOffset(text, start); scan < end; {
		lineEnd := lineContentEndForOffset(text, scan)
		if lineEnd > start && isDocumentSeparatorLine(strings.TrimSpace(text[scan:lineEnd])) {
			return true
		}
		next := lineEndIncludingNewline(text, lineEnd)
		if next <= scan {
			break
		}
		scan = next
	}
	return false
}

func isBareCompletionKeyPrefix(prefix string) bool {
	for _, r := range prefix {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
