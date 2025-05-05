package gui

import (
	"fmt"
	"secshell/colors"
	"strings"
)

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

func (l *Label) Render(buffer *strings.Builder, winX, winY int, contentWidth int) {
	// Calculate absolute position for the start of the label
	absX := winX + l.X
	absY := winY + l.Y

	// Calculate the maximum width available for this label within the content area
	maxWidth := contentWidth - l.X
	if maxWidth < 1 {
		maxWidth = 1 // Need at least 1 character width to render anything
	}

	text := l.Text
	lineIndex := 0

	buffer.WriteString(l.Color) // Set color before rendering lines

	for len(text) > 0 {
		currentLineY := absY + lineIndex
		buffer.WriteString(MoveCursorCmd(currentLineY, absX))

		var lineText string
		if len(text) <= maxWidth {
			// Remaining text fits on one line
			lineText = text
			text = "" // No more text left
		} else {
			// Text needs wrapping
			wrapIndex := -1
			// Try to find a space to wrap at within maxWidth
			possibleWrapPoint := text[:maxWidth]
			wrapIndex = strings.LastIndex(possibleWrapPoint, " ")

			if wrapIndex != -1 {
				// Found a space, wrap there
				lineText = text[:wrapIndex]
				text = strings.TrimPrefix(text[wrapIndex:], " ") // Remove the space and continue
			} else {
				// No space found, force break at maxWidth
				lineText = text[:maxWidth]
				text = text[maxWidth:]
			}
		}

		buffer.WriteString(lineText)
		// Clear the rest of the line within the max width if needed (optional, depends on desired look)
		// buffer.WriteString(strings.Repeat(" ", maxWidth-len(lineText)))

		lineIndex++ // Move to the next line for subsequent text
	}

	buffer.WriteString(colors.Reset) // Reset color after rendering all lines
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

// CheckBox represents a toggleable checkbox element.
type CheckBox struct {
	Label       string
	Color       string
	ActiveColor string // Color when selected/active
	Checked     bool   // State of the checkbox
	X, Y        int    // Position relative to window content area
	IsActive    bool   // State for rendering/input handling
}

// NewCheckBox creates a new CheckBox instance.
func NewCheckBox(label string, x, y int, initialChecked bool, color, activeColor string) *CheckBox {
	return &CheckBox{
		Label:       label,
		X:           x,
		Y:           y,
		Checked:     initialChecked,
		Color:       color,
		ActiveColor: activeColor,
		IsActive:    false,
	}
}

// Render draws the checkbox element.
func (cb *CheckBox) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + cb.X
	absY := winY + cb.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	renderColor := cb.Color
	if cb.IsActive {
		renderColor = cb.ActiveColor
		buffer.WriteString(ReverseVideo()) // Indicate active state visually
	}
	buffer.WriteString(renderColor)

	checkMark := " "
	if cb.Checked {
		checkMark = "X" // Or use a unicode checkmark if preferred: "✔"
	}
	buffer.WriteString(fmt.Sprintf("[%s] %s", checkMark, cb.Label))

	buffer.WriteString(colors.Reset) // Reset color and video attributes
}

// --- Spacer ---

// Spacer represents a vertical empty space.
type Spacer struct {
	Height int // Number of empty rows
	X, Y   int // Position (X is usually ignored, Y marks the top)
}

// NewSpacer creates a new Spacer instance.
// X and Y define the top-left starting point, Height defines the vertical space.
func NewSpacer(x, y, height int) *Spacer {
	return &Spacer{
		X:      x, // X is often irrelevant for a vertical spacer but included for consistency
		Y:      y,
		Height: height,
	}
}

// Render for Spacer does nothing visually, as spacing is handled by the Y coordinates
// of subsequent elements. It fulfills the UIElement interface.
func (s *Spacer) Render(buffer *strings.Builder, winX, winY int, contentWidth int) {
	// No visual output needed. The layout logic relies on the Y coordinates
	// of elements placed *after* the spacer.
	// We could potentially add blank lines to the buffer if needed for some reason,
	// but it's generally unnecessary with absolute positioning.
	// Example: Move cursor down conceptually
	// absY := winY + s.Y
	// buffer.WriteString(MoveCursorCmd(absY+s.Height, winX+s.X))
}

// --- Radio Buttons ---

// Forward declaration for RadioButton's reference
type RadioGroup struct {
	Buttons       []*RadioButton
	SelectedIndex int
	SelectedValue string // Or int, depending on your needs
}

// RadioButton represents a single option in a radio button group.
type RadioButton struct {
	Label       string
	Value       string // The value associated with this radio button
	Color       string
	ActiveColor string // Color when selected/active
	X, Y        int    // Position relative to window content area
	IsActive    bool   // State for rendering/input handling
	IsSelected  bool   // State of the radio button within its group
	Group       *RadioGroup
}

// NewRadioGroup creates a new RadioGroup.
func NewRadioGroup() *RadioGroup {
	return &RadioGroup{
		Buttons:       make([]*RadioButton, 0),
		SelectedIndex: -1, // Nothing selected initially
		SelectedValue: "",
	}
}

