package tools

import (
	"bufio"
	"fmt"
	"os"
	"secshell/ui/gui"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// Key constants for special keys (using common ANSI escape sequences)
const (
	keyArrowLeft       rune = 1000 // Arbitrary values > 255
	keyArrowRight      rune = 1001
	keyArrowUp         rune = 1002
	keyArrowDown       rune = 1003
	keyHome            rune = 1004
	keyEnd             rune = 1005
	keyPageUp          rune = 1006
	keyPageDown        rune = 1007
	keyBackspace       rune = 127  // ASCII Backspace
	keyDelete          rune = 1008 // Often sends \x1b[3~
	keyEnter           rune = 13   // ASCII Carriage Return (often used for Enter)
	keyCtrlS           rune = 19   // Ctrl+S
	keyCtrlQ           rune = 17   // Ctrl+Q
	keyCtrlL           rune = 12   // Ctrl+L (Select Line)
	keyCtrlA           rune = 1    // Ctrl+A (Select All)
	keyEscape          rune = 27   // ASCII Escape
	keyShiftArrowLeft  rune = 1009 // Arbitrary
	keyShiftArrowRight rune = 1010 // Arbitrary
)

// Position represents a coordinate in the text buffer
type Position struct {
	line int // Line index (0-based)
	col  int // Rune index within the line (0-based)
}

// Editor holds the state of the text editor
type Editor struct {
	lines           []string // Content of the file, line by line
	cursorX         int      // Horizontal cursor position (rune index)
	cursorY         int      // Vertical cursor position (line index)
	offsetX         int      // Horizontal scroll offset (rune index)
	offsetY         int      // Vertical scroll offset (line index)
	termWidth       int      // Terminal width
	termHeight      int      // Terminal height (usable area)
	fileName        string   // Name of the file being edited
	statusMsg       string   // Message to display in the status bar
	isDirty         bool     // True if the buffer has been modified since the last save
	originalTerm    *term.State
	selection       Selection // Added selection state
	selectionAnchor Position  // Anchor point for shift-selection
}

// Selection holds the start and end points of selected text
type Selection struct {
	active bool     // Is selection currently active?
	start  Position // Start position of selection
	end    Position // End position of selection
}

// NewEditor initializes a new editor instance
func NewEditor() *Editor {
	w, h := gui.GetTerminalWidth(), gui.GetTerminalHeight()
	return &Editor{
		lines:      []string{""}, // Start with one empty line
		cursorX:    0,
		cursorY:    0,
		offsetX:    0,
		offsetY:    0,
		termWidth:  w,
		termHeight: h - 2, // Reserve space for status bar and command line/prompt
		fileName:   "[No Name]",
		statusMsg:  "HELP: Ctrl+S = Save | Ctrl+Q = Quit | Ctrl+L = Select Line | Ctrl+A = Select All | Shift+Arrows = Select | Esc = Cancel Select",
		isDirty:    false,
		selection: Selection{ // Initialize selection
			active: false,
		},
	}
}

// enableRawMode puts the terminal into raw mode
func (e *Editor) enableRawMode() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	e.originalTerm = oldState
	return nil
}

// disableRawMode restores the terminal to its original state
func (e *Editor) disableRawMode() {
	if e.originalTerm != nil {
		// Clear the screen and move cursor to top-left before restoring
		// Use ClearScreenAndBuffer to clear scrollback as well on exit
		fmt.Print(gui.ClearScreenAndBuffer())
		fmt.Print(gui.MoveCursorCmd(0, 0))
		term.Restore(int(os.Stdin.Fd()), e.originalTerm)
	}
}

