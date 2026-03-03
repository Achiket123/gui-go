package goui

import (
	"runtime"
	"unsafe"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/platform"
	"github.com/achiket/gui-go/render"
	"github.com/achiket/gui-go/render/gl"
	"github.com/achiket/gui-go/ui"
)

// Window represents a top-level X11 window.
// Create one with NewWindow, register callbacks, then call Show() to display it.
type Window struct {
	// X11 core handles (unsafe.Pointer to keep this file CGo-free)
	display  unsafe.Pointer
	xwin     uintptr
	gc       unsafe.Pointer
	colormap unsafe.Pointer
	pixmap   uintptr // offscreen double-buffer pixmap
	screen   int
	depth    int

	// Font state
	currentFont unsafe.Pointer

	// Window state
	width, height int
	title         string
	running       bool

	// Colors
	bgColor Color

	// WM_DELETE_WINDOW atom
	wmDeleteWindow uint64

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
	onDraw       func(c *Canvas)        // v1: old Xlib canvas
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

// NewWindow creates and configures an X11 window.
// The window is not yet visible — call Show() to make it appear.
// This is the canonical constructor; use it directly or via the goui package.
func NewWindow(title string, width, height int) *Window {
	// Open display
	display := platform.OpenDisplay()
	if display == nil {
		panic("goui: cannot open X display — is DISPLAY set?")
	}

	screen := platform.DefaultScreen(display)
	root := platform.DefaultRootWindow(display)
	colormap := platform.DefaultColormap(display, screen)
	depth := platform.DefaultDepth(display, screen)

	// Allocate colors for window border and background
	bgColor := Color{30, 30, 30}
	bgPixel := bgColor.ToXPixel(display, colormap)
	fgPixel := White.ToXPixel(display, colormap)

	xwin := platform.CreateSimpleWindow(
		display, root,
		0, 0, width, height,
		1,
		fgPixel, bgPixel,
	)

	// Set title
	platform.StoreName(display, xwin, title)

	// Register interest in events
	mask := platform.ExposureMask |
		platform.KeyPressMask |
		platform.KeyReleaseMask |
		platform.ButtonPressMask |
		platform.ButtonReleaseMask |
		platform.PointerMotionMask |
		platform.StructureNotifyMask
	platform.SelectInput(display, xwin, mask)

	// Create Graphics Context
	gc := platform.CreateGC(display, xwin)

	// Register WM_DELETE_WINDOW so we get notified when user clicks ✕
	wmDelete := platform.InternAtom(display, "WM_DELETE_WINDOW", false)
	platform.SetWMProtocols(display, xwin, []uint64{wmDelete})

	return &Window{
		display:        display,
		xwin:           xwin,
		gc:             gc,
		colormap:       colormap,
		screen:         screen,
		depth:          depth,
		width:          width,
		height:         height,
		title:          title,
		bgColor:        bgColor,
		wmDeleteWindow: wmDelete,
	}
}

// --- Callback registration ---

// OnDraw registers the v1 Xlib draw callback (backward compat).
func (w *Window) OnDraw(fn func(c *Canvas)) {
	w.onDraw = fn
}

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
	platform.StoreName(w.display, w.xwin, title)
}

// Resize changes the window dimensions.
func (w *Window) Resize(width, height int) {
	w.width = width
	w.height = height
	platform.ResizeWindow(w.display, w.xwin, width, height)
	w.rebuildPixmap()
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
	// Pin this goroutine to a single OS thread — required for X11 and OpenGL.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	platform.MapWindow(w.display, w.xwin)

	// If an OnDrawGL callback was registered, initialise the GL renderer.
	if w.onDrawCanvas != nil {
		glr := gl.NewGL2DRenderer()
		if err := glr.Init(w.display, w.xwin, w.width, w.height); err == nil {
			w.renderer = glr
			w.rendererIsGL = true
		} else {
			// GL failed — fall through to Pixmap loop (OnDrawGL won't render, but won't crash).
			w.onDrawCanvas = nil
		}
	}

	if !w.rendererIsGL {
		// v1: Create double-buffering pixmap
		w.rebuildPixmap()
	}

	// Start the render loop
	lp := newLoop(w)
	lp.run()

	// Cleanup
	if w.renderer != nil {
		// GL cleanup handled by renderer
	} else if w.pixmap != 0 {
		platform.FreePixmap(w.display, w.pixmap)
	}
	platform.FreeGC(w.display, w.gc)
	platform.DestroyWindow(w.display, w.xwin)
	platform.CloseDisplay(w.display)
}

