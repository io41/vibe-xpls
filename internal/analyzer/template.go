package analyzer

import "strings"

type Span struct {
	Start int
	End   int
}

type TemplateAction struct {
	Text         string
	Span         Span
	Unterminated bool
}

type Diagnostic struct {
	URI      string
	Source   string
	Message  string
	Severity string
	Span     Span
}

type MixedDocument struct {
	RawText             string
	MaskedText          string
	Actions             []TemplateAction
	TemplateDiagnostics []Diagnostic
}

func ParseMixedDocument(text string) MixedDocument {
	actions, diagnostics := findTemplateActions(text)
	return MixedDocument{
		RawText:             text,
		MaskedText:          maskTemplateActions(text, actions),
		Actions:             actions,
		TemplateDiagnostics: diagnostics,
	}
}

func findTemplateActions(text string) ([]TemplateAction, []Diagnostic) {
	var actions []TemplateAction
	var diagnostics []Diagnostic
	for scan := 0; scan < len(text); {
		openRel := strings.Index(text[scan:], "{{")
		if openRel < 0 {
			break
		}
		start := scan + openRel
		end, ok := findTemplateActionEnd(text, start+2)
		if !ok {
			span := Span{Start: start, End: len(text)}
			actions = append(actions, TemplateAction{Text: text[start:], Span: span, Unterminated: true})
			return actions, append(diagnostics, Diagnostic{
				Source:   "template",
				Severity: "error",
				Message:  "template action is missing closing delimiter",
				Span:     span,
			})
		}
		actions = append(actions, TemplateAction{Text: text[start:end], Span: Span{Start: start, End: end}})
		scan = end
	}
	return actions, diagnostics
}

func findTemplateActionEnd(text string, scan int) (int, bool) {
	var quote byte
	for i := scan; i < len(text); i++ {
		c := text[i]
		if quote == 0 {
			if c == '}' && i+1 < len(text) && text[i+1] == '}' {
				return i + 2, true
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

func maskTemplateActions(text string, actions []TemplateAction) string {
	masked := []byte(text)
	for _, action := range actions {
		if action.Unterminated {
			maskBytes(masked, action.Span.Start, action.Span.End, ' ')
			continue
		}
		if start, end, ok := standaloneTemplateLine(text, action.Span); ok {
			maskBytes(masked, start, end, ' ')
			continue
		}
		maskBytes(masked, action.Span.Start, action.Span.End, 'x')
	}
	return string(masked)
}

func standaloneTemplateLine(text string, span Span) (int, int, bool) {
	lineStart := span.Start
	for lineStart > 0 && text[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := span.End
	for lineEnd < len(text) && text[lineEnd] != '\n' {
		lineEnd++
	}
	if lineEnd < len(text) && text[lineEnd] == '\n' {
		lineEnd++
	}
	contentEnd := lineEnd
	if contentEnd > 0 && text[contentEnd-1] == '\n' {
		contentEnd--
	}
	if contentEnd > 0 && text[contentEnd-1] == '\r' {
		contentEnd--
	}
	if span.End > contentEnd {
		return 0, 0, false
	}
	if !onlyHorizontalWhitespace(text[lineStart:span.Start]) || !onlyHorizontalWhitespace(text[span.End:contentEnd]) {
		return 0, 0, false
	}
	return lineStart, lineEnd, true
}

func onlyHorizontalWhitespace(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] != ' ' && text[i] != '\t' {
			return false
		}
	}
	return true
}

func maskBytes(text []byte, start, end int, replacement byte) {
	for i := start; i < end; i++ {
		if text[i] != '\n' && text[i] != '\r' {
			text[i] = replacement
		}
	}
}
