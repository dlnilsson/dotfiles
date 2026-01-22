package window

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

type ScreencastHandler struct {
	event.DefaultEventHandler
	ch     chan<- messages.Msg
	Client *Client
	cfgFn  func() *config.Config
}

func NewScreencastHandler(ch chan<- messages.Msg, client *Client, cfgFn func() *config.Config) *ScreencastHandler {
	return &ScreencastHandler{
		ch:     ch,
		Client: client,
		cfgFn:  cfgFn,
	}
}

func (h *ScreencastHandler) Screencast(w event.Screencast) {
	slog.Info("Screencast", "sharing", w.Sharing, "owner", w.Owner)
	var m messages.Msg
	if w.Sharing {
		m = messages.Msg{Kind: messages.ScOn, Source: messages.SourceHyprland}
	} else {
		m = messages.Msg{Kind: messages.ScOff, Source: messages.SourceHyprland}
		h.Client.ClearHandled()
	}
	select {
	case h.ch <- m:
		slog.Debug("Hyprland sent message", "message", m)
	default:
		slog.Warn("Hyprland failed to send message (channel full)", "message", m)
	}
}

func (h *ScreencastHandler) OpenWindow(w event.OpenWindow) {
	addr := NormalizeAddress(w.Address)
	slog.Debug("OpenWindow", "address", addr, "title", w.Title, "class", w.Class)
	switch {
	case IsHangoutWindow(w, h.cfgFn()):
		slog.Debug("Switch OK hangout window")
		h.handleHangoutWindow(addr)
	case IsPictureInPictureWindow(w):
		h.handlePictureInPictureWindow(addr)
	}
}

func (h *ScreencastHandler) CloseWindow(w event.CloseWindow) {
	addr := NormalizeAddress(w.Address)
	h.Client.RemoveHandled(addr)
	if addr != w.Address {
		slog.Debug("RemoveHandled w.Address", "address", w.Address)
		h.Client.RemoveHandled(w.Address)
	}
}

func (h *ScreencastHandler) ActiveWindow(w event.ActiveWindow) {
	activeWindow, err := h.Client.ActiveWindow()
	if err != nil {
		return
	}
	var (
		address = activeWindow.Address
		addr    = fmt.Sprintf("address:%s", address)
	)
	cfg := h.cfgFn()
	slog.Debug("active Window starship", "is_starship", IsStarShipWindow(w.Title, cfg))
	if IsStarShipWindow(w.Title, cfg) {
		// The only way to reliably detect if it's a popup (starship)
		// is to rely on user behavior: the browser is the only window in the workspace.
		time.Sleep(100 * time.Millisecond)
		clients, err := h.Client.Clients()
		var siblings []hyprland.Client
		if err == nil {
			slog.Debug("lookup siblings", "workspace", activeWindow.Workspace.Id, "class", activeWindow.Class)
			for _, client := range clients {
				if client.Workspace.Id != activeWindow.Workspace.Id {
					continue
				}
				if client.Class == activeWindow.Class {
					continue
				}
				slog.Debug("Other client", "title", client.Title, "addr", client.Address, "size", client.Size)
				siblings = append(siblings, client)
			}
		}
		windows := 0
		workspaces, err := h.Client.Workspaces()
		if err == nil {
			for _, w := range workspaces {
				if w.Id == activeWindow.Workspace.Id {
					windows = w.Windows
				}
			}
		}
		slog.Debug("active window size initial size",
			"title", w.Title,
			"size", activeWindow.Size,
			"siblings", len(siblings),
			"workspace", activeWindow.Workspace.Id,
			"workspace_windows", windows,
			"has", hasAnyTag(activeWindow.Tags, cfg.WindowMatching.ExcludeFromStarship...),
			"full", (len(siblings) > 0 || windows > 1 && !hasAnyTag(activeWindow.Tags, cfg.WindowMatching.ExcludeFromStarship...)),
		)
		if len(siblings) >= 1 || windows > 1 &&
			!hasAnyTag(activeWindow.Tags, cfg.WindowMatching.ExcludeFromStarship...) {
			DispatchCommands(h.Client, []string{
				"denywindowfromgroup on",
				"tagwindow +starship",
				fmt.Sprintf("setfloating %s", addr),
				fmt.Sprintf("resizewindowpixel exact 900 725,%s", addr),
				fmt.Sprintf("centerwindow %s", addr),
			})
		}
	}
	if IsHangoutTitle(w.Title, cfg) {
		h.handleHangoutWindow(address)
	}
}

func (h *ScreencastHandler) handleHangoutWindow(address string) {
	if !h.Client.MarkHandled(address) {
		slog.Debug("MarkHandled failed", "address", address)
		return
	}

	time.Sleep(100 * time.Millisecond)
	slog.Debug("handling hangout window", "address", address)

	// resize before we calculate position
	cmd := fmt.Sprintf("resizewindowpixel exact 512 360,%s", fmt.Sprintf("address:%s", address))
	DispatchCommands(h.Client, []string{
		"denywindowfromgroup on",
		cmd,
	})
	cfg := h.cfgFn()
	position := GetMeetingPosition(h.Client, address, cfg)
	commands := BuildCommands(address, position)

	DispatchCommands(h.Client, append([]string{cmd}, commands...))
	slog.Debug("Ok, hangout window handled")
}

func (h *ScreencastHandler) handlePictureInPictureWindow(address string) {
	time.Sleep(100 * time.Millisecond)
	slog.Debug("handling picture-in-picture window", "address", address)

	position := GetMeetingPosition(h.Client, address, h.cfgFn())
	DispatchCommands(h.Client, []string{
		fmt.Sprintf("movewindowpixel exact %s,address:%s", position, address),
	})
}

// hasAnyTag checks if any of the given values are present in the tags slice.
func hasAnyTag(tags []string, values ...string) bool {
	for _, v := range values {
		if slices.Contains(tags, v) {
			return true
		}
	}
	return false
}

func (e *ScreencastHandler) MonitorRemoved(m event.MonitorName) {
	slog.Debug("MonitorRemoved", "monitor", m)
	// sleep if multiple monitors are removed at once. give it a second to detect
	time.Sleep(7 * time.Second)
	monitors, err := e.Client.Monitors()
	if err != nil {
		slog.Error("Failed to get monitors", "error", err)
		return
	}
	if !allMonitorsDisabled(monitors) {
		return
	}

	var (
		monitorName string
		lowestID    int
		found       bool
	)

	for _, monitor := range monitors {
		if !monitor.Disabled {
			continue
		}
		if !found || monitor.Id < lowestID {
			lowestID = monitor.Id
			monitorName = monitor.Name
			found = true
		}
	}

	if !found {
		slog.Error("no disabled monitors found")
		return
	}

	keyword := fmt.Sprintf("monitor %s,highres,auto,1", monitorName)
	slog.Warn("all monitors disabled, enabling monitor", "monitor", monitorName, "id", lowestID)
	response, err := e.Client.Keyword(keyword)
	slog.Debug(keyword, "response", response)
	if err != nil {
		slog.Error("Failed to enable monitor", "error", err)
	}
	r, err := e.Client.Reload()
	if err != nil {
		slog.Error("Failed to reload", "error", err)
	}
	slog.Debug("Reload response", "response", r)
}

func allMonitorsDisabled(monitors []hyprland.Monitor) bool {
	for _, monitor := range monitors {
		if !monitor.Disabled {
			return false
		}
	}
	return true
}
