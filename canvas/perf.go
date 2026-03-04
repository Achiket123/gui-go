// Package canvas — perf.go
//
// Performance helpers:
//   DirtyTracker  — per-region invalidation; merges overlapping rects; thread-safe
//   FrameThrottle — adaptive FPS (active/idle) with delta-time; no busy-wait
//   RenderCache   — per-component dirty flag for expensive sub-tree skipping
package canvas

import (
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// DirtyTracker
// ─────────────────────────────────────────────────────────────────────────────

// DirtyTracker records rectangular regions that need to be redrawn.
// It merges overlapping or adjacent rects automatically.
//
//	dirty := canvas.NewDirtyTracker(1280, 800)
//
//	// Invalidate one widget's area:
//	dirty.Invalidate(canvas.Rect{X: 100, Y: 200, W: 300, H: 50})
//
//	// In the render loop:
//	if !dirty.HasAnyDirty() { return }
//	dirty.Each(func(r canvas.Rect) { redraw(r) })
//	dirty.Clean()
type DirtyTracker struct {
	mu     sync.Mutex
	rects  []Rect
	screenW, screenH float32
}

// NewDirtyTracker creates a DirtyTracker for a screen of the given size.
func NewDirtyTracker(w, h float32) *DirtyTracker {
	return &DirtyTracker{screenW: w, screenH: h}
}

// Resize updates the screen size (call from window resize handler).
func (dt *DirtyTracker) Resize(w, h float32) {
	dt.mu.Lock()
	dt.screenW = w
	dt.screenH = h
	dt.mu.Unlock()
}

// Invalidate marks r as dirty. Overlapping dirty rects are merged.
func (dt *DirtyTracker) Invalidate(r Rect) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	for i, existing := range dt.rects {
		if rectsOverlapOrAdjacent(existing, r) {
			dt.rects[i] = unionRect(existing, r)
			return
		}
	}
	dt.rects = append(dt.rects, r)
}

// InvalidateAll marks the entire screen as dirty.
func (dt *DirtyTracker) InvalidateAll() {
	dt.mu.Lock()
	dt.rects = []Rect{{0, 0, dt.screenW, dt.screenH}}
	dt.mu.Unlock()
}

// HasAnyDirty returns true if there are any dirty regions.
func (dt *DirtyTracker) HasAnyDirty() bool {
	dt.mu.Lock()
	n := len(dt.rects)
	dt.mu.Unlock()
	return n > 0
}

// Each calls fn for every dirty rect. Safe to call from the render goroutine
// while Invalidate is called from other goroutines.
func (dt *DirtyTracker) Each(fn func(Rect)) {
	dt.mu.Lock()
	snapshot := make([]Rect, len(dt.rects))
	copy(snapshot, dt.rects)
	dt.mu.Unlock()
	for _, r := range snapshot {
		fn(r)
	}
}

// Clean clears all dirty regions (call after a full redraw).
func (dt *DirtyTracker) Clean() {
	dt.mu.Lock()
	dt.rects = dt.rects[:0]
	dt.mu.Unlock()
}

// DirtyRects returns a snapshot of the current dirty regions.
func (dt *DirtyTracker) DirtyRects() []Rect {
	dt.mu.Lock()
	out := make([]Rect, len(dt.rects))
	copy(out, dt.rects)
	dt.mu.Unlock()
	return out
}

func rectsOverlapOrAdjacent(a, b Rect) bool {
	return a.X <= b.X+b.W+1 && a.X+a.W+1 >= b.X &&
		a.Y <= b.Y+b.H+1 && a.Y+a.H+1 >= b.Y
}

