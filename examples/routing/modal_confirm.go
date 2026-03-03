package main

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

// ConfirmModal is a modal dialog with confirm + cancel buttons.
// It is pushed via nav.PushModal and calls onConfirm/onCancel which
// are responsible for calling nav.PopModal().
type ConfirmModal struct {
	ui.BaseScreen
	st        *Styles
	title     string
	message   string
	onConfirm func()
	onCancel  func()

	btnConfirm *ui.Button
	btnCancel  *ui.Button
}

func NewConfirmModal(st *Styles, title, message string, onConfirm, onCancel func()) *ConfirmModal {
	m := &ConfirmModal{
		st:        st,
		title:     title,
		message:   message,
		onConfirm: onConfirm,
		onCancel:  onCancel,
	}

	confirmStyle := ui.DefaultButtonStyle()
	confirmStyle.Background = colRed
	confirmStyle.HoverColor = canvas.Hex("#F7A8B8")
	confirmStyle.PressColor = canvas.Hex("#C0526A")
	confirmStyle.TextStyle = canvas.TextStyle{Color: canvas.Hex("#1E1E2E"), Size: 13, FontPath: st.FontPath}
	confirmStyle.BorderRadius = 8
	m.btnConfirm = ui.NewButton("Confirm", onConfirm)
	m.btnConfirm.Style = confirmStyle

	cancelStyle := ui.DefaultButtonStyle()
	cancelStyle.Background = colOverlay
	cancelStyle.HoverColor = canvas.Hex("#45475A")
	cancelStyle.PressColor = canvas.Hex("#313244")
	cancelStyle.TextStyle = canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 13, FontPath: st.FontPath}
	cancelStyle.BorderRadius = 8
	m.btnCancel = ui.NewButton("Cancel", onCancel)
	m.btnCancel.Style = cancelStyle

	return m
}

func (m *ConfirmModal) Tick(delta float64) {
	m.btnConfirm.Tick(delta)
	m.btnCancel.Tick(delta)
}

func (m *ConfirmModal) HandleEvent(e ui.Event) bool {
	if m.btnConfirm.HandleEvent(e) {
		return true
	}
	if m.btnCancel.HandleEvent(e) {
		return true
	}
	return false
}

func (m *ConfirmModal) Draw(c *canvas.Canvas, x, y, w, h float32) {
	// Dialog panel — centered in the full window
	const dialogW = float32(380)
	const dialogH = float32(180)
	dx := x + (w-dialogW)/2
	dy := y + (h-dialogH)/2

	// Panel background + border
	c.DrawRoundedRect(dx, dy, dialogW, dialogH, 14,
		canvas.FillPaint(canvas.Hex("#24273A")))
	c.DrawRoundedRect(dx, dy, dialogW, dialogH, 14,
		canvas.StrokePaint(colOverlay, 1.5))

	m.SetBounds(canvas.Rect{X: dx, Y: dy, W: dialogW, H: dialogH})

	// Title
	titleStyle := canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 16, FontPath: m.st.FontPath}
	c.DrawText(dx+24, dy+36, m.title, titleStyle)

	// Message
	msgStyle := canvas.TextStyle{Color: canvas.Hex("#A6ADC8"), Size: 13, FontPath: m.st.FontPath}
	c.DrawTextInRect(canvas.Rect{X: dx + 24, Y: dy + 52, W: dialogW - 48, H: 60},
		m.message, msgStyle)

	// Buttons
	btnY := dy + dialogH - 54
	m.btnCancel.Draw(c, dx+24, btnY, 110, 36)
	m.btnConfirm.Draw(c, dx+dialogW-134, btnY, 110, 36)
}
