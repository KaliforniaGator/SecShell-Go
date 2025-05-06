package gui

import (
	"bufio" // Keep for potential future use, but not for raw input loop
	"fmt"
	"os"
	"secshell/colors"
	"strings"

	// Added for potential brief pauses if needed
	"golang.org/x/term" // Import the term package
)

// UIElement represents any element that can be rendered within a window.
type UIElement interface {
	Render(buffer *strings.Builder, x, y int, width int) // Renders the element onto a buffer at given coords
	// Add methods for interaction later if needed (e.g., HandleInput)
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

	isFocusable := false
	var focusableElement UIElement = nil // Element to potentially add to focus list

	switch v := element.(type) {
	case *Button:
		isFocusable = true
		focusableElement = v
	case *TextBox:
		isFocusable = true
		focusableElement = v
		v.IsActive = false // Explicitly set inactive
	case *CheckBox:
		isFocusable = true
		focusableElement = v
		v.IsActive = false // Explicitly set inactive
	case *RadioButton:
		isFocusable = true
		focusableElement = v
		v.IsActive = false // Explicitly set inactive
	case *ScrollBar: // Handle scrollbars added directly
		isFocusable = true
		focusableElement = v
		v.IsActive = false // Explicitly set inactive
	case *Container: // Check container for internal scrollbar
		scrollbar := v.GetScrollbar()
		if scrollbar != nil {
			isFocusable = true
			focusableElement = scrollbar // Make the scrollbar focusable, not the container
			scrollbar.IsActive = false   // Ensure scrollbar starts inactive
		}
		// Note: The Container itself is not directly focusable in this design.
	}

	if isFocusable && focusableElement != nil {
		// Check if this specific focusable element (e.g., button, textbox, scrollbar) is already in the list
		alreadyAdded := false
		for _, fe := range w.focusableElements {
			if fe == focusableElement {
				alreadyAdded = true
				break
			}
		}

		if !alreadyAdded {
			w.focusableElements = append(w.focusableElements, focusableElement)
			// If this is the first focusable element added, focus it immediately
			if w.focusedIndex == -1 {
				w.focusedIndex = 0
				// Activate the first focusable element by setting its IsActive flag
				switch el := w.focusableElements[0].(type) {
				case *Button:
					el.IsActive = true
				case *TextBox:
					el.IsActive = true
				case *CheckBox:
					el.IsActive = true
				case *RadioButton:
					el.IsActive = true
				case *ScrollBar: // This covers both direct and container scrollbars
					el.IsActive = true
				}
			}
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
		switch el := w.focusableElements[w.focusedIndex].(type) {
		case *Button:
			el.IsActive = false
		case *TextBox:
			el.IsActive = false
		case *CheckBox: // Add CheckBox case
			el.IsActive = false
		case *RadioButton: // Add RadioButton case
			el.IsActive = false
		case *ScrollBar: // Add ScrollBar case
			el.IsActive = false
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
		switch el := w.focusableElements[w.focusedIndex].(type) {
		case *Button:
			el.IsActive = true
		case *TextBox:
			el.IsActive = true
		case *CheckBox: // Add CheckBox case
			el.IsActive = true
		case *RadioButton: // Add RadioButton case
			el.IsActive = true
		case *ScrollBar: // Add ScrollBar case
			el.IsActive = true
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
	inputBuf := make([]byte, 6) // Increased buffer for escape sequences (arrows, delete)

	for {
		// Hide cursor before reading input to prevent flicker at old position
		// Render will show it again if a TextBox is active
		fmt.Print(HideCursor())

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

		// Get the currently focused element, if any
		var focusedElement UIElement
		var focusedTextBox *TextBox
		var focusedCheckBox *CheckBox       // Add variable for focused CheckBox
		var focusedRadioButton *RadioButton // Add variable for focused RadioButton
		var focusedScrollBar *ScrollBar     // Add variable for focused ScrollBar
		if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
			focusedElement = w.focusableElements[w.focusedIndex]
			// Check if the focused element is a TextBox and cast it
			if tb, ok := focusedElement.(*TextBox); ok {
				focusedTextBox = tb
			}
			// Check if the focused element is a CheckBox and cast it
			if cb, ok := focusedElement.(*CheckBox); ok {
				focusedCheckBox = cb
			}
			// Check if the focused element is a RadioButton and cast it
			if rb, ok := focusedElement.(*RadioButton); ok {
				focusedRadioButton = rb
			}
			// Check if the focused element is a ScrollBar and cast it
			if sb, ok := focusedElement.(*ScrollBar); ok {
				focusedScrollBar = sb
			}
		}

		// --- Key Handling ---
		// Check if the focused element is an active TextBox first
		if focusedTextBox != nil && focusedTextBox.IsActive {
			isPrintable := n == 1 && key[0] >= 32 && key[0] < 127 // Printable ASCII (excluding DEL)

			if isPrintable {
				// If it's the first keypress in a pristine box, clear it first.
				if focusedTextBox.isPristine {
					focusedTextBox.Text = ""
					focusedTextBox.cursorPos = 0
					focusedTextBox.isPristine = false
				}
				// Insert character at cursor position
				focusedTextBox.Text = focusedTextBox.Text[:focusedTextBox.cursorPos] + string(key[0]) + focusedTextBox.Text[focusedTextBox.cursorPos:]
				focusedTextBox.cursorPos++
				needsRender = true
			} else if n == 1 {
				switch key[0] {
				case 127, 8: // Backspace (DEL or ASCII BS)
					if focusedTextBox.cursorPos > 0 {
						focusedTextBox.Text = focusedTextBox.Text[:focusedTextBox.cursorPos-1] + focusedTextBox.Text[focusedTextBox.cursorPos:]
						focusedTextBox.cursorPos--
						focusedTextBox.isPristine = false // Edited
						needsRender = true
					}
				case '\t': // Tab - Move focus to next element
					w.setFocus(w.focusedIndex + 1)
					needsRender = true
				case '\r': // Enter - Treat like Tab for now (move focus)
					w.setFocus(w.focusedIndex + 1)
					needsRender = true
				case 3: // Ctrl+C - Quit
					shouldQuit = true
				}
			} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
				switch key[2] {
				case 'D': // Left Arrow
					if focusedTextBox.cursorPos > 0 {
						focusedTextBox.cursorPos--
						focusedTextBox.isPristine = false // Interacted
						needsRender = true                // Need re-render to show cursor move
					}
				case 'C': // Right Arrow
					if focusedTextBox.cursorPos < len(focusedTextBox.Text) {
						focusedTextBox.cursorPos++
						focusedTextBox.isPristine = false // Interacted
						needsRender = true                // Need re-render to show cursor move
					}
				case 'Z': // Shift+Tab
					w.setFocus(w.focusedIndex - 1)
					needsRender = true
				}
			} else if n == 4 && key[0] == '\x1b' && key[1] == '[' && key[3] == '~' { // More escape sequences
				switch key[2] {
				case '3': // Delete key (\x1b[3~)
					if focusedTextBox.cursorPos < len(focusedTextBox.Text) {
						focusedTextBox.Text = focusedTextBox.Text[:focusedTextBox.cursorPos] + focusedTextBox.Text[focusedTextBox.cursorPos+1:]
						focusedTextBox.isPristine = false // Edited
						needsRender = true
					}
				}
			}
		} else if focusedScrollBar != nil && focusedScrollBar.IsActive { // Handle ScrollBar input
			if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
				switch key[2] {
				case 'A': // Up Arrow
					focusedScrollBar.SetValue(focusedScrollBar.Value - 1)
					needsRender = true
				case 'B': // Down Arrow
					focusedScrollBar.SetValue(focusedScrollBar.Value + 1)
					needsRender = true
				case 'Z': // Shift+Tab
					w.setFocus(w.focusedIndex - 1)
					needsRender = true
				}
			} else if n == 1 {
				switch key[0] {
				case '\t': // Tab - Move focus to next element
					w.setFocus(w.focusedIndex + 1)
					needsRender = true
				case '\r': // Enter - Treat like Tab for now (move focus)
					w.setFocus(w.focusedIndex + 1)
					needsRender = true
				case 3: // Ctrl+C - Quit
					shouldQuit = true
				case 'q', 'Q': // Quit key
					shouldQuit = true
				}
			}
		} else {
			// --- Input Handling when TextBox/ScrollBar is NOT active (or no focus) ---
			if n == 1 {
				switch key[0] {
				case '\t': // Tab key
					if len(w.focusableElements) > 0 {
						w.setFocus(w.focusedIndex + 1)
						needsRender = true
					}
				case '\r': // Enter key (Carriage Return in raw mode)
					// Activate focused button if it's a button
					if btn, ok := focusedElement.(*Button); ok && btn.IsActive {
						if btn.Action != nil {
							if btn.Action() { // Execute action, check quit signal
								shouldQuit = true
							} else {
								// Action might have changed UI state
								needsRender = true
							}
						}
					} else if focusedCheckBox != nil && focusedCheckBox.IsActive { // Check if it's an active CheckBox
						focusedCheckBox.Checked = !focusedCheckBox.Checked // Toggle state
						needsRender = true
					} else if focusedRadioButton != nil && focusedRadioButton.IsActive { // Check if it's an active RadioButton
						// Find the index of the focused radio button within its group
						targetIndex := -1
						for i, rb := range focusedRadioButton.Group.Buttons {
							if rb == focusedRadioButton {
								targetIndex = i
								break
							}
						}
						if targetIndex != -1 {
							focusedRadioButton.Group.Select(targetIndex) // Select this button in its group
							needsRender = true
						}
						// Optionally move focus to the next element after selection
						// w.setFocus(w.focusedIndex + 1)
						// needsRender = true
					} else {
						// If Enter is pressed and not on an active button, checkbox, radio button, or scrollbar,
						// move focus like Tab.
						w.setFocus(w.focusedIndex + 1)
						needsRender = true
					}
				case 'q', 'Q': // Quit key
					shouldQuit = true
				case 3: // Ctrl+C
					shouldQuit = true
				}
			} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // Check for escape sequences
				switch key[2] {
				case 'Z': // Shift+Tab (Common sequence, might vary)
					if len(w.focusableElements) > 0 {
						w.setFocus(w.focusedIndex - 1)
						needsRender = true
					}
				}
			}
		}

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
