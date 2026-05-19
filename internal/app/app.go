package app

import (
	"fmt"
	"io"
	"runtime/debug"
)

const defaultVersion = "v0.0.1"

var version string

type ServerRunner func(stdin io.Reader, stdout io.Writer, stderr io.Writer) int
type DebugRunner func(args []string, stdout io.Writer) int

type Runners struct {
	Serve ServerRunner
	Debug DebugRunner
}

func Run(args []string, stdout io.Writer, stderr io.Writer, runners Runners) int {
	return RunWithIO(args, nil, stdout, stderr, runners)
}

func RunWithIO(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, runners Runners) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing command: use serve, debug, or --version")
		return 2
	}

	switch args[0] {
	case "--version", "version":
		fmt.Fprintf(stdout, "vibe-xpls %s\n", Version())
		return 0
	case "serve":
		if runners.Serve == nil {
			fmt.Fprintln(stderr, "serve command is unavailable")
			return 2
		}
		return runners.Serve(stdin, stdout, stderr)
	case "debug":
		if runners.Debug == nil {
			fmt.Fprintln(stderr, "debug command is unavailable")
			return 2
		}
		return runners.Debug(args[1:], stdout)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func Version() string {
	if version != "" {
		return version
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok || buildInfo.Main.Version == "" || buildInfo.Main.Version == "(devel)" {
		return defaultVersion
	}
	return buildInfo.Main.Version
}
