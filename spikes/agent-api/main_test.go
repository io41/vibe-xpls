package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCommandsReturnValidJSON(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "list-compositions",
			args: []string{"list-compositions"},
		},
		{
			name: "find-schema",
			args: []string{"find-schema", "--api-version", "platform.example.org/v1alpha1", "--kind", "XBucket"},
		},
		{
			name: "validate-workspace",
			args: []string{"validate-workspace"},
		},
		{
			name: "render",
			args: []string{"render"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, response := runAndDecode(t, tt.args...)
			if code != 0 {
				t.Fatalf("expected exit code 0, got %d", code)
			}
			if !response.OK {
				t.Fatalf("expected ok true, got %#v", response)
			}
			if response.Command != tt.args[0] {
				t.Fatalf("expected command %q, got %q", tt.args[0], response.Command)
			}
		})
	}
}

func TestListCompositionsReturnsPipelineSummary(t *testing.T) {
	_, response := runAndDecode(t, "list-compositions")
	data := asObject(t, response.Data)
	compositions := asSlice(t, data["compositions"])
	if len(compositions) != 1 {
		t.Fatalf("expected one composition, got %d", len(compositions))
	}

	composition := asObject(t, compositions[0])
	if composition["mode"] != "Pipeline" {
		t.Fatalf("expected Pipeline mode, got %#v", composition["mode"])
	}
	pipeline := asSlice(t, composition["pipeline"])
	if len(pipeline) != 2 {
		t.Fatalf("expected two pipeline steps, got %d", len(pipeline))
	}
}

func TestFindSchemaReturnsFixtureFields(t *testing.T) {
	_, response := runAndDecode(t, "find-schema", "--api-version", "platform.example.org/v1alpha1", "--kind", "XBucket")
	data := asObject(t, response.Data)
	if data["found"] != true {
		t.Fatalf("expected found true, got %#v", data["found"])
	}

	schema := asObject(t, data["schema"])
	fields := asSlice(t, schema["fields"])
	if len(fields) == 0 {
		t.Fatal("expected schema fields")
	}
}

func TestValidateWorkspaceReportsReadOnlyLimits(t *testing.T) {
	_, response := runAndDecode(t, "validate-workspace")
	data := asObject(t, response.Data)
	if data["valid"] != true {
		t.Fatalf("expected valid true, got %#v", data["valid"])
	}

	limits := asSlice(t, data["limits"])
	if len(limits) == 0 {
		t.Fatal("expected validation limits")
	}
}

func TestRenderIsFixtureBackedAndNonExecuting(t *testing.T) {
	_, response := runAndDecode(t, "render")
	data := asObject(t, response.Data)
	if data["fixtureBacked"] != true {
		t.Fatalf("expected fixtureBacked true, got %#v", data["fixtureBacked"])
	}
	if data["authoritative"] != false {
		t.Fatalf("expected authoritative false, got %#v", data["authoritative"])
	}

	execution := asObject(t, data["execution"])
	for _, key := range []string{"dockerInvoked", "crossplaneCliInvoked", "networkAccess", "clusterAccess"} {
		if execution[key] != false {
			t.Fatalf("expected execution.%s false, got %#v", key, execution[key])
		}
	}
}

func TestErrorsAlsoReturnStructuredJSON(t *testing.T) {
	code, response := runAndDecode(t, "find-schema")
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if response.OK {
		t.Fatalf("expected ok false, got %#v", response)
	}
	if len(response.Errors) != 1 {
		t.Fatalf("expected one structured error, got %#v", response.Errors)
	}
}

func TestSuccessEnvelopeWireFormatUsesEmptyArrays(t *testing.T) {
	code, raw := runRawJSON(t, "list-compositions")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	fields := topLevelFields(t, raw)
	assertRawJSON(t, fields, "diagnostics", "[]")
	assertRawJSON(t, fields, "errors", "[]")
}

func TestErrorEnvelopeWireFormatUsesEmptyObjectAndArray(t *testing.T) {
	code, raw := runRawJSON(t, "find-schema")
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}

	fields := topLevelFields(t, raw)
	assertRawJSON(t, fields, "data", "{}")
	assertRawJSON(t, fields, "diagnostics", "[]")
}

func runAndDecode(t *testing.T, args ...string) (int, envelope) {
	t.Helper()

	code, raw := runRawJSON(t, args...)

	var response envelope
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return code, response
}

func runRawJSON(t *testing.T, args ...string) (int, []byte) {
	t.Helper()

	var out bytes.Buffer
	code := run(args, &out)
	raw := out.Bytes()
	if !json.Valid(raw) {
		t.Fatalf("expected valid JSON, got %q", out.String())
	}
	return code, raw
}

func topLevelFields(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal top-level fields: %v", err)
	}
	return fields
}

func assertRawJSON(t *testing.T, fields map[string]json.RawMessage, key, want string) {
	t.Helper()

	got, ok := fields[key]
	if !ok {
		t.Fatalf("expected top-level field %q", key)
	}
	if string(got) != want {
		t.Fatalf("expected %s to encode as %s, got %s", key, want, got)
	}
}

func asObject(t *testing.T, value any) map[string]any {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %#v", value)
	}
	return object
}

func asSlice(t *testing.T, value any) []any {
	t.Helper()

	slice, ok := value.([]any)
	if !ok {
		t.Fatalf("expected slice, got %#v", value)
	}
	return slice
}
