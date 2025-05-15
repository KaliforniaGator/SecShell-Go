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

// ZIndexer defines an interface for elements that support z-index ordering
type ZIndexer interface {
	GetZIndex() int // Returns the z-index value of the element
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

// --- Gradient Progress Bar ---

// GradientProgressBar represents a visual progress indicator with a gradient fill.
type GradientProgressBar struct {
	Value          float64 // Current value
	MaxValue       float64 // Maximum value (represents 100%)
	StartColorHex  string  // Hex string for the start of the gradient (e.g., "#FF0000")
	EndColorHex    string  // Hex string for the end of the gradient (e.g., "#00FF00")
	UnfilledColor  string  // Color of the unfilled portion
	ShowPercentage bool    // Whether to display the percentage text
	X, Y           int     // Position relative to window content area
	Width          int     // Total width of the bar in characters
}

// NewGradientProgressBar creates a new GradientProgressBar instance.
func NewGradientProgressBar(x, y, width int, initialValue, maxValue float64, startColorHex, endColorHex, unfilledColor string, showPercentage bool) *GradientProgressBar {
	if maxValue <= 0 {
		maxValue = 100 // Default max value if invalid
	}
	if initialValue < 0 {
		initialValue = 0
	}
	if initialValue > maxValue {
		initialValue = maxValue
	}
	if unfilledColor == "" {
		unfilledColor = colors.Gray2 // Default unfilled color
	}
	return &GradientProgressBar{
		Value:          initialValue,
		MaxValue:       maxValue,
		StartColorHex:  startColorHex,
		EndColorHex:    endColorHex,
		UnfilledColor:  unfilledColor,
		ShowPercentage: showPercentage,
		X:              x,
		Y:              y,
		Width:          width,
	}
}

// SetValue updates the gradient progress bar's current value, clamping it between 0 and MaxValue.
func (gpb *GradientProgressBar) SetValue(value float64) {
	if value < 0 {
		gpb.Value = 0
	} else if value > gpb.MaxValue {
		gpb.Value = gpb.MaxValue
	} else {
		gpb.Value = value
	}
}

// Render draws the gradient progress bar element.
func (gpb *GradientProgressBar) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + gpb.X
	absY := winY + gpb.Y
	buffer.WriteString(MoveCursorCmd(absY, absX))

	percentage := 0.0
	if gpb.MaxValue > 0 {
		percentage = gpb.Value / gpb.MaxValue
	}

	barWidth := gpb.Width
	percentageText := ""
	if gpb.ShowPercentage {
		percentageText = fmt.Sprintf(" %.0f%%", percentage*100)
		barWidth -= len(percentageText)
		if barWidth < 0 {
			barWidth = 0
		}
	}

	filledWidth := int(float64(barWidth) * percentage)
	emptyWidth := barWidth - filledWidth

	// Draw the filled part with gradient
	if filledWidth > 0 {
		gradient := colors.GenerateGradient(gpb.StartColorHex, gpb.EndColorHex, filledWidth)
		for i := 0; i < filledWidth; i++ {
			buffer.WriteString(gradient[i])
			buffer.WriteString("█") // Use a block character for filled part
		}
		buffer.WriteString(colors.Reset) // Reset after gradient
	}

	// Draw the empty part
	buffer.WriteString(gpb.UnfilledColor)
	buffer.WriteString(strings.Repeat("░", emptyWidth)) // Use a lighter shade or space for empty part

	// Draw the percentage text if enabled
	if gpb.ShowPercentage {
		buffer.WriteString(colors.Reset) // Ensure text color is reset or set explicitly
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
	sbX := width - 1 // Scrollbar always occupies the last column conceptually
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

// --- TextArea ---

// TextArea represents a multi-line text input area with scrolling.
type TextArea struct {
	X, Y           int      // Position relative to window content area
	Width, Height  int      // Dimensions of the text area
	Color          string   // Default text color
	ActiveColor    string   // Color when active (e.g., border or cursor)
	IsActive       bool     // State for rendering/input handling
	Lines          []string // Content stored as lines
	cursorLine     int      // Cursor's line index (0-based)
	cursorCol      int      // Cursor's column index (rune-based, 0-based) within the line
	viewTopLine    int      // Index of the topmost visible line
	scrollBar      *ScrollBar
	needsScroll    bool
	maxChars       int    // Optional maximum character limit (0 for unlimited)
	wordCount      int    // Current word count
	charCount      int    // Current character count
	cursorAbsX     int    // Absolute X position of cursor (set during Render)
	cursorAbsY     int    // Absolute Y position of cursor (set during Render)
	showWordCount  bool   // Flag to control word count visibility
	showCharCount  bool   // Flag to control char count visibility
	bottomLineText string // Text to display on the bottom line (word/char count)
}

// NewTextArea creates a new TextArea instance.
func NewTextArea(initialText string, x, y, width, height, maxChars int, color, activeColor string, showWordCount, showCharCount bool) *TextArea {
	if width < 3 { // Need space for text and potentially scrollbar + border
		width = 3
	}
	if height < 2 { // Need space for text and word count line
		height = 2
	}

	lines := strings.Split(strings.ReplaceAll(initialText, "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		lines = []string{""} // Ensure at least one empty line
	}

	// Scrollbar position relative to the TextArea's content area
	sbX := width - 1 // Scrollbar on the far right
	sbY := 0
	sbHeight := height - 1 // Leave space for word count line if shown
	if sbHeight < 1 {
		sbHeight = 1 // Minimum height for scrollbar
	}

	containerID := fmt.Sprintf("textarea_%d_%d_scrollbar", x, y)
	scrollBar := NewScrollBar(sbX, sbY, sbHeight, 0, 0, colors.Gray, colors.BoldWhite, containerID)
	scrollBar.Visible = false // Start hidden

	ta := &TextArea{
		X:             x,
		Y:             y,
		Width:         width,
		Height:        height,
		Color:         color,
		ActiveColor:   activeColor,
		IsActive:      false,
		Lines:         lines,
		cursorLine:    0, // Start at the beginning
		cursorCol:     0,
		viewTopLine:   0,
		scrollBar:     scrollBar,
		needsScroll:   false,
		maxChars:      maxChars,
		showWordCount: showWordCount,
		showCharCount: showCharCount,
	}

	// Set the scrollbar's OnScroll callback to update the viewTopLine
	ta.scrollBar.OnScroll = func(newValue int) {
		ta.viewTopLine = newValue
	}

	ta.calculateCounts()     // Calculate initial counts
	ta.updateScrollState()   // Calculate initial scroll state
	ta.ensureCursorVisible() // Ensure initial cursor position is visible

	return ta
}

// calculateCounts updates word and character counts.
func (ta *TextArea) calculateCounts() {
	ta.charCount = 0
	totalWords := 0
	fullText := strings.Join(ta.Lines, " ") // Join with space to count words across lines correctly
	words := strings.Fields(fullText)       // Split by whitespace
	totalWords = len(words)

	// Calculate character count accurately (including newlines)
	for i, line := range ta.Lines {
		ta.charCount += len([]rune(line)) // Use rune count for accuracy
		if i < len(ta.Lines)-1 {
			ta.charCount++ // Add 1 for the newline character between lines
		}
	}

	ta.wordCount = totalWords

	// Update bottom line text
	parts := []string{}
	if ta.showWordCount {
		parts = append(parts, fmt.Sprintf("Words: %d", ta.wordCount))
	}
	if ta.showCharCount {
		charStr := fmt.Sprintf("Chars: %d", ta.charCount)
		if ta.maxChars > 0 {
			charStr += fmt.Sprintf("/%d", ta.maxChars)
		}
		parts = append(parts, charStr)
	}
	ta.bottomLineText = strings.Join(parts, " | ")
}

// updateScrollState determines if scrolling is needed and updates the scrollbar.
func (ta *TextArea) updateScrollState() {
	contentHeight := len(ta.Lines)
	// Height available for text lines (excluding bottom count line)
	visibleHeight := ta.Height - 1
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	ta.needsScroll = contentHeight > visibleHeight
	ta.scrollBar.Visible = ta.needsScroll

	if ta.needsScroll {
		sbMaxValue := contentHeight - visibleHeight
		if sbMaxValue < 0 {
			sbMaxValue = 0
		}
		ta.scrollBar.MaxValue = sbMaxValue
		// Adjust scrollbar height in case text area height changed
		ta.scrollBar.Height = visibleHeight
		// Clamp current scroll value
		ta.scrollBar.SetValue(ta.scrollBar.Value) // This uses the setter which clamps
		ta.viewTopLine = ta.scrollBar.Value       // Sync viewTopLine with potentially clamped value
	} else {
		ta.scrollBar.MaxValue = 0
		ta.scrollBar.SetValue(0)
		ta.viewTopLine = 0
	}
}

// ensureCursorVisible adjusts viewTopLine so the cursor is visible.
func (ta *TextArea) ensureCursorVisible() {
	visibleHeight := ta.Height - 1
	if visibleHeight < 1 {
		visibleHeight = 1
	}
	bottomVisibleLine := ta.viewTopLine + visibleHeight - 1

	if ta.cursorLine < ta.viewTopLine {
		// Cursor is above the view
		ta.viewTopLine = ta.cursorLine
		ta.scrollBar.SetValue(ta.viewTopLine)
	} else if ta.cursorLine > bottomVisibleLine {
		// Cursor is below the view
		ta.viewTopLine = ta.cursorLine - visibleHeight + 1
		ta.scrollBar.SetValue(ta.viewTopLine)
	}
}

// Render draws the TextArea element.
func (ta *TextArea) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + ta.X
	absY := winY + ta.Y
	renderColor := ta.Color
	if ta.IsActive {
		renderColor = ta.ActiveColor
		// Optionally draw a border or change background when active
	}
	buffer.WriteString(renderColor)

	// --- Render Text Content ---
	textRenderWidth := ta.Width
	if ta.needsScroll {
		textRenderWidth-- // Make space for the scrollbar
	}
	if textRenderWidth < 0 {
		textRenderWidth = 0
	}
	// Height available for text lines
	visibleHeight := ta.Height - 1
	if visibleHeight < 0 {
		visibleHeight = 0
	}

	for i := 0; i < visibleHeight; i++ {
		lineIndex := ta.viewTopLine + i
		currentLineY := absY + i
		buffer.WriteString(MoveCursorCmd(currentLineY, absX))

		if lineIndex >= 0 && lineIndex < len(ta.Lines) {
			line := ta.Lines[lineIndex]
			// Basic line rendering (no horizontal scrolling or wrapping yet)
			visibleLine := ""
			runes := []rune(line)
			if len(runes) > textRenderWidth {
				// Naive truncation for now
				visibleLine = string(runes[:textRenderWidth])
			} else {
				visibleLine = line
			}
			buffer.WriteString(visibleLine)
			// Clear rest of the line within the text area width
			buffer.WriteString(strings.Repeat(" ", textRenderWidth-len([]rune(visibleLine))))
		} else {
			// Empty line within the text area
			buffer.WriteString(strings.Repeat(" ", textRenderWidth))
		}
	}
	buffer.WriteString(colors.Reset) // Reset color after text lines
	// --- End Text Content ---

	// --- Render ScrollBar ---
	// Pass absolute coordinates of the TextArea's top-left corner
	// The scrollbar's X, Y are relative to this origin.
	ta.scrollBar.Render(buffer, absX, absY, ta.Width)
	// --- End ScrollBar ---

	// --- Render Bottom Line (Word Count/Char Count) ---
	bottomLineY := absY + ta.Height - 1
	buffer.WriteString(MoveCursorCmd(bottomLineY, absX))
	buffer.WriteString(colors.Gray) // Use gray color for the status line
	countText := ta.bottomLineText
	countRunes := []rune(countText)
	if len(countRunes) > ta.Width {
		countText = string(countRunes[:ta.Width])
	}
	buffer.WriteString(countText)
	// Clear rest of bottom line
	buffer.WriteString(strings.Repeat(" ", ta.Width-len([]rune(countText))))
	buffer.WriteString(colors.Reset)
	// --- End Bottom Line ---

	// --- Calculate Cursor Position ---
	// This needs refinement based on horizontal scrolling/wrapping if implemented
	cursorScreenLine := ta.cursorLine - ta.viewTopLine
	cursorScreenCol := ta.cursorCol // Assuming no horizontal scroll/wrap for now

	// Clamp cursor screen position to be within the visible text area bounds
	if cursorScreenLine < 0 {
		cursorScreenLine = 0
		cursorScreenCol = 0 // Force to start if line is scrolled off top
	} else if cursorScreenLine >= visibleHeight {
		cursorScreenLine = visibleHeight - 1
		// Place cursor at the end of the last visible line if scrolled off bottom
		lastVisibleLineIdx := ta.viewTopLine + visibleHeight - 1
		if lastVisibleLineIdx >= 0 && lastVisibleLineIdx < len(ta.Lines) {
			lastLineLen := len([]rune(ta.Lines[lastVisibleLineIdx]))
			if cursorScreenCol > lastLineLen {
				cursorScreenCol = lastLineLen
			}
		} else {
			cursorScreenCol = 0 // Fallback if last visible line is invalid
		}
		// Clamp column to width as well
		if cursorScreenCol > textRenderWidth {
			cursorScreenCol = textRenderWidth
		}
	}

	// Clamp column based on current line length and visible width
	currentLineLen := 0
	if ta.cursorLine >= 0 && ta.cursorLine < len(ta.Lines) {
		currentLineLen = len([]rune(ta.Lines[ta.cursorLine]))
	}
	if cursorScreenCol > currentLineLen {
		cursorScreenCol = currentLineLen // Don't go past end of line
	}
	if cursorScreenCol < 0 {
		cursorScreenCol = 0
	} else if cursorScreenCol > textRenderWidth {
		cursorScreenCol = textRenderWidth // Clamp to visible width
	}

	ta.cursorAbsX = absX + cursorScreenCol
	ta.cursorAbsY = absY + cursorScreenLine
	// --- End Cursor Position Calculation ---
}

// NeedsCursor implements CursorManager interface
func (ta *TextArea) NeedsCursor() bool {
	return ta.IsActive
}

// GetCursorPosition implements CursorManager interface
func (ta *TextArea) GetCursorPosition() (int, int, bool) {
	if !ta.NeedsCursor() {
		return 0, 0, false
	}
	// Check if the calculated cursor position is within the visible text area
	visibleHeight := ta.Height - 1
	if visibleHeight < 0 {
		visibleHeight = 0
	}
	textRenderWidth := ta.Width
	if ta.needsScroll {
		textRenderWidth--
	}
	if textRenderWidth < 0 {
		textRenderWidth = 0
	}

	cursorScreenLine := ta.cursorLine - ta.viewTopLine
	cursorScreenCol := ta.cursorCol // Simplified check for now

	isCursorVisible := cursorScreenLine >= 0 && cursorScreenLine < visibleHeight &&
		cursorScreenCol >= 0 && cursorScreenCol <= textRenderWidth // Allow cursor at end of width

	return ta.cursorAbsX, ta.cursorAbsY, isCursorVisible
}

// --- Text Manipulation Methods ---

// clampCursorCol ensures cursor column is valid for the current line.
func (ta *TextArea) clampCursorCol() {
	if ta.cursorLine < 0 {
		ta.cursorLine = 0
	}
	if ta.cursorLine >= len(ta.Lines) {
		if len(ta.Lines) > 0 {
			ta.cursorLine = len(ta.Lines) - 1
		} else {
			ta.Lines = []string{""}
			ta.cursorLine = 0
		}
	}
	if len(ta.Lines) == 0 {
		ta.Lines = []string{""}
		ta.cursorLine = 0
		ta.cursorCol = 0
		return
	}
	currentLineLen := len([]rune(ta.Lines[ta.cursorLine]))
	if ta.cursorCol < 0 {
		ta.cursorCol = 0
	} else if ta.cursorCol > currentLineLen {
		ta.cursorCol = currentLineLen
	}
}

// InsertChar inserts a rune at the cursor position.
func (ta *TextArea) InsertChar(r rune) {
	if ta.IsActive {
		if ta.maxChars > 0 && ta.charCount >= ta.maxChars && r != '\n' {
			return
		}
		if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
			ta.clampCursorCol()
		}

		currentLineRunes := []rune(ta.Lines[ta.cursorLine])

		if r == '\n' {
			textAfterCursor := string(currentLineRunes[ta.cursorCol:])
			ta.Lines[ta.cursorLine] = string(currentLineRunes[:ta.cursorCol])
			nextLineIndex := ta.cursorLine + 1
			ta.Lines = append(ta.Lines[:nextLineIndex], append([]string{textAfterCursor}, ta.Lines[nextLineIndex:]...)...)
			ta.cursorLine = nextLineIndex
			ta.cursorCol = 0
		} else {
			newLine := string(currentLineRunes[:ta.cursorCol]) + string(r) + string(currentLineRunes[ta.cursorCol:])
			ta.Lines[ta.cursorLine] = newLine
			ta.cursorCol++
		}

		ta.clampCursorCol()
		ta.calculateCounts()
		ta.updateScrollState()
		ta.ensureCursorVisible()
	} else {
		return // Ignore input if not active
	}
}

// DeleteChar deletes the character before the cursor (Backspace).
func (ta *TextArea) DeleteChar() {
	if ta.IsActive {
		if ta.cursorLine == 0 && ta.cursorCol == 0 {
			return
		}
		if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
			ta.clampCursorCol()
		}

		if ta.cursorCol > 0 {
			currentLineRunes := []rune(ta.Lines[ta.cursorLine])
			newLine := string(currentLineRunes[:ta.cursorCol-1]) + string(currentLineRunes[ta.cursorCol:])
			ta.Lines[ta.cursorLine] = newLine
			ta.cursorCol--
		} else {
			prevLineIndex := ta.cursorLine - 1
			prevLineRunes := []rune(ta.Lines[prevLineIndex])
			currentLineRunes := []rune(ta.Lines[ta.cursorLine])
			newCursorCol := len(prevLineRunes)
			ta.Lines[prevLineIndex] = string(prevLineRunes) + string(currentLineRunes)
			ta.Lines = append(ta.Lines[:ta.cursorLine], ta.Lines[ta.cursorLine+1:]...)
			ta.cursorLine = prevLineIndex
			ta.cursorCol = newCursorCol
		}

		ta.clampCursorCol()
		ta.calculateCounts()
		ta.updateScrollState()
		ta.ensureCursorVisible()
	} else {
		return
	}
}

