// Package ui — accessibility.go
//
// Accessibility helpers:
//
//	A11yRole      — semantic role constants (button, checkbox, slider …)
//	Accessible    — wraps any Component with role metadata + disabled state
//	FocusManager  — Tab / Shift-Tab / click-to-focus across registered items
//	FocusRing     — visible keyboard-focus indicator drawn on top of everything
//	FocusableButton — Button that participates in the FocusManager
package ui

import (
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/theme"
)

// ─────────────────────────────────────────────────────────────────────────────
// A11yRole
// ─────────────────────────────────────────────────────────────────────────────

// A11yRole classifies a component for tooling and screen-reader bridges.
type A11yRole string

const (
	RoleButton    A11yRole = "button"
	RoleTextInput A11yRole = "textinput"
	RoleCheckbox  A11yRole = "checkbox"
	RoleRadio     A11yRole = "radio"
	RoleSlider    A11yRole = "slider"
	RoleListItem  A11yRole = "listitem"
	RoleHeading   A11yRole = "heading"
	RoleLink      A11yRole = "link"
	RoleImage     A11yRole = "image"
	RoleDialog    A11yRole = "dialog"
	RoleNone      A11yRole = ""
)

// A11yMeta holds the accessibility metadata for a component.
type A11yMeta struct {
	Role        A11yRole
	Label       string // accessible name
	Description string // longer description
	Disabled    bool
	Required    bool
}

// ─────────────────────────────────────────────────────────────────────────────
// Accessible — metadata wrapper
// ─────────────────────────────────────────────────────────────────────────────

// Accessible wraps a Component with A11yMeta.
// All drawing and events are forwarded unchanged; disabled state is respected.
type Accessible struct {
	Child  Component
	Meta   A11yMeta
	bounds canvas.Rect
}

// NewAccessible wraps child with the given metadata.
func NewAccessible(child Component, meta A11yMeta) *Accessible {
	return &Accessible{Child: child, Meta: meta}
}

func (a *Accessible) Bounds() canvas.Rect { return a.bounds }

func (a *Accessible) Tick(delta float64) {
	if a.Child != nil {
		a.Child.Tick(delta)
	}
}

func (a *Accessible) Draw(c *canvas.Canvas, x, y, w, h float32) {
	a.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if a.Child != nil {
		a.Child.Draw(c, x, y, w, h)
	}
}

