package window

import (
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
	"github.com/thiagokokada/hyprland-go"
	"github.com/thiagokokada/hyprland-go/event"
)

func TestStarshipStandaloneWindow(t *testing.T) {
	for _, tt := range []struct {
		name         string
		title        string
		tags         []string
		wantDispatch bool
	}{
		{name: "configured standalone window", title: "Bitwarden", wantDispatch: true},
		{name: "another configured title", title: "Settings", wantDispatch: true},
		{name: "unconfigured title", title: "Other"},
		{name: "excluded tag", title: "Bitwarden", tags: []string{"browser"}},
		{name: "browser popup with automatic tag", title: "Sign in - Google Accounts — Zen Browser", tags: []string{"browser*"}, wantDispatch: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "hypr.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			client := hyprland.Client{Address: "0x123", Class: "Standalone", Title: tt.title, Tags: tt.tags}
			var requests []string
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					buf := make([]byte, 8192)
					n, err := conn.Read(buf)
					if err != nil {
						conn.Close()
						return
					}
					request := string(buf[:n])
					requests = append(requests, request)
					var payload any
					switch request {
					case "j/activewindow":
						payload = client
					case "j/clients":
						payload = []hyprland.Client{client}
					case "j/workspaces":
						payload = []hyprland.Workspace{{Windows: 1}}
					case "j/monitors":
						payload = []hyprland.Monitor{{Width: 1920, Height: 1080}}
					default:
						conn.Write([]byte("ok"))
						conn.Close()
						continue
					}
					json.NewEncoder(conn).Encode(payload)
					conn.Close()
				}
			}()
			t.Cleanup(func() { listener.Close(); <-done })
			cfg := &config.Config{WindowMatching: config.WindowMatchingConfig{
				StarshipTitles:      []string{"Bitwarden", "Settings", "Sign in - Google Accounts — Zen Browser"},
				ExcludeFromStarship: []string{"browser"},
			}}
			handler := NewScreencastHandler(nil, NewClient(hyprland.NewClient(socket)), func() *config.Config { return cfg }, nil)
			handler.ActiveWindow(event.ActiveWindow{Title: tt.title})
			listener.Close()
			<-done
			var dispatches []string
			for _, request := range requests {
				if strings.Contains(request, "dispatch ") {
					dispatches = append(dispatches, request)
				}
			}
			if !tt.wantDispatch {
				if len(dispatches) != 0 {
					t.Fatalf("unexpected actions: %v", dispatches)
				}
				return
			}
			for _, action := range []string{"tag", "float", "resize", "center"} {
				if !strings.Contains(strings.Join(dispatches, "\n"), "hl.dsp.window."+action+"(") {
					t.Errorf("missing %s action for standalone window: %v", action, dispatches)
				}
			}
		})
	}
}

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
