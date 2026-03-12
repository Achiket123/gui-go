// Package ui — responsive.go
// Responsive layout helpers: breakpoints, percentage-based sizing,
// min/max constraints, and a root ResponsiveLayout that reacts to resize events.
//
// Usage:
//
//	root := ui.NewResponsiveLayout(800, 600)
//	root.OnBreakpoint(480, smallLayout)
//	root.OnBreakpoint(1024, mediumLayout)
//	root.Default(largeLayout)
//	window.Register(root)
//
// Whenever the window fires a resize, root.Resize(w, h) propagates the new
// dimensions so every child re-lays-out correctly.
package ui

import (
	"sort"

	"github.com/achiket123/gui-go/canvas"
)

// ────────────────────────────────────────────────────────────────────────────
// ResponsiveLayout — breakpoint-based root layout
// ────────────────────────────────────────────────────────────────────────────

type breakpointEntry struct {
	maxWidth float32
	child    Component
}

// ResponsiveLayout switches between different Component trees depending on the
// current window width.  It implements Component and must be registered with
// the Window.
type ResponsiveLayout struct {
	breakpoints []breakpointEntry // sorted ascending by maxWidth
	defaultComp Component
	w, h        float32
	active      Component
	bounds      canvas.Rect
}

// NewResponsiveLayout creates a ResponsiveLayout initialised to w×h.
func NewResponsiveLayout(w, h float32) *ResponsiveLayout {
	return &ResponsiveLayout{w: w, h: h}
}

// OnBreakpoint registers a child layout that is active when window width ≤ maxWidth.
// Call in ascending order of maxWidth; OnBreakpoint sorts automatically.
func (r *ResponsiveLayout) OnBreakpoint(maxWidth float32, child Component) {
	r.breakpoints = append(r.breakpoints, breakpointEntry{maxWidth, child})
	sort.Slice(r.breakpoints, func(i, j int) bool {
		return r.breakpoints[i].maxWidth < r.breakpoints[j].maxWidth
	})
	r.updateActive()
}

// Default sets the layout used when no breakpoint matches.
func (r *ResponsiveLayout) Default(child Component) {
	r.defaultComp = child
	r.updateActive()
}

// Resize must be called when the window is resized (from the OnResize callback).
func (r *ResponsiveLayout) Resize(w, h float32) {
	r.w, r.h = w, h
	r.updateActive()
}

func (r *ResponsiveLayout) updateActive() {
	for _, bp := range r.breakpoints {
		if r.w <= bp.maxWidth {
			r.active = bp.child
			return
		}
	}
	r.active = r.defaultComp
}

func (r *ResponsiveLayout) Bounds() canvas.Rect { return r.bounds }
func (r *ResponsiveLayout) Tick(delta float64) {
	if r.active != nil {
		r.active.Tick(delta)
	}
}
func (r *ResponsiveLayout) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	r.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	r.w, r.h = w, h
	r.updateActive()
	if r.active != nil {
		r.active.Draw(cv, x, y, w, h)
	}
}
func (r *ResponsiveLayout) HandleEvent(e Event) bool {
	if r.active != nil {
		return r.active.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// PercentBox — percentage-based sizing
// ────────────────────────────────────────────────────────────────────────────

// PercentBox allocates a fraction of the parent's bounds to its child.
// WidthPct and HeightPct are in [0, 1]; 0 means "use the full allocated size".
//
//	half := ui.NewPercentBox(0.5, 1.0, myWidget) // half-width, full height
type PercentBox struct {
	WidthPct  float32
	HeightPct float32
	Align     Alignment
	Child     Component
	bounds    canvas.Rect
}

func NewPercentBox(wPct, hPct float32, child Component) *PercentBox {
	return &PercentBox{WidthPct: wPct, HeightPct: hPct, Align: AlignmentCenter, Child: child}
}

func (p *PercentBox) Bounds() canvas.Rect { return p.bounds }
func (p *PercentBox) Tick(delta float64) {
	if p.Child != nil {
		p.Child.Tick(delta)
	}
}
func (p *PercentBox) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	p.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	cw := w
	if p.WidthPct > 0 {
		cw = w * p.WidthPct
	}
	ch := h
	if p.HeightPct > 0 {
		ch = h * p.HeightPct
	}
	ox, oy := alignOffset(w, h, cw, ch, p.Align)
	if p.Child != nil {
		p.Child.Draw(cv, x+ox, y+oy, cw, ch)
	}
}
func (p *PercentBox) HandleEvent(e Event) bool {
	if p.Child != nil {
		return p.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// Constrained — enforces min/max width and height constraints
// ────────────────────────────────────────────────────────────────────────────

// Constrained clamps its child's bounds between minimum and maximum pixel sizes.
// Set any value to 0 to leave it unconstrained on that axis.
//
//	c := ui.Constrain(canvas.Size{W: 320}, canvas.Size{W: 800, H: 600}, myWidget)
type Constrained struct {
	Min    canvas.Size
	Max    canvas.Size
	Align  Alignment
	Child  Component
	bounds canvas.Rect
}

func Constrain(min, max canvas.Size, child Component) *Constrained {
	return &Constrained{Min: min, Max: max, Align: AlignmentCenter, Child: child}
}

func clampF(v, lo, hi float32) float32 {
	if lo > 0 && v < lo {
		return lo
	}
	if hi > 0 && v > hi {
		return hi
	}
	return v
}

func (c *Constrained) Bounds() canvas.Rect { return c.bounds }
func (c *Constrained) Tick(delta float64) {
	if c.Child != nil {
		c.Child.Tick(delta)
	}
}
func (c *Constrained) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	c.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	cw := clampF(w, c.Min.W, c.Max.W)
	ch := clampF(h, c.Min.H, c.Max.H)
	ox, oy := alignOffset(w, h, cw, ch, c.Align)
	if c.Child != nil {
		c.Child.Draw(cv, x+ox, y+oy, cw, ch)
	}
}
func (c *Constrained) HandleEvent(e Event) bool {
	if c.Child != nil {
		return c.Child.HandleEvent(e)
	}
	return false
}

// ────────────────────────────────────────────────────────────────────────────
// FillExpanded — a transparent spacer that expands to fill its container
// ────────────────────────────────────────────────────────────────────────────

// Spacer is a no-op Component that takes up space in a Flex layout.
type Spacer struct {
	bounds canvas.Rect
}

func NewSpacer() *Spacer { return &Spacer{} }

func (s *Spacer) Bounds() canvas.Rect { return s.bounds }
func (s *Spacer) Tick(_ float64)      {}
func (s *Spacer) Draw(_ *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
}
func (s *Spacer) HandleEvent(_ Event) bool { return false }
