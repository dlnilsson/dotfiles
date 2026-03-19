package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type statusInput struct {
	SessionID      string         `json:"session_id,omitempty,omitzero"`
	TranscriptPath string         `json:"transcript_path,omitempty,omitzero"`
	CWD            string         `json:"cwd,omitempty,omitzero"`
	Model          statusModel    `json:"model,omitzero"`
	Workspace      statusWS       `json:"workspace,omitzero"`
	Version        string         `json:"version,omitempty,omitzero"`
	OutputStyle    statusOutStyle `json:"output_style,omitzero"`
	ContextWindow  statusContext  `json:"context_window,omitzero"`
	Exceeds200K    bool           `json:"exceeds_200k_tokens,omitempty,omitzero"`
	Cost           statusCost     `json:"cost,omitzero"`
}

type statusModel struct {
	ID          string `json:"id,omitempty,omitzero"`
	DisplayName string `json:"display_name,omitempty,omitzero"`
}

type statusWS struct {
	CurrentDir string   `json:"current_dir,omitempty,omitzero"`
	ProjectDir string   `json:"project_dir,omitempty,omitzero"`
	AddedDirs  []string `json:"added_dirs,omitempty,omitzero"`
}

type statusOutStyle struct {
	Name string `json:"name,omitempty,omitzero"`
}

type statusContext struct {
	TotalInputTokens    int         `json:"total_input_tokens,omitempty,omitzero"`
	TotalOutputTokens   int         `json:"total_output_tokens,omitempty,omitzero"`
	ContextWindowSize   int         `json:"context_window_size,omitempty,omitzero"`
	CurrentUsage        statusUsage `json:"current_usage,omitzero"`
	UsedPercentage      *float64    `json:"used_percentage,omitempty,omitzero"`
	RemainingPercentage *float64    `json:"remaining_percentage,omitempty,omitzero"`
}

type statusUsage struct {
	InputTokens              int `json:"input_tokens,omitempty,omitzero"`
	OutputTokens             int `json:"output_tokens,omitempty,omitzero"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty,omitzero"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty,omitzero"`
}

type statusCost struct {
	TotalCostUSD *float64 `json:"total_cost_usd,omitempty,omitzero"`
}

var (
	greenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	blueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	whiteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	boldStyle  = lipgloss.NewStyle().Bold(true)
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}

	if err := forwardToStatusLog(input); err != nil {
		fmt.Fprintf(os.Stderr, "claude-statusline: forward to status log: %v\n", err)
	}

	var status statusInput
	if err := json.Unmarshal(input, &status); err != nil {
		fmt.Fprintf(os.Stderr, "claude-statusline: unmarshal status: %v\n", err)
		os.Exit(1)
	}

	var (
		cwd       = status.Workspace.CurrentDir
		model     = status.Model.DisplayName
		context   = status.ContextWindow.RemainingPercentage
		cost      = status.Cost.TotalCostUSD
		sessionID = status.SessionID
	)

	if err := updateAgentRC(cwd, sessionID); err != nil {
		fmt.Fprintf(os.Stderr, "claude-statusline: update agentrc: %v\n", err)
	}

	displayCWD := shortenHome(cwd)
	title := fmt.Sprintf("Claude %s using %s", displayCWD, model)

	var out strings.Builder
	out.WriteString(green(displayCWD))
	out.WriteByte(' ')

	if branch, dirty, ok := gitStatus(cwd); ok && branch != "" {
		indicator := green("✔")
		if dirty {
			indicator = red("✗")
		}
		out.WriteString(white("on "))
		out.WriteString(blue(branch))
		out.WriteByte(' ')
		out.WriteString(indicator)
		out.WriteByte(' ')
		titleCWD := displayCWD
		if cut, ok := strings.CutSuffix(titleCWD, "."+branch); ok {
			titleCWD = cut
		}
		title = fmt.Sprintf("Claude %s on %s %s", titleCWD, branch, model)
	}

	if err := setKittyTitle(title); err != nil {
		fmt.Fprintf(os.Stderr, "claude-statusline: set kitty title: %v\n", err)
	}

	out.WriteString(white("| "))
	out.WriteString(blue(model))

	if context != nil {
		out.WriteByte(' ')
		out.WriteString(white("| "))
		out.WriteString(renderProgress(*context, 100, 20, true))
	}

	if cost != nil {
		out.WriteByte(' ')
		out.WriteString(white("| "))
		out.WriteString(green(fmt.Sprintf("$%.3f USD", *cost)))
	}

	fmt.Print(out.String())
}

