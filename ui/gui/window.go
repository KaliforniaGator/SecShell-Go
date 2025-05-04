package gui

import (
	"bufio" // Keep for potential future use, but not for raw input loop
	"fmt"
	"os"
	"secshell/colors"
	"strings"
	"time" // Added for potential brief pauses if needed

	"golang.org/x/term" // Import the term package
)

// UIElement represents any element that can be rendered within a window.
type UIElement interface {
	Render(buffer *strings.Builder, x, y int, width int) // Renders the element onto a buffer at given coords
	// Add methods for interaction later if needed (e.g., HandleInput)
}

// --- Basic UI Elements ---

// Label represents a simple text element.
type Label struct {
	Text  string
	Color string
	X, Y  int // Position relative to window content area
}

func NewLabel(text string, x, y int, color string) *Label {
	return &Label{Text: text, X: x, Y: y, Color: color}
}

func (l *Label) Render(buffer *strings.Builder, winX, winY int, _ int) {
	// Calculate absolute position
	absX := winX + l.X
	absY := winY + l.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))
	buffer.WriteString(l.Color)
	buffer.WriteString(l.Text)
	buffer.WriteString(colors.Reset) // Reset color after element
}

// Button represents a clickable button element.
type Button struct {
	Text        string
	Color       string
	ActiveColor string // Color when selected/active
	X, Y        int    // Position relative to window content area
	Width       int
	Action      func() bool // Function to call when activated. Returns true to stop interaction loop.
	IsActive    bool        // State for rendering
}

func NewButton(text string, x, y, width int, color, activeColor string, action func() bool) *Button {
	return &Button{
		Text:        text,
		X:           x,
		Y:           y,
		Width:       width,
		Color:       color,
		ActiveColor: activeColor,
		Action:      action,
		IsActive:    false,
	}
}

func (b *Button) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + b.X
	absY := winY + b.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	renderColor := b.Color
	if b.IsActive {
		renderColor = b.ActiveColor
		buffer.WriteString(ReverseVideo()) // Indicate active state
	}
	buffer.WriteString(renderColor)

	// Basic button rendering (text centered within width)
	padding := (b.Width - len(b.Text)) / 2
	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", b.Width-len(b.Text)-padding)
	buffer.WriteString(fmt.Sprintf("[%s%s%s]", leftPad, b.Text, rightPad))

	buffer.WriteString(colors.Reset) // Reset color and video attributes
}

// --- Window Structure ---

// Window represents a bordered area on the screen containing UI elements.
type Window struct {
	Title             string
	Icon              string
	X, Y              int // Top-left corner position
	Width, Height     int
	BoxStyle          string
	TitleColor        string
	BorderColor       string
	BgColor           string // Background color for the content area
	ContentColor      string // Default text color for content area (can be overridden by elements)
	Elements          []UIElement
	buffer            strings.Builder // Internal buffer for drawing commands
	focusableElements []UIElement     // Slice to hold focusable elements (like buttons)
	focusedIndex      int             // Index of the currently focused element in focusableElements
}

// NewWindow creates a new Window instance.
func NewWindow(icon, title string, x, y, width, height int, boxStyle, titleColor, borderColor, bgColor, contentColor string) *Window {
	if _, exists := BoxTypes[boxStyle]; !exists {
		boxStyle = "single" // Default style
	}
	return &Window{
		Icon:              icon,
		Title:             title,
		X:                 x,
		Y:                 y,
		Width:             width,
		Height:            height,
		BoxStyle:          boxStyle,
		TitleColor:        titleColor,
		BorderColor:       borderColor,
		BgColor:           bgColor,
		ContentColor:      contentColor,
		Elements:          make([]UIElement, 0),
		focusableElements: make([]UIElement, 0), // Initialize focusable elements slice
		focusedIndex:      -1,                   // No element focused initially
	}
}

// AddElement adds a UIElement to the window.
func (w *Window) AddElement(element UIElement) {
	w.Elements = append(w.Elements, element)

	// Check if the element is focusable (currently, only Buttons)
	if btn, ok := element.(*Button); ok {
		w.focusableElements = append(w.focusableElements, btn)
		// If this is the first focusable element, focus it
		if w.focusedIndex == -1 {
			w.focusedIndex = 0
			btn.IsActive = true // Activate the first button
		}
	}
}

