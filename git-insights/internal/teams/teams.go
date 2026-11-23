// Package teams loads the team configuration used to filter merge requests
// by author.
package teams

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
)

// Member identifies a person by name, username or any of their emails.
type Member struct {
	Name     string   `json:"name"`
	Username string   `json:"username"`
	Email    []string `json:"email"`
}

// Teams maps team name to its members.
type Teams map[string][]Member

// File returns $GIT_INSIGHTS_TEAMS_FILE, else
// $XDG_CONFIG_HOME/git-insights/teams.json (XDG_CONFIG_HOME defaults to ~/.config).
func File() (string, error) {
	if path := os.Getenv("GIT_INSIGHTS_TEAMS_FILE"); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "git-insights", "teams.json"), nil
}

// Load reads and validates the team configuration at path.
func Load(path string) (Teams, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("team configuration is not readable: %w", err)
	}
	return Parse(data, path)
}

// Parse validates team configuration data; path is used in errors.
func Parse(data []byte, path string) (Teams, error) {
	var teams Teams
	if err := json.Unmarshal(data, &teams); err != nil {
		return nil, fmt.Errorf("invalid team configuration %s: %w", path, err)
	}
	for team, members := range teams {
		for i, m := range members {
			if m.Name == "" && m.Username == "" && !hasNonEmpty(m.Email) {
				return nil, fmt.Errorf("invalid team configuration %s: team %q member %d has no name, username or email", path, team, i)
			}
		}
	}
	return teams, nil
}

func hasNonEmpty(values []string) bool {
	for _, v := range values {
		if v != "" {
			return true
		}
	}
	return false
}

// Matches reports whether the member is p, comparing case-insensitively.
func (m Member) Matches(p forge.Person) bool {
	eq := func(a, b string) bool { return b != "" && strings.EqualFold(a, b) }
	if eq(p.Name, m.Name) || eq(p.Username, m.Username) {
		return true
	}
	for _, e := range m.Email {
		for _, pe := range p.Emails {
			if eq(pe, e) {
				return true
			}
		}
	}
	return false
}

// Names returns the team names, sorted.
func (t Teams) Names() []string {
	names := make([]string, 0, len(t))
	for name := range t {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
