// Command lark is the lark CLI. The REPL and subcommands land in phase 5;
// for now it only confirms the toolchain builds.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "lark: CLI lands in phase 5 (see docs/PLAN.md); run `go test ./...` for now")
	os.Exit(2)
}
