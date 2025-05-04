package logging

import (
	"bufio"
	"crypto/sha256" // Added for hashing
	"encoding/hex"  // Added for hex encoding of hash
	"encoding/json" // Added for JSON marshalling/unmarshalling
	"fmt"
	"net" // Added for IP address retrieval
	"os"
	"os/user" // Added for user info
	"path/filepath"
	"secshell/core"
	"secshell/ui/gui"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	UserID      string                 `json:"user_id,omitempty"`      // User ID from OS
	UserName    string                 `json:"user_name,omitempty"`    // User Name from OS
	Hostname    string                 `json:"hostname,omitempty"`     // Hostname from OS
	PID         int                    `json:"pid,omitempty"`          // Process ID
	IPAddresses []string               `json:"ip_addresses,omitempty"` // Local IP Addresses
	Message     string                 `json:"message"`
	ExitCode    int                    `json:"exit_code,omitempty"` // Use omitempty for cleaner JSON
	ErrorDetail string                 `json:"error_detail,omitempty"`
	Hash        string                 `json:"hash,omitempty"`       // Hash of the entry content (excluding the hash itself)
	OtherData   map[string]interface{} `json:"other_data,omitempty"` // For future flexibility
}

var (
	logFileName    = ".secshell_audit.log"
	oldLogFileName = ".secshell.log"
	LogFile        = filepath.Join(core.GetExecutablePath(), logFileName)
	oldLogFile     = filepath.Join(core.GetExecutablePath(), oldLogFileName)
)

// init runs once when the package is loaded.
// It now checks for the *old* log file (.secshell.log) and backs it up
// if it exists and is not in the expected JSON format, allowing the new
// log file (secshell_audit.log) to start fresh with the hashed format.
func init() {
	fileInfo, err := os.Stat(oldLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Old log file doesn't exist, nothing to do regarding backup.
			return
		}
		// Other error (e.g., permissions)
		fmt.Fprintf(os.Stderr, "[Logging Init] Error checking old log file status %s: %v\n", oldLogFile, err)
		return
	}

	// Old file exists, check if it's empty
	if fileInfo.Size() == 0 {
		// Empty old file, safe to remove.
		fmt.Fprintf(os.Stderr, "[Logging Init] Removing empty old log file %s\n", oldLogFile)
		_ = os.Remove(oldLogFile) // Attempt removal, ignore error
		return
	}

	// Old file exists and is not empty, try reading the first line to check format
	f, err := os.Open(oldLogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Logging Init] Error opening existing old log file %s for format check: %v\n", oldLogFile, err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		firstLine := scanner.Bytes()
		var entry LogEntry // Use LogEntry to check if it's JSON-like
		// Attempt to unmarshal the first line as JSON
		if err := json.Unmarshal(firstLine, &entry); err != nil {
			// If unmarshalling fails, assume it's the old plain text format.
			backupName := oldLogFile + ".backup" // Changed suffix
			fmt.Fprintf(os.Stderr, "[Logging Init] Detected non-JSON log format in old log file %s. Backing up to %s\n", oldLogFile, backupName)

			// IMPORTANT: Close the file *before* renaming/deleting on some OSes.
			f.Close() // Close explicitly before rename/remove

			// Remove existing backup if it exists
			_ = os.Remove(backupName) // Ignore error if backup doesn't exist

			// Rename the current log file
			errRename := os.Rename(oldLogFile, backupName)
			if errRename != nil {
				fmt.Fprintf(os.Stderr, "[Logging Init] Error renaming old log file %s to %s: %v. Manual cleanup may be required.\n", oldLogFile, backupName, errRename)
				// Do not attempt deletion if rename fails, preserve data.
			}
		} else {
			// It's JSON, but it's the old log file. We want to start fresh with the new name and hashed format.
			// Backup the old JSON log file as well.
			backupName := oldLogFile + ".json.backup"
			fmt.Fprintf(os.Stderr, "[Logging Init] Detected old JSON log file %s. Backing up to %s to start fresh with hashed logs.\n", oldLogFile, backupName)
			f.Close() // Close explicitly
			_ = os.Remove(backupName)
			errRename := os.Rename(oldLogFile, backupName)
			if errRename != nil {
				fmt.Fprintf(os.Stderr, "[Logging Init] Error renaming old JSON log file %s to %s: %v. Manual cleanup may be required.\n", oldLogFile, backupName, errRename)
			}
		}
	} else if err := scanner.Err(); err != nil {
		// Error reading the first line
		fmt.Fprintf(os.Stderr, "[Logging Init] Error reading first line of old log file %s: %v\n", oldLogFile, err)
	}
	// If scanner.Scan() returned false without error, the file was likely empty after stat check,
	// which should have been handled above by removing the empty file.
}

