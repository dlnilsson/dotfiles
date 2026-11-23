package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const approvalsUsage = "usage: git-insights approvals [-C dir] [--csv] [count]"

// runApprovals prints a leaderboard of who approved the most recently merged
// merge requests, skipping those labeled Release.
func runApprovals(args []string) error {
	var (
		count  = 200
		useCSV bool
	)
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "-C":
			i++
			if i >= len(args) {
				return errors.New(approvalsUsage)
			}
			if err := os.Chdir(args[i]); err != nil {
				return err
			}
		case "--csv":
			useCSV = true
		default:
			n, err := strconv.Atoi(arg)
			if err != nil || n < 1 {
				return errors.New(approvalsUsage)
			}
			count = n
		}
	}

	if !isGitRepo() {
		return errors.New("not a git repository")
	}
	forge, err := detectForge()
	if err != nil {
		return err
	}
	project, err := forge.Project()
	if err != nil {
		return err
	}

	progress := func(int) {}
	if isTerminal(os.Stderr) {
		progress = func(scanned int) {
			fmt.Fprintf(os.Stderr, "\r\033[KFetching merged MRs in %s (%d/%d scanned)...", project, scanned, count)
		}
	}
	mrs, err := forge.MergedMergeRequests(count, "Release", progress)
	if isTerminal(os.Stderr) {
		fmt.Fprint(os.Stderr, "\r\033[K")
	}
	if err != nil {
		return err
	}

	ranks := rankApprovers(mrs)

	if useCSV {
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"Rank", "Approver", "Approvals"})
		for i, r := range ranks {
			_ = w.Write([]string{strconv.Itoa(i + 1), r.Username, strconv.Itoa(r.Count)})
		}
		w.Flush()
		return w.Error()
	}

	rows := make([][]string, len(ranks))
	for i, r := range ranks {
		rows[i] = []string{strconv.Itoa(i + 1), r.Username, strconv.Itoa(r.Count)}
	}
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		Headers("Rank", "Approver", "Approvals").
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true)
			}
			return style
		})
	fmt.Println(t)
	fmt.Printf("Based on the last %d merged MRs in %s.\n", len(mrs), project)
	return nil
}

type approverCount struct {
	Username string
	Count    int
}

// rankApprovers counts approvals per username, most first, ties by name.
func rankApprovers(mrs []MergeRequest) []approverCount {
	counts := make(map[string]int)
	for _, mr := range mrs {
		for _, a := range mr.Approvers {
			counts[a]++
		}
	}
	result := make([]approverCount, 0, len(counts))
	for name, n := range counts {
		result = append(result, approverCount{Username: name, Count: n})
	}
	slices.SortFunc(result, func(a, b approverCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		return strings.Compare(a.Username, b.Username)
	})
	return result
}
