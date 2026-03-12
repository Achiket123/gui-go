// Package state provides fine-grained reactive state for goui.
//
// Three primitives:
//
//	Signal[T]   — a mutable value that notifies subscribers on change
//	Computed[T] — a derived value recomputed lazily when dependencies change
//	Effect      — a side-effect that re-runs when any Signal it reads changes
//
// Usage:
//
//	count  := state.New(0)
//	double := state.Derive(func() int { return count.Get() * 2 })
//	state.Watch(func() {
//	    fmt.Println("count =", count.Get(), "double =", double.Get())
//	})
//	count.Set(5) // prints: count = 5  double = 10
//
// Integration with goui:
//
//	score := state.New(0)
//	score.OnChange(func(v int) { dirty.InvalidateAll() })
package state

import (
	"sync"
	"sync/atomic"
)

// ─────────────────────────────────────────────────────────────────────────────
// internal subscriber list
// ─────────────────────────────────────────────────────────────────────────────

type subscriber struct {
	id uint64
	fn func()
}

var subIDCounter uint64

// nextSubID returns a globally unique, atomically incremented subscriber ID.
func nextSubID() uint64 { return atomic.AddUint64(&subIDCounter, 1) }

type subList struct {
	mu   sync.Mutex
	subs []subscriber
}

func (s *subList) add(fn func()) uint64 {
	id := nextSubID()
	s.mu.Lock()
	s.subs = append(s.subs, subscriber{id, fn})
	s.mu.Unlock()
	return id
}

