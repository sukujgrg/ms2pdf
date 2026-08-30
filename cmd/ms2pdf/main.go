package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sukujgrg/ms2pdf/internal/cli"
)

func main() {
	if err := cli.New().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
