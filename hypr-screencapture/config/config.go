package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Paths         PathsConfig
	Processes     ProcessesConfig
	Notifications NotificationsConfig
	PinWindow     PinWindowConfig
	WindowMatching WindowMatchingConfig
	Positioning   PositioningConfig
}

type PathsConfig struct {
	StatusFile string
}

type ProcessesConfig struct {
	Monitor      []string
	PollInterval time.Duration
}

type NotificationsConfig struct {
	Cooldown time.Duration
}

type PinWindowConfig struct {
	InitialDelay  time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
	MaxRetryDelay time.Duration
	PollInterval  time.Duration
	MaxPollTime   time.Duration
}

type WindowMatchingConfig struct {
	BitwardenTitles    []string
	MeetTitlePrefixes  []string
	ScreencastKeywords []string
}

type PositioningConfig struct {
	DefaultPosition string
	RightPadding    int
	YPosition       int
}

func Default() *Config {
	return &Config{
		Paths: PathsConfig{
			StatusFile: filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "hypr", "screencast.status"),
		},
		Processes: ProcessesConfig{
			Monitor:      []string{"wl-screenrec"},
			PollInterval: 2 * time.Second,
		},
		Notifications: NotificationsConfig{
			Cooldown: 10 * time.Second,
		},
		PinWindow: PinWindowConfig{
			InitialDelay:  500 * time.Millisecond,
			MaxRetries:    3,
			RetryDelay:    500 * time.Millisecond,
			MaxRetryDelay: 2 * time.Second,
			PollInterval:  500 * time.Millisecond,
			MaxPollTime:   10 * time.Second,
		},
		WindowMatching: WindowMatchingConfig{
			BitwardenTitles: []string{
				"Extension: (Bitwarden Password Manager) - Bitwarden — Zen Browser",
				"Extension: (Bitwarden Password Manager) - Bitwarden — Mozilla Firefox",
				"Bitwarden",
			},
			MeetTitlePrefixes: []string{
				"https://meet.google.com - Meet –",
				"Meet – ",
			},
			ScreencastKeywords: []string{
				"sharing indicator", "sharing", "screen", "recording", "capture",
				"cast", "obs", "streamlabs", "discord", "zoom", "teams", "meet",
				"chrome", "firefox", "browser", "portal", "slack", "zen-browser",
			},
		},
		Positioning: PositioningConfig{
			DefaultPosition: "74% 2%",
			RightPadding:    2,
			YPosition:       20,
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return cfg, fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir := filepath.Join(homeDir, ".config", "hypr-screencapture")
		path = filepath.Join(configDir, "hypr-screencapture.conf")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	parser := newINIParser(file)
	if err := parser.parse(cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

type iniParser struct {
	scanner *bufio.Scanner
	section string
	values  map[string][]string
}

func newINIParser(file *os.File) *iniParser {
	return &iniParser{
		scanner: bufio.NewScanner(file),
		values:  make(map[string][]string),
	}
}

func (p *iniParser) parse(cfg *Config) error {
	for p.scanner.Scan() {
		line := strings.TrimSpace(p.scanner.Text())

		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			p.section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = os.ExpandEnv(value)

		fullKey := p.section + "." + key
		p.values[fullKey] = append(p.values[fullKey], value)
	}

	if err := p.scanner.Err(); err != nil {
		return err
	}

	return p.apply(cfg)
}

func (p *iniParser) apply(cfg *Config) error {
	if err := p.applyPaths(cfg); err != nil {
		return err
	}
	if err := p.applyProcesses(cfg); err != nil {
		return err
	}
	if err := p.applyNotifications(cfg); err != nil {
		return err
	}
	if err := p.applyPinWindow(cfg); err != nil {
		return err
	}
	if err := p.applyWindowMatching(cfg); err != nil {
		return err
	}
	if err := p.applyPositioning(cfg); err != nil {
		return err
	}
	return nil
}

func (p *iniParser) getString(section, key string) string {
	fullKey := section + "." + key
	vals := p.values[fullKey]
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (p *iniParser) getStringArray(section, key string) []string {
	fullKey := section + "." + key
	vals := p.values[fullKey]
	if len(vals) == 0 {
		return nil
	}
	return vals
}

func (p *iniParser) getDuration(section, key string) (time.Duration, error) {
	val := p.getString(section, key)
	if val == "" {
		return 0, nil
	}
	return time.ParseDuration(val)
}

func (p *iniParser) getInt(section, key string) (int, error) {
	val := p.getString(section, key)
	if val == "" {
		return 0, nil
	}
	return strconv.Atoi(val)
}

func (p *iniParser) applyPaths(cfg *Config) error {
	if val := p.getString("paths", "status_file"); val != "" {
		cfg.Paths.StatusFile = val
	}
	return nil
}

func (p *iniParser) applyProcesses(cfg *Config) error {
	if vals := p.getStringArray("processes", "monitor[]"); len(vals) > 0 {
		cfg.Processes.Monitor = vals
	}
	if val, err := p.getDuration("processes", "poll_interval"); err != nil {
		return fmt.Errorf("invalid poll_interval: %w", err)
	} else if val > 0 {
		cfg.Processes.PollInterval = val
	}
	return nil
}

func (p *iniParser) applyNotifications(cfg *Config) error {
	if val, err := p.getDuration("notifications", "cooldown"); err != nil {
		return fmt.Errorf("invalid cooldown: %w", err)
	} else if val > 0 {
		cfg.Notifications.Cooldown = val
	}
	return nil
}

func (p *iniParser) applyPinWindow(cfg *Config) error {
	if val, err := p.getDuration("pin_window", "initial_delay"); err != nil {
		return fmt.Errorf("invalid initial_delay: %w", err)
	} else if val > 0 {
		cfg.PinWindow.InitialDelay = val
	}
	if val, err := p.getInt("pin_window", "max_retries"); err != nil {
		return fmt.Errorf("invalid max_retries: %w", err)
	} else if val > 0 {
		cfg.PinWindow.MaxRetries = val
	}
	if val, err := p.getDuration("pin_window", "retry_delay"); err != nil {
		return fmt.Errorf("invalid retry_delay: %w", err)
	} else if val > 0 {
		cfg.PinWindow.RetryDelay = val
	}
	if val, err := p.getDuration("pin_window", "max_retry_delay"); err != nil {
		return fmt.Errorf("invalid max_retry_delay: %w", err)
	} else if val > 0 {
		cfg.PinWindow.MaxRetryDelay = val
	}
	if val, err := p.getDuration("pin_window", "poll_interval"); err != nil {
		return fmt.Errorf("invalid poll_interval: %w", err)
	} else if val > 0 {
		cfg.PinWindow.PollInterval = val
	}
	if val, err := p.getDuration("pin_window", "max_poll_time"); err != nil {
		return fmt.Errorf("invalid max_poll_time: %w", err)
	} else if val > 0 {
		cfg.PinWindow.MaxPollTime = val
	}
	return nil
}

func (p *iniParser) applyWindowMatching(cfg *Config) error {
	if vals := p.getStringArray("window_matching", "bitwarden_titles[]"); len(vals) > 0 {
		cfg.WindowMatching.BitwardenTitles = vals
	}
	if vals := p.getStringArray("window_matching", "meet_title_prefixes[]"); len(vals) > 0 {
		cfg.WindowMatching.MeetTitlePrefixes = vals
	}
	if vals := p.getStringArray("window_matching", "screencast_keywords[]"); len(vals) > 0 {
		cfg.WindowMatching.ScreencastKeywords = vals
	}
	return nil
}

func (p *iniParser) applyPositioning(cfg *Config) error {
	if val := p.getString("positioning", "default_position"); val != "" {
		cfg.Positioning.DefaultPosition = val
	}
	if val, err := p.getInt("positioning", "right_padding"); err != nil {
		return fmt.Errorf("invalid right_padding: %w", err)
	} else if val > 0 {
		cfg.Positioning.RightPadding = val
	}
	if val, err := p.getInt("positioning", "y_position"); err != nil {
		return fmt.Errorf("invalid y_position: %w", err)
	} else if val > 0 {
		cfg.Positioning.YPosition = val
	}
	return nil
}

func EnsureConfigDir() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "hypr-screencapture")
	return os.MkdirAll(configDir, 0o755)
}

