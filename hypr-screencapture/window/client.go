package window

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go"
)

const maxHandledAddresses = 1000

type Client struct {
	handledAddresses map[string]bool
	handledMu        sync.Mutex
	*hyprland.RequestClient
}

func NewClient(hc *hyprland.RequestClient) *Client {
	return &Client{
		handledAddresses: make(map[string]bool),
		RequestClient:    hc,
	}
}

func (c *Client) MarkHandled(address string) bool {
	c.handledMu.Lock()
	defer c.handledMu.Unlock()
	if c.handledAddresses == nil {
		c.handledAddresses = make(map[string]bool)
	}
	if c.handledAddresses[address] {
		return false
	}
	if len(c.handledAddresses) >= maxHandledAddresses {
		c.handledAddresses = make(map[string]bool)
	}
	c.handledAddresses[address] = true
	return true
}

func (c *Client) ClearHandled() {
	c.handledMu.Lock()
	defer c.handledMu.Unlock()
	c.handledAddresses = make(map[string]bool)
}

func (c *Client) RemoveHandled(address string) {
	c.handledMu.Lock()
	defer c.handledMu.Unlock()
	if c.handledAddresses != nil {
		delete(c.handledAddresses, address)
	}
}

func isSmallWindow(size []int) bool {
	if len(size) < 2 {
		return false
	}
	return (size[0] <= 620 && size[1] <= 64) || (size[0] <= 64 && size[1] <= 620)
}

func (c *Client) FindPinCandidates(clients []hyprland.Client, alreadyPinned map[string]bool, cfg *config.Config) []hyprland.Client {
	var candidates []hyprland.Client

	for _, client := range clients {
		if alreadyPinned[client.Address] {
			continue
		}

		if !client.Floating {
			continue
		}

		if !isSmallWindow(client.Size) {
			continue
		}

		if c.IsLikelyScreencastWindow(client, cfg) {
			slog.Debug("found candidate window", "address", client.Address, "title", client.Title, "size", client.Size)
			candidates = append(candidates, client)
		}
	}

	return candidates
}

func (c *Client) IsLikelyScreencastWindow(client hyprland.Client, cfg *config.Config) bool {
	var (
		title    = strings.ToLower(client.Title)
		class    = strings.ToLower(client.Class)
		keywords = cfg.WindowMatching.ScreencastKeywords
	)

	for _, keyword := range keywords {
		if strings.Contains(title, keyword) || strings.Contains(class, keyword) {
			return true
		}
	}

	return false
}
