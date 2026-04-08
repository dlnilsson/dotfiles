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

// FrequencyFile represents a file and how many distinct revisions it has.
type FrequencyFile struct {
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
// merging entries that share the same name (different emails) and grouping
// variations like "jane", "janedoe", and "Jane Doe".
func parseAuthorsOutput(output string) []Author {
	// Phase 1: parse and merge exact names (after NFC normalization)
	counts := make(map[string]int)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

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

	// Phase 2: group similar names
	return groupAuthors(counts)
}

// normalizeAuthor strips spaces, hyphens, underscores, dots and lowercases
// for fuzzy author comparison.
func normalizeAuthor(name string) string {
	var sb strings.Builder
	sb.Grow(len(name))
	for _, r := range strings.ToLower(name) {
		switch r {
		case ' ', '-', '_', '.':
			continue
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// bestDisplayName picks the most human-readable name from a set of variants.
// Prefers names with spaces (real names) over usernames, then longest.
func bestDisplayName(names []string) string {
	best := names[0]
	for _, n := range names[1:] {
		nHasSpace := strings.Contains(n, " ")
		bestHasSpace := strings.Contains(best, " ")
		if nHasSpace && !bestHasSpace {
			best = n
		} else if nHasSpace == bestHasSpace && len(n) > len(best) {
			best = n
		}
	}
	return best
}

// groupAuthors merges author name variants into single entries.
// Groups names where their normalized forms (lowercase, no separators) match
// or one is a prefix of another (minimum 3 chars).
func groupAuthors(counts map[string]int) []Author {
	type entry struct {
		name  string
		norm  string
		count int
		group int
	}

	entries := make([]entry, 0, len(counts))
	for name, count := range counts {
		entries = append(entries, entry{
			name:  name,
			norm:  normalizeAuthor(name),
			count: count,
			group: -1,
		})
	}

	// Sort by normalized length descending so longer names come first
	slices.SortFunc(entries, func(a, b entry) int {
		return len(b.norm) - len(a.norm)
	})

	// Assign groups: each entry joins the first group whose normalized name
	// matches or is a prefix/suffix relationship.
	var groups [][]int // group index -> entry indices
	for i := range entries {
		matched := -1
		for gi, members := range groups {
			rep := entries[members[0]].norm
			if entriesMatch(entries[i].norm, rep) {
				matched = gi
				break
			}
		}
		if matched >= 0 {
			entries[i].group = matched
			groups[matched] = append(groups[matched], i)
		} else {
			entries[i].group = len(groups)
			groups = append(groups, []int{i})
		}
	}

	// Merge groups into final authors
	result := make([]Author, 0, len(groups))
	for _, members := range groups {
		var (
			totalCount int
			names      = make([]string, 0, len(members))
		)
		for _, idx := range members {
			totalCount += entries[idx].count
			names = append(names, entries[idx].name)
		}
		result = append(result, Author{
			Name:  bestDisplayName(names),
			Count: totalCount,
		})
	}

	slices.SortFunc(result, func(a, b Author) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Name, b.Name)
	})

	return result
}

// entriesMatch returns true if two normalized names should be grouped.
// Matches on equality, or prefix with minimum 3 chars.
func entriesMatch(a, b string) bool {
	if a == b {
		return true
	}
	short, long := a, b
	if len(short) > len(long) {
		short, long = long, short
	}
	return len(short) >= 3 && strings.HasPrefix(long, short)
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

// parseFrequency runs git rev-list to count distinct object revisions per file.
func parseFrequency() ([]FrequencyFile, error) {
	output, err := runGit("rev-list", "--objects", "--all", "--", ".")
	if err != nil {
		return nil, err
	}
	return parseFrequencyOutput(output), nil
}

// parseFrequencyOutput parses git rev-list --objects output, counting
// how many distinct blob revisions each file path has.
func parseFrequencyOutput(output string) []FrequencyFile {
	counts := make(map[string]int)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "<hash> <path>" or just "<hash>" (for commits/trees without paths)
		_, path, hasPath := strings.Cut(line, " ")
		if !hasPath || path == "" {
			continue
		}
		counts[path]++
	}

	result := make([]FrequencyFile, 0, len(counts))
	for file, count := range counts {
		result = append(result, FrequencyFile{File: file, Count: count})
	}

	slices.SortFunc(result, func(a, b FrequencyFile) int {
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
