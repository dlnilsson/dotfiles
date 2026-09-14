package window

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/thiagokokada/hyprland-go"
)

func TestEnableMonitor(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{name: "success", response: "ok"},
		{name: "Hyprland rejects expression", response: "error: invalid expression", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "hypr.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()

			requestCh := make(chan string, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				request := make([]byte, 4096)
				n, err := conn.Read(request)
				if err != nil {
					return
				}
				requestCh <- string(request[:n])
				_, _ = conn.Write([]byte(tt.response))
			}()

			client := NewClient(hyprland.NewClient(socket))
			_, err = client.EnableMonitor("eDP-1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("EnableMonitor() error = %v; want error %t", err, tt.wantErr)
			}

			const wantRequest = `eval hl.monitor({ output = "eDP-1", mode = "highres", position = "auto", scale = 1.0, disabled = false })`
			if got := <-requestCh; got != wantRequest {
				t.Fatalf("request = %q; want %q", got, wantRequest)
			}
		})
	}
}
