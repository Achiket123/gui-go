// Demo showcases the goui library:
//   - Colored shape drawing
//   - Bouncing ball with EaseInOutQuad tween
//   - Rotating triangle with a Sequence animation
//   - Mouse click feedback in the title bar
//   - Key press display
//   - Clean close on ✕ or Escape
package main

import (
	"fmt"
	"math"
	"time"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/animation"
)

const (
	winW = 800
	winH = 600
)

func main() {
	w := goui.NewWindow("gui-go Demo", winW, winH)

	// --- Bouncing ball horizontal tween (ping-pong) ---
	ballX := animation.NewTween(60, float64(winW-60), 2000*time.Millisecond, animation.EaseInOutQuad)
	ballX.SetPingPong(true)
	ballX.SetLoop(true)
	w.AddAnimation(ballX)

	// --- Vertical bounce using EaseOutBounce ---
	ballY := animation.NewTween(float64(winH/2-100), float64(winH/2+100), 900*time.Millisecond, animation.EaseOutBounce)
	ballY.SetPingPong(true)
	ballY.SetLoop(true)
	w.AddAnimation(ballY)

	// --- Triangle rotation sequence ---
	// We drive rotation angle with a looping linear tween 0 → 2π
	angle := animation.NewTween(0, math.Pi*2, 3000*time.Millisecond, animation.Linear)
	angle.SetLoop(true)
	w.AddAnimation(angle)

	// --- Track last key press ---
	lastKey := "—"
	w.OnKeyPress(func(e goui.KeyEvent) {
		lastKey = e.KeySym
		if e.KeySym == "Escape" {
			w.Close()
		}
	})

	// --- Mouse click → title bar ---
	w.OnMouseClick(func(e goui.MouseEvent) {
		if e.Type == "press" {
			w.SetTitle(fmt.Sprintf("gui-go Demo  |  click at (%d, %d)  btn=%d", e.X, e.Y, e.Button))
		}
	})

	// --- Draw callback ---
	w.OnDraw(func(c *goui.Canvas) {
		// Background
		c.SetColor(goui.RGB(18, 18, 28))
		c.FillRect(0, 0, c.Width(), c.Height())

		// --- Grid lines ---
		c.SetColor(goui.RGB(35, 35, 55))
		c.SetLineWidth(1)
		for x := 0; x < c.Width(); x += 40 {
			c.DrawLine(x, 0, x, c.Height())
		}
		for y := 0; y < c.Height(); y += 40 {
			c.DrawLine(0, y, c.Width(), y)
		}

		// --- Static colored rectangle ---
		c.SetColor(goui.RGB(60, 100, 200))
		c.SetLineWidth(3)
		c.DrawRect(40, 40, 160, 90)
		c.SetColor(goui.RGB(80, 130, 255))
		c.SetLineWidth(1)
		c.DrawText(48, 92, "gui-go  X11")

		// --- Rotating triangle (drawn from center 200,460) ---
		const cx, cy, r = 200, 460, 60
		a := angle.Value()
		tri := []goui.Point{
			{X: cx + int(r*math.Cos(a)), Y: cy + int(r*math.Sin(a))},
			{X: cx + int(r*math.Cos(a+2.094)), Y: cy + int(r*math.Sin(a+2.094))},
			{X: cx + int(r*math.Cos(a+4.189)), Y: cy + int(r*math.Sin(a+4.189))},
		}
		c.SetColor(goui.RGB(220, 80, 80))
		c.FillPolygon(tri)
		c.SetColor(goui.RGB(255, 120, 120))
		c.SetLineWidth(2)
		c.DrawPolygon(tri)

		// --- Bouncing ball shadow ---
		bx := int(ballX.Value())
		by := int(ballY.Value())
		shadowR := 18 + (by-winH/2+100)/8
		if shadowR < 5 {
			shadowR = 5
		}
		c.SetColor(goui.RGB(10, 10, 20))
		c.FillCircle(bx, winH/2+110, shadowR)

		// --- Bouncing ball ---
		c.SetColor(goui.Orange)
		c.FillCircle(bx, by, 28)
		c.SetColor(goui.Yellow)
		c.FillCircle(bx-8, by-8, 8) // highlight

		// --- Static circles decoration ---
		colors := []goui.Color{
			goui.Cyan, goui.Magenta, goui.Green, goui.Purple, goui.Pink,
		}
		for i, col := range colors {
			cx2 := 580 + i*0
			cy2 := 100 + i*70
			cx2 = 580 + (i%3)*60
			cy2 = 80 + (i/3)*120 + i*60
			c.SetColor(col)
			c.FillCircle(cx2, cy2, 22)
		}

		// --- Last key display ---
		c.SetColor(goui.LightGray)
		c.DrawText(40, winH-40, fmt.Sprintf("Last key: %s", lastKey))
		c.DrawText(40, winH-20, "Click anywhere · Press Escape to quit")

		// --- FPS label (static text) ---
		c.SetColor(goui.White)
		c.DrawText(winW-120, 20, "60 fps target")
	})

	w.OnClose(func() {
		fmt.Println("Window closed — bye!")
	})

	w.Show()
}