// DeleteForward deletes the character after the cursor (Delete).
func (ta *TextArea) DeleteForward() {
	if ta.IsActive {
		if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
			ta.clampCursorCol()
		}
		if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
			return
		}

		currentLineRunes := []rune(ta.Lines[ta.cursorLine])

		if ta.cursorCol < len(currentLineRunes) {
			newLine := string(currentLineRunes[:ta.cursorCol]) + string(currentLineRunes[ta.cursorCol+1:])
			ta.Lines[ta.cursorLine] = newLine
		} else if ta.cursorLine < len(ta.Lines)-1 {
			nextLineIndex := ta.cursorLine + 1
			nextLineRunes := []rune(ta.Lines[nextLineIndex])
			ta.Lines[ta.cursorLine] = string(currentLineRunes) + string(nextLineRunes)
			ta.Lines = append(ta.Lines[:nextLineIndex], ta.Lines[nextLineIndex+1:]...)
		} else {
			return
		}

		ta.clampCursorCol()
		ta.calculateCounts()
		ta.updateScrollState()
		ta.ensureCursorVisible()
	} else {
		return // Ignore input if not active
	}
}

// MoveCursorLeft moves the cursor one position left.
func (ta *TextArea) MoveCursorLeft() {
	if ta.cursorCol > 0 {
		ta.cursorCol--
	} else if ta.cursorLine > 0 {
		ta.cursorLine--
		if ta.cursorLine >= 0 && ta.cursorLine < len(ta.Lines) {
			ta.cursorCol = len([]rune(ta.Lines[ta.cursorLine]))
		} else {
			ta.cursorCol = 0
		}
	}
	ta.ensureCursorVisible()
}

