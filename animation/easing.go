package animation

import "math"

// EasingFn is the signature for all easing functions.
// t is a normalized time in [0.0, 1.0]; the return value is the eased progress.
type EasingFn func(t float64) float64

// --- Linear ---

// Linear applies no curve — constant rate throughout.
func Linear(t float64) float64 { return t }

// --- Quad ---

// EaseInQuad accelerates from zero (t²).
func EaseInQuad(t float64) float64 { return t * t }

// EaseOutQuad decelerates to zero.
func EaseOutQuad(t float64) float64 { return t * (2 - t) }

// EaseInOutQuad accelerates then decelerates.
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

// --- Cubic ---

// EaseInCubic accelerates from zero (t³).
func EaseInCubic(t float64) float64 { return t * t * t }

// EaseOutCubic decelerates to zero.
func EaseOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

// EaseInOutCubic smooth S-curve.
func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t = 2*t - 2
	return 0.5*t*t*t + 1
}

// --- Elastic ---

const elasticC4 = (2 * math.Pi) / 3
const elasticC5 = (2 * math.Pi) / 4.5

// EaseInElastic springs back at the start.
func EaseInElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return -math.Pow(2, 10*t-10) * math.Sin((t*10-10.75)*elasticC4)
}

// EaseOutElastic bouncy spring at the end.
func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t*10-0.75)*elasticC4) + 1
}

// EaseInOutElastic springs both ends.
func EaseInOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	if t < 0.5 {
		return -(math.Pow(2, 20*t-10) * math.Sin((20*t-11.125)*elasticC5)) / 2
	}
	return (math.Pow(2, -20*t+10)*math.Sin((20*t-11.125)*elasticC5))/2 + 1
}

// --- Bounce ---

// EaseOutBounce bounces at the end.
func EaseOutBounce(t float64) float64 {
	const n1 = 7.5625
	const d1 = 2.75
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	default:
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}

// EaseInBounce bounces at the start.
func EaseInBounce(t float64) float64 { return 1 - EaseOutBounce(1-t) }

// --- Back ---

const backC1 = 1.70158
const backC2 = backC1 * 1.525
const backC3 = backC1 + 1

// EaseInBack overshoots backward before starting.
func EaseInBack(t float64) float64 {
	return backC3*t*t*t - backC1*t*t
}

// EaseOutBack overshoots forward at the end.
func EaseOutBack(t float64) float64 {
	t--
	return 1 + backC3*t*t*t + backC1*t*t
}

// EaseInOutBack overshoots both ends.
func EaseInOutBack(t float64) float64 {
	if t < 0.5 {
		return (math.Pow(2*t, 2) * ((backC2+1)*2*t - backC2)) / 2
	}
	t = 2*t - 2
	return (math.Pow(t, 2)*((backC2+1)*t+backC2) + 2) / 2
}
