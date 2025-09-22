package logger

import (
	"log"
	"os"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	dryRun      bool
)

// Init initializes the logger
func Init() {
	infoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// SetDryRun enables or disables dry run mode
func SetDryRun(enabled bool) {
	dryRun = enabled
}

// IsDryRun returns whether dry run mode is enabled
func IsDryRun() bool {
	return dryRun
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}

// Warning logs a warning message
func Warning(format string, v ...interface{}) {
	errorLogger.Printf("WARNING: "+format, v...)
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	debugLogger.Printf(format, v...)
}

// DryRunInfo logs a message only in dry run mode
func DryRunInfo(format string, v ...interface{}) {
	if dryRun {
		infoLogger.Printf("[DRY-RUN] "+format, v...)
	}
}

// Fatal logs an error message and exits
func Fatal(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
	os.Exit(1)
}
