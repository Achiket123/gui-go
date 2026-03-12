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

		delta := frameStart.Sub(lastFrame).Seconds()
		lastFrame = frameStart
		if delta > 0.1 {
			delta = 0.1
		}

		// 1. Drain all pending GLFW events.
		platform.PollEvents()
		if w.handle.ShouldClose() {
			w.running = false
			if w.onClose != nil {
				w.onClose()
			}
			break
		}

		for _, ev := range coalesceScroll(w.handle.DrainEvents()) {
			w.dispatchEvent(ev)
			if !w.running {
				break
			}
		}

		if !w.running {
			break
		}

		// 2. Tick all animations.
		l.tickAnimations(delta)

		// 3. Render frame — branch on renderer mode.
		if w.rendererIsGL && w.renderer != nil {
			// --- v2: OpenGL path ---
			w.renderer.BeginFrame([4]float32{0.08, 0.08, 0.12, 1})
			if w.onDrawCanvas != nil || len(w.components) > 0 {
				c := canvas.NewCanvas(w.renderer, w.width, w.height)
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
			// --- v1 fallback ---
			w.handle.SwapBuffers()
		}

		// 4. Sleep to maintain target FPS.
		elapsed := time.Since(frameStart)
		if elapsed < frameDuration {
			time.Sleep(frameDuration - elapsed)
		}
	}
}

// coalesceScroll merges consecutive EventScroll events at the same cursor
// position into a single event by summing their ScrollX/ScrollY deltas.
// This prevents trackpad flicks (which fire many events per PollEvents call)
// from accumulating an unbounded velocity before the first Tick.
func coalesceScroll(evs []platform.EventData) []platform.EventData {
	if len(evs) == 0 {
		return evs
	}
	out := evs[:0]
	for _, ev := range evs {
		if ev.Type == platform.EventScroll && len(out) > 0 && out[len(out)-1].Type == platform.EventScroll {
			out[len(out)-1].ScrollX += ev.ScrollX
			out[len(out)-1].ScrollY += ev.ScrollY
		} else {
			out = append(out, ev)
		}
	}
	return out
}

func (l *loop) tickAnimations(delta float64) {
	w := l.win

	aliveV1 := w.animations[:0]
	for _, a := range w.animations {
		a.Tick(delta)
		if !a.IsFinished() {
			aliveV1 = append(aliveV1, a)
		}
	}
	w.animations = aliveV1

	aliveCtrl := w.controllers[:0]
	for _, c := range w.controllers {
		if c.ctrl.Tick(delta) {
			aliveCtrl = append(aliveCtrl, c)
		}
	}
	w.controllers = aliveCtrl

	aliveTl := w.timelines[:0]
	for _, t := range w.timelines {
		if t.tl.Tick(delta) {
			aliveTl = append(aliveTl, t)
		}
	}
	w.timelines = aliveTl
}
