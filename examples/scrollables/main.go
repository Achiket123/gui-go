// examples/scroll/main.go
package main

import (
	"fmt"

	goui "github.com/achiket/gui-go"
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

// ── palette ───────────────────────────────────────────────────────────────

var (
	colBase    = canvas.Hex("#1E1E2E")
	colSurface = canvas.Hex("#181825")
	colOverlay = canvas.Hex("#313244")
	colText    = canvas.Hex("#CDD6F4")
	colSubtext = canvas.Hex("#A6ADC8")
	colAccent  = canvas.Hex("#89B4FA")
	colGreen   = canvas.Hex("#A6E3A1")
	colRed     = canvas.Hex("#F38BA8")
	colYellow  = canvas.Hex("#F9E2AF")
	colPink    = canvas.Hex("#F5C2E7")
)

const (
	winW    = 720
	winH    = 600
	itemH   = 72
	itemGap = 4
	itemPad = 12
)

// ── data ──────────────────────────────────────────────────────────────────

type Item struct {
	Title    string
	Subtitle string
	Tag      string
	TagColor canvas.Color
}

func makeItems() []Item {
	tags := []struct {
		label string
		color canvas.Color
	}{
		{"feature", colGreen},
		{"bug", colRed},
		{"docs", colYellow},
		{"refactor", colAccent},
		{"test", colPink},
	}
	items := make([]Item, 40)
	for i := range items {
		t := tags[i%len(tags)]
		items[i] = Item{
			Title:    fmt.Sprintf("Item %02d — %s", i+1, t.label),
			Subtitle: fmt.Sprintf("Subtitle for item %d. Scroll down to see more.", i+1),
			Tag:      t.label,
			TagColor: t.color,
		}
	}
	return items
}

// ── item renderer ─────────────────────────────────────────────────────────

func makeDrawItems(items []Item) func(c *canvas.Canvas, x, y, w, h float32) {
	titleStyle := canvas.TextStyle{Color: colText, Size: 15}
	subStyle := canvas.TextStyle{Color: colSubtext, Size: 12}
	tagStyle := canvas.TextStyle{Color: colBase, Size: 11}

	return func(c *canvas.Canvas, x, y, w, _ float32) {
		for i, item := range items {
			iy := y + float32(i)*(itemH+itemGap)

			bg := colSurface
			if i%2 == 0 {
				bg = colBase
			}
			c.DrawRoundedRect(x+itemPad, iy+2, w-itemPad*2, itemH-4, 8, canvas.FillPaint(bg))
			c.DrawRoundedRect(x+itemPad, iy+2, 3, itemH-4, 2, canvas.FillPaint(item.TagColor))
			c.DrawText(x+itemPad+16, iy+itemH/2-4, item.Title, titleStyle)
			c.DrawText(x+itemPad+16, iy+itemH/2+14, item.Subtitle, subStyle)

			tagSz := c.MeasureText(item.Tag, tagStyle)
			pillW := tagSz.W + 16
			pillX := x + w - float32(itemPad)*2 - pillW
			pillY := iy + itemH/2 - 9
			c.DrawRoundedRect(pillX, pillY, pillW, 18, 9, canvas.FillPaint(item.TagColor))
			c.DrawText(pillX+8, pillY+13, item.Tag, tagStyle)
		}
	}
}

// ── main ──────────────────────────────────────────────────────────────────

func main() {
	w := goui.NewWindow("goui — Scroll Example", winW, winH)

	items := makeItems()
	contentH := float32(len(items)) * (itemH + itemGap)

	scroll := ui.NewScrollView(contentH, makeDrawItems(items))
	w.AddComponent(scroll)

	headerStyle := canvas.TextStyle{Color: colText, Size: 18}
	hintStyle := canvas.TextStyle{Color: colSubtext, Size: 12}

	w.OnDrawGL(func(c *canvas.Canvas) {
		cw, ch := c.Width(), c.Height()

		c.DrawRect(0, 0, cw, ch, canvas.FillPaint(colBase))

		c.DrawRect(0, 0, cw, 56, canvas.FillPaint(colSurface))
		c.DrawText(20, 35, "Scroll Demo", headerStyle)

		hint := fmt.Sprintf("%d items  ·  wheel / drag scrollbar / arrow keys", len(items))
		c.DrawText(cw-c.MeasureText(hint, hintStyle).W-20, 35, hint, hintStyle)

		c.DrawRect(0, 56, cw, 1, canvas.FillPaint(colOverlay))

		scroll.Draw(c, 0, 57, cw, ch-57)

		if scroll.MaxScroll() > 0 {
			pct := int(scroll.ScrollFraction() * 100)
			label := fmt.Sprintf("%d%%", pct)
			ls := canvas.TextStyle{Color: colSubtext, Size: 11}
			c.DrawText(cw-c.MeasureText(label, ls).W-float32(ui.ScrollbarWidth)-14, 42, label, ls)
		}
	})

	w.Show()
}
