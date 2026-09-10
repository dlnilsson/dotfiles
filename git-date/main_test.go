package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPickerConfirmAndCancel(t *testing.T) {
	t.Run("confirm", func(t *testing.T) {
		model := newPickerModel()
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		_, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if model.state != pickerConfirmed || command == nil {
			t.Fatal("second enter should confirm the selection")
		}
	})

	t.Run("cancel", func(t *testing.T) {
		model := newPickerModel()
		_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		if model.state != pickerCancelled || command == nil {
			t.Fatal("q should cancel the selection")
		}
	})
}

func TestPickerAcceptsTypedTime(t *testing.T) {
	tests := []struct {
		input      string
		wantHour   int
		wantMinute int
		wantSecond int
	}{
		{input: "10:00", wantHour: 10},
		{input: "23:59:58", wantHour: 23, wantMinute: 59, wantSecond: 58},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			model := newPickerModel()
			model.Update(tea.KeyMsg{Type: tea.KeyEnter})
			model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(test.input)})
			model.Update(tea.KeyMsg{Type: tea.KeyEnter})

			selected := model.Time()
			if selected.Hour() != test.wantHour || selected.Minute() != test.wantMinute || selected.Second() != test.wantSecond {
				t.Fatalf("selected time is %s, want %02d:%02d:%02d", selected.Format("15:04:05"), test.wantHour, test.wantMinute, test.wantSecond)
			}
		})
	}
}

func TestPickerRejectsInvalidTypedTime(t *testing.T) {
	model := newPickerModel()
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("25:00")})
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if model.state == pickerConfirmed {
		t.Fatal("invalid time should keep the time input open")
	}
	if model.validationErr == nil {
		t.Fatal("invalid time should show a validation error")
	}
}

func TestPickerShowsControlsForEachStep(t *testing.T) {
	model := newPickerModel()
	if view := model.View(); !strings.Contains(view, "enter set time") {
		t.Fatalf("date view does not show how to continue:\n%s", view)
	}

	model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := model.View()
	for _, text := range []string{"Time (HH:MM or HH:MM:SS)", "enter confirm", "esc back to date"} {
		if !strings.Contains(view, text) {
			t.Fatalf("time view does not contain %q:\n%s", text, view)
		}
	}
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
