// Package ui — inputs_advanced.go
//
// Advanced input widgets:
//   MultiLineTextInput — scrollable textarea with full cursor editing
//   NumberInput        — numeric field with +/− stepper buttons
//   SearchInput        — text field with search icon, clear button, debounce
package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
)

// ═══════════════════════════════════════════════════════════════════════════════
// MultiLineTextInput
// ═══════════════════════════════════════════════════════════════════════════════

// MultiLineStyle configures a MultiLineTextInput.
type MultiLineStyle struct {
	Background   canvas.Color
	Border       canvas.Color
	FocusBorder  canvas.Color
	Radius       float32
	TextStyle    canvas.TextStyle
	HintStyle    canvas.TextStyle
	CursorColor  canvas.Color
	SelectBg     canvas.Color
	LineHeight   float32 // multiplier (default 1.5)
	Padding      canvas.EdgeInsets
	ScrollbarW   float32
}

// DefaultMultiLineStyle returns a theme-aware MultiLineStyle.
func DefaultMultiLineStyle() MultiLineStyle {
	th := theme.Current()
	return MultiLineStyle{
		Background:  th.Colors.BgSurface,
		Border:      th.Colors.Border,
		FocusBorder: th.Colors.BorderFocus,
		Radius:      th.Radius.MD,
		TextStyle:   th.Type.Body,
		HintStyle:   canvas.TextStyle{Color: th.Colors.TextDisabled, Size: th.Type.Body.Size},
		CursorColor: th.Colors.Accent,
		SelectBg:    canvas.Color{R: th.Colors.Accent.R, G: th.Colors.Accent.G, B: th.Colors.Accent.B, A: 0.25},
		LineHeight:  1.5,
		Padding:     canvas.All(8),
		ScrollbarW:  6,
	}
}

// MultiLineTextInput is a scrollable multi-line text editor.
type MultiLineTextInput struct {
	Text     string
	Hint     string
	Style    MultiLineStyle
	ReadOnly bool
	MaxLines int // 0 = unlimited
	OnChange func(string)
	OnSubmit func(string) // Shift+Enter

	lines      []string
	cursorLine int
	cursorCol  int
	scrollY    float32
	focused    bool
	blinkOff   bool
	blinkAccum float64
	lastChange time.Time

	bounds canvas.Rect
}

// NewMultiLineTextInput creates a MultiLineTextInput.
func NewMultiLineTextInput(hint string, style MultiLineStyle) *MultiLineTextInput {
	mt := &MultiLineTextInput{Hint: hint, Style: style}
	mt.syncLines()
	return mt
}

func (mt *MultiLineTextInput) syncLines() {
	mt.lines = strings.Split(mt.Text, "\n")
}

func (mt *MultiLineTextInput) lh() float32 {
	lhm := mt.Style.LineHeight
	if lhm == 0 {
		lhm = 1.5
	}
	return mt.Style.TextStyle.Size * lhm
}

func (mt *MultiLineTextInput) totalH() float32 {
	return float32(len(mt.lines)) * mt.lh()
}

func (mt *MultiLineTextInput) clampScroll(viewH float32) {
	max := mt.totalH() - viewH
	if max < 0 {
		max = 0
	}
	if mt.scrollY < 0 {
		mt.scrollY = 0
	}
	if mt.scrollY > max {
		mt.scrollY = max
	}
}

func (mt *MultiLineTextInput) ensureCursorVisible(viewH float32) {
	lh := mt.lh()
	curY := float32(mt.cursorLine) * lh
	if curY < mt.scrollY {
		mt.scrollY = curY
	}
	if curY+lh > mt.scrollY+viewH {
		mt.scrollY = curY + lh - viewH
	}
}

// ── Component interface ───────────────────────────────────────────────────────

func (mt *MultiLineTextInput) Bounds() canvas.Rect { return mt.bounds }

func (mt *MultiLineTextInput) Tick(delta float64) {
	mt.blinkAccum += delta
	if mt.blinkAccum >= 0.5 {
		mt.blinkOff = !mt.blinkOff
		mt.blinkAccum = 0
	}
}

func (mt *MultiLineTextInput) HandleEvent(e Event) bool {
	b := mt.bounds
	in := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	p := mt.Style.Padding
	viewH := b.H - p.Top - p.Bottom

	switch e.Type {
	case EventMouseDown:
		if in && e.Button == 1 {
			mt.focused = true
			lh := mt.lh()
			contentY := e.Y - b.Y + mt.scrollY - p.Top
			line := int(contentY / lh)
			if line < 0 {
				line = 0
			}
			if line >= len(mt.lines) {
				line = len(mt.lines) - 1
			}
			mt.cursorLine = line
			// Approximate column from x.
			avgW := mt.Style.TextStyle.Size * 0.6
			col := int((e.X - b.X - p.Left) / avgW)
			rc := utf8.RuneCountInString(mt.lines[line])
			if col < 0 {
				col = 0
			}
			if col > rc {
				col = rc
			}
			mt.cursorCol = col
			return true
		}
		mt.focused = false

	case EventScroll:
		if in {
			mt.scrollY -= e.ScrollY * mt.lh() * 3
			mt.clampScroll(viewH)
			return true
		}

	case EventKeyDown:
		if mt.focused {
			mt.handleKey(e, viewH)
			return true
		}
	}
	return false
}

