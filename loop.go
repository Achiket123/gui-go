package goui

import (
	"time"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/platform"
)

const targetFPS = 60
const frameDuration = time.Second / targetFPS

// loop manages the per-frame rendering cycle for a Window.
type loop struct {
	win *Window
}

func newLoop(w *Window) *loop {
	return &loop{win: w}
}

// run is the main render loop. It blocks until w.running becomes false.
func (l *loop) run() {
	w := l.win
	w.running = true

	lastFrame := time.Now()

	for w.running {
		frameStart := time.Now()

		// Delta = time since the last frame completed (frame-to-frame).
		delta := frameStart.Sub(lastFrame).Seconds()
		lastFrame = frameStart
		if delta > 0.1 {
			delta = 0.1
		}

		// 1. Drain all pending X11 events (non-blocking).
		for platform.Pending(w.display) > 0 {
			ev := platform.NextEvent(w.display)
			w.dispatchEvent(ev)
			if !w.running {
				return
			}
		}

		// 2. Tick all v1 animations.
		l.tickAnimations(delta)

		// 3. Render frame — branch on renderer mode.
		if w.rendererIsGL && w.renderer != nil {
			// --- v2: OpenGL path ---
			w.renderer.BeginFrame([4]float32{0.08, 0.08, 0.12, 1})
			if w.onDrawCanvas != nil || len(w.components) > 0 {
				c := canvas.NewCanvas(w.renderer, w.width, w.height)
				// Tick and draw registered UI components.
				for _, comp := range w.components {
					comp.Tick(delta)
					comp.Draw(c, 0, 0, float32(w.width), float32(w.height))
				}
				if w.onDrawCanvas != nil {
					w.onDrawCanvas(c)
				}
			}
			w.renderer.EndFrame()
		} else {
			// --- v1: Xlib Pixmap path ---
			if w.onDraw != nil || len(w.components) > 0 {
				c := newCanvas(w)
				for _, comp := range w.components {
					comp.Tick(delta)
					// Note: v1 components use the old Xlib canvas which is different.
					// Implementation for v1 components not fully supported here.
				}
				if w.onDraw != nil {
					w.onDraw(c)
				}
			}
			if w.pixmap != 0 {
				platform.CopyArea(w.display, w.pixmap, w.xwin, w.gc,
					0, 0, w.width, w.height, 0, 0)
			}
			platform.Flush(w.display)
		}

		// 4. Sleep to maintain target FPS.
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}
	}
}

// tickAnimations advances v1 Tween/Sprite/Sequence plus v2 Controllers/Timelines.
func (l *loop) tickAnimations(delta float64) {
	w := l.win

	// v1 animations
	aliveV1 := w.animations[:0]
	for _, a := range w.animations {
		a.Tick(delta)
		if !a.IsFinished() {
			aliveV1 = append(aliveV1, a)
		}
	}
	w.animations = aliveV1

	// v2 controllers
	aliveCtrl := w.controllers[:0]
	for _, c := range w.controllers {
		if c.ctrl.Tick(delta) {
			aliveCtrl = append(aliveCtrl, c)
		}
	}
	w.controllers = aliveCtrl

	// v2 timelines
	aliveTl := w.timelines[:0]
	for _, t := range w.timelines {
		if t.tl.Tick(delta) {
			aliveTl = append(aliveTl, t)
		}
	}
	w.timelines = aliveTl
}
