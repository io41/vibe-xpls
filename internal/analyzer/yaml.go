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
	if len(mixed.TemplateDiagnostics) > 0 {
		return doc
	}

	file, err := parser.ParseBytes([]byte(mixed.MaskedText), 0)
	if err != nil {
		if diagnostic, ok := yamlDiagnosticFromError(err, mixed.RawText); ok && !doc.overlapsTemplateAction(diagnostic.Span) {
			doc.Diagnostics = append(doc.Diagnostics, diagnostic)
		}
		return doc
	}

	for _, yamlDoc := range file.Docs {
		if yamlDoc == nil || yamlDoc.Body == nil {
			continue
		}
		doc.walkNode(yamlDoc.Body, "", true)
	}
	return doc
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
	for path, span := range d.PathSpans {
		if !d.IsStablePath(path) || !spanContains(span, offset) {
			continue
		}
		spanLen := span.End - span.Start
		depth := pathDepth(path)
		if spanLen < bestLen || (spanLen == bestLen && depth > bestDepth) {
			bestPath = path
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
			d.recordPath(elementPath, stable, entrySpan, entryOK, Span{}, false, valueSpan, valueOK)
			d.walkNode(value, elementPath, stable)
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
	keySpan, keyOK := d.nodeSpan(entry.Key)
	valueSpan, valueOK := d.nodeSpan(entry.Value)
	entrySpan, entryOK := d.mappingValueSpan(entry)
	stable := parentStable && keyOK && !d.overlapsTemplateAction(keySpan)

	d.recordPath(path, stable, entrySpan, entryOK, keySpan, keyOK, valueSpan, valueOK)
	if stable {
		if value, ok := scalarValue(entry.Value); ok {
			d.Values[path] = value
		}
	}
	d.walkNode(entry.Value, path, stable)
}

func (d *YAMLDocument) recordPath(path string, stable bool, pathSpan Span, pathOK bool, keySpan Span, keyOK bool, valueSpan Span, valueOK bool) {
	if path == "" {
		return
	}
	d.StablePaths[path] = stable
	if pathOK {
		d.PathSpans[path] = pathSpan
	} else if keyOK && valueOK {
		d.PathSpans[path] = unionSpan(keySpan, valueSpan)
	} else if keyOK {
		d.PathSpans[path] = keySpan
	} else if valueOK {
		d.PathSpans[path] = valueSpan
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
	return fmt.Sprint(scalar.GetValue()), true
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
	keySpan, keyOK := d.nodeSpan(entry.Key)
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

func (d YAMLDocument) tokenSpan(tk *token.Token) (Span, bool) {
	if tk == nil || tk.Position == nil {
		return Span{}, false
	}
	start := tk.Position.Offset - 1
	if start < 0 || start > len(d.Mixed.RawText) {
		return Span{}, false
	}
	width := tokenTextWidth(tk)
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
	origin := strings.TrimRight(tk.Origin, " \t\r\n")
	if origin != "" {
		return len(origin)
	}
	return len(tk.Value)
}

func (d YAMLDocument) overlapsTemplateAction(span Span) bool {
	for _, action := range d.Mixed.Actions {
		if spansOverlap(span, action.Span) {
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
