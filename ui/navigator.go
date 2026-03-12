package ui

import (
	"github.com/achiket123/gui-go/canvas"
)

// Screen is a full-page UI component with lifecycle hooks.
type Screen interface {
	Component
	OnEnter(nav *Navigator) // called when screen becomes active
	OnLeave()               // called when navigating away (not destroyed)
}

// TransitionState tracks a simple fade between screens.
type transitionState struct {
	active  bool
	opacity float32 // 0→1 on enter, 1→0 on leave
}

// Navigator manages a screen stack, an optional tab bar, and an optional modal.
// Register it once with w.AddComponent — it forwards all Draw/Tick/HandleEvent
// calls to the correct layer automatically.
type Navigator struct {
	stack  []Screen
	modal  Screen
	tabBar *TabBar

	// Independent stacks for tabs
	tabStacks  map[int][]Screen
	currentTab int

	// transition
	trans   transitionState
	onClose func() // called when Pop() empties the stack
}

// NewNavigator creates a Navigator with an initial screen.
func NewNavigator(initial Screen) *Navigator {
	n := &Navigator{
		tabStacks:  make(map[int][]Screen),
		currentTab: 0,
	}
	n.stack = []Screen{initial}
	n.tabStacks[0] = n.stack
	initial.OnEnter(n)
	return n
}

// SetTabBar attaches a persistent TabBar rendered below the current screen.
func (n *Navigator) SetTabBar(tb *TabBar) {
	n.tabBar = tb
}

// OnClose registers a callback for when Pop() is called on a single-item stack.
func (n *Navigator) OnClose(fn func()) {
	n.onClose = fn
}

// Push navigates to a new screen, keeping the current one in history.
func (n *Navigator) Push(s Screen) {
	if cur := n.current(); cur != nil {
		cur.OnLeave()
	}
	n.stack = append(n.stack, s)
	s.OnEnter(n)
	n.beginTransition()
}

// Pop returns to the previous screen. If the stack has one item, calls OnClose.
func (n *Navigator) Pop() {
	if len(n.stack) <= 1 {
		if n.onClose != nil {
			n.onClose()
		}
		return
	}
	n.current().OnLeave()
	n.stack = n.stack[:len(n.stack)-1]
	n.current().OnEnter(n)
	n.beginTransition()
}

// Replace swaps the current screen without keeping history.
func (n *Navigator) Replace(s Screen) {
	if cur := n.current(); cur != nil {
		cur.OnLeave()
	}
	n.stack[len(n.stack)-1] = s
	s.OnEnter(n)
	n.beginTransition()
}

// ResetTo clears the entire stack and starts fresh with s.
func (n *Navigator) ResetTo(s Screen) {
	for _, sc := range n.stack {
		sc.OnLeave()
	}
	n.stack = []Screen{s}
	// Update the current tab's saved stack as well
	n.tabStacks[n.currentTab] = n.stack
	s.OnEnter(n)
	n.beginTransition()
}

// SelectTab switches to a different tab stack.
// If the stack for that tab doesn't exist, it's initialized with root.
func (n *Navigator) SelectTab(index int, root Screen) {
	if n.currentTab == index {
		return
	}

	// 1. Save current stack
	if cur := n.current(); cur != nil {
		cur.OnLeave()
	}
	n.tabStacks[n.currentTab] = n.stack

	// 2. Load new stack
	n.currentTab = index
	stack, ok := n.tabStacks[index]
	if !ok {
		// New tab — initialize with root
		stack = []Screen{root}
		n.tabStacks[index] = stack
	}
	n.stack = stack

	// 3. Enter new screen
	if cur := n.current(); cur != nil {
		cur.OnEnter(n)
	}
	n.beginTransition()
}

// PushModal shows a screen as an overlay. Only one modal at a time.
func (n *Navigator) PushModal(s Screen) {
	if n.modal != nil {
		n.modal.OnLeave()
	}
	n.modal = s
	s.OnEnter(n)
}

// PopModal dismisses the current modal.
func (n *Navigator) PopModal() {
	if n.modal == nil {
		return
	}
	n.modal.OnLeave()
	n.modal = nil
}

// CanGoBack reports whether there is a previous screen to return to.
func (n *Navigator) CanGoBack() bool {
	return len(n.stack) > 1
}

// Depth returns how many screens are on the stack.
func (n *Navigator) Depth() int { return len(n.stack) }

func (n *Navigator) current() Screen {
	if len(n.stack) == 0 {
		return nil
	}
	return n.stack[len(n.stack)-1]
}

func (n *Navigator) beginTransition() {
	n.trans = transitionState{active: true, opacity: 0}
}

// ── Component interface ───────────────────────────────────────────────────

func (n *Navigator) Bounds() canvas.Rect {
	if cur := n.current(); cur != nil {
		return cur.Bounds()
	}
	return canvas.Rect{}
}

func (n *Navigator) Tick(delta float64) {
	// Advance fade-in transition
	if n.trans.active {
		n.trans.opacity += float32(delta) * 6 // ~160ms fade
		if n.trans.opacity >= 1 {
			n.trans.opacity = 1
			n.trans.active = false
		}
	}

	if cur := n.current(); cur != nil {
		cur.Tick(delta)
	}
	if n.modal != nil {
		n.modal.Tick(delta)
	}
	if n.tabBar != nil {
		n.tabBar.Tick(delta)
	}
}

func (n *Navigator) HandleEvent(e Event) bool {
	// Modal captures all input while active
	if n.modal != nil {
		// Clicking the dim overlay dismisses the modal
		if e.Type == EventMouseDown {
			mb := n.modal.Bounds()
			inModal := e.X >= mb.X && e.X <= mb.X+mb.W &&
				e.Y >= mb.Y && e.Y <= mb.Y+mb.H
			if !inModal {
				n.PopModal()
				return true
			}
		}
		return n.modal.HandleEvent(e)
	}

	// Tab bar only gets MouseDown — never MouseUp/Move
	// (prevents it from stealing MouseUp from a button mid-click)
	if n.tabBar != nil && e.Type == EventMouseDown {
		if n.tabBar.HandleEvent(e) {
			return true
		}
	}

	// Current screen
	if cur := n.current(); cur != nil {
		return cur.HandleEvent(e)
	}
	return false
}

func (n *Navigator) Draw(c *canvas.Canvas, x, y, w, h float32) {
	contentH := h
	if n.tabBar != nil {
		contentH = h - n.tabBar.Height
	}

	// Draw current screen
	if cur := n.current(); cur != nil {
		if n.trans.active {
			c.Save()
			c.SetGlobalOpacity(n.trans.opacity)
			cur.Draw(c, x, y, w, contentH)
			c.Restore()
		} else {
			cur.Draw(c, x, y, w, contentH)
		}
	}

	// Tab bar sits at the bottom
	if n.tabBar != nil {
		n.tabBar.Draw(c, x, y+contentH, w, n.tabBar.Height)
	}

	// Modal on top with dimmed background
	if n.modal != nil {
		// Dim overlay
		c.DrawRect(x, y, w, h, canvas.FillPaint(canvas.RGBA8(0, 0, 0, 160)))
		// Modal content — centered by convention, sized by the screen itself
		n.modal.Draw(c, x, y, w, h)
	}
}
