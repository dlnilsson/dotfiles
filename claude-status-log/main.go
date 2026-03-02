package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS status (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    recorded_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f', 'now')),
    session_id  TEXT NOT NULL,
    raw         JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_status_session_id ON status (session_id);
CREATE INDEX IF NOT EXISTS idx_status_recorded_at ON status (recorded_at);
`

func run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var msg struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}
	if msg.SessionID == "" {
		return fmt.Errorf("missing session_id")
	}

	dbPath, err := dbPath()
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("setting %s: %w", pragma, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	if _, err := db.Exec(
		"INSERT INTO status (session_id, raw) VALUES (?, jsonb(?))",
		msg.SessionID, string(data),
	); err != nil {
		return fmt.Errorf("inserting row: %w", err)
	}

	return nil
}

func dbPath() (string, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	dir := filepath.Join(dataHome, "claude-status-log")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating data directory: %w", err)
	}

	return filepath.Join(dir, "status.db"), nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "claude-status-log: %v\n", err)
		os.Exit(1)
	}
}