func unionRect(a, b Rect) Rect {
	x0 := minF(a.X, b.X)
	y0 := minF(a.Y, b.Y)
	x1 := maxF(a.X+a.W, b.X+b.W)
	y1 := maxF(a.Y+a.H, b.Y+b.H)
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func minF(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxF(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// ─────────────────────────────────────────────────────────────────────────────
// FrameThrottle
// ─────────────────────────────────────────────────────────────────────────────

// FrameThrottle regulates the render loop rate.
//
// When the app is actively receiving input it targets activeFPS.
// After idleTimeout of no activity it drops to idleFPS to save CPU.
// Call Wait() at the top of each render loop iteration; it blocks for
// the remaining frame budget and returns the real delta time in seconds.
// Call MarkActive() on any user input event.
//
//	throttle := canvas.NewFrameThrottle(60, 10, 3*time.Second)
//
//	for { // render loop
//	    dt := throttle.Wait()
//	    render(dt)
//	}
//
//	// In event handler:
//	throttle.MarkActive()
type FrameThrottle struct {
	activeFPS   float64
	idleFPS     float64
	idleTimeout time.Duration

	lastFrame  time.Time
	lastActive time.Time
	mu         sync.Mutex
}

// NewFrameThrottle creates a FrameThrottle.
func NewFrameThrottle(activeFPS, idleFPS float64, idleTimeout time.Duration) *FrameThrottle {
	now := time.Now()
	return &FrameThrottle{
		activeFPS:   activeFPS,
		idleFPS:     idleFPS,
		idleTimeout: idleTimeout,
		lastFrame:   now,
		lastActive:  now,
	}
}

// MarkActive resets the idle timer (call on any user input).
func (ft *FrameThrottle) MarkActive() {
	ft.mu.Lock()
	ft.lastActive = time.Now()
	ft.mu.Unlock()
}

// Wait blocks until the next frame budget expires and returns the real
// delta time in seconds since the last call.
func (ft *FrameThrottle) Wait() float64 {
	ft.mu.Lock()
	isIdle := time.Since(ft.lastActive) > ft.idleTimeout
	fps := ft.activeFPS
	if isIdle {
		fps = ft.idleFPS
	}
	last := ft.lastFrame
	ft.mu.Unlock()

	frameDur := time.Duration(float64(time.Second) / fps)
	elapsed := time.Since(last)
	if remaining := frameDur - elapsed; remaining > 0 {
		time.Sleep(remaining)
	}

	now := time.Now()
	delta := now.Sub(last).Seconds()

	ft.mu.Lock()
	ft.lastFrame = now
	ft.mu.Unlock()

	return delta
}

// IsIdle returns true when the throttle is currently in idle mode.
func (ft *FrameThrottle) IsIdle() bool {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return time.Since(ft.lastActive) > ft.idleTimeout
}

// ─────────────────────────────────────────────────────────────────────────────
// RenderCache
// ─────────────────────────────────────────────────────────────────────────────

// RenderCache provides a dirty flag per component ID.
// Use it to skip re-drawing expensive sub-trees that haven't changed.
//
//	cache := canvas.NewRenderCache()
//	cache.Invalidate("sidebar")     // mark sidebar dirty
//
//	if cache.IsDirty("sidebar") {
//	    sidebar.Draw(...)
//	    cache.Clean("sidebar")
//	}
type RenderCache struct {
	mu    sync.RWMutex
	dirty map[string]bool
}

// NewRenderCache creates a RenderCache.
func NewRenderCache() *RenderCache {
	return &RenderCache{dirty: make(map[string]bool)}
}

// Invalidate marks component id as needing a redraw.
func (rc *RenderCache) Invalidate(id string) {
	rc.mu.Lock()
	rc.dirty[id] = true
	rc.mu.Unlock()
}

// InvalidateAll marks every tracked component as dirty.
func (rc *RenderCache) InvalidateAll() {
	rc.mu.Lock()
	for k := range rc.dirty {
		rc.dirty[k] = true
	}
	rc.mu.Unlock()
}

// IsDirty returns true if id needs a redraw.
// Unknown ids are treated as dirty (safe default).
func (rc *RenderCache) IsDirty(id string) bool {
	rc.mu.RLock()
	v, known := rc.dirty[id]
	rc.mu.RUnlock()
	if !known {
		return true
	}
	return v
}

// Clean marks component id as up-to-date (call after drawing it).
func (rc *RenderCache) Clean(id string) {
	rc.mu.Lock()
	rc.dirty[id] = false
	rc.mu.Unlock()
}

// CleanAll marks every component as up-to-date.
func (rc *RenderCache) CleanAll() {
	rc.mu.Lock()
	for k := range rc.dirty {
		rc.dirty[k] = false
	}
	rc.mu.Unlock()
}