// MoveCursorRight moves the cursor one position right.
func (ta *TextArea) MoveCursorRight() {
	if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
		ta.clampCursorCol()
	}
	if ta.cursorLine < 0 || ta.cursorLine >= len(ta.Lines) {
		return
	}

	currentLineLen := len([]rune(ta.Lines[ta.cursorLine]))
	if ta.cursorCol < currentLineLen {
		ta.cursorCol++
	} else if ta.cursorLine < len(ta.Lines)-1 {
		ta.cursorLine++
		ta.cursorCol = 0
	}
	ta.ensureCursorVisible()
}

// MoveCursorUp moves the cursor one line up.
func (ta *TextArea) MoveCursorUp() {
	if ta.cursorLine > 0 {
		ta.cursorLine--
		ta.clampCursorCol()
		ta.ensureCursorVisible()
	}
}

// MoveCursorDown moves the cursor one line down.
func (ta *TextArea) MoveCursorDown() {
	if ta.cursorLine < len(ta.Lines)-1 {
		ta.cursorLine++
		ta.clampCursorCol()
		ta.ensureCursorVisible()
	}
}

// MoveCursor is a general handler (can be used if input library provides deltas)
func (ta *TextArea) MoveCursor(deltaLine, deltaCol int) {
	targetLine := ta.cursorLine + deltaLine
	targetCol := ta.cursorCol + deltaCol

	if targetLine < 0 {
		targetLine = 0
	} else if targetLine >= len(ta.Lines) {
		targetLine = len(ta.Lines) - 1
	}

	if targetLine != ta.cursorLine {
		ta.cursorLine = targetLine
		ta.clampCursorCol()
		if deltaCol != 0 {
			ta.cursorCol = targetCol
			ta.clampCursorCol()
		}
	} else if deltaCol != 0 {
		ta.cursorCol = targetCol
		ta.clampCursorCol()
	}

	ta.ensureCursorVisible()
}

