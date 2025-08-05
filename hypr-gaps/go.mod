module github.com/dlnilsson/dotfiles/hypr-gaps

go 1.24.5

require (
	github.com/thiagokokada/hyprland-go v0.4.1
	golang.org/x/sys v0.34.0
)

// replace github.com/thiagokokada/hyprland-go => /home/dln/private/hyprland-go

replace github.com/thiagokokada/hyprland-go => github.com/dlnilsson/hyprland-go v0.0.0-20250805142704-f8acd29fd377
