package main

import (
	"time"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

// SettingsScreen lets the user configure app preferences.
type SettingsScreen struct {
	ui.BaseScreen
	st     *Styles
	store  *FinanceStore
	toasts *ui.ToastManager

	// Widgets
	currencyDrop   *ui.Dropdown
	notifyBudget   *ui.Toggle
	notifyTx       *ui.Toggle
	darkModeToggle *ui.Toggle
	exportBtn      *ui.Button
	resetBtn       *ui.Button
	modal          *ui.ModalManager
}

func NewSettingsScreen(st *Styles, store *FinanceStore, toasts *ui.ToastManager) *SettingsScreen {
	s := &SettingsScreen{st: st, store: store, toasts: toasts}

	// Currency dropdown
	dropStyle := ui.DefaultDropdownStyle()
	dropStyle.Background = colMantle
	dropStyle.HoverBg = colSurface0
	dropStyle.Border = colSurface1
	dropStyle.TextStyle = canvas.TextStyle{Color: colSubtext0, Size: 13, FontPath: st.FontPath}
	s.currencyDrop = ui.NewDropdown(
		[]string{"USD ($)", "EUR (€)", "GBP (£)", "JPY (¥)"},
		dropStyle,
	)
	s.currencyDrop.Placeholder = "USD ($)"
	s.currencyDrop.Selected = 0

	// Toggles
	tStyle := ui.DefaultToggleStyle()
	tStyle.TrackOn = colAccent
	tStyle.TrackOff = colSurface1

	s.notifyBudget = ui.NewToggle(true, tStyle, func(v bool) {
		store.Mutate(func(st *FinanceState) { st.NotifyBudget = v })
		msg := "Budget notifications disabled"
		if v {
			msg = "Budget notifications enabled"
		}
		toasts.Show(msg, ui.ToastInfo, 2*time.Second)
	})

	s.notifyTx = ui.NewToggle(true, tStyle, func(v bool) {
		store.Mutate(func(st *FinanceState) { st.NotifyTx = v })
		msg := "Transaction alerts disabled"
		if v {
			msg = "Transaction alerts enabled"
		}
		toasts.Show(msg, ui.ToastInfo, 2*time.Second)
	})

	s.darkModeToggle = ui.NewToggle(true, tStyle, func(v bool) {
		store.Mutate(func(st *FinanceState) { st.DarkMode = v })
		toasts.Show("Theme preference saved (restart to apply)", ui.ToastInfo, 2*time.Second)
	})

	// Export button
	expStyle := ui.DefaultButtonStyle()
	expStyle.Background = colAccent
	expStyle.HoverColor = canvas.Hex("#A6C8FF")
	expStyle.PressColor = canvas.Hex("#6A9FDB")
	expStyle.TextStyle = canvas.TextStyle{Color: colBase, Size: 13, FontPath: st.FontPath}
	expStyle.BorderRadius = 8
	s.exportBtn = ui.NewButton("Export CSV", func() {
		toasts.Show("Data exported to ~/finance_export.csv", ui.ToastSuccess, 3*time.Second)
	})
	s.exportBtn.Style = expStyle

	// Reset button
	resetStyle := ui.DefaultButtonStyle()
	resetStyle.Background = canvas.Hex("#3B1219")
	resetStyle.HoverColor = canvas.Hex("#4E1A24")
	resetStyle.PressColor = canvas.Hex("#2A0D11")
	resetStyle.TextStyle = canvas.TextStyle{Color: colRed, Size: 13, FontPath: st.FontPath}
	resetStyle.BorderRadius = 8
	s.resetBtn = ui.NewButton("Reset All Data", func() {
		s.showResetConfirm()
	})
	s.resetBtn.Style = resetStyle

	s.modal = ui.NewModalManager()
	return s
}

func (s *SettingsScreen) showResetConfirm() {
	dlg := ui.NewConfirmDialog(
		"Reset All Data?",
		"This will permanently delete all your transactions, budgets and account data. This action cannot be undone.",
		func() {
			// Reset to seed data
			*s.store = *NewFinanceStore()
			s.toasts.Show("All data has been reset", ui.ToastWarning, 3*time.Second)
			s.modal.Pop()
		},
		func() { s.modal.Pop() },
	)
	s.modal.Push(dlg)
}

func (s *SettingsScreen) Tick(delta float64) {
	s.currencyDrop.Tick(delta)
	s.notifyBudget.Tick(delta)
	s.notifyTx.Tick(delta)
	s.darkModeToggle.Tick(delta)
	s.exportBtn.Tick(delta)
	s.resetBtn.Tick(delta)
	s.modal.Tick(delta)
}

func (s *SettingsScreen) HandleEvent(e ui.Event) bool {
	if s.modal.HandleEvent(e) {
		return true
	}
	if s.currencyDrop.HandleEvent(e) {
		if s.currencyDrop.Selected >= 0 {
			codes := []string{"USD", "EUR", "GBP", "JPY"}
			s.store.Mutate(func(st *FinanceState) {
				st.Currency = codes[s.currencyDrop.Selected]
			})
			s.toasts.Show("Currency updated", ui.ToastSuccess, 2*time.Second)
		}
		return true
	}
	if s.notifyBudget.HandleEvent(e) {
		return true
	}
	if s.notifyTx.HandleEvent(e) {
		return true
	}
	if s.darkModeToggle.HandleEvent(e) {
		return true
	}
	if s.exportBtn.HandleEvent(e) {
		return true
	}
	if s.resetBtn.HandleEvent(e) {
		return true
	}
	return false
}

func (s *SettingsScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})
	st := s.store.Get()

	// ── Header ────────────────────────────────────────────────────────────────
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colMantle))
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colSurface0))
	c.DrawText(x+24, y+38, "Settings", s.st.H1)

	const padX = float32(24)
	const leftW = float32(600)

	// ── Settings Sections ─────────────────────────────────────────────────────

	type settingRow struct {
		label    string
		subLabel string
		widget   func(x, y, w, h float32)
		widgetW  float32
		widgetH  float32
	}

	sectionY := y + 80
	drawSection := func(title string, rows []settingRow) {
		// Section header
		c.DrawText(x+padX, sectionY+14, title, s.st.H2)
		sectionY += 32

		for _, row := range rows {
			rowY := sectionY
			rowH := float32(56)

			c.DrawRoundedRect(x+padX, rowY, leftW, rowH, 8, canvas.FillPaint(colMantle))
			c.DrawRoundedRect(x+padX, rowY, leftW, rowH, 8, canvas.StrokePaint(colSurface0, 1))

			c.DrawText(x+padX+16, rowY+22, row.label, s.st.BodyBold)
			c.DrawText(x+padX+16, rowY+40, row.subLabel, s.st.Caption)

			if row.widget != nil {
				wx := x + padX + leftW - row.widgetW - 16
				wy := rowY + (rowH-row.widgetH)/2
				row.widget(wx, wy, row.widgetW, row.widgetH)
			}

			sectionY += rowH + 4
		}

		sectionY += 16
	}

	// General
	drawSection("General", []settingRow{
		{
			"Currency",
			"Select your preferred currency for display",
			func(wx, wy, ww, wh float32) {
				s.currencyDrop.Draw(c, wx, wy, ww, wh)
			},
			160, 36,
		},
		{
			"Dark Mode",
			"Use dark theme (requires restart)",
			func(wx, wy, ww, wh float32) {
				s.darkModeToggle.Draw(c, wx+ww-38, wy+(wh-20)/2, 38, 20)
			},
			100, 36,
		},
	})

	// Notifications
	drawSection("Notifications", []settingRow{
		{
			"Budget Alerts",
			"Get notified when you approach your budget limit",
			func(wx, wy, ww, wh float32) {
				s.notifyBudget.Draw(c, wx+ww-38, wy+(wh-20)/2, 38, 20)
			},
			100, 36,
		},
		{
			"Transaction Alerts",
			"Get notified when large transactions are added",
			func(wx, wy, ww, wh float32) {
				s.notifyTx.Draw(c, wx+ww-38, wy+(wh-20)/2, 38, 20)
			},
			100, 36,
		},
	})

	// Data
	drawSection("Data Management", []settingRow{
		{
			"Export Data",
			"Download all transactions as a CSV file",
			func(wx, wy, ww, wh float32) {
				s.exportBtn.Draw(c, wx, wy+(wh-36)/2, ww, 36)
			},
			140, 36,
		},
		{
			"Reset Application",
			"Permanently delete all data and restore defaults",
			func(wx, wy, ww, wh float32) {
				s.resetBtn.Draw(c, wx, wy+(wh-36)/2, ww, 36)
			},
			160, 36,
		},
	})

	// ── Account info (right panel) ────────────────────────────────────────────
	infoX := x + leftW + padX*3
	infoW := w - leftW - padX*4 - 10

	c.DrawText(infoX, y+80+14, "App Info", s.st.H2)

	infoItems := []struct {
		label string
		value string
	}{
		{"Version", "1.0.0"},
		{"Framework", "gui-go (goui)"},
		{"Renderer", "OpenGL 2.1"},
		{"Currency", st.Currency},
		{"Transactions", itoa(len(st.Transactions))},
		{"Budgets", itoa(len(st.Budgets))},
		{"Accounts", itoa(len(st.Accounts))},
	}

	for i, item := range infoItems {
		iy := y + 80 + 34 + float32(i)*36

		c.DrawRoundedRect(infoX, iy, infoW, 30, 6, canvas.FillPaint(colMantle))
		c.DrawRoundedRect(infoX, iy, infoW, 30, 6, canvas.StrokePaint(colSurface0, 1))

		c.DrawText(infoX+12, iy+20, item.label, s.st.Caption)
		valStyle := canvas.TextStyle{Color: colAccent, Size: 12, FontPath: s.st.FontPath}
		vsz := c.MeasureText(item.value, valStyle)
		c.DrawText(infoX+infoW-vsz.W-12, iy+20, item.value, valStyle)
	}

	// ── Keyboard shortcuts ────────────────────────────────────────────────────
	shortcutsY := y + 80 + 34 + float32(len(infoItems)+1)*36

	c.DrawText(infoX, shortcutsY, "Keyboard Shortcuts", s.st.H2)

	shortcuts := []struct {
		key  string
		desc string
	}{
		{"Escape", "Quit app"},
		{"F5", "Refresh / show toast"},
	}

	for i, sc := range shortcuts {
		sy := shortcutsY + 24 + float32(i)*28
		keyStyle := canvas.TextStyle{Color: colYellow, Size: 11, FontPath: s.st.FontPath}
		c.DrawRoundedRect(infoX, sy, 70, 20, 4, canvas.FillPaint(colSurface0))
		c.DrawText(infoX+6, sy+14, sc.key, keyStyle)
		c.DrawText(infoX+78, sy+14, sc.desc, s.st.Caption)
	}

	// Modal
	s.modal.Draw(c, x, y, w, h)
}
