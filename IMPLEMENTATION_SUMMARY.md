# Code Context Analysis

Generated: 2026-03-04T21:03:22+05:30
Files Scanned: 12
Analysis Tools: regex, gosec, eslint, flawfinder

## Security Summary

**Total Issues: 1**

- 🔴 CRITICAL: 0
- 🟠 HIGH: 0
- 🟡 MEDIUM: 0
- 🟢 LOW: 1

**Issues by Tool:**
- regex: 1

---

## File: stream.go
Language: go | Tokens: 4553 | Size: 18215 bytes

**⚠️ Security Issues:**

🟢 **[LOW]** Line 177 - Debug Code
   *Debug code or security TODO in production*
   Tool: regex
   ```
   // logging/debugging) then forwards the value unchanged.
   ```

```go
// Package ui — stream.go
//
// Reactive stream primitives.
//
// Primitives
//   Stream[T]          — push-based multicast event channel
//   Subject[T]         — Stream with an external Push method
//   BehaviorSubject[T] — Subject that replays the latest value to new subscribers
//   Signal[T]          — reactive cell; holds a value, notifies on change
//   Computed[T]        — read-only Signal derived from other Signals
//   Effect             — side-effect that reruns when dependency streams fire
//
// All primitives are goroutine-safe.
//
// Quick example:
//
//	count := ui.NewSignal(0)
//	doubled := ui.MapSignal(count, func(n int) int { return n * 2 })
//	ui.NewEffect(func() {
//	    fmt.Println("doubled =", doubled.Get())
//	}, ui.AnyStream(doubled.Changed()))
//	count.Set(5) // prints "doubled = 10"
package ui

import (
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Subscription
// ═══════════════════════════════════════════════════════════════════════════════

// Subscription is a handle returned by every Subscribe call.
// Call Unsubscribe() to stop receiving values and release resources.
type Subscription struct {
	cancel func()
	once   sync.Once
}

// Unsubscribe stops the subscription. Safe to call multiple times and from
// multiple goroutines.
func (s *Subscription) Unsubscribe() {
	if s == nil {
		return
	}
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
}

func newSub(cancel func()) *Subscription { return &Subscription{cancel: cancel} }

// ═══════════════════════════════════════════════════════════════════════════════
// Stream[T]
// ═══════════════════════════════════════════════════════════════════════════════

type streamEntry[T any] struct {
	id uint64
	fn func(T)
}

// Stream[T] is a push-based, multicast event channel.
//
// Values are delivered synchronously to all active subscribers at the moment
// of emission.  Use Subject[T] to push values externally.
type Stream[T any] struct {
	mu  sync.RWMutex
	seq atomic.Uint64
	sub []streamEntry[T]
}

// Subscribe registers fn and returns a Subscription.
// fn is called on each emitted value until Unsubscribe is called.
func (s *Stream[T]) Subscribe(fn func(T)) *Subscription {
	id := s.seq.Add(1)
	s.mu.Lock()
	s.sub = append(s.sub, streamEntry[T]{id, fn})
	s.mu.Unlock()
	return newSub(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, e := range s.sub {
			if e.id == id {
				s.sub = append(s.sub[:i], s.sub[i+1:]...)
				return
			}
		}
	})
}

// emit delivers value to all current subscribers.
func (s *Stream[T]) emit(value T) {
	s.mu.RLock()
	cp := make([]streamEntry[T], len(s.sub))
	copy(cp, s.sub)
	s.mu.RUnlock()
	for _, e := range cp {
		e.fn(value)
	}
}

// Len returns the current subscriber count.
func (s *Stream[T]) Len() int {
	s.mu.RLock()
	n := len(s.sub)
	s.mu.RUnlock()
	return n
}

// ── Operators ─────────────────────────────────────────────────────────────────

// Map returns a new Stream that applies fn to every emitted value.
func (s *Stream[T]) Map(fn func(T) T) *Stream[T] {
	out := &Stream[T]{}
	s.Subscribe(func(v T) { out.emit(fn(v)) })
	return out
}

// Filter returns a new Stream that only forwards values for which pred is true.
func (s *Stream[T]) Filter(pred func(T) bool) *Stream[T] {
	out := &Stream[T]{}
	s.Subscribe(func(v T) {
		if pred(v) {
			out.emit(v)
		}
	})
	return out
}

// Take returns a new Stream that emits at most n values, then stops.
func (s *Stream[T]) Take(n int) *Stream[T] {
	out := &Stream[T]{}
	var (
		mu    sync.Mutex
		count int
		sub   *Subscription
	)
	sub = s.Subscribe(func(v T) {
		mu.Lock()
		if count >= n {
			mu.Unlock()
			sub.Unsubscribe()
			return
		}
		count++
		mu.Unlock()
		out.emit(v)
	})
	return out
}

// Skip returns a new Stream that skips the first n values.
func (s *Stream[T]) Skip(n int) *Stream[T] {
	out := &Stream[T]{}
	var (
		mu    sync.Mutex
		count int
	)
	s.Subscribe(func(v T) {
		mu.Lock()
		if count < n {
			count++
			mu.Unlock()
			return
		}
		mu.Unlock()
		out.emit(v)
	})
	return out
}

// Do returns a new Stream that calls side for each value (useful for
// logging/debugging) then forwards the value unchanged.
func (s *Stream[T]) Do(side func(T)) *Stream[T] {
	out := &Stream[T]{}
	s.Subscribe(func(v T) { side(v); out.emit(v) })
	return out
}

// Distinct returns a new Stream that only emits when the value has changed
// (requires T to be comparable).
func Distinct[T comparable](s *Stream[T]) *Stream[T] {
	out := &Stream[T]{}
	var (
		mu   sync.Mutex
		last T
		set  bool
	)
	s.Subscribe(func(v T) {
		mu.Lock()
		if set && v == last {
			mu.Unlock()
			return
		}
		last = v
		set = true
		mu.Unlock()
		out.emit(v)
	})
	return out
}

// Debounce returns a Stream that suppresses rapid emissions, forwarding only
// after at least wait has elapsed since the last emission.
func Debounce[T any](s *Stream[T], wait time.Duration) *Stream[T] {
	out := &Subject[T]{}
	var (
		mu    sync.Mutex
		timer *time.Timer
		last  T
	)
	s.Subscribe(func(v T) {
		mu.Lock()
		last = v
		if timer != nil {
			timer.Reset(wait)
			mu.Unlock()
			return
		}
		timer = time.AfterFunc(wait, func() {
			mu.Lock()
			val := last
			timer = nil
			mu.Unlock()
			out.Push(val)
		})
		mu.Unlock()
	})
	return out.AsStream()
}

// Throttle returns a Stream that forwards at most one emission per wait period.
func Throttle[T any](s *Stream[T], wait time.Duration) *Stream[T] {
	out := &Subject[T]{}
	var (
		mu      sync.Mutex
		blocked bool
	)
	s.Subscribe(func(v T) {
		mu.Lock()
		if blocked {
			mu.Unlock()
			return
		}
		blocked = true
		mu.Unlock()
		out.Push(v)
		time.AfterFunc(wait, func() {
			mu.Lock()
			blocked = false
			mu.Unlock()
		})
	})
	return out.AsStream()
}

// Scan returns a Stream that accumulates a running value using fn(acc, value).
func Scan[T, A any](s *Stream[T], seed A, fn func(A, T) A) *Stream[A] {
	out := &Stream[A]{}
	var mu sync.Mutex
	acc := seed
	s.Subscribe(func(v T) {
		mu.Lock()
		acc = fn(acc, v)
		cur := acc
		mu.Unlock()
		out.emit(cur)
	})
	return out
}

// Buffer collects up to size values then emits the slice.
func Buffer[T any](s *Stream[T], size int) *Stream[[]T] {
	out := &Stream[[]T]{}
	var (
		mu  sync.Mutex
		buf []T
	)
	s.Subscribe(func(v T) {
		mu.Lock()
		buf = append(buf, v)
		if len(buf) >= size {
			batch := make([]T, len(buf))
			copy(batch, buf)
			buf = buf[:0]
			mu.Unlock()
			out.emit(batch)
			return
		}
		mu.Unlock()
	})
	return out
}

// ── Merge/Combine ─────────────────────────────────────────────────────────────

// Merge returns a Stream that emits whenever any of the source streams emit.
func Merge[T any](streams ...*Stream[T]) *Stream[T] {
	out := &Stream[T]{}
	for _, st := range streams {
		st.Subscribe(func(v T) { out.emit(v) })
	}
	return out
}

// Zip2 pairs emissions from two streams. Emits when both have a pending value.
func Zip2[A, B any](a *Stream[A], b *Stream[B]) *Stream[[2]any] {
	out := &Stream[[2]any]{}
	var (
		mu   sync.Mutex
		aQ   []A
		bQ   []B
	)
	a.Subscribe(func(v A) {
		mu.Lock()
		aQ = append(aQ, v)
		if len(bQ) > 0 {
			pair := [2]any{aQ[0], bQ[0]}
			aQ, bQ = aQ[1:], bQ[1:]
			mu.Unlock()
			out.emit(pair)
			return
		}
		mu.Unlock()
	})
	b.Subscribe(func(v B) {
		mu.Lock()
		bQ = append(bQ, v)
		if len(aQ) > 0 {
			pair := [2]any{aQ[0], bQ[0]}
			aQ, bQ = aQ[1:], bQ[1:]
			mu.Unlock()
			out.emit(pair)
			return
		}
		mu.Unlock()
	})
	return out
}

// WithLatestFrom returns a Stream that emits [main, latest-b] whenever main
// emits, using the most recent value from b (b must have emitted at least once).
func WithLatestFrom[A, B any](main *Stream[A], b *Stream[B]) *Stream[[2]any] {
	out := &Stream[[2]any]{}
	var (
		mu    sync.Mutex
		latB  B
		hasB  bool
	)
	b.Subscribe(func(v B) {
		mu.Lock()
		latB = v
		hasB = true
		mu.Unlock()
	})
	main.Subscribe(func(v A) {
		mu.Lock()
		if !hasB {
			mu.Unlock()
			return
		}
		pair := [2]any{v, latB}
		mu.Unlock()
		out.emit(pair)
	})
	return out
}

// ─ Channel bridge ─────────────────────────────────────────────────────────────

// FromChannel wraps a Go channel in a Stream. The goroutine exits when ch closes.
func FromChannel[T any](ch <-chan T) *Stream[T] {
	s := &Subject[T]{}
	go func() {
		for v := range ch {
			s.Push(v)
		}
	}()
	return s.AsStream()
}

// ToChannel drains a Stream into a buffered channel (capacity buf).
// Returns the channel and a Subscription; unsubscribe to stop.
func ToChannel[T any](stream *Stream[T], buf int) (<-chan T, *Subscription) {
	ch := make(chan T, buf)
	sub := stream.Subscribe(func(v T) {
		select {
		case ch <- v:
		default: // drop if full; callers should size buf appropriately
		}
	})
	return ch, sub
}

// ═══════════════════════════════════════════════════════════════════════════════
// Subject[T]
// ═══════════════════════════════════════════════════════════════════════════════

// Subject[T] is a Stream with an external Push method.
type Subject[T any] struct {
	Stream[T]
}

// NewSubject creates a new Subject.
func NewSubject[T any]() *Subject[T] { return &Subject[T]{} }

// Push emits value to all current subscribers.
func (s *Subject[T]) Push(value T) { s.emit(value) }

// AsStream returns the read-only Stream view.
func (s *Subject[T]) AsStream() *Stream[T] { return &s.Stream }

// ═══════════════════════════════════════════════════════════════════════════════
// BehaviorSubject[T]
// ═══════════════════════════════════════════════════════════════════════════════

// BehaviorSubject[T] behaves like Subject[T] but replays the latest value to
// every new subscriber immediately on Subscribe.
type BehaviorSubject[T any] struct {
	mu      sync.RWMutex
	current T
	inner   Subject[T]
}

// NewBehaviorSubject creates a BehaviorSubject with an initial value.
func NewBehaviorSubject[T any](initial T) *BehaviorSubject[T] {
	return &BehaviorSubject[T]{current: initial}
}

// Push emits a new value and stores it as current.
func (b *BehaviorSubject[T]) Push(value T) {
	b.mu.Lock()
	b.current = value
	b.mu.Unlock()
	b.inner.Push(value)
}

// Value returns the current (most recently pushed) value.
func (b *BehaviorSubject[T]) Value() T {
	b.mu.RLock()
	v := b.current
	b.mu.RUnlock()
	return v
}

// Subscribe registers fn and immediately calls it with the current value.
func (b *BehaviorSubject[T]) Subscribe(fn func(T)) *Subscription {
	sub := b.inner.Subscribe(fn)
	b.mu.RLock()
	cur := b.current
	b.mu.RUnlock()
	fn(cur)
	return sub
}

// AsStream exposes the inner stream (no replay on new subscribes from this view).
func (b *BehaviorSubject[T]) AsStream() *Stream[T] { return &b.inner.Stream }

// ═══════════════════════════════════════════════════════════════════════════════
// Signal[T]  —  reactive cell
// ═══════════════════════════════════════════════════════════════════════════════

// Signal[T] holds a comparable value and notifies subscribers when it changes.
//
// Notifications are skipped when the new value equals the current value (==).
// For slice/map/struct values, use SignalAny[T] which always notifies.
type Signal[T comparable] struct {
	mu   sync.RWMutex
	val  T
	subj Subject[T]
}

// NewSignal creates a Signal with the given initial value.
func NewSignal[T comparable](initial T) *Signal[T] {
	return &Signal[T]{val: initial}
}

// Get returns the current value.
func (s *Signal[T]) Get() T {
	s.mu.RLock()
	v := s.val
	s.mu.RUnlock()
	return v
}

// Set stores v. If v == current, no notification is emitted.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	if s.val == v {
		s.mu.Unlock()
		return
	}
	s.val = v
	s.mu.Unlock()
	s.subj.Push(v)
}

// Update applies fn to the current value and sets the result.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	next := fn(s.val)
	changed := s.val != next
	s.val = next
	s.mu.Unlock()
	if changed {
		s.subj.Push(next)
	}
}

// Subscribe registers fn and immediately calls it with the current value.
func (s *Signal[T]) Subscribe(fn func(T)) *Subscription {
	sub := s.subj.Subscribe(fn)
	fn(s.Get())
	return sub
}

// Changed returns a Stream that emits on every value change (no initial emit).
func (s *Signal[T]) Changed() *Stream[T] { return &s.subj.Stream }

// AnyStream converts Changed() to Stream[any] for use with Effect/Computed.
func (s *Signal[T]) AnyStream() *Stream[any] {
	out := &Stream[any]{}
	s.Changed().Subscribe(func(v T) { out.emit(any(v)) })
	return out
}

// ── MapSignal ─────────────────────────────────────────────────────────────────

// MapSignal creates a read-only Signal whose value is fn(src.Get()) and is
// recomputed whenever src changes.
func MapSignal[A, B comparable](src *Signal[A], fn func(A) B) *Signal[B] {
	out := NewSignal(fn(src.Get()))
	src.Changed().Subscribe(func(v A) { out.Set(fn(v)) })
	return out
}

// ── SignalAny[T] ─────────────────────────────────────────────────────────────

// SignalAny[T] is like Signal[T] but works for any type (including slices and
// maps). It always notifies on Set, even if the value is pointer-equal.
type SignalAny[T any] struct {
	mu   sync.RWMutex
	val  T
	subj Subject[T]
}

// NewSignalAny creates a SignalAny with an initial value.
func NewSignalAny[T any](initial T) *SignalAny[T] {
	return &SignalAny[T]{val: initial}
}

func (s *SignalAny[T]) Get() T {
	s.mu.RLock()
	v := s.val
	s.mu.RUnlock()
	return v
}

func (s *SignalAny[T]) Set(v T) {
	s.mu.Lock()
	s.val = v
	s.mu.Unlock()
	s.subj.Push(v)
}

func (s *SignalAny[T]) Subscribe(fn func(T)) *Subscription {
	sub := s.subj.Subscribe(fn)
	fn(s.Get())
	return sub
}

func (s *SignalAny[T]) Changed() *Stream[T] { return &s.subj.Stream }

// ═══════════════════════════════════════════════════════════════════════════════
// Effect
// ═══════════════════════════════════════════════════════════════════════════════

// Effect runs fn immediately and then re-runs it whenever any dep stream emits.
// Call Dispose to stop and free resources.
type Effect struct {
	subs []*Subscription
}

// NewEffect creates and immediately runs an Effect.
// deps is a variadic list of Stream[any] that trigger re-execution.
// Use Signal.AnyStream() to convert a Signal's Changed() to Stream[any].
func NewEffect(fn func(), deps ...*Stream[any]) *Effect {
	e := &Effect{}
	fn()
	for _, d := range deps {
		d := d
		e.subs = append(e.subs, d.Subscribe(func(_ any) { fn() }))
	}
	return e
}

// Dispose removes all subscriptions, preventing future runs.
func (e *Effect) Dispose() {
	for _, s := range e.subs {
		s.Unsubscribe()
	}
	e.subs = nil
}

// AnyStream is a helper that wraps a Stream[T] into Stream[any].
// Useful when composing typed streams with Effect or Computed.
func AnyStream[T any](s *Stream[T]) *Stream[any] {
	out := &Stream[any]{}
	s.Subscribe(func(v T) { out.emit(any(v)) })
	return out
}

// ═══════════════════════════════════════════════════════════════════════════════
// CombineLatest2
// ═══════════════════════════════════════════════════════════════════════════════

// CombineLatest2 emits a [2]any pair whenever either signal changes.
func CombineLatest2[A, B comparable](a *Signal[A], b *Signal[B]) *Stream[[2]any] {
	out := &Stream[[2]any]{}
	pair := func() [2]any { return [2]any{a.Get(), b.Get()} }
	a.Changed().Subscribe(func(_ A) { out.emit(pair()) })
	b.Changed().Subscribe(func(_ B) { out.emit(pair()) })
	return out
}

```

