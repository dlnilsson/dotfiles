package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// gitLab implements Forge with the glab CLI.
type gitLab struct{}

const gitLabMRFields = `
      nodes {
        title webUrl draft
        author { name username publicEmail }
        approvedBy { nodes { username } }
      }
      pageInfo { hasNextPage endCursor }`

// GraphQL, since the REST list has no approvals.
const gitLabOpenMRQuery = `
query($project: ID!, $endCursor: String) {
  project(fullPath: $project) {
    mergeRequests(state: opened, first: 100, after: $endCursor) {` + gitLabMRFields + `
    }
  }
}`

// %s is the optional label filter.
const gitLabMergedMRQuery = `
query($project: ID!, $count: Int, $after: String) {
  project(fullPath: $project) {
    mergeRequests(state: merged, sort: MERGED_AT_DESC, first: $count, after: $after%s) {` + gitLabMRFields + `
    }
  }
}`

// gitLabPageSize is GitLab's max for first:.
const gitLabPageSize = 100

type gitLabMRNode struct {
	Title  string `json:"title"`
	WebURL string `json:"webUrl"`
	Draft  bool   `json:"draft"`
	Author struct {
		Name        string `json:"name"`
		Username    string `json:"username"`
		PublicEmail string `json:"publicEmail"`
	} `json:"author"`
	ApprovedBy struct {
		Nodes []struct {
			Username string `json:"username"`
		} `json:"nodes"`
	} `json:"approvedBy"`
}

type gitLabPage struct {
	Data struct {
		Project *struct {
			MergeRequests struct {
				Nodes    []gitLabMRNode `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"mergeRequests"`
		} `json:"project"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (gitLab) Project() (string, error) {
	// :fullpath only works in REST endpoints, so GraphQL queries need this.
	out, err := runCommand("glab", "api", "projects/:fullpath")
	if err != nil {
		return "", err
	}
	var project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	}
	if err := json.Unmarshal(out, &project); err != nil {
		return "", fmt.Errorf("glab: parse project: %w", err)
	}
	return project.PathWithNamespace, nil
}

func (g gitLab) OpenMergeRequests() ([]MergeRequest, error) {
	project, err := g.Project()
	if err != nil {
		return nil, err
	}
	out, err := runCommand("glab", "api", "graphql", "--paginate",
		"-f", "project="+project,
		"-f", "query="+gitLabOpenMRQuery)
	if err != nil {
		return nil, err
	}
	return parseGitLabMRs(out)
}

func (g gitLab) MergedMergeRequests(count int, skipLabel string, progress func(scanned int)) ([]MergeRequest, error) {
	project, err := g.Project()
	if err != nil {
		return nil, err
	}
	filter := ""
	if skipLabel != "" {
		filter = ", not: {labels: [" + strconv.Quote(skipLabel) + "]}"
	}
	query := fmt.Sprintf(gitLabMergedMRQuery, filter)

	var (
		result []MergeRequest
		cursor string
	)
	for len(result) < count {
		progress(len(result))
		args := []string{"api", "graphql",
			"-f", "query=" + query,
			"-f", "project=" + project,
			"-F", "count=" + strconv.Itoa(min(count-len(result), gitLabPageSize)),
		}
		if cursor != "" {
			args = append(args, "-f", "after="+cursor)
		}
		out, err := runCommand("glab", args...)
		if err != nil {
			return nil, err
		}
		var page gitLabPage
		if err := json.Unmarshal(out, &page); err != nil {
			return nil, fmt.Errorf("glab: parse merge requests: %w", err)
		}
		if err := page.err(); err != nil {
			return nil, err
		}
		mrs := page.Data.Project.MergeRequests
		for _, n := range mrs.Nodes {
			result = append(result, n.toMergeRequest())
		}
		if !mrs.PageInfo.HasNextPage {
			break
		}
		cursor = mrs.PageInfo.EndCursor
	}
	return result, nil
}

func (gitLab) Description(url string) (string, error) {
	out, err := runCommand("glab", "mr", "view", url, "--output", "json")
	if err != nil {
		return "", err
	}
	var mr struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &mr); err != nil {
		return "", fmt.Errorf("glab: parse merge request: %w", err)
	}
	return mr.Description, nil
}

// parseGitLabMRs parses paginated GraphQL output: one JSON document per page.
func parseGitLabMRs(data []byte) ([]MergeRequest, error) {
	var result []MergeRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var p gitLabPage
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("glab: parse merge requests: %w", err)
		}
		if err := p.err(); err != nil {
			return nil, err
		}
		for _, n := range p.Data.Project.MergeRequests.Nodes {
			result = append(result, n.toMergeRequest())
		}
	}
	return result, nil
}

// err returns the GraphQL error of the page, if any.
func (p gitLabPage) err() error {
	if len(p.Errors) > 0 {
		return errors.New("glab: GitLab request failed: " + p.Errors[0].Message)
	}
	if p.Data.Project == nil {
		return errors.New("glab: project not found")
	}
	return nil
}

func (n gitLabMRNode) toMergeRequest() MergeRequest {
	mr := MergeRequest{
		Title: n.Title,
		URL:   n.WebURL,
		Draft: n.Draft,
		Author: Person{
			Name:     n.Author.Name,
			Username: n.Author.Username,
		},
	}
	if n.Author.PublicEmail != "" {
		mr.Author.Emails = []string{n.Author.PublicEmail}
	}
	for _, a := range n.ApprovedBy.Nodes {
		mr.Approvers = append(mr.Approvers, a.Username)
		// Self-approval by the author does not count.
		if a.Username != n.Author.Username {
			mr.Approved = true
		}
	}
	return mr
}
