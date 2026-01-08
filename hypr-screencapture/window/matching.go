package window

import (
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go/event"
)

func IsHangoutTitle(title string, cfg *config.Config) bool {
	prefixes := cfg.WindowMatching.MeetTitlePrefixes
	foo := slices.ContainsFunc(prefixes, func(prefix string) bool {
		return strings.HasPrefix(title, prefix)
	})
	slog.Debug("IsHangoutTitle", "title", title, "prefixes", prefixes, "result", foo)
	return foo
}

func IsHangoutWindow(w event.OpenWindow, cfg *config.Config) bool {
	return IsHangoutTitle(w.Title, cfg)
}

func IsPictureInPictureWindow(w event.OpenWindow) bool {
	pipRegex := regexp.MustCompile(`(?i)(Picture-in-Picture|Picture in Picture)`)
	return pipRegex.MatchString(w.Title)
}

func IsStarShipWindow(title string, cfg *config.Config) bool {
	starshipTitles := cfg.WindowMatching.StarshipTitles
	return slices.ContainsFunc(starshipTitles, func(match string) bool {
		return title == match
	})
}
