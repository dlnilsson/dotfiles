package window

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go"
)

// Position is a window position in global (layout) pixel coordinates. The Lua
// dispatchers take absolute pixels only, so percentages from the config are
// resolved against the target monitor before dispatching.
type Position struct {
	X, Y int
}

func CalculatePosition(c *Client, window hyprland.Client, cfg *config.Config) (Position, error) {
	monitors, err := c.Monitors()
	if err != nil {
		return Position{}, fmt.Errorf("failed to get monitors: %w", err)
	}

	var monitor *hyprland.Monitor
	for i := range monitors {
		if monitors[i].Id == window.Monitor {
			monitor = &monitors[i]
			break
		}
	}

	if monitor == nil {
		return Position{}, fmt.Errorf("monitor %d not found", window.Monitor)
	}

	var (
		windowWidth  = window.Size[0]
		monitorWidth = monitor.Width
		rightPadding = cfg.Positioning.RightPadding
		yPos         = cfg.Positioning.YPosition
	)

	return Position{
		X: monitor.X + monitorWidth - windowWidth - rightPadding,
		Y: monitor.Y + max(yPos, monitor.Height*2/100),
	}, nil
}

// DefaultPosition resolves the configured percentage based default position
// (e.g. "74% 2%") into global pixel coordinates on the given monitor, falling
// back to the focused monitor when that monitor is unknown.
func DefaultPosition(c *Client, cfg *config.Config, monitorID int) Position {
	monitors, err := c.Monitors()
	if err != nil {
		slog.Warn("Failed to get monitors while resolving default position", "error", err)
		return Position{}
	}

	var monitor *hyprland.Monitor
	for i := range monitors {
		if monitors[i].Id == monitorID {
			monitor = &monitors[i]
			break
		}
		if monitors[i].Focused && monitor == nil {
			monitor = &monitors[i]
		}
	}
	if monitor == nil {
		slog.Warn("No monitor to resolve default position against", "monitor", monitorID)
		return Position{}
	}

	xPercent, yPercent, err := parsePercentPosition(cfg.Positioning.DefaultPosition)
	if err != nil {
		slog.Warn("Invalid default position", "position", cfg.Positioning.DefaultPosition, "error", err)
		return Position{X: monitor.X, Y: monitor.Y}
	}

	return Position{
		X: monitor.X + int(xPercent/100*float64(monitor.Width)),
		Y: monitor.Y + int(yPercent/100*float64(monitor.Height)),
	}
}

// parsePercentPosition parses a "74% 2%" style position into its two
// percentage values.
func parsePercentPosition(position string) (float64, float64, error) {
	fields := strings.Fields(position)
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("expected two percentage values, got %q", position)
	}

	values := make([]float64, 2)
	for i, field := range fields {
		v, err := strconv.ParseFloat(strings.TrimSuffix(field, "%"), 64)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid percentage %q: %w", field, err)
		}
		values[i] = v
	}

	return values[0], values[1], nil
}

func GetMeetingPosition(c *Client, addr string, cfg *config.Config) Position {
	clients, err := c.Clients()
	if err != nil {
		slog.Warn("Failed to get clients during meeting position calculation", "error", err)
		return DefaultPosition(c, cfg, -1)
	}
	for _, client := range clients {
		if client.Address == addr {
			p, err := CalculatePosition(c, client, cfg)
			if err != nil {
				slog.Warn("Using default position because failed to calculate position", "address", addr, "error", err)
				return DefaultPosition(c, cfg, client.Monitor)
			}
			slog.Debug("Calculated position", "address", addr, "position", p)
			return p
		}
	}
	slog.Warn("Using default position because", "address", addr)
	return DefaultPosition(c, cfg, -1)
}
