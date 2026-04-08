package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if !isGitRepo() {
		fmt.Fprintln(os.Stderr, "git-insights: not a git repository")
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "git-insights: %v\n", err)
		os.Exit(1)
	}
}
