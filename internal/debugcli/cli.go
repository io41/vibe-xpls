package debugcli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/io41/vibe-xpls/internal/analyzer"
)

const contract = "internal-debug"

type Envelope struct {
	OK       bool   `json:"ok"`
	Contract string `json:"contract"`
	Command  string `json:"command,omitempty"`
	Data     any    `json:"data,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Run(args []string, out io.Writer) int {
	if len(args) == 0 {
		return write(out, Envelope{
			OK:       false,
			Contract: contract,
			Error:    "missing debug command",
		})
	}

	switch args[0] {
	case "diagnostics":
		return diagnostics(args[1:], out)
	default:
		return write(out, Envelope{
			OK:       false,
			Contract: contract,
			Command:  args[0],
			Error:    fmt.Sprintf("unknown debug command %q", args[0]),
		})
	}
}

func diagnostics(args []string, out io.Writer) int {
	flags := flag.NewFlagSet("diagnostics", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	workspace := flags.String("workspace", ".", "workspace root")
	uri := flags.String("uri", "file:///debug.yaml", "document URI")
	text := flags.String("text", "", "document text")
	if err := flags.Parse(args); err != nil {
		return write(out, Envelope{
			OK:       false,
			Contract: contract,
			Command:  "diagnostics",
			Error:    err.Error(),
		})
	}

	a, err := analyzer.New(analyzer.Options{
		WorkspaceRoot: *workspace,
		Limits:        analyzer.DefaultLimits(),
	})
	if err != nil {
		return write(out, Envelope{
			OK:       false,
			Contract: contract,
			Command:  "diagnostics",
			Error:    err.Error(),
		})
	}
	a.OpenDocument(*uri, *text)
	return write(out, Envelope{
		OK:       true,
		Contract: contract,
		Command:  "diagnostics",
		Data:     a.Diagnostics(*uri),
	})
}

func write(out io.Writer, envelope Envelope) int {
	if out == nil {
		out = io.Discard
	}
	if err := json.NewEncoder(out).Encode(envelope); err != nil {
		return 2
	}
	if envelope.OK {
		return 0
	}
	return 2
}
