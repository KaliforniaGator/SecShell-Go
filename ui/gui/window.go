package gui

import (
	"bufio" // Keep for potential future use, but not for raw input loop
	"fmt"
	"os"
	"secshell/colors"
	"sort"
	"strings"
	"unicode/utf8"

	// Added for potential brief pauses if needed
	"golang.org/x/term" // Import the term package
	"golang.org/x/text/width"
)

// KeyStrokeHandler defines an interface for custom keyboard input handling.
type KeyStrokeHandler interface {
	// HandleKeyStroke processes a key press.
	// It returns:
	// - handled: true if the key was processed by this handler, false otherwise.
	// - needsRender: true if the window should be re-rendered.
	// - shouldQuit: true if the application should quit.
	HandleKeyStroke(key []byte, w *Window) (handled bool, needsRender bool, shouldQuit bool)
}

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
	buffer            strings.Builder  // Internal buffer for drawing commands
	focusableElements []UIElement      // Slice to hold focusable elements (like buttons)
	focusedIndex      int              // Index of the currently focused element in focusableElements
	KeyHandler        KeyStrokeHandler // Optional custom key stroke handler
	sortedElements    []UIElement      // Cache for z-index sorted elements
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
		KeyHandler:        nil,                  // Initialize custom key handler as nil
	}
}

// SetKeyStrokeHandler sets a custom key stroke handler for the window.
func (w *Window) SetKeyStrokeHandler(handler KeyStrokeHandler) {
	w.KeyHandler = handler
}

// AddElement adds a UIElement to the window.
func (w *Window) AddElement(element UIElement) {
	w.Elements = append(w.Elements, element)

	elementsToAdd := []UIElement{} // Collect focusable elements to add

	switch v := element.(type) {
	case *Button:
		elementsToAdd = append(elementsToAdd, v)
	case *TextBox:
		v.IsActive = false // Explicitly set inactive
		elementsToAdd = append(elementsToAdd, v)
	case *CheckBox:
		v.IsActive = false // Explicitly set inactive
		elementsToAdd = append(elementsToAdd, v)
	case *RadioButton:
		v.IsActive = false // Explicitly set inactive
		elementsToAdd = append(elementsToAdd, v)
	case *ScrollBar: // Handle scrollbars added directly
		v.IsActive = false // Explicitly set inactive
		elementsToAdd = append(elementsToAdd, v)
	case *TextArea: // Add TextArea as a focusable element
		v.IsActive = false // Explicitly set inactive
		elementsToAdd = append(elementsToAdd, v)
	case *Container: // Make the Container AND its ScrollBar focusable
		v.IsActive = false                       // Ensure container starts inactive
		elementsToAdd = append(elementsToAdd, v) // Add the container
		// Check for and add the container's scrollbar
		scrollbar := v.GetScrollbar()
		if scrollbar != nil {
			scrollbar.IsActive = false // Ensure scrollbar starts inactive
			elementsToAdd = append(elementsToAdd, scrollbar)
		}
	case *MenuBar: // Add MenuBar as a focusable element
		v.IsActive = false // Ensure menubar starts inactive
		elementsToAdd = append(elementsToAdd, v)
	case *Prompt: // Add Prompt as a focusable element
		v.SetActive(false) // Ensure prompt starts inactive
		elementsToAdd = append(elementsToAdd, v)
	}

	// Add collected elements to the focus list, checking for duplicates
	for _, focusableElement := range elementsToAdd {
		if focusableElement == nil {
			continue
		}

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
				// (The setFocus function handles the type switching)
				w.setFocus(0) // Call setFocus to activate the first element correctly
			}
		}
	}
}

// RemoveElement removes a UIElement from the window
func (w *Window) RemoveElement(element UIElement) {
	// Remove from main elements slice
	for i, e := range w.Elements {
		if e == element {
			w.Elements = append(w.Elements[:i], w.Elements[i+1:]...)
			break
		}
	}

	// Remove from focusable elements if present
	for i, e := range w.focusableElements {
		if e == element {
			w.focusableElements = append(w.focusableElements[:i], w.focusableElements[i+1:]...)
			if w.focusedIndex >= i {
				w.focusedIndex-- // Adjust focused index if needed
			}
			break
		}
	}
}

