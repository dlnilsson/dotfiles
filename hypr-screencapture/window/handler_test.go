package window

import (
	"testing"

	"github.com/thiagokokada/hyprland-go"
)

func TestStarshipSize(t *testing.T) {
	tests := []struct {
		name    string
		monitor hyprland.Monitor
		wantW   int
		wantH   int
	}{
		{
			name: "1920x1080 reference resolution",
			monitor: hyprland.Monitor{
				Width:  1920,
				Height: 1080,
			},
			wantW: 900,
			wantH: 825,
		},
		{
			name: "3440x1440 ultrawide",
			monitor: hyprland.Monitor{
				Width:  3440,
				Height: 1440,
			},
			wantW: 1200,
			wantH: 1100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				gotW, gotH = StarshipSize(tt.monitor)
			)

			if gotW != tt.wantW || gotH != tt.wantH {
				t.Fatalf("StarshipSize(%dx%d) = %d, %d; want %d, %d",
					tt.monitor.Width, tt.monitor.Height,
					gotW, gotH,
					tt.wantW, tt.wantH,
				)
			}
		})
	}
}