// Run starts the editor's main loop
func (e *Editor) Run() error {
	if err := e.enableRawMode(); err != nil {
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}
	// disableRawMode now handles screen clearing and buffer clearing
	defer e.disableRawMode()

	reader := bufio.NewReader(os.Stdin)

	for {
		// Refresh screen before reading input to show current state
		e.refreshScreen()

		// Read input
		runeVal, err := e.readKey(reader) // Use readKey to handle escape sequences
		if err != nil {
			// Handle potential errors (e.g., EOF)
			break
		}

		// Process input
		quit := e.processKeyPress(runeVal)
		if quit {
			// Handle unsaved changes before quitting
			if e.isDirty {
				// Basic prompt, replace with a better UI later
				e.statusMsg = "Save changes before quitting? (y/n)"
				e.refreshScreen()
				// Simplified: read one more rune for y/n
				confirm, _, _ := reader.ReadRune()
				if confirm == 'y' || confirm == 'Y' {
					if err := e.SaveFile(); err != nil {
						e.statusMsg = fmt.Sprintf("Error saving: %s. Quit anyway? (y/n)", err)
						e.refreshScreen()
						confirmQuit, _, _ := reader.ReadRune()
						if confirmQuit != 'y' && confirmQuit != 'Y' {
							e.statusMsg = "Quit aborted."
							continue // Abort quit
						}
					}
				} else if confirm != 'n' && confirm != 'N' {
					e.statusMsg = "Quit aborted."
					continue // Abort quit if not 'n'
				}
			}
			break // Exit loop
		}
	}

	return nil // Or return an error if something went wrong
}

// readKey reads a single keypress, handling escape sequences for special keys.
func (e *Editor) readKey(reader *bufio.Reader) (rune, error) {
	r, _, err := reader.ReadRune()
	if err != nil {
		return 0, err
	}

	if r == keyEscape { // Check for escape sequence
		buf := make([]byte, 5) // Read up to 5 more bytes for sequences like \x1b[1;2D
		reader.Read(buf[:1])   // Try reading '[' or 'O'

		if buf[0] == '[' {
			n, _ := reader.Read(buf[1:5]) // Read up to 4 more bytes
			fullSeq := append([]byte{'['}, buf[1:1+n]...)

			switch string(fullSeq) {
			case "[A":
				return keyArrowUp, nil
			case "[B":
				return keyArrowDown, nil
			case "[C":
				return keyArrowRight, nil
			case "[D":
				return keyArrowLeft, nil
			case "[H", "[1~", "[7~": // Home variations
				return keyHome, nil
			case "[F", "[4~", "[8~": // End variations
				return keyEnd, nil
			case "[5~":
				return keyPageUp, nil
			case "[6~":
				return keyPageDown, nil
			case "[3~":
				return keyDelete, nil
			case "[1;2C": // Shift+Right (common variant)
				return keyShiftArrowRight, nil
			case "[1;2D": // Shift+Left (common variant)
				return keyShiftArrowLeft, nil
			}
			return keyEscape, nil // Unrecognized CSI sequence

		} else if buf[0] == 'O' {
			n, _ := reader.Read(buf[1:2])
			if n == 1 {
				switch buf[1] {
				case 'H':
					return keyHome, nil
				case 'F':
					return keyEnd, nil
				}
			}
			return keyEscape, nil // Unrecognized 'O' sequence
		}

		return keyEscape, nil
	}

	return r, nil
}

// scroll adjusts offsetX and offsetY if the cursor is outside the visible window
func (e *Editor) scroll() {
	// Vertical scroll
	if e.cursorY < e.offsetY {
		e.offsetY = e.cursorY
	}
	if e.cursorY >= e.offsetY+e.termHeight {
		e.offsetY = e.cursorY - e.termHeight + 1
	}
	// Ensure offsetY doesn't go negative
	if e.offsetY < 0 {
		e.offsetY = 0
	}

	// Horizontal scroll
	if e.cursorX < e.offsetX {
		e.offsetX = e.cursorX
	}
	if e.cursorX >= e.offsetX+e.termWidth {
		e.offsetX = e.cursorX - e.termWidth + 1
	}
}

// refreshScreen clears the terminal and redraws the editor UI
func (e *Editor) refreshScreen() {
	e.scroll() // Adjust scroll offsets before drawing

	gui.HideCursor()
	// Use a buffer to minimize flicker (optional but good practice)
	var sb strings.Builder
	sb.WriteString(gui.ClearScreen())       // \x1b[2J
	sb.WriteString(gui.MoveCursorCmd(0, 0)) // \x1b[H

	e.drawRows(&sb)
	e.drawStatusBar(&sb)

	// Position cursor after drawing everything else
	cursorDrawX := e.cursorX - e.offsetX
	cursorDrawY := e.cursorY - e.offsetY
	sb.WriteString(gui.MoveCursorCmd(cursorDrawY, cursorDrawX)) // Note: row, col order for ANSI

	sb.WriteString(gui.ShowCursor()) // \x1b[?25h

	// Write the buffer to the terminal at once
	fmt.Print(sb.String())
}

