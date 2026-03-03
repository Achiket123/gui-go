package main

import (
	"fmt"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

type libraryItem struct {
	name string
	kind string
	col  canvas.Color
}

// LibraryScreen is a tab — shows a list of items, each pushes a detail screen.
type LibraryScreen struct {
	ui.BaseScreen
	st      *Styles
	items   []libraryItem
	buttons []*ui.Button
	scroll  *ui.ScrollView
}

func NewLibraryScreen(st *Styles) *LibraryScreen {
	s := &LibraryScreen{st: st}

	s.items = []libraryItem{
		{"Alpha Component", "widget", colAccent},
		{"Beta Module", "package", colGreen},
		{"Gamma Renderer", "system", colYellow},
		{"Delta Animator", "widget", colRed},
		{"Epsilon Layout", "package", canvas.Hex("#CBA6F7")},
		{"Zeta FontAtlas", "system", canvas.Hex("#FAB387")},
		{"Eta ScrollView", "widget", colAccent},
		{"Theta Navigator", "package", colGreen},
		{"Iota Transform", "system", colYellow},
		{"Kappa EventBus", "widget", colRed},
	}

	const rowH = float32(64)
	const gap = float32(4)
	contentH := float32(len(s.items)) * (rowH + gap)

	for i, item := range s.items {
		item := item
		i := i
		btn := ui.NewButton(fmt.Sprintf("View %s", item.name), func() {
			s.Nav.Push(NewDetailScreen(st, item.name, "kind: "+item.kind, item.col))
		})
		btnStyle := ui.DefaultButtonStyle()
		btnStyle.Background = item.col
		btnStyle.HoverColor = canvas.Lerp(item.col, canvas.White, 0.15)
		btnStyle.PressColor = canvas.Lerp(item.col, canvas.Black, 0.2)
		btnStyle.TextStyle = canvas.TextStyle{Color: canvas.Hex("#1E1E2E"), Size: 12, FontPath: st.FontPath}
		btnStyle.BorderRadius = 6
		btn.Style = btnStyle
		s.buttons = append(s.buttons, btn)
		_ = i
	}

	s.scroll = ui.NewScrollView(contentH, func(c *canvas.Canvas, x, y, w, _ float32) {
		for i, item := range s.items {
			ry := y + float32(i)*(rowH+gap)
			c.DrawRoundedRect(x+12, ry+2, w-24, rowH-4, 8, canvas.FillPaint(colSurface))
			c.DrawRoundedRect(x+12, ry+2, 3, rowH-4, 2, canvas.FillPaint(item.col))

			ts := canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 14, FontPath: st.FontPath}
			cs := canvas.TextStyle{Color: colMuted, Size: 11, FontPath: st.FontPath}
			c.DrawText(x+28, ry+rowH/2-4, item.name, ts)
			c.DrawText(x+28, ry+rowH/2+14, item.kind, cs)

			// Tag pill
			pillW := float32(58)
			pillX := x + w - 36 - pillW
			pillY := ry + rowH/2 - 9
			kindStyle := canvas.TextStyle{Color: canvas.Hex("#1E1E2E"), Size: 11, FontPath: st.FontPath}
			c.DrawRoundedRect(pillX, pillY, pillW, 18, 9, canvas.FillPaint(item.col))
			kw := c.MeasureText(item.kind, kindStyle).W
			c.DrawText(pillX+(pillW-kw)/2, pillY+13, item.kind, kindStyle)

			// Row border
			c.DrawRoundedRect(x+12, ry+2, w-24, rowH-4, 8,
				canvas.StrokePaint(colOverlay, 1))
		}
	})

	return s
}

func (s *LibraryScreen) OnEnter(nav *ui.Navigator) {
	s.BaseScreen.OnEnter(nav)
}

func (s *LibraryScreen) Tick(delta float64) {
	s.scroll.Tick(delta)
	for _, b := range s.buttons {
		b.Tick(delta)
	}
}

func (s *LibraryScreen) HandleEvent(e ui.Event) bool {
	if s.scroll.HandleEvent(e) {
		return true
	}

	// Hit-test each row manually — translate Y by scroll offset
	const rowH = float32(64)
	const gap = float32(4)
	sb := s.scroll.Bounds()
	scrollY := s.scroll.ScrollOffset()

	for i := range s.items {
		ry := sb.Y + float32(i)*(rowH+gap) - scrollY
		if ry+rowH < sb.Y || ry > sb.Y+sb.H {
			continue // off screen
		}
		if e.Type == ui.EventMouseDown &&
			e.X >= sb.X && e.X <= sb.X+sb.W &&
			e.Y >= ry && e.Y <= ry+rowH {
			item := s.items[i]
			s.Nav.Push(NewDetailScreen(s.st, item.name, "kind: "+item.kind, item.col))
			return true
		}
	}
	return false
}

func (s *LibraryScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})

	c.DrawRect(x, y, w, 64, canvas.FillPaint(colSurface))
	c.DrawText(x+24, y+40, "Library", s.st.H1)
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colOverlay))

	countStyle := canvas.TextStyle{Color: colMuted, Size: 12, FontPath: s.st.FontPath}
	label := fmt.Sprintf("%d items — tap any row to open", len(s.items))
	c.DrawText(x+24, y+84, label, countStyle)

	s.scroll.Draw(c, x, y+96, w, h-96)
}