## File: animation.go
Language: go | Tokens: 4428 | Size: 17715 bytes

**Imports:** math

```go
// Package ui — animation.go
//
// Declarative animation system.
//
//   Tween          — interpolate a value from A→B over a duration with easing
//   Spring         — physically-simulated spring (mass-spring-damper)
//   Sequence       — run animations one after another
//   Parallel       — run animations simultaneously
//   Timeline       — keyframe-based multi-property animation
//   Animator       — central tick-driven animation scheduler
//
// Usage
//
//	anim := ui.NewAnimator()   // one per window; call Tick(delta) every frame
//
//	t := ui.NewTween(0, 100, 0.4, ui.EaseOutCubic)
//	anim.Play(t)
//	// in draw loop: x := t.Value()
//
//	spring := ui.NewSpring(0, 100, 200, 20)   // target, stiffness, damping
//	anim.Play(spring)
package ui

import "math"

// ═══════════════════════════════════════════════════════════════════════════════
// Easing functions
// ═══════════════════════════════════════════════════════════════════════════════

// EasingFn maps a progress value t ∈ [0,1] to an eased value.
type EasingFn func(t float64) float64

// Standard easing functions.
var (
	Linear      EasingFn = func(t float64) float64 { return t }
	EaseInQuad  EasingFn = func(t float64) float64 { return t * t }
	EaseOutQuad EasingFn = func(t float64) float64 { return t * (2 - t) }
	EaseInOutQuad EasingFn = func(t float64) float64 {
		if t < 0.5 {
			return 2 * t * t
		}
		return -1 + (4-2*t)*t
	}
	EaseInCubic  EasingFn = func(t float64) float64 { return t * t * t }
	EaseOutCubic EasingFn = func(t float64) float64 {
		t--
		return t*t*t + 1
	}
	EaseInOutCubic EasingFn = func(t float64) float64 {
		if t < 0.5 {
			return 4 * t * t * t
		}
		t = 2*t - 2
		return (t*t*t)/2 + 1
	}
	EaseInQuart  EasingFn = func(t float64) float64 { return t * t * t * t }
	EaseOutQuart EasingFn = func(t float64) float64 {
		t--
		return 1 - t*t*t*t
	}
	EaseInOutQuart EasingFn = func(t float64) float64 {
		if t < 0.5 {
			return 8 * t * t * t * t
		}
		t = t*2 - 2
		return 1 - t*t*t*t/2
	}
	EaseInBack EasingFn = func(t float64) float64 {
		const c = 1.70158
		return t * t * ((c+1)*t - c)
	}
	EaseOutBack EasingFn = func(t float64) float64 {
		const c = 1.70158
		t--
		return t*t*((c+1)*t+c) + 1
	}
	EaseInElastic EasingFn = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*2*math.Pi/3)
	}
	EaseOutElastic EasingFn = func(t float64) float64 {
		if t == 0 || t == 1 {
			return t
		}
		return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*2*math.Pi/3) + 1
	}
	EaseOutBounce EasingFn = func(t float64) float64 {
		const n1, d1 = 7.5625, 2.75
		if t < 1/d1 {
			return n1 * t * t
		} else if t < 2/d1 {
			t -= 1.5 / d1
			return n1*t*t + 0.75
		} else if t < 2.5/d1 {
			t -= 2.25 / d1
			return n1*t*t + 0.9375
		}
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
)

// CubicBezier returns a custom cubic-bezier easing (like CSS).
func CubicBezier(x1, y1, x2, y2 float64) EasingFn {
	// Newton-Raphson approximation.
	sampleCurveX := func(t float64) float64 {
		return ((1.5*x2-2.5*x1+1)*t*t*t + (-1.5*x2+2*x1)*t*t + (0.5*x1)*t) * 2
	}
	sampleCurveY := func(t float64) float64 {
		return ((1.5*y2-2.5*y1+1)*t*t*t + (-1.5*y2+2*y1)*t*t + (0.5*y1)*t) * 2
	}
	sampleCurveDerivX := func(t float64) float64 {
		return ((4.5*x2-7.5*x1+3)*t*t+((-3*x2+4*x1)*t)+x1) * 2
	}
	solveCurveX := func(x float64) float64 {
		t := x
		for i := 0; i < 8; i++ {
			dx := sampleCurveDerivX(t)
			if math.Abs(dx) < 1e-6 {
				break
			}
			t -= (sampleCurveX(t) - x) / dx
		}
		return t
	}
	return func(p float64) float64 {
		return sampleCurveY(solveCurveX(p))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Animatable interface
// ═══════════════════════════════════════════════════════════════════════════════

// Animatable is implemented by anything the Animator can update.
type Animatable interface {
	// Tick advances the animation by delta seconds.
	// Returns true when the animation is still running, false when finished.
	TickAnim(delta float64) bool
}

// ═══════════════════════════════════════════════════════════════════════════════
// Tween
// ═══════════════════════════════════════════════════════════════════════════════

// Tween interpolates a float64 from From to To over Duration seconds.
type Tween struct {
	From     float64
	To       float64
	Duration float64  // seconds
	Easing   EasingFn
	Loop     bool
	Yoyo     bool // reverse direction on alternate loops

	elapsed float64
	dir     float64 // 1 = forward, -1 = reverse
	done    bool

	OnUpdate   func(float64)
	OnComplete func()
}

// NewTween creates a new Tween.
func NewTween(from, to, duration float64, easing EasingFn) *Tween {
	if easing == nil {
		easing = Linear
	}
	return &Tween{From: from, To: to, Duration: duration, Easing: easing, dir: 1}
}

// Value returns the current interpolated value.
func (t *Tween) Value() float64 {
	if t.Duration <= 0 {
		return t.To
	}
	progress := t.elapsed / t.Duration
	if progress > 1 {
		progress = 1
	}
	if t.dir < 0 {
		progress = 1 - progress
	}
	eased := t.Easing(progress)
	return t.From + (t.To-t.From)*eased
}

// Progress returns the normalized progress [0,1].
func (t *Tween) Progress() float64 {
	if t.Duration <= 0 {
		return 1
	}
	p := t.elapsed / t.Duration
	if p > 1 {
		return 1
	}
	return p
}

// Done reports whether the tween has finished.
func (t *Tween) Done() bool { return t.done }

// Reset restarts the tween from the beginning.
func (t *Tween) Reset() {
	t.elapsed = 0
	t.done = false
	t.dir = 1
}

// Reverse starts the tween backwards from the current position.
func (t *Tween) Reverse() { t.dir = -t.dir }

// TickAnim advances the tween.
func (t *Tween) TickAnim(delta float64) bool {
	if t.done {
		return false
	}
	t.elapsed += delta
	v := t.Value()
	if t.OnUpdate != nil {
		t.OnUpdate(v)
	}
	if t.elapsed >= t.Duration {
		if t.Loop {
			if t.Yoyo {
				t.dir = -t.dir
			}
			t.elapsed -= t.Duration
		} else {
			t.elapsed = t.Duration
			t.done = true
			if t.OnComplete != nil {
				t.OnComplete()
			}
			return false
		}
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Spring
// ═══════════════════════════════════════════════════════════════════════════════

// Spring simulates a mass-spring-damper physical animation.
//
// It converges toward Target using Stiffness and Damping.
// Typical values: stiffness 100–500, damping 10–30.
type Spring struct {
	Target    float64
	Stiffness float64 // spring constant k
	Damping   float64 // damping coefficient c
	Mass      float64 // default 1

	position float64
	velocity float64
	done     bool

	OnUpdate   func(float64)
	OnComplete func()
}

// NewSpring creates a Spring.
func NewSpring(initialPos, target, stiffness, damping float64) *Spring {
	return &Spring{
		Target: target, Stiffness: stiffness, Damping: damping,
		Mass: 1, position: initialPos,
	}
}

// Position returns the current simulated position.
func (s *Spring) Position() float64 { return s.position }

// Done reports whether the spring has settled.
func (s *Spring) Done() bool { return s.done }

// TickAnim advances the spring simulation.
func (s *Spring) TickAnim(delta float64) bool {
	if s.done {
		return false
	}
	mass := s.Mass
	if mass <= 0 {
		mass = 1
	}
	// Semi-implicit Euler integration.
	force := -s.Stiffness*(s.position-s.Target) - s.Damping*s.velocity
	s.velocity += (force / mass) * delta
	s.position += s.velocity * delta

	if s.OnUpdate != nil {
		s.OnUpdate(s.position)
	}

	// Settle if close enough.
	if math.Abs(s.position-s.Target) < 0.01 && math.Abs(s.velocity) < 0.01 {
		s.position = s.Target
		s.velocity = 0
		s.done = true
		if s.OnComplete != nil {
			s.OnComplete()
		}
		return false
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Sequence
// ═══════════════════════════════════════════════════════════════════════════════

// Sequence runs a list of Animatable values one after another.
type Sequence struct {
	anims   []Animatable
	current int
	done    bool

	OnComplete func()
}

// NewSequence creates a Sequence.
func NewSequence(anims ...Animatable) *Sequence {
	return &Sequence{anims: anims}
}

// Done reports whether all animations have completed.
func (s *Sequence) Done() bool { return s.done }

// TickAnim advances the currently active animation.
func (s *Sequence) TickAnim(delta float64) bool {
	if s.done || len(s.anims) == 0 {
		return false
	}
	if !s.anims[s.current].TickAnim(delta) {
		s.current++
		if s.current >= len(s.anims) {
			s.done = true
			if s.OnComplete != nil {
				s.OnComplete()
			}
			return false
		}
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Parallel
// ═══════════════════════════════════════════════════════════════════════════════

// Parallel runs multiple animations simultaneously and finishes when all are done.
type Parallel struct {
	anims  []Animatable
	active []bool
	done   bool

	OnComplete func()
}

// NewParallel creates a Parallel animation group.
func NewParallel(anims ...Animatable) *Parallel {
	p := &Parallel{anims: anims, active: make([]bool, len(anims))}
	for i := range p.active {
		p.active[i] = true
	}
	return p
}

// Done reports whether all animations are complete.
func (p *Parallel) Done() bool { return p.done }

// TickAnim ticks all active child animations.
func (p *Parallel) TickAnim(delta float64) bool {
	if p.done {
		return false
	}
	anyRunning := false
	for i, anim := range p.anims {
		if p.active[i] {
			if !anim.TickAnim(delta) {
				p.active[i] = false
			} else {
				anyRunning = true
			}
		}
	}
	if !anyRunning {
		p.done = true
		if p.OnComplete != nil {
			p.OnComplete()
		}
		return false
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Keyframe / Timeline
// ═══════════════════════════════════════════════════════════════════════════════

// Keyframe defines a value at a specific time (0.0–1.0 progress).
type Keyframe struct {
	At     float64 // normalized time [0,1]
	Value  float64
	Easing EasingFn // easing from this keyframe to the next (nil = linear)
}

// Timeline interpolates a float64 across multiple Keyframes over Duration seconds.
type Timeline struct {
	Keyframes []Keyframe
	Duration  float64
	Loop      bool

	elapsed float64
	done    bool

	OnUpdate   func(float64)
	OnComplete func()
}

// NewTimeline creates a Timeline.
func NewTimeline(duration float64, keyframes ...Keyframe) *Timeline {
	return &Timeline{Duration: duration, Keyframes: keyframes}
}

// Value returns the current interpolated value based on elapsed time.
func (tl *Timeline) Value() float64 {
	if len(tl.Keyframes) == 0 {
		return 0
	}
	t := tl.elapsed / tl.Duration
	if t > 1 {
		t = 1
	}
	kf := tl.Keyframes
	for i := 0; i < len(kf)-1; i++ {
		if t >= kf[i].At && t <= kf[i+1].At {
			span := kf[i+1].At - kf[i].At
			if span == 0 {
				return kf[i+1].Value
			}
			local := (t - kf[i].At) / span
			ease := kf[i].Easing
			if ease == nil {
				ease = Linear
			}
			return kf[i].Value + (kf[i+1].Value-kf[i].Value)*ease(local)
		}
	}
	return kf[len(kf)-1].Value
}

// Done reports whether the timeline has finished.
func (tl *Timeline) Done() bool { return tl.done }

// TickAnim advances the timeline.
func (tl *Timeline) TickAnim(delta float64) bool {
	if tl.done {
		return false
	}
	tl.elapsed += delta
	v := tl.Value()
	if tl.OnUpdate != nil {
		tl.OnUpdate(v)
	}
	if tl.elapsed >= tl.Duration {
		if tl.Loop {
			tl.elapsed -= tl.Duration
		} else {
			tl.elapsed = tl.Duration
			tl.done = true
			if tl.OnComplete != nil {
				tl.OnComplete()
			}
			return false
		}
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Delay
// ═══════════════════════════════════════════════════════════════════════════════

// Delay is an Animatable that waits for duration seconds then fires OnComplete.
type Delay struct {
	Duration float64
	elapsed  float64
	done     bool

	OnComplete func()
}

// NewDelay creates a Delay.
func NewDelay(duration float64) *Delay { return &Delay{Duration: duration} }

// Done reports whether the delay has elapsed.
func (d *Delay) Done() bool { return d.done }

// TickAnim advances the delay.
func (d *Delay) TickAnim(delta float64) bool {
	if d.done {
		return false
	}
	d.elapsed += delta
	if d.elapsed >= d.Duration {
		d.done = true
		if d.OnComplete != nil {
			d.OnComplete()
		}
		return false
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Animator  —  central scheduler
// ═══════════════════════════════════════════════════════════════════════════════

type animEntry struct {
	id   uint64
	anim Animatable
}

// Animator manages a set of active animations and advances them each frame.
// Create one per window; call Tick(delta) from the window's main loop.
type Animator struct {
	entries []animEntry
	seq     uint64
}

// NewAnimator creates an Animator.
func NewAnimator() *Animator { return &Animator{} }

// Play adds an Animatable to the scheduler.
// Returns an ID that can be used with Cancel.
func (a *Animator) Play(anim Animatable) uint64 {
	a.seq++
	a.entries = append(a.entries, animEntry{id: a.seq, anim: anim})
	return a.seq
}

// Cancel removes the animation with the given id.
func (a *Animator) Cancel(id uint64) {
	for i, e := range a.entries {
		if e.id == id {
			a.entries = append(a.entries[:i], a.entries[i+1:]...)
			return
		}
	}
}

// CancelAll removes all active animations.
func (a *Animator) CancelAll() { a.entries = a.entries[:0] }

// Tick advances all animations by delta seconds and removes finished ones.
func (a *Animator) Tick(delta float64) {
	live := a.entries[:0]
	for _, e := range a.entries {
		if e.anim.TickAnim(delta) {
			live = append(live, e)
		}
	}
	a.entries = live
}

// Len returns the number of active animations.
func (a *Animator) Len() int { return len(a.entries) }

// ─── Convenience constructors ─────────────────────────────────────────────────

// FadeIn returns a Tween that goes from 0 to 1 using EaseOutQuad.
func FadeIn(duration float64) *Tween { return NewTween(0, 1, duration, EaseOutQuad) }

// FadeOut returns a Tween that goes from 1 to 0 using EaseInQuad.
func FadeOut(duration float64) *Tween { return NewTween(1, 0, duration, EaseInQuad) }

// SlideIn returns a Tween starting at offset and ending at 0.
func SlideIn(offset, duration float64) *Tween {
	return NewTween(offset, 0, duration, EaseOutCubic)
}

// ScaleBounce returns a Tween that overshoots 1.1 then settles at 1.
func ScaleBounce(duration float64) *Tween {
	return NewTween(0, 1, duration, EaseOutBack)
}

```

