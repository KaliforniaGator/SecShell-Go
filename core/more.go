package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"secshell/terminal"
	"strings"

	"golang.org/x/term"
)

// MorePager represents a terminal-based pager for displaying content
type MorePager struct {
	items         []string      // Content to display
	reader        *bufio.Reader // Reader for input
	pageSize      int           // Number of lines per page
	currentPage   int           // Current page index (0-based)
	totalPages    int           // Total number of pages
	searchMode    bool          // Whether search mode is active
	searchQuery   string        // Current search query
	searchMatches []int         // Indices of items matching search query
	currentMatch  int           // Current match index (-1 if none)
	showHelp      bool          // Whether to show help
	termWidth     int           // Terminal width
	termHeight    int           // Terminal height
	wrapText      bool          // Whether to wrap text instead of truncating
}

// NewMorePager creates and initializes a new MorePager
func NewMorePager(items []string) (*MorePager, error) {
	// Enter raw mode using the terminal package
	if err := terminal.EnterRawMode(); err != nil {
		return nil, err
	}

	// Get terminal size
	width, height, err := terminal.GetTerminalSize()
	if err != nil {
		terminal.ExitRawMode()
		return nil, err
	}

	pageSize := height - 2 // One line for status, one for search/help

	return &MorePager{
		items:         items,
		reader:        bufio.NewReader(os.Stdin),
		pageSize:      pageSize,
		currentPage:   0,
		totalPages:    (len(items) + pageSize - 1) / pageSize,
		searchMode:    false,
		searchQuery:   "",
		searchMatches: []int{},
		currentMatch:  -1,
		showHelp:      false,
		termWidth:     width,
		termHeight:    height,
		wrapText:      true, // Enable text wrapping by default
	}, nil
}

// More displays a scrollable list of items in the terminal, paginated by screen height
func More(items []string) error {
	if len(items) == 0 {
		return nil
	}

	pager, err := NewMorePager(items)
	if err != nil {
		return err
	}

	// Enter alternate screen buffer and hide cursor using the terminal package
	terminal.EnterAlternateScreen()
	fmt.Print("\x1b[?25l") // Hide cursor

	// Clear the alternate screen to start fresh
	fmt.Print("\x1b[2J\x1b[H")

	// Ensure terminal state and alternate buffer are restored on exit
	defer func() {
		// Clear the entire screen before exiting to ensure no content remains
		fmt.Print("\x1b[2J\x1b[H")

		// Show cursor and exit alternate screen buffer
		fmt.Print("\x1b[?25h")
		terminal.ExitAlternateScreen()

		// Restore terminal state
		terminal.ExitRawMode()

		// Clear screen and buffer again after returning to main screen
		fmt.Print("\x1b[2J\x1b[H\x1b[3J")

		// Ensure all output is flushed
		os.Stdout.Sync()
	}()

	// Start the pager
	return pager.Run()
}

// Run starts the pager interface
func (m *MorePager) Run() error {
	// Initial render with full clear
	m.fullClearScreen()
	m.renderPage()

	for {
		// Read a single byte
		char, err := m.reader.ReadByte()
		if err != nil {
			return err // Defer will handle cleanup
		}

		if m.searchMode {
			m.handleSearchInput(char)
		} else {
			// Handle exit condition
			if m.handleNavigationInput(char) {
				// Clear screen before exiting to remove any scrollable content
				m.fullClearScreen()
				return nil
			}
		}
	}
}

// fullClearScreen completely clears the screen and resets cursor position
func (m *MorePager) fullClearScreen() {
	// Clear entire screen
	fmt.Print("\x1b[2J")
	// Move cursor to home position
	fmt.Print("\x1b[H")
}

// handleSearchInput processes input while in search mode
func (m *MorePager) handleSearchInput(char byte) {
	switch char {
	case 13, 10: // Enter key
		m.searchMode = false
		if m.searchQuery != "" {
			m.performSearch()
		}
	case 27: // Escape
		m.searchMode = false
	case 127, 8: // Backspace/Delete
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			// Disable search mode if query becomes empty
			if m.searchQuery == "" {
				m.searchMode = false
				m.searchMatches = []int{}
				m.currentMatch = -1
			}
		}
	default:
		// Add printable characters to query
		if char >= 32 && char <= 126 {
			m.searchQuery += string(char)
		}
	}
	m.fullClearScreen()
	m.renderPage()
}

