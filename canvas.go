package goui

import (
	"github.com/achiket/gui-go/platform"
)

// Point is a 2D integer coordinate used for polygon drawing.
type Point = platform.Point

// Canvas is the drawing surface passed to every OnDraw callback.
// It wraps the underlying drawable (either the window or an offscreen pixmap)
// and translates Go-friendly calls into raw Xlib calls via the platform package.
//
// Canvas methods must only be called from within an OnDraw callback.
type Canvas struct {
	display interface{} // *Window — avoid circular type; accessed via win
	win     *Window

	// drawable is the X11 Drawable ID we actually draw to (may be a pixmap).
	drawable uintptr

	width  int
	height int
}

func newCanvas(w *Window) *Canvas {
	drawable := w.pixmap
	if drawable == 0 {
		drawable = w.xwin
	}
	return &Canvas{
		win:      w,
		drawable: drawable,
		width:    w.width,
		height:   w.height,
	}
}

// --- State ---

// SetColor sets the current draw color.
func (c *Canvas) SetColor(col Color) {
	w := c.win
	pixel := col.ToXPixel(w.display, w.colormap)
	platform.SetForeground(w.display, w.gc, pixel)
}

// SetLineWidth sets the line thickness (in pixels).
func (c *Canvas) SetLineWidth(width int) {
	w := c.win
	platform.SetLineWidth(w.display, w.gc, width)
}

// SetFont loads and activates an X11 XLFD font by name.
// Example: "-*-fixed-medium-r-normal--13-*-*-*-*-*-*-*"
func (c *Canvas) SetFont(name string) {
	w := c.win
	font := platform.LoadFont(w.display, name)
	if font != nil {
		platform.SetFont(w.display, w.gc, font)
		w.currentFont = font
	}
}

// --- Shapes ---

// FillRect draws a solid filled rectangle.
func (c *Canvas) FillRect(x, y, width, height int) {
	w := c.win
	platform.FillRectangle(w.display, c.drawable, w.gc, x, y, width, height)
}

// DrawRect draws a rectangle outline.
func (c *Canvas) DrawRect(x, y, width, height int) {
	w := c.win
	platform.DrawRectangle(w.display, c.drawable, w.gc, x, y, width, height)
}

// FillCircle draws a filled circle centered at (cx, cy) with radius r.
func (c *Canvas) FillCircle(cx, cy, r int) {
	w := c.win
	platform.FillArc(w.display, c.drawable, w.gc, cx-r, cy-r, r*2, r*2, 0, 360*64)
}

// DrawCircle draws a circle outline centered at (cx, cy) with radius r.
func (c *Canvas) DrawCircle(cx, cy, r int) {
	w := c.win
	platform.DrawArc(w.display, c.drawable, w.gc, cx-r, cy-r, r*2, r*2, 0, 360*64)
}

// DrawLine draws a straight line from (x1,y1) to (x2,y2).
func (c *Canvas) DrawLine(x1, y1, x2, y2 int) {
	w := c.win
	platform.DrawLine(w.display, c.drawable, w.gc, x1, y1, x2, y2)
}

// FillPolygon draws a filled polygon defined by a slice of points.
func (c *Canvas) FillPolygon(points []Point) {
	w := c.win
	platform.FillPolygon(w.display, c.drawable, w.gc, points)
}

// DrawPolygon draws a polygon outline (automatically closed).
func (c *Canvas) DrawPolygon(points []Point) {
	w := c.win
	platform.DrawPolygon(w.display, c.drawable, w.gc, points)
}

// --- Text ---

// DrawText draws a string at position (x, y).
// y is the baseline position.
func (c *Canvas) DrawText(x, y int, text string) {
	w := c.win
	platform.DrawString(w.display, c.drawable, w.gc, x, y, text)
}

// MeasureText returns the pixel width and height of the given string
// using the currently active font.
func (c *Canvas) MeasureText(text string) (width, height int) {
	w := c.win
	if w.currentFont == nil {
		return len(text) * 6, 13 // fallback estimate
	}
	return platform.TextExtents(w.display, w.currentFont, text)
}

// --- Images ---

// DrawImage draws a goui.Image at position (x, y).
func (c *Canvas) DrawImage(img *Image, x, y int) {
	if img == nil {
		return
	}
	w := c.win
	h := img.xImageHandle(w.display, w.screen)
	platform.PutXImage(w.display, c.drawable, w.gc, h, x, y, img.Width(), img.Height())
}

// SetPixel sets a single pixel at (x, y) to the given color.
func (c *Canvas) SetPixel(x, y int, col Color) {
	w := c.win
	pixel := col.ToXPixel(w.display, w.colormap)
	platform.SetForeground(w.display, w.gc, pixel)
	platform.FillRectangle(w.display, c.drawable, w.gc, x, y, 1, 1)
}

// --- Utility ---

// Clear fills the entire canvas with the window's background color.
func (c *Canvas) Clear() {
	w := c.win
	pixel := w.bgColor.ToXPixel(w.display, w.colormap)
	platform.SetForeground(w.display, w.gc, pixel)
	platform.FillRectangle(w.display, c.drawable, w.gc, 0, 0, c.width, c.height)
}

// Width returns the current canvas width in pixels.
func (c *Canvas) Width() int { return c.width }

// Height returns the current canvas height in pixels.
func (c *Canvas) Height() int { return c.height }
