// Package ui — layout.go
// Provides high-level layout widgets: Center, AspectRatio, Flex (row/column),
// Expanded, Padding, SizedBox, and Align.
//
// All widgets implement Component and respond correctly to window resize events.
package ui

import (
	"github.com/achiket/gui-go/canvas"
)

// ────────────────────────────────────────────────────────────────────────────
// Align constants
// ────────────────────────────────────────────────────────────────────────────

// Alignment describes how a child widget is positioned inside its parent's bounds.
type Alignment int

const (
	AlignmentCenter       Alignment = iota // both axes centered
	AlignmentTopLeft                       // top-left corner
	AlignmentTopCenter                     // top edge, horizontally centered
	AlignmentTopRight                      // top edge, right-aligned
	AlignmentCenterLeft                    // vertically centered, left-aligned
	AlignmentCenterRight                   // vertically centered, right-aligned
	AlignmentBottomLeft                    // bottom-left corner
	AlignmentBottomCenter                  // bottom edge, horizontally centered
	AlignmentBottomRight                   // bottom-right corner
)

// alignOffset returns the (x, y) offset within a container of (cw, ch)
// for a child of (childW, childH) given the alignment.
func alignOffset(cw, ch, childW, childH float32, a Alignment) (float32, float32) {
	var ox, oy float32
	switch a {
	case AlignmentTopLeft:
		ox, oy = 0, 0
	case AlignmentTopCenter:
		ox, oy = (cw-childW)/2, 0
	case AlignmentTopRight:
		ox, oy = cw-childW, 0
	case AlignmentCenterLeft:
		ox, oy = 0, (ch-childH)/2
	case AlignmentCenter:
		ox, oy = (cw-childW)/2, (ch-childH)/2
	case AlignmentCenterRight:
		ox, oy = cw-childW, (ch-childH)/2
	case AlignmentBottomLeft:
		ox, oy = 0, ch-childH
	case AlignmentBottomCenter:
		ox, oy = (cw-childW)/2, ch-childH
	case AlignmentBottomRight:
		ox, oy = cw-childW, ch-childH
	}
	return ox, oy
}

// ────────────────────────────────────────────────────────────────────────────
// Center widget
// ────────────────────────────────────────────────────────────────────────────

// Center positions a child Component at the centre of its allocated bounds.
// It delegates all event handling and drawing to the child.
//
// Example:
//
//	btn := ui.NewButton("OK", nil)
//	centered := ui.NewCenter(btn)
//	window.Register(centered)
type Center struct {
	Child     Component
	Alignment Alignment // defaults to AlignmentCenter; override for fine-grained control
	bounds    canvas.Rect
}

// NewCenter wraps child and centres it within the parent's bounds.
func NewCenter(child Component) *Center {
	return &Center{Child: child, Alignment: AlignmentCenter}
}

func (c *Center) Bounds() canvas.Rect { return c.bounds }
func (c *Center) Tick(delta float64) {
	if c.Child != nil {
		c.Child.Tick(delta)
	}
}

func (c *Center) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	c.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if c.Child == nil {
		return
	}
	// The child reports its own preferred size via Bounds(); if not yet laid out
	// we give it the full area first, then re-draw at the centred position.
	c.Child.Draw(cv, x, y, w, h)
	cb := c.Child.Bounds()
	ox, oy := alignOffset(w, h, cb.W, cb.H, c.Alignment)
	// Only re-draw if the child actually has a smaller size than the container.
	if ox != 0 || oy != 0 {
		c.Child.Draw(cv, x+ox, y+oy, cb.W, cb.H)
	}
}

