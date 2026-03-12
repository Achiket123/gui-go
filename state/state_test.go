package state

import (
	"testing"
)

func TestSignal(t *testing.T) {
	s := New(10)
	if s.Get() != 10 {
		t.Errorf("Expected 10, got %d", s.Get())
	}

	notified := 0
	s.Subscribe(func(v int) {
		notified++
	})

	s.Set(20)
	if s.Get() != 20 {
		t.Errorf("Expected 20, got %d", s.Get())
	}
	if notified != 1 {
		t.Errorf("Expected 1 notification, got %d", notified)
	}

	s.Set(20) // Should not notify
	if notified != 1 {
		t.Errorf("Expected 1 notification after redundant set, got %d", notified)
	}

	s.Update(func(v int) int { return v + 1 }) // 21
	if s.Get() != 21 {
		t.Errorf("Expected 21, got %d", s.Get())
	}
	if notified != 2 {
		t.Errorf("Expected 2 notifications, got %d", notified)
	}
}

func TestSignalAny(t *testing.T) {
	type NoCompare struct{ m map[string]int }
	s := NewSignalAny(NoCompare{m: make(map[string]int)})

	notified := 0
	s.Subscribe(func(v NoCompare) {
		notified++
	})

	s.Set(NoCompare{m: make(map[string]int)})
	if notified != 1 {
		t.Errorf("Expected 1 notification, got %d", notified)
	}

	s.Update(func(v NoCompare) NoCompare {
		v.m["a"] = 1
		return v
	})
	if notified != 2 {
		t.Errorf("Expected 2 notifications, got %d", notified)
	}
}

func TestStore(t *testing.T) {
	type MyState struct{ Count int }
	store := NewStore(MyState{Count: 0})

	notified := 0
	store.Subscribe(func(s MyState) {
		notified++
	})

	store.Mutate(func(s *MyState) {
		s.Count = 10
	})

	if store.Get().Count != 10 {
		t.Errorf("Expected 10, got %d", store.Get().Count)
	}
	if notified != 1 {
		t.Errorf("Expected 1 notification, got %d", notified)
	}
}
