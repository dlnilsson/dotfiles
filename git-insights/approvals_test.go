package main

import (
	"slices"
	"testing"
)

func TestRankApprovers(t *testing.T) {
	t.Parallel()

	mrs := []MergeRequest{
		{Approvers: []string{"bob", "ann"}},
		{Approvers: []string{"cid"}},
		{Approvers: []string{"bob"}},
		{},
	}
	got := rankApprovers(mrs)
	want := []approverCount{{"bob", 2}, {"ann", 1}, {"cid", 1}}
	if !slices.Equal(got, want) {
		t.Errorf("rankApprovers = %v, want %v", got, want)
	}
}

func TestGitLabMRApprovers(t *testing.T) {
	t.Parallel()

	input := `{"data":{"project":{"mergeRequests":{"nodes":[
		{"author":{"username":"ann"},"approvedBy":{"nodes":[{"username":"ann"},{"username":"bob"}]}}
	]}}}}`
	got, err := parseGitLabMRs([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got[0].Approvers, []string{"ann", "bob"}) || !got[0].Approved {
		t.Errorf("got %+v", got[0])
	}

	_, err = parseGitLabMRs([]byte(`{"data":null,"errors":[{"message":"boom"}]}`))
	if err == nil || err.Error() != "glab: GitLab request failed: boom" {
		t.Errorf("err = %v", err)
	}
}
