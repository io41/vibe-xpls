package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemaGenCLIExplicitGenerateAndCompatibilityAlias(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "explicit generate", args: []string{"generate"}},
		{name: "legacy alias", args: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := t.TempDir()
			args := append([]string{"run", "."}, tc.args...)
			args = append(args, "--config", fixtureConfigPath(), "--out", out)
			cmd := exec.Command("go", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, output)
			}
			if _, err := os.Stat(filepath.Join(out, "manifest.json")); err != nil {
				t.Fatalf("stat generated manifest: %v", err)
			}
			if _, err := os.Stat(filepath.Join(out, "schemas", "v1.20.7", "apiextensions.crossplane.io_v1_Composition.json")); err != nil {
				t.Fatalf("stat generated fixture schema: %v", err)
			}
		})
	}
}

func TestSchemaGenCLIRejectsUnknownCommand(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "unknown")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("go run . unknown succeeded")
	}
	if !strings.Contains(string(output), "usage: vibe-xpls-schema-gen") {
		t.Fatalf("unknown command output missing usage:\n%s", output)
	}
}

func TestSchemaGenCLIDriftRequiresTokenWhenRequested(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "drift", "check", "--config", fixtureConfigPath(), "--require-token")
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN=")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("go run . drift check --require-token succeeded without GITHUB_TOKEN")
	}
	if !strings.Contains(string(output), "GITHUB_TOKEN is required") {
		t.Fatalf("drift check output missing token requirement:\n%s", output)
	}
}

func fixtureConfigPath() string {
	return filepath.Join("..", "..", "internal", "schemagen", "testdata", "config.json")
}
