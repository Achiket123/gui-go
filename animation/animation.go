// Package animation provides Tween, Sequence, and Sprite animation types
// for use with the goui render loop.
package animation

// Animatable is implemented by all animation types in this package.
// The render loop calls Tick each frame and discards finished non-loopers.
type Animatable interface {
	// Tick advances the animation by delta seconds (time since last frame).
	Tick(delta float64)
	// IsFinished returns true when a non-looping animation has fully played.
	IsFinished() bool
}