// getStringDisplayWidth returns the display width of a string, handling emoji and wide characters
func getStringDisplayWidth(s string) int {
	displayWidth := 0
	for _, r := range s {
		p := width.LookupRune(r)
		switch p.Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			displayWidth += 2
		case width.Neutral:
			if utf8.RuneLen(r) >= 4 { // Most emoji are 4 bytes
				displayWidth += 2
			} else {
				displayWidth++
			}
		default:
			displayWidth++
		}
	}
	return displayWidth
}

// Render draws the window and its elements to the terminal.
func (w *Window) Render() {
	w.buffer.Reset()                   // Clear previous rendering commands
	w.buffer.WriteString(HideCursor()) // Start with cursor hidden by default

	box := BoxTypes[w.BoxStyle]
	fullTitle := w.Icon + " " + w.Title

	// Calculate actual display width of the title
	titleDisplayWidth := getStringDisplayWidth(fullTitle)

	// --- Draw Border and Background ---
	w.buffer.WriteString(w.BorderColor)
	w.buffer.WriteString(w.BgColor) // Set background for the whole area initially

	// Top border with Title
	contentWidth := w.Width // Available space between corners
	leftPadding := 0
	rightPadding := 0

	if contentWidth < 0 {
		contentWidth = 0 // Avoid negative width
	}

	if titleDisplayWidth > contentWidth {
		// Title is too long, truncate it with ellipsis if possible
		if contentWidth > 3 {
			// Smart truncation that respects display width
			truncated := ""
			currentWidth := 0
			for _, r := range fullTitle {
				charWidth := getStringDisplayWidth(string(r))
				if currentWidth+charWidth+3 <= contentWidth {
					truncated += string(r)
					currentWidth += charWidth
				} else {
					break
				}
			}
			fullTitle = truncated + "..."
			titleDisplayWidth = getStringDisplayWidth(fullTitle) // Recalculate after truncation
		} else {
			// If we have very little space, do a hard truncate
			fullTitle = string([]rune(fullTitle)[:contentWidth])
			titleDisplayWidth = contentWidth
		}
	}

	// Calculate padding based on actual display width
	totalPadding := contentWidth - titleDisplayWidth
	leftPadding = totalPadding / 2
	rightPadding = totalPadding - leftPadding - 2

	// Wider emojis
	wideEmojis := []string{
		"🗂️", // Folder
		"📁",  // File Folder
		"📂",  // Open File Folder
		"📊",  // Bar Chart
		"📈",  // Chart Increasing
		"📉",  // Chart Decreasing
		"📅",  // Calendar
		"📆",  // Tear-Off Calendar
		"📇",  // Card Index
		"📋",  // Clipboard
		"📌",  // Pushpin
		"📍",  // Round Pushpin
		"📎",  // Paperclip
		"📏",  // Straight Ruler
		"📐",  // Triangular Ruler
		"📓",  // Notebook
		"📔",  // Notebook with Decorative Cover
		"📒",  // Ledger
		"📚",  // Books
		"📖",  // Open Book
		"📜",  // Scroll
	}

	// Smaller width emojis
	smallEmojis := []string{
		"😀",  // Grinning Face
		"😊",  // Smiling Face with Smiling Eyes
		"👍",  // Thumbs Up
		"❤️", // Red Heart
		"✨",  // Sparkles
		"🌟",  // Glowing Star
		"🔥",  // Fire
		"🎉",  // Party Popper
		"🎈",  // Balloon
		"🌈",  // Rainbow
	}

	// Check for wide emojis and adjust padding
	for _, emoji := range wideEmojis {
		if strings.Contains(fullTitle, emoji) {
			rightPadding += 1 // Adjust for wider emoji visual width
		}
	}

	// Check for small emojis and adjust padding
	for _, emoji := range smallEmojis {
		if strings.Contains(fullTitle, emoji) {
			rightPadding -= 1 // Adjust for smaller emoji visual width
		}
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

	// Sort elements by z-index before rendering
	sortedElements := w.getSortedElements()

	// Set default content color before rendering elements
	w.buffer.WriteString(w.ContentColor)
	for _, element := range sortedElements {
		// Pass the window's buffer, content area origin, and content width
		element.Render(&w.buffer, contentX, contentY, contentWidth)
	}

	// --- Cursor Management ---
	// After rendering all elements, check if any element needs the cursor
	needsCursor := false
	var finalCursorX, finalCursorY int

	// Check for active element that wants the cursor
	for _, element := range w.Elements {
		if cursorManager, ok := element.(CursorManager); ok {
			if cursorManager.NeedsCursor() {
				x, y, valid := cursorManager.GetCursorPosition()
				if valid {
					needsCursor = true
					finalCursorX = x
					finalCursorY = y
					break // Use the first element that needs cursor
				}
			}
		}
	}

	if needsCursor {
		// Position and show cursor
		w.buffer.WriteString(MoveCursorCmd(finalCursorY, finalCursorX))
		w.buffer.WriteString(ShowCursor())
	} else {
		// Ensure cursor is hidden if no element needs it
		w.buffer.WriteString(HideCursor())
	}

	// Reset colors at the end and print the buffer
	w.buffer.WriteString(colors.Reset)
	fmt.Print(w.buffer.String())
}

// Add method to collect all submenus
func (w *Window) getAllElements() []UIElement {
	elements := make([]UIElement, len(w.Elements))
	copy(elements, w.Elements)

	// Find MenuBar and collect all submenus
	for _, element := range w.Elements {
		if menuBar, ok := element.(*MenuBar); ok {
			// Add all open submenus to the elements list
			var collectSubmenus func(menu *Menu)
			collectSubmenus = func(menu *Menu) {
				if menu == nil {
					return
				}
				for _, item := range menu.Items {
					if item.SubMenu != nil && item.SubMenu.IsOpen {
						elements = append(elements, item.SubMenu)
						collectSubmenus(item.SubMenu) // Recursively collect nested submenus
					}
				}
			}
			collectSubmenus(menuBar.Menu)
		}
	}

	return elements
}

// Modify getSortedElements to use getAllElements
func (w *Window) getSortedElements() []UIElement {
	// Get all elements including submenus
	elements := w.getAllElements()

	// Sort elements based on z-index
	sort.SliceStable(elements, func(i, j int) bool {
		iZ := 0
		jZ := 0

		if zi, ok := elements[i].(ZIndexer); ok {
			iZ = zi.GetZIndex()
		}
		if zj, ok := elements[j].(ZIndexer); ok {
			jZ = zj.GetZIndex()
		}

		return iZ < jZ
	})

	return elements
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
		case *CheckBox:
			el.IsActive = false
		case *RadioButton:
			el.IsActive = false
		case *ScrollBar: // Handles both direct and container scrollbars
			el.IsActive = false
		case *Container:
			el.IsActive = false
		case *TextArea: // Handle TextArea focus
			el.IsActive = false
		case *MenuBar: // Handle MenuBar focus
			el.IsActive = false
			el.Deactivate() // Properly deactivate menu bar (closes submenus)
		case *Prompt: // Handle Prompt focus
			el.SetActive(false) // Use the prompt's SetActive method
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
		case *CheckBox:
			el.IsActive = true
		case *RadioButton:
			el.IsActive = true
		case *ScrollBar: // Handles both direct and container scrollbars
			el.IsActive = true
		case *Container:
			el.IsActive = true
		case *TextArea: // Handle TextArea focus
			el.IsActive = true
		case *MenuBar: // Handle MenuBar focus
			el.IsActive = true
			el.Activate() // Properly activate the menu bar
		case *Prompt: // Handle Prompt focus
			el.SetActive(true) // Use the prompt's SetActive method
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

	// Initial render
	w.Render()

	// Buffer for reading input bytes
	inputBuf := make([]byte, 6) // Increased buffer for escape sequences (arrows, delete)

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
		var loopShouldQuit bool = false  // Flag to control quitting the loop for this iteration
		var loopNeedsRender bool = false // Flag to control re-rendering for this iteration

		// --- Custom Key Handler ---
		customKeyProcessed := false
		if w.KeyHandler != nil {
			handled, render, quit := w.KeyHandler.HandleKeyStroke(key, w)
			if handled {
				customKeyProcessed = true
				if render {
					loopNeedsRender = true
				}
				if quit {
					loopShouldQuit = true
				}
			}
		}

		if !customKeyProcessed {
			// --- Original Key Handling Logic ---
			// This block contains the original key handling logic.
			// It will set loopNeedsRender and loopShouldQuit directly.

			// Get the currently focused element, if any
			var focusedElement UIElement
			var focusedTextBox *TextBox
			var focusedCheckBox *CheckBox
			var focusedRadioButton *RadioButton
			var focusedContainer *Container
			var focusedScrollBar *ScrollBar
			var focusedTextArea *TextArea
			var focusedMenuBar *MenuBar // Add variable for focused MenuBar
			var focusedPrompt *Prompt   // Add variable for focused Prompt

			if w.focusedIndex >= 0 && w.focusedIndex < len(w.focusableElements) {
				focusedElement = w.focusableElements[w.focusedIndex]
				// Type assertions to get specific element types
				if tb, ok := focusedElement.(*TextBox); ok {
					focusedTextBox = tb
				}
				if cb, ok := focusedElement.(*CheckBox); ok {
					focusedCheckBox = cb
				}
				if rb, ok := focusedElement.(*RadioButton); ok {
					focusedRadioButton = rb
				}
				if ct, ok := focusedElement.(*Container); ok {
					focusedContainer = ct
				}
				if sb, ok := focusedElement.(*ScrollBar); ok {
					focusedScrollBar = sb
				}
				// Add check for TextArea
				if ta, ok := focusedElement.(*TextArea); ok {
					focusedTextArea = ta
				}
				// Add check for MenuBar
				if mb, ok := focusedElement.(*MenuBar); ok {
					focusedMenuBar = mb
				}
				// Add check for Prompt
				if p, ok := focusedElement.(*Prompt); ok {
					focusedPrompt = p
				}
			}

			// --- Key Handling ---
			// Priority: Active MenuBar > Active TextArea > Active TextBox > Active Container > Active ScrollBar > Other focusable elements
			if focusedMenuBar != nil && focusedMenuBar.IsActive {
				// Handle MenuBar input
				if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrow keys)
					switch key[2] {
					case 'A': // Up Arrow - Move up in menu
						focusedMenuBar.MoveUp()
						loopNeedsRender = true
					case 'B': // Down Arrow - Move down in menu or open submenu
						focusedMenuBar.MoveDown()
						loopNeedsRender = true
					case 'C': // Right Arrow - Move right in menu bar or into submenu
						focusedMenuBar.MoveRight()
						loopNeedsRender = true
					case 'D': // Left Arrow - Move left in menu bar or back from submenu
						focusedMenuBar.MoveLeft()
						loopNeedsRender = true
					case 'Z': // Shift+Tab - Move focus to previous focusable element
						w.setFocus(w.focusedIndex - 1)
						loopNeedsRender = true
					}
				} else if n == 1 {
					switch key[0] {
					case '\t': // Tab - Move focus to next element
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case '\r': // Enter - Activate selected menu item
						shouldQuit := focusedMenuBar.ActivateSelected()
						loopNeedsRender = true
						if shouldQuit {
							loopShouldQuit = true
						}
					case 27: // Escape - Deactivate menu
						focusedMenuBar.Deactivate()
						loopNeedsRender = true
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					}
				}
			} else if focusedPrompt != nil && focusedPrompt.IsActive {
				// Handle Prompt input
				if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrow keys)
					switch key[2] {
					case 'C': // Right Arrow - Select next button
						focusedPrompt.SelectNext()
						loopNeedsRender = true
					case 'D': // Left Arrow - Select previous button
						focusedPrompt.SelectPrevious()
						loopNeedsRender = true
					case 'Z': // Shift+Tab - Move focus to previous element
						if !focusedPrompt.IsModal() { // Only allow focus change if not modal
							w.setFocus(w.focusedIndex - 1)
							loopNeedsRender = true
						}
					}
				} else if n == 1 {
					switch key[0] {
					case '\t': // Tab - Move focus to next element or between buttons
						if focusedPrompt.IsModal() {
							focusedPrompt.SelectNext()
						} else {
							w.setFocus(w.focusedIndex + 1)
						}
						loopNeedsRender = true
					case '\r': // Enter - Activate selected button
						shouldQuit := focusedPrompt.ActivateSelected()
						loopNeedsRender = true
						// If the action signaled to quit, set the quit flag
						if shouldQuit {
							loopShouldQuit = true
						}
					case 27: // Escape - Close non-modal prompt
						if !focusedPrompt.IsModal() {
							focusedPrompt.SetActive(false)
							w.setFocus(w.focusedIndex + 1)
							loopNeedsRender = true
						}
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					}
				}
			} else if focusedTextArea != nil && focusedTextArea.IsActive {
				// Handle TextArea input
				isPrintable := n == 1 && key[0] >= 32 && key[0] < 127 // Printable ASCII (excluding DEL)

				if isPrintable {
					// Insert character at cursor position
					focusedTextArea.InsertChar(rune(key[0]))
					loopNeedsRender = true
				} else if n == 1 {
					switch key[0] {
					case 127, 8: // Backspace (DEL or ASCII BS)
						focusedTextArea.DeleteChar()
						loopNeedsRender = true
					case '\t': // Tab - Move focus to next element
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case '\r': // Enter - Insert newline
						focusedTextArea.InsertChar('\n')
						loopNeedsRender = true
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					}
				} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
					switch key[2] {
					case 'D': // Left Arrow
						focusedTextArea.MoveCursorLeft()
						loopNeedsRender = true
					case 'C': // Right Arrow
						focusedTextArea.MoveCursorRight()
						loopNeedsRender = true
					case 'A': // Up Arrow
						focusedTextArea.MoveCursorUp()
						loopNeedsRender = true
					case 'B': // Down Arrow
						focusedTextArea.MoveCursorDown()
						loopNeedsRender = true
					case 'Z': // Shift+Tab
						w.setFocus(w.focusedIndex - 1)
						loopNeedsRender = true
					}
				} else if n == 4 && key[0] == '\x1b' && key[1] == '[' && key[3] == '~' { // More escape sequences
					switch key[2] {
					case '3': // Delete key (\x1b[3~)
						focusedTextArea.DeleteForward()
						loopNeedsRender = true
					}
				}
			} else if focusedTextBox != nil && focusedTextBox.IsActive {
				// ... (TextBox input handling remains the same) ...
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
					loopNeedsRender = true
				} else if n == 1 {
					switch key[0] {
					case 127, 8: // Backspace (DEL or ASCII BS)
						if focusedTextBox.cursorPos > 0 {
							focusedTextBox.Text = focusedTextBox.Text[:focusedTextBox.cursorPos-1] + focusedTextBox.Text[focusedTextBox.cursorPos:]
							focusedTextBox.cursorPos--
							focusedTextBox.isPristine = false // Edited
							loopNeedsRender = true
						}
					case '\t': // Tab - Move focus to next element
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case '\r': // Enter - Treat like Tab for now (move focus)
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					}
				} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
					switch key[2] {
					case 'D': // Left Arrow
						if focusedTextBox.cursorPos > 0 {
							focusedTextBox.cursorPos--
							focusedTextBox.isPristine = false // Interacted
							loopNeedsRender = true            // Need re-render to show cursor move
						}
					case 'C': // Right Arrow
						if focusedTextBox.cursorPos < len(focusedTextBox.Text) {
							focusedTextBox.cursorPos++
							focusedTextBox.isPristine = false // Interacted
							loopNeedsRender = true            // Need re-render to show cursor move
						}
					case 'Z': // Shift+Tab
						w.setFocus(w.focusedIndex - 1)
						loopNeedsRender = true
					}
				} else if n == 4 && key[0] == '\x1b' && key[1] == '[' && key[3] == '~' { // More escape sequences
					switch key[2] {
					case '3': // Delete key (\x1b[3~)
						if focusedTextBox.cursorPos < len(focusedTextBox.Text) {
							focusedTextBox.Text = focusedTextBox.Text[:focusedTextBox.cursorPos] + focusedTextBox.Text[focusedTextBox.cursorPos+1:]
							focusedTextBox.isPristine = false // Edited
							loopNeedsRender = true
						}
					}
				}
			} else if focusedContainer != nil && focusedContainer.IsActive { // Handle Container input
				if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
					switch key[2] {
					case 'A': // Up Arrow - Select previous item
						focusedContainer.SelectPrevious()
						loopNeedsRender = true
					case 'B': // Down Arrow - Select next item
						focusedContainer.SelectNext()
						loopNeedsRender = true
					case 'Z': // Shift+Tab
						w.setFocus(w.focusedIndex - 1)
						loopNeedsRender = true
					}
				} else if n == 1 {
					switch key[0] {
					case '\t': // Tab - Move focus to next element
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case '\r': // Enter - Trigger item selection callback and move focus
						// Call the OnItemSelected callback if it exists and selection is valid
						if focusedContainer.OnItemSelected != nil && focusedContainer.SelectedIndex >= 0 {
							focusedContainer.OnItemSelected(focusedContainer.SelectedIndex)
							// Callback might have updated UI elements, so render is needed
							loopNeedsRender = true
						}
						// Ensure render happens even if callback didn't exist (focus changed)
						loopNeedsRender = true
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					case 'q', 'Q': // Quit key
						loopShouldQuit = true
					}
				}
				// Potentially add PageUp/PageDown handling here later
			} else if focusedScrollBar != nil && focusedScrollBar.IsActive { // Handle ScrollBar input
				if n == 3 && key[0] == '\x1b' && key[1] == '[' { // ANSI Escape sequences (Arrows, etc.)
					// NEW: Only process scroll actions if the scrollbar is visible
					if focusedScrollBar.Visible {
						switch key[2] {
						case 'A': // Up Arrow - Scroll up
							focusedScrollBar.SetValue(focusedScrollBar.Value - 1)
							loopNeedsRender = true
						case 'B': // Down Arrow - Scroll down
							focusedScrollBar.SetValue(focusedScrollBar.Value + 1)
							loopNeedsRender = true
						}
					}
					// Handle focus navigation regardless of visibility
					switch key[2] {
					case 'Z': // Shift+Tab
						w.setFocus(w.focusedIndex - 1)
						loopNeedsRender = true
					}
				} else if n == 1 {
					// Handle focus navigation / quit regardless of visibility
					switch key[0] {
					case '\t': // Tab - Move focus to next element
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case '\r': // Enter - Treat like Tab for now (move focus away from scrollbar)
						w.setFocus(w.focusedIndex + 1)
						loopNeedsRender = true
					case 3: // Ctrl+C - Quit
						loopShouldQuit = true
					case 'q', 'Q': // Quit key
						loopShouldQuit = true
					}
				}
				// Potentially add PageUp/PageDown handling here later (checking Visible)
			} else {
				// --- Input Handling when TextBox/Container/ScrollBar is NOT active (handles Buttons, CheckBoxes, RadioButtons, etc.) ---
				if n == 1 {
					switch key[0] {
					case '\t': // Tab key
						if len(w.focusableElements) > 0 {
							w.setFocus(w.focusedIndex + 1)
							loopNeedsRender = true
						}
					case '\r': // Enter key (Carriage Return in raw mode)
						// Activate focused button if it's a button
						if btn, ok := focusedElement.(*Button); ok && btn.IsActive {
							if btn.Action != nil {
								// Restore terminal before action if it prints outside the UI area
								term.Restore(fd, oldState)
								fmt.Print(ClearScreenAndBuffer()) // Clear UI before action output

								quitAction := btn.Action() // Execute action

								// If action didn't quit, re-setup terminal and UI
								if !quitAction {
									_, err = term.MakeRaw(fd) // Re-enter raw mode
									if err != nil {
										fmt.Printf("Error re-entering raw mode: %v\n", err)
										loopShouldQuit = true // Quit if we can't restore raw mode
									} else {
										loopNeedsRender = true // Re-render the UI
									}
								} else {
									loopShouldQuit = true // Action signaled quit
								}
							}
						} else if focusedCheckBox != nil && focusedCheckBox.IsActive { // Check if it's an active CheckBox
							focusedCheckBox.Checked = !focusedCheckBox.Checked // Toggle state
							loopNeedsRender = true
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
								loopNeedsRender = true
							}
							// Optionally move focus to the next element after selection
							// w.setFocus(w.focusedIndex + 1)
							// loopNeedsRender = true
						} else {
							// If Enter is pressed and not on an active Button, CheckBox, RadioButton,
							// move focus like Tab.
							w.setFocus(w.focusedIndex + 1)
							loopNeedsRender = true
						}
					case 'q', 'Q': // Quit key
						loopShouldQuit = true
					case 3: // Ctrl+C
						loopShouldQuit = true
					}
				} else if n == 3 && key[0] == '\x1b' && key[1] == '[' { // Check for escape sequences (Shift+Tab)
					switch key[2] {
					case 'Z': // Shift+Tab (Common sequence, might vary)
						if len(w.focusableElements) > 0 {
							w.setFocus(w.focusedIndex - 1)
							loopNeedsRender = true
						}
					}
				}
			}
		} // end if !customKeyProcessed

		// --- Loop Control and Rendering ---
		if loopShouldQuit {
			break // Exit the interaction loop
		}

		// Re-render ONLY if necessary
		if loopNeedsRender {
			// Optimization: If only cursor moved in textbox, could potentially just move cursor
			// But full render is safer for now.
			w.Render() // Re-render the window state
		}
	}

	// Cleanup is handled by defers (Restore terminal state, Show cursor)
	// Clear the screen after finishing interaction
	fmt.Print(ClearScreenAndBuffer())
	fmt.Print(ShowCursor()) // Explicitly show cursor after clearing
}
