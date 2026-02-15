package gist

import (
	"fmt"
	"os/exec"
	"strings"
)

// Publish creates a GitHub gist from the given file using the gh CLI.
// Returns the gist URL on success.
func Publish(filePath string, description string, public bool) (string, error) {
	return publish(filePath, description, public, execCommand)
}

// commandRunner abstracts exec.Command for testing.
type commandRunner func(name string, args ...string) ([]byte, error)

func execCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func publish(filePath string, description string, public bool, run commandRunner) (string, error) {
	args := []string{"gist", "create", filePath}

	if description != "" {
		args = append(args, "--desc", description)
	}

	if public {
		args = append(args, "--public")
	}

	output, err := run("gh", args...)
	if err != nil {
		return "", fmt.Errorf("gh gist create failed: %w\noutput: %s", err, string(output))
	}

	url := strings.TrimSpace(string(output))
	return url, nil
}
