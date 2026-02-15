package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cclog",
		Short:   "Export Claude Code session transcripts as readable documents",
		Long:    "cclog reads Claude Code JSONL session transcripts and exports them as HTML or Markdown, with secret scrubbing and smart boundary detection.",
		Version: version,
	}

	cmd.AddCommand(exportCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(setupCmd())
	cmd.AddCommand(serveCmd())

	return cmd
}
