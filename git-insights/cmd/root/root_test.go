package root

import (
	"bytes"
	"strings"
	"testing"
)

func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := New()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestRoot_RegistersCommands(t *testing.T) {
	out, err := execute(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mrs", "approvals", "reviews", "unapproved", "completion", "--directory"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q:\n%s", want, out)
		}
	}
	for _, hidden := range []string{"open-url", "list-rows", "describe"} {
		if strings.Contains(out, hidden) {
			t.Fatalf("help lists hidden command %q:\n%s", hidden, out)
		}
	}
}

func TestRoot_RejectsBadArguments(t *testing.T) {
	cases := [][]string{
		{"approvals", "abc"},
		{"approvals", "0"},
		{"approvals", "1", "2"},
		{"approvals", "--timeline"},
		{"reviews", "--csv", "--timeline"},
		{"unapproved", "-5"},
		{"mrs", "a", "b"},
		{"stray"},
	}
	for _, args := range cases {
		if _, err := execute(t, args...); err == nil {
			t.Errorf("%v: expected error", args)
		}
	}
}
