package main

import (
	"fmt"
	"math"
	"time"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/animation"
	"github.com/achiket123/gui-go/canvas"
)

func main() {
	const W, H = 900, 650

	w := goui.NewWindow("goui v2 — Shapes Demo", W, H)

	// AnimationController driving a rotation angle.
	rotCtrl := animation.NewController(3 * time.Second)
	rotCtrl.PingPong()
	rotCtrl.Forward()

	// Timeline: card slides in from left while fading in.
	tl := animation.NewTimeline(800 * time.Millisecond)
	tl.AddTrack("cardX", -300, 50, 0.0, 0.8, animation.EaseOutBack)
	tl.AddTrack("alpha", 0, 1.0, 0.0, 0.6, animation.EaseOutQuad)
	tl.Play()

	w.AddController(rotCtrl)
	w.AddTimeline(tl)

	angle := 0.0
	frameN := 0

	w.OnDrawGL(func(c *canvas.Canvas) {
		frameN++

		// Background.
		c.Clear(canvas.RGB8(12, 13, 20))

		// --- Gradient panel (top-left) ---
		gp := canvas.GradientPaint(canvas.Hex("#6366F1"), canvas.Hex("#EC4899"))
		gp.LinearGrad.From = canvas.Point{X: 30, Y: 30}
		gp.LinearGrad.To = canvas.Point{X: 330, Y: 30}
		c.DrawRoundedRect(30, 30, 300, 160, 20, gp)

		c.DrawText(50, 90, "goui v2", canvas.TextStyle{Color: canvas.White, Size: 36})
		c.DrawText(50, 130, "GPU-accelerated 2D canvas", canvas.TextStyle{Color: canvas.RGBA8(255, 255, 255, 180), Size: 14})
		c.DrawText(50, 160, fmt.Sprintf("Frame %d", frameN), canvas.TextStyle{Color: canvas.RGBA8(255, 255, 255, 120), Size: 12})

		// --- Spinning polygon (right side) ---
		angle = rotCtrl.Drive(animation.NewTween(0, math.Pi*2, 0, animation.Linear))
		cx, cy := float32(700), float32(200)
		c.Save()
		c.RotateAround(cx, cy, float32(angle))
		poly := makeStarPoints(cx, cy, 90, 45, 6)
		c.DrawPolygon(poly, canvas.FillPaint(canvas.Hex("#F59E0B")))
		c.Restore()

		// --- Circles row ---
		colors := []canvas.Color{
			canvas.Hex("#EF4444"), canvas.Hex("#3B82F6"),
			canvas.Hex("#10B981"), canvas.Hex("#8B5CF6"),
		}
		for i, col := range colors {
			cx2 := float32(50 + i*90)
			c.DrawCircle(cx2, 280, 35, canvas.FillPaint(col))
		}

		// --- Rounded rects row ---
		for i := 0; i < 4; i++ {
			rx := float32(30 + i*200)
			alpha := float32(0.4 + float32(i)*0.15)
			c.DrawRoundedRect(rx, 350, 170, 80, float32(i*8+4),
				canvas.FillPaint(canvas.Hex("#1E40AF").WithAlpha(alpha)))
		}

		// --- Sliding card (timeline driven) ---
		cardX := float32(tl.Value("cardX"))
		cardAlpha := float32(tl.Value("alpha"))
		c.DrawRoundedRect(cardX, 460, 280, 140, 16,
			canvas.FillPaint(canvas.Hex("#0F172A").WithAlpha(cardAlpha)))
		c.DrawRoundedRect(cardX, 460, 280, 140, 16,
			canvas.StrokePaint(canvas.Hex("#334155"), 1.5))
		c.DrawText(cardX+20, 510, "Slide-in card", canvas.TextStyle{
			Color: canvas.White.WithAlpha(cardAlpha), Size: 16})
		c.DrawText(cardX+20, 540, "driven by Timeline", canvas.TextStyle{
			Color: canvas.RGBA8(148, 163, 184, 200).WithAlpha(cardAlpha), Size: 13})

		// --- Gradient strip at bottom ---
		gp2 := canvas.GradientPaint(canvas.Hex("#6366F1"), canvas.Hex("#EC4899"))
		gp2.LinearGrad.From = canvas.Point{X: 0, Y: 0}
		gp2.LinearGrad.To = canvas.Point{X: float32(W), Y: 0}
		c.DrawRect(0, float32(H-6), float32(W), 6, gp2)
	})

	w.OnKeyPress(func(e goui.KeyEvent) {
		if e.KeySym == "Escape" {
			w.Close()
		}
	})

	w.Show()
	fmt.Println("Window closed.")
}

// makeStarPoints generates a star/polygon with alternating outer/inner radii.
func makeStarPoints(cx, cy, outerR, innerR float32, points int) []canvas.Point {
	total := points * 2
	pts := make([]canvas.Point, total)
	for i := 0; i < total; i++ {
		angle := -math.Pi/2 + float64(i)*math.Pi/float64(points)
		r := outerR
		if i%2 == 1 {
			r = innerR
		}
		pts[i] = canvas.Point{
			X: cx + r*float32(math.Cos(angle)),
			Y: cy + r*float32(math.Sin(angle)),
		}
	}
	return pts
}
