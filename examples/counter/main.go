package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/animation"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
	"github.com/achiket123/gui-go/ui/layout"
)

const (
	W = 400
	H = 600
)

// findFont searches common system font directories for a TTF by name fragment.
func findFont(name string) string {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		os.Getenv("HOME") + "/.fonts",
	}
	for _, dir := range dirs {
		patterns := []string{
			filepath.Join(dir, "**", "*.ttf"),
			filepath.Join(dir, "*.ttf"),
			filepath.Join(dir, "**", "**", "*.ttf"),
		}
		for _, pat := range patterns {
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
	w := goui.NewWindow("Counter App", W, H)

	fontPath := ""
	for _, name := range []string{"DejaVuSans", "LiberationSans", "FreeSans", "Arial", "Ubuntu"} {
		if p := findFont(name); p != "" {
			fontPath = p
			break
		}
	}
	if fontPath == "" {
		fmt.Fprintln(os.Stderr, "warning: no TTF font found — text will not render")
	} else {
		fmt.Println("Using font:", fontPath)
	}

	txt := func(col canvas.Color, size float32) canvas.TextStyle {
		return canvas.TextStyle{Color: col, Size: size, FontPath: fontPath}
	}

	count := 0

	scaleCtrl := animation.NewController(180 * time.Millisecond)
	scaleTween := animation.NewTween(1.0, 1.18, 0, animation.EaseOutBack)

	triggerBounce := func() {
		scaleCtrl.Reset()
		scaleCtrl.Forward()
	}

	fabStyle := ui.DefaultButtonStyle()
	fabStyle.Background = canvas.Hex("#6366F1")
	fabStyle.HoverColor = canvas.Hex("#818CF8")
	fabStyle.PressColor = canvas.Hex("#4338CA")
	fabStyle.TextStyle = canvas.TextStyle{Color: canvas.White, Size: 28, FontPath: fontPath}
	fabStyle.BorderRadius = 28

	fabBtn := ui.NewButton("+", func() { count++; triggerBounce() })
	fabBtn.Style = fabStyle

	decStyle := ui.DefaultButtonStyle()
	decStyle.Background = canvas.RGBA8(255, 255, 255, 20)
	decStyle.HoverColor = canvas.RGBA8(255, 255, 255, 40)
	decStyle.PressColor = canvas.RGBA8(255, 255, 255, 10)
	decStyle.TextStyle = canvas.TextStyle{Color: canvas.White, Size: 20, FontPath: fontPath}
	decStyle.BorderRadius = 20

	decBtn := ui.NewButton("−", func() { count--; triggerBounce() })
	decBtn.Style = decStyle

	resetStyle := ui.DefaultButtonStyle()
	resetStyle.Background = canvas.RGBA8(0, 0, 0, 0)
	resetStyle.HoverColor = canvas.RGBA8(255, 255, 255, 15)
	resetStyle.PressColor = canvas.RGBA8(255, 255, 255, 5)
	resetStyle.TextStyle = canvas.TextStyle{Color: canvas.RGBA8(148, 163, 184, 255), Size: 13, FontPath: fontPath}
	resetStyle.BorderRadius = 10

	resetBtn := ui.NewButton("Reset", func() { count = 0; triggerBounce() })
	resetBtn.Style = resetStyle

	w.AddController(scaleCtrl)
	w.AddComponent(fabBtn)
	w.AddComponent(decBtn)
	w.AddComponent(resetBtn)

	// ── Draw ───────────────────────────────────────────────────────────────
	w.OnDrawGL(func(c *canvas.Canvas) {

		// Background gradient (top → bottom)
		bgPaint := canvas.GradientPaint(canvas.Hex("#0F0F1A"), canvas.Hex("#1E1B4B"))
		bgPaint.LinearGrad.From = canvas.Point{X: 0, Y: 0}
		bgPaint.LinearGrad.To = canvas.Point{X: 0, Y: H}
		c.DrawRect(0, 0, W, H, bgPaint)

		// Decorative glow circles
		c.DrawCircle(W+60, -60, 160, canvas.FillPaint(canvas.Hex("#6366F1").WithAlpha(0.08)))
		c.DrawCircle(-40, H+40, 140, canvas.FillPaint(canvas.Hex("#818CF8").WithAlpha(0.07)))

		// ── App bar ───────────────────────────────────────────────────────
		c.DrawRoundedRect(0, 0, W, 64, 0,
			canvas.FillPaint(canvas.RGBA8(15, 15, 30, 220)))

		titleStyle := txt(canvas.White, 16)
		titleSize := c.MeasureText("Counter App", titleStyle)
		c.DrawText(float32(W)/2-titleSize.W/2, 42, "Counter App", titleStyle)

		// ── Card ──────────────────────────────────────────────────────────
		const cardX, cardY, cardW, cardH float32 = 40, 110, W - 80, 180

		// Drop shadow
		c.DrawRoundedRect(cardX+4, cardY+6, cardW, cardH, 20,
			canvas.FillPaint(canvas.RGBA8(0, 0, 0, 80)))
		// Card fill
		c.DrawRoundedRect(cardX, cardY, cardW, cardH, 20,
			canvas.FillPaint(canvas.RGBA8(255, 255, 255, 8)))
		// Card border
		c.DrawRoundedRect(cardX, cardY, cardW, cardH, 20,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 20), 1))

		// Card subtitle
		subStyle := txt(canvas.RGBA8(148, 163, 184, 255), 12)
		c.DrawText(cardX+20, cardY+30, "You have pushed the button", subStyle)
		c.DrawText(cardX+20, cardY+48, "this many times:", subStyle)

		// ── Animated count number ─────────────────────────────────────────
		rawT := scaleCtrl.Value()
		pingT := rawT * 2
		if pingT > 1.0 {
			pingT = 2.0 - pingT
		}
		scale := float32(scaleTween.Map(pingT))

		numStr := fmt.Sprintf("%d", count)
		numStyle := txt(canvas.White, 72)
		numSize := c.MeasureText(numStr, numStyle)
		cx := cardX + cardW/2
		cy := cardY + cardH*0.80

		c.Save()
		c.Translate(cx, cy)
		c.Scale(scale, scale)
		c.Translate(-cx, -cy)
		c.DrawText(cx-numSize.W/2, cy, numStr, numStyle)
		c.Restore()

		c.DrawLine(40, 308, float32(W)-40, 308,
			canvas.StrokePaint(canvas.RGBA8(255, 255, 255, 20), 1))

		hintStyle := txt(canvas.RGBA8(100, 116, 139, 255), 12)
		hintText := "Tap + to increment  ·  − to decrement"
		hintSize := c.MeasureText(hintText, hintStyle)
		c.DrawText(float32(W)/2-hintSize.W/2, 332, hintText, hintStyle)

		btnRects := layout.Row(40, 352, W-80, 52, []float32{-1, 90}, 12, layout.AlignStretch)
		decBtn.Draw(c, btnRects[0].X, btnRects[0].Y, btnRects[0].W, btnRects[0].H)
		resetBtn.Draw(c, btnRects[1].X, btnRects[1].Y, btnRects[1].W, btnRects[1].H)

		const fabSize float32 = 64
		fabX := float32(W)/2 - fabSize/2
		fabY := float32(H) - fabSize - 56

		c.DrawCircle(fabX+fabSize/2, fabY+fabSize/2+6,
			fabSize*0.52, canvas.FillPaint(canvas.RGBA8(99, 102, 241, 50)))

		fabBtn.Draw(c, fabX, fabY, fabSize, fabSize)

		footStyle := txt(canvas.RGBA8(51, 65, 85, 255), 11)
		footText := "+ / − keys also work  ·  Escape to quit"
		footSize := c.MeasureText(footText, footStyle)
		c.DrawText(float32(W)/2-footSize.W/2, float32(H)-16, footText, footStyle)
	})

	w.OnKeyPress(func(e goui.KeyEvent) {
		switch e.KeySym {
		case "Escape":
			w.Close()
		case "plus", "equal":
			count++
			triggerBounce()
		case "minus":
			count--
			triggerBounce()
		case "r", "R":
			count = 0
			triggerBounce()
		}
	})

	w.OnClose(func() { fmt.Println("Counter closed.") })
	w.Show()
}
