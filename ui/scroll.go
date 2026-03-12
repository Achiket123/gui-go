package ui

import (
	"math"

	"github.com/achiket123/gui-go/canvas"
)

const (
	ScrollbarWidth  = 10
	scrollMomentumK = 4.0    // lower = longer-lasting momentum
	scrollSpeed     = 500.0  // pixels/s of velocity per scroll notch
	scrollMaxV      = 3000.0 // px/s velocity cap
)

// ScrollView is a retained component that clips and scrolls any content
// taller than its bounds. Assign ContentH and DrawContent, then call
// Draw each frame.
type ScrollView struct {
	// ContentH is the total pixel height of the scrollable content.
	ContentH float32
	// DrawContent is called with a canvas translated so y=0 is the top of
	// content. x, y, w, h describe the unclipped content rect.
	DrawContent func(c *canvas.Canvas, x, y, w, h float32)

	// Style controls scrollbar appearance.
	Style ScrollStyle

	bounds       canvas.Rect
	scrollY      float32
	velocity     float32
	barDragging  bool
	barDragStart float32
	scrollAtDrag float32
	barHover     bool
}

// ScrollStyle describes the scrollbar appearance.
type ScrollStyle struct {
	TrackColor canvas.Color
	ThumbColor canvas.Color
	ThumbHover canvas.Color
	Width      float32
}

// DefaultScrollStyle returns a sensible translucent dark-theme scrollbar.
func DefaultScrollStyle() ScrollStyle {
	return ScrollStyle{
		TrackColor: canvas.RGBA8(255, 255, 255, 25),
		ThumbColor: canvas.RGBA8(255, 255, 255, 80),
		ThumbHover: canvas.Hex("#89B4FA"),
		Width:      10,
	}
}

// ScrollOffset returns the current scroll position in pixels.
func (s *ScrollView) ScrollOffset() float32 { return s.scrollY }

// NewScrollView creates a ScrollView with default style.
func NewScrollView(contentH float32, draw func(c *canvas.Canvas, x, y, w, h float32)) *ScrollView {
	return &ScrollView{
		ContentH:    contentH,
		DrawContent: draw,
		Style:       DefaultScrollStyle(),
	}
}

func (s *ScrollView) MaxScroll() float32 {
	if ex := s.ContentH - s.bounds.H; ex > 0 {
		return ex
	}
	return 0
}

// ScrollTo jumps to a specific Y offset (clamped).
func (s *ScrollView) ScrollTo(y float32) {
	s.scrollY = y
	s.velocity = 0
	s.clamp()
}

// ScrollFraction returns the current scroll position as 0.0–1.0.
func (s *ScrollView) ScrollFraction() float32 {
	if s.MaxScroll() == 0 {
		return 0
	}
	return s.scrollY / s.MaxScroll()
}

func (s *ScrollView) clamp() {
	if s.scrollY < 0 {
		s.scrollY = 0
	}
	if m := s.MaxScroll(); s.scrollY > m {
		s.scrollY = m
	}
}

func (s *ScrollView) thumbHeight() float32 {
	if s.ContentH <= 0 {
		return s.bounds.H
	}
	th := s.bounds.H * (s.bounds.H / s.ContentH)
	if th < 24 {
		th = 24
	}
	if th > s.bounds.H {
		th = s.bounds.H
	}
	return th
}

func (s *ScrollView) Bounds() canvas.Rect { return s.bounds }

func (s *ScrollView) Tick(delta float64) {
	if math.Abs(float64(s.velocity)) > 0.5 {
		s.scrollY += s.velocity * float32(delta)
		s.clamp()
		s.velocity *= float32(math.Exp(-scrollMomentumK * delta))
	} else {
		s.velocity = 0
	}
}

func (s *ScrollView) HandleEvent(e Event) bool {
	inBounds := e.X >= s.bounds.X && e.X <= s.bounds.X+s.bounds.W &&
		e.Y >= s.bounds.Y && e.Y <= s.bounds.Y+s.bounds.H

	barW := s.Style.Width
	barX := s.bounds.X + s.bounds.W - barW - 4

	switch e.Type {
	case EventScroll:
		if inBounds {
			s.velocity += e.ScrollY * scrollSpeed
			if s.velocity > scrollMaxV {
				s.velocity = scrollMaxV
			} else if s.velocity < -scrollMaxV {
				s.velocity = -scrollMaxV
			}
			return true
		}

	case EventMouseDown:
		if inBounds && e.X >= barX {
			s.barDragging = true
			s.barDragStart = e.Y
			s.scrollAtDrag = s.scrollY
			return true
		}

	case EventMouseMove:
		s.barHover = inBounds && e.X >= barX
		if s.barDragging {
			dy := e.Y - s.barDragStart
			trackH := s.bounds.H - 8
			thumbH := s.thumbHeight()
			if avail := trackH - thumbH; avail > 0 {
				s.scrollY = s.scrollAtDrag + dy*(s.MaxScroll()/avail)
				s.clamp()
			}
			return true
		}

	case EventMouseUp:
		if s.barDragging {
			s.barDragging = false
			return true
		}

	case EventKeyDown:
		if !inBounds {
			return false
		}
		switch e.Key {
		case "Down":
			s.velocity = scrollSpeed
		case "Up":
			s.velocity = -scrollSpeed
		case "Next": // Page Down
			s.scrollY += s.bounds.H * 0.8
			s.clamp()
		case "Prior": // Page Up
			s.scrollY -= s.bounds.H * 0.8
			s.clamp()
		case "Home":
			s.scrollY, s.velocity = 0, 0
		case "End":
			s.scrollY, s.velocity = s.MaxScroll(), 0
		default:
			return false
		}
		return true
	}
	return false
}

func (s *ScrollView) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	s.clamp()

	barW := s.Style.Width
	contentW := w - barW - 8

	c.Save()
	c.ClipRect(x, y, contentW, h)
	if s.DrawContent != nil {
		s.DrawContent(c, x, y-s.scrollY, contentW, s.ContentH)
	}
	c.Restore()

	if s.MaxScroll() <= 0 {
		return
	}

	bx := x + contentW + 4
	trackH := h - 8

	c.DrawRoundedRect(bx, y+4, barW, trackH, barW/2,
		canvas.FillPaint(s.Style.TrackColor))

	thumbH := s.thumbHeight()
	thumbY := y + 4
	if s.MaxScroll() > 0 {
		thumbY += (s.scrollY / s.MaxScroll()) * (trackH - thumbH)
	}
	tc := s.Style.ThumbColor
	if s.barHover || s.barDragging {
		tc = s.Style.ThumbHover
	}
	c.DrawRoundedRect(bx, thumbY, barW, thumbH, barW/2,
		canvas.FillPaint(tc))
}
