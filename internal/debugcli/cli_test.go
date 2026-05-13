package debugcli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDiagnosticsCommandIsInternalJSON(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"diagnostics", "--workspace", "../analyzer/testdata/workspaces/root", "--uri", "file:///composition.yaml", "--text", "apiVersion: apiextensions.crossplane.io/v1\nkind: Composition\n"}, &out)

	if code != 0 {
		t.Fatalf("exit code = %d, output=%s", code, out.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("decode json: %v; output=%s", err, out.String())
	}
	if envelope["contract"] != "internal-debug" {
		t.Fatalf("contract = %#v, want internal-debug", envelope["contract"])
	}
}

func TestUnknownCommandFails(t *testing.T) {
	var out bytes.Buffer
	code := Run([]string{"render"}, &out)

	if code == 0 {
		t.Fatal("unknown debug command should fail")
	}
	if !strings.Contains(out.String(), "unknown debug command") {
		t.Fatalf("output = %q", out.String())
	}
}