## File: accessibility.go
Language: go | Tokens: 3329 | Size: 13319 bytes

```go
// Package ui — accessibility.go
//
// Accessibility (a11y) support.
//
//   Role         — semantic widget role (button, checkbox, dialog, …)
//   AriaProps    — live region, labels, states (disabled/checked/expanded/…)
//   A11yNode     — virtual accessibility tree node
//   A11yTree     — collects nodes from the component hierarchy
//   ContrastCheck — WCAG 2.1 relative-luminance contrast-ratio helpers
//
// Platform screen-reader integration is left to the backend (window layer).
// This file provides the data model and helpers the backend can query.
package ui

import (
	"fmt"
	"math"

	"github.com/achiket/gui-go/canvas"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Role
// ═══════════════════════════════════════════════════════════════════════════════

// Role describes the semantic type of a UI element (mirrors ARIA roles).
type Role string

const (
	RoleNone        Role = "none"
	RoleButton      Role = "button"
	RoleCheckbox    Role = "checkbox"
	RoleRadio       Role = "radio"
	RoleSlider      Role = "slider"
	RoleTextInput   Role = "textinput"
	RoleLink        Role = "link"
	RoleImage       Role = "img"
	RoleHeading     Role = "heading"
	RoleLabel       Role = "label"
	RoleList        Role = "list"
	RoleListItem    Role = "listitem"
	RoleTab         Role = "tab"
	RoleTabPanel    Role = "tabpanel"
	RoleTabList     Role = "tablist"
	RoleDialog      Role = "dialog"
	RoleAlert       Role = "alert"
	RoleMenu        Role = "menu"
	RoleMenuItem    Role = "menuitem"
	RoleProgressBar Role = "progressbar"
	RoleScrollBar   Role = "scrollbar"
	RoleComboBox    Role = "combobox"
	RoleGrid        Role = "grid"
	RoleGridCell    Role = "gridcell"
	RoleRegion      Role = "region"
	RoleToolbar     Role = "toolbar"
	RoleTooltip     Role = "tooltip"
	RoleStatus      Role = "status"
	RoleMain        Role = "main"
	RoleNavigation  Role = "navigation"
	RoleBanner      Role = "banner"
	RoleContentInfo Role = "contentinfo"
)

// ═══════════════════════════════════════════════════════════════════════════════
// AriaProps
// ═══════════════════════════════════════════════════════════════════════════════

// AriaProps holds ARIA-equivalent properties for a node.
type AriaProps struct {
	Label       string // accessible name (aria-label)
	Description string // aria-description / aria-describedby text
	Role        Role

	// Boolean states.
	Disabled bool
	Checked  *bool // nil = not applicable, true/false = tri-state
	Pressed  *bool
	Expanded *bool
	Selected bool
	Required bool
	ReadOnly bool
	Hidden   bool // true = excluded from accessibility tree

	// Value information (for sliders, progress bars, etc.)
	ValueNow float64
	ValueMin float64
	ValueMax float64
	ValueText string // human-readable value override

	// Heading level 1–6.
	HeadingLevel int

	// Live region.
	Live      LiveRegion
	Atomic    bool
	Relevant  string // "additions removals text all"

	// Relationship IDs (parallel to ARIA relationship attributes).
	Controls  string // id of controlled element
	LabeledBy string // id of labeling element
	DescribedBy string
	OwnsIDs   []string
}

// LiveRegion indicates how assertively a live region announces updates.
type LiveRegion string

const (
	LiveOff       LiveRegion = "off"
	LivePolite    LiveRegion = "polite"
	LiveAssertive LiveRegion = "assertive"
)

// DescribeState returns a short human-readable state description.
func (a *AriaProps) DescribeState() string {
	states := []string{}
	if a.Disabled {
		states = append(states, "disabled")
	}
	if a.ReadOnly {
		states = append(states, "read-only")
	}
	if a.Required {
		states = append(states, "required")
	}
	if a.Checked != nil {
		if *a.Checked {
			states = append(states, "checked")
		} else {
			states = append(states, "unchecked")
		}
	}
	if a.Expanded != nil {
		if *a.Expanded {
			states = append(states, "expanded")
		} else {
			states = append(states, "collapsed")
		}
	}
	if a.Selected {
		states = append(states, "selected")
	}
	result := ""
	for i, s := range states {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════════════
// Accessible interface
// ═══════════════════════════════════════════════════════════════════════════════

// Accessible is implemented by components that expose accessibility metadata.
type Accessible interface {
	Component
	// A11yProps returns this component's accessibility properties.
	A11yProps() AriaProps
}

// ═══════════════════════════════════════════════════════════════════════════════
// A11yNode  —  virtual accessibility tree node
// ═══════════════════════════════════════════════════════════════════════════════

// A11yNode represents one node in the virtual accessibility tree.
type A11yNode struct {
	ID       string
	Props    AriaProps
	Bounds   canvas.Rect
	Children []*A11yNode
	Source   Component // the originating component
}

// String returns a compact text representation useful for snapshot testing.
func (n *A11yNode) String() string {
	s := fmt.Sprintf("[%s] %s", n.Props.Role, n.Props.Label)
	if state := n.Props.DescribeState(); state != "" {
		s += " (" + state + ")"
	}
	return s
}

// Walk calls fn on this node and all descendants in depth-first order.
func (n *A11yNode) Walk(fn func(*A11yNode)) {
	fn(n)
	for _, child := range n.Children {
		child.Walk(fn)
	}
}

// Find returns the first node matching pred, or nil.
func (n *A11yNode) Find(pred func(*A11yNode) bool) *A11yNode {
	if pred(n) {
		return n
	}
	for _, child := range n.Children {
		if found := child.Find(pred); found != nil {
			return found
		}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// A11yTree  —  builder
// ═══════════════════════════════════════════════════════════════════════════════

// A11yTree builds a virtual accessibility tree from a component hierarchy.
type A11yTree struct {
	Root  *A11yNode
	nodes map[string]*A11yNode
	seq   int
}

// NewA11yTree creates an empty A11yTree.
func NewA11yTree() *A11yTree {
	return &A11yTree{nodes: make(map[string]*A11yNode)}
}

// Add registers a component as an accessibility node under parent.
// Pass nil parent to add at the root level.
func (t *A11yTree) Add(comp Accessible, parent *A11yNode) *A11yNode {
	t.seq++
	id := fmt.Sprintf("a11y-%d", t.seq)
	node := &A11yNode{
		ID:     id,
		Props:  comp.A11yProps(),
		Bounds: comp.Bounds(),
		Source: comp,
	}
	if node.Props.Hidden {
		return nil
	}
	t.nodes[id] = node
	if parent == nil {
		if t.Root == nil {
			t.Root = node
		} else {
			t.Root.Children = append(t.Root.Children, node)
		}
	} else {
		parent.Children = append(parent.Children, node)
	}
	return node
}

// FindByLabel returns the first node whose label contains substr (case-sensitive).
func (t *A11yTree) FindByLabel(substr string) *A11yNode {
	if t.Root == nil {
		return nil
	}
	return t.Root.Find(func(n *A11yNode) bool {
		return containsStr(n.Props.Label, substr)
	})
}

// FindByRole returns all nodes with the given role.
func (t *A11yTree) FindByRole(role Role) []*A11yNode {
	var out []*A11yNode
	if t.Root != nil {
		t.Root.Walk(func(n *A11yNode) {
			if n.Props.Role == role {
				out = append(out, n)
			}
		})
	}
	return out
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstr(s, substr) >= 0)
}

func findSubstr(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ═══════════════════════════════════════════════════════════════════════════════
// WCAG Contrast helpers
// ═══════════════════════════════════════════════════════════════════════════════

// RelativeLuminance computes the WCAG 2.1 relative luminance of a color.
// Values range from 0 (black) to 1 (white).
func RelativeLuminance(c canvas.Color) float64 {
	linearize := func(v float32) float64 {
		f := float64(v)
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	r := linearize(c.R)
	g := linearize(c.G)
	b := linearize(c.B)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// ContrastRatio returns the WCAG contrast ratio between two colors [1, 21].
func ContrastRatio(c1, c2 canvas.Color) float64 {
	l1 := RelativeLuminance(c1)
	l2 := RelativeLuminance(c2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// WCAGLevel indicates the WCAG compliance level for a given contrast ratio.
type WCAGLevel string

const (
	WCAGFail  WCAGLevel = "fail"
	WCAGAA    WCAGLevel = "AA"
	WCAGAALarge WCAGLevel = "AA-large"
	WCAGAAA   WCAGLevel = "AAA"
)

// ContrastLevel returns the WCAG compliance level for the given contrast ratio
// and whether the text is "large" (≥18pt normal or ≥14pt bold).
func ContrastLevel(ratio float64, largeText bool) WCAGLevel {
	if largeText {
		switch {
		case ratio >= 7:
			return WCAGAAA
		case ratio >= 4.5:
			return WCAGAA
		case ratio >= 3:
			return WCAGAALarge
		}
		return WCAGFail
	}
	switch {
	case ratio >= 7:
		return WCAGAAA
	case ratio >= 4.5:
		return WCAGAA
	}
	return WCAGFail
}

// EnsureContrast darkens or lightens fg until it meets the target ratio against bg.
// Returns the adjusted foreground color.
func EnsureContrast(fg, bg canvas.Color, targetRatio float64) canvas.Color {
	if ContrastRatio(fg, bg) >= targetRatio {
		return fg
	}
	bgL := RelativeLuminance(bg)
	// Decide whether to lighten or darken fg.
	toWhite := ContrastRatio(canvas.White, bg)
	toBlack := ContrastRatio(canvas.Black, bg)
	if toWhite > toBlack {
		// Lighten toward white.
		for i := 0; i < 100; i++ {
			fg = canvas.Lerp(fg, canvas.White, 0.05)
			if ContrastRatio(fg, bg) >= targetRatio {
				return fg
			}
		}
	} else {
		_ = bgL
		// Darken toward black.
		for i := 0; i < 100; i++ {
			fg = canvas.Lerp(fg, canvas.Black, 0.05)
			if ContrastRatio(fg, bg) >= targetRatio {
				return fg
			}
		}
	}
	return fg
}

// ═══════════════════════════════════════════════════════════════════════════════
// Announcement  —  screen-reader live region helper
// ═══════════════════════════════════════════════════════════════════════════════

// Announcement represents a message to be announced by a screen reader.
type Announcement struct {
	Message  string
	Priority LiveRegion // Polite or Assertive
}

// AnnouncementQueue is a simple FIFO for accessibility announcements.
// The backend window layer should drain this each frame and relay to the OS.
type AnnouncementQueue struct {
	items []Announcement
}

// Push adds a new announcement.
func (q *AnnouncementQueue) Push(msg string, priority LiveRegion) {
	q.items = append(q.items, Announcement{msg, priority})
}

// Pop removes and returns the oldest announcement, and a bool indicating presence.
func (q *AnnouncementQueue) Pop() (Announcement, bool) {
	if len(q.items) == 0 {
		return Announcement{}, false
	}
	a := q.items[0]
	q.items = q.items[1:]
	return a, true
}

// Len returns the number of pending announcements.
func (q *AnnouncementQueue) Len() int { return len(q.items) }

```

## File: context_menu.go
Language: go | Tokens: 3297 | Size: 13190 bytes

