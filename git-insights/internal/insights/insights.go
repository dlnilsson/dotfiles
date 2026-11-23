// Package insights is the interactive git history TUI.
package insights

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// Run shows the TUI for the repository in the current directory. Selecting a
// firefighting commit replaces the process with git show for that commit.
func Run() error {
	if !isGitRepo() {
		return errors.New("not a git repository")
	}

	result, err := tea.NewProgram(newModel(), tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}

	if m, ok := result.(model); ok && m.selectedSHA != "" {
		gitPath, err := exec.LookPath("git")
		if err != nil {
			return err
		}
		return syscall.Exec(gitPath, []string{"git", "show", "-p", m.selectedSHA}, os.Environ())
	}
	return nil
}
