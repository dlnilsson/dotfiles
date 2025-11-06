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

	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/pipewire"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

var (
	statusFile = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", "screencast.status")
	procNames  = []string{
		"wl-screenrec",
	}
	pollEvery            = 2 * time.Second
	notificationCount    = 0
	lastMessage          = ""
	lastNotification     = time.Time{}
	notificationCooldown = 3 * time.Second
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

	args := []string{
		title,
		message,
		"-a",
		"hypr-screencapture",
		"-i", tmp,
		"-t", "3000",
		"-u", "normal",
		"--transient",
	}
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

type msg = messages.Msg

const (
	msgScOn  = messages.ScOn
	msgScOff = messages.ScOff
)

type screencastHandler struct {
	event.DefaultEventHandler
	ch chan<- msg
}

func (h *screencastHandler) Screencast(w event.Screencast) {
	log.Printf("Screencast %v --- %v\n", w.Sharing, w.Owner)
	if w.Sharing {
		h.ch <- msg{Kind: msgScOn, Source: messages.SourceHyprland}
		return
	}
	h.ch <- msg{Kind: msgScOff, Source: messages.SourceHyprland}
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

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	var (
		// signal channel for SIGINT and SIGTERM
		sigCh = make(chan os.Signal, 1)
		// main channel for Hyprland and PipeWire events
		ch = make(chan msg, 8)
		// fan-out channel for main manager from main channel
		managerCh = make(chan msg, 4)
		// fan-out channel for pinWindow from main channel
		pinWindowCh = make(chan msg, 4)
		// Hyprland event client
		cli         = event.MustClient()
		ctx, cancel = context.WithCancel(context.Background())
	)
	defer cli.Close()
	defer cancel()

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()

	defer pipewire.StopPipeWireLoop()

	go func() {
		if ret := pipewire.StartPipeWireLoop(); ret != 0 {
			log.Printf("PipeWire loop exited with code: %d", ret)
		}
	}()

	pipewire.SetChannel(ch)

	go func() {
		if err := cli.Subscribe(ctx, &screencastHandler{ch: ch}, event.EventScreencast); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "Subscribe exited with error: %v", err)
			cancel()
		}
	}()

	log.Printf("XDG_RUNTIME_DIR=%q HYPRLAND_INSTANCE_SIGNATURE=%q", os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"))
	if c, err := hyprland.MustClient().Clients(); err != nil {
		log.Printf("IPC check: Clients() error: %v", err)
	} else {
		log.Printf("IPC check: saw %d clients at start", len(c))
	}

	// Fan out messages to both channels with rate limiting and PipeWire prioritization
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
				log.Printf("DEBUG: Fan-out received message: %+v", m)

				// Check rate limit with PipeWire prioritization
				now := time.Now()
				if lastTime, exists := lastSent[m.Kind]; exists {
					timeSinceLastSent := now.Sub(lastTime)
					if timeSinceLastSent < rateLimitDuration {
						// If last message was from Hyprland and this is from PipeWire, allow it through
						if lastSrc, ok := lastSource[m.Kind]; ok {
							if lastSrc == messages.SourceHyprland && m.Source == messages.SourcePipeWire {
								log.Printf("DEBUG: Prioritizing PipeWire message over Hyprland (last sent %v ago)", timeSinceLastSent)
							} else {
								log.Printf("DEBUG: Rate limiting message %+v (last sent %v ago)", m, timeSinceLastSent)
								continue
							}
						} else {
							log.Printf("DEBUG: Rate limiting message %+v (last sent %v ago)", m, timeSinceLastSent)
							continue
						}
					}
				}

				// Update last sent time and source
				lastSent[m.Kind] = now
				lastSource[m.Kind] = m.Source

				select {
				case managerCh <- m:
					log.Printf("DEBUG: Fan-out sent message to manager")
				case <-ctx.Done():
					return
				default:
					log.Printf("DEBUG: Fan-out failed to send to manager (channel full)")
				}
				select {
				case pinWindowCh <- m:
					log.Printf("DEBUG: Fan-out sent message to pinWindow")
				case <-ctx.Done():
					return
				default:
					log.Printf("DEBUG: Fan-out failed to send to pinWindow (channel full)")
				}
			}
		}
	}()

	// Manager owns the state machine and file writes
	go manager(ctx, managerCh)
	go pinWindow(ctx, pinWindowCh)

	<-ctx.Done()
	// allow final rename to flush
	time.Sleep(50 * time.Millisecond)
}

func isSmallWindow(size []int) bool {
	if len(size) < 2 {
		return false
	}
	return (size[0] <= 620 && size[1] <= 64) || (size[0] <= 64 && size[1] <= 620)
}

type pinConfig struct {
	initialDelay  time.Duration
	maxRetries    int
	retryDelay    time.Duration
	maxRetryDelay time.Duration
	pollInterval  time.Duration
	maxPollTime   time.Duration
}

func pinWindow(ctx context.Context, ch <-chan msg) {
	var (
		client = hyprland.MustClient()
		config = pinConfig{
			initialDelay:  500 * time.Millisecond,
			maxRetries:    3,
			retryDelay:    500 * time.Millisecond,
			maxRetryDelay: 2 * time.Second,
			pollInterval:  500 * time.Millisecond,
			maxPollTime:   10 * time.Second,
		}
	)
	log.Printf("DEBUG: pinWindow started with config: %+v", config)
	for {
		select {
		case <-ctx.Done():
			log.Printf("DEBUG: pinWindow context done")
			return
		case m := <-ch:
			log.Printf("DEBUG: pinWindow received message: %+v", m)
			log.Printf("DEBUG: pinWindow received message: KIND %+v", m.Kind)
			switch m.Kind {
			case msgScOn:
				log.Printf("DEBUG: screencast started, looking for windows to pin")
				go pinWindowsWithRetry(ctx, client, config)
			case msgScOff:
				log.Printf("DEBUG: screencast stopped, no action needed for pinWindow")
			}
		}
	}
}

