package logger

import (
	"log/slog"
	"os"
)

// NewJSONLogger creates and returns a new slog.Logger that logs in JSON format to stdout.
// The logger includes source information (file and line number) in the log entries.
func NewJSONLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
}
