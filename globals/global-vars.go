package globals

import (
	"path/filepath"
	"secshell/core"
)

var ConfigDir = core.GetExecutablePath()
var BlacklistPath = filepath.Join(ConfigDir, ".blacklist")
var WhitelistPath = filepath.Join(ConfigDir, ".whitelist")
var VersionPath = filepath.Join(ConfigDir, ".ver")
var HistoryPath = filepath.Join(ConfigDir, ".history")
var LogFilePath = filepath.Join(ConfigDir, ".secshell_audit.log")

// Define a list of built-in commands
var BuiltInCommands = []string{
	//Regular commands
	"allowed", "help", "exit", "logs", "more", "services", "jobs", "cd", "history", "export", "env", "unset",
	"reload-blacklist", "blacklist", "edit-blacklist", "whitelist", "edit-whitelist",
	"reload-whitelist", "download", "time", "date", "--version", "--update",
	// Add pentesting commands
	"portscan", "hostscan", "webscan", "payload", "session",
	// Tools
	"./", "base64", "hex", "urlencode", "hash", "extract-strings", "binary"}

var TrustedDirs = []string{"/usr/bin/", "/bin/", "/opt/", "/usr/local/bin/"}

// List of commands that require admin privileges
var RestrictedCommands = map[string]bool{
	"exit":             true,
	"logs":             true,
	"export":           true,
	"unset":            true,
	"edit-blacklist":   true,
	"edit-whitelist":   true,
	"reload-blacklist": true,
	"reload-whitelist": true,
	"toggle-security":  true,
	"portscan":         true,
	"hostscan":         true,
	"webscan":          true,
	"payload":          true,
	"session":          true,
}

// isCommandAllowed checks if a command should be visible to non-admin users
func IsCommandAllowed(cmd string) bool {
	return !RestrictedCommands[cmd]
}
