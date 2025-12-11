package window

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go/event"
)

func IsHangoutTitle(title string, cfg *config.Config) bool {
	prefixes := cfg.WindowMatching.MeetTitlePrefixes
	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		return strings.HasPrefix(title, prefix)
	})
}

func IsHangoutWindow(w event.OpenWindow, cfg *config.Config) bool {
	return IsHangoutTitle(w.Title, cfg)
}

func IsPictureInPictureWindow(w event.OpenWindow) bool {
	pipRegex := regexp.MustCompile(`(?i)(Picture-in-Picture|Picture in Picture)`)
	return pipRegex.MatchString(w.Title)
}

func IsBitwardenWindow(title string, cfg *config.Config) bool {
	bitwardenTitles := cfg.WindowMatching.BitwardenTitles
	return slices.ContainsFunc(bitwardenTitles, func(match string) bool {
		return title == match
	})
}
