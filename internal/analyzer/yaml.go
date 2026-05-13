package analyzer

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

type YAMLDocument struct {
	Mixed       MixedDocument
	Values      map[string]string
	StablePaths map[string]bool
	PathSpans   map[string]Span
	KeySpans    map[string]Span
	ValueSpans  map[string]Span
	Diagnostics []Diagnostic

	occurrences []pathOccurrence
}

type pathOccurrence struct {
	Path        string
	Stable      bool
	PathSpan    Span
	KeySpan     Span
	KeySpanOK   bool
	ValueSpan   Span
	ValueSpanOK bool
}

func ParseYAMLDocument(text string) YAMLDocument {
	mixed := ParseMixedDocument(text)
	doc := YAMLDocument{
		Mixed:       mixed,
		Values:      map[string]string{},
		StablePaths: map[string]bool{},
		PathSpans:   map[string]Span{},
		KeySpans:    map[string]Span{},
		ValueSpans:  map[string]Span{},
		Diagnostics: append([]Diagnostic(nil), mixed.TemplateDiagnostics...),
	}

	file, err := parseYAMLText(mixed.MaskedText)
	if err != nil {
		if diagnostic, ok := yamlDiagnosticFromError(err, mixed.RawText); ok {
			if !doc.overlapsTemplateAction(diagnostic.Span) {
				doc.Diagnostics = append(doc.Diagnostics, diagnostic)
			}
			doc.walkBestEffortPrefix(diagnostic.Span.Start)
		}
		return doc
	}

	doc.walkFile(file)
	return doc
}

func parseYAMLText(text string) (*ast.File, error) {
	return parser.ParseBytes([]byte(text), 0, parser.AllowDuplicateMapKey())
}

func (d *YAMLDocument) walkFile(file *ast.File) {
	for _, yamlDoc := range file.Docs {
		if yamlDoc == nil || yamlDoc.Body == nil {
			continue
		}
		d.walkNode(yamlDoc.Body, "", true)
	}
}

func (d *YAMLDocument) walkBestEffortPrefix(errorOffset int) {
	prefixEnd := lineStartForOffset(d.Mixed.MaskedText, errorOffset)
	if prefixEnd <= 0 {
		return
	}
	file, err := parseYAMLText(d.Mixed.MaskedText[:prefixEnd])
	if err != nil {
		return
	}
	d.walkFile(file)
}

func (d YAMLDocument) IsStablePath(path string) bool {
	return d.StablePaths[path]
}

func (d YAMLDocument) PathAtOffset(offset int) (string, bool) {
	if offset < 0 || offset >= len(d.Mixed.RawText) {
		return "", false
	}
	if d.offsetInTemplateAction(offset) {
		return "", false
	}

	bestPath := ""
	bestLen := math.MaxInt
	bestDepth := -1
	for _, occurrence := range d.occurrences {
		if !occurrence.Stable || !spanContains(occurrence.PathSpan, offset) {
			continue
		}
		spanLen := occurrence.PathSpan.End - occurrence.PathSpan.Start
		depth := pathDepth(occurrence.Path)
		if spanLen < bestLen || (spanLen == bestLen && depth > bestDepth) {
			bestPath = occurrence.Path
			bestLen = spanLen
			bestDepth = depth
		}
	}
	if bestPath == "" {
		return "", false
	}
	return bestPath, true
}

func (d *YAMLDocument) walkNode(node ast.Node, path string, stable bool) {
	switch n := node.(type) {
	case *ast.MappingNode:
		for _, entry := range n.Values {
			d.walkMappingValue(entry, path, stable)
		}
	case *ast.SequenceNode:
		for idx, value := range n.Values {
			elementPath := fmt.Sprintf("%s[%d]", path, idx)
			entrySpan, entryOK := d.sequenceEntrySpan(n, idx)
			valueSpan, valueOK := d.nodeSpan(value)
			elementStable := stable && !d.overlapsKnownSpan(entrySpan, entryOK) && !d.overlapsKnownSpan(valueSpan, valueOK)
			d.recordPath(elementPath, elementStable, entrySpan, entryOK, Span{}, false, valueSpan, valueOK)
			d.walkNode(value, elementPath, elementStable)
		}
	case *ast.MappingValueNode:
		d.walkMappingValue(n, path, stable)
	}
}

