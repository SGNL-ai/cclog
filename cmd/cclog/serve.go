package main

import (
	"context"

	mcpserver "github.com/sgnl-ai/cclog/internal/mcp"
	"github.com/spf13/cobra"
)

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run as MCP server (stdio transport)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe()
		},
	}
}

func runServe() error {
	return mcpserver.Run(context.Background())
}
