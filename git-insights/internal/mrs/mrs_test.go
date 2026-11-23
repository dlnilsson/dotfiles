package mrs

import (
	"slices"
	"strings"
	"testing"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
	"github.com/dlnilsson/dotfiles/git-insights/internal/teams"
)

func TestBuildRows(t *testing.T) {
	t.Parallel()

	mrs := []forge.MergeRequest{
		{Title: "zeta", URL: "u1", Author: forge.Person{Name: "bob"}},
		{Title: "Alpha", URL: "u2", Draft: true, Approved: true, Author: forge.Person{Name: "Bob"}},
		{Title: "x", URL: "u3", Author: forge.Person{Username: "ann"}},
		{Title: "y", URL: "u4", Author: forge.Person{}},
	}
	teams := teams.Teams{"t": {{Name: "bob"}}}

	var all []string
	for _, r := range buildRows(mrs, teams, "") {
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

	team := buildRows(mrs, teams, "t")
	if len(team) != 2 || team[0].URL != "u2" || team[1].URL != "u1" {
		t.Errorf("team rows = %+v", team)
	}
}
