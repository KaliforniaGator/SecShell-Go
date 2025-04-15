package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"secshell/core"
	"time"
)

type LogEntry struct {
	Timestamp   time.Time
	Type        string
	Message     string
	ExitCode    int
	ErrorDetail string
}

var logFile = filepath.Join(core.GetExecutablePath(), ".secshell.log")

// Log a command execution
func LogCommand(command string, exitCode int) error {
	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      "COMMAND",
		Message:   command,
		ExitCode:  exitCode,
	}
	return saveLog(entry)
}

// LogError logs an error with proper formatting
func LogError(err error) error {
	if err == nil {
		return nil
	}

	entry := LogEntry{
		Timestamp:   time.Now(),
		Type:        "ERROR",
		Message:     err.Error(), // Use the error message directly
		ErrorDetail: err.Error(),
		ExitCode:    1,
	}
	return saveLog(entry)
}

// LogAlert logs an alert with proper formatting
func LogAlert(alert string) error {
	if alert == "" {
		return nil
	}

	entry := LogEntry{
		Timestamp:   time.Now(),
		Type:        "ALERT",
		Message:     alert,
		ExitCode:    0,  // Alerts don't typically have error codes
		ErrorDetail: "", // Alerts don't have error details
	}
	return saveLog(entry)
}

// Save log entry to file
func saveLog(entry LogEntry) error {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	var logLine string
	switch entry.Type {
	case "ERROR":
		logLine = fmt.Sprintf("[%s] %s: %s (Exit: %d) Error: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Type,
			entry.Message,
			entry.ExitCode,
			entry.ErrorDetail)
	case "ALERT":
		logLine = fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Type,
			entry.Message)
	default:
		logLine = fmt.Sprintf("[%s] %s: %s (Exit: %d)\n",
			entry.Timestamp.Format(time.RFC3339),
			entry.Type,
			entry.Message,
			entry.ExitCode)
	}

	if _, err := f.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	return nil
}
