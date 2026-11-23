package teams

import (
	"testing"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
)

func TestParseTeams(t *testing.T) {
	t.Parallel()

	teams, err := Parse([]byte(`{"a":[{"name":"Ann"},{"email":["b@x.com"]},{"username":"c"}]}`), "teams.json")
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
		if _, err := Parse([]byte(bad), "teams.json"); err == nil {
			t.Errorf("expected error for %s", bad)
		}
	}
}

func TestMemberMatches(t *testing.T) {
	t.Parallel()

	author := forge.Person{Name: "Ann Lee", Username: "ann", Emails: []string{"Ann@X.com"}}
	cases := []struct {
		member Member
		want   bool
	}{
		{Member{Name: "ann lee"}, true},
		{Member{Username: "ANN"}, true},
		{Member{Email: []string{"other@x.com", "ann@x.com"}}, true},
		{Member{Name: "Bob"}, false},
		{Member{Email: []string{""}}, false},
	}
	for _, c := range cases {
		if got := c.member.Matches(author); got != c.want {
			t.Errorf("%+v.matches = %v, want %v", c.member, got, c.want)
		}
	}

	// An empty member value never matches an empty author value.
	if (Member{Name: "x"}).Matches(forge.Person{}) {
		t.Error("expected no match for empty author")
	}
}
