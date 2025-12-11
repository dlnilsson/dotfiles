package window

import (
	"fmt"
	"log/slog"
)

func BuildCommands(address, position string) []string {
	var (
		commands = make([]string, 0, 18)
		addr     = fmt.Sprintf("address:%s", address)
	)

	commands = append(commands,
		fmt.Sprintf("tagwindow +meeting %s", addr),
		fmt.Sprintf("pin %s", addr),
		fmt.Sprintf("setfloating %s", addr),
		fmt.Sprintf("movewindowpixel exact %s,%s", position, addr),
		fmt.Sprintf("setprop %s rounding 1", addr),
		fmt.Sprintf("setprop %s noanim 1", addr),
		fmt.Sprintf("setprop %s nomaxsize 0", addr),
		fmt.Sprintf("setprop %s opaque toggle", addr),
		fmt.Sprintf("setprop %s immediate unset", addr),
		fmt.Sprintf("setprop %s bordersize relative -2", addr),
		fmt.Sprintf("setprop %s roundingpower relative 0.1", addr),
		fmt.Sprintf("setprop %s decorate 0", addr),
		fmt.Sprintf("setprop %s noborder 1", addr),
		fmt.Sprintf("setprop %s noshadow 1", addr),
		fmt.Sprintf("setprop %s opacity 1.0 1.0", addr),
		fmt.Sprintf("setprop %s noblur 1", addr),
	)

	return commands
}

func DispatchCommands(c *Client, commands []string) {
	for _, cmd := range commands {
		if _, err := c.Dispatch(cmd); err != nil {
			slog.Error("failed to dispatch command", "command", cmd, "error", err)
		}
	}
}
