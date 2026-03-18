// Package main is the entry point for the Unraid CLI.
package main

import (
	"os"

	"github.com/medzin/unraid-cli/internal/commands"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
