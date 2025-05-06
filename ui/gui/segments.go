package gui

import (
	"secshell/colors"
	"strings"
)

// Segment represents a vertical section of the screen that can contain
// multiple UI elements. Segments can be arranged horizontally next to each other.
type Segment struct {
	X, Y          int         // Position relative to parent container/window
	Width, Height int         // Dimensions
	Elements      []UIElement // Elements contained within this segment
	BgColor       string      // Background color
	BorderStyle   string      // Optional border style from BoxTypes
	BorderColor   string      // Border color if border is used
	Title         string      // Optional title for bordered segments
	TitleColor    string      // Title color
}

// NewSegment creates a new segment with the specified dimensions
func NewSegment(x, y, width, height int, bgColor string) *Segment {
	return &Segment{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		Elements:    make([]UIElement, 0),
		BgColor:     bgColor,
		BorderStyle: "", // No border by default
	}
}

// NewBorderedSegment creates a new segment with a border
func NewBorderedSegment(x, y, width, height int, bgColor, borderStyle, borderColor, title, titleColor string) *Segment {
	// Use default border style if invalid
	if _, exists := BoxTypes[borderStyle]; !exists {
		borderStyle = "single" // Default style
	}

	return &Segment{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		Elements:    make([]UIElement, 0),
		BgColor:     bgColor,
		BorderStyle: borderStyle,
		BorderColor: borderColor,
		Title:       title,
		TitleColor:  titleColor,
	}
}

// AddElement adds a UI element to the segment
func (s *Segment) AddElement(element UIElement) {
	s.Elements = append(s.Elements, element)
}

// Render draws the segment and all elements within it
func (s *Segment) Render(buffer *strings.Builder, winX, winY int, _ int) {
	// Calculate absolute position
	absX := winX + s.X
	absY := winY + s.Y

	// 1. Fill background color for the entire segment area first
	// Ensure filling respects exact width and height.
	if s.BgColor != "" {
		buffer.WriteString(s.BgColor)
		for y := 0; y < s.Height; y++ {
			buffer.WriteString(MoveCursorCmd(absY+y, absX))
			buffer.WriteString(strings.Repeat(" ", s.Width))
		}
		buffer.WriteString(colors.Reset) // Reset after filling background
	} else {
		// If no background color, explicitly clear the area with spaces
		// to prevent artifacts from previous renders.
		for y := 0; y < s.Height; y++ {
			buffer.WriteString(MoveCursorCmd(absY+y, absX))
			buffer.WriteString(strings.Repeat(" ", s.Width))
		}
	}

	// Store content area dimensions and starting position (relative to absolute segment pos)
	contentOffsetX := 0
	contentOffsetY := 0
	contentWidth := s.Width
	contentHeight := s.Height

	// 2. Draw border if specified (over the background)
	if s.BorderStyle != "" {
		box, exists := BoxTypes[s.BorderStyle]
		if !exists {
			box = BoxTypes["single"] // Fallback
		}
		buffer.WriteString(s.BorderColor)

		// Draw top border with optional title
		buffer.WriteString(MoveCursorCmd(absY, absX))
		buffer.WriteString(box.TopLeft)

		titleStr := ""
		titleLen := len([]rune(s.Title))             // Use rune count for title length
		if titleLen > 0 && titleLen <= (s.Width-4) { // Ensure space for border corners and padding
			titleStr = " " + s.Title + " "
			titleLen += 2 // Account for padding spaces
		}

		if titleLen > 0 {
			leftBorderLen := (s.Width - 2 - titleLen) / 2
			rightBorderLen := s.Width - 2 - titleLen - leftBorderLen
			if leftBorderLen < 0 {
				leftBorderLen = 0
			} // Prevent negative repeats
			if rightBorderLen < 0 {
				rightBorderLen = 0
			}

			buffer.WriteString(strings.Repeat(box.Horizontal, leftBorderLen))
			buffer.WriteString(s.TitleColor) // Switch to title color
			buffer.WriteString(titleStr)
			buffer.WriteString(s.BorderColor) // Switch back to border color
			buffer.WriteString(strings.Repeat(box.Horizontal, rightBorderLen))
		} else {
			// No title or doesn't fit
			buffer.WriteString(strings.Repeat(box.Horizontal, s.Width-2))
		}
		buffer.WriteString(box.TopRight)

		// Draw sides
		for i := 1; i < s.Height-1; i++ {
			buffer.WriteString(MoveCursorCmd(absY+i, absX))
			buffer.WriteString(box.Vertical)
			// No need to fill inside here, background was done first
			buffer.WriteString(MoveCursorCmd(absY+i, absX+s.Width-1))
			buffer.WriteString(box.Vertical)
		}

		// Draw bottom border
		buffer.WriteString(MoveCursorCmd(absY+s.Height-1, absX))
		buffer.WriteString(box.BottomLeft)
		buffer.WriteString(strings.Repeat(box.Horizontal, s.Width-2))
		buffer.WriteString(box.BottomRight)

		buffer.WriteString(colors.Reset) // Reset after drawing border

		// Adjust content area for border
		contentOffsetX = 1
		contentOffsetY = 1
		contentWidth -= 2
		contentHeight -= 2
		if contentWidth < 0 {
			contentWidth = 0
		} // Prevent negative dimensions
		if contentHeight < 0 {
			contentHeight = 0
		}
	}

	// 3. Render all elements within segment's adjusted content area
	// Elements are rendered relative to the content area's top-left corner.
	contentAbsX := absX + contentOffsetX
	contentAbsY := absY + contentOffsetY
	for _, element := range s.Elements {
		// Pass the absolute top-left of the content area and the content width/height
		element.Render(buffer, contentAbsX, contentAbsY, contentWidth)
	}
}

