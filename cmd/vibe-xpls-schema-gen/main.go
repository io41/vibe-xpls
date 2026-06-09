package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/io41/vibe-xpls/internal/schemagen"
)

const (
	defaultConfigPath = "internal/analyzer/schemadata/config.json"
	defaultOutDir     = "internal/analyzer/schemadata"
)

func main() {
	if code := run(os.Args[1:], os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func run(args []string, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runGenerate(args, stderr)
	}

	switch args[0] {
	case "generate":
		return runGenerate(args[1:], stderr)
	case "coverage":
		return runCoverage(args[1:], stderr)
	case "drift":
		return runDrift(args[1:], stderr)
	default:
		printUsage(stderr)
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func runGenerate(args []string, stderr io.Writer) int {
	cfg, outDir, code, ok := parseConfigOutFlags("generate", args, stderr)
	if !ok {
		return code
	}
	if err := schemagen.Generate(cfg, outDir); err != nil {
		fmt.Fprintf(stderr, "generate schemas: %v\n", err)
		return 1
	}
	return 0
}

func runCoverage(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "generate":
		cfg, outDir, code, ok := parseConfigOutFlags("coverage generate", args[1:], stderr)
		if !ok {
			return code
		}
		if err := schemagen.GenerateCoverage(cfg, outDir); err != nil {
			fmt.Fprintf(stderr, "generate coverage: %v\n", err)
			return 1
		}
		return 0
	case "check":
		cfg, outDir, code, ok := parseConfigOutFlags("coverage check", args[1:], stderr)
		if !ok {
			return code
		}
		if err := schemagen.CheckCoverage(cfg, outDir); err != nil {
			fmt.Fprintf(stderr, "check coverage: %v\n", err)
			return 1
		}
		return 0
	default:
		printUsage(stderr)
		fmt.Fprintf(stderr, "unknown coverage command %q\n", args[0])
		return 2
	}
}

func runDrift(args []string, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	if args[0] != "check" {
		printUsage(stderr)
		fmt.Fprintf(stderr, "unknown drift command %q\n", args[0])
		return 2
	}

	fs := newFlagSet("drift check", stderr)
	configPath := fs.String("config", defaultConfigPath, "schema generator config")
	requireToken := fs.Bool("require-token", false, "require GITHUB_TOKEN for drift checks")
	if code, ok := parseFlags(fs, args[1:], stderr); !ok {
		return code
	}
	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return 1
	}
	if err := schemagen.CheckDrift(cfg, schemagen.DriftOptions{
		Token:        os.Getenv("GITHUB_TOKEN"),
		RequireToken: *requireToken,
	}); err != nil {
		fmt.Fprintf(stderr, "check drift: %v\n", err)
		return 1
	}
	return 0
}

func parseConfigOutFlags(name string, args []string, stderr io.Writer) (schemagen.Config, string, int, bool) {
	fs := newFlagSet(name, stderr)
	configPath := fs.String("config", defaultConfigPath, "schema generator config")
	outDir := fs.String("out", defaultOutDir, "schema data output directory")
	if code, ok := parseFlags(fs, args, stderr); !ok {
		return schemagen.Config{}, "", code, false
	}
	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "load config: %v\n", err)
		return schemagen.Config{}, "", 1, false
	}
	return cfg, *outDir, 0, true
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		printUsage(stderr)
	}
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string, stderr io.Writer) (int, bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, false
		}
		return 2, false
	}
	if fs.NArg() != 0 {
		printUsage(stderr)
		fmt.Fprintf(stderr, "unexpected argument %q\n", fs.Arg(0))
		return 2, false
	}
	return 0, true
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `usage: vibe-xpls-schema-gen <command> [options]

commands:
  generate
  coverage generate
  coverage check
  drift check
`)
}
