package main

import (
	"os"

	"github.com/github/vulcanizer/pkg/cli"
)

func main() {
	cli.InitializeCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
}