func (d *YAMLDocument) walkMappingValue(entry *ast.MappingValueNode, parentPath string, parentStable bool) {
	if entry == nil || entry.Key == nil {
		return
	}
	key, ok := yamlKeyName(entry.Key)
	if !ok || key == "" {
		return
	}
	path := joinYAMLPath(parentPath, key)
	keySpan, keyOK := d.keyNodeSpan(entry.Key)
	valueSpan, valueOK := d.nodeSpan(entry.Value)
	entrySpan, entryOK := d.mappingValueSpan(entry)
	stable := parentStable && keyOK && !d.overlapsTemplateAction(keySpan)
	scalar, scalarOK := scalarValue(entry.Value)
	if scalarOK && d.scalarValueOverlapsTemplate(entry, scalar, valueSpan, valueOK, entrySpan, entryOK) {
		stable = false
	}
	if nilScalarValue(entry.Value) && d.mappingValueLineOverlapsTemplate(entry) {
		stable = false
	}

	d.recordPath(path, stable, entrySpan, entryOK, keySpan, keyOK, valueSpan, valueOK)
	if stable && scalarOK && valueOK && !d.overlapsTemplateAction(valueSpan) {
		d.Values[path] = scalar
	}
	d.walkNode(entry.Value, path, stable)
}

func (d YAMLDocument) scalarValueOverlapsTemplate(entry *ast.MappingValueNode, scalar string, valueSpan Span, valueOK bool, entrySpan Span, entryOK bool) bool {
	if d.overlapsKnownSpan(valueSpan, valueOK) {
		return true
	}
	if d.mappingValueLineOverlapsTemplate(entry) && (!valueOK || scalar == "" || d.overlapsKnownSpan(entrySpan, entryOK)) {
		return true
	}
	return false
}

func (d *YAMLDocument) recordPath(path string, stable bool, pathSpan Span, pathOK bool, keySpan Span, keyOK bool, valueSpan Span, valueOK bool) {
	if path == "" {
		return
	}
	if stable {
		d.StablePaths[path] = true
	} else if _, ok := d.StablePaths[path]; !ok {
		d.StablePaths[path] = false
	}
	effectivePathSpan, effectivePathOK := pathSpan, pathOK
	if !effectivePathOK && keyOK && valueOK {
		effectivePathSpan, effectivePathOK = unionSpan(keySpan, valueSpan), true
	} else if !effectivePathOK && keyOK {
		effectivePathSpan, effectivePathOK = keySpan, true
	} else if !effectivePathOK && valueOK {
		effectivePathSpan, effectivePathOK = valueSpan, true
	}
	if effectivePathOK {
		d.PathSpans[path] = effectivePathSpan
		d.occurrences = append(d.occurrences, pathOccurrence{
			Path:        path,
			Stable:      stable,
			PathSpan:    effectivePathSpan,
			KeySpan:     keySpan,
			KeySpanOK:   keyOK,
			ValueSpan:   valueSpan,
			ValueSpanOK: valueOK,
		})
	}
	if keyOK {
		d.KeySpans[path] = keySpan
	}
	if valueOK {
		d.ValueSpans[path] = valueSpan
	}
}

func yamlKeyName(key ast.MapKeyNode) (string, bool) {
	switch n := key.(type) {
	case ast.ScalarNode:
		return fmt.Sprint(n.GetValue()), true
	case *ast.MappingKeyNode:
		return yamlNodeName(n.Value)
	default:
		name := strings.TrimSpace(key.String())
		return name, name != ""
	}
}