// SegmentGroup manages a collection of segments arranged horizontally
type SegmentGroup struct {
	X, Y           int        // Position relative to window
	Segments       []*Segment // List of segments in this group
	SeparatorChar  string     // Character for the vertical separator
	SeparatorColor string     // Color for the separator
}

// NewSegmentGroup creates a new segment group at the specified position
func NewSegmentGroup(x, y int) *SegmentGroup {
	return &SegmentGroup{
		X:              x,
		Y:              y,
		Segments:       make([]*Segment, 0),
		SeparatorChar:  "│",         // Default separator character is the vertical line
		SeparatorColor: colors.Gray, // Default separator color
	}
}

// AddSegment adds a segment to the group and adjusts its X position
func (sg *SegmentGroup) AddSegment(segment *Segment) {
	// Calculate the X position for the new segment based on previous segments + separators
	currentX := sg.X // Start at the group's X

	// If this isn't the first segment, add space for the divider
	if len(sg.Segments) > 0 {
		currentX += 1 // Add space for the divider column
	}

	for _, s := range sg.Segments {
		currentX += s.Width + 1 // Add width of segment + 1 for separator
	}

	// Set segment's position relative to the window (using calculated X)
	segment.X = currentX
	segment.Y = sg.Y // Align Y with the group's Y

	// Add to the segments list
	sg.Segments = append(sg.Segments, segment)
}

// AddSegments adds multiple segments at once
func (sg *SegmentGroup) AddSegments(segments ...*Segment) {
	for _, segment := range segments {
		sg.AddSegment(segment)
	}
}

// GetTotalWidth returns the combined width of all segments including separators
func (sg *SegmentGroup) GetTotalWidth() int {
	totalWidth := 0
	for i, segment := range sg.Segments {
		totalWidth += segment.Width
		if i < len(sg.Segments)-1 {
			totalWidth++ // Add 1 for separator after each segment except the last
		}
	}
	return totalWidth
}

// GetMaxHeight returns the height of the tallest segment
func (sg *SegmentGroup) GetMaxHeight() int {
	maxHeight := 0
	for _, segment := range sg.Segments {
		if segment.Height > maxHeight {
			maxHeight = segment.Height
		}
	}
	return maxHeight
}

// Render implements the UIElement interface for the segment group
func (sg *SegmentGroup) Render(buffer *strings.Builder, winX, winY int, _ int) {
	maxHeight := sg.GetMaxHeight() // Determine max height for drawing separators

	// Render each segment and draw separators between them
	for i, segment := range sg.Segments {
		// Render the segment itself (it uses its own X, Y relative to winX, winY)
		segment.Render(buffer, winX, winY, segment.Width)

		// Draw separator *after* the segment, if it's not the last one
		if i < len(sg.Segments)-1 {
			separatorX := winX + segment.X + segment.Width // Position after the segment
			separatorY := winY + sg.Y                      // Align with group's Y

			// Make the separator more prominent - use full-height line
			buffer.WriteString(sg.SeparatorColor)
			for row := 0; row < maxHeight; row++ {
				buffer.WriteString(MoveCursorCmd(separatorY+row, separatorX))
				buffer.WriteString(sg.SeparatorChar) // Uses the configured separator character (default: "│")
			}
			buffer.WriteString(colors.Reset)
		}
	}
}