```go
// Package ui — context_menu.go
//
// Context menus, pop-up menus, and a chainable MenuBuilder.
//
//   MenuItem      — a single entry (action, separator, or submenu)
//   Menu          — a list of MenuItems that renders as a floating popup
//   MenuBuilder   — fluent builder for constructing menus
//   ContextMenu   — attaches a Menu to any component's right-click event
//   MenuManager   — global overlay manager; keeps at most one menu open
//
// Usage
//
//	menu := ui.NewMenuBuilder().
//	    Item("Cut",  "Ctrl+X", onCut).
//	    Item("Copy", "Ctrl+C", onCopy).
//	    Sep().
//	    Sub("Paste Special", ui.NewMenuBuilder().
//	        Item("Plain Text", "", onPastePlain).
//	        Item("Formatted",  "", onPasteFormatted).
//	        Build()).
//	    Build()
//
//	mgr := ui.NewMenuManager()
//	cm := ui.NewContextMenu(myComponent, menu)
//	mgr.Register(cm)
//
//	// In window HandleEvent: mgr.HandleEvent(e)
//	// In window Draw (last): mgr.Draw(canvas)
package ui

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
)

// ═══════════════════════════════════════════════════════════════════════════════
// MenuItem
// ═══════════════════════════════════════════════════════════════════════════════

// MenuItemKind distinguishes menu entry types.
type MenuItemKind int

const (
	MenuItemAction    MenuItemKind = iota // a clickable action
	MenuItemSeparator                     // a visual divider
	MenuItemSubmenu                       // opens a child menu
)

// MenuItem is one entry in a Menu.
type MenuItem struct {
	Kind     MenuItemKind
	Label    string
	Shortcut string // e.g. "Ctrl+C"
	Disabled bool
	Checked  bool
	Action   func()
	Sub      *Menu
	Icon     canvas.TextureID // optional icon (0 = none)
}

// ═══════════════════════════════════════════════════════════════════════════════
// MenuStyle
// ═══════════════════════════════════════════════════════════════════════════════

// MenuStyle configures menu appearance.
type MenuStyle struct {
	Background  canvas.Color
	HoverBg     canvas.Color
	Border      canvas.Color
	TextStyle   canvas.TextStyle
	ShortcutStyle canvas.TextStyle
	DisabledText canvas.Color
	SepColor    canvas.Color
	ItemHeight  float32
	SepHeight   float32
	Padding     float32 // horizontal padding
	Radius      float32
	MinWidth    float32
	Shadow      bool
}

// DefaultMenuStyle returns a theme-aware MenuStyle.
func DefaultMenuStyle() MenuStyle {
	th := theme.Current()
	return MenuStyle{
		Background:   th.Colors.BgSurface,
		HoverBg:      th.Colors.Border,
		Border:       th.Colors.Border,
		TextStyle:    th.Type.Body,
		ShortcutStyle: canvas.TextStyle{Color: th.Colors.TextSecondary, Size: 11},
		DisabledText: th.Colors.TextSecondary,
		SepColor:     th.Colors.Border,
		ItemHeight:   32,
		SepHeight:    9,
		Padding:      12,
		Radius:       th.Radius.MD,
		MinWidth:     160,
		Shadow:       true,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Menu
// ═══════════════════════════════════════════════════════════════════════════════

// Menu is a floating popup containing MenuItems.
type Menu struct {
	Items  []MenuItem
	Style  MenuStyle

	x, y    float32
	w, h    float32
	hover   int
	open    bool
	subMenu *menuPopup // active child submenu
}

// NewMenu creates a Menu with the given items and style.
func NewMenu(items []MenuItem, style MenuStyle) *Menu {
	return &Menu{Items: items, Style: style, hover: -1}
}

// menuPopup is an internal open-menu instance tracking position.
type menuPopup struct {
	menu      *Menu
	x, y      float32
	hoverItem int
	child     *menuPopup
}

// ── width calculation ─────────────────────────────────────────────────────────

func (m *Menu) calcWidth(c *canvas.Canvas) float32 {
	s := m.Style
	maxW := s.MinWidth
	for _, item := range m.Items {
		if item.Kind == MenuItemSeparator {
			continue
		}
		tw := c.MeasureText(item.Label, s.TextStyle).W
		if item.Shortcut != "" {
			tw += c.MeasureText(item.Shortcut, s.ShortcutStyle).W + 24
		}
		if item.Kind == MenuItemSubmenu {
			tw += 16 // chevron
		}
		tw += s.Padding*2
		if tw > maxW {
			maxW = tw
		}
	}
	return maxW
}

func (m *Menu) calcHeight() float32 {
	s := m.Style
	h := float32(0)
	for _, item := range m.Items {
		if item.Kind == MenuItemSeparator {
			h += s.SepHeight
		} else {
			h += s.ItemHeight
		}
	}
	return h
}

// ── draw ─────────────────────────────────────────────────────────────────────

func (m *Menu) draw(c *canvas.Canvas, x, y float32) {
	s := m.Style
	w := m.calcWidth(c)
	h := m.calcHeight()
	m.x, m.y, m.w, m.h = x, y, w, h

	// Shadow.
	if s.Shadow {
		for i := float32(1); i <= 6; i++ {
			alpha := 0.04 - float32(i)*0.004
			if alpha < 0 {
				alpha = 0
			}
			c.DrawRoundedRect(x+i, y+i, w, h, s.Radius, canvas.FillPaint(canvas.Color{A: alpha}))
		}
	}
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.FillPaint(s.Background))
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.StrokePaint(s.Border, 1))

	iy := y
	for i, item := range m.Items {
		if item.Kind == MenuItemSeparator {
			mid := iy + s.SepHeight/2
			c.DrawRect(x+s.Padding, mid, w-s.Padding*2, 1, canvas.FillPaint(s.SepColor))
			iy += s.SepHeight
			continue
		}
		if i == m.hover && !item.Disabled {
			c.DrawRoundedRect(x+2, iy+2, w-4, s.ItemHeight-4, s.Radius-1, canvas.FillPaint(s.HoverBg))
		}
		ts := s.TextStyle
		if item.Disabled {
			ts.Color = s.DisabledText
		}
		if item.Checked {
			c.DrawText(x+4, iy+(s.ItemHeight+ts.Size)/2, "✓", ts)
		}
		c.DrawText(x+s.Padding, iy+(s.ItemHeight+ts.Size)/2, item.Label, ts)
		if item.Shortcut != "" {
			sw := c.MeasureText(item.Shortcut, s.ShortcutStyle).W
			c.DrawText(x+w-sw-s.Padding, iy+(s.ItemHeight+s.ShortcutStyle.Size)/2, item.Shortcut, s.ShortcutStyle)
		}
		if item.Kind == MenuItemSubmenu {
			chevX := x + w - 14
			chevY := iy + s.ItemHeight/2
			p := canvas.StrokePaint(s.TextStyle.Color, 1.5)
			c.DrawLine(chevX, chevY-4, chevX+5, chevY, p)
			c.DrawLine(chevX+5, chevY, chevX, chevY+4, p)
		}
		iy += s.ItemHeight
	}
}

// itemAt returns the index of the item under pixel y relative to menu origin, or -1.
func (m *Menu) itemAt(localY float32) int {
	s := m.Style
	iy := float32(0)
	for i, item := range m.Items {
		if item.Kind == MenuItemSeparator {
			iy += s.SepHeight
			continue
		}
		if localY >= iy && localY < iy+s.ItemHeight {
			return i
		}
		iy += s.ItemHeight
	}
	return -1
}

// ═══════════════════════════════════════════════════════════════════════════════
// MenuBuilder  —  fluent API
// ═══════════════════════════════════════════════════════════════════════════════

// MenuBuilder builds a Menu using method chaining.
type MenuBuilder struct {
	items []MenuItem
	style MenuStyle
}

// NewMenuBuilder creates a MenuBuilder with the default style.
func NewMenuBuilder() *MenuBuilder {
	return &MenuBuilder{style: DefaultMenuStyle()}
}

// WithStyle sets a custom style.
func (b *MenuBuilder) WithStyle(s MenuStyle) *MenuBuilder { b.style = s; return b }

// Item adds an action item.
func (b *MenuBuilder) Item(label, shortcut string, action func()) *MenuBuilder {
	b.items = append(b.items, MenuItem{Kind: MenuItemAction, Label: label, Shortcut: shortcut, Action: action})
	return b
}

// CheckItem adds a checkable item.
func (b *MenuBuilder) CheckItem(label string, checked bool, action func()) *MenuBuilder {
	b.items = append(b.items, MenuItem{Kind: MenuItemAction, Label: label, Checked: checked, Action: action})
	return b
}

// Disabled adds a disabled item.
func (b *MenuBuilder) Disabled(label string) *MenuBuilder {
	b.items = append(b.items, MenuItem{Kind: MenuItemAction, Label: label, Disabled: true})
	return b
}

// Sep adds a visual separator.
func (b *MenuBuilder) Sep() *MenuBuilder {
	b.items = append(b.items, MenuItem{Kind: MenuItemSeparator})
	return b
}

// Sub adds a submenu.
func (b *MenuBuilder) Sub(label string, sub *Menu) *MenuBuilder {
	b.items = append(b.items, MenuItem{Kind: MenuItemSubmenu, Label: label, Sub: sub})
	return b
}

// Build returns the constructed Menu.
func (b *MenuBuilder) Build() *Menu {
	return NewMenu(b.items, b.style)
}

// ═══════════════════════════════════════════════════════════════════════════════
// ContextMenu
// ═══════════════════════════════════════════════════════════════════════════════

// ContextMenu binds a right-click trigger to a component.
type ContextMenu struct {
	Component Component
	Menu      *Menu
}

// NewContextMenu creates a ContextMenu for a component.
func NewContextMenu(comp Component, menu *Menu) *ContextMenu {
	return &ContextMenu{Component: comp, Menu: menu}
}

// ═══════════════════════════════════════════════════════════════════════════════
// MenuManager  —  global overlay manager
// ═══════════════════════════════════════════════════════════════════════════════

// MenuManager keeps at most one menu open and manages the event routing.
type MenuManager struct {
	contextMenus []*ContextMenu
	active       *Menu
	activeX      float32
	activeY      float32
}

// NewMenuManager creates a MenuManager.
func NewMenuManager() *MenuManager { return &MenuManager{} }

// Register adds a ContextMenu to the manager.
func (mm *MenuManager) Register(cm *ContextMenu) {
	mm.contextMenus = append(mm.contextMenus, cm)
}

// Open opens menu at position (x, y).
func (mm *MenuManager) Open(menu *Menu, x, y float32) {
	mm.Close()
	mm.active = menu
	mm.activeX = x
	mm.activeY = y
	menu.hover = -1
	menu.open = true
}

// Close dismisses the active menu.
func (mm *MenuManager) Close() {
	if mm.active != nil {
		mm.active.open = false
		mm.active = nil
	}
}

// HandleEvent routes events through the menu system.
// Returns true if the event was consumed.
func (mm *MenuManager) HandleEvent(e Event) bool {
	if e.Type == EventMouseDown && e.Button == 3 {
		// Right-click: check context menus.
		for _, cm := range mm.contextMenus {
			b := cm.Component.Bounds()
			if e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H {
				mm.Open(cm.Menu, e.X, e.Y)
				return true
			}
		}
	}
	if mm.active == nil {
		return false
	}
	m := mm.active
	localX := e.X - mm.activeX
	localY := e.Y - mm.activeY
	switch e.Type {
	case EventMouseMove:
		if localX >= 0 && localX <= m.w && localY >= 0 && localY <= m.h {
			m.hover = m.itemAt(localY)
			return true
		}
		m.hover = -1
	case EventMouseDown:
		if e.Button != 1 {
			mm.Close()
			return false
		}
		if localX >= 0 && localX <= m.w && localY >= 0 && localY <= m.h {
			idx := m.itemAt(localY)
			if idx >= 0 {
				item := m.Items[idx]
				if !item.Disabled && item.Action != nil {
					item.Action()
					mm.Close()
				}
			}
			return true
		}
		mm.Close()
	case EventKeyDown:
		switch e.Key {
		case "Escape":
			mm.Close()
			return true
		}
	}
	return false
}

// Draw renders the active menu on top of everything else.
// Call this as the very last thing in your window's Draw method.
func (mm *MenuManager) Draw(c *canvas.Canvas) {
	if mm.active == nil || !mm.active.open {
		return
	}
	mm.active.draw(c, mm.activeX, mm.activeY)
}

```

## File: i18n.go
Language: go | Tokens: 3202 | Size: 12809 bytes

```go
// Package ui — i18n.go
//
// Internationalization (i18n) and localization (l10n) support.
//
//   Locale       — language/region identifier (e.g. "en-US", "fr-FR")
//   Catalog      — map of translation keys → translated strings for one locale
//   I18n         — registry of catalogs + active locale with reactive updates
//   Pluralizer   — cardinal plural selection (zero/one/few/many/other)
//   Formatter    — number, currency, and date formatting by locale
//
// Usage
//
//	i18n := ui.NewI18n("en-US")
//	i18n.Load("en-US", map[string]string{
//	    "greeting":       "Hello, {name}!",
//	    "items.zero":     "No items",
//	    "items.one":      "One item",
//	    "items.other":    "{count} items",
//	})
//	i18n.Load("fr-FR", map[string]string{
//	    "greeting": "Bonjour, {name}!",
//	})
//
//	t := i18n.T("greeting", "name", "Alice")  // "Hello, Alice!"
//	i18n.SetLocale("fr-FR")
//	t = i18n.T("greeting", "name", "Alice")   // "Bonjour, Alice!"
package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Locale
// ═══════════════════════════════════════════════════════════════════════════════

// Locale is a BCP-47 language tag string (e.g. "en", "en-US", "zh-Hant-TW").
type Locale = string

// ParseLocale splits a locale tag into language and region components.
// "en-US" → ("en", "US").  "fr" → ("fr", "").
func ParseLocale(locale Locale) (lang, region string) {
	parts := strings.SplitN(locale, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return locale, ""
}

// ═══════════════════════════════════════════════════════════════════════════════
// Catalog
// ═══════════════════════════════════════════════════════════════════════════════

// Catalog maps translation keys to template strings.
// Template variables are wrapped in {braces}.
type Catalog map[string]string

// Get looks up key in the catalog, returning the key itself if not found.
func (c Catalog) Get(key string) (string, bool) {
	v, ok := c[key]
	return v, ok
}

// Interpolate replaces {key} placeholders using the supplied key-value pairs.
// Extra pairs are ignored; missing placeholders are left as-is.
func Interpolate(tmpl string, kvs ...string) string {
	if len(kvs)%2 != 0 {
		return tmpl
	}
	for i := 0; i < len(kvs); i += 2 {
		tmpl = strings.ReplaceAll(tmpl, "{"+kvs[i]+"}", kvs[i+1])
	}
	return tmpl
}

// ═══════════════════════════════════════════════════════════════════════════════
// Plural categories
// ═══════════════════════════════════════════════════════════════════════════════

// PluralCategory represents a CLDR plural form.
type PluralCategory string

const (
	PluralZero  PluralCategory = "zero"
	PluralOne   PluralCategory = "one"
	PluralTwo   PluralCategory = "two"
	PluralFew   PluralCategory = "few"
	PluralMany  PluralCategory = "many"
	PluralOther PluralCategory = "other"
)

// PluralRule maps a count to a PluralCategory.
type PluralRule func(n int) PluralCategory

// EnglishPlural is the plural rule for English (and many other languages).
func EnglishPlural(n int) PluralCategory {
	if n == 0 {
		return PluralZero
	}
	if n == 1 {
		return PluralOne
	}
	return PluralOther
}

// FrenchPlural treats 0 and 1 as singular.
func FrenchPlural(n int) PluralCategory {
	if n == 0 || n == 1 {
		return PluralOne
	}
	return PluralOther
}

// RussianPlural implements the Russian plural rules.
func RussianPlural(n int) PluralCategory {
	mod10 := n % 10
	mod100 := n % 100
	if mod10 == 1 && mod100 != 11 {
		return PluralOne
	}
	if mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20) {
		return PluralFew
	}
	return PluralMany
}

// arabicPlural implements Arabic plural rules.
func ArabicPlural(n int) PluralCategory {
	if n == 0 {
		return PluralZero
	}
	if n == 1 {
		return PluralOne
	}
	if n == 2 {
		return PluralTwo
	}
	mod100 := n % 100
	if mod100 >= 3 && mod100 <= 10 {
		return PluralFew
	}
	if mod100 >= 11 {
		return PluralMany
	}
	return PluralOther
}

// BuiltinPluralRules maps known language codes to their plural rule function.
var BuiltinPluralRules = map[string]PluralRule{
	"en": EnglishPlural,
	"de": EnglishPlural,
	"es": EnglishPlural,
	"it": EnglishPlural,
	"pt": EnglishPlural,
	"fr": FrenchPlural,
	"ru": RussianPlural,
	"uk": RussianPlural,
	"pl": RussianPlural,
	"ar": ArabicPlural,
	"zh": func(_ int) PluralCategory { return PluralOther },
	"ja": func(_ int) PluralCategory { return PluralOther },
	"ko": func(_ int) PluralCategory { return PluralOther },
}

// ═══════════════════════════════════════════════════════════════════════════════
// I18n
// ═══════════════════════════════════════════════════════════════════════════════

// I18n manages catalogs for multiple locales and exposes a reactive active-locale
// signal so UI components can rerender on locale change.
type I18n struct {
	mu       sync.RWMutex
	catalogs map[Locale]Catalog
	locale   *Signal[Locale]
	fallback Locale

	pluralRules map[string]PluralRule // indexed by language code
}

// NewI18n creates an I18n instance with the given default locale.
func NewI18n(defaultLocale Locale) *I18n {
	return &I18n{
		catalogs:    make(map[Locale]Catalog),
		locale:      NewSignal(defaultLocale),
		fallback:    "en",
		pluralRules: make(map[string]PluralRule),
	}
}

// Load registers a Catalog for the given locale.
// Calling Load twice for the same locale merges keys.
func (i *I18n) Load(locale Locale, catalog Catalog) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if existing, ok := i.catalogs[locale]; ok {
		for k, v := range catalog {
			existing[k] = v
		}
	} else {
		cp := make(Catalog, len(catalog))
		for k, v := range catalog {
			cp[k] = v
		}
		i.catalogs[locale] = cp
	}
}

// SetLocale changes the active locale, notifying all reactive subscribers.
func (i *I18n) SetLocale(locale Locale) { i.locale.Set(locale) }

// Locale returns the currently active locale.
func (i *I18n) Locale() Locale { return i.locale.Get() }

// LocaleSignal returns the reactive locale Signal for use with Effect/Computed.
func (i *I18n) LocaleSignal() *Signal[Locale] { return i.locale }

// SetFallback sets the fallback locale used when a key is missing.
func (i *I18n) SetFallback(locale Locale) {
	i.mu.Lock()
	i.fallback = locale
	i.mu.Unlock()
}

// RegisterPluralRule registers a custom plural rule for a language code.
func (i *I18n) RegisterPluralRule(lang string, rule PluralRule) {
	i.mu.Lock()
	i.pluralRules[lang] = rule
	i.mu.Unlock()
}

// lookup finds the template for key in the given locale, falling back as needed.
func (i *I18n) lookup(locale Locale, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if cat, ok := i.catalogs[locale]; ok {
		if v, ok := cat[key]; ok {
			return v
		}
	}
	// Try language-only ("en" from "en-US").
	lang, _ := ParseLocale(locale)
	if lang != locale {
		if cat, ok := i.catalogs[lang]; ok {
			if v, ok := cat[key]; ok {
				return v
			}
		}
	}
	// Fallback locale.
	if locale != i.fallback {
		if cat, ok := i.catalogs[i.fallback]; ok {
			if v, ok := cat[key]; ok {
				return v
			}
		}
	}
	return key // return the key itself as last resort
}

// T translates key with optional interpolation key-value pairs.
//
//	i18n.T("greeting", "name", "Bob")
func (i *I18n) T(key string, kvs ...string) string {
	tmpl := i.lookup(i.locale.Get(), key)
	return Interpolate(tmpl, kvs...)
}

// TL translates key in the specified locale (without affecting the active locale).
func (i *I18n) TL(locale Locale, key string, kvs ...string) string {
	tmpl := i.lookup(locale, key)
	return Interpolate(tmpl, kvs...)
}

// TN translates a plural key (e.g. "items") by selecting the appropriate
// plural form ("items.one", "items.few", etc.) based on count.
//
//	i18n.TN("items", 3, "count", "3")
func (i *I18n) TN(key string, count int, kvs ...string) string {
	locale := i.locale.Get()
	lang, _ := ParseLocale(locale)

	// Find plural rule.
	rule, ok := i.pluralRules[lang]
	if !ok {
		rule, ok = BuiltinPluralRules[lang]
	}
	if !ok {
		rule = EnglishPlural
	}

	cat := string(rule(count))
	pluralKey := key + "." + cat
	tmpl := i.lookup(locale, pluralKey)
	if tmpl == pluralKey {
		// Try "other" as a further fallback.
		tmpl = i.lookup(locale, key+".other")
	}
	return Interpolate(tmpl, kvs...)
}

// Has returns true if the key exists in the current locale (or fallback).
func (i *I18n) Has(key string) bool {
	result := i.lookup(i.locale.Get(), key)
	return result != key
}

// ═══════════════════════════════════════════════════════════════════════════════
// Formatter  —  locale-aware number / date formatting
// ═══════════════════════════════════════════════════════════════════════════════

// NumberFormat holds locale-specific number formatting options.
type NumberFormat struct {
	DecimalSep   string // "." or ","
	ThousandsSep string // "," or "." or " " or ""
	Precision    int    // decimal places (-1 = auto)
}

// DefaultNumberFormat returns a NumberFormat for the given locale.
func DefaultNumberFormat(locale Locale) NumberFormat {
	lang, region := ParseLocale(locale)
	_ = region
	switch lang {
	case "de", "fr", "es", "it", "pt":
		return NumberFormat{DecimalSep: ",", ThousandsSep: ".", Precision: -1}
	case "ar":
		return NumberFormat{DecimalSep: "٫", ThousandsSep: "٬", Precision: -1}
	default:
		return NumberFormat{DecimalSep: ".", ThousandsSep: ",", Precision: -1}
	}
}

// FormatNumber formats n according to nf.
func FormatNumber(n float64, nf NumberFormat) string {
	prec := nf.Precision
	if prec < 0 {
		// Auto: show up to 2 decimal places, trimming trailing zeros.
		s := strconv.FormatFloat(n, 'f', 2, 64)
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
		if nf.DecimalSep != "." {
			s = strings.ReplaceAll(s, ".", nf.DecimalSep)
		}
		return applyThousands(s, nf)
	}
	s := strconv.FormatFloat(n, 'f', prec, 64)
	if nf.DecimalSep != "." {
		s = strings.ReplaceAll(s, ".", nf.DecimalSep)
	}
	return applyThousands(s, nf)
}

func applyThousands(s string, nf NumberFormat) string {
	if nf.ThousandsSep == "" {
		return s
	}
	sep := strings.Index(s, nf.DecimalSep)
	intPart := s
	decPart := ""
	if sep >= 0 {
		intPart = s[:sep]
		decPart = s[sep:]
	}
	// Insert thousands separator every 3 digits from the right.
	var buf strings.Builder
	start := len(intPart) % 3
	if start == 0 {
		start = 3
	}
	buf.WriteString(intPart[:start])
	for i := start; i < len(intPart); i += 3 {
		buf.WriteString(nf.ThousandsSep)
		buf.WriteString(intPart[i : i+3])
	}
	return buf.String() + decPart
}

// FormatCurrency formats amount with a currency symbol and locale rules.
func FormatCurrency(amount float64, symbol string, locale Locale) string {
	nf := DefaultNumberFormat(locale)
	nf.Precision = 2
	lang, _ := ParseLocale(locale)
	num := FormatNumber(amount, nf)
	// Symbol position: most locales prefix, some suffix.
	switch lang {
	case "de", "fr", "nl", "sv", "no", "da":
		return fmt.Sprintf("%s %s", num, symbol)
	default:
		return fmt.Sprintf("%s%s", symbol, num)
	}
}

// FormatPercent formats a fraction [0,1] as a percentage string.
func FormatPercent(fraction float64, locale Locale) string {
	nf := DefaultNumberFormat(locale)
	nf.Precision = 0
	return FormatNumber(fraction*100, nf) + "%"
}

```

