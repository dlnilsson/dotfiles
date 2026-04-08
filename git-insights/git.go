package main

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// BugCluster represents a file and how many bug-related commits touched it.
type BugCluster struct {
	File  string
	Count int
}

// ActivityMonth represents commit count for a year-month bucket.
type ActivityMonth struct {
	Month string // "2025-03" format
	Count int
}

// FirefightEntry represents a single firefighting commit.
type FirefightEntry struct {
	Hash    string
	Subject string
}

// Author represents a contributor and their commit count.
type Author struct {
	Name  string
	Count int
}

// ChurnFile represents a file and how many times it changed.
type ChurnFile struct {
	File  string
	Count int
}


// runGit executes a git command and returns stdout as a string.
func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("git %s: %w\n%s", args[0], err, exitErr.Stderr)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}

// isGitRepo checks if the current directory is inside a git repository.
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// repoScope returns the current directory's path relative to the repo root.
// Returns empty string if at the repo root.
func repoScope() string {
	output, err := runGit("rev-parse", "--show-prefix")
	if err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(output), "/")
}

// parseBugClusters runs the git command and parses output.
func parseBugClusters() ([]BugCluster, error) {
	output, err := runGit("log", "-i", "-E", "--grep=fix|bug|broken", "--name-only", "--format=", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseBugClustersOutput(output), nil
}

// parseBugClustersOutput parses raw git output into BugCluster entries.
func parseBugClustersOutput(output string) []BugCluster {
	counts := make(map[string]int)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}

	result := make([]BugCluster, 0, len(counts))
	for file, count := range counts {
		result = append(result, BugCluster{File: file, Count: count})
	}

	slices.SortFunc(result, func(a, b BugCluster) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.File, b.File)
	})

	if len(result) > 20 {
		result = result[:20]
	}
	return result
}

// parseActivity runs the git command and parses output.
func parseActivity() ([]ActivityMonth, error) {
	output, err := runGit("log", "--format=%ad", "--date=format:%Y-%m", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseActivityOutput(output), nil
}

// parseActivityOutput parses raw git output into ActivityMonth entries.
func parseActivityOutput(output string) []ActivityMonth {
	counts := make(map[string]int)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}

	result := make([]ActivityMonth, 0, len(counts))
	for month, count := range counts {
		result = append(result, ActivityMonth{Month: month, Count: count})
	}

	slices.SortFunc(result, func(a, b ActivityMonth) int {
		return strings.Compare(a.Month, b.Month)
	})

	return result
}

// parseFirefighting runs the git command and parses output.
func parseFirefighting() ([]FirefightEntry, error) {
	output, err := runGit("log", "--oneline", "--since=1 year ago", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseFirefightingOutput(output), nil
}

// parseFirefightingOutput parses raw git output, filtering for firefighting commits.
func parseFirefightingOutput(output string) []FirefightEntry {
	var keywords = []string{"revert", "hotfix", "emergency", "rollback"}

	result := make([]FirefightEntry, 0, 16)

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		matched := false
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		var (
			hash    = parts[0]
			subject string
		)
		if len(parts) > 1 {
			subject = parts[1]
		}
		result = append(result, FirefightEntry{Hash: hash, Subject: subject})
	}

	return result
}

// parseAuthors runs the git command and parses output.
func parseAuthors() ([]Author, error) {
	output, err := runGit("shortlog", "-sn", "--no-merges", "HEAD", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseAuthorsOutput(output), nil
}

// parseAuthorsOutput parses raw git shortlog output into Author entries,
// merging entries that share the same name (different emails).
func parseAuthorsOutput(output string) []Author {
	counts := make(map[string]int)

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "  123\tAuthor Name"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}

		name := norm.NFC.String(strings.TrimSpace(parts[1]))
		counts[name] += count
	}

	result := make([]Author, 0, len(counts))
	for name, count := range counts {
		result = append(result, Author{Name: name, Count: count})
	}

	slices.SortFunc(result, func(a, b Author) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name, b.Name)
	})

	return result
}

// parseChurn runs the git command and parses output.
func parseChurn() ([]ChurnFile, error) {
	output, err := runGit("log", "--format=format:", "--name-only", "--since=1 year ago", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseChurnOutput(output), nil
}

// parseChurnOutput parses raw git output into ChurnFile entries.
func parseChurnOutput(output string) []ChurnFile {
	counts := make(map[string]int)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}

	result := make([]ChurnFile, 0, len(counts))
	for file, count := range counts {
		result = append(result, ChurnFile{File: file, Count: count})
	}

	slices.SortFunc(result, func(a, b ChurnFile) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.File, b.File)
	})

	if len(result) > 20 {
		result = result[:20]
	}
	return result
}
