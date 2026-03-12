package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/animation"
	"github.com/achiket123/gui-go/canvas"
)

const (
	W = 900
	H = 650
)

func findFont(name string) string {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		os.Getenv("HOME") + "/.fonts",
	}
	for _, dir := range dirs {
		for _, pat := range []string{
			filepath.Join(dir, "**", "*.ttf"),
			filepath.Join(dir, "*.ttf"),
			filepath.Join(dir, "**", "**", "*.ttf"),
		} {
			matches, _ := filepath.Glob(pat)
			for _, p := range matches {
				if strings.Contains(strings.ToLower(filepath.Base(p)), strings.ToLower(name)) {
					return p
				}
			}
		}
	}
	return ""
}

func main() {
	w := goui.NewWindow("goui — Animation Showcase", W, H)

	// ── Font ──────────────────────────────────────────────────────────────
	fontPath := ""
	for _, name := range []string{"DejaVuSans.ttf", "LiberationSans-Regular", "FreeSans", "Ubuntu-R", "Arial"} {
		if p := findFont(name); p != "" {
			fontPath = p
			break
		}
	}
	if fontPath != "" {
		fmt.Println("Font:", fontPath)
	}
	txt := func(col canvas.Color, size float32) canvas.TextStyle {
		return canvas.TextStyle{Color: col, Size: size, FontPath: fontPath}
	}

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 1 — Orbiting particles
	// AnimationController, Repeat(-1) = infinite, drives angle 0→2π
	// ══════════════════════════════════════════════════════════════════════
	orbitCtrl := animation.NewController(3 * time.Second)
	orbitCtrl.Repeat(-1)
	orbitCtrl.Forward()
	w.AddController(orbitCtrl)

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 2 — Pulsing centre circle
	// PingPong oscillates 0→1→0 forever (breathe effect)
	// ══════════════════════════════════════════════════════════════════════
	pulseCtrl := animation.NewController(900 * time.Millisecond)
	pulseCtrl.PingPong()
	pulseCtrl.Forward()
	w.AddController(pulseCtrl)

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 3 — Sine wave scroll
	// Continuous loop, drives horizontal phase shift
	// ══════════════════════════════════════════════════════════════════════
	waveCtrl := animation.NewController(2 * time.Second)
	waveCtrl.Repeat(-1)
	waveCtrl.Forward()
	w.AddController(waveCtrl)

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 4 — Spinning star (RotateAround now works!)
	// ══════════════════════════════════════════════════════════════════════
	spinCtrl := animation.NewController(4 * time.Second)
	spinCtrl.Repeat(-1)
	spinCtrl.Forward()
	w.AddController(spinCtrl)

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 5 — Card cascade (Timeline, plays once, R to replay)
	// ══════════════════════════════════════════════════════════════════════
	cardTL := animation.NewTimeline(1200 * time.Millisecond)
	cardTL.AddTrack("c0x", float64(W+20), 540, 0.00, 0.50, animation.EaseOutBack)
	cardTL.AddTrack("c1x", float64(W+20), 540, 0.12, 0.62, animation.EaseOutBack)
	cardTL.AddTrack("c2x", float64(W+20), 540, 0.24, 0.74, animation.EaseOutBack)
	cardTL.AddTrack("c0a", 0, 1, 0.00, 0.25, animation.EaseOutQuad)
	cardTL.AddTrack("c1a", 0, 1, 0.12, 0.37, animation.EaseOutQuad)
	cardTL.AddTrack("c2a", 0, 1, 0.24, 0.49, animation.EaseOutQuad)
	cardTL.Play()
	w.AddTimeline(cardTL)

	// ══════════════════════════════════════════════════════════════════════
	// ANIMATION 6 — Progress bar (Timeline, Space to replay)
	// ══════════════════════════════════════════════════════════════════════
	const barW float32 = W - 80
	progressTL := animation.NewTimeline(3 * time.Second)
	progressTL.AddTrack("fill", 0, float64(barW), 0.0, 1.0, animation.EaseInOutCubic)
	progressTL.AddTrack("glow", 0, 1, 0.0, 0.2, animation.EaseOutQuad)
	progressTL.Play()
	w.AddTimeline(progressTL)

	// ══════════════════════════════════════════════════════════════════════
	// Tweens — pure range mappers, driven by controllers above
	// ══════════════════════════════════════════════════════════════════════
	angleTween := animation.NewTween(0, math.Pi*2, 0, animation.Linear)
	pulseRTween := animation.NewTween(40.0, 58.0, 0, animation.EaseInOutQuad)
	spinTween := animation.NewTween(0, math.Pi*2, 0, animation.Linear)

	// Orbit particle colors
	pColors := []canvas.Color{
		canvas.Hex("#6366F1"), canvas.Hex("#EC4899"),
		canvas.Hex("#F59E0B"), canvas.Hex("#10B981"),
		canvas.Hex("#3B82F6"), canvas.Hex("#EF4444"),
		canvas.Hex("#8B5CF6"), canvas.Hex("#06B6D4"),
	}

	// Card data
	type card struct {
		title, sub string
		accent     canvas.Color
	}
	cards := []card{
		{"AnimationController", "Drives 0 → 1 over duration", canvas.Hex("#6366F1")},
		{"Timeline", "Multi-track keyframe sequencing", canvas.Hex("#EC4899")},
		{"Tween + EaseOutBack", "Overshoot & settle on arrival", canvas.Hex("#F59E0B")},
	}

	// Star polygon helper
	starPoints := func(cx, cy, r1, r2 float32, n int) []canvas.Point {
		pts := make([]canvas.Point, n*2)
		for i := 0; i < n*2; i++ {
			a := -math.Pi/2 + float64(i)*math.Pi/float64(n)
			r := r1
			if i%2 == 1 {
				r = r2
			}
			pts[i] = canvas.Point{
				X: cx + r*float32(math.Cos(a)),
				Y: cy + r*float32(math.Sin(a)),
			}
		}
		return pts
	}

	frame := 0

	w.OnDrawGL(func(c *canvas.Canvas) {
		frame++

		// ── Background ────────────────────────────────────────────────────
		c.DrawRect(0, 0, W, H, canvas.FillPaint(canvas.RGB8(10, 11, 18)))

		// Subtle dot grid
		for x := float32(0); x < W; x += 40 {
			for y := float32(0); y < H; y += 40 {
				c.DrawCircle(x, y, 1, canvas.FillPaint(canvas.RGBA8(255, 255, 255, 12)))
			}
		}

		// ── Title bar ─────────────────────────────────────────────────────
		c.DrawRect(0, 0, W, 48, canvas.FillPaint(canvas.RGBA8(14, 14, 26, 240)))
		c.DrawLine(0, 48, W, 48, canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 15), 1))

		title := "goui — Animation Showcase"
		tsz := c.MeasureText(title, txt(canvas.White, 15))
		c.DrawText(float32(W)/2-tsz.W/2, 31, title, txt(canvas.White, 15))

		fstr := fmt.Sprintf("frame %d", frame)
		fsz := c.MeasureText(fstr, txt(canvas.RGBA8(55, 65, 85, 255), 11))
		c.DrawText(float32(W)-fsz.W-14, 31, fstr, txt(canvas.RGBA8(55, 65, 85, 255), 11))

		hint := "R = replay cards   Space = replay bar   Escape = quit"
		hsz := c.MeasureText(hint, txt(canvas.RGBA8(55, 65, 85, 255), 10))
		c.DrawText(float32(W)/2-hsz.W/2, float32(H)-10, hint, txt(canvas.RGBA8(55, 65, 85, 255), 10))

		// ══════════════════════════════════════════════════════════════════
		// LEFT PANEL — Orbit + Pulse   (x: 0..450, y: 48..450)
		// ══════════════════════════════════════════════════════════════════
		const ocx, ocy float32 = 225, 260
		const orbitR float32 = 110

		// Glow halo behind orbit
		c.DrawCircle(ocx, ocy, orbitR+50,
			canvas.FillPaint(canvas.Hex("#6366F1").WithAlpha(0.05)))
		// Orbit track
		c.DrawCircle(ocx, ocy, orbitR,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 15), 1))
		// Inner ring
		c.DrawCircle(ocx, ocy, orbitR*0.6,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 8), 1))

		baseAngle := orbitCtrl.Drive(angleTween)
		for i, col := range pColors {
			a := baseAngle + float64(i)*math.Pi*2/float64(len(pColors))
			px := ocx + orbitR*float32(math.Cos(a))
			py := ocy + orbitR*float32(math.Sin(a))
			// Trail glow
			c.DrawCircle(px, py, 12, canvas.FillPaint(col.WithAlpha(0.2)))
			// Core particle
			c.DrawCircle(px, py, 6, canvas.FillPaint(col))
			// Inner glow dot
			c.DrawCircle(px, py, 2, canvas.FillPaint(canvas.White.WithAlpha(0.8)))

			// Inner orbit particles (half radius, opposite phase)
			a2 := baseAngle*1.7 + float64(i)*math.Pi*2/float64(len(pColors))
			px2 := ocx + orbitR*0.6*float32(math.Cos(a2))
			py2 := ocy + orbitR*0.6*float32(math.Sin(a2))
			c.DrawCircle(px2, py2, 3, canvas.FillPaint(col.WithAlpha(0.6)))
		}

		// Pulsing centre
		pulseR := float32(pulseCtrl.Drive(pulseRTween))
		c.DrawCircle(ocx, ocy, pulseR+10,
			canvas.FillPaint(canvas.Hex("#6366F1").WithAlpha(0.15)))
		c.DrawCircle(ocx, ocy, pulseR,
			canvas.FillPaint(canvas.Hex("#6366F1")))
		c.DrawCircle(ocx, ocy, pulseR*0.55,
			canvas.FillPaint(canvas.Hex("#818CF8")))
		lsz := c.MeasureText("goui", txt(canvas.White, 13))
		c.DrawText(ocx-lsz.W/2, ocy+5, "goui", txt(canvas.White, 13))

		// Section label
		sl1 := "AnimationController · PingPong"
		sl1sz := c.MeasureText(sl1, txt(canvas.RGBA8(100, 116, 139, 255), 11))
		c.DrawText(ocx-sl1sz.W/2, ocy+orbitR+28, sl1, txt(canvas.RGBA8(100, 116, 139, 255), 11))

		// ══════════════════════════════════════════════════════════════════
		// LEFT BOTTOM — Spinning star (uses RotateAround — GPU transform)
		// ══════════════════════════════════════════════════════════════════
		const scx, scy float32 = 225, 520
		spinAngle := float32(spinCtrl.Drive(spinTween))

		c.DrawCircle(scx, scy, 55, canvas.FillPaint(canvas.RGBA8(255, 255, 255, 4)))

		c.Save()
		c.RotateAround(scx, scy, spinAngle)
		pts := starPoints(scx, scy, 44, 20, 6)
		c.DrawPolygon(pts, canvas.FillPaint(canvas.Hex("#F59E0B")))
		c.DrawPolygon(pts, canvas.StrokePaint(canvas.Hex("#FCD34D"), 1.5))
		c.Restore()

		c.Save()
		c.RotateAround(scx, scy, -spinAngle*1.4)
		pts2 := starPoints(scx, scy, 28, 14, 4)
		c.DrawPolygon(pts2, canvas.FillPaint(canvas.Hex("#EC4899")))
		c.Restore()

		c.DrawCircle(scx, scy, 8, canvas.FillPaint(canvas.White))

		sl2 := "RotateAround · GPU transform"
		sl2sz := c.MeasureText(sl2, txt(canvas.RGBA8(100, 116, 139, 255), 11))
		c.DrawText(scx-sl2sz.W/2, scy+62, sl2, txt(canvas.RGBA8(100, 116, 139, 255), 11))

		// ══════════════════════════════════════════════════════════════════
		// RIGHT TOP — Sine wave (waveCtrl scrolls phase)
		// ══════════════════════════════════════════════════════════════════
		const wl, wr, wcy float32 = 470, W - 20, 190
		const wamp float32 = 60
		const wfreq = 2.8
		ww := wr - wl

		// Panel
		c.DrawRoundedRect(wl-8, wcy-wamp-24, ww+16, wamp*2+48, 10,
			canvas.FillPaint(canvas.RGBA8(255, 255, 255, 4)))
		c.DrawRoundedRect(wl-8, wcy-wamp-24, ww+16, wamp*2+48, 10,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 12), 1))

		shift := float32(waveCtrl.Value())
		segs := 100
		for i := 0; i < segs; i++ {
			t0 := float32(i) / float32(segs)
			t1 := float32(i+1) / float32(segs)
			x0 := wl + t0*ww
			x1 := wl + t1*ww
			ph0 := (t0*wfreq - shift) * math.Pi * 2
			ph1 := (t1*wfreq - shift) * math.Pi * 2
			y0 := wcy + wamp*float32(math.Sin(float64(ph0)))
			y1 := wcy + wamp*float32(math.Sin(float64(ph1)))
			col := canvas.Lerp(canvas.Hex("#6366F1"), canvas.Hex("#EC4899"), t0)
			c.DrawLine(x0, y0, x1, y1, canvas.StrokePaint(col, 2.5))
		}

		// Centre axis
		c.DrawLine(wl, wcy, wr, wcy,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 18), 1))

		// Travelling dot
		dotX := wl + float32(waveCtrl.Value())*ww
		dotPh := (float32(waveCtrl.Value())*wfreq - shift) * math.Pi * 2
		dotY := wcy + wamp*float32(math.Sin(float64(dotPh)))
		c.DrawCircle(dotX, dotY, 8, canvas.FillPaint(canvas.White.WithAlpha(0.3)))
		c.DrawCircle(dotX, dotY, 5, canvas.FillPaint(canvas.White))

		wlbl := "Tween — scrolling sine wave"
		wlsz := c.MeasureText(wlbl, txt(canvas.RGBA8(100, 116, 139, 255), 11))
		c.DrawText(wl+ww/2-wlsz.W/2, wcy+wamp+18, wlbl, txt(canvas.RGBA8(100, 116, 139, 255), 11))

		// ══════════════════════════════════════════════════════════════════
		// RIGHT MIDDLE — Cascade cards (Timeline)
		// ══════════════════════════════════════════════════════════════════
		cxKeys := []string{"c0x", "c1x", "c2x"}
		caKeys := []string{"c0a", "c1a", "c2a"}
		const cardCW, cardCH float32 = 330, 62

		for i, cd := range cards {
			cx2 := float32(cardTL.Value(cxKeys[i]))
			ca := float32(cardTL.Value(caKeys[i]))
			cy2 := float32(305 + i*74)

			// Shadow
			c.DrawRoundedRect(cx2+3, cy2+4, cardCW, cardCH, 10,
				canvas.FillPaint(canvas.RGBA8(0, 0, 0, 70).WithAlpha(ca)))
			// Body
			c.DrawRoundedRect(cx2, cy2, cardCW, cardCH, 10,
				canvas.FillPaint(canvas.RGBA8(18, 19, 32, 230).WithAlpha(ca)))
			// Accent bar
			c.DrawRoundedRect(cx2, cy2, 4, cardCH, 4,
				canvas.FillPaint(cd.accent.WithAlpha(ca)))
			// Border
			c.DrawRoundedRect(cx2, cy2, cardCW, cardCH, 10,
				canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 18).WithAlpha(ca), 1))
			// Icon dot
			c.DrawCircle(cx2+26, cy2+cardCH/2, 7,
				canvas.FillPaint(cd.accent.WithAlpha(ca)))
			// Text
			c.DrawText(cx2+44, cy2+24, cd.title,
				txt(canvas.White.WithAlpha(ca), 13))
			c.DrawText(cx2+44, cy2+42, cd.sub,
				txt(canvas.RGBA8(148, 163, 184, 255).WithAlpha(ca), 11))
		}

		clbl := "Timeline — cascade slide-in"
		clsz := c.MeasureText(clbl, txt(canvas.RGBA8(100, 116, 139, 255), 11))
		c.DrawText(540+cardCW/2-clsz.W/2, 530, clbl, txt(canvas.RGBA8(100, 116, 139, 255), 11))

		// ══════════════════════════════════════════════════════════════════
		// BOTTOM — Progress bar (Timeline + EaseInOutCubic)
		// ══════════════════════════════════════════════════════════════════
		const barX, barY, barH float32 = 40, H - 52, 16
		fill := float32(progressTL.Value("fill"))
		glow := float32(progressTL.Value("glow"))
		pct := int(fill / barW * 100)

		// Track
		c.DrawRoundedRect(barX, barY, barW, barH, barH/2,
			canvas.FillPaint(canvas.RGBA8(255, 255, 255, 12)))

		// Filled portion
		if fill > barH {
			fillPaint := canvas.GradientPaint(canvas.Hex("#6366F1"), canvas.Hex("#EC4899"))
			fillPaint.LinearGrad.From = canvas.Point{X: barX, Y: 0}
			fillPaint.LinearGrad.To = canvas.Point{X: barX + barW, Y: 0}
			c.DrawRoundedRect(barX, barY, fill, barH, barH/2, fillPaint)
			// Shine strip
			c.DrawRoundedRect(barX+2, barY+2, fill-4, barH/2-2, barH/4,
				canvas.FillPaint(canvas.RGBA8(255, 255, 255, 30)))
			// Leading glow
			c.DrawCircle(barX+fill, barY+barH/2, barH*0.9,
				canvas.FillPaint(canvas.Hex("#818CF8").WithAlpha(glow*0.6)))
		}

		// Border
		c.DrawRoundedRect(barX, barY, barW, barH, barH/2,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 18), 1))

		// Labels
		blbl := "Timeline · EaseInOutCubic progress"
		c.DrawText(barX, barY-16, blbl,
			txt(canvas.RGBA8(100, 116, 139, 255), 11))
		ps := fmt.Sprintf("%d%%", pct)
		psz := c.MeasureText(ps, txt(canvas.White, 11))
		c.DrawText(barX+barW-psz.W, barY-16, ps,
			txt(canvas.White, 11))

		if !progressTL.IsPlaying() {
			rp := "[ Space ] to replay"
			rpsz := c.MeasureText(rp, txt(canvas.Hex("#818CF8"), 11))
			c.DrawText(float32(W)/2-rpsz.W/2, barY+barH+14, rp,
				txt(canvas.Hex("#818CF8"), 11))
		}
	})

	// ── Keys ──────────────────────────────────────────────────────────────
	w.OnKeyPress(func(e goui.KeyEvent) {
		switch e.KeySym {
		case "Escape":
			w.Close()
		case "space":
			progressTL.Seek(0)
			progressTL.Play()
		case "r", "R":
			cardTL.Seek(0)
			cardTL.Play()
		}
	})

	w.OnClose(func() { fmt.Println("closed.") })
	w.Show()
}