// Render draws the window and its elements to the terminal.
func (w *Window) Render() {
	w.buffer.Reset() // Clear previous rendering commands

	box := BoxTypes[w.BoxStyle]
	fullTitle := w.Icon + " " + w.Title

	// --- Draw Border and Background ---
	w.buffer.WriteString(w.BorderColor)
	w.buffer.WriteString(w.BgColor) // Set background for the whole area initially

	// Top border with Title
	contentWidth := w.Width // Available space between corners
	titleLen := len(fullTitle)
	leftPadding := 0
	rightPadding := 0

	if contentWidth < 0 {
		contentWidth = 0 // Avoid negative width
	}

	if titleLen > contentWidth {
		// Title is too long, truncate it with ellipsis if possible
		if contentWidth > 3 {
			fullTitle = fullTitle[:contentWidth-3] + "..."
		} else {
			fullTitle = fullTitle[:contentWidth] // Truncate without ellipsis if space is tiny
		}
		leftPadding = 0
		rightPadding = 0
	} else {
		// Title fits, calculate padding
		totalPadding := contentWidth - titleLen
		leftPadding = totalPadding / 2
		rightPadding = totalPadding - leftPadding // Ensures correct total padding for odd/even
	}

	w.buffer.WriteString(MoveCursorCmd(w.Y, w.X))
	w.buffer.WriteString(box.TopLeft)
	w.buffer.WriteString(strings.Repeat(box.Horizontal, leftPadding))
	w.buffer.WriteString(w.TitleColor)  // Title color might differ from border
	w.buffer.WriteString(fullTitle)     // Print potentially truncated title
	w.buffer.WriteString(w.BorderColor) // Back to border color
	w.buffer.WriteString(strings.Repeat(box.Horizontal, rightPadding))
	w.buffer.WriteString(box.TopRight)

	// Middle rows (Vertical borders and background fill)
	contentBg := w.BgColor + strings.Repeat(" ", w.Width-2) // Precompute background fill string
	for i := 1; i < w.Height-1; i++ {
		w.buffer.WriteString(MoveCursorCmd(w.Y+i, w.X))
		w.buffer.WriteString(box.Vertical)
		w.buffer.WriteString(contentBg)                           // Fill background
		w.buffer.WriteString(MoveCursorCmd(w.Y+i, w.X+w.Width-1)) // Move explicitly to end
		w.buffer.WriteString(box.Vertical)
	}

	// Bottom border
	w.buffer.WriteString(MoveCursorCmd(w.Y+w.Height-1, w.X))
	w.buffer.WriteString(box.BottomLeft)
	w.buffer.WriteString(strings.Repeat(box.Horizontal, w.Width-2))
	w.buffer.WriteString(box.BottomRight)

	// --- Render Elements ---
	// Elements are rendered relative to the top-left corner of the *content area*
	contentX := w.X + 1
	contentY := w.Y + 1
	contentWidth = w.Width - 2
	// Set default content color before rendering elements
	w.buffer.WriteString(w.ContentColor)
	for _, element := range w.Elements {
		// Pass the window's buffer, content area origin, and content width
		element.Render(&w.buffer, contentX, contentY, contentWidth)
	}

	// Reset colors at the end and print the buffer
	w.buffer.WriteString(colors.Reset)
	fmt.Print(w.buffer.String())
}

// setFocus updates the IsActive state of focusable elements.
func (w *Window) setFocus(newIndex int) {
	if len(w.focusableElements) == 0 {
		w.focusedIndex = -1
		return
	}

	// Deactivate the previously focused element (if any)
	if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
		if btn, ok := w.focusableElements[w.focusedIndex].(*Button); ok {
			btn.IsActive = false
		}
	}

	// Validate and set the new index
	if newIndex < 0 {
		w.focusedIndex = len(w.focusableElements) - 1 // Wrap around to the end
	} else if newIndex >= len(w.focusableElements) {
		w.focusedIndex = 0 // Wrap around to the start
	} else {
		w.focusedIndex = newIndex
	}

	// Activate the newly focused element
	if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
		if btn, ok := w.focusableElements[w.focusedIndex].(*Button); ok {
			btn.IsActive = true
		}
	}
}

func ClearLine() {
	// Clear the entire current line and return carriage
	fmt.Print("\033[2K\r")

}

