package yamltemplatemapping

import "strings"

type Position struct {
	Offset int
	Line   int
	Column int
}

type Span struct {
	Start Position
	End   Position
}

type TemplateAction struct {
	Text     string
	Body     string
	Span     Span
	BodySpan Span
}

type Diagnostic struct {
	Source         string
	Message        string
	Span           Span
	InsideTemplate bool
}

type Analysis struct {
	Original            string
	Masked              string
	Actions             []TemplateAction
	YAMLDiagnostics     []Diagnostic
	TemplateDiagnostics []Diagnostic
	Diagnostics         []Diagnostic
}

func Analyze(source string) Analysis {
	actions, templateDiagnostics := ParseTemplateActions(source)
	masked := MaskTemplateActions(source, actions)
	yamlDiagnostics := MapMaskedDiagnostics(source, ParseMaskedYAML(masked), actions)

	diagnostics := make([]Diagnostic, 0, len(yamlDiagnostics)+len(templateDiagnostics))
	diagnostics = append(diagnostics, yamlDiagnostics...)
	diagnostics = append(diagnostics, templateDiagnostics...)

	return Analysis{
		Original:            source,
		Masked:              masked,
		Actions:             actions,
		YAMLDiagnostics:     yamlDiagnostics,
		TemplateDiagnostics: templateDiagnostics,
		Diagnostics:         diagnostics,
	}
}

func ParseTemplateActions(source string) ([]TemplateAction, []Diagnostic) {
	var actions []TemplateAction
	var diagnostics []Diagnostic

	for scan := 0; scan < len(source); {
		openRel := strings.Index(source[scan:], "{{")
		if openRel < 0 {
			break
		}
		start := scan + openRel
		bodyStart := start + len("{{")
		closeRel := strings.Index(source[bodyStart:], "}}")
		if closeRel < 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Source:         "template",
				Message:        "template action is missing closing delimiter",
				Span:           spanFromOffsets(source, start, len(source)),
				InsideTemplate: true,
			})
			break
		}

		bodyEnd := bodyStart + closeRel
		end := bodyEnd + len("}}")
		action := TemplateAction{
			Text:     source[start:end],
			Body:     source[bodyStart:bodyEnd],
			Span:     spanFromOffsets(source, start, end),
			BodySpan: spanFromOffsets(source, bodyStart, bodyEnd),
		}
		actions = append(actions, action)
		diagnostics = append(diagnostics, validateTemplateAction(source, action)...)
		scan = end
	}

	return actions, diagnostics
}

func MaskTemplateActions(source string, actions []TemplateAction) string {
	masked := []byte(source)
	for _, action := range actions {
		for i := action.Span.Start.Offset; i < action.Span.End.Offset; i++ {
			if masked[i] == '\n' || masked[i] == '\r' {
				continue
			}
			masked[i] = 'x'
		}
	}
	return string(masked)
}

func ParseMaskedYAML(masked string) []Diagnostic {
	var diagnostics []Diagnostic
	blockIndent := -1

	forEachLine(masked, func(line string, lineStart int) {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)

		if blockIndent >= 0 {
			if trimmed == "" {
				return
			}
			indent := leadingSpaces(line)
			if indent > blockIndent {
				return
			}
			blockIndent = -1
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			return
		}

		indent := leadingSpaces(line)
		content := line[indent:]
		if content == "---" || strings.HasPrefix(content, "--- ") || content == "..." {
			return
		}

		if strings.HasPrefix(content, "-") {
			if len(content) > 1 && content[1] != ' ' && content[1] != '\t' {
				diagnostics = append(diagnostics, lineDiagnostic(masked, lineStart, indent, content, "list marker must be followed by a space"))
				return
			}
			item := strings.TrimSpace(content[1:])
			if isBlockScalarValue(item) {
				blockIndent = indent
			}
			return
		}

		colon := strings.Index(content, ":")
		if colon < 0 {
			diagnostics = append(diagnostics, lineDiagnostic(masked, lineStart, indent, content, "line is not a mapping, list item, document marker, or block content"))
			return
		}

		key := strings.TrimSpace(content[:colon])
		if key == "" {
			diagnostics = append(diagnostics, lineDiagnostic(masked, lineStart, indent, content, "mapping key is empty"))
			return
		}

		value := strings.TrimSpace(content[colon+1:])
		if isBlockScalarValue(value) {
			blockIndent = indent
		}
	})

	return diagnostics
}

