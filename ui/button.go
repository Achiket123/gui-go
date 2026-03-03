package ui

import (
	"time"

	"github.com/achiket/gui-go/animation"
	"github.com/achiket/gui-go/canvas"
)

// ButtonStyle describes the visual appearance of a Button.
type ButtonStyle struct {
	Background   canvas.Color
	HoverColor   canvas.Color
	PressColor   canvas.Color
	TextStyle    canvas.TextStyle
	BorderRadius float32
	Padding      canvas.EdgeInsets
}

// DefaultButtonStyle returns a sensible dark-themed button style.
func DefaultButtonStyle() ButtonStyle {
	return ButtonStyle{
		Background:   canvas.Hex("#3B82F6"),
		HoverColor:   canvas.Hex("#60A5FA"),
		PressColor:   canvas.Hex("#1D4ED8"),
		TextStyle:    canvas.TextStyle{Color: canvas.White, Size: 14},
		BorderRadius: 8,
		Padding:      canvas.Symmetric(16, 10),
	}
}

// Button is a retained clickable button component with animated hover/press states.
type Button struct {
	Label   string
	OnClick func()
	Style   ButtonStyle

	bounds    canvas.Rect
	hover     bool
	pressed   bool
	colorCtrl *animation.AnimationController
	curColor  canvas.Color
}

// NewButton creates a Button with the default style.
func NewButton(label string, onClick func()) *Button {
	s := DefaultButtonStyle()
	b := &Button{
		Label:    label,
		OnClick:  onClick,
		Style:    s,
		curColor: s.Background,
	}
	b.colorCtrl = animation.NewController(150 * time.Millisecond)
	return b
}

func (b *Button) Bounds() canvas.Rect { return b.bounds }

func (b *Button) Tick(delta float64) {
	b.colorCtrl.Tick(delta)
	t := float32(b.colorCtrl.Value())

	var target canvas.Color
	if b.pressed {
		target = b.Style.PressColor
	} else if b.hover {
		target = b.Style.HoverColor
	} else {
		target = b.Style.Background
	}
	// Snap lerp: faster approach using t as a speed multiplier
	b.curColor = canvas.Lerp(b.curColor, target, t*0.4+0.15)
}

func (b *Button) Draw(c *canvas.Canvas, x, y, w, h float32) {
	b.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	p := canvas.FillPaint(b.curColor)
	c.DrawRoundedRect(x, y, w, h, b.Style.BorderRadius, p)

	ts := b.Style.TextStyle
	sz := c.MeasureText(b.Label, ts)
	tx := x + (w-sz.W)/2
	ty := y + (h+sz.H)/2 - 2
	c.DrawText(tx, ty, b.Label, ts)
}

func (b *Button) HandleEvent(e Event) bool {
	inBounds := e.X >= b.bounds.X && e.X <= b.bounds.X+b.bounds.W &&
		e.Y >= b.bounds.Y && e.Y <= b.bounds.Y+b.bounds.H

	switch e.Type {
	case EventMouseMove:
		b.hover = inBounds
	case EventMouseDown:
		if inBounds {
			b.pressed = true
			b.colorCtrl.Forward()
			return true
		}
	case EventMouseUp:
		if b.pressed {
			b.pressed = false
			b.colorCtrl.Reset()   // jump to 0
			b.colorCtrl.Forward() // now animates 0→1 toward Background
			if inBounds && b.OnClick != nil {
				b.OnClick()
			}
			return true
		}
	}
	return false
}