// GetText returns the full text content as a single string.
func (ta *TextArea) GetText() string {
	return strings.Join(ta.Lines, "\n")
}

// SetText replaces the entire content of the text area.
func (ta *TextArea) SetText(text string) {
	ta.Lines = strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if len(ta.Lines) == 0 {
		ta.Lines = []string{""}
	}
	ta.cursorLine = 0
	ta.cursorCol = 0
	ta.viewTopLine = 0
	ta.calculateCounts()
	ta.updateScrollState()
	ta.ensureCursorVisible()
}

// GetScrollbar returns the internal scrollbar.
func (ta *TextArea) GetScrollbar() *ScrollBar {
	return ta.scrollBar
}

// --- Menu Bar ---

// MenuItem represents a menu item that can be clicked to trigger an action or open a submenu
type MenuItem struct {
	Text        string
	Color       string
	ActiveColor string
	Action      func() bool // Function to execute when clicked (returns true to close menu)
	SubMenu     *Menu       // Optional submenu that opens when this item is activated
	IsActive    bool        // Whether this item is currently selected/active
	Width       int         // Width of this item
	X, Y        int         // Position relative to parent menu
	Parent      *Menu       // Reference to parent menu (nil for top-level items)
}

// NewMenuItem creates a new menu item with the given text and action
func NewMenuItem(text string, color, activeColor string, action func() bool) *MenuItem {
	return &MenuItem{
		Text:        text,
		Color:       color,
		ActiveColor: activeColor,
		Action:      action,
		Width:       len(text) + 2, // Add padding
		IsActive:    false,
	}
}

// Menu represents a menu containing menu items
type Menu struct {
	Items       []*MenuItem
	X, Y        int    // Position relative to parent (or window for top-level)
	Width       int    // Total width of the menu
	Height      int    // Total height of the menu
	Color       string // Background color
	BorderColor string // Border color (for submenus)
	SelectedIdx int    // Index of currently selected item
	IsOpen      bool   // Whether this menu is currently open
	IsTopLevel  bool   // Whether this is a top-level menu (in menu bar) or submenu
	zIndex      int    // Z-index for submenus
}

// GetZIndex implements ZIndexer interface for Menu
func (m *Menu) GetZIndex() int {
	if m.IsTopLevel {
		return 100 // Same as MenuBar
	}
	return 150 // Submenus appear above MenuBar
}

