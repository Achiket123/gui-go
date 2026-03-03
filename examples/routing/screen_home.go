package main

import (
	"fmt"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
	"github.com/achiket/gui-go/ui/layout"
)

type card struct {
	title    string
	subtitle string
	color    canvas.Color
	btn      *ui.Button
}

// HomeScreen shows a grid of cards. Clicking one pushes the DetailScreen.
type HomeScreen struct {
	ui.BaseScreen
	st    *Styles
	cards []card
}

func NewHomeScreen(st *Styles) *HomeScreen {
	s := &HomeScreen{st: st}

	entries := []struct {
		title, sub string
		color      canvas.Color
	}{
		{"Getting Started", "Learn the basics of goui", colAccent},
		{"Animation", "Tweens, controllers, timelines", colGreen},
		{"Scrolling", "ScrollView with momentum", colYellow},
		{"Text Input", "Keyboard handling, cursor", colRed},
		{"Gradients", "Linear and diagonal fills", canvas.Hex("#CBA6F7")},
		{"Transforms", "Rotate, scale, translate", canvas.Hex("#FAB387")},
	}

	for _, e := range entries {
		e := e
		c := card{
			title:    e.title,
			subtitle: e.sub,
			color:    e.color,
		}
		btnStyle := ui.DefaultButtonStyle()
		btnStyle.Background = e.color
		btnStyle.HoverColor = canvas.Lerp(e.color, canvas.White, 0.15)
		btnStyle.PressColor = canvas.Lerp(e.color, canvas.Black, 0.2)
		btnStyle.TextStyle = canvas.TextStyle{
			Color:    canvas.Hex("#1E1E2E"),
			Size:     12,
			FontPath: st.FontPath,
		}
		btnStyle.BorderRadius = 6
		c.btn = ui.NewButton("Open →", func() {
			s.Nav.Push(NewDetailScreen(st, e.title, e.sub, e.color))
		})
		c.btn.Style = btnStyle
		s.cards = append(s.cards, c)
	}
	return s
}

func (s *HomeScreen) OnEnter(nav *ui.Navigator) {
	s.BaseScreen.OnEnter(nav)
}

func (s *HomeScreen) Tick(delta float64) {
	for i := range s.cards {
		s.cards[i].btn.Tick(delta)
	}
}

func (s *HomeScreen) HandleEvent(e ui.Event) bool {
	for i := range s.cards {
		if s.cards[i].btn.HandleEvent(e) {
			return true
		}
	}
	return false
}

func (s *HomeScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})

	// Header
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colSurface))
	c.DrawText(x+24, y+40, "Home", s.st.H1)
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colOverlay))

	// Subtitle
	c.DrawText(x+24, y+88, "Pick a topic to explore", s.st.Body)

	// 2-column grid of cards
	const cols = 2
	const cardH = float32(130)
	const pad = float32(16)
	const gap = float32(12)

	gridY := y + 104
	gridW := w - pad*2
	colW := (gridW - gap) / cols

	rects := layout.Grid(x+pad, gridY, gridW, cardH*3+gap*2, cols, 3, gap)

	for i, r := range rects {
		if i >= len(s.cards) {
			break
		}
		cd := s.cards[i]

		// Card background
		c.DrawRoundedRect(r.X, r.Y, r.W, r.H, 12, canvas.FillPaint(colSurface))

		// Accent top bar
		c.DrawRoundedRect(r.X, r.Y, r.W, 4, 2, canvas.FillPaint(cd.color))

		// Title & subtitle
		c.DrawText(r.X+14, r.Y+26, cd.title, s.st.H2)
		c.DrawText(r.X+14, r.Y+46, cd.subtitle, s.st.Body)

		// "Open" button at bottom-right of card
		btnW := float32(80)
		btnH := float32(28)
		btnX := r.X + r.W - btnW - 12
		btnY := r.Y + r.H - btnH - 12
		cd.btn.Draw(c, btnX, btnY, btnW, btnH)

		// Card border
		c.DrawRoundedRect(r.X, r.Y, r.W, r.H, 12,
			canvas.StrokePaint(colOverlay, 1))

		_ = colW        // suppress unused
		_ = fmt.Sprintf // suppress unused
	}
}