func yamlNodeName(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case ast.ScalarNode:
		return fmt.Sprint(n.GetValue()), true
	default:
		if node == nil {
			return "", false
		}
		name := strings.TrimSpace(node.String())
		return name, name != ""
	}
}

func scalarValue(node ast.Node) (string, bool) {
	scalar, ok := node.(ast.ScalarNode)
	if !ok {
		return "", false
	}
	value := scalar.GetValue()
	if value == nil {
		return "", false
	}
	return fmt.Sprint(value), true
}

func nilScalarValue(node ast.Node) bool {
	scalar, ok := node.(ast.ScalarNode)
	return ok && scalar.GetValue() == nil
}

func joinYAMLPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func (d YAMLDocument) mappingValueSpan(entry *ast.MappingValueNode) (Span, bool) {
	if entry == nil {
		return Span{}, false
	}
	var span Span
	ok := false
	keySpan, keyOK := d.keyNodeSpan(entry.Key)
	span, ok = unionOptionalSpan(span, ok, keySpan, keyOK)
	startSpan, startOK := d.tokenSpan(entry.Start)
	span, ok = unionOptionalSpan(span, ok, startSpan, startOK)
	valueSpan, valueOK := d.nodeSpan(entry.Value)
	span, ok = unionOptionalSpan(span, ok, valueSpan, valueOK)
	return span, ok
}

func (d YAMLDocument) sequenceEntrySpan(seq *ast.SequenceNode, idx int) (Span, bool) {
	if seq == nil {
		return Span{}, false
	}
	if idx >= 0 && idx < len(seq.Entries) {
		return d.nodeSpan(seq.Entries[idx])
	}
	if idx >= 0 && idx < len(seq.Values) {
		return d.nodeSpan(seq.Values[idx])
	}
	return Span{}, false
}

func (d YAMLDocument) nodeSpan(node ast.Node) (Span, bool) {
	switch n := node.(type) {
	case nil:
		return Span{}, false
	case *ast.MappingNode:
		return d.mappingSpan(n)
	case *ast.MappingValueNode:
		return d.mappingValueSpan(n)
	case *ast.SequenceNode:
		return d.sequenceSpan(n)
	case *ast.SequenceEntryNode:
		return d.sequenceEntryNodeSpan(n)
	case *ast.LiteralNode:
		return d.literalSpan(n)
	default:
		return d.tokenSpan(node.GetToken())
	}
}

func (d YAMLDocument) mappingSpan(node *ast.MappingNode) (Span, bool) {
	if node == nil {
		return Span{}, false
	}
	var span Span
	ok := false
	for _, entry := range node.Values {
		entrySpan, entryOK := d.mappingValueSpan(entry)
		if !entryOK {
			continue
		}
		if !ok {
			span = entrySpan
			ok = true
			continue
		}
		span = unionSpan(span, entrySpan)
	}
	if ok {
		return span, true
	}
	return d.tokenSpan(node.GetToken())
}

func (d YAMLDocument) sequenceSpan(node *ast.SequenceNode) (Span, bool) {
	if node == nil {
		return Span{}, false
	}
	var span Span
	ok := false
	for idx := range node.Values {
		entrySpan, entryOK := d.sequenceEntrySpan(node, idx)
		if !entryOK {
			continue
		}
		if !ok {
			span = entrySpan
			ok = true
			continue
		}
		span = unionSpan(span, entrySpan)
	}
	if ok {
		return span, true
	}
	return d.tokenSpan(node.GetToken())
}

func (d YAMLDocument) sequenceEntryNodeSpan(node *ast.SequenceEntryNode) (Span, bool) {
	if node == nil {
		return Span{}, false
	}
	startSpan, startOK := d.tokenSpan(node.Start)
	valueSpan, valueOK := d.nodeSpan(node.Value)
	switch {
	case startOK && valueOK:
		return unionSpan(startSpan, valueSpan), true
	case startOK:
		return startSpan, true
	case valueOK:
		return valueSpan, true
	default:
		return Span{}, false
	}
}