// NewMenu creates a new menu
func NewMenu(x, y int, color, borderColor string, isTopLevel bool) *Menu {
	return &Menu{
		Items:       make([]*MenuItem, 0),
		X:           x,
		Y:           y,
		Color:       color,
		BorderColor: borderColor,
		SelectedIdx: -1,
		IsOpen:      isTopLevel, // Top-level menus are always "open" (visible)
		IsTopLevel:  isTopLevel,
		zIndex:      150, // Higher than MenuBar but lower than prompts
	}
}

// AddItem adds a menu item to this menu
func (m *Menu) AddItem(item *MenuItem) {
	item.Parent = m
	if m.IsTopLevel {
		// For top-level menu, position items horizontally
		if len(m.Items) > 0 {
			prevItem := m.Items[len(m.Items)-1]
			item.X = prevItem.X + prevItem.Width
		} else {
			item.X = 0
		}
		item.Y = 0
	} else {
		// For submenus, position items vertically
		item.X = 1                // Account for border
		item.Y = len(m.Items) + 1 // Account for top border and previous items
	}
	m.Items = append(m.Items, item)

	// Update menu dimensions
	m.recalculateSize()
}

// recalculateSize updates the width and height of the menu based on its items
func (m *Menu) recalculateSize() {
	if m.IsTopLevel {
		// Top-level menu width is sum of all item widths
		width := 0
		for _, item := range m.Items {
			width += item.Width
		}
		m.Width = width
		m.Height = 1 // Top-level menus are one row high
	} else {
		// Submenu width is based on the widest item plus borders
		width := 0
		for _, item := range m.Items {
			if item.Width > width {
				width = item.Width
			}
		}
		m.Width = width + 4         // Add padding and borders
		m.Height = len(m.Items) + 2 // Items + top/bottom borders
	}
}

// AddSubMenu adds a submenu item to this menu
func (m *Menu) AddSubMenu(text string, color, activeColor string) *Menu {
	// Create the submenu
	submenu := NewMenu(0, 0, m.Color, m.BorderColor, false)

	// Create menu item that opens this submenu
	item := NewMenuItem(text, color, activeColor, nil)
	item.SubMenu = submenu

	// Add the item to this menu
	m.AddItem(item)

	return submenu
}

// SelectNext selects the next item in the menu
func (m *Menu) SelectNext() {
	if len(m.Items) == 0 {
		return
	}

	// Clear current selection
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.Items) {
		m.Items[m.SelectedIdx].IsActive = false
	}

	// Select next item
	m.SelectedIdx = (m.SelectedIdx + 1) % len(m.Items)
	m.Items[m.SelectedIdx].IsActive = true
}

// SelectPrevious selects the previous item in the menu
func (m *Menu) SelectPrevious() {
	if len(m.Items) == 0 {
		return
	}

	// Clear current selection
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.Items) {
		m.Items[m.SelectedIdx].IsActive = false
	}

	// Select previous item
	m.SelectedIdx--
	if m.SelectedIdx < 0 {
		m.SelectedIdx = len(m.Items) - 1
	}
	m.Items[m.SelectedIdx].IsActive = true
}

// ActivateSelected activates the currently selected item
func (m *Menu) ActivateSelected() bool {
	if m.SelectedIdx < 0 || m.SelectedIdx >= len(m.Items) {
		return false
	}

	item := m.Items[m.SelectedIdx]

	// If item has submenu, open it
	if item.SubMenu != nil {
		// Calculate submenu position relative to this item
		if m.IsTopLevel {
			// Position submenu directly below the menu item
			item.SubMenu.X = m.X + item.X
			item.SubMenu.Y = m.Y + 1 // Below top-level menu
		} else {
			// Position submenu to the right of this menu
			item.SubMenu.X = m.X + m.Width
			item.SubMenu.Y = m.Y + item.Y - 1 // Align with the current item
		}

		item.SubMenu.IsOpen = true
		item.SubMenu.SelectedIdx = 0
		if len(item.SubMenu.Items) > 0 {
			item.SubMenu.Items[0].IsActive = true
		}
		return true
	}

	// Otherwise, execute the action if defined
	if item.Action != nil {
		return item.Action()
	}

	return false
}

// CloseSubMenus recursively closes all open submenus
func (m *Menu) CloseSubMenus() {
	for _, item := range m.Items {
		if item.SubMenu != nil {
			item.SubMenu.IsOpen = false
			item.SubMenu.CloseSubMenus() // Recursively close nested submenus
		}
	}
}

// Render draws the menu
func (m *Menu) Render(buffer *strings.Builder, winX, winY int, _ int) {
	if !m.IsOpen {
		return
	}

	absX := winX + m.X
	absY := winY + m.Y

	if m.IsTopLevel {
		// Render top-level menu items horizontally
		for _, item := range m.Items {
			itemX := absX + item.X
			itemY := absY

			buffer.WriteString(MoveCursorCmd(itemY, itemX))

			// Select appropriate color
			if item.IsActive {
				buffer.WriteString(item.ActiveColor)
				buffer.WriteString(ReverseVideo())
			} else {
				buffer.WriteString(item.Color)
			}

			// Draw menu item with padding
			buffer.WriteString(" " + item.Text + " ")

			buffer.WriteString(colors.Reset)

			// Render submenu if active
			if item.SubMenu != nil && item.SubMenu.IsOpen {
				item.SubMenu.Render(buffer, winX, winY, 0)
			}
		}
	} else {
		// Render submenu with border
		buffer.WriteString(m.BorderColor)

		// Top border
		buffer.WriteString(MoveCursorCmd(absY, absX))
		buffer.WriteString("┌" + strings.Repeat("─", m.Width-2) + "┐")

		// Menu items
		for i, item := range m.Items {
			itemY := absY + i + 1

			// Left border
			buffer.WriteString(MoveCursorCmd(itemY, absX))
			buffer.WriteString("│")

			// Item text with appropriate color
			if item.IsActive {
				buffer.WriteString(item.ActiveColor)
				buffer.WriteString(ReverseVideo())
			} else {
				buffer.WriteString(item.Color)
			}

			// Pad item text to fill menu width
			paddedText := " " + item.Text
			padding := m.Width - 3 - len(item.Text)
			if padding > 0 {
				paddedText += strings.Repeat(" ", padding)
			}

			buffer.WriteString(paddedText)
			buffer.WriteString(colors.Reset)

			// Right border with submenu indicator if applicable
			buffer.WriteString(m.BorderColor)
			if item.SubMenu != nil {
				buffer.WriteString("▶")
			} else {
				buffer.WriteString("│")
			}
		}

		// Bottom border
		buffer.WriteString(MoveCursorCmd(absY+m.Height-1, absX))
		buffer.WriteString("└" + strings.Repeat("─", m.Width-2) + "┘")

		buffer.WriteString(colors.Reset)

		// Render any open submenu
		for _, item := range m.Items {
			if item.SubMenu != nil && item.SubMenu.IsOpen {
				item.SubMenu.Render(buffer, winX, winY, 0)
				break // Only one submenu can be open at a time
			}
		}
	}
}

