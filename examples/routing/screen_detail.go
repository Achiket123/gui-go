package main

import (
	"fmt"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

// DetailScreen is pushed onto the stack when a card is tapped.
// It shows a back button and a "Delete" that triggers a modal confirmation.
type DetailScreen struct {
	ui.BaseScreen
	st       *Styles
	title    string
	subtitle string
	color    canvas.Color

	btnBack   *ui.Button
	btnDelete *ui.Button
	btnShare  *ui.Button
}

func NewDetailScreen(st *Styles, title, subtitle string, color canvas.Color) *DetailScreen {
	s := &DetailScreen{
		st:       st,
		title:    title,
		subtitle: subtitle,
		color:    color,
	}

	// Back button — calls Nav.Pop()
	backStyle := ui.DefaultButtonStyle()
	backStyle.Background = colOverlay
	backStyle.HoverColor = canvas.Hex("#45475A")
	backStyle.PressColor = canvas.Hex("#313244")
	backStyle.TextStyle = canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 13, FontPath: st.FontPath}
	backStyle.BorderRadius = 8
	s.btnBack = ui.NewButton("← Back", func() { s.Nav.Pop() })
	s.btnBack.Style = backStyle

	// Share button — just a visual example, no navigation
	shareStyle := ui.DefaultButtonStyle()
	shareStyle.Background = colAccent
	shareStyle.HoverColor = canvas.Hex("#B4D0FD")
	shareStyle.PressColor = canvas.Hex("#5A8DF7")
	shareStyle.TextStyle = canvas.TextStyle{Color: canvas.Hex("#1E1E2E"), Size: 13, FontPath: st.FontPath}
	shareStyle.BorderRadius = 8
	s.btnShare = ui.NewButton("Share", nil)
	s.btnShare.Style = shareStyle

	// Delete button — opens a modal confirmation
	delStyle := ui.DefaultButtonStyle()
	delStyle.Background = canvas.Hex("#3B1219")
	delStyle.HoverColor = canvas.Hex("#4E1A24")
	delStyle.PressColor = canvas.Hex("#2A0D11")
	delStyle.TextStyle = canvas.TextStyle{Color: colRed, Size: 13, FontPath: st.FontPath}
	delStyle.BorderRadius = 8
	s.btnDelete = ui.NewButton("Delete", func() {
		s.Nav.PushModal(NewConfirmModal(st,
			"Delete this item?",
			fmt.Sprintf("\"%s\" will be permanently removed.", s.title),
			func() {
				s.Nav.PopModal()
				s.Nav.Pop() // go back after delete
			},
			func() {
				s.Nav.PopModal() // cancelled
			},
		))
	})
	s.btnDelete.Style = delStyle

	return s
}

func (s *DetailScreen) Tick(delta float64) {
	s.btnBack.Tick(delta)
	s.btnDelete.Tick(delta)
	s.btnShare.Tick(delta)
}

func (s *DetailScreen) HandleEvent(e ui.Event) bool {
	if s.btnBack.HandleEvent(e) {
		return true
	}
	if s.btnDelete.HandleEvent(e) {
		return true
	}
	if s.btnShare.HandleEvent(e) {
		return true
	}
	return false
}

func (s *DetailScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})

	// Top nav bar
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colSurface))
	s.btnBack.Draw(c, x+16, y+16, 90, 34)

	// Stack depth indicator (shows how deep you are)
	depthLabel := fmt.Sprintf("depth %d", s.Nav.Depth())
	dl := canvas.TextStyle{Color: colMuted, Size: 11, FontPath: s.st.FontPath}
	dw := c.MeasureText(depthLabel, dl).W
	c.DrawText(x+w-dw-20, y+38, depthLabel, dl)

	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colOverlay))

	// Hero colour band
	c.DrawRect(x, y+64, w, 140, canvas.FillPaint(s.color))

	// Title over the hero
	titleStyle := canvas.TextStyle{Color: canvas.Hex("#1E1E2E"), Size: 28, FontPath: s.st.FontPath}
	c.DrawText(x+28, y+140, s.title, titleStyle)

	// Body content area
	bodyY := y + 220
	c.DrawText(x+28, bodyY, s.subtitle, s.st.H2)

	body := "This detail screen was pushed onto the navigator stack.\n" +
		"Press ← Back or the back button to return.\n" +
		"Press Delete to trigger a modal confirmation dialog."

	lineH := float32(22)
	lines := splitLines(body)
	for i, line := range lines {
		c.DrawText(x+28, bodyY+float32(i+1)*lineH+12, line, s.st.Body)
	}

	// Action buttons at the bottom
	btnY := y + h - 60
	s.btnShare.Draw(c, x+28, btnY, 110, 38)
	s.btnDelete.Draw(c, x+w-138, btnY, 110, 38)
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
