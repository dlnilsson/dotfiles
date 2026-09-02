package wallpaper

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dlnilsson/dotfiles/hypr-screencapture/config"
)

var supportedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".jxl":  true,
}

var errNoWallpapers = errors.New("no wallpapers found in directory")

type Rotator struct {
	cfgFn       func() *config.Config
	mu          sync.Mutex
	assignments map[string]string // workspace ID -> wallpaper path
	wallpapers  []string          // cached file list from directory scan
}

func New(cfgFn func() *config.Config) *Rotator {
	return &Rotator{
		cfgFn:       cfgFn,
		assignments: make(map[string]string),
	}
}

func (r *Rotator) Run(ctx context.Context, workspaceCh <-chan string) {
	cfg := r.cfgFn()
	if !cfg.Wallpaper.Enabled {
		slog.Debug("Wallpaper rotation disabled")
		return
	}

	slog.Debug("Wallpaper rotation enabled",
		"dir", cfg.Wallpaper.WallpaperDir,
		"refresh_interval", cfg.Wallpaper.RefreshInterval,
	)

	ticker := time.NewTicker(cfg.Wallpaper.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case workspaceID := <-workspaceCh:
			r.handleWorkspaceChange(workspaceID)
		case <-ticker.C:
			r.handleRefresh()
			newCfg := r.cfgFn()
			if newCfg.Wallpaper.RefreshInterval != cfg.Wallpaper.RefreshInterval {
				ticker.Reset(newCfg.Wallpaper.RefreshInterval)
				cfg = newCfg
			}
			if !newCfg.Wallpaper.Enabled {
				slog.Debug("Wallpaper rotation disabled via config reload")
				return
			}
		}
	}
}

func (r *Rotator) handleWorkspaceChange(workspaceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if wp, exists := r.assignments[workspaceID]; exists {
		slog.Debug("Using cached wallpaper", "workspace", workspaceID, "wallpaper", filepath.Base(wp))
		r.setWallpaper(wp)
		return
	}

	wp, err := r.pickRandom()
	if err != nil {
		slog.Error("Failed to pick random wallpaper", "error", err)
		return
	}

	r.assignments[workspaceID] = wp
	slog.Debug("Assigned new wallpaper", "workspace", workspaceID, "wallpaper", filepath.Base(wp), "cached_workspaces", len(r.assignments))
	r.setWallpaper(wp)
}

func (r *Rotator) handleRefresh() {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Debug("Wallpaper refresh interval reached, invalidating all assignments", "previous_count", len(r.assignments))
	r.assignments = make(map[string]string)
	r.wallpapers = nil

	wp, err := r.pickRandom()
	if err != nil {
		slog.Error("Failed to pick wallpaper on refresh", "error", err)
		return
	}

	r.setWallpaper(wp)
}

func (r *Rotator) scanWallpapers() error {
	cfg := r.cfgFn()
	entries, err := os.ReadDir(cfg.Wallpaper.WallpaperDir)
	if err != nil {
		return fmt.Errorf("failed to read wallpaper directory %s: %w", cfg.Wallpaper.WallpaperDir, err)
	}

	wallpapers := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if supportedExts[ext] {
			wallpapers = append(wallpapers, filepath.Join(cfg.Wallpaper.WallpaperDir, entry.Name()))
		}
	}

	if len(wallpapers) == 0 {
		return errNoWallpapers
	}

	r.wallpapers = wallpapers
	slog.Debug("Scanned wallpaper directory", "count", len(wallpapers), "dir", cfg.Wallpaper.WallpaperDir)
	return nil
}

// pickRandom must be called with r.mu held.
func (r *Rotator) pickRandom() (string, error) {
	if len(r.wallpapers) == 0 {
		if err := r.scanWallpapers(); err != nil {
			return "", err
		}
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(r.wallpapers))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random index: %w", err)
	}
	return r.wallpapers[n.Int64()], nil
}

func (r *Rotator) setWallpaper(path string) {
	arg := fmt.Sprintf(", %s", path)
	cmd := exec.Command("hyprctl", "hyprpaper", "wallpaper", arg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to set wallpaper", "path", path, "error", err, "output", string(output))
		return
	}
	slog.Debug("Set wallpaper", "path", filepath.Base(path), "output", strings.TrimSpace(string(output)))
}
