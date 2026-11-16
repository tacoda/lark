// Command lark is the lark CLI: REPL, run, -e, parse, emit.
package main

import (
	"os"

	"github.com/tacoda/lark/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