// MenuBar is the main container for a menu system
type MenuBar struct {
	Menu            *Menu  // Top-level menu
	X, Y            int    // Position relative to window
	Width           int    // Total width of the menu bar
	BackgroundColor string // Background color for unused space
	IsActive        bool   // Whether the menu is currently active
	ActiveMenu      *Menu  // Currently active submenu (or nil if none)
	zIndex          int    // Default z-index for menus
}

// NewMenuBar creates a new menu bar
func NewMenuBar(x, y, width int, color, borderColor, bgColor string) *MenuBar {
	return &MenuBar{
		Menu:            NewMenu(x, y, color, borderColor, true),
		X:               x,
		Y:               y,
		Width:           width,
		BackgroundColor: bgColor,
		IsActive:        false,
		zIndex:          100, // Menus should appear above most elements
	}
}

// AddItem adds a menu item to the top-level menu
func (mb *MenuBar) AddItem(text string, color, activeColor string, action func() bool) *MenuItem {
	item := NewMenuItem(text, color, activeColor, action)
	mb.Menu.AddItem(item)
	return item
}

// AddSubMenu adds a submenu to the top-level menu
func (mb *MenuBar) AddSubMenu(text string, color, activeColor string) *Menu {
	return mb.Menu.AddSubMenu(text, color, activeColor)
}

// Activate activates the menu bar
func (mb *MenuBar) Activate() {
	mb.IsActive = true
	if mb.Menu.SelectedIdx < 0 && len(mb.Menu.Items) > 0 {
		mb.Menu.SelectedIdx = 0
		mb.Menu.Items[0].IsActive = true
	}
}

// Deactivate deactivates the menu bar and closes all submenus
func (mb *MenuBar) Deactivate() {
	mb.IsActive = false
	mb.ActiveMenu = nil

	// Clear selection but keep menus visible
	if mb.Menu.SelectedIdx >= 0 && mb.Menu.SelectedIdx < len(mb.Menu.Items) {
		mb.Menu.Items[mb.Menu.SelectedIdx].IsActive = false
	}
	mb.Menu.SelectedIdx = -1

	// Close all submenus
	mb.Menu.CloseSubMenus()
}

// NeedsCursor implements CursorManager interface
func (mb *MenuBar) NeedsCursor() bool {
	return false
}

// GetCursorPosition implements CursorManager interface
func (mb *MenuBar) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
}

// GetZIndex implements ZIndexer for MenuBar
func (mb *MenuBar) GetZIndex() int {
	return 100
}

// SelectNext selects the next menu item or delegates to active submenu
func (mb *MenuBar) SelectNext() {
	if !mb.IsActive {
		return
	}

	if mb.ActiveMenu != nil {
		mb.ActiveMenu.SelectNext()
	} else {
		mb.Menu.SelectNext()
	}
}

// SelectPrevious selects the previous menu item or delegates to active submenu
func (mb *MenuBar) SelectPrevious() {
	if !mb.IsActive {
		return
	}

	if mb.ActiveMenu != nil {
		mb.ActiveMenu.SelectPrevious()
	} else {
		mb.Menu.SelectPrevious()
	}
}

// MoveRight moves selection right in top-level menu
func (mb *MenuBar) MoveRight() {
	if !mb.IsActive || mb.ActiveMenu != nil {
		return
	}

	mb.Menu.SelectNext()
}

// MoveLeft moves selection left in top-level menu
func (mb *MenuBar) MoveLeft() {
	if !mb.IsActive || mb.ActiveMenu != nil {
		return
	}

	mb.Menu.SelectPrevious()
}

// MoveDown opens submenu if available
func (mb *MenuBar) MoveDown() {
	if !mb.IsActive {
		return
	}

	if mb.ActiveMenu != nil {
		mb.ActiveMenu.SelectNext()
		return
	}

	// Check if current item has submenu
	if mb.Menu.SelectedIdx >= 0 && mb.Menu.SelectedIdx < len(mb.Menu.Items) {
		item := mb.Menu.Items[mb.Menu.SelectedIdx]
		if item.SubMenu != nil {
			item.SubMenu.IsOpen = true
			item.SubMenu.SelectedIdx = 0
			if len(item.SubMenu.Items) > 0 {
				item.SubMenu.Items[0].IsActive = true
			}
			mb.ActiveMenu = item.SubMenu
		}
	}
}

// MoveUp closes current submenu if any
func (mb *MenuBar) MoveUp() {
	if !mb.IsActive {
		return
	}

	if mb.ActiveMenu != nil {
		// Check if this is a top-level submenu or nested
		if mb.ActiveMenu.SelectedIdx > 0 {
			mb.ActiveMenu.SelectPrevious()
		} else {
			// Close this menu and go up to parent
			mb.ActiveMenu.IsOpen = false
			mb.ActiveMenu = nil
		}
	}
}