func (s *subList) remove(id uint64) {
	s.mu.Lock()
	for i, sub := range s.subs {
		if sub.id == id {
			s.subs = append(s.subs[:i], s.subs[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
}

func (s *subList) notify() {
	s.mu.Lock()
	cbs := make([]func(), len(s.subs))
	for i, sub := range s.subs {
		cbs[i] = sub.fn
	}
	s.mu.Unlock()
	for _, fn := range cbs {
		fn()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Reactive primitives (Signal, SignalAny)
// ─────────────────────────────────────────────────────────────────────────────

// baseSignal provides the common storage and subscriber logic for all Signals.
type baseSignal[T any] struct {
	mu    sync.RWMutex
	value T
	subs  subList
}

func (s *baseSignal[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *baseSignal[T]) Subscribe(fn func(T)) func() {
	id := s.subs.add(func() { fn(s.Get()) })
	return func() { s.subs.remove(id) }
}

func (s *baseSignal[T]) notifyOnChange(fn func()) func() {
	id := s.subs.add(fn)
	return func() { s.subs.remove(id) }
}

// Signal is a generic reactive value container for comparable types.
// It only notifies subscribers if the value actually changes (equality check).
type Signal[T comparable] struct {
	baseSignal[T]
}

// New creates a new Signal with an initial value.
func New[T comparable](initial T) *Signal[T] {
	return &Signal[T]{baseSignal[T]{value: initial}}
}

// Set updates the value and notifies subscribers if the value changed.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	old := s.value
	s.value = v
	s.mu.Unlock()
	if old != v {
		s.subs.notify()
	}
}

// Update applies a transform function atomically and notifies on change.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	old := s.value
	nv := fn(old)
	s.value = nv
	s.mu.Unlock()
	// Note: Comparing 'old' and 'nv' after unlocking is safe because they are
	// local copies of the state captured during the locked transform.
	if old != nv {
		s.subs.notify()
	}
}

// OnChange is an alias for Subscribe for ergonomics.
func (s *Signal[T]) OnChange(fn func(T)) func() { return s.Subscribe(fn) }

// SignalAny is a reactive value container for any type (including non-comparable).
// Unlike Signal, it DOES NOT perform an equality check on Set/Update; it always notifies.
type SignalAny[T any] struct {
	baseSignal[T]
}

// NewSignalAny creates a new SignalAny with an initial value.
func NewSignalAny[T any](initial T) *SignalAny[T] {
	return &SignalAny[T]{baseSignal[T]{value: initial}}
}

// Set updates the value and always notifies subscribers.
func (s *SignalAny[T]) Set(v T) {
	s.mu.Lock()
	s.value = v
	s.mu.Unlock()
	s.subs.notify()
}

// Update applies a transform function atomically and always notifies.
func (s *SignalAny[T]) Update(fn func(T) T) {
	s.mu.Lock()
	s.value = fn(s.value)
	s.mu.Unlock()
	s.subs.notify()
}

// OnChange is an alias for Subscribe for ergonomics.
func (s *SignalAny[T]) OnChange(fn func(T)) func() { return s.Subscribe(fn) }

// ─────────────────────────────────────────────────────────────────────────────
// Computed[T]  (derived / memoised value)
// ─────────────────────────────────────────────────────────────────────────────

// Computed holds a lazily-evaluated derived value.
// It recomputes when any dependency Signal it was created with changes.
type Computed[T any] struct {
	mu    sync.Mutex
	value T
	dirty bool
	fn    func() T
	subs  subList
}

// Derive creates a Computed value from a pure function.
// Pass every Signal that fn reads as deps so the computed invalidates correctly.
//
// Example:
//
//	fullName := state.Derive(func() string {
//	    return first.Get() + " " + last.Get()
//	}, first, last)
func Derive[T any, D interface{ notifyOnChange(func()) func() }](fn func() T, deps ...D) *Computed[T] {
	c := &Computed[T]{fn: fn, dirty: true}
	for _, dep := range deps {
		dep.notifyOnChange(func() {
			c.mu.Lock()
			c.dirty = true
			c.mu.Unlock()
			c.subs.notify()
		})
	}
	return c
}

// Get returns the current computed value, recomputing if dirty.
func (c *Computed[T]) Get() T {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.dirty {
		c.value = c.fn()
		c.dirty = false
	}
	return c.value
}

// Subscribe registers a callback for when the computed invalidates.
func (c *Computed[T]) Subscribe(fn func(T)) func() {
	id := c.subs.add(func() { fn(c.Get()) })
	return func() { c.subs.remove(id) }
}

// notifyOnChange satisfies the internal dep interface.
func (s *Signal[T]) notifyOnChange(fn func()) func() {
	id := s.subs.add(fn)
	return func() { s.subs.remove(id) }
}

// ─────────────────────────────────────────────────────────────────────────────
// Effect
// ─────────────────────────────────────────────────────────────────────────────

// Effect runs fn immediately and again whenever any provided Signal changes.
// Returns a stop function that cancels future runs.
//
//	stop := state.Watch(func() {
//	    title.Set("Count: " + strconv.Itoa(count.Get()))
//	}, count)
func Watch[D interface{ notifyOnChange(func()) func() }](fn func(), deps ...D) func() {
	fn() // run immediately
	stops := make([]func(), len(deps))
	for i, dep := range deps {
		stops[i] = dep.notifyOnChange(fn)
	}
	return func() {
		for _, stop := range stops {
			stop()
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Store — a struct-based state container (like Pinia / Zustand)
// ─────────────────────────────────────────────────────────────────────────────

// Store[S] wraps an arbitrary state struct S behind a mutex with a
// subscriber list, emitting the full new state on every mutation.
//
//	type AppState struct { Count int; Name string }
//	store := state.NewStore(AppState{Name: "hello"})
//	store.Mutate(func(s *AppState) { s.Count++ })
//	store.Subscribe(func(s AppState) { fmt.Println(s) })
type Store[S any] struct {
	mu    sync.RWMutex
	state S
	subs  subList
}

func NewStore[S any](initial S) *Store[S] {
	return &Store[S]{state: initial}
}

// Get returns a copy of the current state.
func (s *Store[S]) Get() S {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Mutate applies fn to the state (under lock) then notifies subscribers.
func (s *Store[S]) Mutate(fn func(*S)) {
	s.mu.Lock()
	fn(&s.state)
	s.mu.Unlock()
	s.subs.notify()
}

// Subscribe registers a callback with the full state whenever it changes.
func (s *Store[S]) Subscribe(fn func(S)) func() {
	id := s.subs.add(func() { fn(s.Get()) })
	return func() { s.subs.remove(id) }
}

// ─────────────────────────────────────────────────────────────────────────────
// History[T] — undo / redo stack built on top of Signal
// ─────────────────────────────────────────────────────────────────────────────

// History wraps a Signal and provides Undo/Redo.
type History[T comparable] struct {
	sig    *Signal[T]
	past   []T
	future []T
	mu     sync.Mutex
	max    int // max history depth (0 = unlimited)
}

// NewHistory creates a Signal with undo/redo support.
func NewHistory[T comparable](initial T, maxDepth int) *History[T] {
	return &History[T]{sig: New(initial), max: maxDepth}
}

// Signal returns the underlying Signal (for subscriptions, binding).
func (h *History[T]) Signal() *Signal[T] { return h.sig }

// Get returns the current value.
func (h *History[T]) Get() T { return h.sig.Get() }

// Push sets a new value and records the previous one for undo.
func (h *History[T]) Push(v T) {
	h.mu.Lock()
	old := h.sig.Get()
	h.past = append(h.past, old)
	if h.max > 0 && len(h.past) > h.max {
		h.past = h.past[1:]
	}
	h.future = h.future[:0] // clear redo stack
	h.mu.Unlock()
	h.sig.Set(v)
}

// Undo reverts to the previous value. Returns false if nothing to undo.
func (h *History[T]) Undo() bool {
	h.mu.Lock()
	if len(h.past) == 0 {
		h.mu.Unlock()
		return false
	}
	prev := h.past[len(h.past)-1]
	h.past = h.past[:len(h.past)-1]
	h.future = append(h.future, h.sig.Get())
	h.mu.Unlock()
	h.sig.Set(prev)
	return true
}

// Redo re-applies the last undone value. Returns false if nothing to redo.
func (h *History[T]) Redo() bool {
	h.mu.Lock()
	if len(h.future) == 0 {
		h.mu.Unlock()
		return false
	}
	next := h.future[len(h.future)-1]
	h.future = h.future[:len(h.future)-1]
	h.past = append(h.past, h.sig.Get())
	h.mu.Unlock()
	h.sig.Set(next)
	return true
}

// ── EventBus ─────────────────────────────────────────────────────────────────

type topicHub struct {
	mu   sync.RWMutex
	subs map[uint64]func(any)
}

type EventBus struct {
	mu     sync.RWMutex
	topics map[string]*topicHub
}

func NewEventBus() *EventBus {
	return &EventBus{topics: make(map[string]*topicHub)}
}

func (b *EventBus) hub(topic string) *topicHub {
	b.mu.Lock()
	defer b.mu.Unlock()
	if h, ok := b.topics[topic]; ok {
		return h
	}
	h := &topicHub{subs: make(map[uint64]func(any))}
	b.topics[topic] = h
	return h
}

func (b *EventBus) RawPublish(topic string, payload any) {
	h := b.hub(topic)
	h.mu.RLock()
	cbs := make([]func(any), 0, len(h.subs))
	for _, fn := range h.subs {
		cbs = append(cbs, fn)
	}
	h.mu.RUnlock()
	for _, cb := range cbs {
		cb(payload)
	}
}

type Subscription struct {
	Cancel func()
}

func (b *EventBus) RawSubscribe(topic string, fn func(any)) *Subscription {
	h := b.hub(topic)
	id := nextSubID()
	h.mu.Lock()
	h.subs[id] = fn
	h.mu.Unlock()
	return &Subscription{Cancel: func() {
		h.mu.Lock()
		delete(h.subs, id)
		h.mu.Unlock()
	}}
}

func Publish[T any](bus *EventBus, topic string, payload T) {
	bus.RawPublish(topic, payload)
}

func On[T any](bus *EventBus, topic string, fn func(T)) *Subscription {
	return bus.RawSubscribe(topic, func(val any) {
		if v, ok := val.(T); ok {
			fn(v)
		}
	})
}
