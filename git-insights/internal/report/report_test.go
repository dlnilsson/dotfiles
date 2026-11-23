package report

import (
	"errors"
	"io"
	"iter"
	"slices"
	"testing"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
)

// fakeForge serves merged merge requests from a slice, then err if set.
type fakeForge struct {
	merged []forge.MergeRequest
	err    error
	pulled int // merge requests the caller consumed
}

func (f *fakeForge) Project() (string, error)                         { return "group/repo", nil }
func (f *fakeForge) OpenMergeRequests() ([]forge.MergeRequest, error) { return nil, nil }
func (f *fakeForge) Description(string) (string, error)               { return "", nil }
func (f *fakeForge) MergedMergeRequests(string) iter.Seq2[forge.MergeRequest, error] {
	return func(yield func(forge.MergeRequest, error) bool) {
		for _, mr := range f.merged {
			f.pulled++
			if !yield(mr, nil) {
				return
			}
		}
		if f.err != nil {
			yield(forge.MergeRequest{}, f.err)
		}
	}
}

func TestCollectMerged(t *testing.T) {
	t.Parallel()

	status := func(int, int) string { return "" }
	f := &fakeForge{merged: []forge.MergeRequest{
		{Number: 1, Approved: true}, {Number: 2}, {Number: 3, Approved: true}, {Number: 4}, {Number: 5},
	}}

	got, scanned, err := CollectMerged(f, 2, func(mr forge.MergeRequest) bool { return !mr.Approved }, io.Discard, status)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 2 || got[1].Number != 4 || scanned != 4 || f.pulled != 4 {
		t.Errorf("got %v, scanned %d, pulled %d", got, scanned, f.pulled)
	}

	// Fewer than count available: returns what exists.
	got, _, err = CollectMerged(&fakeForge{merged: f.merged}, 10, nil, io.Discard, status)
	if err != nil || len(got) != 5 {
		t.Errorf("got %d merge requests, err %v", len(got), err)
	}

	boom := errors.New("boom")
	if _, _, err := CollectMerged(&fakeForge{err: boom}, 10, nil, io.Discard, status); !errors.Is(err, boom) {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestRankCounts(t *testing.T) {
	t.Parallel()

	got := RankCounts(map[string]int{"cid": 1, "bob": 2, "ann": 1})
	if want := []UserCount{{"bob", 2}, {"ann", 1}, {"cid", 1}}; !slices.Equal(got, want) {
		t.Errorf("RankCounts = %v, want %v", got, want)
	}
}
