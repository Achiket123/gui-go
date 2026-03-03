package main

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

type settingRow struct {
	label string
	value string
	color canvas.Color
}

// SettingsScreen is a tab — a simple list of settings rows.
type SettingsScreen struct {
	ui.BaseScreen
	st   *Styles
	rows []settingRow
	btn  *ui.Button // "Reset to defaults" — triggers a modal
}

func NewSettingsScreen(st *Styles) *SettingsScreen {
	s := &SettingsScreen{
		st: st,
		rows: []settingRow{
			{"Renderer", "OpenGL 2.1", colAccent},
			{"Font", "DejaVu Sans", colGreen},
			{"Theme", "Catppuccin Mocha", canvas.Hex("#CBA6F7")},
			{"FPS target", "60", colYellow},
			{"VSync", "Enabled", colGreen},
			{"Scale factor", "1.0×", colAccent},
			{"Debug overlay", "Off", colMuted},
		},
	}

	resetStyle := ui.DefaultButtonStyle()
	resetStyle.Background = canvas.Hex("#3B1219")
	resetStyle.HoverColor = canvas.Hex("#4E1A24")
	resetStyle.PressColor = canvas.Hex("#2A0D11")
	resetStyle.TextStyle = canvas.TextStyle{Color: colRed, Size: 13, FontPath: st.FontPath}
	resetStyle.BorderRadius = 8

	s.btn = ui.NewButton("Reset to defaults", func() {
		s.Nav.PushModal(NewConfirmModal(st,
			"Reset settings?",
			"All preferences will be restored to their default values.",
			func() { s.Nav.PopModal() }, // confirmed — handle actual reset here
			func() { s.Nav.PopModal() }, // cancelled
		))
	})
	s.btn.Style = resetStyle
	return s
}

func (s *SettingsScreen) Tick(delta float64) {
	s.btn.Tick(delta)
}

func (s *SettingsScreen) HandleEvent(e ui.Event) bool {
	return s.btn.HandleEvent(e)
}

func (s *SettingsScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})

	// Header
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colSurface))
	c.DrawText(x+24, y+40, "Settings", s.st.H1)
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colOverlay))

	// Rows
	const rowH = float32(52)
	const pad = float32(24)
	ry := y + 80

	keyStyle := canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 13, FontPath: s.st.FontPath}
	valStyle := canvas.TextStyle{Color: colMuted, Size: 13, FontPath: s.st.FontPath}

	for _, row := range s.rows {
		c.DrawRect(x+pad, ry, w-pad*2, 1, canvas.FillPaint(colOverlay))

		// Dot indicator
		c.DrawCircle(x+pad+8, ry+rowH/2, 4, canvas.FillPaint(row.color))

		// Key
		c.DrawText(x+pad+22, ry+rowH/2+5, row.label, keyStyle)

		// Value (right-aligned)
		vw := c.MeasureText(row.value, valStyle).W
		c.DrawText(x+w-pad-vw, ry+rowH/2+5, row.value, valStyle)

		ry += rowH
	}

	c.DrawRect(x+pad, ry, w-pad*2, 1, canvas.FillPaint(colOverlay))

	// Reset button
	s.btn.Draw(c, x+pad, y+h-64, 180, 38)
}
