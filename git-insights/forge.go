package main

import (
	"errors"
	"fmt"
	"iter"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

// Person is the author of a merge request.
type Person struct {
	Name     string
	Username string
	Emails   []string
}

// MergeRequest is an open merge request (GitLab) or pull request (GitHub).
type MergeRequest struct {
	Number    int       // GitLab iid or GitHub number
	Ref       string    // display reference: "!123" on GitLab, "#123" on GitHub
	MergedAt  time.Time // zero unless merged
	Title     string
	URL       string
	Draft     bool
	Approved  bool     // approved by someone other than the author
	Approvers []string // usernames, the author included if they self-approved
	Author    Person
}

// Forge is a code hosting service that serves merge requests for the
// current repository.
type Forge interface {
	// Project returns the repository name on the forge, e.g. "group/repo".
	Project() (string, error)
	OpenMergeRequests() ([]MergeRequest, error)
	// MergedMergeRequests yields merge requests, most recently merged first,
	// leaving out those labeled skipLabel (none when empty). It fetches more
	// pages until the caller stops or none are left; on failure it yields an error.
	MergedMergeRequests(skipLabel string) iter.Seq2[MergeRequest, error]
	// Description returns the markdown description of the merge request at url.
	Description(url string) (string, error)
}

// detectForge picks the forge from the host of the origin remote.
func detectForge() (Forge, error) {
	output, err := runGit("remote", "get-url", "origin")
	if err != nil {
		return nil, err
	}
	if remoteHost(strings.TrimSpace(output)) == "github.com" {
		return nil, errors.New("GitHub is not supported yet")
	}
	// Any other host is treated as GitLab, so self-hosted instances work.
	return gitLab{}, nil
}

// remoteHost returns the host of a git remote URL, either URL form
// (https://host/path, ssh://git@host:22/path) or scp-like form (git@host:path).
func remoteHost(remote string) string {
	if strings.Contains(remote, "://") {
		u, err := url.Parse(remote)
		if err != nil {
			return ""
		}
		return u.Hostname()
	}
	host, _, _ := strings.Cut(remote, ":")
	if _, after, ok := strings.Cut(host, "@"); ok {
		host = after
	}
	return host
}

// runCommand executes a command and returns stdout.
func runCommand(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("%s: %w\n%s", name, err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}
