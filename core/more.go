package core

import (
	"bufio"
	"fmt"
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
	pageSize := rows - 1

	currentPage := 0
	totalPages := (len(items) + pageSize - 1) / pageSize
	reader := bufio.NewReader(os.Stdin)

	for {
		// Clear screen and move cursor to top
		ui.ClearScreenAndBuffer()
		fmt.Print("\033[H\033[2J")

		// Print current page
		start := currentPage * pageSize
		end := min(start+pageSize, len(items))
		for i := start; i < end; i++ {
			fmt.Printf("%s\r\n", items[i])
		}

		// Print status line
		fmt.Printf("\r-- Page %d/%d (UP/DOWN arrows to navigate, q to quit) --",
			currentPage+1, totalPages)

		// Read a single byte
		char, err := reader.ReadByte()
		if err != nil {
			return err
		}

		if char == '\x1b' {
			// Read the rest of escape sequence immediately
			sequence := make([]byte, 2)
			reader.Read(sequence)
			if sequence[0] == '[' {
				switch sequence[1] {
				case 'A': // Up arrow
					if currentPage > 0 {
						currentPage--
					}
				case 'B': // Down arrow
					if currentPage < totalPages-1 {
						currentPage++
					}
				}
			}
		} else if char == 'q' {
			fmt.Print("\033[H\033[2J") // Clear screen before exiting
			return nil
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
