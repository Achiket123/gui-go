package animation

import "time"

// Tween smoothly interpolates a float64 value from Start to End over Duration.
//
// Example:
//
//	t := animation.NewTween(0, 500, 1500*time.Millisecond, animation.EaseInOutQuad)
//	w.AddAnimation(t)
//	w.OnDraw(func(c *goui.Canvas) {
//	    x := t.Value()
//	    c.FillCircle(int(x), 300, 20)
//	})
type Tween struct {
	startValue float64
	endValue   float64
	duration   float64 // seconds
	elapsed    float64 // seconds accumulated
	easingFn   EasingFn
	looping    bool
	pingPong   bool
	forward    bool // current direction for ping-pong
	finished   bool

	onComplete func()
}

// NewTween creates a new Tween that animates from→to over duration using easingFn.
func NewTween(from, to float64, duration time.Duration, easingFn EasingFn) *Tween {
	if easingFn == nil {
		easingFn = Linear
	}
	return &Tween{
		startValue: from,
		endValue:   to,
		duration:   duration.Seconds(),
		easingFn:   easingFn,
		forward:    true,
	}
}

// Tick advances the tween by delta seconds. Called by the render loop.
func (t *Tween) Tick(delta float64) {
	if t.finished {
		return
	}
	t.elapsed += delta
	if t.elapsed >= t.duration {
		if t.pingPong {
			// Flip direction
			t.elapsed = 0
			t.startValue, t.endValue = t.endValue, t.startValue
			t.forward = !t.forward
			if t.looping {
				return // keep going
			}
			// Without looping, one round-trip = done
			if !t.forward { // just flipped back to original direction → done after back trip
				t.finished = true
				if t.onComplete != nil {
					t.onComplete()
				}
			}
		} else if t.looping {
			t.elapsed = 0
		} else {
			t.elapsed = t.duration
			t.finished = true
			if t.onComplete != nil {
				t.onComplete()
			}
		}
	}
}

// Value returns the current interpolated value.
func (t *Tween) Value() float64 {
	if t.duration <= 0 {
		return t.endValue
	}
	raw := t.elapsed / t.duration
	if raw > 1 {
		raw = 1
	}
	eased := t.easingFn(raw)
	return t.startValue + (t.endValue-t.startValue)*eased
}

// Reset restarts the tween from the beginning.
func (t *Tween) Reset() {
	t.elapsed = 0
	t.finished = false
}

// SetLoop enables or disables infinite looping.
func (t *Tween) SetLoop(loop bool) {
	t.looping = loop
}

// SetPingPong enables forward-then-backward oscillation.
func (t *Tween) SetPingPong(pp bool) {
	t.pingPong = pp
}

// SetOnComplete registers a callback fired when the tween finishes.
func (t *Tween) SetOnComplete(fn func()) {
	t.onComplete = fn
}

// IsFinished returns true when a non-looping tween has completed.
func (t *Tween) IsFinished() bool {
	return t.finished
}

// Map maps a normalized value t (0.0–1.0) through this tween's easing and range.
// Used with AnimationController.Drive: ctrl.Drive(tween) = tween.Map(ctrl.Value()).
// Unlike Value(), Map does not depend on elapsed time — the controller drives timing.
func (t *Tween) Map(normalized float64) float64 {
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}
	eased := t.easingFn(normalized)
	return t.startValue + (t.endValue-t.startValue)*eased
}
