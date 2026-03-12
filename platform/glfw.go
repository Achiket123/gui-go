package platform

import (
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"
)

// Init initializes the GLFW platform backend.
func Init() error {
	return glfw.Init()
}

// Terminate shuts down the GLFW platform backend.
func Terminate() {
	glfw.Terminate()
}

// WindowHandle wraps a GLFW window and its events queue.
type WindowHandle struct {
	w      *glfw.Window
	events []EventData
}

// CreateWindow creates a new hardware-accelerated window.
func CreateWindow(title string, width, height int) (*WindowHandle, error) {
	glfw.WindowHint(glfw.Visible, glfw.False) // Hidden initially
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	w, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		return nil, err
	}

	h := &WindowHandle{w: w}

	// Register callbacks for events
	w.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		evType := EventKeyPress
		if action == glfw.Release {
			evType = EventKeyRelease
		} else if action == glfw.Repeat {
			evType = EventKeyPress // Match X11 repeat behavior
		}

		h.events = append(h.events, EventData{
			Type:    evType,
			KeyCode: int(key),
			KeySym:  glfwKeyToString(key, mods),
			State:   glfwModsToState(mods),
		})
	})

	w.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		evType := EventButtonPress
		if action == glfw.Release {
			evType = EventButtonRelease
		}

		x, y := w.GetCursorPos()
		h.events = append(h.events, EventData{
			Type:   evType,
			Button: int(button) + 1, // X11 buttons are 1-indexed
			X:      int(x),
			Y:      int(y),
			State:  glfwModsToState(mods),
		})
	})

	w.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		h.events = append(h.events, EventData{
			Type: EventMotionNotify,
			X:    int(xpos),
			Y:    int(ypos),
		})
	})

	w.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		x, y := w.GetCursorPos()
		h.events = append(h.events, EventData{
			Type:    EventScroll,
			X:       int(x),
			Y:       int(y),
			ScrollX: xoff,
			ScrollY: yoff,
		})
	})

	w.SetSizeCallback(func(w *glfw.Window, width, height int) {
		h.events = append(h.events, EventData{
			Type:   EventConfigureNotify,
			Width:  width,
			Height: height,
		})
	})

	w.SetCloseCallback(func(w *glfw.Window) {
		h.events = append(h.events, EventData{
			Type: EventDestroyNotify,
		})
	})

	return h, nil
}

// Show makes the window visible.
func (h *WindowHandle) Show() {
	h.w.Show()
}

// Destroy destroys the window.
func (h *WindowHandle) Destroy() {
	h.w.Destroy()
}

// SwapBuffers swaps the front and back buffers.
func (h *WindowHandle) SwapBuffers() {
	h.w.SwapBuffers()
}

// MakeContextCurrent makes the OpenGL context current for this window.
func (h *WindowHandle) MakeContextCurrent() {
	h.w.MakeContextCurrent()
}

// ShouldClose returns true if the user requested the window to close.
func (h *WindowHandle) ShouldClose() bool {
	return h.w.ShouldClose()
}

// SetTitle updates the window title.
func (h *WindowHandle) SetTitle(title string) {
	h.w.SetTitle(title)
}

// Resize updates the window size.
func (h *WindowHandle) Resize(width, height int) {
	h.w.SetSize(width, height)
}

// PollEvents polls for pending events and invokes callbacks.
func PollEvents() {
	glfw.PollEvents()
}

// DrainEvents returns all queued events for this window since the last poll.
func (h *WindowHandle) DrainEvents() []EventData {
	if len(h.events) == 0 {
		return nil
	}
	evs := h.events
	h.events = nil
	return evs
}

// GetProcAddress returns the OpenGL function pointer for the given name.
func GetProcAddress(name string) unsafe.Pointer {
	return glfw.GetProcAddress(name)
}

// --- Helpers ---

func glfwModsToState(mods glfw.ModifierKey) int {
	var state int
	if mods&glfw.ModShift != 0 {
		state |= ModShift
	}
	if mods&glfw.ModControl != 0 {
		state |= ModControl
	}
	if mods&glfw.ModAlt != 0 {
		state |= ModAlt
	}
	if mods&glfw.ModSuper != 0 {
		state |= ModSuper
	}
	return state
}

func glfwKeyToString(k glfw.Key, mods glfw.ModifierKey) string {
	switch k {
	case glfw.KeyEnter:
		return "Return"
	case glfw.KeyEscape:
		return "Escape"
	case glfw.KeyBackspace:
		return "BackSpace"
	case glfw.KeyTab:
		return "Tab"
	case glfw.KeySpace:
		return "space"
	case glfw.KeyUp:
		return "Up"
	case glfw.KeyDown:
		return "Down"
	case glfw.KeyLeft:
		return "Left"
	case glfw.KeyRight:
		return "Right"
	case glfw.KeyF1:
		return "F1"
	case glfw.KeyF2:
		return "F2"
	case glfw.KeyF3:
		return "F3"
	case glfw.KeyF4:
		return "F4"
	case glfw.KeyF5:
		return "F5"
	case glfw.KeyF6:
		return "F6"
	case glfw.KeyF7:
		return "F7"
	case glfw.KeyF8:
		return "F8"
	case glfw.KeyF9:
		return "F9"
	case glfw.KeyF10:
		return "F10"
	case glfw.KeyF11:
		return "F11"
	case glfw.KeyF12:
		return "F12"
	}

	name := glfw.GetKeyName(k, 0)
	if name != "" {
		return name
	}
	return ""
}
