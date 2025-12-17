package window

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
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
	switch {
	case IsHangoutWindow(w, h.cfg):
		h.handleHangoutWindow(w.Address)
	case IsPictureInPictureWindow(w):
		h.handlePictureInPictureWindow(w.Address)
	}
}

func (h *ScreencastHandler) ActiveWindow(w event.ActiveWindow) {
	a := func() string {
		activeClient, err := h.Client.ActiveWindow()
		if err != nil {
			return ""
		}
		return activeClient.Address
	}

	if IsBitwardenWindow(w.Title, h.cfg) {
		addr := fmt.Sprintf("address:%s", a())
		DispatchCommands(h.Client, []string{
			"tagwindow +starship",
			fmt.Sprintf("setfloating %s", addr),
			fmt.Sprintf("resizewindowpixel exact 900 725,%s", addr),
			fmt.Sprintf("centerwindow %s", addr),
		})
	}
	if IsHangoutTitle(w.Title, h.cfg) {
		h.handleHangoutWindow(a())
	}
}

func (h *ScreencastHandler) handleHangoutWindow(address string) {
	if !h.Client.MarkHandled(address) {
		return
	}

	time.Sleep(100 * time.Millisecond)
	slog.Debug("handling hangout window", "address", address)

	position := GetMeetingPosition(h.Client, address, h.cfg)
	commands := BuildCommands(address, position)
	DispatchCommands(h.Client, commands)
}

func (h *ScreencastHandler) handlePictureInPictureWindow(address string) {
	time.Sleep(100 * time.Millisecond)
	slog.Debug("handling picture-in-picture window", "address", address)

	position := GetMeetingPosition(h.Client, address, h.cfg)
	DispatchCommands(h.Client, []string{
		fmt.Sprintf("movewindowpixel exact %s,address:%s", position, address),
	})
}
