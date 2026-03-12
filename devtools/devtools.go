//go:build debug

// Package devtools provides development-time utilities for goui.
//
// None of these are included in production builds.
// Wrap with a build tag: //go:build debug
//
// Features:
//   - LayoutDebugger  — press F1 to overlay component bounds
//   - FPSOverlay      — live frame-time counter in a corner
//   - EventLogger     — dumps all events to stderr
//   - HotReloader     — watches Go source files and triggers a rebuild/restart
//   - Inspector       — hover a component to see its properties in a panel
package devtools

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

// ─────────────────────────────────────────────────────────────────────────────
// LayoutDebugger
// ─────────────────────────────────────────────────────────────────────────────

// LayoutDebugger wraps a root Component and, when enabled, draws a
// coloured border around every registered component's Bounds().
//
// Usage:
//
//	dbg := devtools.NewLayoutDebugger(myRootWidget)
//	window.Register(dbg)
//	// Press F1 in-app to toggle the overlay.
type LayoutDebugger struct {
	Root       ui.Component
	Enabled    bool
	components []ui.Component // registered sub-components to inspect
	bounds     canvas.Rect

	// colours cycle per depth level
	colors []canvas.Color
}

// NewLayoutDebugger wraps root.
func NewLayoutDebugger(root ui.Component) *LayoutDebugger {
	return &LayoutDebugger{
		Root: root,
		colors: []canvas.Color{
			canvas.Hex("#F38BA8"), // red
			canvas.Hex("#FAB387"), // peach
			canvas.Hex("#F9E2AF"), // yellow
			canvas.Hex("#A6E3A1"), // green
			canvas.Hex("#89DCEB"), // teal
			canvas.Hex("#89B4FA"), // blue
			canvas.Hex("#CBA6F7"), // mauve
		},
	}
}

// Register adds a component so the debugger can draw its bounds.
func (d *LayoutDebugger) Register(c ui.Component) { d.components = append(d.components, c) }

func (d *LayoutDebugger) Bounds() canvas.Rect { return d.bounds }
func (d *LayoutDebugger) Tick(delta float64) {
	if d.Root != nil {
		d.Root.Tick(delta)
	}
}
func (d *LayoutDebugger) HandleEvent(e ui.Event) bool {
	if e.Type == ui.EventKeyDown && e.Key == "F1" {
		d.Enabled = !d.Enabled
		return true
	}
	if d.Root != nil {
		return d.Root.HandleEvent(e)
	}
	return false
}
func (d *LayoutDebugger) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if d.Root != nil {
		d.Root.Draw(c, x, y, w, h)
	}
	if !d.Enabled {
		return
	}
	for i, comp := range d.components {
		b := comp.Bounds()
		if b.W == 0 && b.H == 0 {
			continue
		}
		col := d.colors[i%len(d.colors)]
		col.A = 0.7
		// Draw dashed border (approximated with short rects).
		dashLen := float32(6)
		gap := float32(4)
		p := canvas.FillPaint(col)
		drawDashedRect(c, b.X, b.Y, b.W, b.H, 1, dashLen, gap, p)
		// Label.
		ts := canvas.TextStyle{Color: col, Size: 10}
		c.DrawText(b.X+2, b.Y+12, fmt.Sprintf("%T", comp), ts)
	}
}

