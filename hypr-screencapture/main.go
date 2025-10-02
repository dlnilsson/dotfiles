// hypr-screencapture: monitor Hyprland screencast events and write status to file
// combine with waybar custom script module to show screencast status in bar
// example:
//
//	"custom/recording": {
//	    "exec": "cat $XDG_RUNTIME_DIR/hypr/screencast.status",
//	    "interval": 3,
//	    "format": "{}",
//	    "tooltip": false,
//	    "on-click": "/home/dln/.dotfiles/bin/screen-recorder"
//	},
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	_ "embed"

	"github.com/thiagokokada/hyprland-go/event"
)

var (
	statusFile = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", "screencast.status")
	procNames  = []string{
		"wl-screenrec",
	}
	pollEvery         = 2 * time.Second
	notificationCount = 0
)

//go:embed camera.png
var icon []byte

func sendNotification(title, message string, iconBytes []byte) error {
	cmd, err := exec.LookPath("notify-send")
	if err != nil {
		return err
	}

	tmp, err := bytesToFilename(iconBytes)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	args := []string{title, message, "-a", "hypr-screencapture", "-i", tmp, "-t", "3000", "-u", "normal"}
	c := exec.Command(cmd, args...)
	return c.Run()
}

func bytesToFilename(data []byte) (string, error) {
	var out string

	tmp, err := os.CreateTemp(os.TempDir(), "hypr-screencapture*.png")
	if err != nil {
		return out, err
	}
	defer tmp.Close()

	if _, err = tmp.Write(data); err != nil {
		return out, err
	}

	out = tmp.Name()
	return out, nil
}

type msgKind int

const (
	msgScOn msgKind = iota
	msgScOff
)

type msg struct{ kind msgKind }

type screencastHandler struct {
	event.DefaultEventHandler
	ch chan<- msg
}

func (h *screencastHandler) Screencast(w event.Screencast) {
	log.Printf("Screencast %v --- %v\n", w.Sharing, w.Owner)
	if w.Sharing {
		h.ch <- msg{kind: msgScOn}
		return
	}
	h.ch <- msg{kind: msgScOff}
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run this program as root or with sudo")
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(statusFile), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "could not create directory for status file: %v", err)
		os.Exit(1)
	}
	writeStatus(false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	// Subscribe to Hyprland screencast events
	cli := event.MustClient()
	defer cli.Close()

	ch := make(chan msg, 16)
	go func() {
		if err := cli.Subscribe(ctx, &screencastHandler{ch: ch}, event.EventScreencast); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "Subscribe exited with error: %v", err)
			cancel()
		}
	}()

	// Manager owns the state machine and file writes
	go manager(ctx, ch)

	<-ctx.Done()
	// allow final rename to flush
	time.Sleep(50 * time.Millisecond)
}

func manager(ctx context.Context, ch <-chan msg) {
	var (
		lastWritten *bool              // last ON/OFF written
		watchCancel context.CancelFunc // cancel current watcher
	)

	writeIfChanged := func(on bool) {
		if lastWritten == nil || *lastWritten != on {
			log.Printf("DEBUG: Writing status: %s\n", onOff(on))
			writeStatus(on)
			v := on
			lastWritten = &v
		}
	}

	startWatcher := func() {
		// stop any existing watcher first
		if watchCancel != nil {
			log.Printf("DEBUG: Stopping existing watcher before starting new one")
			watchCancel()
		}
		log.Printf("DEBUG: Starting new watcher for %v", procNames)
		wctx, cancel := context.WithCancel(ctx)
		watchCancel = cancel
		go watchUntilGone(wctx, procNames, pollEvery, func() {
			log.Print("DEBUG: Watcher onGone callback called")
			writeIfChanged(false)
			cancel()
		})
	}

	stopWatcher := func() {
		if watchCancel != nil {
			log.Print("DEBUG: Stopping watcher")
			watchCancel()
			watchCancel = nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			stopWatcher()
			return
		case m := <-ch:
			switch m.kind {
			case msgScOn:

				alive := isAnyProcessAlive(procNames)
				log.Printf("DEBUG: msgScOn - checking processes %v: alive=%v", procNames, alive)
				if alive {
					log.Printf("DEBUG: At least one process is alive, starting watcher")
					writeIfChanged(true)
					startWatcher() // watcher will flip to OFF if process dies
				} else {
					log.Printf("DEBUG: No processes alive, not starting watcher")
					stopWatcher() // stop any existing watcher since process is not alive
					writeIfChanged(true)
					writeStatus(true)
				}

			case msgScOff:
				// Rule: write OFF when screencast reports OFF
				writeIfChanged(false)
				stopWatcher()
			default:
				writeIfChanged(false)
				stopWatcher()
			}
		}
	}
}

