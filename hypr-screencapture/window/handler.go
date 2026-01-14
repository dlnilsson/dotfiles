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
	cfg    *config.Config
}

func NewScreencastHandler(ch chan<- messages.Msg, client *Client, cfg *config.Config) *ScreencastHandler {
	return &ScreencastHandler{
		ch:     ch,
		Client: client,
		cfg:    cfg,
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
	case IsHangoutWindow(w, h.cfg):
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
	slog.Debug("active Window starship", "is_starship", IsStarShipWindow(w.Title, h.cfg))
	if IsStarShipWindow(w.Title, h.cfg) {
		// The only way to reliably detect if it's a popup (starship)
		// is to rely on user behavior: the browser is the only window in the workspace.
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
		slog.Debug("active window size initial size",
			"title", w.Title,
			"size", activeWindow.Size,
			"siblings", len(siblings),
			"has", hasAnyTag(activeWindow.Tags, h.cfg.WindowMatching.ExcludeFromStarship...),
			"full", (len(siblings) > 0 && !hasAnyTag(activeWindow.Tags, h.cfg.WindowMatching.ExcludeFromStarship...)),
		)
		if len(siblings) > 0 &&
			!hasAnyTag(activeWindow.Tags, h.cfg.WindowMatching.ExcludeFromStarship...) {
			DispatchCommands(h.Client, []string{
				"denywindowfromgroup on",
				"tagwindow +starship",
				fmt.Sprintf("setfloating %s", addr),
				fmt.Sprintf("resizewindowpixel exact 900 725,%s", addr),
				fmt.Sprintf("centerwindow %s", addr),
			})
		}
	}
	if IsHangoutTitle(w.Title, h.cfg) {
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
	position := GetMeetingPosition(h.Client, address, h.cfg)
	commands := BuildCommands(address, position)

	DispatchCommands(h.Client, append([]string{cmd}, commands...))
	slog.Debug("Ok, hangout window handled")
}

func (h *ScreencastHandler) handlePictureInPictureWindow(address string) {
	time.Sleep(100 * time.Millisecond)
	slog.Debug("handling picture-in-picture window", "address", address)

	position := GetMeetingPosition(h.Client, address, h.cfg)
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
