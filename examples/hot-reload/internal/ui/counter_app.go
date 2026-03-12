// Package ui contains the counter application's UI components.
package ui

import (
	"fmt"
	"log"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/state"
	"github.com/achiket123/gui-go/ui"
)

type CounterWidget struct {
	count  *state.Signal[int]
	bounds canvas.Rect
}

func NewCounterWidget(count *state.Signal[int]) *CounterWidget {
	return &CounterWidget{count: count}
}

func (w *CounterWidget) Bounds() canvas.Rect { return w.bounds }
func (w *CounterWidget) Tick(_ float64)      {}
func (w *CounterWidget) HandleEvent(e ui.Event) bool {
	// Scroll up/down also changes the counter
	if e.Type == ui.EventScroll {
		if e.ScrollY > 0 {
			w.count.Update(func(v int) int { return v + 1 })
		} else {
			w.count.Update(func(v int) int { return v - 1 })
		}
		return true
	}
	return false
}

func (w *CounterWidget) Draw(c *canvas.Canvas, x, y, width, height float32) {
	w.bounds = canvas.Rect{X: x, Y: y, W: width, H: height}

	val := w.count.Get()

	// Background pill
	bg := canvas.Color{R: 0.20, G: 0.15, B: 0.25, A: 1}
	c.DrawRoundedRect(x+20, y+20, width-40, height-40, 16, canvas.FillPaint(bg))

	// Value text — colour shifts red→green based on sign
	col := canvas.Color{R: 0.56, G: 0.93, B: 0.56, A: 1} // green
	if val < 0 {
		col = canvas.Color{R: 0.95, G: 0.36, B: 0.42, A: 1} // red
	} else if val == 0 {
		col = canvas.Color{R: 0.8, G: 0.8, B: 0.8, A: 1} // grey
	}

	ts := canvas.TextStyle{Color: col, Size: 72}
	label := fmt.Sprintf("%d", val)
	sz := c.MeasureText(label, ts)
	cx := x + (width-sz.W)/2
	cy := y + (height+sz.H)/2 - 10
	c.DrawText(cx, cy, label, ts)

	// Subtle hint
	hint := canvas.TextStyle{Color: canvas.Color{R: 0.5, G: 0.5, B: 0.5, A: 1}, Size: 12}
	c.DrawText(x+(width-80)/2, y+height-30, "scroll to change", hint)
}

type ButtonBar struct {
	count     *state.Signal[int]
	history   *state.History[int]
	bounds    canvas.Rect
	btnBounds [4]canvas.Rect // -, +, Reset, Undo
	hovered   int            // index of hovered button, -1 = none
}

func NewButtonBar(count *state.Signal[int], history *state.History[int]) *ButtonBar {
	return &ButtonBar{count: count, history: history, hovered: -1}
}

func (b *ButtonBar) Bounds() canvas.Rect { return b.bounds }
func (b *ButtonBar) Tick(_ float64)      {}

func (b *ButtonBar) HandleEvent(e ui.Event) bool {
	switch e.Type {
	case ui.EventMouseMove:
		b.hovered = b.hitTest(e.X, e.Y)
		return b.hovered >= 0

	case ui.EventMouseDown:
		switch b.hitTest(e.X, e.Y) {
		case 0: // decrement
			b.history.Push(b.count.Get() - 1)
			b.count.Set(b.count.Get() - 1)
		case 1: // increment
			b.history.Push(b.count.Get() + 1)
			b.count.Set(b.count.Get() + 1)
		case 2: // reset
			b.history.Push(0)
			b.count.Set(0)
		case 3: // undo
			if !b.history.Undo() {
				log.Println("[counter] nothing to undo")
			} else {
				b.count.Set(b.history.Get())
			}
		}
		return true

	case ui.EventKeyDown:
		switch e.Key {
		case "ArrowUp", "+":
			b.history.Push(b.count.Get() + 1)
			b.count.Set(b.count.Get() + 1)
			return true
		case "ArrowDown", "-":
			b.history.Push(b.count.Get() - 1)
			b.count.Set(b.count.Get() - 1)
			return true
		case "r", "R":
			b.history.Push(0)
			b.count.Set(0)
			return true
		case "z", "Z":
			if b.history.Undo() {
				b.count.Set(b.history.Get())
			}
			return true
		}
	}
	return false
}

func (b *ButtonBar) hitTest(mx, my float32) int {
	for i, r := range b.btnBounds {
		if mx >= r.X && mx <= r.X+r.W && my >= r.Y && my <= r.Y+r.H {
			return i
		}
	}
	return -1
}

