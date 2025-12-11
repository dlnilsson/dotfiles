package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type splitHandler struct {
	stdout slog.Handler
	stderr slog.Handler
}

func (h *splitHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level >= slog.LevelWarn {
		return h.stderr.Enabled(ctx, level)
	}
	return h.stdout.Enabled(ctx, level)
}

func (h *splitHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		return h.stderr.Handle(ctx, r)
	}
	return h.stdout.Handle(ctx, r)
}

func (h *splitHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &splitHandler{
		stdout: h.stdout.WithAttrs(attrs),
		stderr: h.stderr.WithAttrs(attrs),
	}
}

func (h *splitHandler) WithGroup(name string) slog.Handler {
	return &splitHandler{
		stdout: h.stdout.WithGroup(name),
		stderr: h.stderr.WithGroup(name),
	}
}

func Init(level slog.Level) {
	dropDateTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: dropDateTime,
	})
	stderrHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelWarn,
		ReplaceAttr: dropDateTime,
	})

	logger := slog.New(&splitHandler{
		stdout: stdoutHandler,
		stderr: stderrHandler,
	})

	slog.SetDefault(logger)
}

func ParseLogLevel(levelStr string) (slog.Level, error) {
	levelUpper := strings.ToUpper(levelStr)
	switch levelUpper {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelDebug, fmt.Errorf("invalid log level: %q (must be one of: DEBUG, INFO, WARN, ERROR)", levelStr)
	}
}