## File: form.go
Language: go | Tokens: 2966 | Size: 11865 bytes

```go
// Package ui — form.go
//
// Form validation and two-way field binding.
//
//   FieldRule     — a named validation rule + error message
//   FieldState    — live state for one field (value, dirty, touched, errors)
//   FormField     — binds a TextInput (or any Inputable) to a FieldState
//   Form          — collection of FormFields with submit/reset logic
//   Validators    — standard built-in validator factory functions
//
// Usage
//
//	email := ui.NewFormField("email", ui.Required(), ui.Email())
//	password := ui.NewFormField("password", ui.Required(), ui.MinLen(8))
//
//	form := ui.NewForm(email, password)
//	form.OnSubmit = func(values map[string]string) {
//	    // all fields valid; values["email"], values["password"]
//	}
//
//	// Wire into a Button:
//	submitBtn := ui.NewButton("Sign In", form.Submit)
//
//	// Wire a TextInput:
//	emailInput := ui.NewTextInput(style)
//	email.Bind(emailInput)
package ui

import (
	"regexp"
	"strconv"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Validator
// ═══════════════════════════════════════════════════════════════════════════════

// ValidatorFn validates a string value and returns an error message or "".
type ValidatorFn func(value string) string

// FieldRule pairs a name with a validator.
type FieldRule struct {
	Name      string
	Validator ValidatorFn
}

// ═══════════════════════════════════════════════════════════════════════════════
// Built-in validators
// ═══════════════════════════════════════════════════════════════════════════════

// Required fails if the value is empty or whitespace.
func Required() FieldRule {
	return FieldRule{"required", func(v string) string {
		if strings.TrimSpace(v) == "" {
			return "This field is required."
		}
		return ""
	}}
}

// MinLen fails if the value is shorter than n characters.
func MinLen(n int) FieldRule {
	return FieldRule{"minLen", func(v string) string {
		if len(v) < n {
			return "Must be at least " + strconv.Itoa(n) + " characters."
		}
		return ""
	}}
}

// MaxLen fails if the value is longer than n characters.
func MaxLen(n int) FieldRule {
	return FieldRule{"maxLen", func(v string) string {
		if len(v) > n {
			return "Must be at most " + strconv.Itoa(n) + " characters."
		}
		return ""
	}}
}

// Email fails if the value doesn't look like an email address.
func Email() FieldRule {
	re := regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	return FieldRule{"email", func(v string) string {
		if v != "" && !re.MatchString(v) {
			return "Enter a valid email address."
		}
		return ""
	}}
}

// URL fails if the value doesn't start with http:// or https://.
func URL() FieldRule {
	return FieldRule{"url", func(v string) string {
		if v != "" && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			return "Enter a valid URL."
		}
		return ""
	}}
}

// Numeric fails if the value cannot be parsed as a number.
func Numeric() FieldRule {
	return FieldRule{"numeric", func(v string) string {
		if v == "" {
			return ""
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return "Enter a numeric value."
		}
		return ""
	}}
}

// Min fails if the numeric value is less than min.
func Min(min float64) FieldRule {
	return FieldRule{"min", func(v string) string {
		if v == "" {
			return ""
		}
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "Enter a numeric value."
		}
		if n < min {
			return "Must be at least " + strconv.FormatFloat(min, 'f', -1, 64) + "."
		}
		return ""
	}}
}

// Max fails if the numeric value exceeds max.
func Max(max float64) FieldRule {
	return FieldRule{"max", func(v string) string {
		if v == "" {
			return ""
		}
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "Enter a numeric value."
		}
		if n > max {
			return "Must be at most " + strconv.FormatFloat(max, 'f', -1, 64) + "."
		}
		return ""
	}}
}

// Pattern fails if the value doesn't match the given regex pattern.
func Pattern(pattern, message string) FieldRule {
	re := regexp.MustCompile(pattern)
	return FieldRule{"pattern", func(v string) string {
		if v != "" && !re.MatchString(v) {
			return message
		}
		return ""
	}}
}

// Custom wraps an arbitrary validator.
func Custom(name string, fn ValidatorFn) FieldRule { return FieldRule{name, fn} }

// ═══════════════════════════════════════════════════════════════════════════════
// FieldState
// ═══════════════════════════════════════════════════════════════════════════════

// FieldState tracks the live state of a single form field.
type FieldState struct {
	Value   string
	Errors  []string
	Dirty   bool // value has been changed by the user
	Touched bool // field has lost focus at least once
	Valid   bool

	// Reactive signal; UI components can subscribe for rerenders.
	signal *Signal[string]
}

func newFieldState() *FieldState {
	fs := &FieldState{}
	fs.signal = NewSignal("")
	fs.Valid = true
	return fs
}

// ErrorMessage returns the first error message, or "".
func (fs *FieldState) ErrorMessage() string {
	if len(fs.Errors) > 0 {
		return fs.Errors[0]
	}
	return ""
}

// ShowError returns true when an error should be displayed
// (the field is dirty or touched and has errors).
func (fs *FieldState) ShowError() bool {
	return (fs.Dirty || fs.Touched) && !fs.Valid
}

// Changed returns a Stream[string] that emits on every value change.
func (fs *FieldState) Changed() *Stream[string] { return fs.signal.Changed() }

// ═══════════════════════════════════════════════════════════════════════════════
// Inputable  —  interface a widget must satisfy to be bindable
// ═══════════════════════════════════════════════════════════════════════════════

// Inputable is implemented by text-input components that can be bound to a FormField.
type Inputable interface {
	Component
	// GetText returns the current input value.
	GetText() string
	// SetText updates the displayed value.
	SetText(string)
	// OnChange registers a callback that fires on every keystroke.
	OnChange(func(string))
	// OnBlur registers a callback that fires when the input loses focus.
	OnBlur(func())
}

// ═══════════════════════════════════════════════════════════════════════════════
// FormField
// ═══════════════════════════════════════════════════════════════════════════════

// FormField links a set of validators to a FieldState and an optional Inputable.
type FormField struct {
	Name  string
	State *FieldState
	rules []FieldRule

	input Inputable
	subs  []*Subscription
}

// NewFormField creates a FormField with the given validation rules.
func NewFormField(name string, rules ...FieldRule) *FormField {
	return &FormField{Name: name, State: newFieldState(), rules: rules}
}

// Bind attaches an Inputable widget to this field.
// The field will automatically read/write through the widget.
func (f *FormField) Bind(input Inputable) {
	// Unsubscribe previous.
	for _, s := range f.subs {
		s.Unsubscribe()
	}
	f.subs = nil
	f.input = input

	input.OnChange(func(v string) {
		f.State.Value = v
		f.State.Dirty = true
		f.validate()
		f.State.signal.Set(v)
	})
	input.OnBlur(func() {
		f.State.Touched = true
		f.validate()
		f.State.signal.Set(f.State.Value)
	})
	// Seed value from current input.
	f.State.Value = input.GetText()
	f.validate()
}

// SetValue programmatically sets the field value and runs validation.
func (f *FormField) SetValue(v string) {
	f.State.Value = v
	f.State.Dirty = true
	if f.input != nil {
		f.input.SetText(v)
	}
	f.validate()
	f.State.signal.Set(v)
}

// Reset clears the field to its initial empty state.
func (f *FormField) Reset() {
	f.State.Value = ""
	f.State.Dirty = false
	f.State.Touched = false
	f.State.Errors = nil
	f.State.Valid = true
	if f.input != nil {
		f.input.SetText("")
	}
	f.State.signal.Set("")
}

func (f *FormField) validate() {
	var errs []string
	for _, r := range f.rules {
		if msg := r.Validator(f.State.Value); msg != "" {
			errs = append(errs, msg)
		}
	}
	f.State.Errors = errs
	f.State.Valid = len(errs) == 0
}

// ═══════════════════════════════════════════════════════════════════════════════
// Form
// ═══════════════════════════════════════════════════════════════════════════════

// Form groups multiple FormFields and manages submit/reset.
type Form struct {
	Fields   []*FormField
	OnSubmit func(values map[string]string)
	OnReset  func()

	submitted bool
}

// NewForm creates a Form with the given fields.
func NewForm(fields ...*FormField) *Form {
	return &Form{Fields: fields}
}

// Submit validates all fields and, if valid, calls OnSubmit.
// After calling Submit, all fields are marked dirty+touched so errors are shown.
func (f *Form) Submit() {
	f.submitted = true
	for _, field := range f.Fields {
		field.State.Dirty = true
		field.State.Touched = true
		field.validate()
		field.State.signal.Set(field.State.Value)
	}
	if !f.Valid() {
		return
	}
	if f.OnSubmit != nil {
		f.OnSubmit(f.Values())
	}
}

// Reset resets all fields to their initial state.
func (f *Form) Reset() {
	f.submitted = false
	for _, field := range f.Fields {
		field.Reset()
	}
	if f.OnReset != nil {
		f.OnReset()
	}
}

// Valid returns true if all fields are currently valid.
func (f *Form) Valid() bool {
	for _, field := range f.Fields {
		if !field.State.Valid {
			return false
		}
	}
	return true
}

// Values returns a map of field name → current value for all fields.
func (f *Form) Values() map[string]string {
	m := make(map[string]string, len(f.Fields))
	for _, field := range f.Fields {
		m[field.Name] = field.State.Value
	}
	return m
}

// Field returns the FormField with the given name, or nil.
func (f *Form) Field(name string) *FormField {
	for _, field := range f.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}

// SetValues populates multiple fields by name.
func (f *Form) SetValues(values map[string]string) {
	for name, val := range values {
		if field := f.Field(name); field != nil {
			field.SetValue(val)
		}
	}
}

```

## File: drag_drop.go
Language: go | Tokens: 2761 | Size: 11046 bytes

