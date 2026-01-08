package window

import (
	"fmt"
	"log/slog"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go"
)

func CalculatePosition(c *Client, window hyprland.Client, cfg *config.Config) (string, error) {
	monitors, err := c.Monitors()
	if err != nil {
		return "", fmt.Errorf("failed to get monitors: %w", err)
	}

	var monitor *hyprland.Monitor
	for i := range monitors {
		if monitors[i].Id == window.Monitor {
			monitor = &monitors[i]
			break
		}
	}

	if monitor == nil {
		return "", fmt.Errorf("monitor %d not found", window.Monitor)
	}

	var (
		windowWidth  = window.Size[0]
		monitorWidth = monitor.Width
		rightPadding = cfg.Positioning.RightPadding
		yPos         = cfg.Positioning.YPosition
	)

	xPos := monitorWidth - windowWidth - rightPadding
	xPercent := int(float64(xPos) / float64(monitorWidth) * 100)
	yPercent := max(int(float64(yPos)/float64(monitor.Height)*100), 2)

	return fmt.Sprintf("%d%% %d%%", xPercent, yPercent), nil
}

func GetMeetingPosition(c *Client, addr string, cfg *config.Config) string {
	clients, err := c.Clients()
	if err != nil {
		return cfg.Positioning.DefaultPosition
	}
	for _, client := range clients {
		if client.Address == addr {
			p, err := CalculatePosition(c, client, cfg)
			if err != nil {
				slog.Warn("Using default position because failed to calculate position", "address", addr, "error", err)
				return cfg.Positioning.DefaultPosition
			}
			slog.Debug("Calculated position", "address", addr, "position", p)
			return p
		}
	}
	slog.Warn("Using default position because", "address", addr)
	return cfg.Positioning.DefaultPosition
}
