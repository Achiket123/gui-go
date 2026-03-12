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
)

const (
	W         = 1100
	H         = 700
	SidebarW  = 200
	TabHeight = 52
)

func findFont(name string) string {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		os.Getenv("HOME") + "/.fonts",
		os.Getenv("HOME") + "/.local/share/fonts",
	}
	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(d.Name()), ".ttf") &&
				strings.Contains(strings.ToLower(d.Name()), strings.ToLower(name)) {
				name = path
				return fmt.Errorf("found")
			}
			return nil
		})
		if strings.HasPrefix(name, "/") {
			return name
		}
	}
	return ""
}

func main() {
	// ── Seed data ─────────────────────────────────────────────────────────────
	store := NewFinanceStore()

	// ── Window ────────────────────────────────────────────────────────────────
	w := goui.NewWindow("💰 Personal Finance Tracker", W, H)

	// ── Font ──────────────────────────────────────────────────────────────────
	fontPath := ""
	for _, name := range []string{"DejaVuSans", "LiberationSans-Regular", "FreeSans", "Ubuntu-R"} {
		if p := findFont(name); p != "" {
			fontPath = p
			break
		}
	}
	st := NewStyles(fontPath)

	// ── Toast manager (overlay) ────────────────────────────────────────────────
	toasts := ui.NewToastManager()

	// ── Screens ───────────────────────────────────────────────────────────────
	dashScreen := NewDashboardScreen(st, store, toasts)
	txScreen := NewTransactionsScreen(st, store, toasts)
	budgetScreen := NewBudgetScreen(st, store, toasts)
	analyticsScreen := NewAnalyticsScreen(st, store)
	settingsScreen := NewSettingsScreen(st, store, toasts)

	// ── Navigator + TabBar ────────────────────────────────────────────────────
	nav := ui.NewNavigator(dashScreen)
	tabs := ui.NewTabBar(nav, TabHeight, []ui.TabItem{
		{Label: "Dashboard", Screen: dashScreen},
		{Label: "Transactions", Screen: txScreen},
		{Label: "Budget", Screen: budgetScreen},
		{Label: "Analytics", Screen: analyticsScreen},
		{Label: "Settings", Screen: settingsScreen},
	})
	nav.SetTabBar(tabs)
	nav.OnClose(func() { w.Close() })

	// ── Animation controllers ─────────────────────────────────────────────────
	// Intro slide-in animation for the whole UI
	introTL := animation.NewTimeline(800 * time.Millisecond)
	introTL.AddTrack("alpha", 0, 1, 0, 0.6, animation.EaseOutQuad)
	introTL.AddTrack("slide", -30, 0, 0, 0.8, animation.EaseOutCubic)
	introTL.Play()
	w.AddTimeline(introTL)

	// Pulse for balance highlight
	pulseCtrl := animation.NewController(1200 * time.Millisecond)
	pulseCtrl.PingPong()
	pulseCtrl.Forward()
	w.AddController(pulseCtrl)

	// Live clock ticker
	clockCtrl := animation.NewController(1 * time.Second)
	clockCtrl.Repeat(-1)
	clockCtrl.Forward()
	w.AddController(clockCtrl)

	// ── Components ────────────────────────────────────────────────────────────
	w.AddComponent(nav)

	// ── Draw ──────────────────────────────────────────────────────────────────
	w.OnDrawGL(func(c *canvas.Canvas) {
		alpha := float32(introTL.Value("alpha"))
		slide := float32(introTL.Value("slide"))
		_ = slide

		// Background
		c.DrawRect(0, 0, c.Width(), c.Height(), canvas.FillPaint(colBase))

		// Subtle background gradient overlay
		gradPaint := canvas.GradientPaint(
			colBase,
			canvas.RGBA8(30, 30, 55, 255),
		)
		gradPaint.LinearGrad.From = canvas.Point{X: 0, Y: 0}
		gradPaint.LinearGrad.To = canvas.Point{X: c.Width(), Y: c.Height()}
		c.DrawRect(0, 0, c.Width(), c.Height(), gradPaint)

		// Nav + screens
		c.Save()
		if alpha < 1 {
			// Fade in effect — draw to an opacity layer
		}
		nav.Draw(c, 0, 0, c.Width(), c.Height())
		c.Restore()

		// Toast overlay (always on top)
		toasts.Tick(1.0 / 60.0)
		toasts.Draw(c, 0, 0, c.Width(), c.Height())
	})

	// ── Key bindings ──────────────────────────────────────────────────────────
	w.OnKeyPress(func(e goui.KeyEvent) {
		switch e.KeySym {
		case "Escape":
			w.Close()
		case "F5":
			// Refresh data
			toasts.Show("Data refreshed", ui.ToastInfo, 2*time.Second)
		}
	})

	w.OnClose(func() {
		fmt.Println("Finance Tracker closed.")
	})

	w.Show()
}
