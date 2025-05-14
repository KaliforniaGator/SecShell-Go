package history

import (
	"bufio"
	"fmt"
	"os"
	"secshell/colors"
	"secshell/core"
	"secshell/logging"
	"secshell/ui"
	"secshell/ui/gui"
	"strings"

	"golang.org/x/term"
)

// Define processCommand function
type processCommand func(input string)

// displayHistory shows the command history
func DisplayHistory(history []string) {
	gui.TitleBox("Command History")
	if len(history) > gui.GetTerminalHeight()-5 {
		var numberedHistory []string

		for i, cmd := range history {
			numberedHistory = append(numberedHistory, fmt.Sprintf("%d: %s", i+1, cmd))
		}
		core.More(numberedHistory)
	} else {
		for i, cmd := range history {
			fmt.Printf("%d: %s\n", i+1, cmd)
		}
	}
}

func SearchHistory(history []string, query string) {
	gui.TitleBox("History Search: " + query)
	found := false

	for i, cmd := range history {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(query)) {
			highlightedCmd := highlightText(cmd, query)
			fmt.Printf("%d: %s\n", i+1, highlightedCmd)
			found = true
		}
	}

	if !found {
		gui.AlertBox("No matching commands found.")
	}
}

func RunHistoryCommand(history []string, number int, processCommand processCommand) bool {
	if number <= 0 || number > len(history) {
		gui.ErrorBox(fmt.Sprintf("Invalid history number: %d", number))
		return false
	}

	cmd := history[number-1]
	gui.AlertBox(fmt.Sprintf("Running: %s", cmd))
	processCommand(cmd)
	return true
}

