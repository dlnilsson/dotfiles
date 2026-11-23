package main

import (
	"errors"
	"iter"
	"slices"
	"testing"
	"time"
)

// fakeForge serves merged merge requests from a slice, then err if set.
type fakeForge struct {
	merged []MergeRequest
	err    error
	pulled int // merge requests the caller consumed
}

func (f *fakeForge) Project() (string, error)                   { return "group/repo", nil }
func (f *fakeForge) OpenMergeRequests() ([]MergeRequest, error) { return nil, nil }
func (f *fakeForge) Description(string) (string, error)         { return "", nil }
func (f *fakeForge) MergedMergeRequests(string) iter.Seq2[MergeRequest, error] {
	return func(yield func(MergeRequest, error) bool) {
		for _, mr := range f.merged {
			f.pulled++
			if !yield(mr, nil) {
				return
			}
		}
		if f.err != nil {
			yield(MergeRequest{}, f.err)
		}
	}
}

func TestCollectMerged(t *testing.T) {
	t.Parallel()

	status := func(int, int) string { return "" }
	f := &fakeForge{merged: []MergeRequest{
		{Number: 1, Approved: true}, {Number: 2}, {Number: 3, Approved: true}, {Number: 4}, {Number: 5},
	}}

	got, scanned, err := collectMerged(f, 2, func(mr MergeRequest) bool { return !mr.Approved }, status)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 2 || got[1].Number != 4 || scanned != 4 || f.pulled != 4 {
		t.Errorf("got %v, scanned %d, pulled %d", got, scanned, f.pulled)
	}

	// Fewer than count available: returns what exists.
	got, _, err = collectMerged(&fakeForge{merged: f.merged}, 10, nil, status)
	if err != nil || len(got) != 5 {
		t.Errorf("got %d merge requests, err %v", len(got), err)
	}

	boom := errors.New("boom")
	if _, _, err := collectMerged(&fakeForge{err: boom}, 10, nil, status); !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestSortUnapproved(t *testing.T) {
	t.Parallel()

	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	mrs := []MergeRequest{
		{Number: 5, MergedAt: day(1), Author: Person{Username: "bob"}},
		{Number: 9, MergedAt: day(2), Author: Person{Username: "bob"}},
		{Number: 7, MergedAt: day(2), Author: Person{Username: "bob"}},
		{Number: 1, MergedAt: day(1), Author: Person{Username: "ann"}},
	}
	sortUnapproved(mrs)
	var got []int
	for _, mr := range mrs {
		got = append(got, mr.Number)
	}
	if want := []int{1, 7, 9, 5}; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}

	ranks := rankAuthors(mrs)
	if want := []userCount{{"bob", 3}, {"ann", 1}}; !slices.Equal(ranks, want) {
		t.Errorf("rankAuthors = %v, want %v", ranks, want)
	}
}
