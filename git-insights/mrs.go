package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
)

const mrsUsage = "usage: git-insights mrs [-C dir] [--dmenu] [team]"

// runMRs lists open merge requests, optionally filtered to one team.
// With --dmenu it uses a noctalia picker, on a terminal with fzf an fzf
// picker, else it prints TSV rows.
func runMRs(args []string) error {
	// Internal modes, called back from fzf.
	if len(args) == 2 {
		switch args[0] {
		case "--open-url":
			return openURL(args[1])
		case "--list-rows":
			rows, err := fetchMRRows(args[1])
			if err != nil {
				return err
			}
			printRows(rows)
			return nil
		case "--describe":
			forge, err := detectForge()
			if err != nil {
				return err
			}
			desc, err := forge.Description(args[1])
			if err != nil {
				return err
			}
			fmt.Println(desc)
			return nil
		}
	}

	var (
		team     string
		useDmenu bool
	)
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "-C":
			i++
			if i >= len(args) {
				return errors.New(mrsUsage)
			}
			if err := os.Chdir(args[i]); err != nil {
				return err
			}
		case arg == "-dmenu" || arg == "--dmenu":
			useDmenu = true
		case strings.HasPrefix(arg, "-") || team != "":
			return errors.New(mrsUsage)
		default:
			team = arg
		}
	}

	if !isGitRepo() {
		return errors.New("not a git repository")
	}

	useFzf := false
	if useDmenu {
		if _, err := exec.LookPath("noctalia"); err != nil {
			return errors.New("missing required command: noctalia")
		}
	} else if isTerminal(os.Stdin) && isTerminal(os.Stdout) {
		_, err := exec.LookPath("fzf")
		useFzf = err == nil
	}

	path, err := teamsFile()
	if err != nil {
		return err
	}
	teams, err := loadTeams(path)
	if err != nil {
		return err
	}

	if team == "" && (useFzf || useDmenu) {
		names := sortedKeys(teams)
		if len(names) == 0 {
			return fmt.Errorf("no teams configured in: %s", path)
		}
		if useDmenu {
			team, err = pick("noctalia", []string{"dmenu", "-p", "Team"}, names)
		} else {
			team, err = pick("fzf", []string{
				"--header=TEAM (Enter: select, Esc: cancel)",
				"--prompt=team> ",
			}, names)
		}
		if err != nil || team == "" {
			return err
		}
	}

	if _, ok := teams[team]; team != "" && !ok {
		return fmt.Errorf("unknown team: %s\navailable teams: %s", team, strings.Join(sortedKeys(teams), ", "))
	}

	rows, err := mrRowsFor(teams, team)
	if err != nil {
		return err
	}

	if (useFzf || useDmenu) && len(rows) == 0 {
		fmt.Println("No open merge requests found for team: " + team)
		return nil
	}

	switch {
	case useDmenu:
		return dmenuMRs(team, rows)
	case useFzf:
		return fzfMRs(team, rows)
	default:
		fmt.Println("AUTHOR\tMR TITLE\tURL")
		printRows(rows)
		return nil
	}
}

// mrRow is one line in the merge request list.
type mrRow struct {
	Author string
	Title  string // with draft/approved markers
	URL    string
}

func (r mrRow) tsv() string {
	return r.Author + "\t" + r.Title + "\t" + r.URL
}

func printRows(rows []mrRow) {
	for _, r := range rows {
		fmt.Println(r.tsv())
	}
}

// fetchMRRows loads the team config and returns rows for team (all when empty).
func fetchMRRows(team string) ([]mrRow, error) {
	path, err := teamsFile()
	if err != nil {
		return nil, err
	}
	teams, err := loadTeams(path)
	if err != nil {
		return nil, err
	}
	return mrRowsFor(teams, team)
}

func mrRowsFor(teams Teams, team string) ([]mrRow, error) {
	forge, err := detectForge()
	if err != nil {
		return nil, err
	}
	mrs, err := forge.OpenMergeRequests()
	if err != nil {
		return nil, err
	}
	return buildMRRows(mrs, teams, team), nil
}

