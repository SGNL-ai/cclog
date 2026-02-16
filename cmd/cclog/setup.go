package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func setupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Register cclog as an MCP server in Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup()
		},
	}
}

func runSetup() error {
	cclogPath, err := exec.LookPath("cclog")
	if err != nil {
		cclogPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("could not find cclog binary: %w", err)
		}
	}

	cmd := exec.Command("claude", "mcp", "add",
		"--transport", "stdio",
		"--scope", "user",
		"cclog", "--",
		cclogPath, "serve",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register MCP server: %w", err)
	}

	fmt.Println("\ncclog MCP server registered successfully!")
	fmt.Println("You can now ask Claude to export session transcripts.")
	return nil
}
