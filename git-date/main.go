// git-date selects a timestamp and runs a command with Git's commit-date
// environment variables set to that timestamp.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	datetimepicker "github.com/lcc/bubble-datetime-picker"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <command> [arguments...]\n", os.Args[0])
		os.Exit(2)
	}

	selected, err := pickDate()
	if err != nil {
		if errors.Is(err, errCancelled) {
			return
		}
		fmt.Fprintln(os.Stderr, "git-date:", err)
		os.Exit(1)
	}

	command := commandWithDate(selected, os.Args[1:])
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "git-date:", err)
		os.Exit(1)
	}
}

var errCancelled = errors.New("date selection cancelled")

func pickDate() (time.Time, error) {
	model := newPickerModel()
	program := tea.NewProgram(model)
	result, err := program.Run()
	if err != nil {
		return time.Time{}, err
	}

	picker, ok := result.(*pickerModel)
	if !ok {
		return time.Time{}, fmt.Errorf("unexpected picker result %T", result)
	}
	if picker.cancelled {
		return time.Time{}, errCancelled
	}
	if !picker.confirmed {
		return time.Time{}, errCancelled
	}
	return picker.Time(), nil
}

// pickerModel adds an explicit cancel state to the upstream picker. Its model
// treats q and ctrl+c as a normal quit, which would otherwise run the command.
type pickerModel struct {
	datetimepicker.DateAndHourModel
	onTimeStep bool
	confirmed  bool
	cancelled  bool
}

func newPickerModel() *pickerModel {
	return &pickerModel{DateAndHourModel: datetimepicker.NewDateAndHourModel()}
}

func (m *pickerModel) Init() tea.Cmd {
	return m.DateAndHourModel.Init()
}

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if m.onTimeStep {
				m.confirmed = true
				return m, tea.Quit
			}
			m.onTimeStep = true
		case "delete":
			if m.onTimeStep {
				m.onTimeStep = false
			}
		}
	}

	updated, command := m.DateAndHourModel.Update(msg)
	m.DateAndHourModel = *(updated.(*datetimepicker.DateAndHourModel))
	return m, command
}

func commandWithDate(date time.Time, args []string) *exec.Cmd {
	command := exec.Command(args[0], args[1:]...)
	gitDate := gitDateFormat(date)
	command.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+gitDate, "GIT_COMMITTER_DATE="+gitDate)
	return command
}

// gitDateFormat uses Git's documented internal date format: Unix seconds and
// a numeric offset from UTC. It avoids parsing a human-readable date again.
func gitDateFormat(date time.Time) string {
	return fmt.Sprintf("%d %s", date.Unix(), date.Format("-0700"))
}
