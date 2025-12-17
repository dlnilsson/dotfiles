// hypr-screencapture: monitor Hyprland screencast events and expose status via Unix socket
// combine with waybar custom script module to show screencast status in bar
// example:
//
//	"custom/recording": {
//	    "exec": "hypr-screencapture status -subscribe",
//	    "format": "{}",
//	    "tooltip": false,
//	    "on-click": "/home/dln/.dotfiles/bin/screen-recorder"
//	},
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "embed"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/exit"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/logger"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/manager"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/notifier"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/pinner"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/pipewire"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/window"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

//go:embed camera.png
var icon []byte

var (
	cfg   *config.Config
	cfgMu sync.RWMutex
)

func reloadConfig() {
	newCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to reload config", "error", err)
		os.Exit(exit.Config)
	}

	logLevel, err := logger.ParseLogLevel(newCfg.Logging.Level)
	if err != nil {
		slog.Error("failed to parse log level during reload, using default DEBUG", "error", err, "level", newCfg.Logging.Level)
		logLevel = slog.LevelDebug
	}
	logger.Init(logLevel)

	cfgMu.Lock()
	cfg = newCfg
	cfgMu.Unlock()

	slog.Info("Config reloaded successfully", "log_level", newCfg.Logging.Level)
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintf(os.Stderr, "Do not run this program as root or with sudo\n")
		os.Exit(exit.NoPerm)
	}

	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(exit.Config)
	}

	logLevel, err := logger.ParseLogLevel(cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse log level: %v\n", err)
		os.Exit(exit.Config)
	}
	logger.Init(logLevel)

	if len(os.Args) > 1 && os.Args[1] == "status" {
		statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
		subscribe := statusCmd.Bool("subscribe", false, "continuously listen for status updates")
		statusCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s status [flags]\n\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Get the current screencast status from the Unix socket.\n\n")
			fmt.Fprintf(os.Stderr, "Flags:\n")
			statusCmd.PrintDefaults()
		}

		if err := statusCmd.Parse(os.Args[2:]); err != nil {
			statusCmd.Usage()
			os.Exit(exit.Config)
		}

		if *subscribe {
			if err := manager.RunSubscribeCommand(cfg); err != nil {
				slog.Error("subscribe command failed", "error", err)
				os.Exit(exit.Unavailable)
			}
			return
		}

		status, code, err := manager.RunStatusCommand(cfg)
		if err != nil {
			slog.Error("status command failed", "error", err)
		}
		fmt.Println(status)
		os.Exit(code)
	}

	if err := config.EnsureConfigDir(); err != nil {
		slog.Error("failed to create config directory", "error", err)
		os.Exit(exit.CantCreat)
	}

	var (
		notifierState = notifier.NewState(cfg.Notifications.Cooldown)
		socketStatus  = &manager.SocketStatus{}
		mgr           = manager.New(
			cfg,
			notifierState,
			socketStatus,
			icon,
		)

		sigCh       = make(chan os.Signal, 1)
		ch          = make(chan messages.Msg, 8)
		managerCh   = make(chan messages.Msg, 4)
		pinWindowCh = make(chan messages.Msg, 4)
		cli         = event.MustClient()
		ctx, cancel = context.WithCancel(context.Background())
		hc          = hyprland.MustClient()

		windowClient = window.NewClient(hc)
		handler      = window.NewScreencastHandler(ch, windowClient, cfg)
	)

	mgr.WriteStatus(false)

	defer func() {
		if err := cli.Close(); err != nil {
			slog.Error("failed to close client", "error", err)
		}
	}()
	defer cancel()

	cfgMu.RLock()
	socketPath := cfg.Paths.SocketPath
	cfgMu.RUnlock()

	if err := manager.CheckServerRunning(socketPath); err != nil {
		slog.Error("server already running", "error", err)
		os.Exit(exit.Unavailable)
	}

	if err := manager.StartSocketServer(ctx, socketPath, socketStatus); err != nil {
		slog.Error("could not start socket server", "error", err)
		os.Exit(exit.OSErr)
	}

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGHUP:
				reloadConfig()
			case syscall.SIGINT, syscall.SIGTERM:
				cancel()
			}
		}
	}()

	defer pipewire.StopPipeWireLoop()

	go func() {
		if ret := pipewire.StartPipeWireLoop(); ret != 0 {
			slog.Error("PipeWire loop exited", "code", ret)
		}
	}()

	pipewire.SetChannel(ch)

	go func() {
		events := []event.EventType{
			event.EventScreencast,
			event.EventActiveWindow,
			event.EventOpenWindow,
		}
		if err := cli.Subscribe(ctx, handler, events...); err != nil && ctx.Err() == nil {
			slog.Error("Subscribe exited with error", "error", err)
			cancel()
		}
	}()

	slog.Debug("Environment variables",
		"XDG_RUNTIME_DIR", os.Getenv("XDG_RUNTIME_DIR"),
		"HYPRLAND_INSTANCE_SIGNATURE", os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"))
	if clients, err := hc.Clients(); err != nil {
		slog.Error("IPC check: Clients() error", "error", err)
	} else {
		slog.Debug("IPC check: saw clients at start", "count", len(clients))
	}

	go func() {
		var (
			lastSent          = make(map[messages.Kind]time.Time)
			lastSource        = make(map[messages.Kind]messages.Source)
			rateLimitDuration = time.Second
		)
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-ch:
				slog.Debug("Fan-out received message", "kind", m.Kind, "source", m.Source)

				now := time.Now()
				if lastTime, exists := lastSent[m.Kind]; exists {
					timeSinceLastSent := now.Sub(lastTime)
					if timeSinceLastSent < rateLimitDuration {
						if lastSrc, ok := lastSource[m.Kind]; ok {
							if lastSrc == messages.SourceHyprland && m.Source == messages.SourcePipeWire {
								slog.Debug("Prioritizing PipeWire message over Hyprland", "timeSinceLastSent", timeSinceLastSent)
							} else {
								slog.Debug("Rate limiting message", "kind", m.Kind, "source", m.Source, "timeSinceLastSent", timeSinceLastSent)
								continue
							}
						} else {
							slog.Debug("Rate limiting message", "kind", m.Kind, "source", m.Source, "timeSinceLastSent", timeSinceLastSent)
							continue
						}
					}
				}

				lastSent[m.Kind] = now
				lastSource[m.Kind] = m.Source

				select {
				case managerCh <- m:
					slog.Debug("Fan-out sent message to manager", "kind", m.Kind)
				case <-ctx.Done():
					return
				default:
					slog.Debug("Fan-out failed to send to manager (channel full)", "kind", m.Kind)
				}
				select {
				case pinWindowCh <- m:
					slog.Debug("Fan-out sent message to pinWindow", "kind", m.Kind)
				case <-ctx.Done():
					return
				default:
					slog.Debug("Fan-out failed to send to pinWindow (channel full)", "kind", m.Kind)
				}
			}
		}
	}()

	go mgr.Run(ctx, managerCh)

	p := pinner.New(windowClient, cfg)
	go p.Run(ctx, pinWindowCh)

	<-ctx.Done()
	time.Sleep(50 * time.Millisecond)
}
