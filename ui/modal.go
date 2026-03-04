// Package ui — modal.go
//
// Modal overlay system.
//
// Components
//   - ModalManager  — owns the overlay stack; register once with the window
//   - Dialog        — centred panel: title + content area + footer action buttons
//   - AlertDialog   — single-button convenience wrapper around Dialog
//   - ConfirmDialog — Cancel / Confirm two-button wrapper
//   - BottomSheet   — panel that rises from the screen bottom
//
// Usage
//
//	mgr := ui.NewModalManager()
//	window.Register(mgr)
//
//	mgr.Push(ui.NewDialog(ui.DialogOptions{
//	    Title:   "Settings",
//	    Width:   480,
//	    Content: myForm,
//	    OnClose: func() { mgr.Pop() },
//	}))
package ui

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
)

// ─────────────────────────────────────────────────────────────────────────────
// Modal interface
// ─────────────────────────────────────────────────────────────────────────────

// Modal is anything the ModalManager can push/pop.
type Modal interface {
	Component
	IsBlocking() bool // true → consume all events not handled by the modal
}

// ─────────────────────────────────────────────────────────────────────────────
// ModalManager
// ─────────────────────────────────────────────────────────────────────────────

// ModalManager maintains a stack of overlays and renders them in order.
type ModalManager struct {
	stack  []Modal
	bounds canvas.Rect
}

func NewModalManager() *ModalManager { return &ModalManager{} }

// Push adds a modal to the top of the stack.
func (m *ModalManager) Push(mod Modal) { m.stack = append(m.stack, mod) }

// Pop removes the topmost modal.
func (m *ModalManager) Pop() {
	if len(m.stack) > 0 {
		m.stack = m.stack[:len(m.stack)-1]
	}
}

// Clear removes all modals.
func (m *ModalManager) Clear() { m.stack = m.stack[:0] }

// Active returns true when at least one modal is showing.
func (m *ModalManager) Active() bool { return len(m.stack) > 0 }

// Depth returns the number of active modals.
func (m *ModalManager) Depth() int { return len(m.stack) }

func (m *ModalManager) Bounds() canvas.Rect { return m.bounds }

func (m *ModalManager) Tick(delta float64) {
	for _, mod := range m.stack {
		mod.Tick(delta)
	}
}

func (m *ModalManager) Draw(c *canvas.Canvas, x, y, w, h float32) {
	m.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	for _, mod := range m.stack {
		mod.Draw(c, x, y, w, h)
	}
}

func (m *ModalManager) HandleEvent(e Event) bool {
	if len(m.stack) == 0 {
		return false
	}
	top := m.stack[len(m.stack)-1]
	if top.HandleEvent(e) {
		return true
	}
	return top.IsBlocking()
}

// ─────────────────────────────────────────────────────────────────────────────
// Backdrop helper
// ─────────────────────────────────────────────────────────────────────────────

func drawBackdrop(c *canvas.Canvas, x, y, w, h, alpha float32) {
	col := theme.Current().Colors.BgOverlay
	col.A = alpha
	c.DrawRect(x, y, w, h, canvas.FillPaint(col))
}

// ─────────────────────────────────────────────────────────────────────────────
// DialogAction — footer button spec
// ─────────────────────────────────────────────────────────────────────────────

// DialogAction describes one footer button.
type DialogAction struct {
	Label   string
	Primary bool
	OnClick func()
}

// ─────────────────────────────────────────────────────────────────────────────
// DialogOptions
// ─────────────────────────────────────────────────────────────────────────────

// DialogOptions configures a Dialog.
type DialogOptions struct {
	Title   string
	Width   float32 // 0 → 480
	Height  float32 // 0 → auto
	Content Component
	Actions []DialogAction

	OnClose       func()
	CloseOnBack   bool    // close when backdrop clicked (default true)
	BackdropAlpha float32 // 0 → 0.60
}

// ─────────────────────────────────────────────────────────────────────────────
// Dialog
// ─────────────────────────────────────────────────────────────────────────────

// Dialog is a centred, blocking overlay panel.
type Dialog struct {
	opts       DialogOptions
	closeBtn   *Button
	actionBtns []*Button
	bounds     canvas.Rect
}

// NewDialog creates a Dialog from options.
func NewDialog(opts DialogOptions) *Dialog {
	if opts.Width == 0 {
		opts.Width = 480
	}
	if opts.BackdropAlpha == 0 {
		opts.BackdropAlpha = 0.60
	}
	// Default to true.
	if !opts.CloseOnBack {
		opts.CloseOnBack = true
	}

	th := theme.Current()

	// ✕ button.
	closeSt := DefaultButtonStyle()
	closeSt.Background = canvas.Transparent
	closeSt.HoverColor = th.Colors.Border
	closeSt.TextStyle = canvas.TextStyle{Color: th.Colors.TextSecondary, Size: 16}
	closeBtn := NewButton("✕", nil)
	closeBtn.Style = closeSt
	if opts.OnClose != nil {
		closeBtn.OnClick = opts.OnClose
	}

	// Footer buttons.
	var actionBtns []*Button
	for _, act := range opts.Actions {
		act := act
		var st ButtonStyle
		if act.Primary {
			st = DefaultButtonStyle()
		} else {
			st = dialogGhostStyle(th)
		}
		btn := NewButton(act.Label, nil)
		btn.Style = st
		btn.OnClick = act.OnClick
		actionBtns = append(actionBtns, btn)
	}

	return &Dialog{opts: opts, closeBtn: closeBtn, actionBtns: actionBtns}
}

