package pinner

import (
	"context"
	"log/slog"
	"time"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/notifier"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/window"
	"github.com/thiagokokada/hyprland-go"
)

type pinConfig struct {
	initialDelay  time.Duration
	maxRetries    int
	retryDelay    time.Duration
	maxRetryDelay time.Duration
	pollInterval  time.Duration
	maxPollTime   time.Duration
}

type Pinner struct {
	*window.Client
	cfgFn func() *config.Config
}

func New(c *window.Client, cfgFn func() *config.Config) *Pinner {
	return &Pinner{
		Client: c,
		cfgFn:  cfgFn,
	}
}

func (p *Pinner) currentPinConfig() pinConfig {
	cfg := p.cfgFn()
	return pinConfig{
		initialDelay:  cfg.PinWindow.InitialDelay,
		maxRetries:    cfg.PinWindow.MaxRetries,
		retryDelay:    cfg.PinWindow.RetryDelay,
		maxRetryDelay: cfg.PinWindow.MaxRetryDelay,
		pollInterval:  cfg.PinWindow.PollInterval,
		maxPollTime:   cfg.PinWindow.MaxPollTime,
	}
}

func (p *Pinner) Run(ctx context.Context, ch <-chan messages.Msg) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("pinWindow context done")
			return
		case m := <-ch:
			slog.Debug("pinWindow received message", "message", m, "kind", m.Kind)
			switch m.Kind {
			case messages.ScOn:
				slog.Debug("screencast started, looking for windows to pin")
				go p.pinWindowsWithRetry(ctx)
			case messages.ScOff:
				if err := notifier.SendNotification("Camera", "📸 Camera off!", nil); err != nil {
					slog.Error("could not send camera notification", "error", err)
				}
			case messages.CameraOn:
				if err := notifier.SendNotification("Camera", "📸 Camera on!", nil); err != nil {
					slog.Error("could not send camera notification", "error", err)
				}
			}
		}
	}
}

func (p *Pinner) pinWindowsWithRetry(ctx context.Context) {
	pc := p.currentPinConfig()
	time.Sleep(pc.initialDelay)
	var (
		pinnedWindows   = make(map[string]bool)
		ticker          = time.NewTicker(pc.pollInterval)
		pollCtx, cancel = context.WithTimeout(ctx, pc.maxPollTime)
	)
	defer cancel()
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			if len(pinnedWindows) == 0 {
				slog.Debug("timeout reached, no suitable windows found to pin")
			}
			return
		case <-ticker.C:
			clients, err := p.Clients()
			if err != nil {
				slog.Error("failed to get clients", "error", err)
				continue
			}

			cfg := p.cfgFn()
			candidates := p.Client.FindPinCandidates(clients, pinnedWindows, cfg)
			if len(candidates) == 0 {
				continue
			}

			slog.Debug("found new candidate windows to pin", "count", len(candidates))

			for _, candidate := range candidates {
				if p.tryPinWindow(candidate) {
					pinnedWindows[candidate.Address] = true
					slog.Debug("successfully pinned window", "address", candidate.Address, "title", candidate.Title)
				}
			}

			if len(pinnedWindows) > 0 {
				slog.Debug("pinned windows total, stopping search", "count", len(pinnedWindows))
				return
			}
		}
	}
}

func (p *Pinner) tryPinWindow(win hyprland.Client) bool {
	addr := win.Address

	cfg := p.cfgFn()
	position, err := window.CalculatePosition(p.Client, win, cfg)
	if err != nil {
		slog.Warn("failed to calculate window position, using default", "error", err)
		position = cfg.Positioning.DefaultPosition
	}

	pc := p.currentPinConfig()
	for attempt := 0; attempt < pc.maxRetries; attempt++ {
		if attempt > 0 {
			delay := min(time.Duration(attempt)*pc.retryDelay, pc.maxRetryDelay)
			slog.Debug("retrying pin operation", "address", addr, "attempt", attempt+1, "maxRetries", pc.maxRetries, "delay", delay)
			time.Sleep(delay)
		}
		p.executeWindowCommands(addr, position)

		if p.verifyWindowPinned(addr) {
			return true
		}

		slog.Debug("pin command succeeded but window doesn't appear to be pinned", "address", addr)
	}
	slog.Error("failed to pin window after attempts", "address", addr, "attempts", pc.maxRetries)
	return false
}

func (p *Pinner) executeWindowCommands(address, position string) {
	commands := window.BuildCommands(address, position)
	window.DispatchCommands(p.Client, commands)
}

func (p *Pinner) verifyWindowPinned(address string) bool {
	clients, err := p.Clients()
	if err != nil {
		slog.Debug("could not verify pin status", "error", err)
		return false
	}

	for _, c := range clients {
		if c.Address == address {
			return c.Pinned
		}
	}

	slog.Debug("window not found during verification", "address", address)
	return false
}