// WindowActions handles user interaction within the window using raw terminal input.
func (w *Window) WindowActions() {
	// Get the file descriptor for stdin
	fd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		fmt.Println("Error: Standard input is not a terminal.")
		// Fallback to the previous simulated input? Or just exit?
		// For now, just print error and return.
		// A simple fallback:
		fmt.Println("Press Enter to continue...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		return
	}

	// Get the initial state of the terminal
	oldState, err := term.GetState(fd)
	if err != nil {
		fmt.Printf("Error getting terminal state: %v\n", err)
		return
	}
	// Ensure terminal state is restored on exit
	defer term.Restore(fd, oldState)
	// Ensure cursor is shown on exit
	defer fmt.Print(ShowCursor())

	// Put the terminal into raw mode
	_, err = term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("Error setting terminal to raw mode: %v\n", err)
		return
	}

	// Hide cursor during interaction
	fmt.Print(HideCursor())

	// Initial render
	w.Render()

	// Buffer for reading input bytes
	inputBuf := make([]byte, 3) // Read up to 3 bytes for escape sequences

	for {
		// Read input from the raw terminal
		n, err := os.Stdin.Read(inputBuf)
		if err != nil {
			// Handle read errors (e.g., if stdin is closed)
			break // Exit loop on read error
		}

		if n == 0 {
			continue // No input read, continue loop
		}

		key := inputBuf[:n]
		shouldQuit := false
		needsRender := false

		// --- Key Handling ---
		if n == 1 {
			switch key[0] {
			case '\t': // Tab key
				if len(w.focusableElements) > 0 {
					w.setFocus(w.focusedIndex + 1)
					needsRender = true
				}
			case '\r': // Enter key (Carriage Return in raw mode)
				if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
					if btn, ok := w.focusableElements[w.focusedIndex].(*Button); ok {
						if btn.Action != nil {
							if btn.Action() { // Execute action, check quit signal
								shouldQuit = true
							} else {
								// Action might have changed UI state
								needsRender = true
							}
						}
					}
				}
			case 'q', 'Q': // Quit key
				shouldQuit = true
			case 3: // Ctrl+C
				shouldQuit = true
			}
		} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // Check for escape sequences
			switch key[2] {
			// Potentially add arrow keys later if needed
			// case 'A': // Up Arrow
			// case 'B': // Down Arrow
			case 'Z': // Shift+Tab (Common sequence, might vary)
				if len(w.focusableElements) > 0 {
					w.setFocus(w.focusedIndex - 1)
					needsRender = true
				}
			}
		}
		// Add more key handling as needed (e.g., arrow keys)

		// --- Loop Control and Rendering ---
		if shouldQuit {
			break // Exit the interaction loop
		}

		if needsRender {
			w.Render() // Re-render the window state
		}
	}

	// Cleanup is handled by defers (Restore terminal state, Show cursor)
	// Clear the screen after finishing interaction
	fmt.Print(ClearScreenAndBuffer())
}

// TestWindowApp demonstrates creating and rendering a sample window.
func TestWindowApp() {
	// Clear screen before drawing
	fmt.Print(ClearScreenAndBuffer())
	// Hide cursor is now handled within WindowActions start

	// Get terminal dimensions
	termWidth := GetTerminalWidth()
	termHeight := GetTerminalHeight()

	// Define window dimensions and position (centered)
	winWidth := termWidth / 2
	if winWidth < 40 {
		winWidth = 40 // Minimum width
	}
	winHeight := termHeight / 2
	if winHeight < 12 { // Increased min height for more elements
		winHeight = 12
	}
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	// Create the window
	testWin := NewWindow("🚀", "Raw Input Test", winX, winY, winWidth, winHeight,
		"double", colors.BoldCyan, colors.BoldYellow, colors.BgBlack, colors.White)

	// Add elements
	infoLabel := NewLabel("Tab/Shift+Tab: Cycle | Enter: Activate | q/Ctrl+C: Quit", 1, 1, colors.Green)
	testWin.AddElement(infoLabel)

	detailLabel := NewLabel(fmt.Sprintf("Size: %dx%d Pos: (%d,%d)", winWidth, winHeight, winX, winY), 1, 3, colors.Gray)
	testWin.AddElement(detailLabel)

	// --- Buttons ---
	buttonWidth := 12
	buttonSpacing := 2 // Vertical space between buttons
	contentWidth := winWidth - 2
	buttonX := (contentWidth - buttonWidth) / 2 // Center buttons horizontally

	// Button 1: Placeholder Action
	actionButtonY := winHeight - 6 // Position near bottom
	actionButton := NewButton("Action 1", buttonX, actionButtonY, buttonWidth, colors.BoldBlue, colors.BgBlue+colors.BoldWhite, func() bool {
		// Modify UI element state directly instead of printing
		if lbl, ok := testWin.Elements[0].(*Label); ok { // Assuming infoLabel is the first element
			lbl.Text = "Action 1 Executed! (Tab/Shift+Tab, Enter, q/Ctrl+C)"
			lbl.Color = colors.BoldPurple
		}
		// Return false: Don't quit the app on this action
		return false
	})
	testWin.AddElement(actionButton)

	// Button 2: Quit Button
	quitButtonY := actionButtonY + buttonSpacing
	quitButton := NewButton("Quit App", buttonX, quitButtonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		// Modify UI element state directly if needed (e.g., show a quitting message)
		if lbl, ok := testWin.Elements[0].(*Label); ok {
			lbl.Text = "Quitting..."
			lbl.Color = colors.BoldRed
		}
		// Action returns true to signal quitting the interaction loop
		// A small delay can make the "Quitting..." message visible briefly
		testWin.Render() // Render the "Quitting..." message
		time.Sleep(300 * time.Millisecond)
		return true
	})
	testWin.AddElement(quitButton)

	// --- Start Interaction ---
	// WindowActions now handles raw input, rendering loop, and cleanup.
	testWin.WindowActions()

	// Code here runs after WindowActions loop finishes
	// Cursor is shown and terminal restored by defer in WindowActions
	// Screen is cleared by WindowActions before returning
	fmt.Println("Application finished.") // This will print after screen clear
}
