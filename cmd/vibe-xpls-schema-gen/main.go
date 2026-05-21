package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/io41/vibe-xpls/internal/schemagen"
)

func main() {
	configPath := flag.String("config", "internal/analyzer/schemadata/config.json", "schema generator config")
	outDir := flag.String("out", "internal/analyzer/schemadata", "schema data output directory")
	flag.Parse()

	cfg, err := schemagen.LoadConfigFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if err := schemagen.Generate(cfg, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "generate schemas: %v\n", err)
		os.Exit(1)
	}
}
