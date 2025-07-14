package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

const (
	gapsOut    = 10
	gapsIn     = 5
	rounding   = 10
	borderSize = 2
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
)

func updateGaps(client *hyprland.RequestClient) error {
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
		log.Printf("Visible window count unchanged (%d), skipping update", visibleCount)
		return nil
	}
	atomic.StoreInt32(&lastVisibleCount, int32(visibleCount))

	log.Printf("Updating gaps: visible windows in workspace %d: %d", ws.Id, visibleCount)

	var cmds []string
	if visibleCount <= 1 {
		cmds = []string{
			"general:gaps_out 0",
			"general:gaps_in 0",
			"decoration:rounding 0",
			"general:border_size 0",
		}
	} else {
		cmds = []string{
			fmt.Sprintf("general:gaps_out %d", gapsOut),
			fmt.Sprintf("general:gaps_in %d", gapsIn),
			fmt.Sprintf("decoration:rounding %d", rounding),
			fmt.Sprintf("general:border_size %d", borderSize),
		}
	}

	if _, err := client.Keyword(cmds...); err != nil {
		return fmt.Errorf("failed to dispatch keyword batch: %v", err)
	}

	return nil
}

func scheduleUpdate() {
	select {
	case updateChan <- struct{}{}:
	default:
	}
}

func updateLoop(client *hyprland.RequestClient) {
	for range updateChan {
		time.Sleep(10 * time.Millisecond)
		if err := updateGaps(client); err != nil {
			log.Printf("Failed to update gaps: %v", err)
		}
	}
}

type ev struct {
	event.DefaultEventHandler
	client *hyprland.RequestClient
}

func (e *ev) Workspace(w event.WorkspaceName) {
	scheduleUpdate()
}

func (e *ev) ActiveWindow(w event.ActiveWindow) {
	scheduleUpdate()
}

func main() {
	client := hyprland.MustClient()
	go updateLoop(client)

	scheduleUpdate()

	ec := event.MustClient()
	defer ec.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	if err := ec.Subscribe(ctx, &ev{client: client}, event.EventWorkspace, event.EventActiveWindow); err != nil {
		log.Fatalf("Failed to subscribe to events: %v", err)
	}
}