// normalizeSelection ensures start is before end
func (s *Selection) normalize() {
	if s.start.line > s.end.line || (s.start.line == s.end.line && s.start.col > s.end.col) {
		s.start, s.end = s.end, s.start
	}
}

// isSelected checks if a given position (lineIdx, runeIdx) is within the selection
func (e *Editor) isSelected(lineIdx, runeIdx int) bool {
	if !e.selection.active {
		return false
	}
	sel := e.selection
	sel.normalize() // Ensure start is before end for comparison

	currentPos := Position{line: lineIdx, col: runeIdx}

	// Check if the position is after the start and before the end
	if currentPos.line > sel.start.line && currentPos.line < sel.end.line {
		return true // Entire line between start and end lines is selected
	}
	if currentPos.line == sel.start.line && currentPos.line == sel.end.line {
		// Selection within a single line
		return runeIdx >= sel.start.col && runeIdx < sel.end.col
	}
	if currentPos.line == sel.start.line {
		// On the start line
		return runeIdx >= sel.start.col
	}
	if currentPos.line == sel.end.line {
		// On the end line
		return runeIdx < sel.end.col
	}

	return false // Should not be reached if logic is correct
}

// drawRows draws the text buffer content onto the screen buffer
func (e *Editor) drawRows(sb *strings.Builder) {
	for y := 0; y < e.termHeight; y++ {
		fileRow := y + e.offsetY
		if fileRow >= len(e.lines) {
			// Draw tildes for lines beyond the buffer
			if e.selection.active && fileRow == 0 && len(e.lines) == 0 {
				sb.WriteString(gui.ReverseVideo())
				sb.WriteString("~")
				sb.WriteString(gui.ResetVideo())
			} else {
				sb.WriteString("~")
			}
		} else {
			line := e.lines[fileRow]
			runes := []rune(line) // Work with runes for correct indexing/slicing
			lineLen := len(runes)

			for x := 0; x < e.termWidth; x++ {
				fileCol := x + e.offsetX // Actual column index in the file line
				if fileCol < lineLen {
					runeToDraw := runes[fileCol]
					if e.isSelected(fileRow, fileCol) {
						sb.WriteString(gui.ReverseVideo()) // Start highlight
						sb.WriteRune(runeToDraw)
						sb.WriteString(gui.ResetVideo()) // End highlight
					} else {
						sb.WriteRune(runeToDraw)
					}
				} else if fileCol == lineLen && e.isSelected(fileRow, fileCol) {
					sb.WriteString(gui.ReverseVideo())
					sb.WriteString(" ") // Represent selected newline/end-of-line
					sb.WriteString(gui.ResetVideo())
				}
			}
		}
		sb.WriteString(gui.ClearLineSuffix()) // Clear rest of the terminal line
		sb.WriteString("\r\n")                // Use carriage return and newline
	}
}

