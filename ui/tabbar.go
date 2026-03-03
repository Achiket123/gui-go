package ui

import (
	"github.com/achiket/gui-go/canvas"
)

// TabItem describes a single tab entry.
type TabItem struct {
	Label  string
	Screen Screen
}

// TabBar is a persistent bottom navigation bar.
// When a tab is selected, it calls nav.ResetTo with that tab's screen.
type TabBar struct {
	Items       []TabItem
	Height      float32
	ActiveIndex int
	Style       TabBarStyle

	nav    *Navigator
	bounds canvas.Rect
}

// TabBarStyle controls tab bar appearance.
type TabBarStyle struct {
	Background    canvas.Color
	ActiveColor   canvas.Color
	InactiveColor canvas.Color
	BorderColor   canvas.Color
	TextSize      float32
}

// DefaultTabBarStyle returns a dark-themed tab bar style.
func DefaultTabBarStyle() TabBarStyle {
	return TabBarStyle{
		Background:    canvas.Hex("#181825"),
		ActiveColor:   canvas.Hex("#89B4FA"),
		InactiveColor: canvas.Hex("#6C7086"),
		BorderColor:   canvas.Hex("#313244"),
		TextSize:      12,
	}
}

// NewTabBar creates a TabBar bound to a Navigator.
func NewTabBar(nav *Navigator, height float32, items []TabItem) *TabBar {
	tb := &TabBar{
		Items:  items,
		Height: height,
		Style:  DefaultTabBarStyle(),
		nav:    nav,
	}
	return tb
}

func (tb *TabBar) Bounds() canvas.Rect  { return tb.bounds }
func (tb *TabBar) Tick(_ float64)       {}
func (tb *TabBar) OnEnter(_ *Navigator) {}
func (tb *TabBar) OnLeave()             {}

func (tb *TabBar) HandleEvent(e Event) bool {
	if e.Type != EventMouseDown {
		return false
	}
	// Guard: if bounds haven't been set yet, don't consume anything
	if tb.bounds.W == 0 {
		return false
	}
	if e.Y < tb.bounds.Y || e.Y > tb.bounds.Y+tb.bounds.H {
		return false
	}
	tabW := tb.bounds.W / float32(len(tb.Items))
	idx := int((e.X - tb.bounds.X) / tabW)
	if idx < 0 || idx >= len(tb.Items) {
		return false
	}
	if idx == tb.ActiveIndex {
		return true // already active — no-op
	}
	tb.ActiveIndex = idx
	tb.nav.SelectTab(idx, tb.Items[idx].Screen)
	return true
}

func (tb *TabBar) Draw(c *canvas.Canvas, x, y, w, h float32) {
	tb.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	// Background + top border
	c.DrawRect(x, y, w, h, canvas.FillPaint(tb.Style.Background))
	c.DrawRect(x, y, w, 1, canvas.FillPaint(tb.Style.BorderColor))

	tabW := w / float32(len(tb.Items))
	for i, item := range tb.Items {
		tx := x + float32(i)*tabW
		col := tb.Style.InactiveColor
		if i == tb.ActiveIndex {
			col = tb.Style.ActiveColor
			// Active indicator line at top of tab
			c.DrawRect(tx+tabW*0.2, y, tabW*0.6, 2, canvas.FillPaint(col))
		}
		ts := canvas.TextStyle{Color: col, Size: tb.Style.TextSize}
		sz := c.MeasureText(item.Label, ts)
		c.DrawText(tx+(tabW-sz.W)/2, y+h/2+tb.Style.TextSize*0.35, item.Label, ts)
	}
}