// watchUntilGone: immediate check, then poll. When the process is not running
// (or only zombies remain) it calls onGone() exactly once and returns.
func watchUntilGone(ctx context.Context, names []string, every time.Duration, onGone func()) {
	// immediate check (should not be needed since we only start watcher if process is alive)
	if alive := isAnyProcessAlive(names); !alive {
		log.Printf("DEBUG: Watcher immediate check - no processes %v alive, calling onGone", names)
		onGone()
		return
	}
	log.Printf("DEBUG: Watcher started - at least one process from %v is alive", names)

	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if alive := isAnyProcessAlive(names); !alive {
				log.Printf("DEBUG: Watcher polling check - no processes %v alive, calling onGone", names)
				onGone()
				return
			}
		}
	}
}

const (
	messageOn  = "screencast on"
	messageOff = "screencast off"
)

func writeStatus(on bool) {
	message := messageOff
	if on {
		message = messageOn
	}

	// Only send notification if it's not the initial "off" state
	if !(message == messageOff && notificationCount == 0) {
		if err := sendNotification("Screencast", message, icon); err != nil {
			fmt.Fprintf(os.Stderr, "could not send notification: %v", err)
		}
	}
	notificationCount++

	tmp := statusFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(onOff(on)+""), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "could not write status file: %v", err)
		os.Exit(1)
	}
	if err := os.Rename(tmp, statusFile); err != nil {
		fmt.Fprintf(os.Stderr, "could not rename status file: %v", err)
		os.Exit(1)
	}
}

// isAnyProcessAlive returns true if there exists at least one process
// from the given names that is alive and whose state is NOT zombie.
func isAnyProcessAlive(names []string) bool {
	return slices.ContainsFunc(names, isProcessAliveByName)
}

// isProcessAliveByName returns true if there exists at least one process
// whose name matches and whose state is NOT zombie.
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

		// Name match via /proc/<pid>/comm
		comm, err := os.ReadFile(filepath.Join("/proc", n, "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) != name {
			continue
		}

		// Exclude zombies via /proc/<pid>/status → State: Z (zombie)
		if isZombie(n) {
			continue
		}
		return true // a live matching process exists
	}
	return false
}

func isZombie(pid string) bool {
	b, err := os.ReadFile(filepath.Join("/proc", pid, "status"))
	if err != nil {
		return false // if we can't read, treat as not zombie and let next tick handle disappearance
	}
	for line := range bytes.SplitSeq(b, []byte{'\n'}) {
		if bytes.HasPrefix(line, []byte("State:")) {
			// format: "State:	S (sleeping)" → take first rune after tab
			parts := bytes.SplitN(line, []byte{'\t'}, 2)
			if len(parts) == 2 && len(parts[1]) > 0 {
				ch := parts[1][0]
				return ch == 'Z' || ch == 'X' // zombie or dead
			}
		}
	}
	return false
}

func onOff(b bool) string {
	if b {
		return "●"
	}
	return "" // empty for "off" , will be used in waybar
}
