// Package unapproved lists recently merged merge requests that nobody but
// their author approved.
package unapproved

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

// DefaultCount is how many unapproved merge requests Run lists by default.
const DefaultCount = 50

// Options configures Run.
type Options struct {
	Count int  // unapproved merge requests to list
	CSV   bool // print the list as CSV
}

// Run prints the unapproved merge requests and a per-author count to out, and
// fetch progress to statusOut.
func Run(out, statusOut io.Writer, opts Options) error {
	f, err := forge.Detect()
	if err != nil {
		return err
	}
	project, err := f.Project()
	if err != nil {
		return err
	}

	mrs, _, err := report.CollectMerged(f, opts.Count,
		func(mr forge.MergeRequest) bool { return !mr.Approved },
		statusOut,
		func(found, scanned int) string {
			return fmt.Sprintf("Fetching merged MRs in %s (%d/%d unapproved, %d scanned)...", project, found, opts.Count, scanned)
		})
	if err != nil {
		return err
	}
	sortMRs(mrs)

	if opts.CSV {
		records := [][]string{{"MR", "URL", "Merged", "Author", "Title"}}
		for _, mr := range mrs {
			records = append(records, []string{strconv.Itoa(mr.Number), mr.URL, report.Date(mr.MergedAt), mr.Author.Username, mr.Title})
		}
		return report.WriteCSV(out, records)
	}

	rows := make([][]string, len(mrs))
	for i, mr := range mrs {
		ref := mr.Ref
		if report.IsTerminal(out) {
			// OSC 8 hyperlink, so the MR reference is clickable in the terminal.
			ref = "\033]8;;" + mr.URL + "\a" + ref + "\033]8;;\a"
		}
		rows[i] = []string{ref, report.Date(mr.MergedAt), mr.Author.Username, mr.Title}
	}
	fmt.Fprintln(out, report.Table([]string{"MR", "Merged", "Author", "Title"}, rows))

	perAuthor := rankAuthors(mrs)
	authorRows := make([][]string, len(perAuthor))
	for i, a := range perAuthor {
		authorRows[i] = []string{a.Username, strconv.Itoa(a.Count)}
	}
	fmt.Fprintln(out, report.Table([]string{"Author", "Unapproved"}, authorRows))
	return nil
}

// sortMRs sorts by author, then newest merge date first, then number.
func sortMRs(mrs []forge.MergeRequest) {
	slices.SortFunc(mrs, func(a, b forge.MergeRequest) int {
		return cmp.Or(
			strings.Compare(a.Author.Username, b.Author.Username),
			strings.Compare(report.Date(b.MergedAt), report.Date(a.MergedAt)),
			cmp.Compare(a.Number, b.Number),
		)
	})
}

// rankAuthors counts merge requests per author, most first, ties by name.
func rankAuthors(mrs []forge.MergeRequest) []report.UserCount {
	counts := make(map[string]int)
	for _, mr := range mrs {
		counts[mr.Author.Username]++
	}
	return report.RankCounts(counts)
}
