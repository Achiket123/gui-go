// Package ui provides retained, stateful UI components for goui v2.
// Unlike a full widget tree, these are "draw-yourself" components that you
// create once, register with the window, and call Draw(canvas, x, y, w, h) each frame.
//
// Event handling is driven by the window's event dispatching: the window
// forwards events to all registered components in reverse paint order.
package ui

import (
	"github.com/achiket123/gui-go/canvas"
)

// Event is a minimal event struct for component hit-testing.
// Mirrors the top-level event types without creating a circular import.
type Event struct {
	Type    EventType
	X, Y    float32
	Button  int
	Key     string
	Shift   bool
	ScrollY float32
}

// EventType identifies the kind of UI event.
type EventType int

const (
	EventMouseMove EventType = iota
	EventMouseDown
	EventMouseUp
	EventKeyDown
	EventScroll
)

// Component is the interface all retained UI components implement.
type Component interface {
	// Draw paints the component into c at the given bounds.
	Draw(c *canvas.Canvas, x, y, w, h float32)

	// HandleEvent processes a UI event. Returns true if consumed.
	HandleEvent(e Event) bool

	// Bounds returns the last drawn bounds (for hit testing).
	Bounds() canvas.Rect

	// Tick advances internal animation timers by delta seconds.
	Tick(delta float64)
}
