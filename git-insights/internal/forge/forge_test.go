package forge

import (
	"slices"
	"testing"
	"time"
)

func TestRemoteHost(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"git@gitlab.com:group/repo.git":       "gitlab.com",
		"git@github.com:owner/repo.git":       "github.com",
		"https://github.com/owner/repo.git":   "github.com",
		"ssh://git@gitlab.example:22/g/r.git": "gitlab.example",
		"gitlab.example:group/repo.git":       "gitlab.example",
	}
	for remote, want := range cases {
		if got := remoteHost(remote); got != want {
			t.Errorf("remoteHost(%q) = %q, want %q", remote, got, want)
		}
	}
}

func TestParseGitLabMRs(t *testing.T) {
	t.Parallel()

	// Two pages, as glab --paginate prints them.
	input := `{"data":{"project":{"mergeRequests":{"nodes":[
		{"title":"A","webUrl":"u1","draft":true,
		 "author":{"name":"Ann","username":"ann","publicEmail":"ann@x.com"},
		 "approvedBy":{"nodes":[{"username":"ann"}]}}
	]}}}}
	{"data":{"project":{"mergeRequests":{"nodes":[
		{"title":"B","webUrl":"u2","draft":false,
		 "author":{"name":"Bob","username":"bob","publicEmail":""},
		 "approvedBy":{"nodes":[{"username":"ann"}]}}
	]}}}}`

	got, err := parseGitLabMRs([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []MergeRequest{
		{Title: "A", URL: "u1", Draft: true, Approved: false,
			Author: Person{Name: "Ann", Username: "ann", Emails: []string{"ann@x.com"}}},
		{Title: "B", URL: "u2", Approved: true,
			Author: Person{Name: "Bob", Username: "bob"}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d merge requests, want %d", len(got), len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Title != w.Title || g.URL != w.URL || g.Draft != w.Draft || g.Approved != w.Approved ||
			g.Author.Name != w.Author.Name || g.Author.Username != w.Author.Username ||
			!slices.Equal(g.Author.Emails, w.Author.Emails) {
			t.Errorf("merge request %d = %+v, want %+v", i, g, w)
		}
	}

	if _, err := parseGitLabMRs([]byte(`{"data":{"project":null}}`)); err == nil {
		t.Error("expected error for missing project")
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