```go
// Package ui — drag_drop.go
//
// Drag-and-drop system.
//
//   DragPayload   — typed data carried during a drag
//   DragSource    — component that initiates drags
//   DropTarget    — component that accepts drops
//   DragDropManager — central coordinator; call from the window event loop
//
// Usage
//
//	ddm := ui.NewDragDropManager()
//
//	// Make a component draggable:
//	src := &ui.DragSource{
//	    Component: myCard,
//	    PayloadFn: func() ui.DragPayload { return ui.DragPayload{Type: "card", Data: card} },
//	}
//	ddm.RegisterSource(src)
//
//	// Make a component a drop zone:
//	zone := &ui.DropTarget{
//	    Component: myZone,
//	    Accept:    func(p ui.DragPayload) bool { return p.Type == "card" },
//	    OnDrop:    func(p ui.DragPayload, x, y float32) { /* handle drop */ },
//	}
//	ddm.RegisterTarget(zone)
//
//	// In window HandleEvent:
//	ddm.HandleEvent(e)
package ui

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
)

// ═══════════════════════════════════════════════════════════════════════════════
// DragPayload
// ═══════════════════════════════════════════════════════════════════════════════

// DragPayload carries data from drag source to drop target.
type DragPayload struct {
	Type   string // MIME-like type string, e.g. "text/plain", "app/card"
	Data   any    // arbitrary application data
	Source Component
}

// ═══════════════════════════════════════════════════════════════════════════════
// DragSource
// ═══════════════════════════════════════════════════════════════════════════════

// DragSource wraps a Component to make it draggable.
type DragSource struct {
	Component Component
	// PayloadFn is called when a drag begins to produce the payload.
	PayloadFn func() DragPayload
	// DragThreshold is the minimum pixel distance before a drag starts.
	// Default: 4.
	DragThreshold float32
	// OnDragStart is called when dragging begins.
	OnDragStart func(DragPayload)
	// OnDragEnd is called when the drag ends (dropped or cancelled).
	OnDragEnd func(dropped bool)
	// Cursor override during drag.
	DragCursor string

	pressing    bool
	pressX      float32
	pressY      float32
}

// ═══════════════════════════════════════════════════════════════════════════════
// DropTarget
// ═══════════════════════════════════════════════════════════════════════════════

// DropTarget wraps a Component to make it accept drops.
type DropTarget struct {
	Component Component
	// Accept returns true if this target can receive the given payload.
	Accept func(DragPayload) bool
	// OnDragEnter is called when a compatible drag enters the target bounds.
	OnDragEnter func(DragPayload)
	// OnDragLeave is called when the drag leaves the target bounds.
	OnDragLeave func()
	// OnDrop is called when the payload is dropped here.
	// x,y are in local component coordinates.
	OnDrop func(DragPayload, float32, float32)

	hovered bool
}

// DropHighlightStyle describes how a drop target renders its hover state.
type DropHighlightStyle struct {
	BorderColor canvas.Color
	BorderWidth float32
	FillColor   canvas.Color
	Radius      float32
}

// DefaultDropHighlightStyle returns the default drop target highlight.
func DefaultDropHighlightStyle() DropHighlightStyle {
	th := theme.Current()
	return DropHighlightStyle{
		BorderColor: th.Colors.Accent,
		BorderWidth: 2,
		FillColor:   canvas.Color{R: th.Colors.Accent.R, G: th.Colors.Accent.G, B: th.Colors.Accent.B, A: 0.08},
		Radius:      th.Radius.MD,
	}
}

// DrawHighlight renders the drop-target hover overlay.
func (dt *DropTarget) DrawHighlight(c *canvas.Canvas, style DropHighlightStyle) {
	if !dt.hovered {
		return
	}
	b := dt.Component.Bounds()
	c.DrawRoundedRect(b.X, b.Y, b.W, b.H, style.Radius, canvas.FillPaint(style.FillColor))
	c.DrawRoundedRect(b.X, b.Y, b.W, b.H, style.Radius, canvas.StrokePaint(style.BorderColor, style.BorderWidth))
}

// ═══════════════════════════════════════════════════════════════════════════════
// Ghost  —  drag preview
// ═══════════════════════════════════════════════════════════════════════════════

// GhostRenderer draws the drag ghost (the floating preview that follows the cursor).
type GhostRenderer func(c *canvas.Canvas, payload DragPayload, x, y float32)

// DefaultGhostRenderer draws a simple semi-transparent label.
func DefaultGhostRenderer(c *canvas.Canvas, p DragPayload, x, y float32) {
	th := theme.Current()
	label := p.Type
	if s, ok := p.Data.(string); ok {
		label = s
	}
	w := float32(120)
	h := float32(36)
	c.DrawRoundedRect(x-w/2, y-h/2, w, h, th.Radius.SM,
		canvas.FillPaint(canvas.Color{R: th.Colors.BgSurface.R, G: th.Colors.BgSurface.G,
			B: th.Colors.BgSurface.B, A: 0.9}))
	c.DrawRoundedRect(x-w/2, y-h/2, w, h, th.Radius.SM,
		canvas.StrokePaint(th.Colors.Accent, 1))
	ts := canvas.TextStyle{Color: th.Colors.TextPrimary, Size: 12}
	tw := c.MeasureText(label, ts).W
	c.DrawText(x-tw/2, y+5, label, ts)
}

// ═══════════════════════════════════════════════════════════════════════════════
// DragDropManager
// ═══════════════════════════════════════════════════════════════════════════════

// DragDropManager coordinates drag sources, drop targets, and the ghost preview.
type DragDropManager struct {
	sources []*DragSource
	targets []*DropTarget

	// Active drag state.
	dragging    bool
	payload     DragPayload
	ghostX      float32
	ghostY      float32
	activeSource *DragSource

	// GhostRenderer can be overridden to customise the drag preview.
	GhostRenderer GhostRenderer

	// HighlightStyle controls how hovering drop targets look.
	HighlightStyle DropHighlightStyle
}

// NewDragDropManager creates a DragDropManager.
func NewDragDropManager() *DragDropManager {
	return &DragDropManager{
		GhostRenderer:  DefaultGhostRenderer,
		HighlightStyle: DefaultDropHighlightStyle(),
	}
}

// RegisterSource adds a DragSource.
func (m *DragDropManager) RegisterSource(src *DragSource) {
	if src.DragThreshold <= 0 {
		src.DragThreshold = 4
	}
	m.sources = append(m.sources, src)
}

// RegisterTarget adds a DropTarget.
func (m *DragDropManager) RegisterTarget(t *DropTarget) {
	m.targets = append(m.targets, t)
}

// IsDragging reports whether a drag is currently in progress.
func (m *DragDropManager) IsDragging() bool { return m.dragging }

// HandleEvent processes raw events. Call this before routing events to components.
// Returns true if the event was consumed by the drag-and-drop system.
func (m *DragDropManager) HandleEvent(e Event) bool {
	switch e.Type {
	case EventMouseDown:
		if e.Button != 1 {
			return false
		}
		// Check if a source is being pressed.
		for _, src := range m.sources {
			b := src.Component.Bounds()
			if e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H {
				src.pressing = true
				src.pressX = e.X
				src.pressY = e.Y
			}
		}

	case EventMouseMove:
		if m.dragging {
			m.ghostX = e.X
			m.ghostY = e.Y
			m.updateHover(e.X, e.Y)
			return true
		}
		// Threshold check.
		for _, src := range m.sources {
			if !src.pressing {
				continue
			}
			dx := e.X - src.pressX
			dy := e.Y - src.pressY
			dist := dx*dx + dy*dy
			thresh := src.DragThreshold
			if dist > thresh*thresh {
				m.beginDrag(src, e.X, e.Y)
				return true
			}
		}

	case EventMouseUp:
		for _, src := range m.sources {
			src.pressing = false
		}
		if m.dragging {
			m.endDrag(e.X, e.Y)
			return true
		}
	}
	return false
}

// DrawGhost renders the drag ghost at the current cursor position.
// Call this at the very end of your Draw method (on top of everything).
func (m *DragDropManager) DrawGhost(c *canvas.Canvas) {
	if !m.dragging || m.GhostRenderer == nil {
		return
	}
	m.GhostRenderer(c, m.payload, m.ghostX, m.ghostY)
	// Draw highlights on hovered targets.
	for _, t := range m.targets {
		t.DrawHighlight(c, m.HighlightStyle)
	}
}

// ── internal ──────────────────────────────────────────────────────────────────

func (m *DragDropManager) beginDrag(src *DragSource, x, y float32) {
	if src.PayloadFn == nil {
		return
	}
	m.payload = src.PayloadFn()
	m.dragging = true
	m.ghostX = x
	m.ghostY = y
	m.activeSource = src
	if src.OnDragStart != nil {
		src.OnDragStart(m.payload)
	}
	m.updateHover(x, y)
}

func (m *DragDropManager) endDrag(x, y float32) {
	dropped := false
	for _, t := range m.targets {
		b := t.Component.Bounds()
		if x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H {
			if t.Accept != nil && !t.Accept(m.payload) {
				continue
			}
			localX := x - b.X
			localY := y - b.Y
			if t.OnDrop != nil {
				t.OnDrop(m.payload, localX, localY)
			}
			dropped = true
		}
	}
	// Clear hover.
	for _, t := range m.targets {
		if t.hovered {
			t.hovered = false
			if t.OnDragLeave != nil {
				t.OnDragLeave()
			}
		}
	}
	if m.activeSource != nil && m.activeSource.OnDragEnd != nil {
		m.activeSource.OnDragEnd(dropped)
	}
	m.dragging = false
	m.activeSource = nil
	m.payload = DragPayload{}
}

func (m *DragDropManager) updateHover(x, y float32) {
	for _, t := range m.targets {
		b := t.Component.Bounds()
		over := x >= b.X && x <= b.X+b.W && y >= b.Y && y <= b.Y+b.H &&
			(t.Accept == nil || t.Accept(m.payload))
		if over && !t.hovered {
			t.hovered = true
			if t.OnDragEnter != nil {
				t.OnDragEnter(m.payload)
			}
		} else if !over && t.hovered {
			t.hovered = false
			if t.OnDragLeave != nil {
				t.OnDragLeave()
			}
		}
	}
}

```

## File: layout_flex.go
Language: go | Tokens: 2657 | Size: 10630 bytes

**Imports:** github.com/achiket/gui-go/canvas

```go
// Package ui — layout_flex.go
//
// FlexLayout — a CSS Flexbox–inspired one-dimensional layout engine.
//
// Features:
//   - Row or Column direction
//   - Wrap / NoWrap
//   - justify-content: Start, End, Center, SpaceBetween, SpaceAround, SpaceEvenly
//   - align-items: Start, End, Center, Stretch, Baseline
//   - align-self per item (overrides align-items)
//   - flex-grow / flex-shrink / flex-basis per item
//   - gap (uniform) or RowGap / ColGap
//   - Padding on the container
//
// Usage
//
//	flex := ui.NewFlexLayout(ui.FlexRow)
//	flex.JustifyContent = ui.JustifySpaceBetween
//	flex.AlignItems = ui.AlignCenter
//	flex.Gap = 8
//	flex.Add(logo, ui.FlexItem{Grow: 0})
//	flex.Add(searchBar, ui.FlexItem{Grow: 1})
//	flex.Add(avatar, ui.FlexItem{Grow: 0})
package ui

import "github.com/achiket/gui-go/canvas"

// ═══════════════════════════════════════════════════════════════════════════════
// Direction / Justify / Align enums
// ═══════════════════════════════════════════════════════════════════════════════

// FlexDirection is the main axis direction.
type FlexDirection int

const (
	FlexRow    FlexDirection = iota // left → right
	FlexColumn                      // top → bottom
	FlexRowReverse                  // right → left
	FlexColumnReverse               // bottom → top
)

// JustifyContent aligns items along the main axis.
type JustifyContent int

const (
	JustifyStart        JustifyContent = iota
	JustifyEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// AlignItems aligns items along the cross axis.
type AlignItems int

const (
	AlignItemsStart   AlignItems = iota
	AlignItemsEnd
	AlignItemsCenter
	AlignItemsStretch
)

// FlexWrap controls line wrapping.
type FlexWrap int

const (
	FlexNoWrap FlexWrap = iota
	FlexWrapLines
	FlexWrapReverse
)

// ═══════════════════════════════════════════════════════════════════════════════
// FlexItem
// ═══════════════════════════════════════════════════════════════════════════════

// FlexItem is the per-child layout descriptor.
type FlexItem struct {
	// Grow is the proportion of remaining space this item takes. 0 = no grow.
	Grow float32
	// Shrink controls how much the item shrinks relative to others. Default 1.
	Shrink float32
	// Basis is the initial main-size before grow/shrink is applied. -1 = auto.
	Basis float32
	// AlignSelf overrides the container's AlignItems for this item.
	// Use -1 (default) to inherit.
	AlignSelf AlignItems
	// Order shifts the render position. Lower order = earlier.
	Order int
	// MinSize is the minimum main-axis size.
	MinSize float32
	// MaxSize is the maximum main-axis size. 0 = no limit.
	MaxSize float32

	child Component
}

// ═══════════════════════════════════════════════════════════════════════════════
// FlexLayout
// ═══════════════════════════════════════════════════════════════════════════════

// FlexLayout arranges children along one axis using flexbox semantics.
type FlexLayout struct {
	Direction      FlexDirection
	JustifyContent JustifyContent
	AlignItems     AlignItems
	Wrap           FlexWrap
	Gap            float32
	RowGap         float32 // overrides Gap for row spacing (wrap mode)
	ColGap         float32 // overrides Gap for column spacing

	// Padding inside the container.
	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft float32

	items  []FlexItem
	bounds canvas.Rect
}

// NewFlexLayout creates a FlexLayout with the given direction.
func NewFlexLayout(direction FlexDirection) *FlexLayout {
	return &FlexLayout{Direction: direction, AlignItems: AlignItemsStretch}
}

// Add appends a child with the given FlexItem descriptor.
func (fl *FlexLayout) Add(child Component, item FlexItem) {
	if item.Shrink == 0 {
		item.Shrink = 1
	}
	if item.Basis == 0 {
		item.Basis = -1
	}
	if item.AlignSelf == 0 {
		item.AlignSelf = -1
	}
	item.child = child
	fl.items = append(fl.items, item)
}

// AddSimple adds a child with default flex properties (no grow, shrink=1, auto basis).
func (fl *FlexLayout) AddSimple(child Component) {
	fl.Add(child, FlexItem{Shrink: 1, Basis: -1, AlignSelf: -1})
}

// AddGrow adds a child that grows to fill remaining space.
func (fl *FlexLayout) AddGrow(child Component, grow float32) {
	fl.Add(child, FlexItem{Grow: grow, Shrink: 1, Basis: -1, AlignSelf: -1})
}

func (fl *FlexLayout) Bounds() canvas.Rect { return fl.bounds }

func (fl *FlexLayout) Tick(delta float64) {
	for _, it := range fl.items {
		if it.child != nil {
			it.child.Tick(delta)
		}
	}
}

func (fl *FlexLayout) HandleEvent(e Event) bool {
	for i := len(fl.items) - 1; i >= 0; i-- {
		if fl.items[i].child != nil && fl.items[i].child.HandleEvent(e) {
			return true
		}
	}
	return false
}

func (fl *FlexLayout) Draw(c *canvas.Canvas, x, y, w, h float32) {
	fl.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	// Apply padding.
	cx := x + fl.PaddingLeft
	cy := y + fl.PaddingTop
	cw := w - fl.PaddingLeft - fl.PaddingRight
	ch := h - fl.PaddingTop - fl.PaddingBottom

	isRow := fl.Direction == FlexRow || fl.Direction == FlexRowReverse
	colGap := fl.ColGap
	if colGap == 0 {
		colGap = fl.Gap
	}
	rowGap := fl.RowGap
	if rowGap == 0 {
		rowGap = fl.Gap
	}
	mainGap := colGap
	if !isRow {
		mainGap = rowGap
	}

	// Sort items by Order.
	sorted := make([]int, len(fl.items))
	for i := range sorted {
		sorted[i] = i
	}
	sortByOrder(sorted, fl.items)

	// Resolve basis sizes.
	mainSize := cw
	if !isRow {
		mainSize = ch
	}
	crossSize := ch
	if !isRow {
		crossSize = cw
	}

	bases := make([]float32, len(sorted))
	totalBasis := float32(0)
	totalGrow := float32(0)
	for _, si := range sorted {
		it := fl.items[si]
		b := it.Basis
		if b < 0 {
			// Auto: use natural bounds if child has been drawn, otherwise 0.
			cb := it.child.Bounds()
			if isRow {
				b = cb.W
			} else {
				b = cb.H
			}
			if b <= 0 {
				b = 0
			}
		}
		if it.MinSize > 0 && b < it.MinSize {
			b = it.MinSize
		}
		if it.MaxSize > 0 && b > it.MaxSize {
			b = it.MaxSize
		}
		bases[si] = b
		totalBasis += b
		totalGrow += it.Grow
	}

	// Total gap space.
	n := len(sorted)
	totalGapSpace := float32(0)
	if n > 1 {
		totalGapSpace = mainGap * float32(n-1)
	}

	// Distribute remaining space.
	remaining := mainSize - totalBasis - totalGapSpace
	sizes := make([]float32, len(sorted))
	for _, si := range sorted {
		it := fl.items[si]
		s := bases[si]
		if totalGrow > 0 && remaining > 0 {
			s += remaining * (it.Grow / totalGrow)
		}
		if it.MinSize > 0 && s < it.MinSize {
			s = it.MinSize
		}
		if it.MaxSize > 0 && s > it.MaxSize {
			s = it.MaxSize
		}
		sizes[si] = s
	}

	// Compute total used space for justify-content.
	usedMain := totalGapSpace
	for _, si := range sorted {
		usedMain += sizes[si]
	}
	freeMain := mainSize - usedMain

	// Starting offset and spacing based on JustifyContent.
	startOffset, spacing := justifyOffsets(fl.JustifyContent, freeMain, n)

	// Reverse direction adjustments.
	if fl.Direction == FlexRowReverse || fl.Direction == FlexColumnReverse {
		startOffset = freeMain - startOffset
	}

	// Draw items.
	pos := startOffset
	if fl.Direction == FlexRowReverse {
		pos = mainSize - startOffset
	} else if fl.Direction == FlexColumnReverse {
		pos = mainSize - startOffset
	}

	for idx, si := range sorted {
		it := fl.items[si]
		if it.child == nil {
			continue
		}
		sz := sizes[si]

		align := fl.AlignItems
		if it.AlignSelf >= 0 {
			align = it.AlignSelf
		}

		var itemX, itemY, itemW, itemH float32
		if isRow {
			itemW = sz
			if align == AlignItemsStretch {
				itemH = crossSize
			} else {
				// Natural height.
				nat := it.child.Bounds().H
				if nat <= 0 {
					nat = crossSize
				}
				itemH = nat
			}
			itemX = cx + pos
			if fl.Direction == FlexRowReverse {
				itemX = cx + pos - sz
			}
			switch align {
			case AlignItemsStart:
				itemY = cy
			case AlignItemsEnd:
				itemY = cy + crossSize - itemH
			case AlignItemsCenter:
				itemY = cy + (crossSize-itemH)/2
			default:
				itemY = cy
			}
		} else {
			itemH = sz
			if align == AlignItemsStretch {
				itemW = crossSize
			} else {
				nat := it.child.Bounds().W
				if nat <= 0 {
					nat = crossSize
				}
				itemW = nat
			}
			itemY = cy + pos
			if fl.Direction == FlexColumnReverse {
				itemY = cy + pos - sz
			}
			switch align {
			case AlignItemsStart:
				itemX = cx
			case AlignItemsEnd:
				itemX = cx + crossSize - itemW
			case AlignItemsCenter:
				itemX = cx + (crossSize-itemW)/2
			default:
				itemX = cx
			}
		}

		it.child.Draw(c, itemX, itemY, itemW, itemH)

		step := sz + mainGap + spacing
		if fl.Direction == FlexRowReverse || fl.Direction == FlexColumnReverse {
			pos -= step
		} else {
			pos += step
		}
		_ = idx
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func justifyOffsets(j JustifyContent, free float32, n int) (start, spacing float32) {
	if n <= 0 {
		return 0, 0
	}
	switch j {
	case JustifyEnd:
		return free, 0
	case JustifyCenter:
		return free / 2, 0
	case JustifySpaceBetween:
		if n == 1 {
			return 0, 0
		}
		return 0, free / float32(n-1)
	case JustifySpaceAround:
		s := free / float32(n)
		return s / 2, s
	case JustifySpaceEvenly:
		s := free / float32(n+1)
		return s, s
	default: // JustifyStart
		return 0, 0
	}
}

func sortByOrder(indices []int, items []FlexItem) {
	// Insertion sort (items are usually few and nearly sorted).
	for i := 1; i < len(indices); i++ {
		key := indices[i]
		j := i - 1
		for j >= 0 && items[indices[j]].Order > items[key].Order {
			indices[j+1] = indices[j]
			j--
		}
		indices[j+1] = key
	}
}

```

