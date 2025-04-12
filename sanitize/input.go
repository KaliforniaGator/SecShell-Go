package sanitize

import (
	"strings"
)

// Input sanitizes shell input with configurable character restrictions
func Input(input string, allowSpecialChars bool) string {
	forbidden := ";`\\"
	if !allowSpecialChars {
		forbidden += "&|><${}[]()\"'"
	}

	// Remove all forbidden characters
	for _, char := range forbidden {
		input = strings.ReplaceAll(input, string(char), "")
	}
	return input
}

// Command sanitizes a command string with strict character restrictions
func Command(cmd string) string {
	// Only allow alphanumeric characters, dashes, and underscores
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, cmd)
}

// Path sanitizes a file path string
func Path(path string) string {
	// Remove common shell injection characters and normalize path
	forbidden := ";`\\{}[]()\"'"
	for _, char := range forbidden {
		path = strings.ReplaceAll(path, string(char), "")
	}
	return path
}
