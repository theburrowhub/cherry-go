package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

var (
	logger        *slog.Logger
	dryRun        bool
	verbose       bool
	verbosityLevel int // 0 = normal, 1 = verbose, 2+ = very verbose (shows diffs)
)

// CustomHandler implements a custom slog.Handler with TIMESTAMP [SEVERITY] MSG format
type CustomHandler struct {
	writer io.Writer
	level  slog.Level
	attrs  []slog.Attr
}

// NewCustomHandler creates a new custom handler
func NewCustomHandler(w io.Writer, level slog.Level) *CustomHandler {
	return &CustomHandler{
		writer: w,
		level:  level,
		attrs:  make([]slog.Attr, 0),
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *CustomHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle formats and writes the log record
func (h *CustomHandler) Handle(_ context.Context, r slog.Record) error {
	// Format: TIMESTAMP [SEVERITY] MSG
	timestamp := r.Time.Format("2006/01/02 15:04:05")
	severity := levelString(r.Level)
	
	// Build the message
	msg := fmt.Sprintf("%s [%s] %s", timestamp, severity, r.Message)
	
	// Add source info in verbose mode
	if verbose && r.PC != 0 {
		// Get source file info from PC
		frames := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := frames.Next()
		if frame.File != "" {
			// Get just the filename, not the full path
			parts := strings.Split(frame.File, "/")
			filename := parts[len(parts)-1]
			msg += fmt.Sprintf(" (%s:%d)", filename, frame.Line)
		}
	}
	
	msg += "\n"
	
	_, err := h.writer.Write([]byte(msg))
	return err
}

// WithAttrs returns a new handler with the given attributes
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	
	return &CustomHandler{
		writer: h.writer,
		level:  h.level,
		attrs:  newAttrs,
	}
}

// WithGroup returns a new handler with the given group
func (h *CustomHandler) WithGroup(name string) slog.Handler {
	// For simplicity, we'll ignore groups in this implementation
	return h
}

// levelString converts slog.Level to string
func levelString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Init initializes the structured logger
func Init() {
	// Create custom handler with TIMESTAMP [SEVERITY] MSG format
	handler := NewCustomHandler(os.Stdout, slog.LevelInfo)
	logger = slog.New(handler)
	
	// Set as default logger
	slog.SetDefault(logger)
}

// SetVerbose enables or disables verbose mode
func SetVerbose(enabled bool) {
	verbose = enabled
	if enabled {
		verbosityLevel = 1
	} else {
		verbosityLevel = 0
	}
	
	// Update logger level based on verbose mode
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}
	
	// Create new custom handler with updated level
	handler := NewCustomHandler(os.Stdout, level)
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// SetVerbosityLevel sets the verbosity level (0=normal, 1=verbose, 2+=very verbose with diffs)
func SetVerbosityLevel(level int) {
	verbosityLevel = level
	if level > 0 {
		verbose = true
		var slogLevel slog.Level
		if level >= 2 {
			slogLevel = slog.LevelDebug
		} else {
			slogLevel = slog.LevelDebug
		}
		handler := NewCustomHandler(os.Stdout, slogLevel)
		logger = slog.New(handler)
		slog.SetDefault(logger)
	} else {
		verbose = false
		handler := NewCustomHandler(os.Stdout, slog.LevelInfo)
		logger = slog.New(handler)
		slog.SetDefault(logger)
	}
}

// GetVerbosityLevel returns the current verbosity level
func GetVerbosityLevel() int {
	return verbosityLevel
}

// ShouldShowDiffs returns true if diffs should be shown (verbosity >= 2)
func ShouldShowDiffs() bool {
	return verbosityLevel >= 2
}

// SetDryRun enables or disables dry run mode
func SetDryRun(enabled bool) {
	dryRun = enabled
}

// IsDryRun returns whether dry run mode is enabled
func IsDryRun() bool {
	return dryRun
}

// IsVerbose returns whether verbose mode is enabled
func IsVerbose() bool {
	return verbose
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if len(v) == 0 {
		logger.Info(format)
	} else {
		// Use fmt.Sprintf for printf-style formatting
		logger.Info(fmt.Sprintf(format, v...))
	}
}

// InfoContext logs an info message with context
func InfoContext(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	// Create error handler that outputs to stderr
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelError
	}
	
	errorHandler := NewCustomHandler(os.Stderr, level)
	errorLogger := slog.New(errorHandler)
	
	var message string
	if len(v) == 0 {
		message = format
	} else {
		message = fmt.Sprintf(format, v...)
	}
	
	errorLogger.Error(message)
}

// ErrorContext logs an error message with context
func ErrorContext(msg string, args ...any) {
	// Create error handler that outputs to stderr
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelError
	}
	
	errorHandler := NewCustomHandler(os.Stderr, level)
	errorLogger := slog.New(errorHandler)
	errorLogger.Error(msg, args...)
}

// Warning logs a warning message
func Warning(format string, v ...interface{}) {
	var message string
	if len(v) == 0 {
		message = format
	} else {
		message = fmt.Sprintf(format, v...)
	}
	
	logger.Warn(message)
}

// WarnContext logs a warning message with context
func WarnContext(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Debug logs a debug message (only shown in verbose mode)
func Debug(format string, v ...interface{}) {
	if len(v) == 0 {
		logger.Debug(format)
	} else {
		logger.Debug(fmt.Sprintf(format, v...))
	}
}

// DebugContext logs a debug message with context
func DebugContext(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// DryRunInfo logs a message only in dry run mode
func DryRunInfo(format string, v ...interface{}) {
	if dryRun {
		msg := "[DRY-RUN] " + format
		if len(v) == 0 {
			logger.Info(msg)
		} else {
			logger.Info(fmt.Sprintf(msg, v...))
		}
	}
}

// DryRunInfoContext logs a message with context only in dry run mode
func DryRunInfoContext(msg string, args ...any) {
	if dryRun {
		allArgs := append([]any{slog.String("mode", "DRY-RUN")}, args...)
		logger.Info(msg, allArgs...)
	}
}

// Fatal logs an error message and exits
func Fatal(format string, v ...interface{}) {
	Error(format, v...)
	os.Exit(1)
}

// FatalContext logs an error message with context and exits
func FatalContext(msg string, args ...any) {
	ErrorContext(msg, args...)
	os.Exit(1)
}

// WithContext creates a logger with additional context
func WithContext(args ...any) *slog.Logger {
	return logger.With(args...)
}

// GetLogger returns the current slog.Logger instance
func GetLogger() *slog.Logger {
	return logger
}