// NewRadioButton creates a new RadioButton instance and adds it to a group.
func NewRadioButton(label, value string, x, y int, color, activeColor string, group *RadioGroup) *RadioButton {
	rb := &RadioButton{
		Label:       label,
		Value:       value,
		X:           x,
		Y:           y,
		Color:       color,
		ActiveColor: activeColor,
		IsActive:    false,
		IsSelected:  false,
		Group:       group,
	}
	group.Buttons = append(group.Buttons, rb)
	// Optionally select the first button added to a group by default
	// if group.SelectedIndex == -1 {
	//  group.Select(0)
	// }
	return rb
}

// Select sets the radio button at the given index as selected within its group.
func (rg *RadioGroup) Select(selectedIndex int) {
	if selectedIndex < 0 || selectedIndex >= len(rg.Buttons) {
		return // Invalid index
	}

	rg.SelectedIndex = selectedIndex
	rg.SelectedValue = rg.Buttons[selectedIndex].Value

	for i, btn := range rg.Buttons {
		btn.IsSelected = (i == selectedIndex)
	}
}

// Render draws the radio button element.
func (rb *RadioButton) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + rb.X
	absY := winY + rb.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	renderColor := rb.Color
	if rb.IsActive {
		renderColor = rb.ActiveColor
		buffer.WriteString(ReverseVideo()) // Indicate active state visually
	}
	buffer.WriteString(renderColor)

	selectionMark := " "
	if rb.IsSelected {
		selectionMark = "*" // Mark for selected radio button
	}
	// Use parentheses for radio buttons
	buffer.WriteString(fmt.Sprintf("(%s) %s", selectionMark, rb.Label))

	buffer.WriteString(colors.Reset) // Reset color and video attributes
}

// --- Progress Bar ---

// ProgressBar represents a visual progress indicator.
type ProgressBar struct {
	Value          float64 // Current value
	MaxValue       float64 // Maximum value (represents 100%)
	Color          string  // Color of the filled portion
	UnfilledColor  string  // Color of the unfilled portion
	ShowPercentage bool    // Whether to display the percentage text
	X, Y           int     // Position relative to window content area
	Width          int     // Total width of the bar in characters
}

// NewProgressBar creates a new ProgressBar instance.
func NewProgressBar(x, y, width int, initialValue, maxValue float64, color, unfilledColor string, showPercentage bool) *ProgressBar {
	if maxValue <= 0 {
		maxValue = 100 // Default max value if invalid
	}
	if initialValue < 0 {
		initialValue = 0
	}
	if initialValue > maxValue {
		initialValue = maxValue
	}
	// Use default unfilled color if none provided
	if unfilledColor == "" {
		unfilledColor = colors.Reset // Default to reset/terminal default
	}
	return &ProgressBar{
		Value:          initialValue,
		MaxValue:       maxValue,
		Color:          color,
		UnfilledColor:  unfilledColor,
		ShowPercentage: showPercentage,
		X:              x,
		Y:              y,
		Width:          width,
	}
}

// SetValue updates the progress bar's current value, clamping it between 0 and MaxValue.
func (pb *ProgressBar) SetValue(value float64) {
	if value < 0 {
		pb.Value = 0
	} else if value > pb.MaxValue {
		pb.Value = pb.MaxValue
	} else {
		pb.Value = value
	}
}

// Render draws the progress bar element.
func (pb *ProgressBar) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + pb.X
	absY := winY + pb.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	percentage := 0.0
	if pb.MaxValue > 0 {
		percentage = pb.Value / pb.MaxValue
	}

	// Calculate the width available for the bar itself
	barWidth := pb.Width
	percentageText := ""
	if pb.ShowPercentage {
		percentageText = fmt.Sprintf(" %.0f%%", percentage*100)
		// Reduce bar width to make space for the text
		barWidth -= len(percentageText)
		if barWidth < 0 {
			barWidth = 0 // Ensure bar width isn't negative
		}
	}

	filledWidth := int(float64(barWidth) * percentage)
	emptyWidth := barWidth - filledWidth

	// Draw the filled part
	buffer.WriteString(pb.Color)
	buffer.WriteString(strings.Repeat("█", filledWidth)) // Use a block character for filled part

	// Draw the empty part (set unfilled color first)
	buffer.WriteString(colors.Reset)                    // Reset to default before unfilled color
	buffer.WriteString(pb.UnfilledColor)                // Set color for the empty part
	buffer.WriteString(strings.Repeat("░", emptyWidth)) // Use a lighter shade or space for empty part

	// Draw the percentage text if enabled
	if pb.ShowPercentage {
		// Ensure percentage text uses a predictable color (e.g., reset)
		// or allow it to inherit the UnfilledColor if desired.
		// Here, we reset before the text for clarity.
		buffer.WriteString(colors.Reset)
		buffer.WriteString(percentageText)
	}

	buffer.WriteString(colors.Reset) // Ensure color is reset at the end
}