func (d *Dialog) IsBlocking() bool    { return true }
func (d *Dialog) Bounds() canvas.Rect { return d.bounds }

func (d *Dialog) Tick(delta float64) {
	d.closeBtn.Tick(delta)
	if d.opts.Content != nil {
		d.opts.Content.Tick(delta)
	}
	for _, btn := range d.actionBtns {
		btn.Tick(delta)
	}
}

const dialogHeaderH = float32(52)
const dialogFooterH = float32(60)

func (d *Dialog) panelRect(cw, ch float32) canvas.Rect {
	th := theme.Current()
	dw := d.opts.Width
	footerH := float32(0)
	if len(d.actionBtns) > 0 {
		footerH = dialogFooterH
	}
	dh := d.opts.Height
	if dh == 0 {
		dh = dialogHeaderH + ch*0.45 + footerH + th.Space.MD*2
	}
	return canvas.Rect{X: (cw - dw) / 2, Y: (ch - dh) / 2, W: dw, H: dh}
}

func (d *Dialog) HandleEvent(e Event) bool {
	b := d.bounds
	dr := d.panelRect(b.W, b.H)

	if e.Type == EventMouseDown && e.Button == 1 && d.opts.CloseOnBack {
		inPanel := e.X >= dr.X && e.X <= dr.X+dr.W && e.Y >= dr.Y && e.Y <= dr.Y+dr.H
		if !inPanel {
			if d.opts.OnClose != nil {
				d.opts.OnClose()
			}
			return true
		}
	}
	if d.closeBtn.HandleEvent(e) {
		return true
	}
	for _, btn := range d.actionBtns {
		if btn.HandleEvent(e) {
			return true
		}
	}
	if d.opts.Content != nil && d.opts.Content.HandleEvent(e) {
		return true
	}
	return true // always block underlying events
}

