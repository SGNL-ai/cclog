package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionInfo describes a Claude Code session found in sessions-index.json.
type SessionInfo struct {
	ID           string
	Slug         string
	Project      string
	Created      time.Time
	Modified     time.Time
	FirstPrompt  string
	Summary      string
	MessageCount int
	FilePath     string
}

// sessionsIndex is the JSON structure of sessions-index.json.
type sessionsIndex struct {
	Version      int          `json:"version"`
	Entries      []indexEntry `json:"entries"`
	OriginalPath string       `json:"originalPath"`
}

type indexEntry struct {
	SessionID    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FirstPrompt  string `json:"firstPrompt"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	ProjectPath  string `json:"projectPath"`
	GitBranch    string `json:"gitBranch"`
}

// DefaultClaudeDir returns the default Claude projects directory.
func DefaultClaudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// Discover finds all sessions across all project directories.
func Discover(claudeDir string) ([]SessionInfo, error) {
	var sessions []SessionInfo

	// Find all sessions-index.json files
	entries, err := os.ReadDir(claudeDir)
	if err != nil {
		return nil, fmt.Errorf("read claude dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		indexPath := filepath.Join(claudeDir, entry.Name(), "sessions-index.json")
		found, err := discoverFromIndex(indexPath)
		if err != nil {
			continue // Skip directories without valid index
		}
		sessions = append(sessions, found...)
	}

	// Also discover sessions from JSONL files without index
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(claudeDir, entry.Name())
		found, err := discoverFromJSONL(dirPath, sessions)
		if err != nil {
			continue
		}
		sessions = append(sessions, found...)
	}

	// Sort by modified time, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})

	return sessions, nil
}

// DiscoverForProject finds sessions for a specific project path.
func DiscoverForProject(claudeDir, projectPath string) ([]SessionInfo, error) {
	all, err := Discover(claudeDir)
	if err != nil {
		return nil, err
	}

	var filtered []SessionInfo
	for _, s := range all {
		if s.Project == projectPath {
			filtered = append(filtered, s)
		}
	}
	return filtered, nil
}

// FindSession finds a specific session by ID (or unique prefix) across all projects.
func FindSession(claudeDir, sessionID string) (*SessionInfo, error) {
	all, err := Discover(claudeDir)
	if err != nil {
		return nil, err
	}

	// Exact match first
	for _, s := range all {
		if s.ID == sessionID {
			return &s, nil
		}
	}

	// Prefix match
	var matches []SessionInfo
	for _, s := range all {
		if strings.HasPrefix(s.ID, sessionID) {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 1:
		return &matches[0], nil
	case 0:
		return nil, fmt.Errorf("session %q not found", sessionID)
	default:
		return nil, fmt.Errorf("session prefix %q is ambiguous (%d matches)", sessionID, len(matches))
	}
}

func discoverFromIndex(indexPath string) ([]SessionInfo, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	var idx sessionsIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}

	var sessions []SessionInfo
	for _, e := range idx.Entries {
		created, _ := time.Parse(time.RFC3339Nano, e.Created)
		modified, _ := time.Parse(time.RFC3339Nano, e.Modified)

		s := SessionInfo{
			ID:           e.SessionID,
			Project:      e.ProjectPath,
			Created:      created,
			Modified:     modified,
			FirstPrompt:  e.FirstPrompt,
			Summary:      e.Summary,
			MessageCount: e.MessageCount,
			FilePath:     e.FullPath,
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

func discoverFromJSONL(dirPath string, existing []SessionInfo) ([]SessionInfo, error) {
	// Build set of already-known session IDs
	known := make(map[string]bool)
	for _, s := range existing {
		known[s.ID] = true
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		if known[sessionID] {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Extract project name from directory name
		project := decodeProjectPath(filepath.Base(dirPath))

		// Read first few lines to get slug and first prompt
		slug, firstPrompt := peekSession(fullPath)

		sessions = append(sessions, SessionInfo{
			ID:          sessionID,
			Slug:        slug,
			Project:     project,
			Modified:    info.ModTime(),
			FirstPrompt: firstPrompt,
			FilePath:    fullPath,
		})
	}

	return sessions, nil
}

// decodeProjectPath converts Claude's encoded directory name back to a path.
// e.g., "-Users-erikgustavson-projects-cclog" → "/Users/erikgustavson/projects/cclog"
func decodeProjectPath(encoded string) string {
	if encoded == "" {
		return ""
	}
	return "/" + strings.ReplaceAll(encoded[1:], "-", "/")
}

// peekSession reads the first user message from a JSONL file to get slug and first prompt.
func peekSession(path string) (slug, firstPrompt string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close() //nolint:errcheck

	// Read first few KB to find user message
	buf := make([]byte, 32*1024)
	n, _ := f.Read(buf)
	if n == 0 {
		return "", ""
	}

	// Split into lines and find first user message
	lines := strings.Split(string(buf[:n]), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry struct {
			Type    string `json:"type"`
			Slug    string `json:"slug"`
			Message struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Slug != "" && slug == "" {
			slug = entry.Slug
		}

		if entry.Type == "user" && entry.Message.Role == "user" {
			// Content can be string or array
			var text string
			if json.Unmarshal(entry.Message.Content, &text) == nil {
				firstPrompt = truncatePrompt(text, 100)
				return
			}
			// Try array
			var blocks []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(entry.Message.Content, &blocks) == nil {
				for _, b := range blocks {
					if b.Type == "text" && b.Text != "" {
						firstPrompt = truncatePrompt(b.Text, 100)
						return
					}
				}
			}
		}
	}

	return slug, firstPrompt
}

func truncatePrompt(s string, max int) string {
	// Take first line only
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
