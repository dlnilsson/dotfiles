// git-date selects a timestamp and runs a command with Git's commit-date
// environment variables set to that timestamp.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	datepicker "github.com/ethanefung/bubble-datepicker"
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
	if picker.state == pickerCancelled {
		return time.Time{}, errCancelled
	}
	if picker.state != pickerConfirmed {
		return time.Time{}, errCancelled
	}
	return picker.Time(), nil
}

type pickerState int

const (
	selectingDate pickerState = iota
	enteringTime
	pickerConfirmed
	pickerCancelled
)

type clockTime struct {
	hour   int
	minute int
	second int
}

type pickerModel struct {
	datepicker    datepicker.Model
	timeInput     textinput.Model
	selectedClock clockTime
	state         pickerState
	validationErr error
}

func newPickerModel() *pickerModel {
	calendar := datepicker.New(time.Now())
	calendar.SelectDate()

	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "00:00"
	input.CharLimit = 8
	input.Width = 20
	input.Validate = validateClockDraft
	input.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	input.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	return &pickerModel{
		datepicker: calendar,
		timeInput:  input,
		state:      selectingDate,
	}
}

func (m *pickerModel) Init() tea.Cmd {
	return nil
}

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			m.state = pickerCancelled
			return m, tea.Quit
		}

		switch m.state {
		case selectingDate:
			if key.String() == "enter" {
				m.state = enteringTime
				m.datepicker.Blur()
				return m, m.timeInput.Focus()
			}
		case enteringTime:
			switch key.String() {
			case "esc":
				m.state = selectingDate
				m.validationErr = nil
				m.timeInput.Blur()
				m.datepicker.SetFocus(datepicker.FocusCalendar)
				return m, nil
			case "enter":
				clock, err := parseClock(m.timeInput.Value())
				if err != nil {
					m.validationErr = err
					return m, nil
				}
				m.selectedClock = clock
				m.state = pickerConfirmed
				return m, tea.Quit
			}
		}
	}

	switch m.state {
	case selectingDate:
		calendar, command := m.datepicker.Update(msg)
		m.datepicker = calendar
		return m, command
	case enteringTime:
		previous := m.timeInput.Value()
		input, command := m.timeInput.Update(msg)
		m.timeInput = input
		if m.timeInput.Value() != previous {
			m.validationErr = nil
		}
		return m, command
	default:
		return m, nil
	}
}

var (
	helpStyle  = lipgloss.NewStyle().Faint(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m *pickerModel) View() string {
	calendar := m.datepicker.View()
	if m.state == selectingDate {
		return fmt.Sprintf("%s\n\n%s\n", calendar, helpStyle.Render("arrows/hjkl move • tab month/year • enter set time • q cancel"))
	}

	view := fmt.Sprintf(
		"%s\n\nTime (HH:MM or HH:MM:SS)\n%s\n",
		calendar,
		m.timeInput.View(),
	)
	if m.validationErr != nil {
		view += errorStyle.Render(m.validationErr.Error()) + "\n"
	} else if m.timeInput.Err != nil {
		view += errorStyle.Render(m.timeInput.Err.Error()) + "\n"
	}
	return view + helpStyle.Render("enter confirm • esc back to date • q cancel") + "\n"
}

func (m *pickerModel) Time() time.Time {
	date := m.datepicker.Time
	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		m.selectedClock.hour,
		m.selectedClock.minute,
		m.selectedClock.second,
		0,
		time.Local,
	)
}

func validateClockDraft(value string) error {
	if strings.Count(value, ":") > 2 {
		return errors.New("use at most two colons")
	}
	for _, char := range value {
		if char != ':' && !isASCIIDigit(char) {
			return errors.New("use only numbers and colons")
		}
	}
	return nil
}

func parseClock(value string) (clockTime, error) {
	if value == "" {
		return clockTime{}, nil
	}

	parts := strings.Split(value, ":")
	if len(parts) != 2 && len(parts) != 3 {
		return clockTime{}, errors.New("enter a time like 10:00 or 10:00:30")
	}

	hour, err := parseClockPart(parts[0], "hour", 1, 2, 23)
	if err != nil {
		return clockTime{}, err
	}
	minute, err := parseClockPart(parts[1], "minute", 2, 2, 59)
	if err != nil {
		return clockTime{}, err
	}

	second := 0
	if len(parts) == 3 {
		second, err = parseClockPart(parts[2], "second", 2, 2, 59)
		if err != nil {
			return clockTime{}, err
		}
	}

	return clockTime{hour: hour, minute: minute, second: second}, nil
}

func parseClockPart(value, name string, minLength, maxLength, maxValue int) (int, error) {
	if len(value) < minLength || len(value) > maxLength {
		if minLength != maxLength {
			return 0, fmt.Errorf("%s must use %d or %d digits", name, minLength, maxLength)
		}
		return 0, fmt.Errorf("%s must use %d digits", name, minLength)
	}
	for _, char := range value {
		if !isASCIIDigit(char) {
			return 0, fmt.Errorf("%s must be a number", name)
		}
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed > maxValue {
		return 0, fmt.Errorf("%s must be between 00 and %02d", name, maxValue)
	}
	return parsed, nil
}

func isASCIIDigit(char rune) bool {
	return char >= '0' && char <= '9'
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
