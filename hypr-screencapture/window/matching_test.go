package window

import (
	"testing"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go/event"
)

func TestIsPictureInPictureWindow(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  bool
	}{
		{
			name:  "matches hyphenated title",
			title: "Picture-in-Picture",
			want:  true,
		},
		{
			name:  "matches spaced title",
			title: "Picture in Picture",
			want:  true,
		},
		{
			name:  "matches with additional text",
			title: "YouTube - Picture-in-Picture",
			want:  true,
		},
		{
			name:  "matches case insensitive",
			title: "picture in picture",
			want:  true,
		},
		{
			name:  "does not match unrelated title",
			title: "Settings",
			want:  false,
		},
		{
			name:  "does not match empty title",
			title: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				w    = event.OpenWindow{Title: tt.title}
				got  = IsPictureInPictureWindow(w)
				want = tt.want
			)

			if got != want {
				t.Fatalf("IsPictureInPictureWindow(%q) = %t, want %t", tt.title, got, want)
			}
		})
	}
}

func TestIsStarShipWindow(t *testing.T) {
	var cfg = &config.Config{
		WindowMatching: config.WindowMatchingConfig{
			StarshipTitles: []string{
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
			},
		},
	}

	tests := []struct {
		name  string
		title string
		want  bool
	}{
		{
			name:  "matches exact title",
			title: "Bitwarden",
			want:  true,
		},
		{
			name:  "matches case insensitive",
			title: "log into facebook — zen browser",
			want:  true,
		},
		{
			name:  "matches another configured title",
			title: "Extension: (Bitwarden Password Manager) - Bitwarden — Mozilla Firefox",
			want:  true,
		},
		{
			name:  "does not match partial title",
			title: "Sign in – Google",
			want:  false,
		},
		{
			name:  "does not match unrelated title",
			title: "Settings",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				got  = IsStarShipWindow(tt.title, cfg)
				want = tt.want
			)

			if got != want {
				t.Fatalf("IsStarShipWindow(%q) = %t, want %t", tt.title, got, want)
			}
		})
	}
}
