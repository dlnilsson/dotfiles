// Package leaderboard ranks who approved the most recently merged merge
// requests.
package leaderboard

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
	"github.com/dlnilsson/dotfiles/git-insights/internal/report"
)

// DefaultCount is how many merged merge requests Run scans by default.
const DefaultCount = 200

// Options configures Run.
type Options struct {
	Count    int  // merged merge requests to scan
	Reviews  bool // leave out self-approvals and label columns as reviews
	CSV      bool // print the leaderboard as CSV
	Timeline bool // print one CSV row per review instead of the leaderboard
}

// Run prints the leaderboard to out and fetch progress to statusOut.
func Run(out, statusOut io.Writer, opts Options) error {
	f, err := forge.Detect()
	if err != nil {
		return err
	}
	project, err := f.Project()
	if err != nil {
		return err
	}

	mrs, _, err := report.CollectMerged(f, opts.Count, nil, statusOut, func(_, scanned int) string {
		return fmt.Sprintf("Fetching merged MRs in %s (%d/%d scanned)...", project, scanned, opts.Count)
	})
	if err != nil {
		return err
	}

	if opts.Timeline {
		return report.WriteCSV(out, append([][]string{{"Merged", "MR", "Author", "Reviewer"}}, reviewTimeline(mrs)...))
	}

	name, total := "Approver", "Approvals"
	if opts.Reviews {
		name, total = "Reviewer", "Reviews"
	}
	ranks := rankApprovers(mrs, opts.Reviews)
	rows := make([][]string, len(ranks))
	for i, r := range ranks {
		rows[i] = []string{strconv.Itoa(i + 1), r.Username, strconv.Itoa(r.Count)}
	}

	if opts.CSV {
		return report.WriteCSV(out, append([][]string{{"Rank", name, total}}, rows...))
	}

	fmt.Fprintln(out, report.Table([]string{"Rank", name, total}, rows))
	fmt.Fprintf(out, "Based on the last %d merged MRs in %s.\n", len(mrs), project)
	return nil
}

// rankApprovers counts approvals per username, most first, ties by name.
// With skipSelf, an author approving their own merge request is not counted.
func rankApprovers(mrs []forge.MergeRequest, skipSelf bool) []report.UserCount {
	counts := make(map[string]int)
	for _, mr := range mrs {
		for _, a := range mr.Approvers {
			if skipSelf && a == mr.Author.Username {
				continue
			}
			counts[a]++
		}
	}
	return report.RankCounts(counts)
}

// reviewTimeline returns one row per non-self approval:
// merge date (UTC), merge request number, author, reviewer; oldest first.
func reviewTimeline(mrs []forge.MergeRequest) [][]string {
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
				date:     report.Date(mr.MergedAt),
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
