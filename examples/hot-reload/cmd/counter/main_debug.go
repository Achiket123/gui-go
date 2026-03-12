//go:build debug

package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/devtools"
	appui "github.com/achiket123/gui-go/examples/hot-reload/internal/ui"
)

func main() {
	log.Println("=== Counter App [debug + hot-reload] ===")

	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")

	// ── Hot Reloader ──────────────────────────────────────────────────────────
	reloader := devtools.NewHotReloader(projectRoot, func() {
		log.Println("[hot-reload] ✅ build succeeded — restarting…")
		os.Exit(0)
	})
	reloader.BuildCmd = []string{"go", "build", "-tags", "debug", "-o", os.Args[0], "./cmd/counter"}
	go reloader.Watch()

	// ── App + Devtools ────────────────────────────────────────────────────────
	app := appui.NewCounterApp()

	dbg := devtools.NewLayoutDebugger(app.Root())
	dbg.Register(app.CounterWidget())
	dbg.Register(app.ButtonBar())

	fps := devtools.NewFPSOverlay()
	app.SetOverlays(dbg, fps)

	logged := devtools.NewEventLogger(dbg, "counter")

	// ── Window ────────────────────────────────────────────────────────────────
	win := goui.NewWindow("Counter App", 480, 360)

	// ✅ FIRST: clear the canvas every frame — prevents garbage pixel noise
	win.AddComponent(appui.NewClearRect(canvas.Color{R: 0.1, G: 0.1, B: 0.12, A: 1}))

	// ✅ Give each component an explicit rect so nothing draws at (0,0,0,0)
	win.AddComponent(appui.NewFixedPosition(logged, 0, 0, 480, 280))
	win.AddComponent(appui.NewFixedPosition(app.ButtonBar(), 0, 280, 480, 80))
	win.AddComponent(fps)

	log.Println("[devtools] F1 → layout overlay | watch:", projectRoot)
	win.Show() // blocks until window is closed
}
