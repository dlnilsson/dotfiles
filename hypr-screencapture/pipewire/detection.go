package pipewire

import (
	"strings"
)

func IsCameraNode(mediaClass, name, descLower string) bool {
	if strings.HasPrefix(name, "v4l2_input.") {
		// slog.Debug("isCameraNode: v4l2_input prefix", "name", name)
		return true
	}
	if strings.Contains(descLower, "v4l2") || strings.Contains(descLower, "camera") || strings.Contains(descLower, "webcam") {
		// slog.Debug("isCameraNode: description match", "name", name, "desc", descLower)
		return true
	}
	if mediaClass == "Video/Source" {
		if strings.Contains(name, "v4l2") || strings.Contains(name, "camera") || strings.Contains(name, "webcam") {
			// slog.Debug("isCameraNode: Video/Source with camera name", "name", name)
			return true
		}
		if !strings.Contains(descLower, "screen") && !strings.Contains(descLower, "display") &&
			!strings.Contains(descLower, "monitor") && name != "xdg-desktop-portal-hyprland" &&
			name != "xdg-desktop-portal-wlr" && name != "xdg-desktop-portal-gnome" &&
			name != "xdg-desktop-portal-kde" && descLower != "" {
			// slog.Debug("isCameraNode: Video/Source (potential camera)", "name", name, "desc", descLower)
			return true
		}
	}
	return false
}

func IsScreenshareNode(mc, name, descLower, appLower string) bool {
	if IsCameraNode(mc, name, descLower) {
		return false
	}
	// slog.Debug("isScreenshareNode", "mediaclass", mc, "name", name, "descLower", descLower, "appLower", appLower)
	if mc == "Video/Source" && (name == "xdg-desktop-portal-hyprland" ||
		name == "xdg-desktop-portal-wlr" ||
		name == "xdg-desktop-portal-gnome" ||
		name == "xdg-desktop-portal-kde") {
		return true
	}
	if mc == "Stream/Input/Video" && appLower != "" && appLower != "pipewire" {
		return true
	}
	if mc == "Video/Source" &&
		!strings.Contains(descLower, "camera") &&
		!strings.HasPrefix(name, "v4l2_input.") {
		return true
	}
	return false
}