func (a *Accessible) HandleEvent(e Event) bool {
	if a.Meta.Disabled {
		return false
	}
	if a.Child != nil {
		return a.Child.HandleEvent(e)
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Focusable interface
// ─────────────────────────────────────────────────────────────────────────────

// Focusable is any component that can receive keyboard focus.
type Focusable interface {
	Component
	Focus()
	Blur()
	IsFocused() bool
}

// ─────────────────────────────────────────────────────────────────────────────
// FocusManager
// ─────────────────────────────────────────────────────────────────────────────

// FocusManager maintains a tab-order list and handles Tab / Shift-Tab / Escape
// keyboard events plus click-to-focus.
//
// Register it once with the window alongside your other components.
//
//	fm := ui.NewFocusManager()
//	fm.Add(nameInput, emailInput, submitBtn)
//	window.Register(fm)
type FocusManager struct {
	items   []Focusable
	current int // -1 = nothing focused
	bounds  canvas.Rect
}

// NewFocusManager creates an empty FocusManager.
func NewFocusManager() *FocusManager { return &FocusManager{current: -1} }

// Add registers focusable components in tab order.
func (f *FocusManager) Add(items ...Focusable) {
	f.items = append(f.items, items...)
}

// FocusIndex programmatically focuses the item at index i.
func (f *FocusManager) FocusIndex(i int) {
	if f.current >= 0 && f.current < len(f.items) {
		f.items[f.current].Blur()
	}
	if i >= 0 && i < len(f.items) {
		f.current = i
		f.items[i].Focus()
	} else {
		f.current = -1
	}
}

// FocusFirst focuses the first registered item.
func (f *FocusManager) FocusFirst() { f.FocusIndex(0) }

// FocusNext advances to the next item (wraps around).
func (f *FocusManager) FocusNext() {
	n := len(f.items)
	if n == 0 {
		return
	}
	f.FocusIndex((f.current + 1) % n)
}

// FocusPrev goes to the previous item (wraps around).
func (f *FocusManager) FocusPrev() {
	n := len(f.items)
	if n == 0 {
		return
	}
	prev := f.current - 1
	if prev < 0 {
		prev = n - 1
	}
	f.FocusIndex(prev)
}

// Blur clears focus from all items.
func (f *FocusManager) Blur() {
	if f.current >= 0 && f.current < len(f.items) {
		f.items[f.current].Blur()
		f.current = -1
	}
}

// CurrentItem returns the currently focused item, or nil.
func (f *FocusManager) CurrentItem() Focusable {
	if f.current >= 0 && f.current < len(f.items) {
		return f.items[f.current]
	}
	return nil
}

func (f *FocusManager) Bounds() canvas.Rect { return f.bounds }
func (f *FocusManager) Tick(_ float64)      {}
func (f *FocusManager) Draw(_ *canvas.Canvas, x, y, w, h float32) {
	f.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
}

func (f *FocusManager) HandleEvent(e Event) bool {
	switch e.Type {
	case EventKeyDown:
		switch e.Key {
		case "Tab":
			if e.Shift {
				f.FocusPrev()
			} else {
				f.FocusNext()
			}
			return true
		case "Escape":
			f.Blur()
			return true
		}
	case EventMouseDown:
		if e.Button == 1 {
			// Click-to-focus: find the item whose bounds contain the click.
			for i, item := range f.items {
				b := item.Bounds()
				if e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H {
					f.FocusIndex(i)
					// Don't consume — let the item handle the click too.
					return false
				}
			}
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// FocusRing
// ─────────────────────────────────────────────────────────────────────────────

// FocusRingStyle configures the visual focus indicator.
type FocusRingStyle struct {
	Color  canvas.Color
	Width  float32
	Radius float32
	Offset float32 // outset from the component bounds
	Dashed bool    // draw a dashed outline instead of solid
}

// DefaultFocusRingStyle returns a theme-aware FocusRingStyle.
func DefaultFocusRingStyle() FocusRingStyle {
	th := theme.Current()
	return FocusRingStyle{
		Color:  th.Colors.BorderFocus,
		Width:  2,
		Radius: th.Radius.SM + 2,
		Offset: 2,
	}
}

// FocusRing draws a coloured ring around the currently focused component.
// Add it as the last registered component so it renders on top.
//
//	ring := ui.NewFocusRing(fm, ui.DefaultFocusRingStyle())
//	window.Register(ring)
type FocusRing struct {
	FM     *FocusManager
	Style  FocusRingStyle
	bounds canvas.Rect
}

func NewFocusRing(fm *FocusManager, style FocusRingStyle) *FocusRing {
	return &FocusRing{FM: fm, Style: style}
}

func (fr *FocusRing) Bounds() canvas.Rect      { return fr.bounds }
func (fr *FocusRing) Tick(_ float64)           {}
func (fr *FocusRing) HandleEvent(_ Event) bool { return false }

func (fr *FocusRing) Draw(c *canvas.Canvas, x, y, w, h float32) {
	fr.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if fr.FM == nil || fr.FM.current < 0 || fr.FM.current >= len(fr.FM.items) {
		return
	}
	item := fr.FM.items[fr.FM.current]
	if !item.IsFocused() {
		return
	}
	b := item.Bounds()
	off := fr.Style.Offset
	rx, ry, rw, rh := b.X-off, b.Y-off, b.W+off*2, b.H+off*2
	r := fr.Style.Radius

	if fr.Style.Dashed {
		col := fr.Style.Color
		dash, gap := float32(5), float32(3)
		fill := canvas.FillPaint(col)
		for cx2 := rx; cx2 < rx+rw; cx2 += dash + gap {
			sw := dash
			if cx2+sw > rx+rw {
				sw = rx + rw - cx2
			}
			c.DrawRect(cx2, ry, sw, fr.Style.Width, fill)
			c.DrawRect(cx2, ry+rh-fr.Style.Width, sw, fr.Style.Width, fill)
		}
		for cy2 := ry; cy2 < ry+rh; cy2 += dash + gap {
			sh := dash
			if cy2+sh > ry+rh {
				sh = ry + rh - cy2
			}
			c.DrawRect(rx, cy2, fr.Style.Width, sh, fill)
			c.DrawRect(rx+rw-fr.Style.Width, cy2, fr.Style.Width, sh, fill)
		}
	} else {
		c.DrawRoundedRect(rx, ry, rw, rh, r, canvas.StrokePaint(fr.Style.Color, fr.Style.Width))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FocusableButton
// ─────────────────────────────────────────────────────────────────────────────

// FocusableButton wraps Button and implements Focusable so it participates in
// keyboard tab order and can be activated with Space/Enter.
type FocusableButton struct {
	*Button
	focused bool
}

// NewFocusableButton creates a button that can receive keyboard focus.
func NewFocusableButton(label string, style ButtonStyle) *FocusableButton {
	btn := NewButton(label, nil)
	btn.Style = style
	return &FocusableButton{Button: btn}
}

func (fb *FocusableButton) Focus()          { fb.focused = true }
func (fb *FocusableButton) Blur()           { fb.focused = false }
func (fb *FocusableButton) IsFocused() bool { return fb.focused }

func (fb *FocusableButton) HandleEvent(e Event) bool {
	if fb.focused && e.Type == EventKeyDown {
		if e.Key == "Return" || e.Key == "space" {
			if fb.OnClick != nil {
				fb.OnClick()
			}
			return true
		}
	}
	return fb.Button.HandleEvent(e)
}
