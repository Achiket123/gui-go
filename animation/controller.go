package animation

import (
	"sync"
	"time"
)

// AnimStatus represents the playback state of an AnimationController.
type AnimStatus int

const (
	StatusIdle     AnimStatus = iota // Not started or stopped
	StatusForward                    // Playing 0 → 1
	StatusReverse                    // Playing 1 → 0
	StatusComplete                   // Reached end of one-shot animation
)

// AnimationController drives a normalized value from 0.0 to 1.0 over a duration.
// Unlike Tween (which self-ticks), the controller is ticked externally by the
// render loop via Tick(delta). Use Drive(tween) to map the 0–1 value to any range.
//
// Example:
//
//	ctrl := animation.NewController(400 * time.Millisecond)
//	ctrl.Forward()
//
//	// in OnDraw:
//	x := ctrl.Drive(animation.NewTween(0, 300, 0, animation.EaseOutBack))
type AnimationController struct {
	mu       sync.Mutex
	duration float64 // seconds
	value    float64 // current normalized value [0, 1]
	status   AnimStatus
	dir      float64 // +1 forward, -1 reverse

	repeat   int  // 0=none, -1=infinite, >0=count
	pingpong bool // reverse direction at boundaries

	tickFns     []func(float64)
	completeFns []func()
}

// NewController creates an AnimationController with the given duration.
func NewController(d time.Duration) *AnimationController {
	return &AnimationController{duration: d.Seconds(), dir: 1}
}

// Forward starts animating from current value → 1.0.
func (c *AnimationController) Forward() {
	c.mu.Lock()
	c.dir = 1
	c.status = StatusForward
	c.mu.Unlock()
}

// Reverse starts animating from current value → 0.0.
func (c *AnimationController) Reverse() {
	c.mu.Lock()
	c.dir = -1
	c.status = StatusReverse
	c.mu.Unlock()
}

// Repeat sets loop count (-1 = forever). Call before Forward/Reverse.
func (c *AnimationController) Repeat(count int) *AnimationController {
	c.mu.Lock()
	c.repeat = count
	c.mu.Unlock()
	return c
}

// PingPong makes the animation oscillate back and forth.
func (c *AnimationController) PingPong() *AnimationController {
	c.mu.Lock()
	c.pingpong = true
	c.repeat = -1
	c.mu.Unlock()
	return c
}

// Stop pauses at current value.
func (c *AnimationController) Stop() {
	c.mu.Lock()
	c.status = StatusIdle
	c.mu.Unlock()
}

// Reset jumps to 0.0 without animating.
func (c *AnimationController) Reset() {
	c.mu.Lock()
	c.value = 0
	c.status = StatusIdle
	c.mu.Unlock()
}

// JumpTo sets the value instantly (0.0–1.0).
func (c *AnimationController) JumpTo(t float64) {
	c.mu.Lock()
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	c.value = t
	c.mu.Unlock()
}

// Value returns the current normalized value [0, 1].
func (c *AnimationController) Value() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Status returns the playback status.
func (c *AnimationController) Status() AnimStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// OnTick registers a callback called every Tick with the current value.
func (c *AnimationController) OnTick(fn func(v float64)) *AnimationController {
	c.mu.Lock()
	c.tickFns = append(c.tickFns, fn)
	c.mu.Unlock()
	return c
}

// OnComplete registers a callback called when the animation finishes.
func (c *AnimationController) OnComplete(fn func()) *AnimationController {
	c.mu.Lock()
	c.completeFns = append(c.completeFns, fn)
	c.mu.Unlock()
	return c
}

// Drive maps the controller's current value through a Tween to produce a real value.
// Example: ctrl.Drive(animation.NewTween(100, 500, 0, EaseOutQuad)) → pixel x position.
func (c *AnimationController) Drive(tw *Tween) float64 {
	return tw.Map(c.Value())
}

// Tick advances the animation by delta seconds. Called by the render loop.
// Returns false when a non-looping animation completes (so the loop can stop calling it).
func (c *AnimationController) Tick(delta float64) bool {
	c.mu.Lock()
	if c.status != StatusForward && c.status != StatusReverse {
		c.mu.Unlock()
		return c.status != StatusComplete
	}
	if c.duration <= 0 {
		c.value = 1
		c.status = StatusComplete
		c.mu.Unlock()
		c.fireComplete()
		return false
	}

	step := delta / c.duration * c.dir
	c.value += step

	completed := false
	if c.value >= 1 {
		c.value = 1
		if c.pingpong {
			c.dir = -1
			c.status = StatusReverse
		} else if c.repeat != 0 {
			if c.repeat > 0 {
				c.repeat--
			}
			c.value = 0
		} else {
			c.status = StatusComplete
			completed = true
		}
	} else if c.value <= 0 {
		c.value = 0
		if c.pingpong {
			c.dir = 1
			c.status = StatusForward
		} else if c.repeat != 0 {
			if c.repeat > 0 {
				c.repeat--
			}
			c.value = 1
		} else {
			c.status = StatusComplete
			completed = true
		}
	}

	v := c.value
	fns := c.tickFns
	c.mu.Unlock()

	for _, fn := range fns {
		fn(v)
	}
	if completed {
		c.fireComplete()
		return false
	}
	return true
}

func (c *AnimationController) fireComplete() {
	c.mu.Lock()
	fns := c.completeFns
	c.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// IsDone returns true for a non-looping controller that has completed.
func (c *AnimationController) IsDone() bool {
	return c.Status() == StatusComplete
}
