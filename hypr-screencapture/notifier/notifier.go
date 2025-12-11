package notifier

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

type State struct {
	mu               sync.Mutex
	count            int
	lastMessageTimes map[string]time.Time
	cooldown         time.Duration
}

func NewState(cooldown time.Duration) *State {
	return &State{
		cooldown: cooldown,
	}
}

func (n *State) ShouldNotify(message string, isInitialOff bool) bool {
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

func (n *State) Count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.count
}

func (n *State) UpdateCooldown(cooldown time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cooldown = cooldown
}

func SendNotification(title, message string, iconBytes []byte) error {
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
		defer func() {
			if err := os.Remove(tmp); err != nil {
				slog.Error("failed to remove temp file", "error", err)
			}
		}()
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
	tmpPath := tmp.Name()
	defer func() {
		if err := tmp.Close(); err != nil {
			slog.Error("failed to close temp file", "error", err)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			slog.Error("failed to remove temp file after write error", "error", removeErr)
		}
		return out, err
	}

	return tmpPath, nil
}

func ToggleNotificationInhibitor(add bool) error {
	if add {
		return exec.Command("swaync-client", "--inhibitor-add", "xdg-desktop-portal-hyprland").Run()
	}
	return exec.Command("swaync-client", "--inhibitor-remove", "xdg-desktop-portal-hyprland").Run()
}