func forwardToStatusLog(input []byte) error {
	cmd := exec.Command("claude-tally")
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

const kittyTitleMinInterval = 5 * time.Minute

func setKittyTitle(title string) error {
	listenOn := os.Getenv("KITTY_LISTEN_ON")
	if listenOn == "" {
		return nil
	}

	stateDir := os.TempDir()
	stateFile := filepath.Join(stateDir, fmt.Sprintf("claude-statusline-title-%d", os.Getppid()))

	prev, _ := os.ReadFile(stateFile)
	if string(prev) == title {
		if info, err := os.Stat(stateFile); err == nil {
			if time.Since(info.ModTime()) < kittyTitleMinInterval {
				return nil
			}
		}
	}

	cmd := exec.Command("kitty", "@", "--to", listenOn, "set-window-title", title)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return err
	}

	os.WriteFile(stateFile, []byte(title), 0600)
	return nil
}

func updateAgentRC(cwd, sessionID string) error {
	if cwd == "" {
		return nil
	}

	agentRCPath := filepath.Join(cwd, ".agentrc")
	file, err := os.OpenFile(agentRCPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", agentRCPath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read %s: %w", agentRCPath, err)
	}

	content := strings.TrimRight(string(data), "\n")
	lines := []string{}
	if content != "" {
		lines = strings.Split(content, "\n")
	}
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "export CLAUDE_SESSION_ID=") {
			lines[i] = "export CLAUDE_SESSION_ID=" + sessionID
			replaced = true
		}
	}
	if !replaced {
		lines = append(lines, "export CLAUDE_SESSION_ID="+sessionID)
	}

	if err := os.WriteFile(agentRCPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		return fmt.Errorf("write %s: %w", agentRCPath, err)
	}
	return nil
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func gitStatus(dir string) (branch string, dirty bool, ok bool) {
	if dir == "" {
		return "", false, false
	}

	if err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run(); err != nil {
		return "", false, false
	}

	branchOut, err := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	if err != nil {
		return "", false, true
	}
	branch = strings.TrimSpace(string(branchOut))
	if branch == "" {
		return "", false, true
	}

	diffCmd := exec.Command("git", "-C", dir, "diff-index", "--quiet", "HEAD", "--")
	dirty = diffCmd.Run() != nil
	return branch, dirty, true
}

func renderProgress(value, limit float64, width int, noLabel bool) string {
	ratio := value / limit
	ratio = min(max(ratio, 0), 1)

	filled := minInt(int(math.Round(ratio*float64(width))), width)
	full := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)

	filledColor := "42"
	switch {
	case ratio < 0.25:
		filledColor = "196"
	case ratio < 0.50:
		filledColor = "208"
	case ratio < 0.75:
		filledColor = "220"
	}

	bar := color(filledColor, full) + color("238", empty)
	if noLabel {
		return bar + " " + color("245", fmt.Sprintf("%g/%g", value, limit))
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		boldStyle.Render(fmt.Sprintf("%5.1f%%", ratio*100)),
		" ",
		bar,
		" ",
		color("245", fmt.Sprintf("%g/%g", value, limit)),
	)
}

func color(code, text string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(code)).Render(text)
}

func green(text string) string { return greenStyle.Render(text) }
func red(text string) string   { return redStyle.Render(text) }
func blue(text string) string  { return blueStyle.Render(text) }
func white(text string) string { return whiteStyle.Render(text) }

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
