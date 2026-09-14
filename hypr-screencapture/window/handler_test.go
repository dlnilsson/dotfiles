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

func TestStarshipWorkspaceWindows(t *testing.T) {
	for _, tt := range []struct {
		name           string
		title          string
		tags           []string
		otherClass     string
		otherWorkspace int
		floating       bool
		wantDispatch   bool
		wantNoFloat    bool
	}{
		{name: "only window on workspace", title: "Bitwarden"},
		{name: "other window on another workspace", title: "Bitwarden", otherClass: "zen", otherWorkspace: 2},
		{name: "different class on same workspace", title: "Bitwarden", otherClass: "zen", otherWorkspace: 1, wantDispatch: true},
		{name: "same class on same workspace", title: "Settings", otherClass: "Standalone", otherWorkspace: 1, wantDispatch: true},
		{name: "unconfigured title", title: "Other", otherClass: "zen", otherWorkspace: 1},
		{name: "excluded tag", title: "Bitwarden", tags: []string{"browser"}, otherClass: "zen", otherWorkspace: 1},
		{name: "browser popup with automatic tag", title: "Sign in - Google Accounts — Zen Browser", tags: []string{"browser*"}, otherClass: "Standalone", otherWorkspace: 1, wantDispatch: true},
		{name: "already floating window", title: "Bitwarden", otherClass: "zen", otherWorkspace: 1, floating: true, wantDispatch: true, wantNoFloat: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "hypr.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			client := hyprland.Client{Address: "0x123", Class: "Standalone", Title: tt.title, Tags: tt.tags, Floating: tt.floating}
			client.Workspace.Id = 1
			clients := []hyprland.Client{client}
			workspaces := []hyprland.Workspace{{Windows: 1}}
			workspaces[0].Id = 1
			if tt.otherClass != "" {
				other := hyprland.Client{Address: "0x456", Class: tt.otherClass}
				other.Workspace.Id = tt.otherWorkspace
				clients = append(clients, other)
				if tt.otherWorkspace == 1 {
					workspaces[0].Windows++
				} else {
					otherWorkspace := hyprland.Workspace{Windows: 1}
					otherWorkspace.Id = tt.otherWorkspace
					workspaces = append(workspaces, otherWorkspace)
				}
			}
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
						payload = clients
					case "j/workspaces":
						payload = workspaces
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
			actions := []string{"tag", "resize", "center"}
			if !tt.wantNoFloat {
				actions = append(actions, "float")
			} else if strings.Contains(strings.Join(dispatches, "\n"), "hl.dsp.window.float(") {
				t.Errorf("unexpected float action for floating window: %v", dispatches)
			}
			for _, action := range actions {
				if !strings.Contains(strings.Join(dispatches, "\n"), "hl.dsp.window."+action+"(") {
					t.Errorf("missing %s action: %v", action, dispatches)
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

func TestNeedsMonitorRecovery(t *testing.T) {
	tests := []struct {
		name     string
		monitors []hyprland.Monitor
		want     bool
	}{
		{
			name: "all monitors disabled",
			monitors: []hyprland.Monitor{
				{Id: 0, Name: "DP-1", Disabled: true},
				{Id: 1, Name: "eDP-1", Disabled: true},
			},
			want: true,
		},
		{
			name: "monitor still enabled",
			monitors: []hyprland.Monitor{
				{Name: "DP-1", Disabled: false},
				{Name: "eDP-1", Disabled: true},
			},
		},
		{
			name: "Hyprland fallback is the only enabled monitor",
			monitors: []hyprland.Monitor{
				{Name: "eDP-1", Disabled: true},
				{Name: "FALLBACK", Disabled: false},
			},
			want: true,
		},
		{name: "only Hyprland fallback returned", monitors: []hyprland.Monitor{{Name: "FALLBACK"}}},
		{name: "no monitors returned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsMonitorRecovery(tt.monitors)
			if got != tt.want {
				t.Fatalf("needsMonitorRecovery() = %t; want %t", got, tt.want)
			}
		})
	}
}

func TestShouldEnableAddedMonitor(t *testing.T) {
	tests := []struct {
		name    string
		monitor hyprland.Monitor
		want    bool
	}{
		{
			name:    "external monitor has no active mode",
			monitor: hyprland.Monitor{Name: "DP-2", Width: 0, Height: 0},
			want:    true,
		},
		{
			name:    "external monitor is disabled",
			monitor: hyprland.Monitor{Name: "DP-2", Width: 3440, Height: 1440, Disabled: true},
			want:    true,
		},
		{
			name:    "external monitor is already active",
			monitor: hyprland.Monitor{Name: "DP-2", Width: 3440, Height: 1440},
		},
		{
			name:    "internal monitor",
			monitor: hyprland.Monitor{Name: "eDP-1", Width: 0, Height: 0},
		},
		{
			name:    "Hyprland fallback",
			monitor: hyprland.Monitor{Name: "FALLBACK", Width: 0, Height: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldEnableAddedMonitor(tt.monitor)
			if got != tt.want {
				t.Fatalf("shouldEnableAddedMonitor() = %t; want %t", got, tt.want)
			}
		})
	}
}
