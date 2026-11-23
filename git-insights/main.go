package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		var run func([]string) error
		switch os.Args[1] {
		case "mrs":
			run = runMRs
		case "approvals":
			run = runApprovals
		}
		if run != nil {
			if err := run(os.Args[2:]); err != nil {
				fmt.Fprintf(os.Stderr, "git-insights: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	if !isGitRepo() {
		fmt.Fprintln(os.Stderr, "git-insights: not a git repository")
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "git-insights: %v\n", err)
		os.Exit(1)
	}

	if m, ok := result.(model); ok && m.selectedSHA != "" {
		gitPath, err := exec.LookPath("git")
		if err != nil {
			fmt.Fprintf(os.Stderr, "git-insights: %v\n", err)
			os.Exit(1)
		}
		if err := syscall.Exec(gitPath, []string{"git", "show", "-p", m.selectedSHA}, os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "git-insights: exec: %v\n", err)
			os.Exit(1)
		}
	}
}