// handleNavigationInput processes input while in navigation mode
// Returns true if the pager should exit
func (m *MorePager) handleNavigationInput(char byte) bool {
	prevPage := m.currentPage
	pageChanged := false

	switch char {
	case 'q', 'Q':
		return true

	case '/':
		m.searchMode = true
		m.showHelp = false

	case 'c', 'C': // Clear search
		m.searchQuery = ""
		m.searchMode = false
		m.searchMatches = []int{}
		m.currentMatch = -1

	case 'h', 'H':
		m.showHelp = !m.showHelp

	case 'w', 'W': // Toggle text wrapping
		m.wrapText = !m.wrapText

	case '\x1b': // Escape sequence
		// Read the rest of escape sequence immediately
		sequence := make([]byte, 2)
		_, err := m.reader.Read(sequence)
		if err != nil {
			// Handle error, perhaps log it or return true to exit
			return true
		}

		if sequence[0] == '[' {
			switch sequence[1] {
			case 'A': // Up arrow - navigate to previous match
				if len(m.searchMatches) > 0 {
					m.navigateToPreviousMatch()
					if prevPage != m.currentPage {
						pageChanged = true
					}
				}
				// No page scroll if not in search
			case 'B': // Down arrow - navigate to next match
				if len(m.searchMatches) > 0 {
					m.navigateToNextMatch()
					if prevPage != m.currentPage {
						pageChanged = true
					}
				}
				// No page scroll if not in search
			case 'D': // Left arrow - previous page
				if m.currentPage > 0 {
					m.currentPage--
					pageChanged = true
				}
			case 'C': // Right arrow - next page
				if m.currentPage < m.totalPages-1 {
					m.currentPage++
					pageChanged = true
				}
			}
		}
	}

	// If page changed, do a full clear to prevent ghosting
	if pageChanged {
		m.fullClearScreen()
	}

	m.renderPage()
	return false
}

// navigateToNextMatch moves to the next search match
func (m *MorePager) navigateToNextMatch() {
	if len(m.searchMatches) > 0 {
		m.currentMatch = (m.currentMatch + 1) % len(m.searchMatches)
		matchPage := m.searchMatches[m.currentMatch] / m.pageSize
		m.currentPage = matchPage
	}
}

// navigateToPreviousMatch moves to the previous search match
func (m *MorePager) navigateToPreviousMatch() {
	if len(m.searchMatches) > 0 {
		m.currentMatch--
		if m.currentMatch < 0 {
			m.currentMatch = len(m.searchMatches) - 1
		}
		matchPage := m.searchMatches[m.currentMatch] / m.pageSize
		m.currentPage = matchPage
	}
}

// performSearch finds all items matching the current search query
func (m *MorePager) performSearch() {
	m.searchMatches = []int{}
	for i, item := range m.items {
		if strings.Contains(strings.ToLower(item), strings.ToLower(m.searchQuery)) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}

	// Reset match index and navigate to first match if found
	m.currentMatch = -1
	if len(m.searchMatches) > 0 {
		m.currentMatch = 0
		matchPage := m.searchMatches[m.currentMatch] / m.pageSize
		m.currentPage = matchPage
	}
}

// renderPage displays the current page and status information
func (m *MorePager) renderPage() {
	// Move cursor to top-left corner
	fmt.Print("\x1b[H")

	// Clear entire screen to prevent ghosting
	fmt.Print("\x1b[2J")

	// Move cursor back to top-left after clear
	fmt.Print("\x1b[H")

	// Print current page
	m.renderContent()

	// Print status line at the correct position (bottom line)
	fmt.Printf("\x1b[%d;1H", m.termHeight)
	m.renderStatusLine()

	// Flush stdout to ensure all content is displayed immediately
	os.Stdout.Sync()
}

