// Package ui — widgets.go
// Additional high-level widgets for goui:
//
//   - Label          — text that auto-wraps, clips, or scales to fit
//   - Badge          — a small pill/count indicator
//   - Tooltip        — floating text shown on hover
//   - Divider        — horizontal or vertical separator line
//   - ProgressBar    — fill-progress indicator
//   - Toggle         — on/off switch
//   - Card           — rounded, shadowed container
//   - IconLabel      — icon + text side-by-side
package ui

import (
	"fmt"

	"github.com/achiket123/gui-go/canvas"
)

// ────────────────────────────────────────────────────────────────────────────
// Label
// ────────────────────────────────────────────────────────────────────────────

// LabelStyle configures a Label widget.
type LabelStyle struct {
	TextBox    canvas.TextBoxStyle // wrapping, align, overflow, etc.
	Background canvas.Color        // zero = transparent
}

// DefaultLabelStyle returns a minimal left-aligned label style.
func DefaultLabelStyle() LabelStyle {
	ts := canvas.DefaultTextBoxStyle()
	ts.Align = canvas.TextAlignLeft
	ts.Overflow = canvas.TextOverflowEllipsis
	return LabelStyle{TextBox: ts}
}

// Label renders text inside its allocated bounds with proper wrapping and clipping.
// Resize the window — Label reflows automatically.
type Label struct {
	Text   string
	Style  LabelStyle
	bounds canvas.Rect
}

func NewLabel(text string, style LabelStyle) *Label {
	return &Label{Text: text, Style: style}
}

func (l *Label) Bounds() canvas.Rect      { return l.bounds }
func (l *Label) Tick(_ float64)           {}
func (l *Label) HandleEvent(_ Event) bool { return false }

func (l *Label) Draw(c *canvas.Canvas, x, y, w, h float32) {
	l.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	// Draw background if set.
	if l.Style.Background != (canvas.Color{}) {
		c.DrawRect(x, y, w, h, canvas.FillPaint(l.Style.Background))
	}
	c.DrawTextBox(l.bounds, l.Text, l.Style.TextBox)
}

// ────────────────────────────────────────────────────────────────────────────
// Badge
// ────────────────────────────────────────────────────────────────────────────

// BadgeStyle configures a Badge.
type BadgeStyle struct {
	Background canvas.Color
	TextStyle  canvas.TextStyle
	Radius     float32
	MinWidth   float32
	Padding    canvas.EdgeInsets
}

// DefaultBadgeStyle returns a red, round badge.
func DefaultBadgeStyle() BadgeStyle {
	return BadgeStyle{
		Background: canvas.Hex("#F38BA8"),
		TextStyle:  canvas.TextStyle{Color: canvas.White, Size: 11},
		Radius:     10,
		MinWidth:   20,
		Padding:    canvas.Symmetric(6, 2),
	}
}

// Badge displays a short count or label in a pill-shaped bubble.
type Badge struct {
	Value  int    // if > 0, shown as number; else Text is used
	Text   string // used when Value == 0
	Style  BadgeStyle
	bounds canvas.Rect
}

func NewBadge(value int, style BadgeStyle) *Badge {
	return &Badge{Value: value, Style: style}
}

func (b *Badge) Bounds() canvas.Rect      { return b.bounds }
func (b *Badge) Tick(_ float64)           {}
func (b *Badge) HandleEvent(_ Event) bool { return false }

func (b *Badge) Draw(c *canvas.Canvas, x, y, w, h float32) {
	b.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	label := b.Text
	if b.Value > 0 {
		label = fmt.Sprintf("%d", b.Value)
	}
	ts := b.Style.TextStyle
	sz := c.MeasureText(label, ts)
	bw := sz.W + b.Style.Padding.Left + b.Style.Padding.Right
	if bw < b.Style.MinWidth {
		bw = b.Style.MinWidth
	}
	bh := ts.Size + b.Style.Padding.Top + b.Style.Padding.Bottom

	bx := x + (w-bw)/2
	by := y + (h-bh)/2
	c.DrawRoundedRect(bx, by, bw, bh, b.Style.Radius, canvas.FillPaint(b.Style.Background))
	// Center text inside pill.
	tx := bx + (bw-sz.W)/2
	ty := by + b.Style.Padding.Top + ts.Size
	c.DrawText(tx, ty, label, ts)
	b.bounds = canvas.Rect{X: bx, Y: by, W: bw, H: bh}
}

// ────────────────────────────────────────────────────────────────────────────
// Divider
// ────────────────────────────────────────────────────────────────────────────

// Divider draws a thin horizontal or vertical separator.
type Divider struct {
	Vertical  bool
	Color     canvas.Color
	Thickness float32
	bounds    canvas.Rect
}

func NewDivider(vertical bool) *Divider {
	return &Divider{Vertical: vertical, Color: canvas.Hex("#313244"), Thickness: 1}
}