## File: virtual_list.go
Language: go | Tokens: 2554 | Size: 10219 bytes

```go
// Package ui — virtual_list.go
//
// VirtualList renders only the visible rows of a large dataset, recycling
// draw calls for items outside the viewport.  Handles 100 000+ items smoothly.
//
//   VirtualList   — fixed row-height virtualized list
//   VirtualGrid   — fixed cell-size virtualized grid (multi-column)
//
// Usage
//
//	list := ui.NewVirtualList(10000, 48, func(c *canvas.Canvas, index int, x, y, w, h float32) {
//	    c.DrawText(x+12, y+h/2+6, fmt.Sprintf("Row %d", index), style)
//	})
//	list.SetScrollOffset(0)
package ui

import (
	"math"

	"github.com/achiket/gui-go/canvas"
)

// RowRenderer draws one row of the list.
//   index — data index of the row
//   x,y   — top-left of the row's bounding box
//   w,h   — dimensions of the row
type RowRenderer func(c *canvas.Canvas, index int, x, y, w, h float32)

// ═══════════════════════════════════════════════════════════════════════════════
// VirtualList
// ═══════════════════════════════════════════════════════════════════════════════

// VirtualList renders only the rows visible within the current scroll window.
type VirtualList struct {
	Count     int         // total number of rows
	RowHeight float32     // fixed row height in pixels
	Render    RowRenderer // callback to draw one row

	// Overscan: extra rows to render above and below the visible area.
	// Higher values reduce flicker at the cost of more draw calls.
	Overscan int

	// OnScroll is called whenever the scroll offset changes.
	OnScroll func(offset float32)

	scroll   *ScrollView
	bounds   canvas.Rect
}

// NewVirtualList creates a VirtualList.
//   count     — total data items
//   rowHeight — fixed row height (px)
//   render    — row draw callback
func NewVirtualList(count int, rowHeight float32, render RowRenderer) *VirtualList {
	vl := &VirtualList{
		Count:     count,
		RowHeight: rowHeight,
		Render:    render,
		Overscan:  3,
	}
	totalH := float32(count) * rowHeight
	vl.scroll = NewScrollView(totalH, func(c *canvas.Canvas, x, y, w, _ float32) {
		vl.drawVisible(c, x, y, w)
	})
	vl.scroll.OnScroll = func(offset float32) {
		if vl.OnScroll != nil {
			vl.OnScroll(offset)
		}
	}
	return vl
}

// UpdateCount changes the total item count and recalculates total scroll height.
func (vl *VirtualList) UpdateCount(count int) {
	vl.Count = count
	vl.scroll.SetContentHeight(float32(count) * vl.RowHeight)
}

// ScrollToIndex scrolls so that the row at index is visible.
func (vl *VirtualList) ScrollToIndex(index int) {
	offset := float32(index) * vl.RowHeight
	vl.scroll.SetScrollOffset(offset)
}

// ScrollOffset returns the current scroll offset.
func (vl *VirtualList) ScrollOffset() float32 { return vl.scroll.ScrollOffset() }

func (vl *VirtualList) Bounds() canvas.Rect { return vl.bounds }

func (vl *VirtualList) Tick(delta float64) { vl.scroll.Tick(delta) }

func (vl *VirtualList) HandleEvent(e Event) bool { return vl.scroll.HandleEvent(e) }

func (vl *VirtualList) Draw(c *canvas.Canvas, x, y, w, h float32) {
	vl.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	vl.scroll.Draw(c, x, y, w, h)
}

// drawVisible is called by the ScrollView with the content area coordinates.
func (vl *VirtualList) drawVisible(c *canvas.Canvas, contentX, contentY, w float32) {
	if vl.Render == nil || vl.Count == 0 || vl.RowHeight <= 0 {
		return
	}
	viewH := vl.bounds.H
	offset := vl.scroll.ScrollOffset()

	firstRow := int(math.Floor(float64(offset/vl.RowHeight))) - vl.Overscan
	if firstRow < 0 {
		firstRow = 0
	}
	lastRow := int(math.Ceil(float64((offset+viewH)/vl.RowHeight))) + vl.Overscan
	if lastRow >= vl.Count {
		lastRow = vl.Count - 1
	}

	for i := firstRow; i <= lastRow; i++ {
		ry := contentY + float32(i)*vl.RowHeight
		vl.Render(c, i, contentX, ry, w, vl.RowHeight)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// VirtualGrid
// ═══════════════════════════════════════════════════════════════════════════════

// CellRenderer draws one cell of the grid.
type CellRenderer func(c *canvas.Canvas, row, col int, x, y, w, h float32)

// VirtualGrid renders a fixed-cell-size grid for large datasets.
type VirtualGrid struct {
	Count      int          // total number of cells
	Cols       int          // number of columns
	CellWidth  float32
	CellHeight float32
	Gap        float32
	Render     CellRenderer
	Overscan   int

	scroll *ScrollView
	bounds canvas.Rect
}

// NewVirtualGrid creates a VirtualGrid.
func NewVirtualGrid(count, cols int, cellW, cellH, gap float32, render CellRenderer) *VirtualGrid {
	vg := &VirtualGrid{
		Count: count, Cols: cols,
		CellWidth: cellW, CellHeight: cellH,
		Gap: gap, Render: render, Overscan: 2,
	}
	rows := int(math.Ceil(float64(count) / float64(cols)))
	totalH := float32(rows)*(cellH+gap) - gap
	vg.scroll = NewScrollView(totalH, func(c *canvas.Canvas, x, y, w, _ float32) {
		vg.drawVisible(c, x, y, w)
	})
	return vg
}

// UpdateCount adjusts the item count and recalculates total height.
func (vg *VirtualGrid) UpdateCount(count int) {
	vg.Count = count
	rows := int(math.Ceil(float64(count) / float64(vg.Cols)))
	vg.scroll.SetContentHeight(float32(rows)*(vg.CellHeight+vg.Gap) - vg.Gap)
}

func (vg *VirtualGrid) Bounds() canvas.Rect { return vg.bounds }
func (vg *VirtualGrid) Tick(d float64)      { vg.scroll.Tick(d) }
func (vg *VirtualGrid) HandleEvent(e Event) bool { return vg.scroll.HandleEvent(e) }
func (vg *VirtualGrid) Draw(c *canvas.Canvas, x, y, w, h float32) {
	vg.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	vg.scroll.Draw(c, x, y, w, h)
}

func (vg *VirtualGrid) drawVisible(c *canvas.Canvas, cx, cy, w float32) {
	if vg.Render == nil || vg.Count == 0 {
		return
	}
	viewH := vg.bounds.H
	offset := vg.scroll.ScrollOffset()
	rowH := vg.CellHeight + vg.Gap

	firstRow := int(math.Floor(float64(offset/rowH))) - vg.Overscan
	if firstRow < 0 {
		firstRow = 0
	}
	lastRow := int(math.Ceil(float64((offset+viewH)/rowH))) + vg.Overscan

	totalRows := int(math.Ceil(float64(vg.Count) / float64(vg.Cols)))
	if lastRow >= totalRows {
		lastRow = totalRows - 1
	}

	for r := firstRow; r <= lastRow; r++ {
		for col := 0; col < vg.Cols; col++ {
			idx := r*vg.Cols + col
			if idx >= vg.Count {
				break
			}
			cellX := cx + float32(col)*(vg.CellWidth+vg.Gap)
			cellY := cy + float32(r)*rowH
			vg.Render(c, r, col, cellX, cellY, vg.CellWidth, vg.CellHeight)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// RecyclingList  —  component-based recycling (like Android RecyclerView)
// ═══════════════════════════════════════════════════════════════════════════════

// RecycleFactory creates a new Component for a given data index.
// Implementations should be as cheap as possible; RecyclingList pools them.
type RecycleFactory func(index int) Component

// RecyclingList pools a fixed number of Component instances and reuses them
// as the user scrolls, calling Rebind to update data binding.
// Use this when your rows are interactive (contain buttons, inputs, etc.)
// and cannot be drawn by a plain RowRenderer.
type RecyclingList struct {
	Count     int
	RowHeight float32
	Factory   RecycleFactory
	Rebind    func(comp Component, index int)

	pool       []Component
	poolIndex  []int // which data index this pool slot currently shows
	firstVisible int

	scroll *ScrollView
	bounds canvas.Rect
}

// NewRecyclingList creates a RecyclingList with poolSize recycled components.
// poolSize should be slightly larger than the number of visible rows.
func NewRecyclingList(count int, rowHeight float32, poolSize int,
	factory RecycleFactory, rebind func(Component, int)) *RecyclingList {

	rl := &RecyclingList{
		Count: count, RowHeight: rowHeight,
		Factory: factory, Rebind: rebind,
	}
	rl.pool = make([]Component, poolSize)
	rl.poolIndex = make([]int, poolSize)
	for i := range rl.pool {
		rl.pool[i] = factory(i)
		rl.poolIndex[i] = i
	}
	rl.scroll = NewScrollView(float32(count)*rowHeight, func(c *canvas.Canvas, x, y, w, _ float32) {
		rl.drawPooled(c, x, y, w)
	})
	return rl
}

func (rl *RecyclingList) Bounds() canvas.Rect        { return rl.bounds }
func (rl *RecyclingList) Tick(d float64)              {
	rl.scroll.Tick(d)
	for _, p := range rl.pool {
		p.Tick(d)
	}
}
func (rl *RecyclingList) HandleEvent(e Event) bool {
	if rl.scroll.HandleEvent(e) {
		return true
	}
	for _, p := range rl.pool {
		if p.HandleEvent(e) {
			return true
		}
	}
	return false
}
func (rl *RecyclingList) Draw(c *canvas.Canvas, x, y, w, h float32) {
	rl.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	rl.scroll.Draw(c, x, y, w, h)
}

func (rl *RecyclingList) drawPooled(c *canvas.Canvas, cx, cy, w float32) {
	if rl.Count == 0 {
		return
	}
	viewH := rl.bounds.H
	offset := rl.scroll.ScrollOffset()

	first := int(math.Floor(float64(offset / rl.RowHeight)))
	if first < 0 {
		first = 0
	}

	// Rebind pool slots to the visible indices.
	for slot, comp := range rl.pool {
		idx := first + slot
		if idx >= rl.Count {
			break
		}
		if rl.poolIndex[slot] != idx {
			rl.poolIndex[slot] = idx
			if rl.Rebind != nil {
				rl.Rebind(comp, idx)
			}
		}
		ry := cy + float32(idx)*rl.RowHeight
		// Clip to visible area.
		if ry+rl.RowHeight < cy+offset || ry > cy+offset+viewH {
			continue
		}
		comp.Draw(c, cx, ry, w, rl.RowHeight)
	}
}

```

## File: focus.go
Language: go | Tokens: 2288 | Size: 9152 bytes

