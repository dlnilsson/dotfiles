// Package mrs lists open merge requests, optionally filtered to one team, in
// fzf, a noctalia dmenu, or as TSV.
package mrs

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"github.com/dlnilsson/dotfiles/git-insights/internal/forge"
	"github.com/dlnilsson/dotfiles/git-insights/internal/report"
	"github.com/dlnilsson/dotfiles/git-insights/internal/teams"
)

// Options configures Run.
type Options struct {
	Team  string // empty: all authors, or pick one in fzf/dmenu
	Dmenu bool   // use a noctalia dmenu picker
}

// Run lists open merge requests. With Dmenu it uses a noctalia picker, on a
// terminal with fzf an fzf picker, else it prints TSV rows to out.
func Run(out io.Writer, opts Options) error {
	team := opts.Team
	useFzf := false
	if opts.Dmenu {
		if _, err := exec.LookPath("noctalia"); err != nil {
			return errors.New("missing required command: noctalia")
		}
	} else if report.IsTerminal(os.Stdin) && report.IsTerminal(out) {
		_, err := exec.LookPath("fzf")
		useFzf = err == nil
	}

	path, err := teams.File()
	if err != nil {
		return err
	}
	all, err := teams.Load(path)
	if err != nil {
		return err
	}

	if team == "" && (useFzf || opts.Dmenu) {
		names := all.Names()
		if len(names) == 0 {
			return fmt.Errorf("no teams configured in: %s", path)
		}
		if opts.Dmenu {
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

	if _, ok := all[team]; team != "" && !ok {
		return fmt.Errorf("unknown team: %s\navailable teams: %s", team, strings.Join(all.Names(), ", "))
	}

	rows, err := rowsFor(all, team)
	if err != nil {
		return err
	}

	if (useFzf || opts.Dmenu) && len(rows) == 0 {
		fmt.Fprintln(out, "No open merge requests found for team: "+team)
		return nil
	}

	switch {
	case opts.Dmenu:
		return dmenu(team, rows)
	case useFzf:
		return fzf(team, rows)
	default:
		fmt.Fprintln(out, "AUTHOR\tMR TITLE\tURL")
		printRows(out, rows)
		return nil
	}
}

// ListRows prints TSV rows for team (all when empty), without a header.
func ListRows(out io.Writer, team string) error {
	path, err := teams.File()
	if err != nil {
		return err
	}
	all, err := teams.Load(path)
	if err != nil {
		return err
	}
	rows, err := rowsFor(all, team)
	if err != nil {
		return err
	}
	printRows(out, rows)
	return nil
}

// Describe prints the description of the merge request at url.
func Describe(out io.Writer, url string) error {
	f, err := forge.Detect()
	if err != nil {
		return err
	}
	desc, err := f.Description(url)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, desc)
	return nil
}

// row is one line in the merge request list.
type row struct {
	Author string
	Title  string // with draft/approved markers
	URL    string
}

func (r row) tsv() string {
	return r.Author + "\t" + r.Title + "\t" + r.URL
}

func printRows(out io.Writer, rows []row) {
	for _, r := range rows {
		fmt.Fprintln(out, r.tsv())
	}
}

func rowsFor(all teams.Teams, team string) ([]row, error) {
	f, err := forge.Detect()
	if err != nil {
		return nil, err
	}
	mrs, err := f.OpenMergeRequests()
	if err != nil {
		return nil, err
	}
	return buildRows(mrs, all, team), nil
}

// buildRows keeps merge requests authored by team members (all when team is
// empty) and sorts them by author, then title.
func buildRows(mrs []forge.MergeRequest, all teams.Teams, team string) []row {
	members := all[team]
	kept := make([]forge.MergeRequest, 0, len(mrs))
	for _, mr := range mrs {
		if team == "" || slices.ContainsFunc(members, func(m teams.Member) bool { return m.Matches(mr.Author) }) {
			kept = append(kept, mr)
		}
	}

	sortName := func(p forge.Person) string {
		if p.Name != "" {
			return strings.ToLower(p.Name)
		}
		return strings.ToLower(p.Username)
	}
	slices.SortStableFunc(kept, func(a, b forge.MergeRequest) int {
		if c := strings.Compare(sortName(a.Author), sortName(b.Author)); c != 0 {
			return c
		}
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	})

	rows := make([]row, 0, len(kept))
	for _, mr := range kept {
		author := cmp.Or(mr.Author.Name, mr.Author.Username, "Unknown")
		title := mr.Title
		if mr.Approved {
			title = "✅ " + title
		}
		if mr.Draft {
			title = "📝 " + title
		}
		rows = append(rows, row{Author: author, Title: title, URL: mr.URL})
	}
	return rows
}

func dmenu(team string, rows []row) error {
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
	return OpenURL(rows[i].URL)
}

func fzf(team string, rows []row) error {
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
		`--preview="$GIT_INSIGHTS_BIN" mrs describe {3} | glow -s dark -w "$FZF_PREVIEW_COLUMNS" -`,
		"--preview-window=right,50%,wrap",
		`--bind=enter:execute-silent("$GIT_INSIGHTS_BIN" mrs open-url {3}),esc:ignore,ctrl-g:ignore,ctrl-c:abort`,
		`--bind=every(60):reload-sync("$GIT_INSIGHTS_BIN" mrs list-rows "$GIT_INSIGHTS_TEAM")`,
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

// OpenURL opens url in the default browser.
func OpenURL(url string) error {
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
