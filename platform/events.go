package platform

// EventType constants for platform-agnostic event handling
const (
	EventKeyPress = iota
	EventKeyRelease
	EventButtonPress
	EventButtonRelease
	EventMotionNotify
	EventConfigureNotify
	EventExpose
	EventDestroyNotify
	EventClientMessage
	EventScroll
)

// Modifier masks
const (
	ModShift = 1 << iota
	ModControl
	ModAlt
	ModSuper
)

// EventData carries platform-neutral event information.
type EventData struct {
	Type    int
	X, Y    int
	Button  int
	KeySym  string
	KeyCode int // Kept for backward compatibility
	Width   int
	Height  int
	State   int
	ScrollX float64
	ScrollY float64
}