func (c *Center) HandleEvent(e Event) bool {
	if c.Child != nil {
		return c.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// AspectRatio widget
// ────────────────────────────────────────────────────────────────────────────

// AspectRatio constrains its child to a fixed width-to-height ratio.
// When the window is resized the child scales uniformly, never stretching.
//
//	img := ui.NewAspectRatio(16.0/9.0, myImageWidget)
type AspectRatio struct {
	Ratio     float32 // width / height, e.g. 16.0/9.0 or 1.0 for square
	Child     Component
	Alignment Alignment // where to place the child inside its allocated space
	bounds    canvas.Rect
}

// NewAspectRatio creates an AspectRatio layout with the given ratio and child.
func NewAspectRatio(ratio float32, child Component) *AspectRatio {
	return &AspectRatio{Ratio: ratio, Child: child, Alignment: AlignmentCenter}
}

func (a *AspectRatio) Bounds() canvas.Rect { return a.bounds }
func (a *AspectRatio) Tick(delta float64) {
	if a.Child != nil {
		a.Child.Tick(delta)
	}
}

// constrainedSize calculates the largest child rect that fits inside (w, h)
// while maintaining a.Ratio.
func (a *AspectRatio) constrainedSize(w, h float32) (cw, ch float32) {
	if a.Ratio <= 0 {
		return w, h
	}
	if w/h > a.Ratio {
		ch = h
		cw = h * a.Ratio
	} else {
		cw = w
		ch = w / a.Ratio
	}
	return
}

func (a *AspectRatio) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	a.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if a.Child == nil {
		return
	}
	cw, ch := a.constrainedSize(w, h)
	ox, oy := alignOffset(w, h, cw, ch, a.Alignment)
	a.Child.Draw(cv, x+ox, y+oy, cw, ch)
}

func (a *AspectRatio) HandleEvent(e Event) bool {
	if a.Child != nil {
		return a.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// Padding widget
// ────────────────────────────────────────────────────────────────────────────

// Padding insets its child by fixed pixel amounts on each side.
//
//	padded := ui.NewPadding(canvas.All(16), myWidget)
type Padding struct {
	Insets canvas.EdgeInsets
	Child  Component
	bounds canvas.Rect
}

// NewPadding creates a Padding layout.
func NewPadding(insets canvas.EdgeInsets, child Component) *Padding {
	return &Padding{Insets: insets, Child: child}
}

func (p *Padding) Bounds() canvas.Rect { return p.bounds }
func (p *Padding) Tick(delta float64) {
	if p.Child != nil {
		p.Child.Tick(delta)
	}
}
func (p *Padding) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if p.Child == nil {
		return
	}
	ix := x + p.Insets.Left
	iy := y + p.Insets.Top
	iw := w - p.Insets.Left - p.Insets.Right
	ih := h - p.Insets.Top - p.Insets.Bottom
	if iw < 0 {
		iw = 0
	}
	if ih < 0 {
		ih = 0
	}
	p.Child.Draw(cv, ix, iy, iw, ih)
}
func (p *Padding) HandleEvent(e Event) bool {
	if p.Child != nil {
		return p.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// SizedBox widget
// ────────────────────────────────────────────────────────────────────────────

// SizedBox gives its child a fixed width and/or height, regardless of the
// parent's allocation. Set W or H to 0 to use the parent's allocated dimension.
//
//	box := ui.NewSizedBox(200, 48, myWidget) // fixed 200×48
type SizedBox struct {
	W, H   float32 // 0 = use allocated dimension
	Child  Component
	bounds canvas.Rect
}

func NewSizedBox(w, h float32, child Component) *SizedBox {
	return &SizedBox{W: w, H: h, Child: child}
}

func (s *SizedBox) Bounds() canvas.Rect { return s.bounds }
func (s *SizedBox) Tick(delta float64) {
	if s.Child != nil {
		s.Child.Tick(delta)
	}
}
func (s *SizedBox) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	fw, fh := w, h
	if s.W > 0 {
		fw = s.W
	}
	if s.H > 0 {
		fh = s.H
	}
	s.bounds = canvas.Rect{X: x, Y: y, W: fw, H: fh}
	if s.Child != nil {
		s.Child.Draw(cv, x, y, fw, fh)
	}
}
func (s *SizedBox) HandleEvent(e Event) bool {
	if s.Child != nil {
		return s.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// FlexItem — a single item in a Flex layout
// ────────────────────────────────────────────────────────────────────────────

// FlexItem wraps a Component with flex layout properties.
type FlexItem struct {
	Child   Component
	Flex    float32   // grow factor; 0 = fixed (use MinSize)
	MinSize float32   // minimum main-axis size in pixels (for fixed items)
	Align   Alignment // cross-axis alignment override; AlignmentCenter = use Flex default
}

// Fixed returns a FlexItem with a fixed pixel size and no grow.
func FixedItem(size float32, child Component) FlexItem {
	return FlexItem{Child: child, MinSize: size}
}

// Flexible returns a FlexItem that grows to fill remaining space.
func Flexible(flex float32, child Component) FlexItem {
	return FlexItem{Child: child, Flex: flex}
}

// ────────────────────────────────────────────────────────────────────────────
// Flex layout (Row / Column)
// ────────────────────────────────────────────────────────────────────────────

// FlexDirection controls the main axis of a Flex layout.
type FlexDirection int

const (
	FlexRow    FlexDirection = iota // children laid out left → right
	FlexColumn                      // children laid out top → bottom
)

// Flex lays out children along a main axis with optional flexing.
// Fixed items use their MinSize on the main axis; Flexible items share the
// remaining space proportionally according to their Flex factor.
//
// Example (horizontal toolbar):
//
//	row := ui.NewFlex(ui.FlexRow,
//	    ui.FixedItem(120, logo),
//	    ui.Flexible(1, spacer),
//	    ui.FixedItem(80, btn),
//	)
type Flex struct {
	Direction      FlexDirection
	CrossAlignment Alignment // how to align children on the cross axis
	Gap            float32   // pixel gap between children
	Items          []FlexItem
	bounds         canvas.Rect
}

// NewFlex creates a Flex layout with the given direction and items.
func NewFlex(dir FlexDirection, items ...FlexItem) *Flex {
	return &Flex{
		Direction:      dir,
		CrossAlignment: AlignmentCenter,
		Items:          items,
	}
}

func (f *Flex) Bounds() canvas.Rect { return f.bounds }
func (f *Flex) Tick(delta float64) {
	for _, item := range f.Items {
		if item.Child != nil {
			item.Child.Tick(delta)
		}
	}
}

func (f *Flex) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	f.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	n := len(f.Items)
	if n == 0 {
		return
	}

	isRow := f.Direction == FlexRow
	totalGap := f.Gap * float32(n-1)

	// Main-axis total available space.
	mainTotal := w
	if !isRow {
		mainTotal = h
	}
	mainTotal -= totalGap

	// First pass: sum fixed sizes and flex factors.
	fixedSum := float32(0)
	flexSum := float32(0)
	for _, item := range f.Items {
		if item.Flex > 0 {
			flexSum += item.Flex
		} else {
			fixedSum += item.MinSize
		}
	}

	// Space available for flex items.
	flexSpace := mainTotal - fixedSum
	if flexSpace < 0 {
		flexSpace = 0
	}

	// Second pass: draw each item.
	cursor := float32(0)
	for i, item := range f.Items {
		if item.Child == nil {
			continue
		}

		// Compute main-axis size for this item.
		var mainSize float32
		if item.Flex > 0 && flexSum > 0 {
			mainSize = flexSpace * (item.Flex / flexSum)
		} else {
			mainSize = item.MinSize
		}

		// Compute cross-axis positioning.
		crossSize := h
		if !isRow {
			crossSize = w
		}

		align := item.Align
		if align == AlignmentCenter {
			align = f.CrossAlignment
		}

		var childX, childY, childW, childH float32
		if isRow {
			childW = mainSize
			childH = h
			childX = x + cursor
			childY = y
			// Cross-axis (vertical) alignment.
			childW2, childH2 := childW, crossSize
			_, oy := alignOffset(childW, crossSize, childW2, childH2, align)
			childY = y + oy
			childH = childH2
		} else {
			childW = w
			childH = mainSize
			childX = x
			childY = y + cursor
			// Cross-axis (horizontal) alignment.
			childW2, childH2 := crossSize, childH
			ox, _ := alignOffset(crossSize, childH, childW2, childH2, align)
			childX = x + ox
			childW = childW2
		}

		item.Child.Draw(cv, childX, childY, childW, childH)

		cursor += mainSize
		if i < n-1 {
			cursor += f.Gap
		}
	}
}

func (f *Flex) HandleEvent(e Event) bool {
	// Dispatch in reverse paint order (top-most first).
	for i := len(f.Items) - 1; i >= 0; i-- {
		if f.Items[i].Child != nil && f.Items[i].Child.HandleEvent(e) {
			return true
		}
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// Stack widget (z-order layering)
// ────────────────────────────────────────────────────────────────────────────

// Stack draws its children in order, each receiving the full allocated bounds.
// Later children appear on top. Useful for overlaying elements (e.g. a
// background + foreground content + HUD).
//
//	stack := ui.NewStack(background, content, tooltip)
type Stack struct {
	Children []Component
	bounds   canvas.Rect
}

func NewStack(children ...Component) *Stack {
	return &Stack{Children: children}
}

func (s *Stack) Bounds() canvas.Rect { return s.bounds }
func (s *Stack) Tick(delta float64) {
	for _, c := range s.Children {
		if c != nil {
			c.Tick(delta)
		}
	}
}
func (s *Stack) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	for _, c := range s.Children {
		if c != nil {
			c.Draw(cv, x, y, w, h)
		}
	}
}
func (s *Stack) HandleEvent(e Event) bool {
	for i := len(s.Children) - 1; i >= 0; i-- {
		if s.Children[i] != nil && s.Children[i].HandleEvent(e) {
			return true
		}
	}
	return false
}
