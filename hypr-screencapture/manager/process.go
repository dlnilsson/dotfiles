package manager

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

func IsAnyProcessAlive(names []string) bool {
	return slices.ContainsFunc(names, isProcessAliveByName)
}

func isProcessAliveByName(name string) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		n := e.Name()
		if len(n) == 0 || n[0] < '0' || n[0] > '9' {
			continue
		}

		comm, err := os.ReadFile(filepath.Join("/proc", n, "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) != name {
			continue
		}

		if isZombie(n) {
			continue
		}
		return true
	}
	return false
}

func isZombie(pid string) bool {
	b, err := os.ReadFile(filepath.Join("/proc", pid, "status"))
	if err != nil {
		return false
	}
	for line := range bytes.SplitSeq(b, []byte{'\n'}) {
		if bytes.HasPrefix(line, []byte("State:")) {
			parts := bytes.SplitN(line, []byte{'\t'}, 2)
			if len(parts) == 2 && len(parts[1]) > 0 {
				ch := parts[1][0]
				return ch == 'Z' || ch == 'X'
			}
		}
	}
	return false
}

type ProcessWatcher struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewProcessWatcher() *ProcessWatcher {
	return &ProcessWatcher{}
}

func (pw *ProcessWatcher) Start(ctx context.Context, names []string, every time.Duration, onGone func()) {
	pw.mu.Lock()
	if pw.cancel != nil {
		pw.cancel()
		pw.cancel = nil
	}
	wctx, cancel := context.WithCancel(ctx)
	pw.cancel = cancel
	pw.mu.Unlock()

	go watchUntilGone(wctx, names, every, onGone)
}

func (pw *ProcessWatcher) Stop() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.cancel != nil {
		slog.Debug("Stopping watcher")
		pw.cancel()
		pw.cancel = nil
	}
}

func watchUntilGone(ctx context.Context, names []string, every time.Duration, onGone func()) {
	if alive := IsAnyProcessAlive(names); !alive {
		slog.Debug("Watcher immediate check - no processes alive, calling onGone", "processes", names)
		onGone()
		return
	}
	slog.Debug("Watcher started - at least one process is alive", "processes", names)

	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if alive := IsAnyProcessAlive(names); !alive {
				slog.Debug("Watcher polling check - no processes alive, calling onGone", "processes", names)
				onGone()
				return
			}
		}
	}
}