// calculateHash generates a SHA-256 hash for the log entry's content.
// It marshals the entry *without* the hash field to get a canonical representation.
func calculateHash(entry LogEntry) (string, error) {
	// Create a temporary copy to zero out the hash field for canonical representation
	entryToHash := entry
	entryToHash.Hash = "" // Ensure hash field is empty for hashing

	// Marshal to JSON
	payloadBytes, err := json.Marshal(entryToHash)
	if err != nil {
		return "", fmt.Errorf("failed to marshal entry for hashing: %v", err)
	}

	hash := sha256.Sum256(payloadBytes)
	return hex.EncodeToString(hash[:]), nil
}

// Log a command execution
func LogCommand(command string, exitCode int) error {
	entry := LogEntry{
		Timestamp: time.Now().UTC(), // Use UTC for consistency
		Type:      "COMMAND",
		Message:   command,
		ExitCode:  exitCode,
	}
	// User/Host/PID info will be added in saveLog
	return saveLog(entry)
}

// LogError logs an error with proper formatting
func LogError(err error) error {
	if err == nil {
		return nil
	}

	entry := LogEntry{
		Timestamp:   time.Now().UTC(), // Use UTC
		Type:        "ERROR",
		Message:     err.Error(), // Use the error message directly
		ErrorDetail: err.Error(), // Keep detail separate if needed, or combine
		ExitCode:    1,           // Assuming errors imply non-zero exit
	}
	// User/Host/PID info will be added in saveLog
	return saveLog(entry)
}

// LogAlert logs an alert with proper formatting
func LogAlert(alert string) error {
	if alert == "" {
		return nil
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(), // Use UTC
		Type:      "ALERT",
		Message:   alert,
	}
	// User/Host/PID info will be added in saveLog
	return saveLog(entry)
}

// Save log entry to file in JSON format with integrity hash
func saveLog(entry LogEntry) error {
	// Populate OS-specific information automatically
	currentUser, err := user.Current()
	if err == nil {
		entry.UserID = currentUser.Uid
		entry.UserName = currentUser.Username
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Could not get current user info: %v\n", err)
		// Optionally set default values or leave empty
		entry.UserName = "unknown"
	}

	hostname, err := os.Hostname()
	if err == nil {
		entry.Hostname = hostname
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Could not get hostname: %v\n", err)
		entry.Hostname = "unknown"
	}

	entry.PID = os.Getpid()

	// Get local IP addresses
	entry.IPAddresses = getLocalIPs()

	// Calculate the hash before final marshalling (now includes user/host/pid/ips)
	hash, err := calculateHash(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating hash for log entry: %v\n", err)
		entry.Hash = "HASH_ERROR" // Option: log with error marker
	} else {
		entry.Hash = hash
	}

	// Marshal the final entry (including the hash) into JSON
	logLineBytes, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling log entry: %v\n", err)
		return fmt.Errorf("failed to marshal log entry: %v", err)
	}

	// Open file using the new LogFile path
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // More restrictive permissions
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", LogFile, err)
		return fmt.Errorf("failed to open log file %s: %v", LogFile, err)
	}
	defer f.Close()

	// Write the JSON string followed by a newline
	if _, err := f.Write(append(logLineBytes, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to log file %s: %v\n", LogFile, err)
		return fmt.Errorf("failed to write to log file %s: %v", LogFile, err)
	}

	return nil
}

