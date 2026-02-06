package window

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/thiagokokada/hyprland-go"
)

func NormalizeAddress(address string) string {
	if !strings.HasPrefix(address, "0x") {
		return "0x" + address
	}
	return address
}

func BuildCommands(win hyprland.Client, position string) []string {
	var (
		commands = make([]string, 0, 18)
		addr     = fmt.Sprintf("address:%s", NormalizeAddress(win.Address))
	)
	if len(win.Grouped) > 0 {
		commands = append(commands, fmt.Sprintf("moveoutofgroup %s", addr))
	}
	commands = append(commands,
		fmt.Sprintf("pin %s", addr),
		fmt.Sprintf("setfloating %s", addr),
		fmt.Sprintf("movewindowpixel exact %s,%s", position, addr),
		fmt.Sprintf("[[BATCH]]setprop %s rounding 1", addr),
		fmt.Sprintf("[[BATCH]]setprop %s no_max_size 0", addr),
		fmt.Sprintf("[[BATCH]]setprop %s opaque toggle", addr),
		fmt.Sprintf("[[BATCH]]setprop %s immediate unset", addr),
		fmt.Sprintf("[[BATCH]]setprop %s border_size relative -2", addr),
		fmt.Sprintf("[[BATCH]]setprop %s rounding_power relative 0.1", addr),
		fmt.Sprintf("[[BATCH]]setprop %s decorate 0", addr),
		fmt.Sprintf("[[BATCH]]setprop %s no_shadow 1", addr),
		fmt.Sprintf("[[BATCH]]setprop %s opacity 1.0 1.0", addr),
		fmt.Sprintf("[[BATCH]]setprop %s no_blur 1", addr),
		fmt.Sprintf("tagwindow +meeting %s", addr),
	)

	return commands
}

func DispatchCommands(c *Client, commands []string) {
	const maxRetries = 5

	var (
		normal = commands[:0]
		batch  = make([]string, 0, len(commands))
	)

	for _, cmd := range commands {
		if trimmed, ok := strings.CutPrefix(cmd, "[[BATCH]]"); ok {
			if !strings.HasPrefix(trimmed, "dispatch ") {
				trimmed = "dispatch " + trimmed
			}
			batch = append(batch, trimmed)
			continue
		}
		normal = append(normal, cmd)
	}

	for _, cmd := range normal {
		var (
			response []hyprland.Response
			err      error
		)
		for attempt := range maxRetries {
			response, err = c.Dispatch(cmd)
			if err == nil {
				break
			}
			msg := err.Error()
			if strings.Contains(msg, "Window does not qualify to be pinned") {
				slog.Debug("window does not qualify to be pinned, aborting",
					"command", cmd,
					"response", response,
					"error", err,
				)
				return
			}
			if !strings.Contains(msg, "Window not found") {
				break
			}
			slog.Debug("window not found, retrying", "command", cmd, "attempt", attempt+1)
			time.Sleep(500 * time.Millisecond)
		}
		if err != nil {
			slog.Error("failed to dispatch command", "command", cmd, "error", err)
		}
	}

	if len(batch) == 0 {
		return
	}
	batches := strings.Join(batch, ";")
	r, err := c.RawRequest(
		hyprland.RawRequest(fmt.Sprintf("[[BATCH]]%s", batches)),
	)
	if err != nil {
		slog.Error("failed to dispatch batch commands", "error", err)
	}
	slog.Debug("batch commands response", "response", r, "batches", batches)
}
