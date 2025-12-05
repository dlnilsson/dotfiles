// hypr-screencapture: monitor Hyprland screencast events and expose status via Unix socket
// combine with waybar custom script module to show screencast status in bar
// example:
//
//	"custom/recording": {
//	    "exec": "hypr-screencapture status",
//	    "interval": 3,
//	    "format": "{}",
//	    "tooltip": false,
//	    "on-click": "/home/dln/.dotfiles/bin/screen-recorder"
//	},
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "embed"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/messages"
	"github.com/dlnilsson/dotfiles/hypr-screencapture/pipewire"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

const (
	// https://www.freedesktop.org/software/systemd/man/latest/systemd.exec.html
	// exit codes for systemd
	exUnavailable = 69
	exOSErr       = 71
	exCantCreat   = 73
	exIOErr       = 74
	exNoPerm      = 77
	exConfig      = 78
)

type notificationState struct {
	// protects all fields in this struct
	mu sync.Mutex
	// total count of notifications sent
	count int
	// tracks the last time each message was sent
	lastMessageTimes map[string]time.Time
	// minimum duration between duplicate notifications
	cooldown time.Duration
}

func (n *notificationState) shouldNotify(message string, isInitialOff bool) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.lastMessageTimes == nil {
		n.lastMessageTimes = make(map[string]time.Time)
	}

	if isInitialOff {
		return false
	}

	lastTime, exists := n.lastMessageTimes[message]
	if !exists || time.Since(lastTime) >= n.cooldown {
		n.lastMessageTimes[message] = time.Now()
		n.count++
		return true
	}

	return false
}

var (
	cfg   *config.Config
	cfgMu sync.RWMutex

	notifications *notificationState

	socketStatus struct {
		mu     sync.RWMutex
		status string
	}
)

//go:embed camera.png
var icon []byte

func sendNotification(title, message string, iconBytes []byte) error {
	cmd, err := exec.LookPath("notify-send")
	if err != nil {
		return err
	}
	args := []string{
		title,
		message,
		"-a",
		"hypr-screencapture",
		"-t", "3000",
		"-u", "normal",
		"--transient",
	}
	if iconBytes != nil {
		tmp, err := bytesToFilename(iconBytes)
		if err != nil {
			return err
		}
		defer os.Remove(tmp)
		args = append(args, "-i", tmp)
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

func stderr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

type msg = messages.Msg

const (
	msgScOn     = messages.ScOn
	msgScOff    = messages.ScOff
	msgCameraOn = messages.CameraOn
)

type screencastHandler struct {
	event.DefaultEventHandler
	// channel for sending messages
	ch chan<- msg
	// Hyprland IPC client for dispatching commands
	client *hyprland.RequestClient
	// tracks window addresses that have already been handled
	handledAddresses map[string]bool
	// protects handledAddresses map
	handledMu sync.Mutex
}

func (h *screencastHandler) Screencast(w event.Screencast) {
	log.Printf("Screencast %v --- %v", w.Sharing, w.Owner)
	var m msg
	if w.Sharing {
		m = msg{Kind: msgScOn, Source: messages.SourceHyprland}
	} else {
		m = msg{Kind: msgScOff, Source: messages.SourceHyprland}
	}
	select {
	case h.ch <- m:
		log.Printf("DEBUG: Hyprland sent message: %+v", m)
	default:
		log.Printf("DEBUG: WARNING - Hyprland failed to send message (channel full): %+v", m)
	}
}

// dispatchCommands executes a slice of commands and logs any errors
func (h *screencastHandler) dispatchCommands(commands []string) {
	for _, cmd := range commands {
		if _, err := h.client.Dispatch(cmd); err != nil {
			log.Printf("ERROR: failed to dispatch %s command: %v", cmd, err)
		}
	}
}

func (h *screencastHandler) handleHangoutWindow(address string) {
	h.handledMu.Lock()
	defer h.handledMu.Unlock()
	if h.handledAddresses == nil {
		h.handledAddresses = make(map[string]bool)
	}
	if h.handledAddresses[address] {
		return
	}
	h.handledAddresses[address] = true

	time.Sleep(100 * time.Millisecond)
	log.Printf("DEBUG: handling hangout window: %s", address)

	h.dispatchCommands([]string{
		fmt.Sprintf("tagwindow +meeting address:%s", address),
		fmt.Sprintf("pin address:%s", address),
		fmt.Sprintf("setfloating address:%s", address),
		fmt.Sprintf("movewindowpixel exact %s,address:%s", h.meetingPosition(address), address),
		// rounding 1 is key to avoid flickering
		fmt.Sprintf("setprop address:%s rounding 1", address),
		fmt.Sprintf("setprop address:%s noanim 1", address),
		fmt.Sprintf("setprop address:%s nomaxsize 0", address),
		fmt.Sprintf("setprop address:%s opaque toggle", address),
		fmt.Sprintf("setprop address:%s immediate unset", address),
		fmt.Sprintf("setprop address:%s bordersize relative -2", address),
		fmt.Sprintf("setprop address:%s roundingpower relative 0.1", address),
	})
}

func (h *screencastHandler) handlePictureInPictureWindow(address string) {
	time.Sleep(100 * time.Millisecond)
	log.Printf("DEBUG: handling picture-in-picture window: %s", address)

	h.dispatchCommands([]string{
		fmt.Sprintf("movewindowpixel exact %s,address:%s", h.meetingPosition(address), address),
	})
}

func (h *screencastHandler) OpenWindow(w event.OpenWindow) {
	// Hyprland IPC events provide addresses without the "0x" prefix,
	// but commands expect the full hexadecimal format
	address := "0x" + w.Address

	switch {
	case h.isHangoutWindow(w):
		h.handleHangoutWindow(address)
	case h.isPictureInPictureWindow(w):
		h.handlePictureInPictureWindow(address)
	default:
		// Not a window we care about
	}
}

func (h *screencastHandler) meetingPosition(addr string) string {
	clients, err := h.client.Clients()
	if err != nil {
		cfgMu.RLock()
		pos := cfg.Positioning.DefaultPosition
		cfgMu.RUnlock()
		return pos
	}
	for _, client := range clients {
		if client.Address == addr {
			p, err := calculateWindowPosition(h.client, client)
			if err != nil {
				cfgMu.RLock()
				pos := cfg.Positioning.DefaultPosition
				cfgMu.RUnlock()
				return pos
			}
			return p
		}
	}
	cfgMu.RLock()
	pos := cfg.Positioning.DefaultPosition
	cfgMu.RUnlock()
	return pos
}

func isHangoutTitle(title string) bool {
	cfgMu.RLock()
	prefixes := cfg.WindowMatching.MeetTitlePrefixes
	cfgMu.RUnlock()
	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		return strings.HasPrefix(title, prefix)
	})
}

