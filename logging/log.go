package logging

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"secshell/core"
	"secshell/drawbox"
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

// Get log entries reads the log file and returns each line as a string in a slice
func GetLogEntries() ([]string, error) {
	LogEntries := []string{}

	// Open the log file for reading
	f, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return empty slice with no error
			return LogEntries, nil
		}
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		LogEntries = append(LogEntries, scanner.Text())
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file: %v", err)
	}

	return LogEntries, nil
}

// Print Log prints the log entries to the console
func PrintLog() error {
	entries, err := GetLogEntries()
	if err != nil {
		LogError(err)
		drawbox.PrintError("Failed to read log file")
	}
	if len(entries) == 0 {
		drawbox.PrintAlert("No log entries found")
		return nil
	}
	drawbox.RunDrawbox("Log Entries:", "bold_white")
	core.More(entries)
	return nil
}

// ClearLog truncates the log file, removing all contents
// Only works if the isAdmin parameter is set to true
func ClearLog(isAdmin bool) error {
	if !isAdmin {
		LogAlert("Insufficient permissions to clear logs")
		drawbox.PrintError("Insufficient permissions to clear logs")
		return fmt.Errorf("insufficient permissions: admin privileges required to clear logs")
	}

	f, err := os.OpenFile(logFile, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		LogError(err)
		return fmt.Errorf("failed to open log file for clearing: %v", err)
	}
	defer f.Close()

	return nil
}