func (d YAMLDocument) literalSpan(node *ast.LiteralNode) (Span, bool) {
	markerSpan, ok := d.tokenSpan(node.GetToken())
	if !ok {
		return Span{}, false
	}
	markerLineStart := lineStartForOffset(d.Mixed.RawText, markerSpan.Start)
	markerIndent := lineIndent(d.Mixed.RawText, markerLineStart)
	end := lineEndIncludingNewline(d.Mixed.RawText, markerSpan.Start)
	for lineStart := end; lineStart < len(d.Mixed.RawText); {
		lineEnd := lineEndIncludingNewline(d.Mixed.RawText, lineStart)
		if isBlankLine(d.Mixed.RawText[lineStart:lineEnd]) {
			end = lineEnd
			lineStart = lineEnd
			continue
		}
		if lineIndent(d.Mixed.RawText, lineStart) <= markerIndent {
			break
		}
		end = lineEnd
		lineStart = lineEnd
	}
	return Span{Start: markerSpan.Start, End: end}, true
}

func (d YAMLDocument) keyNodeSpan(key ast.MapKeyNode) (Span, bool) {
	if key == nil {
		return Span{}, false
	}
	if mappingKey, ok := key.(*ast.MappingKeyNode); ok && mappingKey.Value != nil {
		return d.tokenSpanWithWidth(mappingKey.Value.GetToken(), keyTokenTextWidth(mappingKey.Value.GetToken()))
	}
	return d.tokenSpanWithWidth(key.GetToken(), keyTokenTextWidth(key.GetToken()))
}

func (d YAMLDocument) tokenSpan(tk *token.Token) (Span, bool) {
	return d.tokenSpanWithWidth(tk, tokenTextWidth(tk))
}

func (d YAMLDocument) tokenSpanWithWidth(tk *token.Token, width int) (Span, bool) {
	if tk == nil || tk.Position == nil {
		return Span{}, false
	}
	start := tk.Position.Offset - 1
	if start < 0 || start > len(d.Mixed.RawText) {
		return Span{}, false
	}
	if width == 0 {
		width = 1
	}
	end := start + width
	if end > len(d.Mixed.RawText) {
		end = len(d.Mixed.RawText)
	}
	return Span{Start: start, End: end}, true
}

type yamlPositionedError interface {
	error
	GetToken() *token.Token
	GetMessage() string
}

func yamlDiagnosticFromError(err error, text string) (Diagnostic, bool) {
	diagnostic := Diagnostic{
		Source:   "yaml",
		Severity: "error",
		Message:  err.Error(),
	}
	var yamlErr yamlPositionedError
	if errors.As(err, &yamlErr) {
		diagnostic.Message = yamlErr.GetMessage()
		diagnostic.Span = spanFromToken(yamlErr.GetToken(), text)
		return diagnostic, true
	}
	diagnostic.Span = Span{Start: 0, End: 0}
	return diagnostic, true
}

func spanFromToken(tk *token.Token, text string) Span {
	if tk == nil || tk.Position == nil {
		return Span{Start: 0, End: 0}
	}
	start := tk.Position.Offset - 1
	if start < 0 {
		start = 0
	}
	if start > len(text) {
		start = len(text)
	}
	width := tokenTextWidth(tk)
	if width == 0 {
		width = 1
	}
	end := start + width
	if end > len(text) {
		end = len(text)
	}
	return Span{Start: start, End: end}
}

func tokenTextWidth(tk *token.Token) int {
	if tk == nil {
		return 0
	}
	origin := strings.TrimLeft(strings.TrimRight(tk.Origin, " \t\r\n"), " \t")
	if origin != "" {
		return len(origin)
	}
	return len(tk.Value)
}

