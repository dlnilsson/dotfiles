package manager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
)

const (
	exUnavailable = 69
	exIOErr       = 74
)

func StartSocketServer(ctx context.Context, socketPath string, socketStatus *SocketStatus) error {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		return fmt.Errorf("could not create directory for socket: %w", err)
	}

	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("could not remove stale socket: %w", err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("could not listen on socket: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err := listener.Close(); err != nil {
			slog.Error("failed to close listener", "error", err)
		}
		if err := os.Remove(socketPath); err != nil {
			slog.Error("failed to remove socket", "error", err)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					slog.Error("failed to accept connection", "error", err)
					continue
				}
			}

			go func(c net.Conn) {
				defer func() {
					if err := c.Close(); err != nil {
						slog.Error("failed to close connection", "error", err)
					}
				}()

				socketStatus.mu.RLock()
				status := socketStatus.status
				socketStatus.mu.RUnlock()

				if _, err := fmt.Fprintln(c, status); err != nil {
					slog.Error("failed to write status to client", "error", err)
					return
				}

				if unixConn, ok := c.(*net.UnixConn); ok {
					if err := unixConn.CloseWrite(); err != nil {
						slog.Error("failed to close write", "error", err)
					}
				}
			}(conn)
		}
	}()

	return nil
}

func RunStatusCommand(cfg *config.Config) (string, int, error) {
	socketPath := cfg.Paths.SocketPath

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return "", exUnavailable, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close connection: %v\n", err)
		}
	}()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, conn); err != nil && err != io.EOF {
		return "", exIOErr, err
	}

	return strings.TrimSpace(buf.String()), 0, nil
}

func CheckServerRunning(socketPath string) error {
	if _, err := os.Stat(socketPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("stale socket exists and could not be removed: %w", err)
		}
		return nil
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return errors.New("another instance of hypr-screencapture is already running")
}
