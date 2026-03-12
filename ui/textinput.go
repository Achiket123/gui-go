package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/achiket123/gui-go/animation"
	"github.com/achiket123/gui-go/canvas"
)

// TextInputStyle defines the appearance of a TextInput component.
type TextInputStyle struct {
	Background   canvas.Color
	BorderColor  canvas.Color
	FocusBorder  canvas.Color
	BorderWidth  float32
	BorderRadius float32
	TextStyle    canvas.TextStyle
	HintStyle    canvas.TextStyle
	CursorColor  canvas.Color
}

// DefaultTextInputStyle returns a sensible default styling.
func DefaultTextInputStyle() TextInputStyle {
	return TextInputStyle{
		Background:   canvas.Hex("#1E1E2E"),
		BorderColor:  canvas.Hex("#313244"),
		FocusBorder:  canvas.Hex("#89B4FA"),
		BorderWidth:  1.5,
		BorderRadius: 8,
		TextStyle:    canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 14},
		HintStyle:    canvas.TextStyle{Color: canvas.Hex("#6C7086"), Size: 14},
		CursorColor:  canvas.Hex("#89B4FA"),
	}
}

// TextInput is a single-line text input field.
type TextInput struct {
	Text     string
	Hint     string
	Style    TextInputStyle
	OnChange func(string)

	bounds    canvas.Rect
	focused   bool
	cursorOn  bool
	blinkTime float64
	anim      *animation.AnimationController
}

// NewTextInput creates a new TextInput component.
func NewTextInput(hint string) *TextInput {
	ctrl := animation.NewController(150 * time.Millisecond)
	return &TextInput{
		Hint:  hint,
		Style: DefaultTextInputStyle(),
		anim:  ctrl,
	}
}

func (t *TextInput) Bounds() canvas.Rect { return t.bounds }

func (t *TextInput) Tick(delta float64) {
	t.anim.Tick(delta)
	if t.focused {
		t.blinkTime += delta
		if t.blinkTime > 0.5 { // 500ms blink rate
			t.blinkTime = 0
			t.cursorOn = !t.cursorOn
		}
	} else {
		t.cursorOn = false
		t.blinkTime = 0
	}
}

func (t *TextInput) HandleEvent(e Event) bool {
	switch e.Type {
	case EventMouseDown:
		if e.X >= t.bounds.X && e.X <= t.bounds.X+t.bounds.W &&
			e.Y >= t.bounds.Y && e.Y <= t.bounds.Y+t.bounds.H {
			if !t.focused {
				t.focused = true
				t.anim.Forward()
				t.cursorOn = true
				t.blinkTime = 0
			}
			return true
		}
		if t.focused {
			t.focused = false
			t.anim.Reverse()
		}
		return false
	case EventKeyDown:
		if !t.focused {
			return false
		}
		if e.Key == "BackSpace" {
			if len(t.Text) > 0 {
				_, size := utf8.DecodeLastRuneInString(t.Text)
				t.Text = t.Text[:len(t.Text)-size]
			}
			t.cursorOn = true
			t.blinkTime = 0
			if t.OnChange != nil {
				t.OnChange(t.Text)
			}
			return true
		}
		if len(e.Key) == 1 { // Simple printable character check
			char := e.Key
			if e.Shift {
				char = strings.ToUpper(char)
			} else {
				char = strings.ToLower(char)
			}
			t.Text += char
			t.cursorOn = true
			t.blinkTime = 0
			if t.OnChange != nil {
				t.OnChange(t.Text)
			}
			return true
		}
		if e.Key == "space" {
			t.Text += " "
			t.cursorOn = true
			t.blinkTime = 0
			return true
		}
		return true // Consume other keys while focused
	}
	return false
}

func (t *TextInput) Draw(c *canvas.Canvas, x, y, w, h float32) {
	t.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	// 1. Draw Background
	c.DrawRoundedRect(x, y, w, h, t.Style.BorderRadius, canvas.FillPaint(t.Style.Background))

	// 2. Draw Text or Hint
	padX := float32(12)
	textY := y + h/2 + t.Style.TextStyle.Size*0.35 // vertical center approximation

	var textW float32
	if t.Text == "" {
		if !t.focused {
			c.DrawText(x+padX, textY, t.Hint, t.Style.HintStyle)
		}
	} else {
		c.Save()
		c.ClipRect(x+padX, y, w-padX*2, h)
		sz := c.MeasureText(t.Text, t.Style.TextStyle)
		textW = sz.W

		drawX := x + padX
		// Scroll text if it overflows bounds
		if textW > w-padX*2 {
			drawX = x + padX - (textW - (w - padX*2))
		}
		c.DrawText(drawX, textY, t.Text, t.Style.TextStyle)
		c.Restore()
	}

	// 3. Draw Cursor
	if t.cursorOn && t.focused {
		cx := x + padX + textW
		if textW > w-padX*2 {
			cx = x + w - padX
		}
		cy1 := y + h/2 - t.Style.TextStyle.Size/2
		cy2 := y + h/2 + t.Style.TextStyle.Size/2
		c.DrawLine(cx, cy1, cx, cy2, canvas.StrokePaint(t.Style.CursorColor, 1.5))
	}

	// 4. Draw Border (animated focus transition)
	focusT := float32(t.anim.Value())
	bc := t.Style.BorderColor
	fc := t.Style.FocusBorder
	r := bc.R + (fc.R-bc.R)*focusT
	g := bc.G + (fc.G-bc.G)*focusT
	b := bc.B + (fc.B-bc.B)*focusT
	a := bc.A + (fc.A-bc.A)*focusT
	currBorder := canvas.Color{R: r, G: g, B: b, A: a}

	c.DrawRoundedRect(x, y, w, h, t.Style.BorderRadius, canvas.StrokePaint(currBorder, t.Style.BorderWidth))
}
