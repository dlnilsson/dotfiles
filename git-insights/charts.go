package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const barChar = "▇"

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("62"))
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	barStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	labelStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	countStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hashStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	subjectStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	emptyStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	searchStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	searchCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Blink(true)
)

// barChartEntry represents one row in a bar chart.
type barChartEntry struct {
	Label string
	Value int
}

// renderBarChart renders a horizontal bar chart with Unicode block characters.
func renderBarChart(entries []barChartEntry, availableWidth int) string {
	if len(entries) == 0 {
		return emptyStyle.Render("  No data found")
	}

	var (
		maxLabel = 0
		maxValue = 0
	)
	for _, e := range entries {
		maxLabel = max(maxLabel, len(e.Label))
		maxValue = max(maxValue, e.Value)
	}
	maxLabel = min(maxLabel, 40)

	var (
		countWidth  = len(strconv.Itoa(maxValue)) + 2
		barMaxWidth = availableWidth - maxLabel - countWidth - 6
		out         strings.Builder
	)

	if barMaxWidth < 1 {
		barMaxWidth = 1
	}

	out.Grow(len(entries) * (availableWidth + 1))

	for _, e := range entries {
		label := e.Label
		if len(label) > maxLabel {
			label = label[:maxLabel-1] + "~"
		}

		barLen := 0
		if maxValue > 0 && e.Value > 0 {
			barLen = max(1, (e.Value*barMaxWidth)/maxValue)
		}

		bar := strings.Repeat(barChar, barLen)
		padding := strings.Repeat(" ", barMaxWidth-barLen)

		out.WriteString("  ")
		out.WriteString(labelStyle.Width(maxLabel).Render(label))
		out.WriteString("  ")
		out.WriteString(barStyle.Render(bar))
		out.WriteString(padding)
		out.WriteString("  ")
		out.WriteString(countStyle.Render(fmt.Sprintf("%*d", countWidth-2, e.Value)))
		out.WriteByte('\n')
	}

	return out.String()
}

// renderBugClusters renders the bug clusters tab.
func renderBugClusters(clusters []BugCluster, width int) string {
	entries := make([]barChartEntry, 0, len(clusters))
	for _, c := range clusters {
		entries = append(entries, barChartEntry{Label: c.File, Value: c.Count})
	}
	return renderBarChart(entries, width)
}

// renderActivity renders the commit activity timeline tab.
func renderActivity(months []ActivityMonth, width int) string {
	entries := make([]barChartEntry, 0, len(months))
	for _, m := range months {
		entries = append(entries, barChartEntry{Label: m.Month, Value: m.Count})
	}
	return renderBarChart(entries, width)
}

// renderFirefighting renders the firefighting commits tab as a list.
func renderFirefighting(entries []FirefightEntry, width int) string {
	if len(entries) == 0 {
		return emptyStyle.Render("  No firefighting commits found in the past year")
	}

	var out strings.Builder
	out.Grow(len(entries) * 80)

	for _, e := range entries {
		out.WriteString("  ")
		out.WriteString(hashStyle.Render(e.Hash))
		out.WriteString(" ")

		maxSubject := width - len(e.Hash) - 4
		subject := e.Subject
		if maxSubject > 0 && len(subject) > maxSubject {
			subject = subject[:maxSubject-1] + "~"
		}
		out.WriteString(subjectStyle.Render(subject))
		out.WriteByte('\n')
	}

	return out.String()
}

// renderAuthors renders the contributors tab.
func renderAuthors(authors []Author, width int) string {
	entries := make([]barChartEntry, 0, len(authors))
	for _, a := range authors {
		entries = append(entries, barChartEntry{Label: a.Name, Value: a.Count})
	}
	return renderBarChart(entries, width)
}

// renderChurn renders the file churn tab.
func renderChurn(files []ChurnFile, width int) string {
	entries := make([]barChartEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, barChartEntry{Label: f.File, Value: f.Count})
	}
	return renderBarChart(entries, width)
}

// renderFrequency renders the file revision frequency tab.
func renderFrequency(files []FrequencyFile, width int) string {
	entries := make([]barChartEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, barChartEntry{Label: f.File, Value: f.Count})
	}
	return renderBarChart(entries, width)
}

// renderTabBar renders the tab navigation bar.
func renderTabBar(names []string, active int, _ int) string {
	var out strings.Builder
	out.WriteString("  ")
	for i, name := range names {
		label := fmt.Sprintf(" %d:%s ", i+1, name)
		if i == active {
			out.WriteString(activeTabStyle.Render(label))
		} else {
			out.WriteString(inactiveTabStyle.Render(label))
		}
		if i < len(names)-1 {
			out.WriteString(" ")
		}
	}
	return out.String()
}

// renderHeader renders the application header with optional scope indicator.
func renderHeader(scope string) string {
	if scope == "" {
		return titleStyle.Render("  git-insights")
	}
	return titleStyle.Render("  git-insights") + "  " + helpStyle.Render(scope+"/")
}

// renderFooter renders the keybinding help footer.
func renderFooter(_ int) string {
	return helpStyle.Render("  ←/→ or 1-6: tabs  j/k: scroll  q: quit")
}


// renderError renders an error message.
func renderError(err error) string {
	return errorStyle.Render("  Error: " + err.Error())
}
