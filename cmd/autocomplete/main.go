package main

import (
	"context"
	"os"

	"github.com/kmal808/autocomplete-sh/internal/cli"
)

func main() {
	runner := cli.Runner{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Env:    os.Getenv,
	}
	os.Exit(runner.Run(context.Background(), os.Args[1:]))
}
