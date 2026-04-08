package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// analysisID identifies which analysis completed.
type analysisID int

const (
	analysisBugClusters analysisID = iota
	analysisActivity
	analysisFirefighting
	analysisAuthors
	analysisChurn
	analysisFrequency
	numAnalyses
)

// resultMsg carries the result of a completed git analysis.
type resultMsg struct {
	id  analysisID
	err error

	bugClusters  []BugCluster
	activity     []ActivityMonth
	firefighting []FirefightEntry
	authors      []Author
	churn        []ChurnFile
	frequency    []FrequencyFile
}

type model struct {
	activeTab int
	tabNames  [numAnalyses]string
	scope     string // subdirectory scope (empty = repo root)
	spinner   spinner.Model

	results [numAnalyses]*resultMsg
	loading [numAnalyses]bool

	scrollOffset int
	visibleRows  int

	width  int
	height int
}

func newModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	return model{
		tabNames: [numAnalyses]string{
			"Bugs",
			"Activity",
			"Firefighting",
			"Authors",
			"Churn",
			"Frequency",
		},
		loading: [numAnalyses]bool{true, true, true, true, true, true},
		scope:   repoScope(),
		spinner: s,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchBugClusters,
		fetchActivity,
		fetchFirefighting,
		fetchAuthors,
		fetchChurn,
		fetchFrequency,
	)
}

func fetchBugClusters() tea.Msg {
	clusters, err := parseBugClusters()
	return resultMsg{id: analysisBugClusters, err: err, bugClusters: clusters}
}

func fetchActivity() tea.Msg {
	months, err := parseActivity()
	return resultMsg{id: analysisActivity, err: err, activity: months}
}

func fetchFirefighting() tea.Msg {
	entries, err := parseFirefighting()
	return resultMsg{id: analysisFirefighting, err: err, firefighting: entries}
}

func fetchAuthors() tea.Msg {
	authors, err := parseAuthors()
	return resultMsg{id: analysisAuthors, err: err, authors: authors}
}

func fetchChurn() tea.Msg {
	files, err := parseChurn()
	return resultMsg{id: analysisChurn, err: err, churn: files}
}

func fetchFrequency() tea.Msg {
	files, err := parseFrequency()
	return resultMsg{id: analysisFrequency, err: err, frequency: files}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for header, tab bar, blank line, and footer
		m.visibleRows = max(1, msg.Height-5)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "left", "h":
			if m.activeTab > 0 {
				m.activeTab--
				m.scrollOffset = 0
			}
		case "right", "l":
			if m.activeTab < int(numAnalyses)-1 {
				m.activeTab++
				m.scrollOffset = 0
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % int(numAnalyses)
			m.scrollOffset = 0
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + int(numAnalyses)) % int(numAnalyses)
			m.scrollOffset = 0
		case "1", "2", "3", "4", "5", "6":
			m.activeTab = int(msg.Runes[0]-'0') - 1
			m.scrollOffset = 0
		case "j", "down":
			if ms := m.maxScroll(); m.scrollOffset < ms {
				m.scrollOffset++
			}
		case "k", "up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "g", "home":
			m.scrollOffset = 0
		case "G", "end":
			m.scrollOffset = m.maxScroll()
		}
		return m, nil

	case resultMsg:
		m.results[msg.id] = &msg
		m.loading[msg.id] = false
		return m, nil

	case spinner.TickMsg:
		// Only keep spinning if something is still loading
		anyLoading := false
		for _, l := range m.loading {
			if l {
				anyLoading = true
				break
			}
		}
		if anyLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	var out strings.Builder

	// Header
	out.WriteString(renderHeader(m.scope))
	out.WriteByte('\n')

	// Tab bar
	out.WriteString(renderTabBar(m.tabNames[:], m.activeTab, m.width))
	out.WriteByte('\n')
	out.WriteByte('\n')

	// Content area
	if m.loading[m.activeTab] {
		out.WriteString("  " + m.spinner.View() + " Loading...")
	} else if r := m.results[m.activeTab]; r != nil {
		if r.err != nil {
			out.WriteString(renderError(r.err))
		} else {
			content := m.renderAnalysis(r)
			out.WriteString(m.applyScroll(content))
		}
	}

	// Footer
	out.WriteByte('\n')
	out.WriteString(renderFooter(m.width))

	return out.String()
}

// renderAnalysis dispatches to the correct renderer based on analysis ID.
func (m model) renderAnalysis(r *resultMsg) string {
	switch r.id {
	case analysisBugClusters:
		return renderBugClusters(r.bugClusters, m.width)
	case analysisActivity:
		return renderActivity(r.activity, m.width)
	case analysisFirefighting:
		return renderFirefighting(r.firefighting, m.width)
	case analysisAuthors:
		return renderAuthors(r.authors, m.width)
	case analysisChurn:
		return renderChurn(r.churn, m.width)
	case analysisFrequency:
		return renderFrequency(r.frequency, m.width)
	default:
		return ""
	}
}

// applyScroll returns the visible window of content lines.
func (m model) applyScroll(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) <= m.visibleRows {
		return content
	}

	var (
		start = min(m.scrollOffset, max(0, len(lines)-m.visibleRows))
		end   = min(start+m.visibleRows, len(lines))
	)
	return strings.Join(lines[start:end], "\n")
}

// maxScroll returns the maximum scroll offset for the active tab's content.
func (m model) maxScroll() int {
	r := m.results[m.activeTab]
	if r == nil || r.err != nil {
		return 0
	}

	content := m.renderAnalysis(r)
	lines := strings.Split(content, "\n")
	return max(0, len(lines)-m.visibleRows)
}