// drawStatusBar draws the status bar at the bottom into the screen buffer
func (e *Editor) drawStatusBar(sb *strings.Builder) {
	sb.WriteString(gui.MoveCursorCmd(e.termHeight, 0)) // Move to the status bar line

	status := fmt.Sprintf(" %.80s", e.statusMsg) // Truncate status message
	if len([]rune(status)) > e.termWidth {
		status = string([]rune(status)[:e.termWidth])
	}

	fileNameInfo := fmt.Sprintf(" %s %s ", e.fileName, map[bool]string{true: "(modified)", false: ""}[e.isDirty])
	posInfo := fmt.Sprintf(" %d:%d ", e.cursorY+1, e.cursorX+1) // 1-based indexing for display

	middleWidth := e.termWidth - len([]rune(status)) - len([]rune(posInfo))
	if middleWidth < 0 {
		middleWidth = 0 // Avoid negative repeats
	}

	maxFilenameWidth := middleWidth - 2 // Account for spaces around filename
	if maxFilenameWidth < 1 {
		maxFilenameWidth = 1
	}
	displayFileName := fmt.Sprintf(" %.*s ", maxFilenameWidth, fileNameInfo)
	if len([]rune(displayFileName)) > middleWidth {
		displayFileName = string([]rune(displayFileName)[:middleWidth]) // Final truncation if needed
	}

	padding := middleWidth - len([]rune(displayFileName))
	if padding < 0 {
		padding = 0
	}

	bar := status + strings.Repeat(" ", padding) + displayFileName + posInfo

	if len([]rune(bar)) > e.termWidth {
		bar = string([]rune(bar)[:e.termWidth])
	} else {
		bar += strings.Repeat(" ", e.termWidth-len([]rune(bar)))
	}

	sb.WriteString("\x1b[7m") // Start reverse video
	sb.WriteString(bar)
	sb.WriteString("\x1b[m") // End reverse video
}

// moveCursorOnly handles cursor movement keys (basic movement without selection)
func (e *Editor) moveCursorOnly(key rune) {
	currentLineRunes := []rune{}
	if e.cursorY >= 0 && e.cursorY < len(e.lines) {
		currentLineRunes = []rune(e.lines[e.cursorY])
	}
	currentLineLen := len(currentLineRunes)

	switch key {
	case keyArrowLeft, keyShiftArrowLeft:
		if e.cursorX > 0 {
			e.cursorX--
		} else if e.cursorY > 0 { // Move to end of previous line
			e.cursorY--
			e.cursorX = len([]rune(e.lines[e.cursorY]))
		}
	case keyArrowRight, keyShiftArrowRight:
		if e.cursorX < currentLineLen {
			e.cursorX++
		} else if e.cursorY < len(e.lines)-1 { // Move to start of next line
			e.cursorY++
			e.cursorX = 0
		}
	case keyArrowUp:
		if e.cursorY > 0 {
			e.cursorY--
		}
	case keyArrowDown:
		if e.cursorY < len(e.lines)-1 {
			e.cursorY++
		}
	case keyHome:
		e.cursorX = 0
	case keyEnd:
		e.cursorX = currentLineLen
	case keyPageUp:
		targetY := e.cursorY - e.termHeight
		if targetY < 0 {
			targetY = 0
		}
		e.cursorY = targetY
	case keyPageDown:
		targetY := e.cursorY + e.termHeight
		if targetY >= len(e.lines) {
			targetY = len(e.lines) - 1
		}
		e.cursorY = targetY
	}

	if e.cursorY >= 0 && e.cursorY < len(e.lines) {
		newLineLen := len([]rune(e.lines[e.cursorY]))
		if e.cursorX > newLineLen {
			e.cursorX = newLineLen
		}
	} else if e.cursorY >= len(e.lines) && len(e.lines) > 0 {
		e.cursorY = len(e.lines) - 1
		e.cursorX = len([]rune(e.lines[e.cursorY]))
	} else if len(e.lines) == 0 {
		e.cursorY = 0
		e.cursorX = 0
	}
}

// insertChar inserts a character at the cursor position
func (e *Editor) insertChar(r rune) {
	if e.selection.active {
		e.deleteSelection() // Delete selection before inserting
	} else {
		e.deactivateSelection() // Ensure selection is off if not deleting it
	}

	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		// If lines are empty after deleting selection, add a line
		if len(e.lines) == 0 {
			e.lines = append(e.lines, "")
			e.cursorY = 0
			e.cursorX = 0
		} else {
			return // Should not happen otherwise
		}
	}
	line := e.lines[e.cursorY]
	runes := []rune(line)

	if e.cursorX < 0 {
		e.cursorX = 0
	}
	if e.cursorX > len(runes) {
		e.cursorX = len(runes)
	}

	newRunes := append(runes[:e.cursorX], append([]rune{r}, runes[e.cursorX:]...)...)
	e.lines[e.cursorY] = string(newRunes)
	e.cursorX++
	e.isDirty = true
	e.statusMsg = ""
}