func (h *screencastHandler) isHangoutWindow(w event.OpenWindow) bool {
	return isHangoutTitle(w.Title)
}

func (h *screencastHandler) isPictureInPictureWindow(w event.OpenWindow) bool {
	pipRegex := regexp.MustCompile(`(?i)(Picture-in-Picture|Picture in Picture)`)
	return pipRegex.MatchString(w.Title)
}

func (h *screencastHandler) ActiveWindow(w event.ActiveWindow) {
	a := func() string {
		activeClient, err := h.client.ActiveWindow()
		if err != nil {
			return ""
		}
		return activeClient.Address
	}

	cfgMu.RLock()
	bitwardenTitles := cfg.WindowMatching.BitwardenTitles
	cfgMu.RUnlock()
	if slices.ContainsFunc(bitwardenTitles, func(match string) bool {
		return w.Title == match
	}) {
		addr := fmt.Sprintf("address:%s", a())
		h.dispatchCommands([]string{
			"tagwindow +starship",
			fmt.Sprintf("setfloating %s", addr),
			fmt.Sprintf("resizewindowpixel exact 900 725,%s", addr),
			fmt.Sprintf("centerwindow %s", addr),
		})
	}
	if isHangoutTitle(w.Title) {
		h.handleHangoutWindow(a())
	}
}

func reloadConfig() {
	newCfg, err := config.Load("")
	if err != nil {
		stderr("failed to reload config: %v", err)
		os.Exit(exConfig)
	}

	cfgMu.Lock()
	cfg = newCfg
	notifications.cooldown = cfg.Notifications.Cooldown
	cfgMu.Unlock()

	log.Printf("Config reloaded successfully")
}

func startSocketServer(ctx context.Context, socketPath string) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return fmt.Errorf("could not create directory for socket: %w", err)
	}

	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("could not remove stale socket: %w", err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not listen on socket: %w", err)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
		os.Remove(socketPath)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					stderr("ERROR: failed to accept connection: %v", err)
					continue
				}
			}

			go func(c net.Conn) {
				defer c.Close()

				socketStatus.mu.RLock()
				status := socketStatus.status
				socketStatus.mu.RUnlock()

				if _, err := fmt.Fprintln(c, status); err != nil {
					stderr("ERROR: failed to write status to client: %v", err)
				}
			}(conn)
		}
	}()

	return nil
}

