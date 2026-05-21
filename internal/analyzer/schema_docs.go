package analyzer

import (
	"bytes"
	"encoding/json"
	"strings"
)

func fieldCompletionDocumentation(field FieldDoc) string {
	var sections []string
	if desc := strings.TrimSpace(field.Description); desc != "" {
		sections = append(sections, desc)
	}
	var facts []string
	if fieldType := strings.TrimSpace(field.Type); fieldType != "" {
		facts = append(facts, "_Type: "+fieldType+"_")
	}
	if field.Required {
		facts = append(facts, "_Required_")
	}
	if field.Default != nil {
		facts = append(facts, "_Default: "+compactJSON(*field.Default)+"_")
	}
	if len(field.Enum) != 0 {
		facts = append(facts, "_Allowed: "+strings.Join(field.Enum, ", ")+"_")
	}
	if deprecated := strings.TrimSpace(field.Deprecated); deprecated != "" {
		facts = append(facts, "_Deprecated: "+deprecated+"_")
	}
	if len(facts) != 0 {
		sections = append(sections, strings.Join(facts, "\n"))
	}
	return strings.Join(sections, "\n\n")
}

func compactJSON(raw json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err == nil {
		return buf.String()
	}
	return strings.TrimSpace(string(raw))
}
