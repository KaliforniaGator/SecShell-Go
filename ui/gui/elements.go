package gui

import (
	"fmt"
	"secshell/colors"
	"strings"
)

// CursorManager is an interface for elements that need to manage cursor visibility
type CursorManager interface {
	NeedsCursor() bool                   // Returns true if the element currently wants the cursor visible
	GetCursorPosition() (int, int, bool) // Returns absolute cursor x, y position and whether it's valid
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

// NeedsCursor implements CursorManager interface (never needs cursor)
func (b *Button) NeedsCursor() bool {
	return false
}

func (b *Button) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
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
	cursorAbsX  int  // Absolute X position of cursor (set during Render)
	cursorAbsY  int  // Absolute Y position of cursor (set during Render)
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

// NeedsCursor implements CursorManager interface
func (tb *TextBox) NeedsCursor() bool {
	return tb.IsActive // Only show cursor when the textbox is active
}

// GetCursorPosition implements CursorManager interface
func (tb *TextBox) GetCursorPosition() (int, int, bool) {
	if !tb.NeedsCursor() {
		return 0, 0, false
	}
	return tb.cursorAbsX, tb.cursorAbsY, true
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

	// --- Cursor Position Calculation ---
	// Calculate cursor position relative to the *start* of the textbox's absolute position
	cursorRenderPos := tb.cursorPos - viewStart

	// Clamp the render position to be within the visible bounds of the textbox [0, tb.Width]
	if cursorRenderPos < 0 {
		cursorRenderPos = 0
	} else if cursorRenderPos > tb.Width {
		// This case might happen if text length equals width and cursor is at the end
		cursorRenderPos = tb.Width
	}

	// Store the final absolute screen coordinates for the cursor
	tb.cursorAbsX = absX + cursorRenderPos
	tb.cursorAbsY = absY

	// Don't add cursor show/hide commands here - the Window will handle cursor visibility
	// based on the CursorManager interface implementation
	// --- End Cursor Position Calculation ---

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

// NeedsCursor implements CursorManager interface (never needs cursor)
func (cb *CheckBox) NeedsCursor() bool {
	return false
}

func (cb *CheckBox) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
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

// NeedsCursor implements CursorManager interface (never needs cursor)
func (rb *RadioButton) NeedsCursor() bool {
	return false
}

func (rb *RadioButton) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
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

// --- ScrollBar ---

// ScrollBar represents a vertical scrollbar element.
type ScrollBar struct {
	X, Y        int                // Position relative to window content area (top-left of the scrollbar)
	Height      int                // Height of the scrollbar track in characters
	Value       int                // Current value (e.g., top visible line index), 0-based
	MaxValue    int                // Maximum value (e.g., total lines - visible lines), 0-based
	Color       string             // Color of the scrollbar track and thumb
	ActiveColor string             // Color when focused/active
	IsActive    bool               // State for rendering/input handling
	Visible     bool               // Controls whether the scrollbar is rendered
	ContainerID string             // Identifier for the container this scrollbar controls (for future use)
	thumbChar   string             // Character for the thumb
	trackChar   string             // Character for the track
	OnScroll    func(newValue int) // Callback function when value changes via SetValue
}

// NewScrollBar creates a new ScrollBar instance.
// Value is the initial top visible line index.
// MaxValue is the maximum possible top visible line index (e.g., total lines - viewport height).
func NewScrollBar(x, y, height, value, maxValue int, color, activeColor, containerID string) *ScrollBar {
	if height < 2 {
		height = 2 // Minimum height for track + thumb
	}
	if value < 0 {
		value = 0
	}
	if maxValue < 0 {
		maxValue = 0
	}
	if value > maxValue {
		value = maxValue
	}
	return &ScrollBar{
		X:           x,
		Y:           y,
		Height:      height,
		Value:       value,
		MaxValue:    maxValue,
		Color:       color,
		ActiveColor: activeColor,
		IsActive:    false,
		Visible:     false, // Start hidden by default, container will make it visible
		ContainerID: containerID,
		thumbChar:   "█", // Block character for thumb
		trackChar:   "│", // Line character for track
		OnScroll:    nil, // Initialize callback to nil
	}
}

// SetValue updates the scrollbar's current value, clamping it, and calls the OnScroll callback.
func (sb *ScrollBar) SetValue(value int) {
	oldValue := sb.Value
	newValue := value
	if newValue < 0 {
		newValue = 0
	} else if newValue > sb.MaxValue {
		newValue = sb.MaxValue
	}

	if newValue != oldValue {
		sb.Value = newValue
		// Call the callback if it's set
		if sb.OnScroll != nil {
			sb.OnScroll(sb.Value)
		}
	}
}

// Render draws the scrollbar element.
func (sb *ScrollBar) Render(buffer *strings.Builder, winX, winY int, _ int) {
	// Only render if visible
	if !sb.Visible {
		// If not visible, we might need to clear the area it would occupy
		// This prevents artifacts if it was previously visible.
		absX := winX + sb.X
		absY := winY + sb.Y
		for i := 0; i < sb.Height; i++ {
			buffer.WriteString(MoveCursorCmd(absY+i, absX))
			buffer.WriteString(" ") // Overwrite with space
		}
		return
	}

	absX := winX + sb.X
	absY := winY + sb.Y

	renderColor := sb.Color
	if sb.IsActive {
		renderColor = sb.ActiveColor
		// Optionally add reverse video or other indicators for active state
		// buffer.WriteString(ReverseVideo())
	}
	buffer.WriteString(renderColor)

	// Calculate thumb position
	thumbPos := 0 // Position relative to the top of the scrollbar (0 to Height-1)
	if sb.MaxValue > 0 {
		// Calculate position based on value percentage
		percentage := float64(sb.Value) / float64(sb.MaxValue)
		thumbPos = int(percentage * float64(sb.Height-1)) // Scale to fit height (minus 1 for 0-based index)
	}
	// Clamp thumbPos just in case
	if thumbPos < 0 {
		thumbPos = 0
	} else if thumbPos >= sb.Height {
		thumbPos = sb.Height - 1
	}

	// Draw the scrollbar track and thumb
	for i := 0; i < sb.Height; i++ {
		buffer.WriteString(MoveCursorCmd(absY+i, absX))
		if i == thumbPos {
			buffer.WriteString(sb.thumbChar) // Draw thumb
		} else {
			buffer.WriteString(sb.trackChar) // Draw track
		}
	}

	buffer.WriteString(colors.Reset) // Reset color
}

// NeedsCursor implements CursorManager interface (never needs cursor)
func (sb *ScrollBar) NeedsCursor() bool {
	return false
}

func (sb *ScrollBar) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
}

// --- Container ---

// Container represents a scrollable area for content.
type Container struct {
	X, Y               int
	Width, Height      int
	Content            []string // Initially support only string content
	scrollBar          *ScrollBar
	needsScroll        bool
	totalContentHeight int
	IsActive           bool                    // Tracks if the container itself has focus
	SelectedIndex      int                     // Index of the selected line in Content
	Color              string                  // Default background/text color (use window's if empty)
	ActiveColor        string                  // Border/indicator color when active (unused for now, but good practice)
	SelectionColor     string                  // Background/text color for the selected line
	OnItemSelected     func(selectedIndex int) // Callback when an item is selected via Enter
	cursorAbsX         int                     // Used for cursor position tracking
	cursorAbsY         int                     // Used for cursor position tracking
	// TODO: Add BgColor, ContentColor properties if needed explicitly for container
}

// NewContainer creates a new Container instance.
func NewContainer(x, y, width, height int, content []string) *Container {
	// Ensure minimum dimensions
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	// Determine scrollbar position relative to container
	sbX := width - 2 // Scrollbar always occupies the last column conceptually
	sbY := 0
	sbHeight := height

	// Always create the scrollbar instance
	containerID := fmt.Sprintf("container_%d_%d_scrollbar", x, y)
	// Initial MaxValue is 0, updateScrollState will fix it
	scrollBar := NewScrollBar(sbX, sbY, sbHeight, 0, 0, colors.Gray, colors.BoldWhite, containerID)
	scrollBar.Visible = false // Start hidden

	c := &Container{
		X:              x,
		Y:              y,
		Width:          width,
		Height:         height,
		Content:        content,
		scrollBar:      scrollBar, // Assign the created scrollbar
		needsScroll:    false,     // Will be set by updateScrollState
		IsActive:       false,
		SelectedIndex:  0,
		Color:          "",
		ActiveColor:    colors.BoldWhite,
		SelectionColor: colors.BgBlue + colors.BoldWhite,
		OnItemSelected: nil, // Initialize new callback to nil
	}

	c.updateScrollState() // Calculate initial scroll state and visibility

	// Ensure initial selection is valid
	if c.SelectedIndex >= len(c.Content) && len(c.Content) > 0 {
		c.SelectedIndex = len(c.Content) - 1
	} else if len(c.Content) == 0 {
		c.SelectedIndex = -1 // No selection possible
	}
	// Ensure initial selection is visible after state update
	c.ensureSelectionVisible()

	return c
}

// updateScrollState calculates content height and determines if scrolling is needed.
// It updates the internal scrollbar's visibility and properties.
func (c *Container) updateScrollState() {
	c.totalContentHeight = len(c.Content)
	c.needsScroll = c.totalContentHeight > c.Height

	// Adjust SelectedIndex if it's now out of bounds
	if c.SelectedIndex >= c.totalContentHeight {
		if c.totalContentHeight > 0 {
			c.SelectedIndex = c.totalContentHeight - 1
		} else {
			c.SelectedIndex = -1 // No items left
		}
	}

	// Update scrollbar visibility and MaxValue
	c.scrollBar.Visible = c.needsScroll // Set visibility based on need
	if c.needsScroll {
		sbMaxValue := c.totalContentHeight - c.Height
		if sbMaxValue < 0 {
			sbMaxValue = 0
		}
		c.scrollBar.MaxValue = sbMaxValue
		// Clamp current scroll value if necessary
		c.scrollBar.SetValue(c.scrollBar.Value)
	} else {
		c.scrollBar.MaxValue = 0
		c.scrollBar.SetValue(0) // Reset scroll value if not needed
	}

	// Ensure selection is visible after potential scrollbar update
	c.ensureSelectionVisible()
}

// SetContent updates the container's content and recalculates scrolling state.
func (c *Container) SetContent(content []string) {
	c.Content = content
	c.updateScrollState() // This will also adjust SelectedIndex if needed
}

// GetScrollOffset returns the current vertical scroll offset (top visible line index).
// Returns 0 if scrolling is not needed or the scrollbar doesn't exist.
func (c *Container) GetScrollOffset() int {
	if c.scrollBar != nil {
		return c.scrollBar.Value
	}
	return 0 // No scrollbar means no offset
}

// ensureSelectionVisible adjusts the scroll offset if the selected item is out of view.
func (c *Container) ensureSelectionVisible() {
	// Only adjust if scrollbar is currently needed/visible and selection is valid
	if !c.scrollBar.Visible || c.SelectedIndex < 0 {
		return
	}

	scrollOffset := c.scrollBar.Value
	bottomVisibleIndex := scrollOffset + c.Height - 1

	if c.SelectedIndex < scrollOffset {
		// Selection is above the view, scroll up
		c.scrollBar.SetValue(c.SelectedIndex)
	} else if c.SelectedIndex > bottomVisibleIndex {
		// Selection is below the view, scroll down
		c.scrollBar.SetValue(c.SelectedIndex - c.Height + 1)
	}
}

// SelectNext selects the next item in the container.
func (c *Container) SelectNext() {
	if c.SelectedIndex < c.totalContentHeight-1 {
		c.SelectedIndex++
		c.ensureSelectionVisible()
		// No callback call here anymore
	}
}

// SelectPrevious selects the previous item in the container.
func (c *Container) SelectPrevious() {
	if c.SelectedIndex > 0 {
		c.SelectedIndex--
		c.ensureSelectionVisible()
		// No callback call here anymore
	}
}

// GetSelectedIndex returns the index of the currently selected item.
// Returns -1 if no item is selected (e.g., empty container).
func (c *Container) GetSelectedIndex() int {
	return c.SelectedIndex
}

// NeedsCursor implements CursorManager interface
func (c *Container) NeedsCursor() bool {
	return false // Containers never need a cursor visible
}

// GetCursorPosition implements CursorManager interface
func (c *Container) GetCursorPosition() (int, int, bool) {
	return c.cursorAbsX, c.cursorAbsY, false // Position known but not needed
}

// Render draws the container and its visible content.
func (c *Container) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + c.X // Absolute X of the container's top-left corner
	absY := winY + c.Y // Absolute Y of the container's top-left corner

	// Determine the width available *specifically for text content*
	textContentWidth := c.Width
	// Use scrollBar.Visible to decide if width needs reduction
	if c.scrollBar.Visible {
		textContentWidth--
	}
	// Ensure text content width is never negative
	if textContentWidth < 0 {
		textContentWidth = 0
	}

	scrollOffset := 0
	// Only get offset if scrollbar is visible/active
	if c.scrollBar.Visible {
		scrollOffset = c.scrollBar.Value
	}

	// Render visible lines of string content
	for i := 0; i < c.Height; i++ {
		contentIndex := i + scrollOffset
		lineY := absY + i // Absolute Y for the current line

		// Move cursor to the start of the line within the container
		buffer.WriteString(MoveCursorCmd(lineY, absX))

		// Determine line color
		lineColor := c.Color                                                                // Use container's default or inherit window's
		if c.IsActive && contentIndex == c.SelectedIndex && contentIndex < len(c.Content) { // Check contentIndex bounds
			lineColor = c.SelectionColor // Use selection color if active and selected
		}
		buffer.WriteString(lineColor) // Apply line color

		if contentIndex >= 0 && contentIndex < len(c.Content) {
			line := c.Content[contentIndex]
			currentWidth := 0
			truncatedLine := ""
			// Build the line rune by rune, respecting textContentWidth
			for _, r := range line {
				// Assuming standard width characters for now
				runeWidth := 1
				if currentWidth+runeWidth <= textContentWidth {
					truncatedLine += string(r)
					currentWidth += runeWidth
				} else {
					break // Stop adding runes if width exceeded
				}
			}
			buffer.WriteString(truncatedLine)

			// Clear the rest of the line *within the text content area only* with the current line color
			padding := textContentWidth - currentWidth
			if padding > 0 {
				buffer.WriteString(strings.Repeat(" ", padding))
			}
		} else {
			// Render empty line within the text content area with the current line color
			buffer.WriteString(strings.Repeat(" ", textContentWidth))
		}
		buffer.WriteString(colors.Reset) // Reset color after each line to prevent spillover
	} // End of line rendering loop

	// Render the scrollbar (it handles its own visibility check)
	// Pass the container's absolute top-left (absX, absY) as the origin.
	c.scrollBar.Render(buffer, absX, absY, c.Width) // Pass container's abs origin

	c.cursorAbsX = absX // Store position for cursor management (even though not shown)
	c.cursorAbsY = absY
}

// GetScrollbar returns the internal scrollbar if it exists.
// This allows the window to make the scrollbar focusable.
// NOTE: We are changing focus logic, so this might not be needed by Window anymore.
func (c *Container) GetScrollbar() *ScrollBar {
	return c.scrollBar
}
