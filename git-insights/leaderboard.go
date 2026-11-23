package main

import (
	"cmp"
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

const leaderboardHelp = `usage: git-insights approvals [-C dir] [--csv] [count]
       git-insights reviews [-C dir] [--csv | --timeline] [count]

Leaderboard of who approved the most recently merged MRs (default: last 200),
skipping MRs labeled Release. reviews does not count self-approval by the
author; approvals does.

  -C dir      run in dir instead of the current directory
  --csv       print the leaderboard as CSV
  --timeline  (reviews only) print one CSV row per review
              (Merged,MR,Author,Reviewer), oldest first
  -h, --help  print this help`

func runApprovals(args []string) error { return runLeaderboard(args, false) }
func runReviews(args []string) error   { return runLeaderboard(args, true) }

// runLeaderboard ranks approvers of recently merged merge requests. With
// reviews, self-approvals are left out and --timeline is allowed.
func runLeaderboard(args []string, reviews bool) error {
	var (
		count       = 200
		useCSV      bool
		useTimeline bool
	)
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; arg {
		case "-h", "--help":
			fmt.Println(leaderboardHelp)
			return nil
		case "-C":
			i++
			if i >= len(args) {
				return errors.New(leaderboardHelp)
			}
			if err := os.Chdir(args[i]); err != nil {
				return err
			}
		case "--csv":
			useCSV = true
		case "--timeline":
			if !reviews {
				return errors.New(leaderboardHelp)
			}
			useTimeline = true
		default:
			n, err := strconv.Atoi(arg)
			if err != nil || n < 1 {
				return errors.New(leaderboardHelp)
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

	mrs, _, err := collectMerged(forge, count, nil, func(_, scanned int) string {
		return fmt.Sprintf("Fetching merged MRs in %s (%d/%d scanned)...", project, scanned, count)
	})
	if err != nil {
		return err
	}

	if useTimeline {
		return writeCSV(append([][]string{{"Merged", "MR", "Author", "Reviewer"}}, reviewTimeline(mrs)...))
	}

	name, total := "Approver", "Approvals"
	if reviews {
		name, total = "Reviewer", "Reviews"
	}
	ranks := rankApprovers(mrs, reviews)
	rows := make([][]string, len(ranks))
	for i, r := range ranks {
		rows[i] = []string{strconv.Itoa(i + 1), r.Username, strconv.Itoa(r.Count)}
	}

	if useCSV {
		return writeCSV(append([][]string{{"Rank", name, total}}, rows...))
	}

	fmt.Println(renderTable([]string{"Rank", name, total}, rows))
	fmt.Printf("Based on the last %d merged MRs in %s.\n", len(mrs), project)
	return nil
}

// collectMerged returns the first count merged merge requests, skipping those
// labeled Release, for which keep is true (all when nil), and how many it
// scanned. On a terminal it shows status(found, scanned) on stderr meanwhile.
func collectMerged(forge Forge, count int, keep func(MergeRequest) bool, status func(found, scanned int) string) ([]MergeRequest, int, error) {
	show := func(string) {}
	if isTerminal(os.Stderr) {
		show = func(s string) { fmt.Fprint(os.Stderr, "\r\033[K"+s) }
		defer show("")
	}

	var (
		result  []MergeRequest
		scanned int
	)
	show(status(0, 0))
	for mr, err := range forge.MergedMergeRequests("Release") {
		if err != nil {
			return nil, scanned, err
		}
		scanned++
		if keep == nil || keep(mr) {
			result = append(result, mr)
		}
		show(status(len(result), scanned))
		if len(result) == count {
			break
		}
	}
	return result, scanned, nil
}

func renderTable(headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			style := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return style.Bold(true)
			}
			return style
		}).
		String()
}

func writeCSV(records [][]string) error {
	w := csv.NewWriter(os.Stdout)
	if err := w.WriteAll(records); err != nil {
		return err
	}
	return w.Error()
}

type userCount struct {
	Username string
	Count    int
}

// rankApprovers counts approvals per username, most first, ties by name.
// With skipSelf, an author approving their own merge request is not counted.
func rankApprovers(mrs []MergeRequest, skipSelf bool) []userCount {
	counts := make(map[string]int)
	for _, mr := range mrs {
		for _, a := range mr.Approvers {
			if skipSelf && a == mr.Author.Username {
				continue
			}
			counts[a]++
		}
	}
	return rankCounts(counts)
}

// rankCounts sorts counts per username, most first, ties by name.
func rankCounts(counts map[string]int) []userCount {
	result := make([]userCount, 0, len(counts))
	for name, n := range counts {
		result = append(result, userCount{Username: name, Count: n})
	}
	slices.SortFunc(result, func(a, b userCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Username, b.Username))
	})
	return result
}

// reviewTimeline returns one row per non-self approval:
// merge date (UTC), merge request number, author, reviewer; oldest first.
func reviewTimeline(mrs []MergeRequest) [][]string {
	type review struct {
		date     string
		number   int
		author   string
		reviewer string
	}
	var reviews []review
	for _, mr := range mrs {
		for _, a := range mr.Approvers {
			if a == mr.Author.Username {
				continue
			}
			reviews = append(reviews, review{
				date:     mr.MergedAt.UTC().Format("2006-01-02"),
				number:   mr.Number,
				author:   mr.Author.Username,
				reviewer: a,
			})
		}
	}
	slices.SortFunc(reviews, func(a, b review) int {
		return cmp.Or(
			strings.Compare(a.date, b.date),
			cmp.Compare(a.number, b.number),
			strings.Compare(a.reviewer, b.reviewer),
		)
	})
	rows := make([][]string, len(reviews))
	for i, r := range reviews {
		rows[i] = []string{r.date, strconv.Itoa(r.number), r.author, r.reviewer}
	}
	return rows
}
