package sanitize

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	maxURLLength   = 2083 // Maximum URL length (IE limit, widely accepted)
	maxFileNameLen = 255  // Maximum filename length for most filesystems
)

var (
	// Blocked file extensions that could be malicious
	blockedExtensions = []string{
		".exe", ".dll", ".so", ".dylib", ".bat", ".cmd", ".sh",
		".com", ".bin", ".ps1", ".msi", ".vbs", ".jar",
	}
)

// SanitizeURL checks and sanitizes the URL for secure downloads
func SanitizeURL(rawURL string) (string, error) {
	// Check URL length
	if len(rawURL) == 0 {
		return "", fmt.Errorf("empty URL")
	}
	if len(rawURL) > maxURLLength {
		return "", fmt.Errorf("URL exceeds maximum length of %d characters", maxURLLength)
	}

	// Parse and validate URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %v", err)
	}

	// Verify scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
	}

	// Check for empty or invalid hostname
	if u.Host == "" {
		return "", fmt.Errorf("missing or invalid hostname")
	}

	// Prevent path traversal attempts
	cleanPath := filepath.Clean(u.Path)
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("path traversal detected")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(cleanPath))
	for _, blocked := range blockedExtensions {
		if ext == blocked {
			return "", fmt.Errorf("blocked file extension: %s", ext)
		}
	}

	// Normalize the URL
	normalized := u.String()

	return normalized, nil
}

// SanitizeFileName ensures the filename is safe for the filesystem
func SanitizeFileName(name string) (string, error) {
	if len(name) == 0 {
		return "", fmt.Errorf("empty filename")
	}
	if len(name) > maxFileNameLen {
		return "", fmt.Errorf("filename exceeds maximum length of %d characters", maxFileNameLen)
	}

	// Remove any directory traversal attempts
	name = filepath.Base(name)

	// Remove or replace potentially dangerous characters
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)

	return name, nil
}
