package analyzer

import "strings"

type Span struct {
	Start int
	End   int
}

type TemplateAction struct {
	Text string
	Span Span
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
		closeRel := strings.Index(text[start+2:], "}}")
		if closeRel < 0 {
			return actions, append(diagnostics, Diagnostic{
				Source:   "template",
				Severity: "error",
				Message:  "template action is missing closing delimiter",
				Span:     Span{Start: start, End: len(text)},
			})
		}
		end := start + 2 + closeRel + 2
		actions = append(actions, TemplateAction{Text: text[start:end], Span: Span{Start: start, End: end}})
		scan = end
	}
	return actions, diagnostics
}

func maskTemplateActions(text string, actions []TemplateAction) string {
	masked := []byte(text)
	for _, action := range actions {
		for i := action.Span.Start; i < action.Span.End; i++ {
			if masked[i] != '\n' && masked[i] != '\r' {
				masked[i] = 'x'
			}
		}
	}
	return string(masked)
}
