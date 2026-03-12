package canvas

import "github.com/achiket123/gui-go/render"

// clipState tracks the currently active clip region.
type clipState struct {
	active bool
	x, y   float32
	w, h   float32
}

// applyClip calls the appropriate renderer method.
func applyClip(r render.Renderer, s clipState) {
	if s.active {
		r.SetClipRect(s.x, s.y, s.w, s.h)
	} else {
		r.ClearClip()
	}
}

// ClipRect sets a rectangular clip region.
// All subsequent draws are clipped to this rectangle until ResetClip or Restore().
func (c *Canvas) ClipRect(x, y, w, h float32) {
	c.clip = clipState{active: true, x: x, y: y, w: w, h: h}
	c.renderer.SetClipRect(x, y, w, h)
}

// ClipRoundedRect clips to a rounded rectangle.
// (Approximated as an axis-aligned rect in the renderer; full stencil support requires GL stencil buffer.)
// TODO: implement true shape clipping using GL stencil buffer.
func (c *Canvas) ClipRoundedRect(x, y, w, h, _ float32) {
	c.ClipRect(x, y, w, h)
}

// ClipCircle clips to a circle bounding box (approximation).
// TODO: implement true shape clipping using GL stencil buffer.
func (c *Canvas) ClipCircle(cx, cy, r float32) {
	c.ClipRect(cx-r, cy-r, r*2, r*2)
}

// ClipPath clips to the bounding box of a path (approximation).
// TODO: implement true shape clipping using GL stencil buffer.
func (c *Canvas) ClipPath(p *Path) {
	b := p.Bounds()
	c.ClipRect(b.X, b.Y, b.W, b.H)
}

// ResetClip removes any active clip region.
func (c *Canvas) ResetClip() {
	c.clip = clipState{}
	c.renderer.ClearClip()
}
