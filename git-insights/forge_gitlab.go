package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// gitLab implements Forge with the glab CLI.
type gitLab struct{}

// GraphQL, since the REST list has no approvals.
const gitLabMRQuery = `
query($project: ID!, $endCursor: String) {
  project(fullPath: $project) {
    mergeRequests(state: opened, first: 100, after: $endCursor) {
      nodes {
        title webUrl draft
        author { name username publicEmail }
        approvedBy { nodes { username } }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

func (gitLab) OpenMergeRequests() ([]MergeRequest, error) {
	// :fullpath only works in REST endpoints, so resolve the project path first.
	out, err := runCommand("glab", "api", "projects/:fullpath")
	if err != nil {
		return nil, err
	}
	var project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	}
	if err := json.Unmarshal(out, &project); err != nil {
		return nil, fmt.Errorf("glab: parse project: %w", err)
	}

	out, err = runCommand("glab", "api", "graphql", "--paginate",
		"-f", "project="+project.PathWithNamespace,
		"-f", "query="+gitLabMRQuery)
	if err != nil {
		return nil, err
	}
	return parseGitLabMRs(out)
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
	type page struct {
		Data struct {
			Project *struct {
				MergeRequests struct {
					Nodes []struct {
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
					} `json:"nodes"`
				} `json:"mergeRequests"`
			} `json:"project"`
		} `json:"data"`
	}

	var result []MergeRequest
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var p page
		if err := dec.Decode(&p); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("glab: parse merge requests: %w", err)
		}
		if p.Data.Project == nil {
			return nil, errors.New("glab: project not found")
		}
		for _, n := range p.Data.Project.MergeRequests.Nodes {
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
			// Self-approval by the author does not count.
			for _, a := range n.ApprovedBy.Nodes {
				if a.Username != n.Author.Username {
					mr.Approved = true
					break
				}
			}
			result = append(result, mr)
		}
	}
	return result, nil
}