// renderContent displays the current page content
func (m *MorePager) renderContent() {
	start := m.currentPage * m.pageSize
	end := min(start+m.pageSize, len(m.items))

	// Keep track of lines displayed
	displayedLines := 0
	maxDisplayLines := m.pageSize

	for i := start; i < end && displayedLines < maxDisplayLines; i++ {
		line := m.items[i]

		// Highlight search matches
		if m.searchQuery != "" && strings.Contains(strings.ToLower(line), strings.ToLower(m.searchQuery)) {
			line = highlightMatch(line, m.searchQuery)
		}

		// Determine prefix based on whether this is the current match
		prefix := "  "
		if m.currentMatch >= 0 && m.searchMatches[m.currentMatch] == i {
			prefix = "→ \033[1;32m" // Arrow and green text
			line = line + "\033[0m" // Reset at the end
		}

		// Available width for text (considering prefix)
		availWidth := m.termWidth - len(prefix) + 5 // +5 accounts for ANSI color sequences

		// Handle line wrapping or truncation
		if m.wrapText {
			wrappedLines := m.wrapLine(line, availWidth)

			// Print first line with prefix
			fmt.Printf("%s%s\r\n", prefix, wrappedLines[0])
			displayedLines++

			// Print continuation lines if any and if we have space
			for j := 1; j < len(wrappedLines) && displayedLines < maxDisplayLines; j++ {
				fmt.Printf("  %s\r\n", wrappedLines[j])
				displayedLines++
			}
		} else {
			// Truncate if not wrapping
			if len(line) > availWidth {
				line = line[:availWidth-3] + "..."
			}
			fmt.Printf("%s%s\r\n", prefix, line)
			displayedLines++
		}
	}

	// Fill remaining lines with blank space to clear old content
	for i := displayedLines; i < m.pageSize; i++ {
		fmt.Print("\r\n")
	}
}

// wrapLine wraps a line of text to fit within the specified width
// Returns an array of wrapped lines
func (m *MorePager) wrapLine(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	var currentLine strings.Builder
	currentLineLen := 0
	words := strings.Fields(text)

	// Handle case with no spaces for wrapping
	if len(words) == 0 {
		// Split by character for very long words with no spaces
		var chunks []string
		for i := 0; i < len(text); i += width {
			end := i + width
			if end > len(text) {
				end = len(text)
			}
			chunks = append(chunks, text[i:end])
		}
		return chunks
	}

	for _, word := range words {
		// Check if adding this word would exceed the width
		if currentLineLen+len(word)+1 > width && currentLineLen > 0 {
			// Line would be too long, start a new one
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLineLen = 0
		}

		// Special handling for very long words
		if len(word) > width {
			// If we have content in the current line, add it first
			if currentLineLen > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLineLen = 0
			}

			// Break the long word into chunks
			for i := 0; i < len(word); i += width {
				end := i + width
				if end > len(word) {
					end = len(word)
				}

				if i == 0 && currentLineLen > 0 {
					// First chunk goes on the current line if there's content
					if currentLineLen > 0 {
						currentLine.WriteString(" ")
					}
					currentLine.WriteString(word[i:end])
					currentLineLen += 1 + (end - i)
				} else {
					// Other chunks go on new lines
					lines = append(lines, word[i:end])
				}
			}
			continue
		}

		// Add space before word if not at start of line
		if currentLineLen > 0 {
			currentLine.WriteString(" ")
			currentLineLen++
		}

		currentLine.WriteString(word)
		currentLineLen += len(word)
	}

	// Don't forget to add the last line if it has content
	if currentLineLen > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// renderStatusLine displays the status line at the bottom of the screen
func (m *MorePager) renderStatusLine() {
	// Clear the status line first
	fmt.Print("\x1b[K")

	if m.searchMode {
		fmt.Printf("\r/\033[36m%s\033[0m", m.searchQuery)
	} else if m.showHelp {
		fmt.Print("\r\033[33mCommands: q:quit, /:search, c:clear, ↑↓:matches, ←→:pages, w:wrap, h:help\033[0m")
	} else {
		matchInfo := ""
		if len(m.searchMatches) > 0 {
			matchInfo = fmt.Sprintf(", Match %d/%d", m.currentMatch+1, len(m.searchMatches))
		}
		wrapStatus := ""
		if m.wrapText {
			wrapStatus = ", Wrap:on"
		} else {
			wrapStatus = ", Wrap:off"
		}
		fmt.Printf("\r-- Page %d/%d%s%s (h for help) --",
			m.currentPage+1, m.totalPages, matchInfo, wrapStatus)
	}
}

// HandleTermResize handles terminal resize events by recalculating page size and redrawing
func (m *MorePager) HandleTermResize() error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	m.termWidth = width
	m.termHeight = height
	m.pageSize = height - 2
	m.totalPages = (len(m.items) + m.pageSize - 1) / m.pageSize

	// Ensure current page is still valid
	if m.currentPage >= m.totalPages {
		m.currentPage = m.totalPages - 1
	}

	// Full clear on resize to prevent ghosting
	m.fullClearScreen()

	// Redraw page with new dimensions
	m.renderPage()
	return nil
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