// insertNewline handles the Enter key
func (e *Editor) insertNewline() {
	if e.selection.active {
		e.deleteSelection() // Delete selection before inserting newline
	} else {
		e.deactivateSelection()
	}

	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		// If lines are empty after deleting selection, add a line
		if len(e.lines) == 0 {
			e.lines = append(e.lines, "")
			e.cursorY = 0
			e.cursorX = 0
		} else {
			return // Should not happen otherwise
		}
	}
	line := e.lines[e.cursorY]
	runes := []rune(line)

	if e.cursorX < 0 {
		e.cursorX = 0
	}
	if e.cursorX > len(runes) {
		e.cursorX = len(runes)
	}

	firstPart := string(runes[:e.cursorX])
	secondPart := string(runes[e.cursorX:])

	e.lines[e.cursorY] = firstPart
	e.lines = append(e.lines[:e.cursorY+1], append([]string{secondPart}, e.lines[e.cursorY+1:]...)...)

	e.cursorY++
	e.cursorX = 0
	e.isDirty = true
	e.statusMsg = ""
}

// deleteChar handles Backspace (deletes character before cursor or selection)
func (e *Editor) deleteChar() {
	if e.selection.active {
		e.deleteSelection()
		return
	}

	e.deactivateSelection()

	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		return
	}

	if e.cursorX == 0 && e.cursorY == 0 {
		return // Nothing to delete at the very beginning
	}

	if e.cursorX > 0 {
		line := e.lines[e.cursorY]
		runes := []rune(line)
		if e.cursorX <= len(runes) {
			newRunes := append(runes[:e.cursorX-1], runes[e.cursorX:]...)
			e.lines[e.cursorY] = string(newRunes)
			e.cursorX--
			e.isDirty = true
			e.statusMsg = ""
		}
	} else {
		prevLineLen := len([]rune(e.lines[e.cursorY-1]))
		e.lines[e.cursorY-1] += e.lines[e.cursorY]
		e.lines = append(e.lines[:e.cursorY], e.lines[e.cursorY+1:]...)
		e.cursorY--
		e.cursorX = prevLineLen
		e.isDirty = true
		e.statusMsg = ""
	}
}

// deleteForwardChar handles the Delete key (deletes character under/after cursor or selection)
func (e *Editor) deleteForwardChar() {
	if e.selection.active {
		e.deleteSelection()
		return
	}

	e.deactivateSelection()

	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		return
	}
	line := e.lines[e.cursorY]
	runes := []rune(line)
	lineLen := len(runes)

	if e.cursorX == lineLen && e.cursorY == len(e.lines)-1 {
		return // Nothing to delete at the very end of the file
	}

	if e.cursorX < lineLen {
		newRunes := append(runes[:e.cursorX], runes[e.cursorX+1:]...)
		e.lines[e.cursorY] = string(newRunes)
		e.isDirty = true
		e.statusMsg = ""
	} else {
		if e.cursorY < len(e.lines)-1 {
			e.lines[e.cursorY] += e.lines[e.cursorY+1]
			e.lines = append(e.lines[:e.cursorY+1], e.lines[e.cursorY+2:]...)
			e.isDirty = true
			e.statusMsg = ""
		}
	}
}

