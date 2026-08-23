// Package logger provides the structured JSON logger used across the bot.
package logger

import (
	"log/slog"
	"os"
)

// New builds a JSON slog.Logger writing to stdout at the given level
// ("debug", "info", "warn" or "error").
func New(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}