// --- Internal helpers ---

func (w *Window) rebuildPixmap() {
	if w.pixmap != 0 {
		platform.FreePixmap(w.display, w.pixmap)
	}
	w.pixmap = platform.CreatePixmap(w.display, w.xwin, w.width, w.height, w.depth)
}

// dispatchEvent translates a raw platform.XEventData into typed goui events
// and fires the registered callbacks.
func (w *Window) dispatchEvent(ev platform.XEventData) {
	switch ev.Type {
	case platform.Expose:
		// Just trigger a redraw — handled by loop.

	case platform.KeyPress:
		shift := ev.State&platform.ShiftMask != 0
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
				Shift:   ev.State&platform.ShiftMask != 0,
				Ctrl:    ev.State&platform.ControlMask != 0,
				Alt:     ev.State&platform.Mod1Mask != 0,
			})
		}

	case platform.KeyRelease:
		if w.onKeyRelease != nil {
			w.onKeyRelease(KeyEvent{
				KeyCode: ev.KeyCode,
				KeySym:  ev.KeySym,
				Type:    "release",
				Shift:   ev.State&platform.ShiftMask != 0,
				Ctrl:    ev.State&platform.ControlMask != 0,
				Alt:     ev.State&platform.Mod1Mask != 0,
			})
		}

	case platform.ButtonPress:
		if ev.Button == 4 || ev.Button == 5 {
			sy := float32(-1)
			if ev.Button == 5 {
				sy = 1
			}
			e := ui.Event{
				Type:    ui.EventScroll,
				X:       float32(ev.X),
				Y:       float32(ev.Y),
				ScrollY: sy,
			}
			for i := len(w.components) - 1; i >= 0; i-- {
				if w.components[i].HandleEvent(e) {
					return
				}
			}
			if w.onMouseClick != nil {
				w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: ev.Button, Type: "press"})
			}
			return
		}

		e := ui.Event{Type: ui.EventMouseDown, X: float32(ev.X), Y: float32(ev.Y), Button: ev.Button}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseClick != nil {
			w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: ev.Button, Type: "press"})
		}

	case platform.ButtonRelease:
		e := ui.Event{Type: ui.EventMouseUp, X: float32(ev.X), Y: float32(ev.Y), Button: ev.Button}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseClick != nil {
			w.onMouseClick(MouseEvent{X: ev.X, Y: ev.Y, Button: ev.Button, Type: "release"})
		}

	case platform.MotionNotify:
		e := ui.Event{Type: ui.EventMouseMove, X: float32(ev.X), Y: float32(ev.Y)}
		for i := len(w.components) - 1; i >= 0; i-- {
			if w.components[i].HandleEvent(e) {
				return
			}
		}
		if w.onMouseMove != nil {
			w.onMouseMove(MouseEvent{X: ev.X, Y: ev.Y, Type: "move"})
		}

	case platform.ConfigureNotify:
		if ev.Width > 0 && ev.Height > 0 &&
			(ev.Width != w.width || ev.Height != w.height) {
			w.width = ev.Width
			w.height = ev.Height
			if w.renderer != nil {
				w.renderer.Resize(ev.Width, ev.Height)
			} else {
				w.rebuildPixmap()
			}
			if w.onResize != nil {
				w.onResize(ev.Width, ev.Height)
			}
		}

	case platform.DestroyNotify:
		w.running = false
		if w.onClose != nil {
			w.onClose()
		}

	case platform.ClientMessage:
		if ev.Atom == w.wmDeleteWindow {
			if w.onClose != nil {
				w.onClose()
			}
			w.running = false
		}
	}
}
