package animation

import (
	"sync"
	"time"
)

// track is a single named animation channel within a Timeline.
type track struct {
	name      string
	from, to  float64
	startFrac float32 // 0.0–1.0 fraction of total duration
	endFrac   float32
	easing    EasingFn
	current   float64
}

// Timeline animates multiple named values along a shared time axis.
// Use AddTrack to define channels, Play to start, and Value(name) to read
// the current value each frame.
//
// Example:
//
//	tl := animation.NewTimeline(600 * time.Millisecond)
//	tl.AddTrack("x",       -200, 0,   0.0, 0.8, animation.EaseOutBack)
//	tl.AddTrack("opacity",    0, 1.0, 0.0, 0.6, animation.EaseOutQuad)
//	tl.Play()
//
//	// In OnDraw:
//	canvas.Translate(float32(tl.Value("x")), 0)
type Timeline struct {
	mu         sync.Mutex
	duration   float64 // total seconds
	elapsed    float64
	playing    bool
	tracks     []*track
	completeFn []func()
}

// NewTimeline creates a Timeline with the given total duration.
func NewTimeline(d time.Duration) *Timeline {
	return &Timeline{duration: d.Seconds()}
}

// AddTrack adds a named value channel.
// startFrac and endFrac are fractions of total duration (0.0–1.0).
func (tl *Timeline) AddTrack(name string, from, to float64, startFrac, endFrac float32, easing EasingFn) *Timeline {
	tl.mu.Lock()
	tl.tracks = append(tl.tracks, &track{
		name: name, from: from, to: to,
		startFrac: startFrac, endFrac: endFrac,
		easing:  easing,
		current: from,
	})
	tl.mu.Unlock()
	return tl
}

// Play starts or resumes the timeline.
func (tl *Timeline) Play() *Timeline {
	tl.mu.Lock()
	tl.playing = true
	tl.mu.Unlock()
	return tl
}

// Stop pauses the timeline at its current position.
func (tl *Timeline) Stop() {
	tl.mu.Lock()
	tl.playing = false
	tl.mu.Unlock()
}

// Seek jumps to a specific time position (seconds from start).
func (tl *Timeline) Seek(t float32) {
	tl.mu.Lock()
	tl.elapsed = float64(t)
	if tl.elapsed < 0 {
		tl.elapsed = 0
	}
	if tl.elapsed > tl.duration {
		tl.elapsed = tl.duration
	}
	tl.computeTracks()
	tl.mu.Unlock()
}

// Value returns the current value for a named track.
// Returns 0 if the track name is not found.
func (tl *Timeline) Value(name string) float64 {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	for _, tr := range tl.tracks {
		if tr.name == name {
			return tr.current
		}
	}
	return 0
}

// IsPlaying reports whether the timeline is currently running.
func (tl *Timeline) IsPlaying() bool {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	return tl.playing
}

// OnComplete registers a callback called when the timeline finishes.
func (tl *Timeline) OnComplete(fn func()) *Timeline {
	tl.mu.Lock()
	tl.completeFn = append(tl.completeFn, fn)
	tl.mu.Unlock()
	return tl
}

// Tick advances the timeline by delta seconds. Called by the render loop.
// Returns false when the timeline has finished.
func (tl *Timeline) Tick(delta float64) bool {
	tl.mu.Lock()
	if !tl.playing {
		tl.mu.Unlock()
		return true
	}
	tl.elapsed += delta
	done := false
	if tl.elapsed >= tl.duration {
		tl.elapsed = tl.duration
		tl.playing = false
		done = true
	}
	tl.computeTracks()
	fns := tl.completeFn
	tl.mu.Unlock()

	if done {
		for _, fn := range fns {
			fn()
		}
		return false
	}
	return true
}

// computeTracks updates each track's current value based on elapsed time.
// Must be called with mu held.
func (tl *Timeline) computeTracks() {
	progress := float32(tl.elapsed / tl.duration)
	for _, tr := range tl.tracks {
		if progress <= tr.startFrac {
			tr.current = tr.from
			continue
		}
		if progress >= tr.endFrac {
			tr.current = tr.to
			continue
		}
		// Local progress within this track's window.
		local := float64((progress - tr.startFrac) / (tr.endFrac - tr.startFrac))
		if tr.easing != nil {
			local = tr.easing(local)
		}
		tr.current = tr.from + (tr.to-tr.from)*local
	}
}
