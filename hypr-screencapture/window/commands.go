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

// Hyprland 0.56 replaced the plain string dispatchers with a Lua API: the IPC
// wraps a "dispatch <args>" request into "return hl.dispatch(<args>)", so the
// argument has to be a Lua dispatcher expression instead of a dispatcher name.
func dspMoveOutOfGroup(addr string) string {
	return fmt.Sprintf("hl.dsp.window.move({ out_of_group = true, window = %q })", addr)
}

func dspPin(addr string) string {
	return fmt.Sprintf("hl.dsp.window.pin({ window = %q })", addr)
}

func dspFloat(addr string) string {
	return fmt.Sprintf("hl.dsp.window.float({ action = \"on\", window = %q })", addr)
}

func dspMove(position Position, addr string) string {
	return fmt.Sprintf("hl.dsp.window.move({ x = %d, y = %d, window = %q })", position.X, position.Y, addr)
}

func dspResize(width, height int, addr string) string {
	return fmt.Sprintf("hl.dsp.window.resize({ x = %d, y = %d, window = %q })", width, height, addr)
}

func dspCenter(addr string) string {
	return fmt.Sprintf("hl.dsp.window.center({ window = %q })", addr)
}

func dspTag(tag, addr string) string {
	return fmt.Sprintf("hl.dsp.window.tag({ tag = %q, window = %q })", tag, addr)
}

func dspSetProp(prop, value, addr string) string {
	return fmt.Sprintf("hl.dsp.window.set_prop({ prop = %q, value = %q, window = %q })", prop, value, addr)
}

func BuildCommands(win hyprland.Client, position Position) []string {
	var (
		commands = make([]string, 0, 18)
		addr     = fmt.Sprintf("address:%s", NormalizeAddress(win.Address))
	)
	if len(win.Grouped) > 0 {
		commands = append(commands, dspMoveOutOfGroup(addr))
	}
	commands = append(commands,
		dspPin(addr),
		dspFloat(addr),
		dspMove(position, addr),
		"[[BATCH]]"+dspSetProp("rounding", "1", addr),
		"[[BATCH]]"+dspSetProp("no_max_size", "0", addr),
		"[[BATCH]]"+dspSetProp("opaque", "toggle", addr),
		"[[BATCH]]"+dspSetProp("immediate", "unset", addr),
		"[[BATCH]]"+dspSetProp("border_size", "relative -2", addr),
		"[[BATCH]]"+dspSetProp("rounding_power", "relative 0.1", addr),
		"[[BATCH]]"+dspSetProp("decorate", "0", addr),
		"[[BATCH]]"+dspSetProp("no_shadow", "1", addr),
		"[[BATCH]]"+dspSetProp("opacity", "1.0 1.0", addr),
		"[[BATCH]]"+dspSetProp("no_blur", "1", addr),
		dspTag("+meeting", addr),
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
			if strings.Contains(msg, "empty response") {
				slog.Debug("empty response, skipping",
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