func (d *Dialog) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()

	// 1. Backdrop.
	drawBackdrop(c, x, y, w, h, d.opts.BackdropAlpha)

	dr := d.panelRect(w, h)

	// 2. Shadow.
	for i := float32(1); i <= 10; i++ {
		sc := canvas.Color{A: 0.022 * (11 - i)}
		c.DrawRoundedRect(dr.X+i, dr.Y+i*0.8, dr.W, dr.H, th.Radius.LG, canvas.FillPaint(sc))
	}

	// 3. Panel background + border.
	c.DrawRoundedRect(dr.X, dr.Y, dr.W, dr.H, th.Radius.LG, canvas.FillPaint(th.Colors.BgSurface))
	c.DrawRoundedRect(dr.X, dr.Y, dr.W, dr.H, th.Radius.LG, canvas.StrokePaint(th.Colors.Border, 1))

	// 4. Header.
	c.DrawRect(dr.X, dr.Y+dialogHeaderH-1, dr.W, 1, canvas.FillPaint(th.Colors.Border))
	c.DrawCenteredText(
		canvas.Rect{X: dr.X + 48, Y: dr.Y, W: dr.W - 96, H: dialogHeaderH},
		d.opts.Title, th.Type.H3)
	d.closeBtn.Draw(c, dr.X+dr.W-44, dr.Y+10, 32, 32)

	// 5. Content area.
	footerH := float32(0)
	if len(d.actionBtns) > 0 {
		footerH = dialogFooterH
	}
	contentY := dr.Y + dialogHeaderH
	contentH := dr.H - dialogHeaderH - footerH
	if d.opts.Content != nil {
		pad := th.Space.MD
		c.Save()
		c.ClipRect(dr.X+1, contentY, dr.W-2, contentH)
		d.opts.Content.Draw(c, dr.X+pad, contentY+pad*0.5, dr.W-pad*2, contentH-pad)
		c.Restore()
	}

	// 6. Footer.
	if len(d.actionBtns) > 0 {
		fy := dr.Y + dr.H - footerH
		c.DrawRect(dr.X, fy, dr.W, 1, canvas.FillPaint(th.Colors.Border))
		btnW := float32(100)
		gap := float32(8)
		total := float32(len(d.actionBtns))*(btnW+gap) - gap
		bx := dr.X + (dr.W-total)/2
		for _, btn := range d.actionBtns {
			btn.Draw(c, bx, fy+12, btnW, 36)
			bx += btnW + gap
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Convenience constructors
// ─────────────────────────────────────────────────────────────────────────────

// NewAlertDialog creates a modal with title, message, and a single OK button.
func NewAlertDialog(title, message string, onOK func()) *Dialog {
	return NewDialog(DialogOptions{
		Title:   title,
		Width:   400,
		Content: NewLabel(message, DefaultLabelStyle()),
		Actions: []DialogAction{{Label: "OK", Primary: true, OnClick: onOK}},
		OnClose: onOK,
	})
}

// NewConfirmDialog creates a modal with Cancel and Confirm buttons.
func NewConfirmDialog(title, message string, onConfirm, onCancel func()) *Dialog {
	return NewDialog(DialogOptions{
		Title:   title,
		Width:   420,
		Content: NewLabel(message, DefaultLabelStyle()),
		Actions: []DialogAction{
			{Label: "Cancel", Primary: false, OnClick: onCancel},
			{Label: "Confirm", Primary: true, OnClick: onConfirm},
		},
		OnClose: onCancel,
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// BottomSheet
// ─────────────────────────────────────────────────────────────────────────────

// BottomSheetOptions configures a BottomSheet.
type BottomSheetOptions struct {
	Title         string
	Height        float32 // 0 → 50% of screen
	Content       Component
	OnClose       func()
	CloseOnBack   bool    // default true
	BackdropAlpha float32 // default 0.50
}

// BottomSheet is a Modal that rises from the bottom of the screen.
type BottomSheet struct {
	opts   BottomSheetOptions
	bounds canvas.Rect
}

func NewBottomSheet(opts BottomSheetOptions) *BottomSheet {
	if opts.BackdropAlpha == 0 {
		opts.BackdropAlpha = 0.50
	}
	if !opts.CloseOnBack {
		opts.CloseOnBack = true
	}
	return &BottomSheet{opts: opts}
}

func (bs *BottomSheet) IsBlocking() bool    { return true }
func (bs *BottomSheet) Bounds() canvas.Rect { return bs.bounds }

func (bs *BottomSheet) Tick(delta float64) {
	if bs.opts.Content != nil {
		bs.opts.Content.Tick(delta)
	}
}

func (bs *BottomSheet) sheetH(screenH float32) float32 {
	h := bs.opts.Height
	if h <= 0 {
		h = screenH * 0.50
	}
	return h
}

func (bs *BottomSheet) HandleEvent(e Event) bool {
	b := bs.bounds
	sheetY := b.Y + b.H - bs.sheetH(b.H)

	if e.Type == EventMouseDown && e.Button == 1 && e.Y < sheetY && bs.opts.CloseOnBack {
		if bs.opts.OnClose != nil {
			bs.opts.OnClose()
		}
		return true
	}
	if bs.opts.Content != nil && bs.opts.Content.HandleEvent(e) {
		return true
	}
	return true
}

func (bs *BottomSheet) Draw(c *canvas.Canvas, x, y, w, h float32) {
	bs.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()
	drawBackdrop(c, x, y, w, h, bs.opts.BackdropAlpha)

	sh := bs.sheetH(h)
	sy := y + h - sh

	// Soft shadow above the sheet.
	for i := float32(1); i <= 12; i++ {
		c.DrawRect(x, sy-i, w, i+1, canvas.FillPaint(canvas.Color{A: 0.018 * (13 - i)}))
	}

	// Sheet background (rounded top corners only).
	c.DrawRoundedRect(x, sy, w, sh, th.Radius.LG, canvas.FillPaint(th.Colors.BgSurface))
	// Fill lower rectangle to cover the bottom-corner rounding.
	c.DrawRect(x, sy+th.Radius.LG, w, sh-th.Radius.LG, canvas.FillPaint(th.Colors.BgSurface))

	// Handle pill.
	pillW := float32(40)
	c.DrawRoundedRect(x+(w-pillW)/2, sy+10, pillW, 4, 2, canvas.FillPaint(th.Colors.Border))

	// Title + divider.
	headerH := float32(0)
	if bs.opts.Title != "" {
		headerH = 48
		c.DrawCenteredText(canvas.Rect{X: x, Y: sy + 16, W: w, H: headerH}, bs.opts.Title, th.Type.H3)
		c.DrawRect(x, sy+headerH+16, w, 1, canvas.FillPaint(th.Colors.Border))
	}

	// Content.
	if bs.opts.Content != nil {
		pad := th.Space.MD
		topOff := sy + headerH + 20
		bs.opts.Content.Draw(c, x+pad, topOff, w-pad*2, sh-headerH-20-pad)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal style helper
// ─────────────────────────────────────────────────────────────────────────────

func dialogGhostStyle(th *theme.Theme) ButtonStyle {
	st := DefaultButtonStyle()
	st.Background = canvas.Transparent
	st.HoverColor = th.Colors.BgBase
	st.TextStyle = canvas.TextStyle{Color: th.Colors.TextPrimary, Size: 14}
	st.BorderColor = th.Colors.Border
	st.BorderWidth = 1
	return st
}
