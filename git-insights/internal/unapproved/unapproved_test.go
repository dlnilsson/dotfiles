package unapproved

import (
	"slices"
	"testing"
	"time"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
	"github.com/dlnilsson/dotfiles/git-insights/internal/report"
)

func TestSortMRs(t *testing.T) {
	t.Parallel()

	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	mrs := []forge.MergeRequest{
		{Number: 5, MergedAt: day(1), Author: forge.Person{Username: "bob"}},
		{Number: 9, MergedAt: day(2), Author: forge.Person{Username: "bob"}},
		{Number: 7, MergedAt: day(2), Author: forge.Person{Username: "bob"}},
		{Number: 1, MergedAt: day(1), Author: forge.Person{Username: "ann"}},
	}
	sortMRs(mrs)
	var got []int
	for _, mr := range mrs {
		got = append(got, mr.Number)
	}
	if want := []int{1, 7, 9, 5}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}

	ranks := rankAuthors(mrs)
	if want := []report.UserCount{{Username: "bob", Count: 3}, {Username: "ann", Count: 1}}; !slices.Equal(ranks, want) {
		t.Errorf("rankAuthors = %v, want %v", ranks, want)
	}
}
