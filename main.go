// Command ask is a terminal-first LLM assistant with provider and model management.
package main

import (
	"fmt"
	"os"

	"ask/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
