package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// ANSI color codes
const (
	colorGreen = "\033[32m"
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
)

// LogPrefixEnvVar is the environment variable name for customizing the log prefix.
const LogPrefixEnvVar = "REPUTER_LOG_PREFIX"

// getLogPrefix returns the log prefix from env var or default "cli".
func getLogPrefix() string {
	if prefix := os.Getenv(LogPrefixEnvVar); prefix != "" {
		return prefix
	}
	return "cli"
}

// CLIHandler is a custom slog.Handler for CLI output.
// It formats log messages in a user-friendly way:
// - Non-error messages: just the message text
// - Error messages: message text in red
type CLIHandler struct {
	writer io.Writer
	level  slog.Level
}

// NewCLIHandler creates a new CLI handler that writes to the given writer.
func NewCLIHandler(w io.Writer, level slog.Level) *CLIHandler {
	return &CLIHandler{
		writer: w,
		level:  level,
	}
}

// Enabled returns true if the handler handles records at the given level.
func (h *CLIHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes the log record with attributes.
func (h *CLIHandler) Handle(_ context.Context, r slog.Record) error {
	msg := "[" + getLogPrefix() + "] " + r.Message

	// Append attributes as key=value pairs
	if r.NumAttrs() > 0 {
		var attrs []string
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
			return true
		})
		if len(attrs) > 0 {
			msg = msg + ": " + strings.Join(attrs, " ")
		}
	}

	// Add color for error messages and success messages
	if r.Level >= slog.LevelError {
		msg = colorRed + msg + colorReset
	} else {
		msg = colorGreen + msg + colorReset
	}

	_, err := fmt.Fprintln(h.writer, msg)
	return err
}

// WithAttrs returns a new handler with the given attributes.
// For CLI handler, we ignore attributes for simplicity.
func (h *CLIHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

// WithGroup returns a new handler with the given group.
// For CLI handler, we ignore groups for simplicity.
func (h *CLIHandler) WithGroup(_ string) slog.Handler {
	return h
}

// NewCLILogger creates a new logger with CLI-friendly output format.
// This logger outputs minimal, user-friendly messages:
// - Normal messages: just the message text
// - Error messages: message text in red color
// Parameters:
//   - level: The log level as a string (e.g., "debug", "info", "warn", "error").
//
// Returns:
//   - *slog.Logger: A pointer to the configured slog.Logger instance with CLI handler.
func NewCLILogger(level string) *slog.Logger {
	lev := ParseLogLevel(level)
	handler := NewCLIHandler(os.Stderr, lev)
	return slog.New(handler)
}

// SetDefaultCLILogger initializes the CLI logger with the appropriate log level
// and sets it as the default logger.
// Parameters:
//   - level: The log level as a string (e.g., "debug", "info", "warn", "error").
func SetDefaultCLILogger(level string) {
	slog.SetDefault(NewCLILogger(level))
}
