package config

import (
	"embed"
	"os"
	"slices"
	"testing"
	"time"
)

//go:embed testdata/example.conf
var testdata embed.FS

func writeTempConfig(t *testing.T, content []byte) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "config-*.ini")
	if err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	if _, err := file.Write(content); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("failed to rewind temp config: %v", err)
	}
	return file
}

func TestParseConfigSnippet(t *testing.T) {
	cfg := Default()
	content, err := testdata.ReadFile("testdata/example.conf")
	if err != nil {
		t.Fatalf("failed to read testdata config: %v", err)
	}
	file := writeTempConfig(t, content)
	parser := newINIParser(file)

	if err := parser.parse(cfg); err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if cfg.Logging.Level != "DEBUG" {
		t.Fatalf("unexpected log level: %q", cfg.Logging.Level)
	}
	if cfg.Processes.PollInterval != 2*time.Second {
		t.Fatalf("unexpected poll interval: %v", cfg.Processes.PollInterval)
	}
	if cfg.Notifications.Cooldown != 10*time.Second {
		t.Fatalf("unexpected cooldown: %v", cfg.Notifications.Cooldown)
	}
	if cfg.PinWindow.InitialDelay != 500*time.Millisecond {
		t.Fatalf("unexpected initial delay: %v", cfg.PinWindow.InitialDelay)
	}
	if cfg.PinWindow.MaxRetries != 3 {
		t.Fatalf("unexpected max retries: %v", cfg.PinWindow.MaxRetries)
	}
	if cfg.PinWindow.RetryDelay != 500*time.Millisecond {
		t.Fatalf("unexpected retry delay: %v", cfg.PinWindow.RetryDelay)
	}
	if cfg.PinWindow.MaxRetryDelay != 2*time.Second {
		t.Fatalf("unexpected max retry delay: %v", cfg.PinWindow.MaxRetryDelay)
	}
	if cfg.PinWindow.PollInterval != 500*time.Millisecond {
		t.Fatalf("unexpected poll interval: %v", cfg.PinWindow.PollInterval)
	}
	if cfg.PinWindow.MaxPollTime != 10*time.Second {
		t.Fatalf("unexpected max poll time: %v", cfg.PinWindow.MaxPollTime)
	}
	if cfg.Positioning.DefaultPosition != "74% 2%" {
		t.Fatalf("unexpected default position: %q", cfg.Positioning.DefaultPosition)
	}
	if cfg.Positioning.RightPadding != 2 {
		t.Fatalf("unexpected right padding: %v", cfg.Positioning.RightPadding)
	}
	if cfg.Positioning.YPosition != 20 {
		t.Fatalf("unexpected y position: %v", cfg.Positioning.YPosition)
	}

	expectedStarshipTitles := []string{
		"Extension: (Bitwarden Password Manager) - Bitwarden — Zen Browser",
		"Extension: (Bitwarden Password Manager) - Bitwarden — Mozilla Firefox",
		"Extension: (Bitwarden Password Manager) - — Zen Browser",
		"Bitwarden",
		"Sign in – Google accounts",
		"Sign in – Google Accounts",
		"Sign in – Google accounts — Zen Browser",
		"Sign in - Google Accounts — Zen Browser",
		"Sign in - Google Accounts - Chromium",
		"Sign in – Google accounts - Chromium",
		"Log into Facebook — Zen Browser",
		"Log into Facebook — Mozilla Firefox",
		"Zed — Settings",
	}
	slices.Sort(expectedStarshipTitles)
	if !slices.Equal(cfg.WindowMatching.StarshipTitles, expectedStarshipTitles) {
		t.Fatalf("unexpected starship titles: %#v", cfg.WindowMatching.StarshipTitles)
	}

	expectedMeetPrefixes := []string{
		"https://meet.google.com - Meet –",
		"Meet –",
	}
	slices.Sort(expectedMeetPrefixes)
	if !slices.Equal(cfg.WindowMatching.MeetTitlePrefixes, expectedMeetPrefixes) {
		t.Fatalf("unexpected meet prefixes: %#v", cfg.WindowMatching.MeetTitlePrefixes)
	}

	expectedKeywords := []string{
		"sharing indicator",
		"sharing",
		"screen",
		"recording",
		"capture",
		"cast",
		"obs",
		"streamlabs",
		"discord",
		"zoom",
		"teams",
		"meet",
		"chrome",
		"firefox",
		"browser",
		"portal",
		"slack",
		"zen-browser",
	}
	slices.Sort(expectedKeywords)
	if !slices.Equal(cfg.WindowMatching.ScreencastKeywords, expectedKeywords) {
		t.Fatalf("unexpected screencast keywords: %#v", cfg.WindowMatching.ScreencastKeywords)
	}
}