func (mt *MultiLineTextInput) handleKey(e Event, viewH float32) {
	line := mt.lines[mt.cursorLine]
	runes := []rune(line)
	n := len(mt.lines)

	switch e.Key {
	case "Left":
		if mt.cursorCol > 0 {
			mt.cursorCol--
		} else if mt.cursorLine > 0 {
			mt.cursorLine--
			mt.cursorCol = utf8.RuneCountInString(mt.lines[mt.cursorLine])
		}
	case "Right":
		if mt.cursorCol < len(runes) {
			mt.cursorCol++
		} else if mt.cursorLine < n-1 {
			mt.cursorLine++
			mt.cursorCol = 0
		}
	case "Up":
		if mt.cursorLine > 0 {
			mt.cursorLine--
			mt.cursorCol = clampI(mt.cursorCol, 0, utf8.RuneCountInString(mt.lines[mt.cursorLine]))
		}
	case "Down":
		if mt.cursorLine < n-1 {
			mt.cursorLine++
			mt.cursorCol = clampI(mt.cursorCol, 0, utf8.RuneCountInString(mt.lines[mt.cursorLine]))
		}
	case "Home":
		mt.cursorCol = 0
	case "End":
		mt.cursorCol = utf8.RuneCountInString(line)
	case "BackSpace":
		if !mt.ReadOnly {
			if mt.cursorCol > 0 {
				r := append(runes[:mt.cursorCol-1:mt.cursorCol-1], runes[mt.cursorCol:]...)
				mt.lines[mt.cursorLine] = string(r)
				mt.cursorCol--
			} else if mt.cursorLine > 0 {
				prev := mt.lines[mt.cursorLine-1]
				mt.cursorCol = utf8.RuneCountInString(prev)
				mt.lines[mt.cursorLine-1] = prev + line
				mt.lines = append(mt.lines[:mt.cursorLine], mt.lines[mt.cursorLine+1:]...)
				mt.cursorLine--
			}
			mt.commit()
		}
	case "Delete":
		if !mt.ReadOnly {
			if mt.cursorCol < len(runes) {
				r := append(runes[:mt.cursorCol:mt.cursorCol], runes[mt.cursorCol+1:]...)
				mt.lines[mt.cursorLine] = string(r)
			} else if mt.cursorLine < len(mt.lines)-1 {
				mt.lines[mt.cursorLine] = line + mt.lines[mt.cursorLine+1]
				mt.lines = append(mt.lines[:mt.cursorLine+1], mt.lines[mt.cursorLine+2:]...)
			}
			mt.commit()
		}
	case "Return":
		if !mt.ReadOnly {
			if e.Shift {
				if mt.OnSubmit != nil {
					mt.OnSubmit(mt.Text)
				}
				return
			}
			if mt.MaxLines > 0 && len(mt.lines) >= mt.MaxLines {
				return
			}
			before := string(runes[:mt.cursorCol])
			after := string(runes[mt.cursorCol:])
			mt.lines[mt.cursorLine] = before
			rest := append([]string{after}, mt.lines[mt.cursorLine+1:]...)
			mt.lines = append(mt.lines[:mt.cursorLine+1], rest...)
			mt.cursorLine++
			mt.cursorCol = 0
			mt.commit()
		}
	case "Tab":
		if !mt.ReadOnly {
			mt.insert([]rune("    "))
		}
	default:
		if !mt.ReadOnly && len(e.Key) == 1 {
			r := []rune(e.Key)[0]
			if r >= 32 && r != 127 {
				mt.insert([]rune{r})
			}
		}
	}
	mt.ensureCursorVisible(viewH)
}

func (mt *MultiLineTextInput) insert(rs []rune) {
	runes := []rune(mt.lines[mt.cursorLine])
	updated := make([]rune, 0, len(runes)+len(rs))
	updated = append(updated, runes[:mt.cursorCol]...)
	updated = append(updated, rs...)
	updated = append(updated, runes[mt.cursorCol:]...)
	mt.lines[mt.cursorLine] = string(updated)
	mt.cursorCol += len(rs)
	mt.commit()
}