```go
// Package ui — focus.go
//
// Keyboard focus management.
//
//   FocusManager  — tracks which component has focus; routes KeyDown events
//   Focusable     — interface that components implement to participate
//   FocusRing     — draws an accessibility focus indicator around the focused component
//   TabOrder      — explicit ordered list for Tab/Shift-Tab navigation
//
// Integration
//
//	fm := ui.NewFocusManager()
//	fm.Register(button1, button2, textField)
//
//	// In window HandleEvent:
//	if fm.HandleEvent(e) { return true }
//
//	// In window Draw (after drawing all components):
//	fm.DrawFocusRing(canvas)
package ui

import (
	"github.com/achiket/gui-go/canvas"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Focusable
// ═══════════════════════════════════════════════════════════════════════════════

// Focusable is implemented by components that can receive keyboard focus.
type Focusable interface {
	Component
	// FocusGained is called when this component receives focus.
	FocusGained()
	// FocusLost is called when focus moves away.
	FocusLost()
	// HandleKeyEvent processes a keyboard event. Returns true if consumed.
	HandleKeyEvent(e Event) bool
	// IsFocusable returns false to exclude the component from the tab order
	// (e.g. when disabled).
	IsFocusable() bool
}

// ═══════════════════════════════════════════════════════════════════════════════
// FocusRingStyle
// ═══════════════════════════════════════════════════════════════════════════════

// FocusRingStyle describes how to render the focus indicator.
type FocusRingStyle struct {
	Color   canvas.Color
	Width   float32
	Radius  float32
	Padding float32 // extra space around the component's Bounds
}

// DefaultFocusRingStyle returns an accessible blue focus ring.
func DefaultFocusRingStyle() FocusRingStyle {
	return FocusRingStyle{
		Color:   canvas.Hex("#4D9FFF"),
		Width:   2,
		Radius:  4,
		Padding: 2,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// FocusManager
// ═══════════════════════════════════════════════════════════════════════════════

// FocusManager owns the focus state for a window.
// It routes keyboard events to the focused component and drives Tab navigation.
type FocusManager struct {
	order   []Focusable
	focused int // index into order; -1 = nothing focused
	style   FocusRingStyle

	// OnFocusChange is called with the newly focused component (nil = cleared).
	OnFocusChange func(Focusable)
}

// NewFocusManager creates a FocusManager with the default ring style.
func NewFocusManager() *FocusManager {
	return &FocusManager{focused: -1, style: DefaultFocusRingStyle()}
}

// SetStyle overrides the focus ring style.
func (fm *FocusManager) SetStyle(s FocusRingStyle) { fm.style = s }

// Register appends components to the tab order.
// Order matters: Tab cycles through them in registration order.
func (fm *FocusManager) Register(comps ...Focusable) {
	fm.order = append(fm.order, comps...)
}

// Remove removes a component from the tab order.
// If it was focused, focus is cleared.
func (fm *FocusManager) Remove(comp Focusable) {
	for i, c := range fm.order {
		if c == comp {
			if fm.focused == i {
				fm.clearFocus()
			} else if fm.focused > i {
				fm.focused--
			}
			fm.order = append(fm.order[:i], fm.order[i+1:]...)
			return
		}
	}
}

// Focus explicitly focuses comp. Does nothing if comp is not registered or not focusable.
func (fm *FocusManager) Focus(comp Focusable) {
	for i, c := range fm.order {
		if c == comp && c.IsFocusable() {
			fm.setFocus(i)
			return
		}
	}
}

// FocusIndex focuses the component at position idx in the tab order.
func (fm *FocusManager) FocusIndex(idx int) {
	if idx < 0 || idx >= len(fm.order) {
		return
	}
	if fm.order[idx].IsFocusable() {
		fm.setFocus(idx)
	}
}

// FocusFirst focuses the first focusable component.
func (fm *FocusManager) FocusFirst() {
	for i, c := range fm.order {
		if c.IsFocusable() {
			fm.setFocus(i)
			return
		}
	}
}

// Focused returns the currently focused component (nil if none).
func (fm *FocusManager) Focused() Focusable {
	if fm.focused < 0 || fm.focused >= len(fm.order) {
		return nil
	}
	return fm.order[fm.focused]
}

// ClearFocus removes focus from all components.
func (fm *FocusManager) ClearFocus() { fm.clearFocus() }

// HandleEvent processes mouse clicks (to focus clicked component) and
// Tab/Shift-Tab/Escape for keyboard navigation.
// Returns true if the event was consumed.
func (fm *FocusManager) HandleEvent(e Event) bool {
	switch e.Type {
	case EventMouseDown:
		// Click-to-focus: find topmost registered component that was clicked.
		for i := len(fm.order) - 1; i >= 0; i-- {
			c := fm.order[i]
			if !c.IsFocusable() {
				continue
			}
			b := c.Bounds()
			if e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H {
				fm.setFocus(i)
				return false // let the click fall through to the component
			}
		}
		fm.clearFocus()
		return false

	case EventKeyDown:
		switch e.Key {
		case "Tab":
			if e.Shift {
				fm.focusPrev()
			} else {
				fm.focusNext()
			}
			return true
		case "Escape":
			fm.clearFocus()
			return true
		}
		if f := fm.Focused(); f != nil {
			return f.HandleKeyEvent(e)
		}
	}
	return false
}

// DrawFocusRing draws the focus ring around the currently focused component.
// Call this after all components have been drawn so the ring is on top.
func (fm *FocusManager) DrawFocusRing(c *canvas.Canvas) {
	f := fm.Focused()
	if f == nil {
		return
	}
	b := f.Bounds()
	s := fm.style
	x := b.X - s.Padding
	y := b.Y - s.Padding
	w := b.W + 2*s.Padding
	h := b.H + 2*s.Padding
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.StrokePaint(s.Color, s.Width))
}

// ── internal ──────────────────────────────────────────────────────────────────

func (fm *FocusManager) setFocus(idx int) {
	if fm.focused == idx {
		return
	}
	fm.clearFocus()
	fm.focused = idx
	comp := fm.order[idx]
	comp.FocusGained()
	if fm.OnFocusChange != nil {
		fm.OnFocusChange(comp)
	}
}

func (fm *FocusManager) clearFocus() {
	if fm.focused >= 0 && fm.focused < len(fm.order) {
		fm.order[fm.focused].FocusLost()
	}
	fm.focused = -1
	if fm.OnFocusChange != nil {
		fm.OnFocusChange(nil)
	}
}

func (fm *FocusManager) focusNext() {
	n := len(fm.order)
	if n == 0 {
		return
	}
	start := fm.focused + 1
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		if fm.order[idx].IsFocusable() {
			fm.setFocus(idx)
			return
		}
	}
}

func (fm *FocusManager) focusPrev() {
	n := len(fm.order)
	if n == 0 {
		return
	}
	start := fm.focused - 1
	if start < 0 {
		start = n - 1
	}
	for i := 0; i < n; i++ {
		idx := (start - i + n) % n
		if fm.order[idx].IsFocusable() {
			fm.setFocus(idx)
			return
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// BaseFocusable  —  embed helper
// ═══════════════════════════════════════════════════════════════════════════════

// BaseFocusable provides a no-op implementation of the Focusable interface.
// Embed it in your widget struct and override only the methods you need.
//
//	type MyButton struct {
//	    ui.BaseFocusable
//	    // ...
//	}
type BaseFocusable struct {
	focused   bool
	focusable bool
}

// InitFocusable sets the initial focusable state (default: true).
func (b *BaseFocusable) InitFocusable(focusable bool) { b.focusable = focusable }

func (b *BaseFocusable) FocusGained()              { b.focused = true }
func (b *BaseFocusable) FocusLost()                { b.focused = false }
func (b *BaseFocusable) IsFocused() bool           { return b.focused }
func (b *BaseFocusable) IsFocusable() bool         { return b.focusable }
func (b *BaseFocusable) HandleKeyEvent(_ Event) bool { return false }

```

## File: clipboard.go
Language: go | Tokens: 2159 | Size: 8638 bytes

**Imports:** sync

```go
// Package ui — clipboard.go
//
// Clipboard abstraction for cut/copy/paste.
//
//   ClipboardData  — union of supported clipboard formats
//   Clipboard      — interface the platform backend implements
//   InMemoryClipboard — in-process clipboard for testing / non-platform use
//   ClipboardManager  — app-level clipboard with history and undo support
//
// The real platform clipboard (OS clipboard) is wired up by the window backend.
// Pass it via SetClipboard.  Falls back to InMemoryClipboard if not set.
package ui

import "sync"

// ═══════════════════════════════════════════════════════════════════════════════
// ClipboardFormat
// ═══════════════════════════════════════════════════════════════════════════════

// ClipboardFormat is a MIME-like string identifying the data type.
type ClipboardFormat string

const (
	FormatText  ClipboardFormat = "text/plain"
	FormatHTML  ClipboardFormat = "text/html"
	FormatImage ClipboardFormat = "image/png"
	FormatFiles ClipboardFormat = "text/uri-list"
)

// ═══════════════════════════════════════════════════════════════════════════════
// ClipboardData
// ═══════════════════════════════════════════════════════════════════════════════

// ClipboardData holds one or more representations of clipboard content.
// A copy operation may provide multiple formats (e.g. both plain text and HTML).
type ClipboardData struct {
	entries map[ClipboardFormat][]byte
}

// NewClipboardData creates an empty ClipboardData.
func NewClipboardData() *ClipboardData {
	return &ClipboardData{entries: make(map[ClipboardFormat][]byte)}
}

// Set stores data for the given format.
func (d *ClipboardData) Set(format ClipboardFormat, data []byte) {
	d.entries[format] = data
}

// SetText is a convenience wrapper for plain-text content.
func (d *ClipboardData) SetText(s string) { d.Set(FormatText, []byte(s)) }

// Get retrieves data for the given format (nil if absent).
func (d *ClipboardData) Get(format ClipboardFormat) []byte { return d.entries[format] }

// Text returns the plain-text content, or "".
func (d *ClipboardData) Text() string {
	if b := d.Get(FormatText); b != nil {
		return string(b)
	}
	return ""
}

// HTML returns the HTML content, or "".
func (d *ClipboardData) HTML() string {
	if b := d.Get(FormatHTML); b != nil {
		return string(b)
	}
	return ""
}

// Has reports whether the given format is available.
func (d *ClipboardData) Has(format ClipboardFormat) bool {
	_, ok := d.entries[format]
	return ok
}

// Formats returns a list of available formats.
func (d *ClipboardData) Formats() []ClipboardFormat {
	out := make([]ClipboardFormat, 0, len(d.entries))
	for f := range d.entries {
		out = append(out, f)
	}
	return out
}

// Clone returns a deep copy of the ClipboardData.
func (d *ClipboardData) Clone() *ClipboardData {
	c := NewClipboardData()
	for f, b := range d.entries {
		cp := make([]byte, len(b))
		copy(cp, b)
		c.entries[f] = cp
	}
	return c
}

// ═══════════════════════════════════════════════════════════════════════════════
// Clipboard interface
// ═══════════════════════════════════════════════════════════════════════════════

// Clipboard is implemented by the platform window backend.
type Clipboard interface {
	// Write puts data onto the OS clipboard.
	Write(data *ClipboardData) error
	// Read retrieves data from the OS clipboard.
	// Returns nil if the clipboard is empty or unavailable.
	Read() (*ClipboardData, error)
	// Clear empties the OS clipboard.
	Clear() error
}

// ═══════════════════════════════════════════════════════════════════════════════
// InMemoryClipboard  —  non-platform implementation
// ═══════════════════════════════════════════════════════════════════════════════

// InMemoryClipboard stores clipboard data in memory.
// Useful for testing and environments where the OS clipboard is unavailable.
type InMemoryClipboard struct {
	mu   sync.RWMutex
	data *ClipboardData
}

// NewInMemoryClipboard creates an empty in-memory clipboard.
func NewInMemoryClipboard() *InMemoryClipboard { return &InMemoryClipboard{} }

func (c *InMemoryClipboard) Write(data *ClipboardData) error {
	c.mu.Lock()
	c.data = data.Clone()
	c.mu.Unlock()
	return nil
}

func (c *InMemoryClipboard) Read() (*ClipboardData, error) {
	c.mu.RLock()
	d := c.data
	c.mu.RUnlock()
	if d == nil {
		return nil, nil
	}
	return d.Clone(), nil
}

func (c *InMemoryClipboard) Clear() error {
	c.mu.Lock()
	c.data = nil
	c.mu.Unlock()
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ClipboardManager  —  app-level with history
// ═══════════════════════════════════════════════════════════════════════════════

// ClipboardManager wraps a Clipboard backend and adds a copy history ring-buffer.
type ClipboardManager struct {
	mu      sync.Mutex
	backend Clipboard
	history []*ClipboardData
	maxHist int

	// OnCopy is called after every successful copy.
	OnCopy func(*ClipboardData)
}

// NewClipboardManager creates a ClipboardManager wrapping backend.
// historySize sets how many past clipboard entries to retain (0 = no history).
func NewClipboardManager(backend Clipboard, historySize int) *ClipboardManager {
	if backend == nil {
		backend = NewInMemoryClipboard()
	}
	return &ClipboardManager{backend: backend, maxHist: historySize}
}

// Copy writes data to the clipboard and prepends it to history.
func (m *ClipboardManager) Copy(data *ClipboardData) error {
	if err := m.backend.Write(data); err != nil {
		return err
	}
	if m.maxHist > 0 {
		m.mu.Lock()
		m.history = append([]*ClipboardData{data.Clone()}, m.history...)
		if len(m.history) > m.maxHist {
			m.history = m.history[:m.maxHist]
		}
		m.mu.Unlock()
	}
	if m.OnCopy != nil {
		m.OnCopy(data)
	}
	return nil
}

// CopyText is a convenience wrapper for plain-text copies.
func (m *ClipboardManager) CopyText(s string) error {
	d := NewClipboardData()
	d.SetText(s)
	return m.Copy(d)
}

// Paste reads from the clipboard.
func (m *ClipboardManager) Paste() (*ClipboardData, error) {
	return m.backend.Read()
}

// PasteText reads plain text from the clipboard.
func (m *ClipboardManager) PasteText() (string, error) {
	d, err := m.Paste()
	if err != nil || d == nil {
		return "", err
	}
	return d.Text(), nil
}

// Clear empties the clipboard.
func (m *ClipboardManager) Clear() error { return m.backend.Clear() }

// History returns a copy of the clipboard history (newest first).
func (m *ClipboardManager) History() []*ClipboardData {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*ClipboardData, len(m.history))
	for i, d := range m.history {
		cp[i] = d.Clone()
	}
	return cp
}

// PasteFromHistory pastes the entry at position index (0 = most recent).
func (m *ClipboardManager) PasteFromHistory(index int) (*ClipboardData, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if index < 0 || index >= len(m.history) {
		return nil, false
	}
	return m.history[index].Clone(), true
}

// ClearHistory removes all history entries.
func (m *ClipboardManager) ClearHistory() {
	m.mu.Lock()
	m.history = m.history[:0]
	m.mu.Unlock()
}

```

## File: event_bus.go
Language: go | Tokens: 1536 | Size: 6147 bytes

```go
// Package ui — event_bus.go
//
// EventBus provides a decoupled, type-safe pub/sub message bus for
// application-wide communication between components.
//
// Every message is identified by a string topic.  Subscribers receive a
// typed payload via a Stream[T] or a plain callback.
//
// Usage
//
//	bus := ui.NewEventBus()
//
//	// Publish
//	ui.Publish[string](bus, "user.login", "alice")
//
//	// Subscribe
//	sub := ui.On[string](bus, "user.login", func(name string) {
//	    fmt.Println("logged in:", name)
//	})
//	defer sub.Unsubscribe()
//
//	// Subscribe once
//	ui.Once[string](bus, "app.ready", func(_ string) { boot() })
package ui

import (
	"sync"
)

// ═══════════════════════════════════════════════════════════════════════════════
// EventBus
// ═══════════════════════════════════════════════════════════════════════════════

// topicHub holds all subscribers for one topic (as Subject[any]).
type topicHub struct {
	subj Subject[any]
}

// EventBus is a thread-safe, topic-based event bus.
type EventBus struct {
	mu     sync.RWMutex
	topics map[string]*topicHub
}

// NewEventBus creates an empty EventBus.
func NewEventBus() *EventBus {
	return &EventBus{topics: make(map[string]*topicHub)}
}

func (b *EventBus) hub(topic string) *topicHub {
	b.mu.RLock()
	h, ok := b.topics[topic]
	b.mu.RUnlock()
	if ok {
		return h
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if h, ok = b.topics[topic]; !ok {
		h = &topicHub{}
		b.topics[topic] = h
	}
	return h
}

// RawPublish emits any value on the given topic.
func (b *EventBus) RawPublish(topic string, payload any) {
	b.hub(topic).subj.Push(payload)
}

// RawSubscribe registers a callback for any payload emitted on topic.
func (b *EventBus) RawSubscribe(topic string, fn func(any)) *Subscription {
	return b.hub(topic).subj.Subscribe(fn)
}

// Topics returns a snapshot of all topic names that have at least one subscriber.
func (b *EventBus) Topics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]string, 0, len(b.topics))
	for t, h := range b.topics {
		if h.subj.Len() > 0 {
			out = append(out, t)
		}
	}
	return out
}

// ═══════════════════════════════════════════════════════════════════════════════
// Typed helpers (package-level generics)
// ═══════════════════════════════════════════════════════════════════════════════

// Publish emits a typed payload on the given topic.
func Publish[T any](bus *EventBus, topic string, payload T) {
	bus.RawPublish(topic, any(payload))
}

// On subscribes fn to topic with type assertion to T.
// If a message cannot be asserted to T, it is silently dropped.
func On[T any](bus *EventBus, topic string, fn func(T)) *Subscription {
	return bus.RawSubscribe(topic, func(raw any) {
		if v, ok := raw.(T); ok {
			fn(v)
		}
	})
}

// Once subscribes fn but automatically unsubscribes after the first matching message.
func Once[T any](bus *EventBus, topic string, fn func(T)) *Subscription {
	var sub *Subscription
	sub = On[T](bus, topic, func(v T) {
		sub.Unsubscribe()
		fn(v)
	})
	return sub
}

// StreamOf returns a Stream[T] that emits typed payloads from the given topic.
func StreamOf[T any](bus *EventBus, topic string) *Stream[T] {
	out := &Stream[T]{}
	bus.RawSubscribe(topic, func(raw any) {
		if v, ok := raw.(T); ok {
			out.emit(v)
		}
	})
	return out
}

// ═══════════════════════════════════════════════════════════════════════════════
// Middleware
// ═══════════════════════════════════════════════════════════════════════════════

// MiddlewareFn can inspect or mutate a raw payload before it reaches subscribers.
// Returning nil drops the message.
type MiddlewareFn func(topic string, payload any) any

// MiddlewareBus wraps an EventBus and runs middleware on every Publish.
type MiddlewareBus struct {
	inner      *EventBus
	middleware []MiddlewareFn
}

// NewMiddlewareBus wraps an EventBus with a chain of middleware.
func NewMiddlewareBus(inner *EventBus, mw ...MiddlewareFn) *MiddlewareBus {
	return &MiddlewareBus{inner: inner, middleware: mw}
}

// Publish runs the payload through all middleware then publishes.
func (m *MiddlewareBus) Publish(topic string, payload any) {
	p := payload
	for _, mw := range m.middleware {
		p = mw(topic, p)
		if p == nil {
			return
		}
	}
	m.inner.RawPublish(topic, p)
}

// Use appends more middleware.
func (m *MiddlewareBus) Use(mw MiddlewareFn) { m.middleware = append(m.middleware, mw) }

// ═══════════════════════════════════════════════════════════════════════════════
// LoggingMiddleware  (dev helper)
// ═══════════════════════════════════════════════════════════════════════════════

// LoggingMiddleware returns a MiddlewareFn that prints each event to a writer.
// Import "fmt" or "log" in your application to wrap this.
func LoggingMiddleware(logFn func(string, any)) MiddlewareFn {
	return func(topic string, payload any) any {
		logFn(topic, payload)
		return payload
	}
}

```


---

**Summary:**
- Files Included: 12 / 12
- Total Tokens: 35730 / 100000
