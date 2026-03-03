package animation

import (
	"testing"
	"time"
)

func BenchmarkControllerTick(b *testing.B) {
	ctrl := NewController(time.Second)
	ctrl.Repeat(-1) // infinite loop to avoid idle benchmarking
	ctrl.Forward()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrl.Tick(0.016) // simulate 60fps delta
	}
}

func BenchmarkTimelineWithTracks(b *testing.B) {
	tl := NewTimeline(time.Second)
	tl.AddTrack("x", 0, 500, 0.0, 1.0, EaseOutBack)
	tl.AddTrack("y", 0, 300, 0.0, 0.8, EaseOutQuad)
	tl.AddTrack("opacity", 0, 1.0, 0.0, 0.5, Linear)
	tl.Play()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tl.Tick(0.016)
		if !tl.IsPlaying() {
			tl.Seek(0)
			tl.Play()
		}
	}
}

func BenchmarkTweenMapping(b *testing.B) {
	tween := NewTween(0, 100, 0, EaseInOutCubic)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// simulate driving a value
		_ = tween.Map(0.5)
	}
}

func BenchmarkSequenceTick(b *testing.B) {
	s := NewSequence()
	s.Add(NewTween(0, 1, 500*time.Millisecond, Linear))
	s.Add(NewTween(1, 0, 500*time.Millisecond, Linear))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Tick(0.016)
		if s.IsFinished() {
			s.Reset()
		}
	}
}
