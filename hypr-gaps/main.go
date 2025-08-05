package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
	"golang.org/x/sys/unix"
)

var (
	excludeClasses = map[string]bool{
		"wofi":        true,
		"flameshot":   true,
		"slurp":       true,
		"wf-recorder": true,
	}
	lastVisibleCount int32 = -1
	updateChan             = make(chan struct{}, 1)

	defaultGapsOut       = hyprland.Option{Custom: "10 10 10 10", Set: true}
	defaultGapsIn        = hyprland.Option{Custom: "5 5 5 5", Set: true}
	defaultRounding      = hyprland.Option{Int: 10, Set: true}
	defaultBorderSize    = hyprland.Option{Int: 2, Set: true}
	defaultShadowEnabled = hyprland.Option{Int: 1, Set: true}
)

func updateKeywords(client *hyprland.RequestClient) error {
	ws, err := client.ActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %v", err)
	}

	clients, err := client.Clients()
	if err != nil {
		return fmt.Errorf("failed to list clients: %v", err)
	}

	visibleCount := 0
	for _, c := range clients {
		if c.Workspace.Id == ws.Id && !c.Floating && c.Mapped && !c.Hidden && !excludeClasses[c.Class] {
			visibleCount++
		}
	}

	if int32(visibleCount) == atomic.LoadInt32(&lastVisibleCount) {
		// log.Printf("Visible window count unchanged (%d), skipping update", visibleCount)
		return nil
	}
	atomic.StoreInt32(&lastVisibleCount, int32(visibleCount))

	// log.Printf("Updating gaps: visible windows in workspace %d: %d", ws.Id, visibleCount)
	cmds := defaultKeywords()
	if visibleCount <= 1 {
		cmds = noGaps()
	}

	if _, err := client.Keyword(cmds...); err != nil {
		return fmt.Errorf("failed to dispatch keyword batch: %v", err)
	}

	return nil
}

func noGaps() []string {
	return []string{
		"general:gaps_out 0",
		"general:gaps_in 0",
		"decoration:rounding 0",
		"general:border_size 0",
		"decoration:shadow:enabled 0",
	}
}
func defaultKeywords() []string {
	return []string{
		fmt.Sprintf("animations:enabled %d", 1),
		fmt.Sprintf("decoration:blur:enabled %d", 1),
		fmt.Sprintf("general:gaps_out %s", defaultGapsOut.String()),
		fmt.Sprintf("general:gaps_in %s", defaultGapsIn.String()),
		fmt.Sprintf("decoration:rounding %s", defaultRounding.String()),
		fmt.Sprintf("general:border_size %s", defaultBorderSize.String()),
		fmt.Sprintf("decoration:shadow:enabled %s", defaultShadowEnabled.String()),
	}
}
func gameModeKeywords() []string {
	return []string{
		"animations:enabled 0",
		"decoration:blur:enabled 0",
		"general:gaps_in 0",
		"general:gaps_out 0",
		"general:border_size 1",
		"decoration:rounding 0",
	}

}
func schedule() {
	select {
	case updateChan <- struct{}{}:
	default:
	}
}

func update(client *hyprland.RequestClient) {
	for range updateChan {
		time.Sleep(10 * time.Millisecond)
		if err := updateKeywords(client); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update gaps: %v\n", err)
		}
	}
}

type ev struct {
	event.DefaultEventHandler
}

func (e *ev) Workspace(w event.WorkspaceName)   { schedule() }
func (e *ev) ActiveWindow(w event.ActiveWindow) { schedule() }

func (e *ev) MoveWorkspace(w event.MoveWorkspace)    { schedule() }
func (e *ev) CreateWorkspace(w event.WorkspaceName)  { schedule() }
func (e *ev) DestroyWorkspace(w event.WorkspaceName) { schedule() }
func (e *ev) CloseWindow(w event.CloseWindow)        { schedule() }
func (e *ev) MoveWindow(w event.MoveWindow)          { schedule() }
func (e *ev) ToggleGroup(w event.ToggleGroup)        { schedule() }
func (e *ev) MoveOutofGroup(w event.MoveOutofGroup)  { schedule() }
func (e *ev) MoveIntogroup(w event.MoveIntogroup)    { schedule() }

func lock(file string) (*os.File, error) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("unable to open lock file: %v", err)
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, fmt.Errorf("another instance is already running")
	}
	return f, nil
}

func opt(client *hyprland.RequestClient, key string, fallback hyprland.Option) hyprland.Option {
	opt, err := client.GetOption(key)
	if err != nil {
		return fallback
	}
	return opt
}

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, "Do not run this program as root or with sudo")
		os.Exit(1)
	}

	var (
		client   = hyprland.MustClient()
		reset    = flag.Bool("reset", false, "reset to default values")
		noG      = flag.Bool("no-gaps", false, "set no gaps")
		gameMode = flag.Bool("game-mode", false, "set game mode")
	)

	flag.Parse()
	if reset != nil && *reset {
		if _, err := client.Keyword(defaultKeywords()...); err != nil {
			fmt.Fprintf(os.Stderr, "failed to reset gaps: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if noG != nil && *noG {
		if _, err := client.Keyword(noGaps()...); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set no-gaps: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if gameMode != nil && *gameMode {
		if _, err := client.Keyword(gameModeKeywords()...); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set game mode: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	f, err := lock("/tmp/hypr-gap-manager.lock")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	defaultGapsOut = opt(client, "general:gaps_out", defaultGapsOut)
	defaultGapsIn = opt(client, "general:gaps_in", defaultGapsIn)
	defaultRounding = opt(client, "decoration:rounding", defaultRounding)
	defaultBorderSize = opt(client, "general:border_size", defaultBorderSize)
	defaultShadowEnabled = opt(client, "decoration:shadow:enabled", defaultShadowEnabled)

	go update(client)

	schedule()

	ec := event.MustClient()
	defer ec.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Fprintf(os.Stderr, "Received signal: %v\n", sig)
		cancel()
	}()
	subscribe := []event.EventType{
		event.EventWorkspace,
		event.EventActiveWindow,
		event.EventMoveWorkspace,
		event.EventCreateWorkspace,
		event.EventDestroyWorkspace,
		event.EventCloseWindow,
		event.EventMoveWindow,
		event.EventToggleGroup,
		event.EventMoveIntogroup,
		event.EventMoveOutofGroup,
	}
	if err := ec.Subscribe(ctx, &ev{}, subscribe...); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe to events: %v\n", err)
	}
}