func (d *Divider) Bounds() canvas.Rect      { return d.bounds }
func (d *Divider) Tick(_ float64)           {}
func (d *Divider) HandleEvent(_ Event) bool { return false }

func (d *Divider) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	p := canvas.FillPaint(d.Color)
	t := d.Thickness
	if t <= 0 {
		t = 1
	}
	if d.Vertical {
		c.DrawRect(x+(w-t)/2, y, t, h, p)
	} else {
		c.DrawRect(x, y+(h-t)/2, w, t, p)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ProgressBar
// ────────────────────────────────────────────────────────────────────────────

// ProgressBarStyle configures a ProgressBar.
type ProgressBarStyle struct {
	TrackColor canvas.Color
	FillColor  canvas.Color
	Radius     float32
	Height     float32
}

func DefaultProgressBarStyle() ProgressBarStyle {
	return ProgressBarStyle{
		TrackColor: canvas.Hex("#313244"),
		FillColor:  canvas.Hex("#89B4FA"),
		Radius:     4,
		Height:     8,
	}
}

// ProgressBar displays a horizontal fill indicator.
// Value is in [0, 1].
type ProgressBar struct {
	Value  float32 // 0.0 – 1.0
	Style  ProgressBarStyle
	bounds canvas.Rect
}

func NewProgressBar(value float32, style ProgressBarStyle) *ProgressBar {
	return &ProgressBar{Value: value, Style: style}
}

func (p *ProgressBar) Bounds() canvas.Rect      { return p.bounds }
func (p *ProgressBar) Tick(_ float64)           {}
func (p *ProgressBar) HandleEvent(_ Event) bool { return false }

func (p *ProgressBar) Draw(c *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	bh := p.Style.Height
	if bh <= 0 {
		bh = h
	}
	by := y + (h-bh)/2
	r := p.Style.Radius
	c.DrawRoundedRect(x, by, w, bh, r, canvas.FillPaint(p.Style.TrackColor))
	v := p.Value
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	if v > 0 {
		c.DrawRoundedRect(x, by, w*v, bh, r, canvas.FillPaint(p.Style.FillColor))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Toggle (on/off switch)
// ────────────────────────────────────────────────────────────────────────────

// ToggleStyle configures a Toggle switch.
type ToggleStyle struct {
	TrackOn  canvas.Color
	TrackOff canvas.Color
	Thumb    canvas.Color
	Width    float32
	Height   float32
	Radius   float32
}

func DefaultToggleStyle() ToggleStyle {
	return ToggleStyle{
		TrackOn:  canvas.Hex("#89B4FA"),
		TrackOff: canvas.Hex("#313244"),
		Thumb:    canvas.White,
		Width:    44,
		Height:   24,
		Radius:   12,
	}
}

// Toggle is a binary on/off switch widget.
type Toggle struct {
	Value    bool
	Style    ToggleStyle
	OnChange func(bool)
	hovered  bool
	bounds   canvas.Rect
}

func NewToggle(value bool, style ToggleStyle, onChange func(bool)) *Toggle {
	return &Toggle{Value: value, Style: style, OnChange: onChange}
}

func (t *Toggle) Bounds() canvas.Rect { return t.bounds }
func (t *Toggle) Tick(_ float64)      {}

func (t *Toggle) HandleEvent(e Event) bool {
	b := t.bounds
	inBounds := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	switch e.Type {
	case EventMouseMove:
		t.hovered = inBounds
	case EventMouseDown:
		if inBounds && e.Button == 1 {
			t.Value = !t.Value
			if t.OnChange != nil {
				t.OnChange(t.Value)
			}
			return true
		}
	}
	return false
}

func (t *Toggle) Draw(c *canvas.Canvas, x, y, w, h float32) {
	tw := t.Style.Width
	th := t.Style.Height
	if tw <= 0 {
		tw = w
	}
	if th <= 0 {
		th = h
	}
	tx := x + (w-tw)/2
	ty := y + (h-th)/2
	t.bounds = canvas.Rect{X: tx, Y: ty, W: tw, H: th}

	trackColor := t.Style.TrackOff
	if t.Value {
		trackColor = t.Style.TrackOn
	}
	c.DrawRoundedRect(tx, ty, tw, th, t.Style.Radius, canvas.FillPaint(trackColor))

	// Thumb
	pad := float32(3)
	thumbD := th - pad*2
	thumbX := tx + pad
	if t.Value {
		thumbX = tx + tw - thumbD - pad
	}
	c.DrawCircle(thumbX+thumbD/2, ty+th/2, thumbD/2, canvas.FillPaint(t.Style.Thumb))
}

// ────────────────────────────────────────────────────────────────────────────
// Card
// ────────────────────────────────────────────────────────────────────────────

// CardStyle configures a Card container.
type CardStyle struct {
	Background  canvas.Color
	BorderColor canvas.Color
	BorderWidth float32
	Radius      float32
	Padding     canvas.EdgeInsets
	ShadowAlpha float32 // 0 = no shadow, 1 = full black shadow
	ShadowBlur  float32
}

func DefaultCardStyle() CardStyle {
	return CardStyle{
		Background:  canvas.Hex("#1E1E2E"),
		BorderColor: canvas.Hex("#313244"),
		BorderWidth: 1,
		Radius:      12,
		Padding:     canvas.All(16),
		ShadowAlpha: 0.3,
		ShadowBlur:  8,
	}
}

// Card is a rounded, optionally bordered container that delegates drawing
// to its child.
type Card struct {
	Style  CardStyle
	Child  Component
	bounds canvas.Rect
}

func NewCard(style CardStyle, child Component) *Card {
	return &Card{Style: style, Child: child}
}

func (card *Card) Bounds() canvas.Rect { return card.bounds }
func (card *Card) Tick(delta float64) {
	if card.Child != nil {
		card.Child.Tick(delta)
	}
}
func (card *Card) HandleEvent(e Event) bool {
	if card.Child != nil {
		return card.Child.HandleEvent(e)
	}
	return false
}

func (card *Card) Draw(c *canvas.Canvas, x, y, w, h float32) {
	card.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	s := card.Style

	// Approximate drop shadow: a slightly offset, alpha-reduced background rect.
	if s.ShadowAlpha > 0 {
		blur := s.ShadowBlur
		if blur <= 0 {
			blur = 4
		}
		for i := float32(1); i <= blur; i++ {
			alpha := s.ShadowAlpha * (1 - i/blur) * 0.4
			sc := canvas.Color{A: alpha}
			c.DrawRoundedRect(x+i, y+i, w, h, s.Radius, canvas.FillPaint(sc))
		}
	}

	// Background fill.
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.FillPaint(s.Background))

	// Border.
	if s.BorderWidth > 0 {
		c.DrawRoundedRect(x, y, w, h, s.Radius,
			canvas.StrokePaint(s.BorderColor, s.BorderWidth))
	}

	// Child with padding.
	if card.Child != nil {
		p := s.Padding
		card.Child.Draw(c,
			x+p.Left, y+p.Top,
			w-p.Left-p.Right, h-p.Top-p.Bottom,
		)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Tooltip
// ────────────────────────────────────────────────────────────────────────────

// Tooltip wraps a Component and shows a floating text hint when hovered.
// The tooltip is rendered at the end of Draw so it appears above other widgets.
type Tooltip struct {
	Text       string
	Child      Component
	TextStyle  canvas.TextStyle
	Background canvas.Color
	Radius     float32
	Padding    canvas.EdgeInsets

	hovered bool
	mouseX  float32
	mouseY  float32
	bounds  canvas.Rect
}

func NewTooltip(text string, child Component) *Tooltip {
	return &Tooltip{
		Text:       text,
		Child:      child,
		TextStyle:  canvas.TextStyle{Color: canvas.White, Size: 12},
		Background: canvas.Hex("#181825"),
		Radius:     6,
		Padding:    canvas.Symmetric(8, 4),
	}
}

func (t *Tooltip) Bounds() canvas.Rect { return t.bounds }
func (t *Tooltip) Tick(delta float64) {
	if t.Child != nil {
		t.Child.Tick(delta)
	}
}

func (t *Tooltip) HandleEvent(e Event) bool {
	b := t.bounds
	inBounds := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	if e.Type == EventMouseMove {
		t.hovered = inBounds
		t.mouseX = e.X
		t.mouseY = e.Y
	}
	if t.Child != nil {
		return t.Child.HandleEvent(e)
	}
	return false
}

func (t *Tooltip) Draw(c *canvas.Canvas, x, y, w, h float32) {
	t.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if t.Child != nil {
		t.Child.Draw(c, x, y, w, h)
	}
	if !t.hovered || t.Text == "" {
		return
	}
	// Measure tooltip.
	ts := t.TextStyle
	sz := c.MeasureText(t.Text, ts)
	tw := sz.W + t.Padding.Left + t.Padding.Right
	th := ts.Size + t.Padding.Top + t.Padding.Bottom
	// Position above cursor.
	tx := t.mouseX - tw/2
	ty := t.mouseY - th - 8
	// Clamp to canvas edges.
	cw := c.Width()
	ch := c.Height()
	if tx < 0 {
		tx = 0
	}
	if tx+tw > cw {
		tx = cw - tw
	}
	if ty < 0 {
		ty = t.mouseY + 16
	}
	if ty+th > ch {
		ty = t.mouseY - th - 4
	}
	c.DrawRoundedRect(tx, ty, tw, th, t.Radius, canvas.FillPaint(t.Background))
	c.DrawText(tx+t.Padding.Left, ty+t.Padding.Top+ts.Size, t.Text, ts)
}
