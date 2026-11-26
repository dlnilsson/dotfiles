// Package insights is the interactive git history TUI.
package insights

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
)

// Run shows the TUI for the repository in the current directory. Selecting a
// firefighting commit runs git show for that commit, then returns to the TUI.
func Run() error {
	if !isGitRepo() {
		return errors.New("not a git repository")
	}

	_, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run()
	return err
}
