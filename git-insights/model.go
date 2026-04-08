package main

import (
	"regexp"
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

	searching bool
	filter    string

	cursor      int
	selectedSHA string // set on enter in firefighting tab, triggers quit
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
		if m.searching {
			switch msg.String() {
			case "esc":
				m.searching = false
				m.filter = ""
				m.scrollOffset = 0
			case "enter":
				m.searching = false
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.scrollOffset = 0
				}
			case "ctrl+c":
				return m, tea.Quit
			default:
				if len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
					m.scrollOffset = 0
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.activeTab == int(analysisFirefighting) {
				if sha := m.selectedFirefightSHA(); sha != "" {
					m.selectedSHA = sha
					return m, tea.Quit
				}
			}
		case "/":
			m.searching = true
			m.filter = ""
			m.scrollOffset = 0
		case "left", "h":
			if m.activeTab > 0 {
				m.activeTab--
				m.scrollOffset = 0
				m.cursor = 0
			}
		case "right", "l":
			if m.activeTab < int(numAnalyses)-1 {
				m.activeTab++
				m.scrollOffset = 0
				m.cursor = 0
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % int(numAnalyses)
			m.scrollOffset = 0
			m.cursor = 0
		case "shift+tab":
			m.activeTab = (m.activeTab - 1 + int(numAnalyses)) % int(numAnalyses)
			m.scrollOffset = 0
			m.cursor = 0
		case "1", "2", "3", "4", "5", "6":
			m.activeTab = int(msg.Runes[0]-'0') - 1
			m.scrollOffset = 0
			m.cursor = 0
		case "j", "down":
			if m.activeTab == int(analysisFirefighting) {
				m.cursor = min(m.cursor+1, m.firefightCount()-1)
				// Auto-scroll to keep cursor visible
				if m.cursor >= m.scrollOffset+m.visibleRows {
					m.scrollOffset = m.cursor - m.visibleRows + 1
				}
			} else if ms := m.maxScroll(); m.scrollOffset < ms {
				m.scrollOffset++
			}
		case "k", "up":
			if m.activeTab == int(analysisFirefighting) {
				m.cursor = max(0, m.cursor-1)
				if m.cursor < m.scrollOffset {
					m.scrollOffset = m.cursor
				}
			} else if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "g", "home":
			m.scrollOffset = 0
			m.cursor = 0
		case "G", "end":
			m.scrollOffset = m.maxScroll()
			if m.activeTab == int(analysisFirefighting) {
				m.cursor = m.firefightCount() - 1
			}
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

	// Search bar or footer
	out.WriteByte('\n')
	if m.searching {
		out.WriteString(searchStyle.Render("  /"+m.filter) + searchCursorStyle.Render("_"))
	} else if m.filter != "" {
		out.WriteString(searchStyle.Render("  filter: "+m.filter) + helpStyle.Render("  (/ to edit, esc to clear)"))
	} else {
		out.WriteString(renderFooter(m.activeTab))
	}

	return out.String()
}

// renderAnalysis dispatches to the correct renderer based on analysis ID,
// applying the current filter if set.
func (m model) renderAnalysis(r *resultMsg) string {
	switch r.id {
	case analysisBugClusters:
		return renderBugClusters(filterBy(r.bugClusters, m.filter, func(e BugCluster) string { return e.File }), m.width)
	case analysisActivity:
		return renderActivity(filterBy(r.activity, m.filter, func(e ActivityMonth) string { return e.Month }), m.width)
	case analysisFirefighting:
		return renderFirefighting(filterBy(r.firefighting, m.filter, func(e FirefightEntry) string { return e.Hash + " " + e.Subject }), m.cursor, m.width)
	case analysisAuthors:
		return renderAuthors(filterBy(r.authors, m.filter, func(e Author) string { return e.Name }), m.width)
	case analysisChurn:
		return renderChurn(filterBy(r.churn, m.filter, func(e ChurnFile) string { return e.File }), m.width)
	case analysisFrequency:
		return renderFrequency(filterBy(r.frequency, m.filter, func(e FrequencyFile) string { return e.File }), m.width)
	default:
		return ""
	}
}

// filterBy returns items whose label matches the query regex (case-insensitive).
// Falls back to substring match if the regex is invalid.
// Returns the original slice when query is empty.
func filterBy[T any](items []T, query string, label func(T) string) []T {
	if query == "" {
		return items
	}
	re, err := regexp.Compile("(?i)" + query)
	result := make([]T, 0, len(items))
	for _, item := range items {
		text := label(item)
		if err != nil {
			// Invalid regex, fall back to substring
			if strings.Contains(strings.ToLower(text), query) {
				result = append(result, item)
			}
		} else if re.MatchString(text) {
			result = append(result, item)
		}
	}
	return result
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

// filteredFirefighting returns the current filtered firefighting entries.
func (m model) filteredFirefighting() []FirefightEntry {
	r := m.results[analysisFirefighting]
	if r == nil || r.err != nil {
		return nil
	}
	return filterBy(r.firefighting, m.filter, func(e FirefightEntry) string { return e.Hash + " " + e.Subject })
}

// firefightCount returns the number of visible firefighting entries.
func (m model) firefightCount() int {
	return max(1, len(m.filteredFirefighting()))
}

// selectedFirefightSHA returns the SHA of the currently highlighted entry.
func (m model) selectedFirefightSHA() string {
	entries := m.filteredFirefighting()
	if m.cursor >= 0 && m.cursor < len(entries) {
		return entries[m.cursor].Hash
	}
	return ""
}
