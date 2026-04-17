package logger

import (
	"log/slog"
	"os"
)

// New returns a JSON-formatted slog.Logger writing to stdout.
// level is read from the LOG_LEVEL env var; defaults to INFO.
func New() *slog.Logger {
	level := slog.LevelInfo
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		var l slog.Level
		if err := l.UnmarshalText([]byte(v)); err == nil {
			level = l
		}
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