func (b *ButtonBar) Draw(c *canvas.Canvas, x, y, width, height float32) {
	b.bounds = canvas.Rect{X: x, Y: y, W: width, H: height}

	labels := []string{"−", "+", "Reset", "Undo"}
	colors := []canvas.Color{
		{R: 0.95, G: 0.36, B: 0.42, A: 1}, // red  −
		{R: 0.56, G: 0.93, B: 0.56, A: 1}, // green +
		{R: 0.54, G: 0.74, B: 0.98, A: 1}, // blue  Reset
		{R: 0.80, G: 0.70, B: 0.98, A: 1}, // mauve Undo
	}

	btnW := float32(90)
	btnH := float32(44)
	gap := float32(16)
	total := float32(len(labels))*btnW + float32(len(labels)-1)*gap
	startX := x + (width-total)/2
	cy := y + (height-btnH)/2

	for i, label := range labels {
		bx := startX + float32(i)*(btnW+gap)
		b.btnBounds[i] = canvas.Rect{X: bx, Y: cy, W: btnW, H: btnH}

		col := colors[i]
		alpha := float32(0.18)
		if b.hovered == i {
			alpha = 0.35
		}
		bg := canvas.Color{R: col.R, G: col.G, B: col.B, A: alpha}
		c.DrawRoundedRect(bx, cy, btnW, btnH, 10, canvas.FillPaint(bg))

		// Border
		border := canvas.Color{R: col.R, G: col.G, B: col.B, A: 0.6}
		c.DrawRoundedRect(bx, cy, btnW, btnH, 10, canvas.StrokePaint(border, 1.5))

		// Label
		ts := canvas.TextStyle{Color: col, Size: 16}
		sz := c.MeasureText(label, ts)
		c.DrawText(bx+(btnW-sz.W)/2, cy+(btnH+sz.H)/2-4, label, ts)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CounterApp — wires everything together
// ─────────────────────────────────────────────────────────────────────────────

type CounterApp struct {
	count   *state.Signal[int]
	history *state.History[int]

	counterWidget *CounterWidget
	buttonBar     *ButtonBar

	root     ui.Component
	overlays []ui.Component
}

func NewCounterApp() *CounterApp {
	count := state.New(50)
	history := state.NewHistory(0, 50)

	// Subscribe to log every change
	count.Subscribe(func(v int) {
		log.Printf("[state] count → %d", v)
	})

	cw := NewCounterWidget(count)
	bb := NewButtonBar(count, history)

	return &CounterApp{
		count:         count,
		history:       history,
		counterWidget: cw,
		buttonBar:     bb,
	}
}

// Root returns the top-level component (used by debug wrapper).
func (a *CounterApp) Root() ui.Component          { return a.counterWidget }
func (a *CounterApp) CounterWidget() ui.Component { return a.counterWidget }
func (a *CounterApp) ButtonBar() ui.Component     { return a.buttonBar }

// SetOverlays attaches devtools overlays (only called in debug builds).
func (a *CounterApp) SetOverlays(overlays ...ui.Component) {
	a.overlays = overlays
}

// Run starts the render loop (stubbed here — replace with real window creation).
func (a *CounterApp) Run() {
	log.Println("[app] counter app running — press +/- or arrow keys, R to reset, Z to undo")

	select {}
}

// ─────────────────────────────────────────────────────────────────────────────
// FixedPosition — a simple wrapper that forces a component into a specific rect
// ─────────────────────────────────────────────────────────────────────────────

type FixedPosition struct {
	Child  ui.Component
	rect   canvas.Rect
	bounds canvas.Rect
}

func NewFixedPosition(child ui.Component, x, y, w, h float32) *FixedPosition {
	return &FixedPosition{Child: child, rect: canvas.Rect{X: x, Y: y, W: w, H: h}}
}

func (f *FixedPosition) Bounds() canvas.Rect { return f.bounds }
func (f *FixedPosition) Tick(delta float64) {
	if f.Child != nil {
		f.Child.Tick(delta)
	}
}

func (f *FixedPosition) HandleEvent(e ui.Event) bool {
	if f.Child != nil {
		return f.Child.HandleEvent(e)
	}
	return false
}

func (f *FixedPosition) Draw(c *canvas.Canvas, _, _, _, _ float32) {
	r := f.rect
	f.bounds = r
	if f.Child != nil {
		f.Child.Draw(c, r.X, r.Y, r.W, r.H)
	}
}

type ClearRect struct {
	Color  canvas.Color
	bounds canvas.Rect
}

func NewClearRect(col canvas.Color) *ClearRect {
	return &ClearRect{Color: col}
}

func (c *ClearRect) Bounds() canvas.Rect         { return c.bounds }
func (c *ClearRect) Tick(_ float64)              {}
func (c *ClearRect) HandleEvent(_ ui.Event) bool { return false }
func (c *ClearRect) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	c.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	cv.DrawRect(x, y, w, h, canvas.FillPaint(c.Color))
}
