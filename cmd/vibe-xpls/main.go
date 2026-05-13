package main

import (
	"os"

	"github.com/io41/vibe-xpls/internal/app"
	"github.com/io41/vibe-xpls/internal/debugcli"
)

func main() {
	os.Exit(app.RunWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.Runners{Debug: debugcli.Run}))
}
