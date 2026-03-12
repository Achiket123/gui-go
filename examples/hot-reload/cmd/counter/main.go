// cmd/counter/main.go
package main

import (
	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	appui "github.com/achiket123/gui-go/examples/hot-reload/internal/ui"
)

func main() {
	app := appui.NewCounterApp()

	win := goui.NewWindow("Counter App", 480, 360)
	win.AddComponent(appui.NewClearRect(canvas.Color{R: 0.08, G: 0.08, B: 0.12, A: 1}))
	win.AddComponent(appui.NewFixedPosition(app.CounterWidget(), 0, 0, 480, 280))
	win.AddComponent(appui.NewFixedPosition(app.ButtonBar(), 0, 280, 480, 80))

	win.Show()
}
