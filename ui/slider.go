package ui

import (
	"math"

	"github.com/achiket/gui-go/canvas"
)

// SliderStyle describes the visual appearance of a Slider.
type SliderStyle struct {
	TrackColor  canvas.Color
	FillColor   canvas.Color
	ThumbColor  canvas.Color
	TrackHeight float32
	ThumbRadius float32
}

// DefaultSliderStyle returns a dark-theme slider style.
func DefaultSliderStyle() SliderStyle {
	return SliderStyle{
		TrackColor:  canvas.RGBA8(255, 255, 255, 50),
		FillColor:   canvas.Hex("#3B82F6"),
		ThumbColor:  canvas.White,
		TrackHeight: 6,
		ThumbRadius: 10,
	}
}

// Slider is a horizontal draggable value selector.
type Slider struct {
	Value    *float64 // pointer so caller can bind a variable
	Min, Max float64
	Step     float64
	OnChange func(float64)
	Style    SliderStyle

	bounds   canvas.Rect
	dragging bool
}

// NewSlider creates a Slider bound to value in [min, max].
func NewSlider(value *float64, min, max float64, onChange func(float64)) *Slider {
	return &Slider{
		Value:    value,
		Min:      min,
		Max:      max,
		OnChange: onChange,
		Style:    DefaultSliderStyle(),
	}
}

func (s *Slider) Bounds() canvas.Rect { return s.bounds }
func (s *Slider) Tick(_ float64)      {}
func (s *Slider) HandleEvent(e Event) bool {
	inBounds := e.X >= s.bounds.X && e.X <= s.bounds.X+s.bounds.W &&
		e.Y >= s.bounds.Y && e.Y <= s.bounds.Y+s.bounds.H

	switch e.Type {
	case EventMouseDown:
		if inBounds {
			s.dragging = true
			s.setFromX(e.X)
			return true
		}
	case EventMouseMove:
		if s.dragging {
			s.setFromX(e.X)
			return true
		}
	case EventMouseUp:
		if s.dragging {
			s.dragging = false
			return true
		}
	}
	return false
}

func (s *Slider) setFromX(x float32) {
	if s.Value == nil {
		return
	}
	t := float64((x - s.bounds.X) / s.bounds.W)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	v := s.Min + (s.Max-s.Min)*t
	if s.Step > 0 {
		v = math.Round(v/s.Step) * s.Step
	}
	*s.Value = v
	if s.OnChange != nil {
		s.OnChange(v)
	}
}

func (s *Slider) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	cy := y + h/2

	// Track background.
	c.DrawRoundedRect(x, cy-s.Style.TrackHeight/2, w, s.Style.TrackHeight, s.Style.TrackHeight/2,
		canvas.FillPaint(s.Style.TrackColor))

	// Filled portion.
	if s.Value != nil && s.Max > s.Min {
		t := float32((*s.Value - s.Min) / (s.Max - s.Min))
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		fw := w * t
		c.DrawRoundedRect(x, cy-s.Style.TrackHeight/2, fw, s.Style.TrackHeight, s.Style.TrackHeight/2,
			canvas.FillPaint(s.Style.FillColor))

		// Thumb.
		tx := x + fw
		c.DrawCircle(tx, cy, s.Style.ThumbRadius, canvas.FillPaint(s.Style.ThumbColor))
	}
}
