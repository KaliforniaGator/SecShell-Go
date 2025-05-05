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

// TextBox represents an editable text input field.
type TextBox struct {
	Text        string
	Color       string
	ActiveColor string // Color when selected/active
	X, Y        int    // Position relative to window content area
	Width       int
	IsActive    bool // State for rendering/input handling
	cursorPos   int  // Position of the cursor within the text
	isPristine  bool // Flag to track if default text is present and untouched
}

// NewTextBox creates a new TextBox instance.
func NewTextBox(initialText string, x, y, width int, color, activeColor string) *TextBox {
	tb := &TextBox{
		Text:        initialText,
		X:           x,
		Y:           y,
		Width:       width,
		Color:       color,
		ActiveColor: activeColor,
		IsActive:    false,
		cursorPos:   len(initialText), // Cursor at the end initially
		isPristine:  true,             // Initially contains default text
	}
	// Clamp initial cursor position
	if tb.cursorPos > len(tb.Text) {
		tb.cursorPos = len(tb.Text)
	}
	return tb
}

// Render draws the textbox element.
func (tb *TextBox) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + tb.X
	absY := winY + tb.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	renderColor := tb.Color
	if tb.IsActive {
		renderColor = tb.ActiveColor
	}
	buffer.WriteString(renderColor)

	// --- Text Rendering with Scrolling ---
	textLen := len(tb.Text)
	viewStart := 0 // Index in tb.Text that corresponds to the start of the visible area

	// Adjust viewStart based on cursor position to keep cursor visible
	if tb.cursorPos >= tb.Width {
		viewStart = tb.cursorPos - tb.Width + 1
	}
	if viewStart < 0 { // Should not happen with above logic, but safety check
		viewStart = 0
	}
	// Ensure viewStart doesn't go beyond possible text start
	if viewStart > textLen {
		viewStart = textLen
	}

	viewEnd := viewStart + tb.Width
	if viewEnd > textLen {
		viewEnd = textLen
	}

	// Get the visible portion of the text
	visibleText := ""
	if viewStart < textLen {
		visibleText = tb.Text[viewStart:viewEnd]
	}

	// Render the visible text and padding
	buffer.WriteString(visibleText)
	buffer.WriteString(strings.Repeat(" ", tb.Width-len(visibleText)))
	// --- End Text Rendering ---

	// --- Cursor Rendering ---
	if tb.IsActive {
		// Calculate cursor position relative to the *visible* text area
		cursorRenderPos := tb.cursorPos - viewStart
		if cursorRenderPos >= 0 && cursorRenderPos < tb.Width {
			buffer.WriteString(MoveCursorCmd(absY, absX+cursorRenderPos))
			buffer.WriteString(ShowCursor()) // Make cursor visible at the calculated position
		} else {
			// If cursor is somehow outside visible area (e.g., exactly at tb.Width),
			// place it at the end of the visible area.
			buffer.WriteString(MoveCursorCmd(absY, absX+tb.Width-1))
			buffer.WriteString(ShowCursor())
		}
	}
	// --- End Cursor Rendering ---

	buffer.WriteString(colors.Reset) // Reset color
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

	// Check if the element is focusable (Buttons, TextBoxes)
	isFocusable := false
	switch v := element.(type) {
	case *Button:
		isFocusable = true
	case *TextBox:
		isFocusable = true
		// Ensure cursor is initially hidden for inactive textboxes
		v.IsActive = false // Explicitly set inactive
	}

	if isFocusable {
		w.focusableElements = append(w.focusableElements, element)
		// If this is the first focusable element, focus it
		if w.focusedIndex == -1 {
			w.focusedIndex = 0
			// Activate the first focusable element
			switch el := w.focusableElements[0].(type) {
			case *Button:
				el.IsActive = true
			case *TextBox:
				el.IsActive = true // Activate and make cursor potentially visible on first render
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
		if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
			focusedElement = w.focusableElements[w.focusedIndex]
			// Check if the focused element is a TextBox and cast it
			if tb, ok := focusedElement.(*TextBox); ok {
				focusedTextBox = tb
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
		} else {
			// --- Input Handling when TextBox is NOT active (or no focus) ---
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
					} else {
						// If Enter is pressed and not on an active button,
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

// TestWindowApp demonstrates creating and rendering a sample window.
func TestWindowApp() {
	// Clear screen before drawing
	fmt.Print(ClearScreenAndBuffer())

	// Get terminal dimensions
	termWidth := GetTerminalWidth()
	termHeight := GetTerminalHeight()

	// Define window dimensions and position (centered)
	winWidth := termWidth / 2
	if winWidth < 50 { // Ensure enough width for label + textbox
		winWidth = 50
	}
	winHeight := termHeight / 2
	if winHeight < 17 { // Increased min height again
		winHeight = 17
	}
	winX := (termWidth - winWidth) / 2
	winY := (termHeight - winHeight) / 2

	// Create the window
	testWin := NewWindow("📝", "Input Test", winX, winY, winWidth, winHeight,
		"double", colors.BoldCyan, colors.BoldYellow, colors.BgBlack, colors.White)

	// --- Add Elements ---
	infoLabel := NewLabel("Tab/S-Tab: Cycle | Enter: Next/Activate | Arrows: Move Cursor | q/Ctrl+C: Quit", 1, 1, colors.Green)
	testWin.AddElement(infoLabel)

	// --- First TextBox ---
	nameLabel := NewLabel("Enter Name:", 1, 3, colors.White)
	testWin.AddElement(nameLabel)
	textBoxX := len(nameLabel.Text) + 2
	textBoxWidth := winWidth - 2 - textBoxX - 1
	if textBoxWidth < 10 {
		textBoxWidth = 10
	}
	nameTextBox := NewTextBox("<Type name here>", textBoxX, 3, textBoxWidth, colors.BgWhite+colors.Black, colors.BgCyan+colors.BoldBlack)
	testWin.AddElement(nameTextBox)

	// --- Second TextBox ---
	emailLabelY := 5 // Position below the first textbox
	emailLabel := NewLabel("Enter Email:", 1, emailLabelY, colors.White)
	testWin.AddElement(emailLabel)
	// Use same X and Width calculation, adjust Y
	emailTextBoxX := len(emailLabel.Text) + 2
	emailTextBoxWidth := winWidth - 2 - emailTextBoxX - 1
	if emailTextBoxWidth < 10 {
		emailTextBoxWidth = 10
	}
	emailTextBox := NewTextBox("<Type email here>", emailTextBoxX, emailLabelY, emailTextBoxWidth, colors.BgWhite+colors.Black, colors.BgCyan+colors.BoldBlack)
	testWin.AddElement(emailTextBox) // Add second textbox

	// --- Buttons ---
	buttonWidth := 12

	contentWidth := winWidth - 2
	// Center buttons horizontally below the textbox area
	totalButtonWidth := buttonWidth*2 + 2 // Width of two buttons + space between
	buttonStartX := (contentWidth - totalButtonWidth) / 2
	submitButtonX := buttonStartX
	quitButtonX := buttonStartX + buttonWidth + 2

	// Adjust button Y position further down
	actionButtonY := winHeight - 6 // Position near bottom
	submitButton := NewButton("Submit", submitButtonX, actionButtonY, buttonWidth, colors.BoldGreen, colors.BgGreen+colors.BoldWhite, func() bool {
		// Access the values from both textboxes
		submittedName := nameTextBox.Text
		if nameTextBox.isPristine {
			submittedName = "" // Treat pristine as empty
		}
		submittedEmail := emailTextBox.Text
		if emailTextBox.isPristine {
			submittedEmail = "" // Treat pristine as empty
		}

		infoLabel.Text = fmt.Sprintf("Name: '%s', Email: '%s' | Tab/S-Tab, Enter, q/Ctrl+C", submittedName, submittedEmail)
		infoLabel.Color = colors.BoldGreen
		return false // Don't quit
	})
	testWin.AddElement(submitButton)

	// Button 2: Quit Button
	quitButtonY := actionButtonY
	quitButton := NewButton("Quit App", quitButtonX, quitButtonY, buttonWidth, colors.BoldRed, colors.BgRed+colors.BoldWhite, func() bool {
		infoLabel.Text = "Quitting..."
		infoLabel.Color = colors.BoldRed
		testWin.Render() // Render the "Quitting..." message
		time.Sleep(300 * time.Millisecond)
		return true // Action returns true to signal quitting
	})
	testWin.AddElement(quitButton)

	// --- Start Interaction ---
	// WindowActions now handles raw input, rendering loop, focus, and cleanup.
	testWin.WindowActions()

	// Code here runs after WindowActions loop finishes
	fmt.Println("Application finished.")
	// Access the final state of both textboxes
	finalName := nameTextBox.Text
	if nameTextBox.isPristine {
		finalName = "" // Treat pristine state as empty submission
	}
	finalEmail := emailTextBox.Text
	if emailTextBox.isPristine {
		finalEmail = "" // Treat pristine state as empty submission
	}
	fmt.Printf("Final Name content: '%s'\n", finalName)
	fmt.Printf("Final Email content: '%s'\n", finalEmail)
}
