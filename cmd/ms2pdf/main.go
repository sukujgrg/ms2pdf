package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sukujgrg/ms2pdf/internal/cli"
)

// Set by -ldflags "-X main.version=v0.1.0" on tagged builds.
var version = "dev"

func main() {
	app := cli.New()
	app.Version = version
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
