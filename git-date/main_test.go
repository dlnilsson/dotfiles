package main

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPickerConfirmAndCancel(t *testing.T) {
	t.Run("confirm", func(t *testing.T) {
		model := newPickerModel()
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		_, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if !model.confirmed || model.cancelled || command == nil {
			t.Fatal("second enter should confirm the selection")
		}
	})

	t.Run("cancel", func(t *testing.T) {
		model := newPickerModel()
		_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		if !model.cancelled || model.confirmed || command == nil {
			t.Fatal("q should cancel the selection")
		}
	})
}

func TestCommandWithDate(t *testing.T) {
	date := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.FixedZone("UTC+2", 2*60*60))
	command := commandWithDate(date, []string{"sh", "-c", "printf '%s\\n%s\\n' \"$GIT_AUTHOR_DATE\" \"$GIT_COMMITTER_DATE\""})

	var output bytes.Buffer
	command.Stdout = &output
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}

	const want = "1704157445 +0200\n1704157445 +0200\n"
	if output.String() != want {
		t.Fatalf("child received %q, want %q", output.String(), want)
	}
}