func (mt *MultiLineTextInput) commit() {
	mt.Text = strings.Join(mt.lines, "\n")
	mt.lastChange = time.Now()
	if mt.OnChange != nil {
		mt.OnChange(mt.Text)
	}
}

// ── Draw ──────────────────────────────────────────────────────────────────────

func (mt *MultiLineTextInput) Draw(c *canvas.Canvas, x, y, w, h float32) {
	mt.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()
	s := mt.Style

	// Background + border.
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.FillPaint(s.Background))
	bc := s.Border
	if mt.focused {
		bc = s.FocusBorder
	}
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.StrokePaint(bc, 1.5))

	p := s.Padding
	cx := x + p.Left
	cy := y + p.Top
	cw := w - p.Left - p.Right - s.ScrollbarW - 4
	ch := h - p.Top - p.Bottom
	mt.clampScroll(ch)

	c.Save()
	c.ClipRect(cx, cy, cw, ch)

	lh := mt.lh()
	ts := s.TextStyle

	if mt.Text == "" && mt.Hint != "" && !mt.focused {
		c.DrawText(cx, cy+s.HintStyle.Size, mt.Hint, s.HintStyle)
	} else {
		for i, line := range mt.lines {
			ly := cy + float32(i)*lh - mt.scrollY
			if ly+lh < cy || ly > cy+ch {
				continue
			}
			c.DrawText(cx, ly+ts.Size, line, ts)
		}
	}

	// Cursor (blink).
	if mt.focused && !mt.blinkOff {
		cursorLine := mt.lines[mt.cursorLine]
		prefix := string([]rune(cursorLine)[:mt.cursorCol])
		avgW := ts.Size * 0.6
		cursorX := cx + float32(utf8.RuneCountInString(prefix))*avgW
		cursorY := cy + float32(mt.cursorLine)*lh - mt.scrollY
		c.DrawRect(cursorX, cursorY, 2, lh, canvas.FillPaint(s.CursorColor))
	}
	c.Restore()

	// Scrollbar.
	totalH := mt.totalH()
	if totalH > ch {
		sbX := x + w - s.ScrollbarW - 2
		trackH := h - 8
		c.DrawRoundedRect(sbX, y+4, s.ScrollbarW, trackH, s.ScrollbarW/2,
			canvas.FillPaint(th.Colors.ScrollTrack))
		ratio := ch / totalH
		if ratio > 1 {
			ratio = 1
		}
		thumbH := trackH * ratio
		if thumbH < 20 {
			thumbH = 20
		}
		thumbY := y + 4
		if totalH-ch > 0 {
			thumbY += (mt.scrollY / (totalH - ch)) * (trackH - thumbH)
		}
		c.DrawRoundedRect(sbX, thumbY, s.ScrollbarW, thumbH, s.ScrollbarW/2,
			canvas.FillPaint(th.Colors.ScrollThumb))
	}
	_ = math.Pi // ensure math import used
}

// ═══════════════════════════════════════════════════════════════════════════════
// NumberInput
// ═══════════════════════════════════════════════════════════════════════════════

// NumberInput is a single-line numeric field with +/− steppers.
type NumberInput struct {
	Value    float64
	Min, Max float64
	Step     float64
	Decimals int
	Style    TextInputStyle
	OnChange func(float64)

	inner  *TextInput
	bounds canvas.Rect
}

// NewNumberInput creates a NumberInput.
func NewNumberInput(value, min, max, step float64, style TextInputStyle) *NumberInput {
	ni := &NumberInput{Value: value, Min: min, Max: max, Step: step, Decimals: 2, Style: style}
	if ni.Step == 0 {
		ni.Step = 1
	}
	ni.inner = NewTextInput("")
	ni.refreshText()
	ni.inner.OnChange = func(text string) {
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			ni.Value = ni.clamp(v)
			if ni.OnChange != nil {
				ni.OnChange(ni.Value)
			}
		}
	}
	return ni
}

func (ni *NumberInput) clamp(v float64) float64 {
	if ni.Min != ni.Max {
		if v < ni.Min {
			return ni.Min
		}
		if v > ni.Max {
			return ni.Max
		}
	}
	return v
}

func (ni *NumberInput) refreshText() {
	ni.inner.Text = fmt.Sprintf("%.*f", ni.Decimals, ni.Value)
}

func (ni *NumberInput) step(dir int) {
	ni.Value = ni.clamp(ni.Value + float64(dir)*ni.Step)
	ni.refreshText()
	if ni.OnChange != nil {
		ni.OnChange(ni.Value)
	}
}

func (ni *NumberInput) Bounds() canvas.Rect { return ni.bounds }
func (ni *NumberInput) Tick(d float64)      { ni.inner.Tick(d) }

