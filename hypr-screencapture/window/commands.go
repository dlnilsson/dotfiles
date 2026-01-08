package window

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

func NormalizeAddress(address string) string {
	if !strings.HasPrefix(address, "0x") {
		return "0x" + address
	}
	return address
}

func BuildCommands(address, position string) []string {
	var (
		commands = make([]string, 0, 18)
		addr     = fmt.Sprintf("address:%s", NormalizeAddress(address))
	)

	commands = append(commands,
		fmt.Sprintf("pin %s", addr),
		fmt.Sprintf("setfloating %s", addr),
		fmt.Sprintf("movewindowpixel exact %s,%s", position, addr),
		fmt.Sprintf("setprop %s rounding 1", addr),
		fmt.Sprintf("setprop %s no_anim 1", addr),
		fmt.Sprintf("setprop %s no_max_size 0", addr),
		fmt.Sprintf("setprop %s opaque toggle", addr),
		fmt.Sprintf("setprop %s immediate unset", addr),
		fmt.Sprintf("setprop %s border_size relative -2", addr),
		fmt.Sprintf("setprop %s rounding_power relative 0.1", addr),
		fmt.Sprintf("setprop %s decorate 0", addr),
		fmt.Sprintf("setprop %s no_shadow 1", addr),
		fmt.Sprintf("setprop %s opacity 1.0 1.0", addr),
		fmt.Sprintf("setprop %s no_blur 1", addr),
		fmt.Sprintf("tagwindow +meeting %s", addr),
	)

	return commands
}

func DispatchCommands(c *Client, commands []string) {
	const maxRetries = 5
	for _, cmd := range commands {
		var err error
		for attempt := range maxRetries {
			if _, err = c.Dispatch(cmd); err == nil {
				break
			}
			if strings.Contains(err.Error(), "Window does not qualify to be pinned") {
				slog.Debug("window does not qualify to be pinned, aborting", "command", cmd)
				return
			}
			if !strings.Contains(err.Error(), "Window not found") {
				break
			}
			slog.Debug("window not found, retrying", "command", cmd, "attempt", attempt+1)
			time.Sleep(500 * time.Millisecond)
		}
		if err != nil {
			slog.Error("failed to dispatch command", "command", cmd, "error", err)
		}
	}
}
