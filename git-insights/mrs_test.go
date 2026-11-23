package main

import (
	"slices"
	"strings"
	"testing"
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

func TestParseTeams(t *testing.T) {
	t.Parallel()

	teams, err := parseTeams([]byte(`{"a":[{"name":"Ann"},{"email":["b@x.com"]},{"username":"c"}]}`), "teams.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(teams["a"]) != 3 {
		t.Fatalf("expected 3 members, got %d", len(teams["a"]))
	}

	for _, bad := range []string{
		`[]`,
		`{"a":{}}`,
		`{"a":[{"email":"not-an-array"}]}`,
		`{"a":[{"name":""}]}`,
		`{"a":[{"email":[""]}]}`,
	} {
		if _, err := parseTeams([]byte(bad), "teams.json"); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

func TestTeamMemberMatches(t *testing.T) {
	t.Parallel()

	author := Person{Name: "Ann Lee", Username: "ann", Emails: []string{"Ann@X.com"}}
	cases := []struct {
		member TeamMember
		want   bool
	}{
		{TeamMember{Name: "ann lee"}, true},
		{TeamMember{Username: "ANN"}, true},
		{TeamMember{Email: []string{"other@x.com", "ann@x.com"}}, true},
		{TeamMember{Name: "Bob"}, false},
		{TeamMember{Email: []string{""}}, false},
	}
	for _, c := range cases {
		if got := c.member.matches(author); got != c.want {
			t.Errorf("%+v.matches = %v, want %v", c.member, got, c.want)
		}
	}

	// An empty member value never matches an empty author value.
	if (TeamMember{Name: "x"}).matches(Person{}) {
		t.Error("expected no match for empty author")
	}
}

func TestBuildMRRows(t *testing.T) {
	t.Parallel()

	mrs := []MergeRequest{
		{Title: "zeta", URL: "u1", Author: Person{Name: "bob"}},
		{Title: "Alpha", URL: "u2", Draft: true, Approved: true, Author: Person{Name: "Bob"}},
		{Title: "x", URL: "u3", Author: Person{Username: "ann"}},
		{Title: "y", URL: "u4", Author: Person{}},
	}
	teams := Teams{"t": {{Name: "bob"}}}

	var all []string
	for _, r := range buildMRRows(mrs, teams, "") {
		all = append(all, r.tsv())
	}
	want := []string{
		"Unknown\ty\tu4",
		"ann\tx\tu3",
		"Bob\t📝 ✅ Alpha\tu2",
		"bob\tzeta\tu1",
	}
	if !slices.Equal(all, want) {
		t.Errorf("all rows:\n%s\nwant:\n%s", strings.Join(all, "\n"), strings.Join(want, "\n"))
	}

	team := buildMRRows(mrs, teams, "t")
	if len(team) != 2 || team[0].URL != "u2" || team[1].URL != "u1" {
		t.Errorf("team rows = %+v", team)
	}
}