func runStatusCommand() {
	cfg, err := config.Load("")
	if err != nil {
		stderr("failed to load config: %v", err)
		os.Exit(exConfig)
	}

	socketPath := cfg.Paths.SocketPath

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		stderr("could not connect to socket: %v", err)
		os.Exit(exUnavailable)
	}
	defer conn.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, conn); err != nil {
		stderr("could not read from socket: %v", err)
		os.Exit(exIOErr)
	}

	fmt.Print(strings.TrimSpace(buf.String()))
}

func checkServerRunning(socketPath string) error {
	if _, err := os.Stat(socketPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("stale socket exists and could not be removed: %w", err)
		}
		return nil
	}
	conn.Close()

	return errors.New("another instance of hypr-screencapture is already running")
}

func main() {
	if os.Geteuid() == 0 {
		stderr("Do not run this program as root or with sudo")
		os.Exit(exNoPerm)
	}

	if len(os.Args) > 1 && os.Args[1] == "status" {
		runStatusCommand()
		return
	}

	var err error
	cfg, err = config.Load("")
	if err != nil {
		stderr("failed to load config: %v", err)
		os.Exit(exConfig)
	}

	if err := config.EnsureConfigDir(); err != nil {
		stderr("failed to create config directory: %v", err)
		os.Exit(exCantCreat)
	}

	cfgMu.Lock()
	notifications = &notificationState{
		cooldown: cfg.Notifications.Cooldown,
	}
	cfgMu.Unlock()

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
		hc          = hyprland.MustClient()
	)
	defer cli.Close()
	defer cancel()

	cfgMu.RLock()
	socketPath := cfg.Paths.SocketPath
	cfgMu.RUnlock()

	if err := checkServerRunning(socketPath); err != nil {
		stderr("%v", err)
		os.Exit(exUnavailable)
	}

	if err := startSocketServer(ctx, socketPath); err != nil {
		stderr("could not start socket server: %v", err)
		os.Exit(exOSErr)
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
			stderr("PipeWire loop exited with code: %d", ret)
		}
	}()

	pipewire.SetChannel(ch)

	go func() {
		var (
			handler = &screencastHandler{
				ch:     ch,
				client: hc,
			}
			events = []event.EventType{
				event.EventScreencast,
				event.EventActiveWindow,
				event.EventOpenWindow,
			}
		)
		if err := cli.Subscribe(ctx, handler, events...); err != nil && ctx.Err() == nil {
			stderr("Subscribe exited with error: %v", err)
			cancel()
		}
	}()

	log.Printf("XDG_RUNTIME_DIR=%q HYPRLAND_INSTANCE_SIGNATURE=%q", os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"))
	if c, err := hc.Clients(); err != nil {
		stderr("IPC check: Clients() error: %v", err)
	} else {
		log.Printf("IPC check: saw %d clients at start", len(c))
	}

	// Fan out messages to both channels with rate limiting and PipeWire prioritization
	go func() {
		var (
			// tracks the last time each message kind was sent
			lastSent = make(map[messages.Kind]time.Time)
			// tracks the source of the last message for each kind
			lastSource = make(map[messages.Kind]messages.Source)
			// minimum duration between messages of the same kind
			rateLimitDuration = time.Second
		)
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-ch:
				log.Printf("DEBUG: Fan-out received message: Kind=%v Source=%v", m.Kind, m.Source)

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
								log.Printf("DEBUG: Rate limiting message Kind=%v Source=%v (last sent %v ago)", m.Kind, m.Source, timeSinceLastSent)
								continue
							}
						} else {
							log.Printf("DEBUG: Rate limiting message Kind=%v Source=%v (last sent %v ago)", m.Kind, m.Source, timeSinceLastSent)
							continue
						}
					}
				}

				// Update last sent time and source
				lastSent[m.Kind] = now
				lastSource[m.Kind] = m.Source

				select {
				case managerCh <- m:
					log.Printf("DEBUG: Fan-out sent message Kind=%v to manager", m.Kind)
				case <-ctx.Done():
					return
				default:
					log.Printf("DEBUG: Fan-out failed to send Kind=%v to manager (channel full)", m.Kind)
				}
				select {
				case pinWindowCh <- m:
					log.Printf("DEBUG: Fan-out sent message Kind=%v to pinWindow", m.Kind)
				case <-ctx.Done():
					return
				default:
					log.Printf("DEBUG: Fan-out failed to send Kind=%v to pinWindow (channel full)", m.Kind)
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
		pinCfg pinConfig
	)
	cfgMu.RLock()
	pinCfg = pinConfig{
		initialDelay:  cfg.PinWindow.InitialDelay,
		maxRetries:    cfg.PinWindow.MaxRetries,
		retryDelay:    cfg.PinWindow.RetryDelay,
		maxRetryDelay: cfg.PinWindow.MaxRetryDelay,
		pollInterval:  cfg.PinWindow.PollInterval,
		maxPollTime:   cfg.PinWindow.MaxPollTime,
	}
	cfgMu.RUnlock()
	log.Printf("DEBUG: pinWindow started with config: %+v", pinCfg)
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
				go pinWindowsWithRetry(ctx, client, pinCfg)
			case msgScOff:
				if err := sendNotification("Camera", "📸 Camera off!", nil); err != nil {
					stderr("could not send camera notification: %v", err)
				}
			case msgCameraOn:
				if err := sendNotification("Camera", "📸 Camera on!", nil); err != nil {
					stderr("could not send camera notification: %v", err)
				}
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
		title = strings.ToLower(c.Title)
		class = strings.ToLower(c.Class)
	)

	cfgMu.RLock()
	keywords := cfg.WindowMatching.ScreencastKeywords
	cfgMu.RUnlock()

	for _, keyword := range keywords {
		if strings.Contains(title, keyword) || strings.Contains(class, keyword) {
			return true
		}
	}

	return c.Size[0] > 0 && c.Size[1] > 0
}

