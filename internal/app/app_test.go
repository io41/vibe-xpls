package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr, Runners{})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "vibe-xpls v0.0.1" {
		t.Fatalf("version output = %q", got)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"render"}, &stdout, &stderr, Runners{})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr should explain unknown command, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should be empty, got %q", stdout.String())
	}
}
