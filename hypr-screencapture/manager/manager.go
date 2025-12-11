package manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/notifier"
)

type Manager struct {
	cfg            *config.Config
	notifier       *notifier.State
	socketStatus   *SocketStatus
	processWatcher *ProcessWatcher
	icon           []byte
}

type SocketStatus struct {
	mu     sync.RWMutex
	status string
}

func New(cfg *config.Config, notifier *notifier.State, socketStatus *SocketStatus, icon []byte) *Manager {
	return &Manager{
		cfg:            cfg,
		notifier:       notifier,
		socketStatus:   socketStatus,
		processWatcher: NewProcessWatcher(),
		icon:           icon,
	}
}

func (m *Manager) Run(ctx context.Context, ch <-chan messages.Msg) {
	var lastWritten *bool

	writeIfChanged := func(on bool) {
		if lastWritten == nil || *lastWritten != on {
			slog.Debug("Writing status", "status", onOff(on))
			m.WriteStatus(on)
			v := on
			lastWritten = &v
		}
	}

	startWatcher := func() {
		m.processWatcher.Stop()
		procNames := m.cfg.Processes.Monitor
		pollEvery := m.cfg.Processes.PollInterval
		slog.Debug("Starting new watcher", "processes", procNames)
		m.processWatcher.Start(ctx, procNames, pollEvery, func() {
			slog.Debug("Watcher onGone callback called")
			writeIfChanged(false)
		})
	}

	stopWatcher := func() {
		m.processWatcher.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			stopWatcher()
			return
		case msg := <-ch:
			slog.Debug("manager received message", "message", msg)
			switch msg.Kind {
			case messages.ScOn:
				procNames := m.cfg.Processes.Monitor
				alive := IsAnyProcessAlive(procNames)
				slog.Debug("msgScOn - checking processes", "processes", procNames, "alive", alive)
				if alive {
					slog.Debug("At least one process is alive, starting watcher")
					writeIfChanged(true)
					startWatcher()
				} else {
					slog.Debug("No processes alive, not starting watcher")
					stopWatcher()
					writeIfChanged(true)
					m.WriteStatus(true)
				}

			case messages.ScOff:
				writeIfChanged(false)
				stopWatcher()
			default:
				writeIfChanged(false)
				stopWatcher()
			}
		}
	}
}

func (m *Manager) WriteStatus(on bool) {
	message := messageOff

	if on {
		message = messageOn
	}

	defer func() {
		go func() {
			if err := notifier.ToggleNotificationInhibitor(on); err != nil {
				slog.Error("could not toggle notification inhibitor", "error", err)
			}
		}()
	}()

	isInitialOff := message == messageOff && m.notifier.Count() == 0
	if m.notifier.ShouldNotify(message, isInitialOff) {
		if err := notifier.SendNotification("Screencast", message, m.icon); err != nil {
			slog.Error("could not send notification", "error", err)
		}
	}

	statusStr := onOff(on)
	m.socketStatus.mu.Lock()
	m.socketStatus.status = statusStr
	m.socketStatus.mu.Unlock()
	slog.Debug("Updated socket status", "status", statusStr)
}

const (
	messageOn  = "screencast on"
	messageOff = "screencast off"
)

func onOff(b bool) string {
	if b {
		return "●"
	}
	return ""
}