func keyTokenTextWidth(tk *token.Token) int {
	if tk == nil {
		return 0
	}
	origin := strings.TrimRight(tk.Origin, " \t\r\n")
	if origin != "" {
		if width, ok := mappingKeySourceWidth(origin); ok {
			return width
		}
		if isQuotedSource(origin) {
			return len(origin)
		}
	}
	return len(tk.Value)
}

func mappingKeySourceWidth(origin string) (int, bool) {
	var quote byte
	for i := 0; i < len(origin); i++ {
		c := origin[i]
		if quote == 0 {
			if c == ':' {
				return i, true
			}
			if c == '"' || c == '\'' || c == '`' {
				quote = c
			}
			continue
		}
		if quote == '`' {
			if c == '`' {
				quote = 0
			}
			continue
		}
		if c == '\\' {
			i++
			continue
		}
		if c == quote {
			quote = 0
		}
	}
	return 0, false
}

func isQuotedSource(text string) bool {
	return len(text) >= 2 && (text[0] == '"' || text[0] == '\'' || text[0] == '`')
}

func (d YAMLDocument) overlapsTemplateAction(span Span) bool {
	for _, action := range d.Mixed.Actions {
		if spansOverlap(span, action.Span) {
			return true
		}
	}
	return false
}

func (d YAMLDocument) overlapsKnownSpan(span Span, ok bool) bool {
	return ok && d.overlapsTemplateAction(span)
}

func (d YAMLDocument) mappingValueLineOverlapsTemplate(entry *ast.MappingValueNode) bool {
	if entry == nil {
		return false
	}
	colonSpan, ok := d.tokenSpan(entry.Start)
	if !ok {
		return false
	}
	valueLine := Span{Start: colonSpan.End, End: lineContentEndForOffset(d.Mixed.RawText, colonSpan.End)}
	for _, action := range d.Mixed.Actions {
		if spansOverlap(valueLine, action.Span) {
			return true
		}
	}
	return false
}

func (d YAMLDocument) offsetInTemplateAction(offset int) bool {
	for _, action := range d.Mixed.Actions {
		if spanContains(action.Span, offset) {
			return true
		}
	}
	return false
}

func spansOverlap(a, b Span) bool {
	return a.Start < b.End && b.Start < a.End
}

func spanContains(span Span, offset int) bool {
	return span.Start <= offset && offset < span.End
}

func unionSpan(a, b Span) Span {
	if b.Start < a.Start {
		a.Start = b.Start
	}
	if b.End > a.End {
		a.End = b.End
	}
	return a
}

func unionOptionalSpan(current Span, currentOK bool, next Span, nextOK bool) (Span, bool) {
	if !nextOK {
		return current, currentOK
	}
	if !currentOK {
		return next, true
	}
	return unionSpan(current, next), true
}

func pathDepth(path string) int {
	depth := 1
	for _, r := range path {
		if r == '.' || r == '[' {
			depth++
		}
	}
	return depth
}

func lineStartForOffset(text string, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	for offset > 0 && text[offset-1] != '\n' {
		offset--
	}
	return offset
}

func lineContentEndForOffset(text string, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	for offset < len(text) && text[offset] != '\n' {
		offset++
	}
	if offset > 0 && text[offset-1] == '\r' {
		offset--
	}
	return offset
}

func lineEndIncludingNewline(text string, offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	for offset < len(text) {
		offset++
		if text[offset-1] == '\n' {
			break
		}
	}
	return offset
}

func lineIndent(text string, lineStart int) int {
	indent := 0
	for lineStart+indent < len(text) {
		switch text[lineStart+indent] {
		case ' ', '\t':
			indent++
		default:
			return indent
		}
	}
	return indent
}

func isBlankLine(text string) bool {
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case ' ', '\t', '\r', '\n':
		default:
			return false
		}
	}
	return true
}
