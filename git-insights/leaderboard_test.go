package main

import (
	"slices"
	"testing"
	"time"
)

func TestRankApprovers(t *testing.T) {
	t.Parallel()

	mrs := []MergeRequest{
		{Approvers: []string{"bob", "ann"}},
		{Approvers: []string{"cid"}},
		{Approvers: []string{"bob"}},
		{},
	}
	got := rankApprovers(mrs, false)
	want := []approverCount{{"bob", 2}, {"ann", 1}, {"cid", 1}}
	if !slices.Equal(got, want) {
		t.Errorf("rankApprovers = %v, want %v", got, want)
	}

	mrs[0].Author.Username = "bob"
	got = rankApprovers(mrs, true)
	want = []approverCount{{"ann", 1}, {"bob", 1}, {"cid", 1}}
	if !slices.Equal(got, want) {
		t.Errorf("rankApprovers skipping self = %v, want %v", got, want)
	}
}

func TestReviewTimeline(t *testing.T) {
	t.Parallel()

	day := func(d int) time.Time { return time.Date(2026, 11, d, 23, 0, 0, 0, time.UTC) }
	mrs := []MergeRequest{
		{Number: 10, MergedAt: day(3), Author: Person{Username: "ann"}, Approvers: []string{"cid", "ann", "bob"}},
		{Number: 9, MergedAt: day(3), Author: Person{Username: "bob"}, Approvers: []string{"ann"}},
		{Number: 12, MergedAt: day(1), Author: Person{Username: "cid"}, Approvers: []string{"bob"}},
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

func TestGitLabMRApprovers(t *testing.T) {
	t.Parallel()

	input := `{"data":{"project":{"mergeRequests":{"nodes":[
		{"iid":"42","mergedAt":"2026-11-03T10:00:00Z","author":{"username":"ann"},"approvedBy":{"nodes":[{"username":"ann"},{"username":"bob"}]}}
	]}}}}`
	got, err := parseGitLabMRs([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got[0].Approvers, []string{"ann", "bob"}) || !got[0].Approved ||
		got[0].Number != 42 || !got[0].MergedAt.Equal(time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("got %+v", got[0])
	}

	_, err = parseGitLabMRs([]byte(`{"data":null,"errors":[{"message":"boom"}]}`))
	if err == nil || err.Error() != "glab: GitLab request failed: boom" {
		t.Errorf("err = %v", err)
	}
}
