package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

const (
	gapsOut = 10
	gapsIn  = 5
)

var (
	excludeClasses = map[string]bool{
		"wofi":        true,
		"flameshot":   true,
		"slurp":       true,
		"wf-recorder": true,
	}
	lastVisibleCount = -1
)

func updateGaps(client *hyprland.RequestClient) error {
	activeWindow, err := client.ActiveWindow()
	if err != nil {
		return fmt.Errorf("failed to get active window: %v", err)
	}

	clients, err := client.Clients()
	if err != nil {
		return fmt.Errorf("failed to list clients: %v", err)
	}

	visibleCount := 0
	for _, c := range clients {
		if c.Workspace.Id == activeWindow.Workspace.Id && !c.Floating && c.Mapped && !c.Hidden {
			if !excludeClasses[c.Class] {
				visibleCount++
			}
		}
	}

	if visibleCount == lastVisibleCount {
		log.Printf("Visible window count unchanged (%d), skipping update", visibleCount)
		return nil
	}
	lastVisibleCount = visibleCount

	log.Printf("Updating gaps: visible windows in workspace %d: %d", activeWindow.Workspace.Id, visibleCount)

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
			fmt.Sprintf("decoration:rounding %d", 10),
			fmt.Sprintf("general:border_size %d", 2),
		}
	}

	if _, err := client.Keyword(cmds...); err != nil {
		return fmt.Errorf("failed to dispatch batch keyword: %v", err)
	}

	return nil
}

type ev struct {
	event.DefaultEventHandler
	client *hyprland.RequestClient
}

func (e *ev) Workspace(w event.WorkspaceName) {
	if err := updateGaps(e.client); err != nil {
		log.Printf("Failed to update gaps: %v", err)
	}
}

func (e *ev) ActiveWindow(w event.ActiveWindow) {
	if err := updateGaps(e.client); err != nil {
		log.Printf("Failed to update gaps: %v", err)
	}
}

func main() {
	client := hyprland.MustClient()

	if err := updateGaps(client); err != nil {
		log.Printf("Failed to update gaps: %v", err)
	}

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
