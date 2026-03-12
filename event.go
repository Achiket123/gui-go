package goui

// MouseEvent carries information about a mouse press, release, or move.
type MouseEvent struct {
	// X is the mouse horizontal position relative to the window.
	X int
	// Y is the mouse vertical position relative to the window.
	Y int
	// Button is the mouse button: 1=Left, 2=Middle, 3=Right, 4=ScrollUp, 5=ScrollDown.
	Button int
	// Type is "press", "release", or "move".
	Type string
}

// KeyEvent carries information about a key press or release.
type KeyEvent struct {
	// KeyCode is the raw platform keycode (e.g., GLFW key constant).
	// It is platform-dependent; prefer KeySym for portable character mapping.
	KeyCode int
	// KeySym is the named symbol, e.g. "a", "Return", "Escape", "space".
	KeySym string
	// Type is "press" or "release".
	Type string
	// Shift indicates whether Shift was held.
	Shift bool
	// Ctrl indicates whether Ctrl was held.
	Ctrl bool
	// Alt indicates whether Alt was held.
	Alt bool
}

// ResizeEvent carries the new dimensions after a window resize.
type ResizeEvent struct {
	Width  int
	Height int
}