// ActivateSelected activates the currently selected menu item
func (mb *MenuBar) ActivateSelected() bool {
	if !mb.IsActive {
		return false
	}

	var result bool

	if mb.ActiveMenu != nil {
		// Try to activate item in submenu
		result = mb.ActiveMenu.ActivateSelected()

		// If a submenu was opened, update active menu
		if mb.ActiveMenu.SelectedIdx >= 0 &&
			mb.ActiveMenu.SelectedIdx < len(mb.ActiveMenu.Items) {
			item := mb.ActiveMenu.Items[mb.ActiveMenu.SelectedIdx]
			if item.SubMenu != nil && item.SubMenu.IsOpen {
				mb.ActiveMenu = item.SubMenu
			}
		}
	} else {
		// Try to activate item in top-level menu
		result = mb.Menu.ActivateSelected()

		// If a submenu was opened, update active menu
		if mb.Menu.SelectedIdx >= 0 &&
			mb.Menu.SelectedIdx < len(mb.Menu.Items) {
			item := mb.Menu.Items[mb.Menu.SelectedIdx]
			if item.SubMenu != nil && item.SubMenu.IsOpen {
				mb.ActiveMenu = item.SubMenu
			}
		}
	}

	// If an action was executed and returned true, close the menu
	if result {
		mb.Deactivate()
	}

	return result
}

// Render draws the menu bar
func (mb *MenuBar) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + mb.X
	absY := winY + mb.Y

	// Draw background for entire menu bar width
	buffer.WriteString(mb.BackgroundColor)
	buffer.WriteString(MoveCursorCmd(absY, absX))
	buffer.WriteString(strings.Repeat(" ", mb.Width))
	buffer.WriteString(colors.Reset)

	// Render the menu and all its active submenus

	// Render only the top-level menu items here
	mb.Menu.Render(buffer, winX, winY, 0)
}

// --- Prompt ---

// PromptStyle defines whether the prompt is a single line or dialog box
type PromptStyle int

const (
	SingleLinePrompt PromptStyle = iota
	DialogBoxPrompt
)

// PromptButton represents a button in a prompt
type PromptButton struct {
	Text        string
	Color       string
	ActiveColor string
	IsActive    bool
	Action      func() bool // Returns true to close the prompt
}

// NewPromptButton creates a new button for a prompt
func NewPromptButton(text string, color, activeColor string, action func() bool) *PromptButton {
	return &PromptButton{
		Text:        text,
		Color:       color,
		ActiveColor: activeColor,
		IsActive:    false,
		Action:      action,
	}
}

// Prompt represents a message prompt with buttons
type Prompt struct {
	Title        string
	Message      string
	Buttons      []*PromptButton
	X, Y         int
	Width        int
	Height       int // Calculated based on content for dialog box
	Style        PromptStyle
	Color        string // Background color
	BorderColor  string // Border color for dialog box
	TitleColor   string // Title text color
	MessageColor string // Message text color
	IsActive     bool   // Whether the prompt is active
	SelectedIdx  int    // Index of selected button
	Modal        bool   // Whether the prompt blocks interaction with elements behind it
	zIndex       int    // Default z-index for prompts
}

// NewSingleLinePrompt creates a single-line prompt
func NewSingleLinePrompt(title, message string, x, y, width int, titleColor, messageColor string, buttons []*PromptButton) *Prompt {
	return &Prompt{
		Title:        title,
		Message:      message,
		Buttons:      buttons,
		X:            x,
		Y:            y,
		Width:        width,
		Height:       1,
		Style:        SingleLinePrompt,
		TitleColor:   titleColor,
		MessageColor: messageColor,
		IsActive:     false,
		SelectedIdx:  0,
		Modal:        false, // Single line prompts are not modal by default
		zIndex:       1000,  // Prompts should appear above everything
	}
}

// NewDialogPrompt creates a dialog box prompt
func NewDialogPrompt(title, message string, x, y, width int, color, borderColor, titleColor, messageColor string, buttons []*PromptButton) *Prompt {
	// Calculate height based on message length and width
	messageLines := 0
	messageChars := len(message)
	charsPerLine := width - 4 // Account for borders and padding
	if charsPerLine < 1 {
		charsPerLine = 1
	}

	// Simple word wrap calculation
	messageLines = (messageChars + charsPerLine - 1) / charsPerLine
	if messageLines < 1 {
		messageLines = 1
	}

	// Height = title(1) + padding(1) + messageLines + padding(1) + buttons(1) + borders(2)
	height := messageLines + 5

	return &Prompt{
		Title:        title,
		Message:      message,
		Buttons:      buttons,
		X:            x,
		Y:            y,
		Width:        width,
		Height:       height,
		Style:        DialogBoxPrompt,
		Color:        color,
		BorderColor:  borderColor,
		TitleColor:   titleColor,
		MessageColor: messageColor,
		IsActive:     false,
		SelectedIdx:  0,
		Modal:        true, // Dialog prompts are modal by default
		zIndex:       1000, // Prompts should appear above everything
	}
}

// SetActive activates or deactivates the prompt
func (p *Prompt) SetActive(active bool) {
	p.IsActive = active

	// Reset button state
	for i, button := range p.Buttons {
		button.IsActive = (i == p.SelectedIdx && active)
	}
}

// SelectNext selects the next button
func (p *Prompt) SelectNext() {
	if !p.IsActive || len(p.Buttons) <= 1 {
		return
	}

	// Clear current selection
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Buttons) {
		p.Buttons[p.SelectedIdx].IsActive = false
	}

	// Select next button
	p.SelectedIdx = (p.SelectedIdx + 1) % len(p.Buttons)
	p.Buttons[p.SelectedIdx].IsActive = true
}

// SelectPrevious selects the previous button
func (p *Prompt) SelectPrevious() {
	if !p.IsActive || len(p.Buttons) <= 1 {
		return
	}

	// Clear current selection
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Buttons) {
		p.Buttons[p.SelectedIdx].IsActive = false
	}

	// Select previous button
	p.SelectedIdx--
	if p.SelectedIdx < 0 {
		p.SelectedIdx = len(p.Buttons) - 1
	}
	p.Buttons[p.SelectedIdx].IsActive = true
}

// ActivateSelected activates the currently selected button
func (p *Prompt) ActivateSelected() bool {
	if !p.IsActive || p.SelectedIdx < 0 || p.SelectedIdx >= len(p.Buttons) {
		return false
	}

	button := p.Buttons[p.SelectedIdx]
	if button.Action != nil {
		result := button.Action()
		if result {
			p.SetActive(false)
		}
		return result
	}

	return false
}

