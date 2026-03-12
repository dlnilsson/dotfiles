package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	var (
		limit   = flag.Float64("limit", 100, "maximum value represented by a full bar")
		width   = flag.Int("width", 40, "width of the progress bar in cells")
		noLabel = flag.Bool("no-label", false, "hide the percentage label")
	)

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: echo 10 | %s --limit 100\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "   or: %s --limit 100 10\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *limit <= 0 {
		exitf("limit must be greater than 0")
	}
	if *width <= 0 {
		exitf("width must be greater than 0")
	}

	value, err := readValue(flag.Args())
	if err != nil {
		exitf("%v", err)
	}

	fmt.Println(renderProgress(value, *limit, *width, *noLabel))
}

func readValue(args []string) (float64, error) {
	if len(args) > 0 {
		return parseNumber(args[0])
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return 0, fmt.Errorf("read stdin: %w", err)
	}

	input := strings.TrimSpace(string(data))
	if input == "" {
		return 0, fmt.Errorf("expected a numeric value from stdin or as the first argument")
	}

	return parseNumber(input)
}

func parseNumber(input string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value %q", input)
	}
	return value, nil
}

func renderProgress(value, limit float64, width int, noLabel bool) string {
	ratio := value / limit
	ratio = min(max(ratio, 0), 1)

	filled := min(int(math.Round(ratio*float64(width))), width)

	full := strings.Repeat("█", filled)
	empty := strings.Repeat("░", width-filled)

	var filledColor string
	switch {
	case ratio < 0.25:
		filledColor = "196" // red
	case ratio < 0.50:
		filledColor = "208" // orange
	case ratio < 0.75:
		filledColor = "220" // yellow
	default:
		filledColor = "42" // green
	}
	filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(filledColor))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	bar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		filledStyle.Render(full),
		emptyStyle.Render(empty),
	)

	if noLabel {
		return lipgloss.JoinHorizontal(
			lipgloss.Center,
			bar,
			" ",
			metaStyle.Render(fmt.Sprintf("%g/%g", value, limit)),
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelStyle.Width(7).Render(fmt.Sprintf("%5.1f%%", ratio*100)),
		" ",
		bar,
		" ",
		metaStyle.Render(fmt.Sprintf("%g/%g", value, limit)),
	)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