func InteractiveHistorySearch(history []string, processCommand processCommand) {

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		logging.LogError(err)
		gui.ErrorBox(fmt.Sprintf("Failed to set terminal to raw mode: %s", err))
		return
	}

	// Clear screen and enter alternate screen buffer
	fmt.Print("\033[H\033[2J\x1b[?1049h")

	// Create exit function to handle cleanup
	exitFunc := func(cmd string, runCommand bool) {
		fmt.Print("\033[H\033[2J") // Clear screen
		fmt.Print("\x1b[?1049l")   // Exit alternate screen buffer
		ui.ClearScreenAndBuffer()
		fmt.Print("\033[?25h") // Show cursor
		term.Restore(int(os.Stdin.Fd()), oldState)

		if runCommand && cmd != "" {
			gui.AlertBox("Running: " + cmd)
			processCommand(cmd)
		}
	}

	// Initialize variables
	query := ""
	selectedIndex := 0 // Index within the filteredHistory
	filteredHistory := []string{}
	currentPage := 0
	pageSize := 10 // Default, will be updated

	// Hide cursor while navigating
	fmt.Print("\033[?25l")
	// Defer for showing cursor is handled above
	// Helper function to refresh display
	refreshDisplay := func() {
		// Get terminal height
		_, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			termHeight = 24 // Default height if error
		}
		headerLines := 4 // Header, prompt, blank line, instructions
		statusLine := 1
		pageSize = termHeight - headerLines - statusLine
		if pageSize < 1 {
			pageSize = 1 // Ensure at least one item can be shown
		}

		// Filter history based on query
		filteredHistory = []string{}
		for _, cmd := range history {
			if query == "" || strings.Contains(strings.ToLower(cmd), strings.ToLower(query)) {
				filteredHistory = append(filteredHistory, cmd)
			}
		}

		// Calculate total pages
		totalPages := 1
		if len(filteredHistory) > 0 {
			totalPages = (len(filteredHistory) + pageSize - 1) / pageSize
		}
		if currentPage >= totalPages {
			currentPage = max(0, totalPages-1) // Adjust if current page becomes invalid
		}

		// Ensure selectedIndex is valid
		if selectedIndex < 0 {
			selectedIndex = 0
		}
		if selectedIndex >= len(filteredHistory) && len(filteredHistory) > 0 {
			selectedIndex = len(filteredHistory) - 1
		}

		var sb strings.Builder

		// Display header with drawbox
		sb.WriteString("\n\033[2K\r") // Equivalent of fmt.Print("\n"); ui.ClearLine();
		sb.WriteString(colors.BoldGreen + "┌─[Interactive History Search]" + colors.Reset + "\n")

		sb.WriteString("\033[2K\r") // Equivalent of ui.ClearLine();
		sb.WriteString(fmt.Sprintf(colors.BoldGreen+"└─"+colors.Reset+"$ %s", query))

		// Print instructions
		sb.WriteString("\n\033[2K\r") // Equivalent of fmt.Print("\n"); ui.ClearLine();
		sb.WriteString("Up/Down to select, Left/Right for pages, Enter to run, Esc to cancel\n")

		// Calculate display range for current page
		start := currentPage * pageSize
		end := min(start+pageSize, len(filteredHistory))

		// Display results with selection highlight
		for i := start; i < end; i++ {
			cmd := filteredHistory[i]
			sb.WriteString("\033[2K\r") // Equivalent of ui.ClearLine();
			if i == selectedIndex {
				sb.WriteString(fmt.Sprintf("%s→ %d: %s%s\n", colors.BoldGreen, i+1, highlightText(cmd, query), colors.Reset))
			} else {
				sb.WriteString(fmt.Sprintf("  %d: %s\n", i+1, cmd))
			}
		}

		// Fill remaining lines on the page if necessary
		for i := end - start; i < pageSize; i++ {
			sb.WriteString("\033[2K\r") // Equivalent of ui.ClearLine();
			sb.WriteString("\n")        // Was fmt.Print("\r\n")
		}

		// Print status line
		sb.WriteString("\033[2K\r") // Equivalent of ui.ClearLine();
		if len(filteredHistory) == 0 {
			sb.WriteString("  No matching commands found.")
		} else {
			sb.WriteString(fmt.Sprintf("-- Page %d/%d (%d results) --", currentPage+1, totalPages, len(filteredHistory)))
		}

		// Actual printing:
		fmt.Print("\033[H\033[2J") // Clear screen and move cursor home (as before)
		fmt.Print(sb.String())     // Print the composed buffer
	}

	// Initial display
	refreshDisplay()

	// Input loop

	buf := make([]byte, 3)

	for {

		n, err := os.Stdin.Read(buf)

		if err != nil {

			logging.LogError(err)

			gui.ErrorBox(fmt.Sprintf("Failed to read input: %s", err))

			return // Defer will handle cleanup

		}

		if n == 1 {

			switch buf[0] {

			case 27: // ESC
				exitFunc("", false)
				return

			case 13: // Enter
				if len(filteredHistory) > 0 && selectedIndex >= 0 && selectedIndex < len(filteredHistory) {
					selectedCmd := filteredHistory[selectedIndex]
					exitFunc(selectedCmd, true)
					return
				}

			case 127, 8: // Backspace/Delete

				if len(query) > 0 {

					query = query[:len(query)-1]

					selectedIndex = 0 // Reset selection

					currentPage = 0 // Reset page

					refreshDisplay()

				}

			default:

				// Add printable characters to query

				if buf[0] >= 32 && buf[0] <= 126 {

					query += string(buf[0])

					selectedIndex = 0 // Reset selection

					currentPage = 0 // Reset page

					refreshDisplay()

				}

			}

		} else if n == 3 && buf[0] == 27 && buf[1] == 91 {

			// Handle arrow keys

			if len(filteredHistory) > 0 { // Only navigate if there are results
				pageStartIndex := currentPage * pageSize
				// Ensure pageEndIndex is correctly calculated, even for partially filled last page
				var pageEndIndex int
				if len(filteredHistory) == 0 { // Should be caught by outer if, but defensive
					pageEndIndex = -1 // No items
				} else {
					pageEndIndex = min((currentPage+1)*pageSize, len(filteredHistory)) - 1
				}

				switch buf[2] {

				case 65: // Up arrow
					if selectedIndex > pageStartIndex {
						selectedIndex--
						refreshDisplay()
					}

				case 66: // Down arrow
					if selectedIndex < pageEndIndex {
						selectedIndex++
						refreshDisplay()
					}

				case 68: // Left arrow (D)
					currentTotalPages := 1
					if len(filteredHistory) > 0 {
						currentTotalPages = (len(filteredHistory) + pageSize - 1) / pageSize
					}

					if currentTotalPages > 1 { // Only page if there's more than one page
						currentPage--
						if currentPage < 0 {
							currentPage = currentTotalPages - 1
						}
						selectedIndex = currentPage * pageSize
						// Ensure selectedIndex is not out of bounds if the new page is the last and not full
						if selectedIndex >= len(filteredHistory) {
							selectedIndex = len(filteredHistory) - 1
						}
						refreshDisplay()
					}

				case 67: // Right arrow (C)
					currentTotalPages := 1
					if len(filteredHistory) > 0 {
						currentTotalPages = (len(filteredHistory) + pageSize - 1) / pageSize
					}

					if currentTotalPages > 1 { // Only page if there's more than one page
						currentPage++
						if currentPage >= currentTotalPages {
							currentPage = 0
						}
						selectedIndex = currentPage * pageSize
						// Ensure selectedIndex is not out of bounds if the new page is the last and not full
						// This typically shouldn't happen if totalPages and currentPage are correct,
						// but good for safety.
						if selectedIndex >= len(filteredHistory) {
							selectedIndex = len(filteredHistory) - 1
						}
						refreshDisplay()
					}
				}
			}
		}
	}
}

func GetHistoryFromFile(filePath string) []string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var history []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		history = append(history, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return history
}

func highlightText(text, query string) string {
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
		result.WriteString(colors.BoldYellow)
		result.WriteString(text[index : index+len(query)])
		result.WriteString(colors.Reset)

		// Update lastIndex
		lastIndex = index + len(query)
	}

	// Append the remaining text
	result.WriteString(text[lastIndex:])

	return result.String()
}