// NeedsCursor implements CursorManager interface
func (p *Prompt) NeedsCursor() bool {
	return false
}

// GetCursorPosition implements CursorManager interface
func (p *Prompt) GetCursorPosition() (int, int, bool) {
	return 0, 0, false
}

// renderSingleLinePrompt renders the prompt as a single line
func (p *Prompt) renderSingleLinePrompt(buffer *strings.Builder, absX, absY int) {
	buffer.WriteString(MoveCursorCmd(absY, absX))

	// Calculate available space
	availWidth := p.Width

	// Render title if present
	if p.Title != "" {
		buffer.WriteString(p.TitleColor)
		buffer.WriteString(p.Title)
		buffer.WriteString(": ")
		buffer.WriteString(colors.Reset)
		availWidth -= len(p.Title) + 2
	}

	// Calculate space needed for buttons
	buttonSpace := 0
	for _, button := range p.Buttons {
		buttonSpace += len(button.Text) + 3 // [text] + space
	}

	// Render message with truncation if needed
	messageWidth := availWidth - buttonSpace - 1
	if messageWidth > 0 {
		buffer.WriteString(p.MessageColor)
		if len(p.Message) <= messageWidth {
			buffer.WriteString(p.Message)
		} else {
			buffer.WriteString(p.Message[:messageWidth-3] + "...")
		}
		buffer.WriteString(colors.Reset)
		buffer.WriteString(" ")
	}

	// Render buttons
	for i, button := range p.Buttons {
		if button.IsActive {
			buffer.WriteString(button.ActiveColor)
			buffer.WriteString(ReverseVideo())
		} else {
			buffer.WriteString(button.Color)
		}

		buffer.WriteString("[" + button.Text + "]")
		buffer.WriteString(colors.Reset)

		if i < len(p.Buttons)-1 {
			buffer.WriteString(" ")
		}
	}
}

// renderDialogPrompt renders the prompt as a dialog box
func (p *Prompt) renderDialogPrompt(buffer *strings.Builder, absX, absY int) {
	// Draw border
	buffer.WriteString(p.BorderColor)

	// Top border with title
	buffer.WriteString(MoveCursorCmd(absY, absX))
	buffer.WriteString("┌" + strings.Repeat("─", p.Width-2) + "┐")

	// Title (centered)
	if p.Title != "" {
		titleX := absX + (p.Width-len(p.Title)-2)/2
		buffer.WriteString(MoveCursorCmd(absY, titleX))
		buffer.WriteString("[ ")
		buffer.WriteString(p.TitleColor)
		buffer.WriteString(p.Title)
		buffer.WriteString(p.BorderColor)
		buffer.WriteString(" ]")
	}

	// Sides and background
	for i := 1; i < p.Height-1; i++ {
		buffer.WriteString(MoveCursorCmd(absY+i, absX))
		buffer.WriteString("│")
		buffer.WriteString(p.Color)
		buffer.WriteString(strings.Repeat(" ", p.Width-2))
		buffer.WriteString(p.BorderColor)
		buffer.WriteString("│")
	}

	// Bottom border
	buffer.WriteString(MoveCursorCmd(absY+p.Height-1, absX))
	buffer.WriteString("└" + strings.Repeat("─", p.Width-2) + "┘")

	// Message with simple word wrap
	messageWidth := p.Width - 4 // Account for borders and padding
	buffer.WriteString(p.MessageColor)

	// Simple word wrap implementation
	words := strings.Fields(p.Message)
	lineY := absY + 2 // Start after title and top border
	lineX := absX + 2 // Account for left border and padding
	lineWidth := 0

	for _, word := range words {
		wordLen := len(word)

		// Check if this word fits on the current line
		if lineWidth > 0 && lineWidth+wordLen+1 > messageWidth {
			// Word doesn't fit, move to next line
			lineY++
			lineWidth = 0
			buffer.WriteString(MoveCursorCmd(lineY, lineX))
		} else if lineWidth > 0 {
			// Add space before word
			buffer.WriteString(" ")
			lineWidth++
		}

		// Position cursor if starting a new line
		if lineWidth == 0 {
			buffer.WriteString(MoveCursorCmd(lineY, lineX))
		}

		// Add the word
		buffer.WriteString(word)
		lineWidth += wordLen
	}

	// Render buttons centered at bottom
	buttonY := absY + p.Height - 2 // One row up from bottom

	// Calculate total width of all buttons
	totalButtonWidth := 0
	for i, button := range p.Buttons {
		totalButtonWidth += len(button.Text) + 2 // [text]
		if i < len(p.Buttons)-1 {
			totalButtonWidth += 1 // space between buttons
		}
	}

	// Center buttons
	buttonX := absX + (p.Width-totalButtonWidth)/2
	buffer.WriteString(MoveCursorCmd(buttonY, buttonX))

	for i, button := range p.Buttons {
		if button.IsActive {
			buffer.WriteString(button.ActiveColor)
			buffer.WriteString(ReverseVideo())
		} else {
			buffer.WriteString(button.Color)
		}

		buffer.WriteString("[" + button.Text + "]")
		buffer.WriteString(colors.Reset)

		if i < len(p.Buttons)-1 {
			buffer.WriteString(" ")
		}
	}

	buffer.WriteString(colors.Reset)
}

// Render draws the prompt
func (p *Prompt) Render(buffer *strings.Builder, winX, winY int, _ int) {
	absX := winX + p.X
	absY := winY + p.Y

	if p.Style == SingleLinePrompt {
		p.renderSingleLinePrompt(buffer, absX, absY)
	} else {
		p.renderDialogPrompt(buffer, absX, absY)
	}
}

// GetButtons returns the buttons in this prompt
func (p *Prompt) GetButtons() []*PromptButton {
	return p.Buttons
}

// GetButton returns the button at the specified index or nil if index is invalid
func (p *Prompt) GetButton(index int) *PromptButton {
	if index >= 0 && index < len(p.Buttons) {
		return p.Buttons[index]
	}
	return nil
}

// IsModal returns whether this prompt is modal
func (p *Prompt) IsModal() bool {
	return p.Modal && p.IsActive
}
