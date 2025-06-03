package terminal

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

var (
	// originalTermState stores the terminal state before entering raw mode
	originalTermState *term.State
	// mutex to protect concurrent access to terminal state
	termMutex sync.Mutex
	// track if we're currently in alternate screen mode
	inAlternateScreen bool
	// track if we're currently in raw mode
	inRawMode bool
)

// EnterAlternateScreen enters the alternate screen buffer
// This uses the modern escape sequence \033[?1049h
func EnterAlternateScreen() {
	termMutex.Lock()
	defer termMutex.Unlock()

	if !inAlternateScreen {
		fmt.Print("\033[?1049h") // Enter alternate screen
		inAlternateScreen = true
	}
}

// ExitAlternateScreen exits the alternate screen buffer and returns to the normal screen
// This uses the modern escape sequence \033[?1049l
func ExitAlternateScreen() {
	termMutex.Lock()
	defer termMutex.Unlock()

	if inAlternateScreen {
		fmt.Print("\033[?1049l") // Exit alternate screen
		inAlternateScreen = false
	}
}

// EnterRawMode puts the terminal into raw mode
// Returns an error if it fails
func EnterRawMode() error {
	termMutex.Lock()
	defer termMutex.Unlock()

	if !inRawMode {
		fd := int(os.Stdin.Fd())
		state, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("failed to enter raw mode: %w", err)
		}
		originalTermState = state
		inRawMode = true
	}
	return nil
}

// ExitRawMode restores the terminal to its original state
func ExitRawMode() {
	termMutex.Lock()
	defer termMutex.Unlock()

	if inRawMode && originalTermState != nil {
		term.Restore(int(os.Stdin.Fd()), originalTermState)
		inRawMode = false
	}
}

// EnterInteractiveMode enters both raw mode and alternate screen
// Returns an error if entering raw mode fails
func EnterInteractiveMode() error {
	// Enter alternate screen first
	EnterAlternateScreen()

	// Then enter raw mode
	if err := EnterRawMode(); err != nil {
		// If raw mode fails, exit alternate screen to avoid leaving terminal in a bad state
		ExitAlternateScreen()
		return err
	}

	return nil
}

// ExitInteractiveMode exits both raw mode and alternate screen
func ExitInteractiveMode() {
	// Exit raw mode first
	ExitRawMode()

	// Then exit alternate screen
	ExitAlternateScreen()
}

// GetTerminalSize returns the current terminal dimensions (width, height)
func GetTerminalSize() (width, height int, err error) {
	return term.GetSize(int(os.Stdin.Fd()))
}

// IsTerminal checks if the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// WithInteractiveMode runs the provided function with raw mode and alternate screen enabled
// Ensures proper cleanup even if the function panics
func WithInteractiveMode(fn func() error) error {
	if err := EnterInteractiveMode(); err != nil {
		return fmt.Errorf("failed to enter interactive mode: %w", err)
	}

	defer ExitInteractiveMode()

	return fn()
}
