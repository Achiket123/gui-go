// cmd/app — TaskFlow desktop GUI application.
//
// Usage:
//
//	go run ./cmd/app -api http://localhost:8080
//
// The app connects to the TaskFlow API server and provides a full
// project-management interface using the gui-go UI library.
package main

import (
	"flag"
	"log"
	"time"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"

	apiclient "github.com/achiket123/taskflow/internal/api/client"
	"github.com/achiket123/taskflow/ui/screens"
	"github.com/achiket123/taskflow/ui/state"
	"github.com/achiket123/taskflow/ui/styles"
)

func main() {
	apiURL := flag.String("api", "http://localhost:8080", "TaskFlow API base URL")
	flag.Parse()

	// Apply design tokens.
	styles.ApplyTheme()

	// Bootstrap application state.
	client := apiclient.New(*apiURL)
	appState := state.NewApp(client)

	// Wire token-refresh notification.
	client.OnTokenRefreshed = func() {
		log.Println("[app] tokens refreshed automatically")
	}

	// Create the GUI window.
	window := goui.NewWindow(
		"TaskFlow",
		1280,
		800,
	)

	// Navigator is the screen stack router.
	nav := ui.NewNavigator(screens.NewLoginScreen(appState))

	// Main render loop.
	lastTime := time.Now()
	window.OnDrawGL(func(c *canvas.Canvas) {
		now := time.Now()
		delta := float64(now.Sub(lastTime).Seconds())
		lastTime = now

		nav.Tick(delta)
		// Assuming events are processed directly via inputs now, otherwise just Draw
		nav.Draw(c, 0, 0, float32(c.Width()), float32(c.Height()))

		// Global toast overlay.
		if msg := appState.ErrorMessage.Get(); msg != "" {
			drawToast(c, c.Width(), c.Height(), msg, styles.ColCancelled)
		} else if msg := appState.SuccessMessage.Get(); msg != "" {
			drawToast(c, c.Width(), c.Height(), msg, styles.ColDone)
		}
	})

	window.OnMouseMove(func(e goui.MouseEvent) {
		ev := ui.Event{Type: ui.EventMouseMove, X: float32(e.X), Y: float32(e.Y)}
		nav.HandleEvent(ev)
	})
	window.OnMouseClick(func(e goui.MouseEvent) {
		evType := ui.EventMouseDown
		if e.Type == "release" {
			evType = ui.EventMouseUp
		}
		ev := ui.Event{Type: evType, X: float32(e.X), Y: float32(e.Y)}
		nav.HandleEvent(ev)
	})
	window.OnKeyPress(func(e goui.KeyEvent) {
		// Basic mapping
		ev := ui.Event{Type: ui.EventKeyDown, Key: e.KeySym}
		nav.HandleEvent(ev)
	})

	window.Show()
}

func drawToast(c *canvas.Canvas, winW, winH float32, msg string, col canvas.Color) {
	w := float32(winW)
	h := float32(winH)
	toastH := float32(44)
	c.DrawRect(0, h-toastH, w, toastH, canvas.FillPaint(col))
	ts := canvas.TextStyle{Color: canvas.Hex("#FFFFFF"), Size: 13}
	tw := c.MeasureText(msg, ts).W
	c.DrawText(w/2-tw/2, h-toastH+28, msg, ts)
}
