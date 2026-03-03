package goui

import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/achiket/gui-go/platform"
)

// Color represents an RGB color.
type Color struct {
	R, G, B uint8
}

// ToXPixel converts a Color to an X11 pixel value using XAllocColor.
// display and colormap are C pointers passed as unsafe.Pointer.
func (c Color) ToXPixel(display, colormap unsafe.Pointer) uint64 {
	return platform.AllocColor(display, colormap, c.R, c.G, c.B)
}

// --- Predefined color constants ---

var (
	Black       = Color{0, 0, 0}
	White       = Color{255, 255, 255}
	Red         = Color{255, 0, 0}
	Green       = Color{0, 200, 0}
	Blue        = Color{0, 0, 255}
	Yellow      = Color{255, 255, 0}
	Cyan        = Color{0, 255, 255}
	Magenta     = Color{255, 0, 255}
	Gray        = Color{128, 128, 128}
	LightGray   = Color{192, 192, 192}
	DarkGray    = Color{64, 64, 64}
	Orange      = Color{255, 165, 0}
	Pink        = Color{255, 105, 180}
	Purple      = Color{128, 0, 128}
	Transparent = Color{0, 0, 0} // X11 has no real transparency — treated as black
)

// --- Constructors ---

// RGB creates a Color from red, green, blue components (0–255).
func RGB(r, g, b uint8) Color {
	return Color{r, g, b}
}

// RGBA creates a Color from RGBA components.
// Alpha is accepted for API completeness but ignored (X11 has no compositing here).
func RGBA(r, g, b, _ uint8) Color {
	return Color{r, g, b}
}

// Hex parses a CSS-style hex color string (#RRGGBB or #RGB).
// Panics on invalid input.
func Hex(hex string) Color {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	switch len(hex) {
	case 3:
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	case 6:
		// ok
	default:
		panic(fmt.Sprintf("goui.Hex: invalid hex color %q", "#"+hex))
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		panic(fmt.Sprintf("goui.Hex: invalid hex color %q: %v", "#"+hex, err))
	}
	return Color{
		R: uint8(v >> 16),
		G: uint8((v >> 8) & 0xff),
		B: uint8(v & 0xff),
	}
}
