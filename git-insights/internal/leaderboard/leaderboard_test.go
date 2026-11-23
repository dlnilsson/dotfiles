package leaderboard

import (
	"slices"
	"testing"
	"time"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
	"github.com/dlnilsson/dotfiles/git-insights/internal/report"
)

func TestRankApprovers(t *testing.T) {
	t.Parallel()

	mrs := []forge.MergeRequest{
		{Approvers: []string{"bob", "ann"}},
		{Approvers: []string{"cid"}},
		{Approvers: []string{"bob"}},
		{},
	}
	got := rankApprovers(mrs, false)
	want := []report.UserCount{{Username: "bob", Count: 2}, {Username: "ann", Count: 1}, {Username: "cid", Count: 1}}
	if !slices.Equal(got, want) {
		t.Errorf("rankApprovers = %v, want %v", got, want)
	}

	mrs[0].Author.Username = "bob"
	got = rankApprovers(mrs, true)
	want = []report.UserCount{{Username: "ann", Count: 1}, {Username: "bob", Count: 1}, {Username: "cid", Count: 1}}
	if !slices.Equal(got, want) {
		t.Errorf("rankApprovers skipping self = %v, want %v", got, want)
	}
}

func TestReviewTimeline(t *testing.T) {
	t.Parallel()

	day := func(d int) time.Time { return time.Date(2026, 11, d, 23, 0, 0, 0, time.UTC) }
	mrs := []forge.MergeRequest{
		{Number: 10, MergedAt: day(3), Author: forge.Person{Username: "ann"}, Approvers: []string{"cid", "ann", "bob"}},
		{Number: 9, MergedAt: day(3), Author: forge.Person{Username: "bob"}, Approvers: []string{"ann"}},
		{Number: 12, MergedAt: day(1), Author: forge.Person{Username: "cid"}, Approvers: []string{"bob"}},
	}
	got := reviewTimeline(mrs)
	want := [][]string{
		{"2026-11-01", "12", "cid", "bob"},
		{"2026-11-03", "9", "bob", "ann"},
		{"2026-11-03", "10", "ann", "bob"},
		{"2026-11-03", "10", "ann", "cid"},
	}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("reviewTimeline = %v, want %v", got, want)
	}
}