// buildMRRows keeps merge requests authored by team members (all when team is
// empty) and sorts them by author, then title.
func buildMRRows(mrs []MergeRequest, teams Teams, team string) []mrRow {
	members := teams[team]
	kept := make([]MergeRequest, 0, len(mrs))
	for _, mr := range mrs {
		if team == "" || slices.ContainsFunc(members, func(m TeamMember) bool { return m.matches(mr.Author) }) {
			kept = append(kept, mr)
		}
	}

	sortName := func(p Person) string {
		if p.Name != "" {
			return strings.ToLower(p.Name)
		}
		return strings.ToLower(p.Username)
	}
	slices.SortStableFunc(kept, func(a, b MergeRequest) int {
		if c := strings.Compare(sortName(a.Author), sortName(b.Author)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	})

	rows := make([]mrRow, 0, len(kept))
	for _, mr := range kept {
		author := cmp.Or(mr.Author.Name, mr.Author.Username, "Unknown")
		title := mr.Title
		if mr.Approved {
			title = "✅ " + title
		}
		if mr.Draft {
			title = "📝 " + title
		}
		rows = append(rows, mrRow{Author: author, Title: title, URL: mr.URL})
	}
	return rows
}

func dmenuMRs(team string, rows []mrRow) error {
	labels := make([]string, len(rows))
	for i, r := range rows {
		labels[i] = fmt.Sprintf("%d. %s  %s", i+1, r.Author, r.Title)
	}
	selected, err := pick("noctalia", []string{"dmenu", "-p", team + ": Open MR"}, labels)
	if err != nil || selected == "" {
		return err
	}
	i := slices.Index(labels, selected)
	if i < 0 {
		return fmt.Errorf("unknown merge request selection: %s", selected)
	}
	return openURL(rows[i].URL)
}

func fzfMRs(team string, rows []mrRow) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	lines := make([]string, len(rows))
	for i, r := range rows {
		lines[i] = r.tsv()
	}
	_, err = pick("fzf", []string{
		"--delimiter=\t",
		"--with-nth=1,2",
		"--track",
		"--id-nth=3",
		"--header=AUTHOR  MR TITLE (Enter: open, Ctrl-C: exit)",
		`--preview="$GIT_INSIGHTS_BIN" mrs --describe {3} | glow -s dark -w "$FZF_PREVIEW_COLUMNS" -`,
		"--preview-window=right,50%,wrap",
		`--bind=enter:execute-silent("$GIT_INSIGHTS_BIN" mrs --open-url {3}),esc:ignore,ctrl-g:ignore,ctrl-c:abort`,
		`--bind=every(60):reload-sync("$GIT_INSIGHTS_BIN" mrs --list-rows "$GIT_INSIGHTS_TEAM")`,
		"--prompt=" + team + "> ",
	}, lines, "GIT_INSIGHTS_BIN="+self, "GIT_INSIGHTS_TEAM="+team)
	return err
}

// pick runs a picker with lines on stdin and returns the selected line.
// Returns "" without error when the user cancels (exit code 1 or 130).
func pick(name string, args []string, lines []string, env ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if code := exitErr.ExitCode(); code == 1 || code == 130 {
			return "", nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", name, err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

func openURL(url string) error {
	var openers []string
	switch runtime.GOOS {
	case "darwin":
		openers = []string{"open"}
	case "linux":
		openers = []string{"xdg-open", "gio"}
	}
	for _, opener := range openers {
		if _, err := exec.LookPath(opener); err != nil {
			continue
		}
		args := []string{url}
		if opener == "gio" {
			args = []string{"open", url}
		}
		return exec.Command(opener, args...).Run()
	}
	return fmt.Errorf("could not find a URL opener for this platform\n%s", url)
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func sortedKeys(teams Teams) []string {
	names := make([]string, 0, len(teams))
	for name := range teams {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