// getLocalIPs retrieves non-loopback IP addresses for the host.
func getLocalIPs() []string {
	ips := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not get interface addresses: %v\n", err)
		return ips // Return empty slice on error
	}

	for _, address := range addrs {
		// Check the address type and if it is not a loopback
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil || ipnet.IP.To16() != nil { // Check if it's IPv4 or IPv6
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

// Get log entries reads the log file, verifies hashes, parses JSON lines, and returns formatted strings
func GetLogEntries() ([]string, error) {
	formattedEntries := []string{}

	f, err := os.Open(LogFile) // Use new LogFile path
	if err != nil {
		if os.IsNotExist(err) {
			return formattedEntries, nil // Return empty slice if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open log file %s: %v", LogFile, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes() // Read bytes for unmarshalling
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Handle potentially corrupted lines
			fmt.Fprintf(os.Stderr, "Warning: Error unmarshalling log line %d: %v, line: %s\n", lineNumber, err, string(line))
			// Append a warning about the corrupted line
			formattedEntries = append(formattedEntries, fmt.Sprintf("[!] Line %d: Unparseable log entry: %s", lineNumber, string(line)))
			continue
		}

		// Verify the hash
		storedHash := entry.Hash
		calculatedHash, hashErr := calculateHash(entry) // Recalculate hash from the entry content

		tamperWarning := ""
		if hashErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error calculating hash for verification on line %d: %v\n", lineNumber, hashErr)
			tamperWarning = "[HASH CALCULATION ERROR]"
		} else if storedHash == "" || storedHash == "HASH_ERROR" {
			tamperWarning = "[HASH MISSING/INVALID]"
		} else if storedHash != calculatedHash {
			fmt.Fprintf(os.Stderr, "Warning: Log integrity check failed on line %d. Stored: %s, Calculated: %s\n", lineNumber, storedHash, calculatedHash)
			tamperWarning = "[TAMPERED?]"
		}

		// Format the parsed entry back into a human-readable string
		var formattedLine strings.Builder
		formattedLine.WriteString(fmt.Sprintf("[%s] %s", entry.Timestamp.Format(time.RFC3339), entry.Type))

		// Add user/host/pid/ip info if available
		contextInfo := []string{}
		if entry.UserName != "" {
			userPart := entry.UserName
			if entry.UserID != "" {
				userPart = fmt.Sprintf("%s(%s)", entry.UserName, entry.UserID)
			}
			contextInfo = append(contextInfo, fmt.Sprintf("User:%s", userPart))
		} else if entry.UserID != "" {
			contextInfo = append(contextInfo, fmt.Sprintf("UserID:%s", entry.UserID))
		}
		if entry.Hostname != "" {
			contextInfo = append(contextInfo, fmt.Sprintf("Host:%s", entry.Hostname))
		}
		if entry.PID > 0 {
			contextInfo = append(contextInfo, fmt.Sprintf("PID:%d", entry.PID))
		}
		if len(entry.IPAddresses) > 0 {
			contextInfo = append(contextInfo, fmt.Sprintf("IPs:[%s]", strings.Join(entry.IPAddresses, ",")))
		}
		if len(contextInfo) > 0 {
			formattedLine.WriteString(fmt.Sprintf(" (%s)", strings.Join(contextInfo, ", ")))
		}

		// Add message/details
		if entry.Message != "" {
			formattedLine.WriteString(fmt.Sprintf(": %s", entry.Message))
		}
		if entry.Type == "COMMAND" || entry.Type == "ERROR" {
			formattedLine.WriteString(fmt.Sprintf(" (Exit: %d)", entry.ExitCode))
		}
		if entry.ErrorDetail != "" && entry.ErrorDetail != entry.Message { // Avoid duplicate info
			formattedLine.WriteString(fmt.Sprintf(" Detail: %s", entry.ErrorDetail))
		}
		if tamperWarning != "" {
			formattedLine.WriteString(fmt.Sprintf(" %s", tamperWarning))
		}

		formattedEntries = append(formattedEntries, formattedLine.String())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading log file %s: %v", LogFile, err)
	}

	return formattedEntries, nil
}

// Print Log prints the log entries to the console
func PrintLog() error {
	entries, err := GetLogEntries()
	if err != nil {
		LogError(fmt.Errorf("failed to get log entries: %w", err)) // Wrap error
		gui.ErrorBox("Failed to read or parse log file. Check stderr for details.")
		return err // Return the error
	}
	if len(entries) == 0 {
		gui.AlertBox("No log entries found in " + logFileName) // Use new name
		return nil
	}
	gui.TitleBox(fmt.Sprintf("Log Entries (%s):", logFileName)) // Show filename
	core.More(entries)
	return nil
}