func pinWindowsWithRetry(ctx context.Context, client *hyprland.RequestClient, config pinConfig) {
	time.Sleep(config.initialDelay)
	var (
		pinnedWindows   = make(map[string]bool)
		ticker          = time.NewTicker(config.pollInterval)
		pollCtx, cancel = context.WithTimeout(ctx, config.maxPollTime)
	)
	defer cancel()
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			if len(pinnedWindows) == 0 {
				log.Printf("DEBUG: timeout reached, no suitable windows found to pin")
			}
			return
		case <-ticker.C:
			clients, err := client.Clients()
			if err != nil {
				log.Printf("ERROR: failed to get clients: %v", err)
				continue
			}

			candidates := findPinCandidates(clients, pinnedWindows)
			if len(candidates) == 0 {
				continue
			}

			log.Printf("DEBUG: found %d new candidate windows to pin", len(candidates))

			for _, candidate := range candidates {
				if tryPinWindow(client, candidate, config) {
					pinnedWindows[candidate.Address] = true
					log.Printf("DEBUG: successfully pinned window %s - %s", candidate.Address, candidate.Title)
				}
			}

			if len(pinnedWindows) > 0 {
				log.Printf("DEBUG: pinned %d windows total, stopping search", len(pinnedWindows))
				return
			}
		}
	}
}

func findPinCandidates(clients []hyprland.Client, alreadyPinned map[string]bool) []hyprland.Client {
	var candidates []hyprland.Client

	for _, c := range clients {
		if alreadyPinned[c.Address] {
			continue
		}

		if !c.Floating {
			continue
		}

		if !isSmallWindow(c.Size) {
			continue
		}

		if isLikelyScreencastWindow(c) {
			log.Printf("DEBUG: found candidate window %s - %s - Size:%v",
				c.Address, c.Title, c.Size)
			candidates = append(candidates, c)
		}
	}

	return candidates
}

func isLikelyScreencastWindow(c hyprland.Client) bool {
	var (
		title              = strings.ToLower(c.Title)
		class              = strings.ToLower(c.Class)
		screencastKeywords = []string{
			"Sharing Indicator", "sharing", "screen", "recording", "capture",
			"cast", "obs", "streamlabs", "discord", "zoom", "teams", "meet",
			"chrome", "firefox", "browser", "portal", "slack", "zen-browser",
		}
	)

	for _, keyword := range screencastKeywords {
		if strings.Contains(title, keyword) || strings.Contains(class, keyword) {
			return true
		}
	}

	return c.Size[0] > 0 && c.Size[1] > 0
}

func tryPinWindow(client *hyprland.RequestClient, window hyprland.Client, config pinConfig) bool {
	addr := window.Address

	for attempt := 0; attempt < config.maxRetries; attempt++ {
		if attempt > 0 {
			delay := min(time.Duration(attempt)*config.retryDelay, config.maxRetryDelay)
			log.Printf("DEBUG: retrying pin operation for %s (attempt %d/%d) after %v",
				addr, attempt+1, config.maxRetries, delay)
			time.Sleep(delay)
		}
		commands := []string{
			fmt.Sprintf("pin address:%s", addr),
			fmt.Sprintf("setprop address:%s decorate 0", addr),
			fmt.Sprintf("setprop address:%s noborder 1", addr),
			fmt.Sprintf("movewindowpixel exact 1421 25 %s", addr),
		}
		for _, command := range commands {
			if _, err := client.Dispatch(command); err != nil {
				log.Printf("ERROR: attempt %d failed to execute command %s: %v", attempt+1, command, err)
				continue
			}
		}

		if verifyWindowPinned(client, addr) {
			return true
		}

		log.Printf("DEBUG: pin command succeeded but window %s doesn't appear to be pinned", addr)
	}

	log.Printf("ERROR: failed to pin window %s after %d attempts", addr, config.maxRetries)
	return false
}

func verifyWindowPinned(client *hyprland.RequestClient, address string) bool {
	clients, err := client.Clients()
	if err != nil {
		log.Printf("DEBUG: could not verify pin status: %v", err)
		return false
	}

	for _, c := range clients {
		if c.Address == address {
			return c.Pinned
		}
	}

	log.Printf("DEBUG: window %s not found during verification", address)
	return false
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
			log.Printf("DEBUG: manager received message: %+v", m)
			switch m.Kind {
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
	defer func() {
		go func() {
			if err := toggleNotificationInhibitor(on); err != nil {
				fmt.Fprintf(os.Stderr, "could not toggle notification inhibitor: %v", err)
			}
		}()
	}()

	// Only send notification if it's not the initial "off" state
	// and avoid duplicate notifications in short succession
	now := time.Now()
	shouldNotify := !(message == messageOff && notificationCount == 0) &&
		(message != lastMessage || now.Sub(lastNotification) >= notificationCooldown)

	if shouldNotify {
		if err := sendNotification("Screencast", message, icon); err != nil {
			fmt.Fprintf(os.Stderr, "could not send notification: %v", err)
		}
		lastMessage = message
		lastNotification = now
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

func toggleNotificationInhibitor(add bool) error {
	if add {
		return exec.Command("swaync-client", "--inhibitor-add", "xdg-desktop-portal-hyprland").Run()
	}
	return exec.Command("swaync-client", "--inhibitor-remove", "xdg-desktop-portal-hyprland").Run()
}

func onOff(b bool) string {
	if b {
		return "●"
	}
	return "" // empty for "off" , will be used in waybar
}
