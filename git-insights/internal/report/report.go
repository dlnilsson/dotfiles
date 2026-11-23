// Package report holds output and fetch helpers shared by the merge request
// reporting commands.
package report

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
)

// SkipLabel marks merge requests the reports leave out.
const SkipLabel = "Release"

// IsTerminal reports whether w is a terminal.
func IsTerminal(w any) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// CollectMerged returns the first count merged merge requests, skipping those
// labeled SkipLabel, for which keep is true (all when nil), and how many it
// scanned. When statusOut is a terminal it shows status(found, scanned) there.
func CollectMerged(f forge.Forge, count int, keep func(forge.MergeRequest) bool, statusOut io.Writer, status func(found, scanned int) string) ([]forge.MergeRequest, int, error) {
	show := func(string) {}
	if IsTerminal(statusOut) {
		show = func(s string) { fmt.Fprint(statusOut, "\r\033[K"+s) }
		defer show("")
	}

	var (
		result  []forge.MergeRequest
		scanned int
	)
	show(status(0, 0))
	for mr, err := range f.MergedMergeRequests(SkipLabel) {
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

// Table renders rows as a bordered table with bold headers.
func Table(headers []string, rows [][]string) string {
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

// WriteCSV writes records as CSV.
func WriteCSV(w io.Writer, records [][]string) error {
	cw := csv.NewWriter(w)
	if err := cw.WriteAll(records); err != nil {
		return err
	}
	return cw.Error()
}

// Date formats t as a UTC date.
func Date(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// UserCount is a count for one username.
type UserCount struct {
	Username string
	Count    int
}

// RankCounts sorts counts per username, most first, ties by name.
func RankCounts(counts map[string]int) []UserCount {
	result := make([]UserCount, 0, len(counts))
	for name, n := range counts {
		result = append(result, UserCount{Username: name, Count: n})
	}
	slices.SortFunc(result, func(a, b UserCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Username, b.Username))
	})
	return result
}
