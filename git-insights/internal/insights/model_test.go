package insights

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterOnFirefightingDoesNotQuit(t *testing.T) {
	m := model{activeTab: int(analysisFirefighting)}
	m.results[analysisFirefighting] = &resultMsg{
		id:           analysisFirefighting,
		firefighting: []FirefightEntry{{Hash: "abc1234", Subject: "hotfix: thing"}},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command to show the commit")
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		t.Fatal("enter must not quit the TUI")
	}
}