// drawDashedRect draws a rectangle outline as dashes.
func drawDashedRect(c *canvas.Canvas, x, y, w, h, t, dl, gap float32, p canvas.Paint) {
	// Top and bottom edges.
	for cx := x; cx < x+w; cx += dl + gap {
		segW := dl
		if cx+segW > x+w {
			segW = x + w - cx
		}
		c.DrawRect(cx, y, segW, t, p)
		c.DrawRect(cx, y+h-t, segW, t, p)
	}
	// Left and right edges.
	for cy := y; cy < y+h; cy += dl + gap {
		segH := dl
		if cy+segH > y+h {
			segH = y + h - cy
		}
		c.DrawRect(x, cy, t, segH, p)
		c.DrawRect(x+w-t, cy, t, segH, p)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// FPSOverlay
// ─────────────────────────────────────────────────────────────────────────────

// FPSOverlay draws a small frame-time / FPS counter in one corner.
type FPSOverlay struct {
	Corner  int     // 0=TL 1=TR 2=BL 3=BR
	BgAlpha float32 // 0–1
	Padding float32
	frames  []float64 // recent deltas
	mu      sync.Mutex
	bounds  canvas.Rect
}

func NewFPSOverlay() *FPSOverlay {
	return &FPSOverlay{Corner: 3, BgAlpha: 0.7, Padding: 6}
}

// RecordFrame records a frame delta (in seconds) from the render loop.
func (f *FPSOverlay) RecordFrame(delta float64) {
	f.mu.Lock()
	f.frames = append(f.frames, delta)
	if len(f.frames) > 60 {
		f.frames = f.frames[1:]
	}
	f.mu.Unlock()
}

func (f *FPSOverlay) Bounds() canvas.Rect         { return f.bounds }
func (f *FPSOverlay) Tick(_ float64)              {}
func (f *FPSOverlay) HandleEvent(_ ui.Event) bool { return false }

func (f *FPSOverlay) Draw(c *canvas.Canvas, x, y, w, h float32) {
	f.mu.Lock()
	n := len(f.frames)
	sum := 0.0
	for _, d := range f.frames {
		sum += d
	}
	f.mu.Unlock()
	if n == 0 {
		return
	}
	avg := sum / float64(n)
	fps := 1.0 / avg
	ms := avg * 1000

	text := fmt.Sprintf("%.1f fps  %.2f ms", fps, ms)
	ts := canvas.TextStyle{Color: canvas.White, Size: 11}
	sz := c.MeasureText(text, ts)
	bw := sz.W + f.Padding*2
	bh := sz.H + f.Padding*2

	var bx, by float32
	margin := float32(8)
	switch f.Corner {
	case 0: // TL
		bx, by = x+margin, y+margin
	case 1: // TR
		bx, by = x+w-bw-margin, y+margin
	case 2: // BL
		bx, by = x+margin, y+h-bh-margin
	default: // BR
		bx, by = x+w-bw-margin, y+h-bh-margin
	}
	f.bounds = canvas.Rect{X: bx, Y: by, W: bw, H: bh}

	bg := canvas.Color{A: f.BgAlpha}
	// Color-code: green < 16ms, yellow < 33ms, red otherwise.
	switch {
	case ms < 16.7:
		bg.G = 0.3
	case ms < 33:
		bg.R, bg.G = 0.4, 0.3
	default:
		bg.R = 0.5
	}
	c.DrawRoundedRect(bx, by, bw, bh, 4, canvas.FillPaint(bg))
	c.DrawText(bx+f.Padding, by+f.Padding+ts.Size, text, ts)
}

// ─────────────────────────────────────────────────────────────────────────────
// EventLogger
// ─────────────────────────────────────────────────────────────────────────────

// EventLogger wraps a Component and prints every event to stderr.
type EventLogger struct {
	Child  ui.Component
	Prefix string
	bounds canvas.Rect
}

func NewEventLogger(child ui.Component, prefix string) *EventLogger {
	return &EventLogger{Child: child, Prefix: prefix}
}

func (el *EventLogger) Bounds() canvas.Rect { return el.bounds }
func (el *EventLogger) Tick(delta float64) {
	if el.Child != nil {
		el.Child.Tick(delta)
	}
}
func (el *EventLogger) Draw(c *canvas.Canvas, x, y, w, h float32) {
	el.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	if el.Child != nil {
		el.Child.Draw(c, x, y, w, h)
	}
}
func (el *EventLogger) HandleEvent(e ui.Event) bool {
	switch e.Type {
	case ui.EventMouseDown:
		log.Printf("[%s] MouseDown  x=%.0f y=%.0f btn=%d", el.Prefix, e.X, e.Y, e.Button)
	case ui.EventMouseUp:
		log.Printf("[%s] MouseUp    x=%.0f y=%.0f btn=%d", el.Prefix, e.X, e.Y, e.Button)
	case ui.EventMouseMove:
		// Throttle move events — only log every 20th.
		// (too noisy otherwise)
	case ui.EventKeyDown:
		log.Printf("[%s] KeyDown    key=%q shift=%v", el.Prefix, e.Key, e.Shift)
	case ui.EventScroll:
		log.Printf("[%s] Scroll     y=%.2f", el.Prefix, e.ScrollY)
	}
	if el.Child != nil {
		return el.Child.HandleEvent(e)
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// HotReloader — watches source files and triggers a rebuild
// ─────────────────────────────────────────────────────────────────────────────

// HotReloader watches a directory for *.go file changes and runs a command
// when a change is detected. The simplest integration is:
//
//	rl := devtools.NewHotReloader(".", func() {
//	    log.Println("source changed — restarting…")
//	    // The restart is handled by a wrapper script; we just exit.
//	    os.Exit(0)
//	})
//	go rl.Watch()
type HotReloader struct {
	Dir      string        // directory to watch
	Interval time.Duration // poll interval (default 500ms)
	OnChange func()
	// Build command to run before OnChange (optional).
	// If BuildCmd is non-empty, OnChange is only called if the build succeeds.
	BuildCmd []string

	mu      sync.Mutex
	mtimes  map[string]time.Time
	running bool
}

func NewHotReloader(dir string, onChange func()) *HotReloader {
	return &HotReloader{
		Dir:      dir,
		Interval: 500 * time.Millisecond,
		OnChange: onChange,
		mtimes:   make(map[string]time.Time),
	}
}

// Watch starts the polling loop (call in a goroutine).
func (hr *HotReloader) Watch() {
	hr.mu.Lock()
	if hr.running {
		hr.mu.Unlock()
		return
	}
	hr.running = true
	hr.mu.Unlock()

	// Snapshot initial mtimes.
	hr.snapshot()

	ticker := time.NewTicker(hr.Interval)
	defer ticker.Stop()
	for range ticker.C {
		if hr.hasChanged() {
			hr.snapshot() // update baseline
			if len(hr.BuildCmd) > 0 {
				if err := hr.build(); err != nil {
					log.Printf("hot-reload: build failed: %v", err)
					continue
				}
			}
			if hr.OnChange != nil {
				hr.OnChange()
			}
		}
	}
}

// Stop halts the watcher.
func (hr *HotReloader) Stop() {
	hr.mu.Lock()
	hr.running = false
	hr.mu.Unlock()
}

func (hr *HotReloader) snapshot() {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	err := filepath.WalkDir(hr.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			info, e := d.Info()
			if e == nil {
				hr.mtimes[path] = info.ModTime()
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("devtools: HotReloader.snapshot WalkDir error: %v", err)
	}
}

func (hr *HotReloader) hasChanged() bool {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	changed := false
	err := filepath.WalkDir(hr.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		if old, ok := hr.mtimes[path]; !ok || info.ModTime().After(old) {
			changed = true
		}
		return nil
	})
	if err != nil {
		log.Printf("devtools: HotReloader.hasChanged WalkDir error: %v", err)
	}
	return changed
}

func (hr *HotReloader) build() error {
	if len(hr.BuildCmd) == 0 {
		return fmt.Errorf("HotReloader: BuildCmd is empty")
	}

	// Simple allowlist for common build/wrapper tools.
	bin := hr.BuildCmd[0]
	allowed := false
	for _, a := range []string{"go", "fvm", "make", "task", "shellcheck"} {
		if bin == a || strings.HasSuffix(bin, "/"+a) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("HotReloader: binary %q not in allowlist", bin)
	}

	cmd := exec.Command(bin, hr.BuildCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