func calculateWindowPosition(client *hyprland.RequestClient, window hyprland.Client) (string, error) {
	monitors, err := client.Monitors()
	if err != nil {
		return "", fmt.Errorf("failed to get monitors: %w", err)
	}

	var monitor *hyprland.Monitor
	for i := range monitors {
		if monitors[i].Id == window.Monitor {
			monitor = &monitors[i]
			break
		}
	}

	if monitor == nil {
		return "", fmt.Errorf("monitor %d not found", window.Monitor)
	}
	var (
		windowWidth  = window.Size[0]
		monitorWidth = monitor.Width
	)

	cfgMu.RLock()
	rightPadding := cfg.Positioning.RightPadding
	yPos := cfg.Positioning.YPosition
	cfgMu.RUnlock()

	xPos := monitorWidth - windowWidth - rightPadding
	xPercent := int(float64(xPos) / float64(monitorWidth) * 100)
	yPercent := int(float64(yPos) / float64(monitor.Height) * 100)

	if yPercent < 2 {
		yPercent = 2
	}

	return fmt.Sprintf("%d%% %d%%", xPercent, yPercent), nil
}

func executeWindowCommands(client *hyprland.RequestClient, address, position string) {
	commands := []string{
		"pin",
		"setprop decorate 0",
		"setprop noborder 1",
		"setprop noshadow 1",
		"setprop opacity 1.0 1.0",
		"setprop noblur 1",
		"setprop rounding 0",
		fmt.Sprintf("movewindowpixel exact %s,", position),
	}
	for _, command := range commands {
		formatted := fmt.Sprintf("%s address:%s", command, address)
		if _, err := client.Dispatch(formatted); err != nil {
			log.Printf("ERROR: failed to execute command %s: %v", formatted, err)
		}
	}
}

func tryPinWindow(client *hyprland.RequestClient, window hyprland.Client, config pinConfig) bool {
	addr := window.Address

	position, err := calculateWindowPosition(client, window)
	if err != nil {
		log.Printf("WARN: failed to calculate window position: %v, using default", err)
		cfgMu.RLock()
		position = cfg.Positioning.DefaultPosition
		cfgMu.RUnlock()
	}

	for attempt := 0; attempt < config.maxRetries; attempt++ {
		if attempt > 0 {
			delay := min(time.Duration(attempt)*config.retryDelay, config.maxRetryDelay)
			log.Printf("DEBUG: retrying pin operation for %s (attempt %d/%d) after %v",
				addr, attempt+1, config.maxRetries, delay)
			time.Sleep(delay)
		}
		executeWindowCommands(client, addr, position)

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
		lastWritten *bool
		watchCancel context.CancelFunc
	)

	writeIfChanged := func(on bool) {
		if lastWritten == nil || *lastWritten != on {
			log.Printf("DEBUG: Writing status: %s", onOff(on))
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
		cfgMu.RLock()
		procNames := cfg.Processes.Monitor
		pollEvery := cfg.Processes.PollInterval
		cfgMu.RUnlock()
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
				cfgMu.RLock()
				procNames := cfg.Processes.Monitor
				cfgMu.RUnlock()
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
				stderr("could not toggle notification inhibitor: %v", err)
			}
		}()
	}()

	isInitialOff := message == messageOff && notifications.count == 0
	if notifications.shouldNotify(message, isInitialOff) {
		if err := sendNotification("Screencast", message, icon); err != nil {
			stderr("could not send notification: %v", err)
		}
	}

	statusStr := onOff(on)
	socketStatus.mu.Lock()
	socketStatus.status = statusStr
	socketStatus.mu.Unlock()
	log.Printf("DEBUG: Updated socket status: %s", statusStr)
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
