package logger

import (
	"fmt"
	"log/slog"
	"os"
)

var (
	logger  *slog.Logger
	dryRun  bool
	verbose bool
)

// Init initializes the structured logger
func Init() {
	// Create a text handler with custom options
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Default level
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the time format to be more readable
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006/01/02 15:04:05"))
			}
			// Remove source file info by default (can be enabled in verbose mode)
			if a.Key == slog.SourceKey {
				return slog.Attr{}
			}
			return a
		},
	}
	
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
	
	// Set as default logger
	slog.SetDefault(logger)
}

// SetVerbose enables or disables verbose mode
func SetVerbose(enabled bool) {
	verbose = enabled
	
	// Update logger level based on verbose mode
	var level slog.Level
	if verbose {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}
	
	// Create new handler with updated level
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006/01/02 15:04:05"))
			}
			// Show source in verbose mode
			if a.Key == slog.SourceKey && !verbose {
				return slog.Attr{}
			}
			return a
		},
		AddSource: verbose, // Add source file info in verbose mode
	}
	
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger = slog.New(handler)
	slog.SetDefault(logger)
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
	// Log errors to stderr
	errorHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006/01/02 15:04:05"))
			}
			return a
		},
	})
	errorLogger := slog.New(errorHandler)
	
	if len(v) == 0 {
		errorLogger.Error(format)
	} else {
		errorLogger.Error(fmt.Sprintf(format, v...))
	}
}

// ErrorContext logs an error message with context
func ErrorContext(msg string, args ...any) {
	errorHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006/01/02 15:04:05"))
			}
			return a
		},
	})
	errorLogger := slog.New(errorHandler)
	errorLogger.Error(msg, args...)
}

// Warning logs a warning message
func Warning(format string, v ...interface{}) {
	if len(v) == 0 {
		logger.Warn(format)
	} else {
		logger.Warn(fmt.Sprintf(format, v...))
	}
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