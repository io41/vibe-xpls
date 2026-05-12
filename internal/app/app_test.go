package app

import (
	"bytes"
	"io"
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

func TestRunWithIOServePassesStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdin := strings.NewReader("lsp request")

	code := RunWithIO([]string{"serve"}, stdin, &stdout, &stderr, Runners{
		Serve: func(gotStdin io.Reader, gotStdout io.Writer, gotStderr io.Writer) int {
			body, err := io.ReadAll(gotStdin)
			if err != nil {
				t.Fatalf("read stdin: %v", err)
			}
			if string(body) != "lsp request" {
				t.Fatalf("stdin = %q", string(body))
			}
			if gotStdout != &stdout {
				t.Fatalf("stdout writer was not forwarded")
			}
			if gotStderr != &stderr {
				t.Fatalf("stderr writer was not forwarded")
			}
			return 7
		},
	})

	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
}

func TestRunUnavailableRunners(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "serve", args: []string{"serve"}, want: "serve command is unavailable"},
		{name: "debug", args: []string{"debug"}, want: "debug command is unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr, Runners{})

			if code != 2 {
				t.Fatalf("exit code = %d, want 2", code)
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr should contain %q, got %q", tt.want, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout should be empty, got %q", stdout.String())
			}
		})
	}
}