func MapMaskedDiagnostics(original string, maskedDiagnostics []Diagnostic, actions []TemplateAction) []Diagnostic {
	mapped := make([]Diagnostic, 0, len(maskedDiagnostics))
	for _, diagnostic := range maskedDiagnostics {
		start := diagnostic.Span.Start.Offset
		end := diagnostic.Span.End.Offset
		if end < start {
			end = start
		}
		diagnostic.Span = spanFromOffsets(original, start, end)
		diagnostic.InsideTemplate = spanInsideTemplate(diagnostic.Span, actions)
		mapped = append(mapped, diagnostic)
	}
	return mapped
}

func PositionAt(source string, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(source) {
		offset = len(source)
	}

	line := 0
	column := 0
	for i := 0; i < offset; i++ {
		if source[i] == '\n' {
			line++
			column = 0
			continue
		}
		column++
	}

	return Position{
		Offset: offset,
		Line:   line,
		Column: column,
	}
}

func validateTemplateAction(source string, action TemplateAction) []Diagnostic {
	trimmed := strings.TrimSpace(action.Body)
	if trimmed == "" {
		start := action.BodySpan.Start.Offset
		end := start + 1
		if end > action.BodySpan.End.Offset {
			end = action.BodySpan.End.Offset
		}
		return []Diagnostic{{
			Source:         "template",
			Message:        "template action is empty",
			Span:           spanFromOffsets(source, start, end),
			InsideTemplate: true,
		}}
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 1 && requiresOperand(fields[0]) {
		start := action.BodySpan.Start.Offset + strings.Index(action.Body, fields[0])
		return []Diagnostic{{
			Source:         "template",
			Message:        "template action " + fields[0] + " requires an operand",
			Span:           spanFromOffsets(source, start, start+len(fields[0])),
			InsideTemplate: true,
		}}
	}

	if strings.HasSuffix(trimmed, "|") {
		start := action.BodySpan.Start.Offset + strings.LastIndex(action.Body, "|")
		return []Diagnostic{{
			Source:         "template",
			Message:        "template pipeline is missing a command after pipe",
			Span:           spanFromOffsets(source, start, start+1),
			InsideTemplate: true,
		}}
	}

	return nil
}

func requiresOperand(keyword string) bool {
	switch keyword {
	case "if", "range", "with":
		return true
	default:
		return false
	}
}

func spanInsideTemplate(span Span, actions []TemplateAction) bool {
	for _, action := range actions {
		if span.Start.Offset >= action.Span.Start.Offset && span.End.Offset <= action.Span.End.Offset {
			return true
		}
	}
	return false
}

func spanFromOffsets(source string, start, end int) Span {
	return Span{
		Start: PositionAt(source, start),
		End:   PositionAt(source, end),
	}
}

func lineDiagnostic(source string, lineStart, indent int, content, message string) Diagnostic {
	start := lineStart + indent
	end := start + len(strings.TrimRight(content, " \t"))
	return Diagnostic{
		Source:  "yaml",
		Message: message,
		Span:    spanFromOffsets(source, start, end),
	}
}

func isBlockScalarValue(value string) bool {
	return value == "|" || value == ">" || strings.HasPrefix(value, "|+") || strings.HasPrefix(value, "|-") || strings.HasPrefix(value, ">+") || strings.HasPrefix(value, ">-")
}

func leadingSpaces(line string) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}

func forEachLine(source string, visit func(line string, lineStart int)) {
	lineStart := 0
	for lineStart <= len(source) {
		newlineRel := strings.IndexByte(source[lineStart:], '\n')
		if newlineRel < 0 {
			visit(source[lineStart:], lineStart)
			return
		}

		lineEnd := lineStart + newlineRel
		visit(source[lineStart:lineEnd], lineStart)
		lineStart = lineEnd + 1
		if lineStart == len(source) {
			return
		}
	}
}