// deleteSelection removes the currently selected text
func (e *Editor) deleteSelection() {
	if !e.selection.active {
		return
	}

	sel := e.selection
	sel.normalize() // Ensure start is before end

	startLineIdx, startCol := sel.start.line, sel.start.col
	endLineIdx, endCol := sel.end.line, sel.end.col

	// Ensure indices are within bounds
	if startLineIdx < 0 || startLineIdx >= len(e.lines) || endLineIdx < 0 || endLineIdx >= len(e.lines) {
		e.deactivateSelection()
		return // Invalid selection range
	}

	startLineRunes := []rune(e.lines[startLineIdx])
	endLineRunes := []rune(e.lines[endLineIdx])

	// Clamp column indices
	if startCol < 0 {
		startCol = 0
	}
	if startCol > len(startLineRunes) {
		startCol = len(startLineRunes)
	}
	if endCol < 0 {
		endCol = 0
	}
	if endCol > len(endLineRunes) {
		endCol = len(endLineRunes)
	}

	// Case 1: Selection within a single line
	if startLineIdx == endLineIdx {
		lineRunes := []rune(e.lines[startLineIdx])
		newLineRunes := append(lineRunes[:startCol], lineRunes[endCol:]...)
		e.lines[startLineIdx] = string(newLineRunes)
	} else {
		// Case 2: Selection spans multiple lines
		// Keep the part before selection on the start line
		startLinePart := string(startLineRunes[:startCol])
		// Keep the part after selection on the end line
		endLinePart := string(endLineRunes[endCol:])

		// Combine the parts onto the start line
		e.lines[startLineIdx] = startLinePart + endLinePart

		// Remove the lines between start and end (exclusive of start, inclusive of end)
		if endLineIdx > startLineIdx {
			// Calculate the number of lines to remove
			linesToRemove := endLineIdx - startLineIdx
			if startLineIdx+1 < len(e.lines) { // Ensure there are lines to remove
				copyEnd := startLineIdx + 1 + linesToRemove
				if copyEnd > len(e.lines) {
					copyEnd = len(e.lines)
				}
				// Check if there are lines *after* the removed section
				if copyEnd < len(e.lines) {
					e.lines = append(e.lines[:startLineIdx+1], e.lines[copyEnd:]...)
				} else {
					// If removing until the end, just truncate
					e.lines = e.lines[:startLineIdx+1]
				}
			}
		}
	}

	// Move cursor to the start of the deleted selection
	e.cursorY = startLineIdx
	e.cursorX = startCol

	// Ensure cursor is valid after deletion
	if e.cursorY >= len(e.lines) {
		e.cursorY = len(e.lines) - 1
	}
	if e.cursorY < 0 { // Can happen if all lines were deleted
		e.lines = []string{""} // Add back an empty line
		e.cursorY = 0
		e.cursorX = 0
	} else {
		currentLineRunes := []rune(e.lines[e.cursorY])
		if e.cursorX > len(currentLineRunes) {
			e.cursorX = len(currentLineRunes)
		}
	}

	e.isDirty = true
	e.statusMsg = "Selection deleted"
	e.deactivateSelection() // Turn off selection after deleting
}

// deactivateSelection turns off the selection mode
func (e *Editor) deactivateSelection() {
	if e.selection.active {
		e.selection.active = false
		e.selectionAnchor = Position{} // Reset anchor
		e.statusMsg = ""               // Clear any selection-related status
	}
}

// selectLine selects the entire current line
func (e *Editor) selectLine() {
	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		return // Invalid cursor position
	}
	e.selection.active = true
	e.selection.start = Position{line: e.cursorY, col: 0}
	e.selection.end = Position{line: e.cursorY, col: len([]rune(e.lines[e.cursorY]))}
	e.selectionAnchor = e.selection.start
	e.statusMsg = "Line selected"
}

// selectAll selects the entire buffer content
func (e *Editor) selectAll() {
	e.selection.active = true
	if len(e.lines) == 0 {
		e.selection.start = Position{line: 0, col: 0}
		e.selection.end = Position{line: 0, col: 0}
	} else {
		e.selection.start = Position{line: 0, col: 0}
		lastLine := len(e.lines) - 1
		lastCol := len([]rune(e.lines[lastLine]))
		e.selection.end = Position{line: lastLine, col: lastCol}
	}
	e.selectionAnchor = e.selection.start // Anchor at the beginning
	e.statusMsg = "All text selected"
}

