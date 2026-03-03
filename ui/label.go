package ui

import (
	"github.com/achiket/gui-go/canvas"
)

// LabelAlign describes text alignment in a Label.
type LabelAlign int

const (
	AlignLeft LabelAlign = iota
	AlignCenter
	AlignRight
)

// Label is a simple styled text component with no interaction.
type Label struct {
	Text  string
	Style canvas.TextStyle
	Align LabelAlign

	bounds canvas.Rect
}

// NewLabel creates a Label with given text and style.
func NewLabel(text string, style canvas.TextStyle) *Label {
	return &Label{Text: text, Style: style}
}

func (l *Label) Bounds() canvas.Rect      { return l.bounds }
func (l *Label) Tick(_ float64)           {}
func (l *Label) HandleEvent(_ Event) bool { return false }

func (l *Label) Draw(c *canvas.Canvas, x, y, w, h float32) {
	l.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	sz := c.MeasureText(l.Text, l.Style)
	var tx float32
	switch l.Align {
	case AlignCenter:
		tx = x + (w-sz.W)/2
	case AlignRight:
		tx = x + w - sz.W
	default:
		tx = x
	}
	ty := y + (h+l.Style.Size)/2 - 2
	c.DrawText(tx, ty, l.Text, l.Style)
}

// Panel is a styled box (background, border, optional shadow) that can
// contain other draw logic via a DrawFunc callback.
type DrawFunc func(c *canvas.Canvas, bounds canvas.Rect)

// Panel draws a rounded rectangle with a background color.
type Panel struct {
	Background   canvas.Color
	BorderColor  canvas.Color
	BorderWidth  float32
	BorderRadius float32
	Child        DrawFunc

	bounds canvas.Rect
}

// NewPanel creates a Panel with given background.
func NewPanel(bg canvas.Color, radius float32) *Panel {
	return &Panel{Background: bg, BorderRadius: radius}
}

func (p *Panel) Bounds() canvas.Rect      { return p.bounds }
func (p *Panel) Tick(_ float64)           {}
func (p *Panel) HandleEvent(_ Event) bool { return false }

func (p *Panel) Draw(c *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	c.DrawRoundedRect(x, y, w, h, p.BorderRadius, canvas.FillPaint(p.Background))
	if p.BorderWidth > 0 {
		bp := canvas.StrokePaint(p.BorderColor, p.BorderWidth)
		c.DrawRoundedRect(x, y, w, h, p.BorderRadius, bp)
	}
	if p.Child != nil {
		p.Child(c, p.bounds)
	}
}
