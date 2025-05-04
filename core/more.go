package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"secshell/ui"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// More displays a scrollable list of items in the terminal, paginated by screen height
func More(items []string) error {
	if len(items) == 0 {
		return nil
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Get terminal size
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	size := strings.Split(strings.TrimSpace(string(out)), " ")
	rows, _ := strconv.Atoi(size[0])
	pageSize := rows - 2 // One line for status, one for search/help

	currentPage := 0
	totalPages := (len(items) + pageSize - 1) / pageSize
	reader := bufio.NewReader(os.Stdin)

	// Search related variables
	searchMode := false
	searchQuery := ""
	searchMatches := []int{} // Stores indices of matched items
	currentMatch := -1       // Index in searchMatches
	showHelp := false

	// Function to render current page
	renderPage := func() {
		// Clear screen and move cursor to top
		ui.ClearScreenAndBuffer()
		fmt.Print("\033[H\033[2J")

		// Print current page
		start := currentPage * pageSize
		end := min(start+pageSize, len(items))
		for i := start; i < end; i++ {
			line := items[i]

			// Highlight search matches
			if searchQuery != "" && strings.Contains(strings.ToLower(line), strings.ToLower(searchQuery)) {
				line = highlightMatch(line, searchQuery)

				// Add indicator for current match
				isCurrentMatch := false
				for j, matchIdx := range searchMatches {
					if matchIdx == i && j == currentMatch {
						isCurrentMatch = true
						break
					}
				}

				if isCurrentMatch {
					fmt.Printf("→ %s\r\n", line)
				} else {
					fmt.Printf("  %s\r\n", line)
				}
			} else {
				fmt.Printf("  %s\r\n", line)
			}
		}

		// Print status line
		if searchMode {
			fmt.Printf("\r/\033[36m%s\033[0m", searchQuery)
		} else if showHelp {
			fmt.Print("\r\033[33mCommands: q:quit, /:search, n:next, p:prev, h:help\033[0m")
		} else {
			matchInfo := ""
			if len(searchMatches) > 0 {
				matchInfo = fmt.Sprintf(", Match %d/%d", currentMatch+1, len(searchMatches))
			}
			fmt.Printf("\r-- Page %d/%d%s (h for help) --",
				currentPage+1, totalPages, matchInfo)
		}
	}

	// Function to perform search
	performSearch := func() {
		searchMatches = []int{}
		for i, item := range items {
			if strings.Contains(strings.ToLower(item), strings.ToLower(searchQuery)) {
				searchMatches = append(searchMatches, i)
			}
		}

		// Reset match index and navigate to first match if found
		currentMatch = -1
		if len(searchMatches) > 0 {
			currentMatch = 0
			matchPage := searchMatches[currentMatch] / pageSize
			currentPage = matchPage
		}
	}

	// Function to navigate to next match
	nextMatch := func() {
		if len(searchMatches) == 0 {
			return
		}

		currentMatch = (currentMatch + 1) % len(searchMatches)
		matchPage := searchMatches[currentMatch] / pageSize
		currentPage = matchPage
	}

	// Function to navigate to previous match
	prevMatch := func() {
		if len(searchMatches) == 0 {
			return
		}

		currentMatch = (currentMatch - 1 + len(searchMatches)) % len(searchMatches)
		matchPage := searchMatches[currentMatch] / pageSize
		currentPage = matchPage
	}

	// Initial render
	renderPage()

	for {
		// Read a single byte
		char, err := reader.ReadByte()
		if err != nil {
			return err
		}

		if searchMode {
			// Handle search mode input
			switch char {
			case 13, 10: // Enter key
				searchMode = false
				if searchQuery != "" {
					performSearch()
				}
			case 27: // Escape
				searchMode = false
				// Optionally clear search
				// searchQuery = ""
				// searchMatches = []int{}
			case 127, 8: // Backspace/Delete
				if len(searchQuery) > 0 {
					searchQuery = searchQuery[:len(searchQuery)-1]
				}
			default:
				// Add printable characters to query
				if char >= 32 && char <= 126 {
					searchQuery += string(char)
				}
			}
			renderPage()
			continue
		}

		// Not in search mode, handle navigation commands
		switch char {
		case 'q', 'Q':
			fmt.Print("\033[H\033[2J") // Clear screen before exiting
			return nil

		case '/':
			searchMode = true
			showHelp = false
			renderPage()

		case 'n', 'N':
			nextMatch()
			renderPage()

		case 'p', 'P':
			prevMatch()
			renderPage()

		case 'h', 'H':
			showHelp = !showHelp
			renderPage()

		case '\x1b': // Escape sequence
			// Read the rest of escape sequence immediately
			sequence := make([]byte, 2)
			reader.Read(sequence)
			if sequence[0] == '[' {
				switch sequence[1] {
				case 'A': // Up arrow
					if currentPage > 0 {
						currentPage--
						renderPage()
					}
				case 'B': // Down arrow
					if currentPage < totalPages-1 {
						currentPage++
						renderPage()
					}
				}
			}
		}
	}
}

// RunMore implements a more-like function that can be used from command line
// It supports reading from files or stdin if no file is provided or '<' is used
func RunMore(args []string) error {
	// Check if we're dealing with stdin input (no arguments or explicitly using '<')
	if len(args) == 0 || (len(args) > 1 && args[0] == "<") {
		// Reading from stdin
		fileIndex := 0
		if len(args) > 1 && args[0] == "<" {
			fileIndex = 1
		}

		// Check if there's a file specified after '<'
		if len(args) > fileIndex {
			// Open the specified file
			file, err := os.Open(args[fileIndex])
			if err != nil {
				return fmt.Errorf("failed to open file %s: %v", args[fileIndex], err)
			}
			defer file.Close()
			return moreFromReader(file)
		}

		// No file specified, read from stdin
		return moreFromReader(os.Stdin)
	}

	// Direct file argument(s) provided
	for _, filename := range args {
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", filename, err)
		}

		err = moreFromReader(file)
		file.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// moreFromReader reads content from a reader and displays it using More
func moreFromReader(reader io.Reader) error {
	// Read the file content into memory
	scanner := bufio.NewScanner(reader)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}

	// Use the More function to display the content
	return More(lines)
}

// highlightMatch highlights a search query in a text string
func highlightMatch(text, query string) string {
	if query == "" {
		return text
	}

	// Case-insensitive search
	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	var result strings.Builder
	lastIndex := 0

	for {
		index := strings.Index(lowerText[lastIndex:], lowerQuery)
		if index == -1 {
			break
		}

		// Adjust index to account for the slice
		index += lastIndex

		// Append text before the match
		result.WriteString(text[lastIndex:index])

		// Append the highlighted match
		result.WriteString("\033[1;33m") // Bold yellow
		result.WriteString(text[index : index+len(query)])
		result.WriteString("\033[0m") // Reset

		// Update lastIndex
		lastIndex = index + len(query)
	}

	// Append the remaining text
	result.WriteString(text[lastIndex:])

	return result.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