// processKeyPress handles a single key press event
func (e *Editor) processKeyPress(r rune) bool {
	currentPos := Position{line: e.cursorY, col: e.cursorX}
	selectionWasActive := e.selection.active // Remember if selection was active before processing

	switch r {
	case keyCtrlQ:
		return true // Signal to quit

	case keyCtrlS:
		e.deactivateSelection()
		err := e.SaveFile()
		if err == nil {
			e.statusMsg = fmt.Sprintf("Saved %d lines to %s", len(e.lines), e.fileName)
		} else {
			e.statusMsg = fmt.Sprintf("Save error: %v", err)
		}

	case keyArrowUp, keyArrowDown, keyArrowLeft, keyArrowRight, keyHome, keyEnd, keyPageUp, keyPageDown:
		e.deactivateSelection()
		e.moveCursorOnly(r)
		e.statusMsg = ""

	case keyShiftArrowLeft, keyShiftArrowRight:
		if !e.selection.active {
			e.selection.active = true
			e.selectionAnchor = currentPos
			e.selection.start = currentPos
			e.selection.end = currentPos
		}
		e.moveCursorOnly(r)
		e.selection.end = Position{line: e.cursorY, col: e.cursorX}
		e.selection.start = e.selectionAnchor
		e.statusMsg = "Selecting..."

	case keyEnter:
		e.insertNewline() // Handles selection deletion internally

	case keyBackspace:
		e.deleteChar() // Handles selection deletion internally

	case keyDelete:
		e.deleteForwardChar() // Handles selection deletion internally

	case keyCtrlL:
		e.selectLine()

	case keyCtrlA:
		e.selectAll()

	case keyEscape:
		if e.selection.active {
			e.deactivateSelection()
		} else {
			e.statusMsg = "Escape pressed"
		}

	default:
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			e.insertChar(r) // Handles selection deletion internally
		} else if selectionWasActive {
			// If some other non-printable key was pressed while selecting, deactivate
			e.deactivateSelection()
		}
	}

	// If a movement key was pressed *without* shift while selection was active,
	// deactivate selection *after* moving.
	if !e.selection.active && selectionWasActive {
		switch r {
		case keyArrowUp, keyArrowDown, keyArrowLeft, keyArrowRight, keyHome, keyEnd, keyPageUp, keyPageDown:
			// Already deactivated by moveCursorOnly or other handlers
			break // Selection was already handled
		}
	}

	return false // Don't quit unless Ctrl+Q
}

// OpenFile loads a file into the editor buffer
func (e *Editor) OpenFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			e.fileName = filename
			e.lines = []string{""}
			e.isDirty = false
			e.statusMsg = fmt.Sprintf("New file: %s", filename)
			return nil
		}
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	e.fileName = filename
	e.lines = []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		e.lines = append(e.lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	if len(e.lines) == 0 {
		e.lines = append(e.lines, "")
	}
	e.isDirty = false
	e.cursorX, e.cursorY = 0, 0
	e.offsetX, e.offsetY = 0, 0
	e.statusMsg = fmt.Sprintf("Opened %s", filename)
	return nil
}

// SaveFile writes the buffer content to the current file
func (e *Editor) SaveFile() error {
	if e.fileName == "[No Name]" {
		e.statusMsg = "Cannot save file without a name. (Save As not implemented)"
		return fmt.Errorf("no filename specified")
	}

	content := strings.Join(e.lines, "\n")
	err := os.WriteFile(e.fileName, []byte(content), 0644)
	if err != nil {
		e.statusMsg = fmt.Sprintf("Error writing file: %s", err)
		return fmt.Errorf("could not write file: %w", err)
	}

	e.isDirty = false
	e.statusMsg = fmt.Sprintf("Saved %s", e.fileName)
	return nil
}

// LaunchEditor is the entry point to start the editor tool
func LaunchEditor(args []string) {
	if len(args) == 0 {
		fmt.Println("SecShell Editor")
		fmt.Println("Usage: edit <filename>")
		fmt.Println("\nOpens the specified file in the editor. If the file does not exist, it will be created upon saving.")
		fmt.Println("\nKeybindings:")
		fmt.Println("  Ctrl+S: Save file")
		fmt.Println("  Ctrl+Q: Quit editor (prompts to save if modified)")
		fmt.Println("  Ctrl+L: Select current line")
		fmt.Println("  Ctrl+A: Select all text")
		fmt.Println("  Shift+Left/Right: Select text")
		fmt.Println("  Esc: Cancel selection")
		return
	}

	editor := NewEditor()
	filename := args[0]
	if err := editor.OpenFile(filename); err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", filename, err)
		if !os.IsNotExist(err) {
			return
		}
	}

	err := editor.Run()
	if err != nil {
		editor.disableRawMode()
		fmt.Fprintf(os.Stderr, "Editor error: %v\n", err)
		os.Exit(1)
	}
}
