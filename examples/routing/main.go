package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

const (
	W         = 800
	H         = 620
	TabHeight = 56
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

// Styles holds shared text styles so every screen uses the same font.
type Styles struct {
	FontPath string
	H1       canvas.TextStyle
	H2       canvas.TextStyle
	Body     canvas.TextStyle
	Caption  canvas.TextStyle
}

func NewStyles(fontPath string) *Styles {
	return &Styles{
		FontPath: fontPath,
		H1:       canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 22, FontPath: fontPath},
		H2:       canvas.TextStyle{Color: canvas.Hex("#CDD6F4"), Size: 16, FontPath: fontPath},
		Body:     canvas.TextStyle{Color: canvas.Hex("#A6ADC8"), Size: 13, FontPath: fontPath},
		Caption:  canvas.TextStyle{Color: canvas.Hex("#6C7086"), Size: 11, FontPath: fontPath},
	}
}

var (
	colBase    = canvas.Hex("#1E1E2E")
	colSurface = canvas.Hex("#181825")
	colOverlay = canvas.Hex("#313244")
	colAccent  = canvas.Hex("#89B4FA")
	colGreen   = canvas.Hex("#A6E3A1")
	colRed     = canvas.Hex("#F38BA8")
	colYellow  = canvas.Hex("#F9E2AF")
	colMuted   = canvas.Hex("#6C7086")
)

func main() {
	w := goui.NewWindow("goui — Routing Example", W, H)

	fontPath := ""
	for _, name := range []string{"DejaVuSans", "LiberationSans-Regular", "FreeSans", "Ubuntu-R"} {
		if p := findFont(name); p != "" {
			fontPath = p
			break
		}
	}

	st := NewStyles(fontPath)
	home := NewHomeScreen(st)
	nav := ui.NewNavigator(home)

	tabs := ui.NewTabBar(nav, TabHeight, []ui.TabItem{
		{Label: "Home", Screen: home},
		{Label: "Library", Screen: NewLibraryScreen(st)},
		{Label: "Settings", Screen: NewSettingsScreen(st)},
	})
	nav.SetTabBar(tabs)
	nav.OnClose(func() { w.Close() })

	w.AddComponent(nav)
	w.OnDrawGL(func(c *canvas.Canvas) {
		c.DrawRect(0, 0, c.Width(), c.Height(), canvas.FillPaint(colBase))
		nav.Draw(c, 0, 0, c.Width(), c.Height())
	})
	w.Show()
}
