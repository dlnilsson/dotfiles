package main

import (
	"strings"
	"testing"
)

func TestRenderBarChart(t *testing.T) {
	t.Parallel()

	t.Run("scales bars proportionally", func(t *testing.T) {
		t.Parallel()
		var entries = []barChartEntry{
			{Label: "foo.go", Value: 10},
			{Label: "bar.go", Value: 5},
			{Label: "baz.go", Value: 1},
		}

		result := renderBarChart(entries, 60)

		lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}

		// First entry should have the longest bar
		firstBarCount := strings.Count(lines[0], barChar)
		secondBarCount := strings.Count(lines[1], barChar)
		thirdBarCount := strings.Count(lines[2], barChar)

		if firstBarCount <= secondBarCount {
			t.Fatalf("expected first bar (%d) > second bar (%d)", firstBarCount, secondBarCount)
		}
		if secondBarCount <= thirdBarCount {
			t.Fatalf("expected second bar (%d) > third bar (%d)", secondBarCount, thirdBarCount)
		}
	})

	t.Run("empty entries", func(t *testing.T) {
		t.Parallel()
		result := renderBarChart(nil, 80)
		if !strings.Contains(result, "No data") {
			t.Fatalf("expected 'No data' message, got %q", result)
		}
	})

	t.Run("single entry gets a bar", func(t *testing.T) {
		t.Parallel()
		var entries = []barChartEntry{
			{Label: "only.go", Value: 42},
		}

		result := renderBarChart(entries, 60)

		if !strings.Contains(result, barChar) {
			t.Fatalf("expected bar character in output, got %q", result)
		}
		if !strings.Contains(result, "42") {
			t.Fatalf("expected count 42 in output, got %q", result)
		}
	})

	t.Run("truncates long labels", func(t *testing.T) {
		t.Parallel()
		var entries = []barChartEntry{
			{Label: "this/is/a/very/long/path/to/some/deeply/nested/file.go", Value: 5},
		}

		result := renderBarChart(entries, 80)

		if !strings.Contains(result, "~") {
			t.Fatalf("expected truncated label with ~, got %q", result)
		}
	})

	t.Run("handles narrow width gracefully", func(t *testing.T) {
		t.Parallel()
		var entries = []barChartEntry{
			{Label: "foo.go", Value: 10},
		}

		// Should not panic with very narrow width
		result := renderBarChart(entries, 20)
		if result == "" {
			t.Fatal("expected non-empty output")
		}
	})
}

func TestRenderFirefighting(t *testing.T) {
	t.Parallel()

	t.Run("renders commit list", func(t *testing.T) {
		t.Parallel()
		var entries = []FirefightEntry{
			{Hash: "abc1234", Subject: "Revert broken auth"},
			{Hash: "def5678", Subject: "HOTFIX: patch crash"},
		}

		result := renderFirefighting(entries, 80)

		if !strings.Contains(result, "abc1234") {
			t.Fatalf("expected hash abc1234 in output, got %q", result)
		}
		if !strings.Contains(result, "Revert broken auth") {
			t.Fatalf("expected subject in output, got %q", result)
		}
	})

	t.Run("empty entries", func(t *testing.T) {
		t.Parallel()
		result := renderFirefighting(nil, 80)
		if !strings.Contains(result, "No firefighting") {
			t.Fatalf("expected empty message, got %q", result)
		}
	})
}
