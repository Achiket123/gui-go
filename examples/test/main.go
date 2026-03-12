package main

import (
	"fmt"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/state"
	"github.com/achiket123/gui-go/ui"
)

type Counter struct {
	count  *state.Signal[int]
	bounds canvas.Rect
}

func (c *Counter) Bounds() canvas.Rect { return c.bounds }
func (c *Counter) Tick(_ float64)      {}
func (c *Counter) HandleEvent(e ui.Event) bool {
	if e.Type == ui.EventMouseDown {
		c.count.Update(func(v int) int { return v + 1 })
		return true
	}
	return false
}
func (c *Counter) Draw(cv *canvas.Canvas, x, y, w, h float32) {
	c.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	label := fmt.Sprintf("%d", c.count.Get())
	ts := canvas.TextStyle{Color: canvas.White, Size: 48}
	sz := cv.MeasureText(label, ts)
	cv.DrawText(x+(w-sz.W)/2, y+(h+sz.H)/2, label, ts)
}

func main() {
	count := state.New(0)
	win := goui.NewWindow("Counter", 480, 360)
	win.AddComponent(&Counter{count: count})
	win.Show()
}