func (ni *NumberInput) HandleEvent(e Event) bool {
	b := ni.bounds
	btnW := float32(28)
	// Minus hit-test.
	if e.Type == EventMouseDown && e.Button == 1 {
		if e.X >= b.X && e.X <= b.X+btnW && e.Y >= b.Y && e.Y <= b.Y+b.H {
			ni.step(-1)
			return true
		}
		plusX := b.X + b.W - btnW
		if e.X >= plusX && e.X <= plusX+btnW && e.Y >= b.Y && e.Y <= b.Y+b.H {
			ni.step(1)
			return true
		}
	}
	return ni.inner.HandleEvent(e)
}

func (ni *NumberInput) Draw(c *canvas.Canvas, x, y, w, h float32) {
	ni.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()
	btnW := float32(28)

	// − button.
	c.DrawRoundedRect(x, y, btnW, h, th.Radius.SM, canvas.FillPaint(th.Colors.BgBase))
	c.DrawRoundedRect(x, y, btnW, h, th.Radius.SM, canvas.StrokePaint(th.Colors.Border, 1))
	c.DrawCenteredText(canvas.Rect{X: x, Y: y, W: btnW, H: h}, "−",
		canvas.TextStyle{Color: th.Colors.TextPrimary, Size: 16})

	// Text field.
	ni.inner.Draw(c, x+btnW+1, y, w-btnW*2-2, h)

	// + button.
	px := x + w - btnW
	c.DrawRoundedRect(px, y, btnW, h, th.Radius.SM, canvas.FillPaint(th.Colors.BgBase))
	c.DrawRoundedRect(px, y, btnW, h, th.Radius.SM, canvas.StrokePaint(th.Colors.Border, 1))
	c.DrawCenteredText(canvas.Rect{X: px, Y: y, W: btnW, H: h}, "+",
		canvas.TextStyle{Color: th.Colors.TextPrimary, Size: 16})
}

// ═══════════════════════════════════════════════════════════════════════════════
// SearchInput
// ═══════════════════════════════════════════════════════════════════════════════

// SearchInput wraps TextInput with a search icon, a clear button, and debounced OnSearch.
type SearchInput struct {
	Hint     string
	Style    TextInputStyle
	Debounce time.Duration // default 300 ms
	OnSearch func(query string)

	inner   *TextInput
	timer   *time.Timer
	last    string
	bounds  canvas.Rect
}

// NewSearchInput creates a SearchInput.
func NewSearchInput(hint string, style TextInputStyle) *SearchInput {
	si := &SearchInput{Hint: hint, Style: style, Debounce: 300 * time.Millisecond}
	si.inner = NewTextInput(hint)
	si.inner.OnChange = func(text string) {
		if si.timer != nil {
			si.timer.Stop()
		}
		si.timer = time.AfterFunc(si.Debounce, func() {
			if si.OnSearch != nil && text != si.last {
				si.last = text
				si.OnSearch(text)
			}
		})
	}
	return si
}

// Value returns the current search text.
func (si *SearchInput) Value() string { return si.inner.Text }

// Clear empties the field and fires OnSearch("").
func (si *SearchInput) Clear() {
	si.inner.Text = ""
	si.last = ""
	if si.OnSearch != nil {
		si.OnSearch("")
	}
}

func (si *SearchInput) Bounds() canvas.Rect { return si.bounds }
func (si *SearchInput) Tick(d float64)      { si.inner.Tick(d) }

func (si *SearchInput) HandleEvent(e Event) bool {
	b := si.bounds
	iconW := float32(28)
	clearW := float32(28)
	// Clear button click.
	if e.Type == EventMouseDown && e.Button == 1 && si.inner.Text != "" {
		cx := b.X + b.W - clearW
		if e.X >= cx && e.X <= cx+clearW && e.Y >= b.Y && e.Y <= b.Y+b.H {
			si.Clear()
			return true
		}
	}
	// Redirect to inner input (adjusting coordinate is unnecessary — inner
	// tracks its own bounds from Draw).
	_ = iconW
	return si.inner.HandleEvent(e)
}

func (si *SearchInput) Draw(c *canvas.Canvas, x, y, w, h float32) {
	si.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()
	iconW := float32(28)
	clearW := float32(28)

	// Search icon.
	c.DrawCenteredText(canvas.Rect{X: x, Y: y, W: iconW, H: h}, "⌕",
		canvas.TextStyle{Color: th.Colors.TextSecondary, Size: 16})

	// Text field.
	fieldW := w - iconW
	if si.inner.Text != "" {
		fieldW -= clearW
	}
	si.inner.Draw(c, x+iconW, y, fieldW, h)

	// Clear button.
	if si.inner.Text != "" {
		cx := x + w - clearW
		c.DrawCenteredText(canvas.Rect{X: cx, Y: y, W: clearW, H: h}, "✕",
			canvas.TextStyle{Color: th.Colors.TextSecondary, Size: 12})
	}
}

