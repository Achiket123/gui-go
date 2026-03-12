package goui

import (
	"runtime"
	"sync"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/platform"
	"github.com/achiket123/gui-go/render"
	"github.com/achiket123/gui-go/render/gl"
	"github.com/achiket123/gui-go/ui"
)

// Window represents a top-level window.
// Create one with NewWindow, register callbacks, then call Show() to display it.
type Window struct {
	// Platform handle
	handle *platform.WindowHandle

	// Window state
	width, height int
	title         string
	running       bool

	// Colors
	bgColor Color

	// Animation ticker list — the loop calls Tick(delta) on each per frame
	animations []Animatable

	// v2 renderer + canvas
	renderer     render.Renderer
	rendererIsGL bool

	// Animation v2 controllers/timelines (ticked by loop)
	controllers []*animCtrlWrapper
	timelines   []*tlWrapper

	// UI retained components
	components []ui.Component

	// User callbacks
	onDrawCanvas func(c *canvas.Canvas) // v2: GL/SW canvas
	onMouseClick func(e MouseEvent)
	onMouseMove  func(e MouseEvent)
	onKeyPress   func(e KeyEvent)
	onKeyRelease func(e KeyEvent)
	onResize     func(width, height int)
	onClose      func()
}

// animCtrlWrapper wraps an AnimationController for the loop.
type animCtrlWrapper struct {
	ctrl interface{ Tick(float64) bool }
}

// tlWrapper wraps a Timeline for the loop.
type tlWrapper struct {
	tl interface{ Tick(float64) bool }
}

// windowCount tracks the number of active windows.
var (
	windowInitMu sync.Mutex
	windowCount  int
)

// NewWindow creates and configures a window.
// The window is not yet visible — call Show() to make it appear.
// This is the canonical constructor; use it directly or via the goui package.
func NewWindow(title string, width, height int) *Window {
	// Initialize platform backend if not already done.
	windowInitMu.Lock()
	if windowCount == 0 {
		err := platform.Init()
		if err != nil {
			windowInitMu.Unlock()
			panic("goui: cannot initialize platform: " + err.Error())
		}
	}
	windowCount++
	windowInitMu.Unlock()

	handle, err := platform.CreateWindow(title, width, height)
	if err != nil {
		panic("goui: cannot create window: " + err.Error())
	}

	bgColor := Color{30, 30, 30}

	return &Window{
		handle:  handle,
		width:   width,
		height:  height,
		title:   title,
		bgColor: bgColor,
	}
}

// --- Callback registration ---

// OnDrawGL registers the v2 GPU-accelerated canvas draw callback.
// This enables the OpenGL renderer automatically when Show() is called.
func (w *Window) OnDrawGL(fn func(c *canvas.Canvas)) {
	w.onDrawCanvas = fn
}

// OnMouseClick registers a handler for mouse button press/release events.
func (w *Window) OnMouseClick(fn func(e MouseEvent)) {
	w.onMouseClick = fn
}

// OnMouseMove registers a handler for mouse motion events.
func (w *Window) OnMouseMove(fn func(e MouseEvent)) {
	w.onMouseMove = fn
}

// OnKeyPress registers a handler for key press events.
func (w *Window) OnKeyPress(fn func(e KeyEvent)) {
	w.onKeyPress = fn
}

// OnKeyRelease registers a handler for key release events.
func (w *Window) OnKeyRelease(fn func(e KeyEvent)) {
	w.onKeyRelease = fn
}

// OnResize registers a handler called when the window is resized.
func (w *Window) OnResize(fn func(width, height int)) {
	w.onResize = fn
}

// OnClose registers a handler called when the window is about to close.
func (w *Window) OnClose(fn func()) {
	w.onClose = fn
}

// --- Window controls ---

// SetTitle updates the window title bar text.
func (w *Window) SetTitle(title string) {
	w.title = title
	if w.handle != nil {
		w.handle.SetTitle(title)
	}
}

// Resize changes the window dimensions.
func (w *Window) Resize(width, height int) {
	w.width = width
	w.height = height
	if w.handle != nil {
		w.handle.Resize(width, height)
	}
	if w.renderer != nil {
		w.renderer.Resize(width, height)
	}
}

// AddAnimation registers an Animatable (Tween, Sequence, Sprite) to be ticked
// every frame. Finished non-looping animations are removed automatically.
func (w *Window) AddAnimation(a Animatable) {
	w.animations = append(w.animations, a)
}

// AddComponent registers a ui.Component to be ticked and drawn each frame.
func (w *Window) AddComponent(c ui.Component) {
	w.components = append(w.components, c)
}

