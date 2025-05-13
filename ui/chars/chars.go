package chars

import (
	_ "embed" // Required for embedding font data
)

//go:embed fonts/MesloLGSNerdFont-Regular.ttf
var fontData []byte

const (
	LeftCircleHalfFilled     = ""
	RightCircleHalfFilled    = ""
	LeftCircleHalf           = ""
	RightCircleHalf          = ""
	LeftArrowFilled          = ""
	RightArrowFilled         = ""
	LeftArrow                = ""
	RightArrow               = ""
	ThinRightArrow           = "⟩"
	GlitchDivider            = ""
	ThreeDashedVertical      = "┆"
	SimpleLine               = "│"
	LeftFlameFilled          = ""
	RightFlameFilled         = ""
	LeftFlame                = ""
	RightFlame               = ""
	LeftGlitchFilled         = ""
	RightGlitchFilled        = ""
	RoundedCornerLeftTop     = "╭"
	RoundedCornerRightTop    = "╮"
	RoundedCornerLeftBottom  = "╰"
	RoundedCornerRightBottom = "╯"
	SquareCornerLeftTop      = "┌"
	SquareCornerRightTop     = "┐"
	SquareCornerLeftBottom   = "└"
	SquareCornerRightBottom  = "┘"
	DoubleCornerLeftTop      = "╔"
	DoubleCornerRightTop     = "╗"
	DoubleCornerLeftBottom   = "╚"
	DoubleCornerRightBottom  = "╝"
)

func InitFont() {
	_ = fontData // This is just to ensure the font data is embedded
}
