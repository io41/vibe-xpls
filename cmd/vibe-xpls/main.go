package main

import (
	"io"
	"os"

	"github.com/io41/vibe-xpls/internal/app"
	"github.com/io41/vibe-xpls/internal/debugcli"
	"github.com/io41/vibe-xpls/internal/lsp"
)

func main() {
	os.Exit(app.RunWithIO(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.Runners{
		Debug: debugcli.Run,
		Serve: func(stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
			if stdin == nil {
				stdin = os.Stdin
			}
			return lsp.NewServer(stdin, stdout, stderr).Run()
		},
	}))
}