// AddController registers a v2 AnimationController to be ticked each frame.
func (w *Window) AddController(ctrl interface{ Tick(float64) bool }) {
	w.controllers = append(w.controllers, &animCtrlWrapper{ctrl})
}

// AddTimeline registers a v2 Timeline to be ticked each frame.
func (w *Window) AddTimeline(tl interface{ Tick(float64) bool }) {
	w.timelines = append(w.timelines, &tlWrapper{tl})
}

// Close stops the render loop and destroys the window.
func (w *Window) Close() {
	w.running = false
}

// Show makes the window visible and starts the blocking render loop.
// It must be called from the goroutine that owns the window.
// It does not return until the window is closed.
func (w *Window) Show() {
	// Pin this goroutine to a single OS thread — required for OpenGL.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	w.handle.Show()

	// If an OnDrawGL callback was registered OR components were added, initialise the GL renderer.
	if w.onDrawCanvas != nil || len(w.components) > 0 {
		glr := gl.NewGL2DRenderer()
		if err := glr.Init(w.handle, platform.GetProcAddress, w.width, w.height); err == nil {
			w.renderer = glr
			w.rendererIsGL = true
		} else {
			w.onDrawCanvas = nil
		}
	}

	// Start the render loop
	lp := newLoop(w)
	lp.run()

	// Cleanup
	if w.renderer != nil {
		w.renderer.(*gl.GL2DRenderer).Destroy()
	}
	w.handle.Destroy()

	windowInitMu.Lock()
	windowCount--
	if windowCount == 0 {
		platform.Terminate()
	}
	windowInitMu.Unlock()
}

// --- Internal helpers ---

// dispatchEvent translates a raw platform.EventData into typed goui events
// and fires the registered callbacks.
func (w *Window) dispatchEvent(ev platform.EventData) {
	switch ev.Type {
	case platform.EventExpose:
		// Just trigger a redraw — handled by loop.

	case platform.EventKeyPress:
		shift := ev.State&platform.ModShift != 0
		e := ui.Event{Type: ui.EventKeyDown, Key: ev.KeySym, Shift: shift}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onKeyPress != nil {
			w.onKeyPress(KeyEvent{
				KeyCode: ev.KeyCode,
				KeySym:  ev.KeySym,
				Type:    "press",
				Shift:   ev.State&platform.ModShift != 0,
				Ctrl:    ev.State&platform.ModControl != 0,
				Alt:     ev.State&platform.ModAlt != 0,
			})
		}

	case platform.EventKeyRelease:
		if w.onKeyRelease != nil {
			w.onKeyRelease(KeyEvent{
				KeyCode: ev.KeyCode,
				KeySym:  ev.KeySym,
				Type:    "release",
				Shift:   ev.State&platform.ModShift != 0,
				Ctrl:    ev.State&platform.ModControl != 0,
				Alt:     ev.State&platform.ModAlt != 0,
			})
		}

	case platform.EventButtonPress:
		e := ui.Event{Type: ui.EventMouseDown, X: float32(ev.X), Y: float32(ev.Y), Button: ev.Button}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseClick != nil {
			w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: ev.Button, Type: "press"})
		}

	case platform.EventButtonRelease:
		e := ui.Event{Type: ui.EventMouseUp, X: float32(ev.X), Y: float32(ev.Y), Button: ev.Button}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseClick != nil {
			w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: ev.Button, Type: "release"})
		}

	case platform.EventMotionNotify:
		e := ui.Event{Type: ui.EventMouseMove, X: float32(ev.X), Y: float32(ev.Y)}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseMove != nil {
			w.onMouseMove(MouseEvent{X: ev.X, Y: ev.Y, Type: "move"})
		}

	case platform.EventScroll:
		e := ui.Event{
			Type:    ui.EventScroll,
			X:       float32(ev.X),
			Y:       float32(ev.Y),
			ScrollY: float32(-ev.ScrollY),
		}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseClick != nil {
			// Legacy compat: map scroll to button 4 (up) / 5 (down), X11 convention.
			// ev.ScrollY > 0 from GLFW means scroll up.
			btn := 5 // scroll down
			if ev.ScrollY > 0 {
				btn = 4 // scroll up
			}
			w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: btn, Type: "press"})
		}

	case platform.EventConfigureNotify:
		if ev.Width > 0 && ev.Height > 0 &&
			(ev.Width != w.width || ev.Height != w.height) {
			w.width = ev.Width
			w.height = ev.Height
			if w.renderer != nil {
				w.renderer.Resize(ev.Width, ev.Height)
			}
			if w.onResize != nil {
				w.onResize(ev.Width, ev.Height)
			}
		}

	case platform.EventDestroyNotify:
		w.running = false
		if w.onClose != nil {
			w.onClose()
		}
	}
}
