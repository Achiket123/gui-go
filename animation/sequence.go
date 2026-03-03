package animation

import "time"

// step is one element in a Sequence — either a Tween or a delay.
type step struct {
	tween    *Tween
	delay    float64 // seconds (used when tween == nil)
	delayAcc float64 // accumulated seconds for this delay
}

// Sequence chains multiple Tweens and delays one after another.
//
// Example:
//
//	seq := animation.NewSequence()
//	seq.Add(animation.NewTween(0, 400, time.Second, animation.EaseOutQuad))
//	seq.AddDelay(500 * time.Millisecond)
//	seq.Add(animation.NewTween(400, 0, time.Second, animation.EaseInQuad))
//	seq.SetLoop(true)
//	w.AddAnimation(seq)
type Sequence struct {
	steps    []step
	current  int
	looping  bool
	finished bool

	onStepComplete func(stepIndex int)
	onComplete     func()
}

// NewSequence creates an empty Sequence.
func NewSequence() *Sequence {
	return &Sequence{}
}

// Add appends a Tween step.
func (s *Sequence) Add(t *Tween) *Sequence {
	s.steps = append(s.steps, step{tween: t})
	return s
}

// AddDelay appends a pause of the given duration.
func (s *Sequence) AddDelay(d time.Duration) *Sequence {
	s.steps = append(s.steps, step{delay: d.Seconds()})
	return s
}

// SetLoop enables infinite looping of the whole sequence.
func (s *Sequence) SetLoop(loop bool) {
	s.looping = loop
}

// OnStepComplete registers a callback called when each step finishes.
func (s *Sequence) OnStepComplete(fn func(stepIndex int)) {
	s.onStepComplete = fn
}

// OnComplete registers a callback called when the full sequence finishes.
func (s *Sequence) OnComplete(fn func()) {
	s.onComplete = fn
}

// Reset restarts the sequence from the first step.
func (s *Sequence) Reset() {
	s.current = 0
	s.finished = false
	for i := range s.steps {
		if s.steps[i].tween != nil {
			s.steps[i].tween.Reset()
		}
		s.steps[i].delayAcc = 0
	}
}

// IsFinished returns true when a non-looping sequence has completed.
func (s *Sequence) IsFinished() bool {
	return s.finished
}

// Tick advances the current step by delta seconds.
func (s *Sequence) Tick(delta float64) {
	if s.finished || len(s.steps) == 0 {
		return
	}

	st := &s.steps[s.current]
	if st.tween != nil {
		// Tween step
		st.tween.Tick(delta)
		if st.tween.IsFinished() {
			s.advanceStep()
		}
	} else {
		// Delay step
		st.delayAcc += delta
		if st.delayAcc >= st.delay {
			s.advanceStep()
		}
	}
}

func (s *Sequence) advanceStep() {
	idx := s.current
	if s.onStepComplete != nil {
		s.onStepComplete(idx)
	}
	s.current++
	if s.current >= len(s.steps) {
		if s.looping {
			s.Reset()
		} else {
			s.finished = true
			if s.onComplete != nil {
				s.onComplete()
			}
		}
	}
}
