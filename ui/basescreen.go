package ui

import "github.com/achiket/gui-go/canvas"

// BaseScreen provides no-op implementations of OnEnter/OnLeave/Bounds/Tick
// so concrete screens only override what they need.
type BaseScreen struct {
	Nav    *Navigator // set automatically by OnEnter
	bounds canvas.Rect
}

func (b *BaseScreen) OnEnter(nav *Navigator)  { b.Nav = nav }
func (b *BaseScreen) OnLeave()                {}
func (b *BaseScreen) Bounds() canvas.Rect     { return b.bounds }
func (b *BaseScreen) Tick(_ float64)          {}
func (b *BaseScreen) SetBounds(r canvas.Rect) { b.bounds = r }
