// Package canvas provides the 2D drawing API for goui v2.
// Every frame, your OnDraw callback receives a *Canvas backed by a Renderer
// (GL2DRenderer or SWRenderer). All drawing is batched and flushed at EndFrame.
//
// Example:
//
//	w.OnDraw(func(c *canvas.Canvas) {
//	    c.Clear(canvas.RGB8(20, 20, 30))
//	    c.DrawRoundedRect(50, 50, 200, 100, 12, canvas.FillPaint(canvas.Blue))
//	    c.DrawText(60, 95, "Hello GPU!", canvas.DefaultTextStyle())
//	})
package canvas

import (
	"math"
	"strings"
	"sync"

	"github.com/achiket/gui-go/render"
	"github.com/achiket/gui-go/render/gl"
)

// fontKey uniquely identifies a loaded font.
type fontKey struct {
	path string
	size float32
}

var (
	fontCacheMu sync.Mutex
	fontCache   = map[fontKey]*gl.FontAtlas{}
)

func loadFont(path string, size float32) *gl.FontAtlas {
	k := fontKey{path, size}
	fontCacheMu.Lock()
	defer fontCacheMu.Unlock()
	if f, ok := fontCache[k]; ok {
		return f
	}
	var f *gl.FontAtlas
	var err error
	if path == "" {
		f, err = gl.LoadSystemFont("DejaVuSans", size)
		if err != nil {
			f, _ = gl.LoadSystemFont("liberation", size)
		}
	} else {
		f, err = gl.LoadFont(path, size)
	}
	if err == nil && f != nil {
		fontCache[k] = f
	}
	return f
}

// Canvas is the 2D drawing surface for one frame.
// Obtain one from the Window's OnDraw callback; never construct directly.
type Canvas struct {
	renderer render.Renderer
	ts       *TransformStack
	clip     clipState

	w, h float32
}

// NewCanvas creates a Canvas backed by r, with pixel dimensions w×h.
func NewCanvas(r render.Renderer, w, h int) *Canvas {
	return &Canvas{
		renderer: r,
		ts:       NewTransformStack(),
		w:        float32(w),
		h:        float32(h),
	}
}

// --- Transform ---

// Save pushes the current transform and clip state onto the stack.
func (c *Canvas) Save() {
	c.ts.Save(c.clip.active, c.clip.x, c.clip.y, c.clip.w, c.clip.h)
}

// Restore pops the last saved transform and clip state.
func (c *Canvas) Restore() {
	clipActive, cx, cy, cw, ch := c.ts.Restore()
	c.renderer.PushTransform(c.ts.Current())
	c.clip = clipState{active: clipActive, x: cx, y: cy, w: cw, h: ch}
	applyClip(c.renderer, c.clip)
}

func (c *Canvas) Translate(x, y float32) {
	c.ts.Translate(x, y)
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) Rotate(angle float32) {
	c.ts.Rotate(angle)
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) RotateAround(cx, cy, angle float32) {
	c.ts.RotateAround(cx, cy, angle)
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) Scale(sx, sy float32) {
	c.ts.Scale(sx, sy)
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) ScaleUniform(s float32) {
	c.ts.ScaleUniform(s)
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) ResetTransform() {
	c.ts.ResetTransform()
	c.renderer.PushTransform(c.ts.Current())
}
func (c *Canvas) CurrentTransform() Mat3 { return c.ts.Current() }

func (c *Canvas) SetGlobalOpacity(opacity float32) {
	c.renderer.SetGlobalOpacity(opacity)
}

// --- Shapes ---

// Clear fills the entire canvas with a solid color.
func (c *Canvas) Clear(col Color) {
	c.renderer.DrawFilledRect(0, 0, c.w, c.h, 0, col.ToArray(), 1)
}

// DrawRect draws a filled or stroked axis-aligned rectangle.
func (c *Canvas) DrawRect(x, y, w, h float32, p Paint) {
	if p.Fill || p.StrokeWidth == 0 {
		if p.LinearGrad != nil {
			c0 := p.LinearGrad.Stops[0].Color.ToArray()
			c1 := p.LinearGrad.Stops[len(p.LinearGrad.Stops)-1].Color.ToArray()
			p1 := [2]float32{p.LinearGrad.From.X, p.LinearGrad.From.Y}
			p2 := [2]float32{p.LinearGrad.To.X, p.LinearGrad.To.Y}
			c.renderer.DrawGradientRect(x, y, w, h, c0, c1, p1, p2, p.alpha())
		} else {
			c.renderer.DrawFilledRect(x, y, w, h, 0, p.Color.ToArray(), p.alpha())
		}
	} else {
		c.renderer.DrawStrokedRect(x, y, w, h, 0, p.StrokeWidth, p.Color.ToArray(), p.alpha())
	}
}

// DrawRoundedRect draws a rectangle with rounded corners.
func (c *Canvas) DrawRoundedRect(x, y, w, h, radius float32, p Paint) {
	if p.Fill || p.StrokeWidth == 0 {
		c.renderer.DrawFilledRect(x, y, w, h, radius, p.Color.ToArray(), p.alpha())
	} else {
		c.renderer.DrawStrokedRect(x, y, w, h, radius, p.StrokeWidth, p.Color.ToArray(), p.alpha())
	}
}

// DrawCircle draws a filled or stroked circle centered at (cx, cy).
func (c *Canvas) DrawCircle(cx, cy, radius float32, p Paint) {
	if p.Fill || p.StrokeWidth == 0 {
		c.renderer.DrawFilledCircle(cx, cy, radius, p.Color.ToArray(), p.alpha())
	} else {
		// Stroke approximated with an outer-inner circle pair.
		sw := p.StrokeWidth
		c.renderer.DrawFilledCircle(cx, cy, radius+sw/2, p.Color.ToArray(), p.alpha())
		c.renderer.DrawFilledCircle(cx, cy, radius-sw/2, [4]float32{0, 0, 0, 0}, 1)
	}
}

// DrawEllipse draws a filled ellipse.
func (c *Canvas) DrawEllipse(cx, cy, rx, ry float32, p Paint) {
	c.renderer.DrawFilledEllipse(cx, cy, rx, ry, p.Color.ToArray(), p.alpha())
}

// DrawLine draws a line from (x1,y1) to (x2,y2). p.StrokeWidth controls thickness.
func (c *Canvas) DrawLine(x1, y1, x2, y2 float32, p Paint) {
	sw := p.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	c.renderer.DrawLine(x1, y1, x2, y2, sw, p.Color.ToArray(), p.alpha())
}

// DrawLines draws a connected polyline through points.
func (c *Canvas) DrawLines(points []Point, p Paint) {
	for i := 0; i+1 < len(points); i++ {
		c.DrawLine(points[i].X, points[i].Y, points[i+1].X, points[i+1].Y, p)
	}
}

// DrawPolygon draws a filled or stroked polygon.
func (c *Canvas) DrawPolygon(points []Point, p Paint) {
	if len(points) < 3 {
		return
	}
	flat := make([]float32, len(points)*2)
	for i, pt := range points {
		flat[i*2] = pt.X
		flat[i*2+1] = pt.Y
	}
	if p.Fill || p.StrokeWidth == 0 {
		c.renderer.DrawFilledPolygon(flat, p.Color.ToArray(), p.alpha())
	} else {
		// Stroke: draw edges as lines.
		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			c.DrawLine(points[i].X, points[i].Y, points[j].X, points[j].Y, p)
		}
	}
}

// DrawArc draws a circular arc centered at (cx, cy).
// startAngle and sweepAngle are in radians.
func (c *Canvas) DrawArc(cx, cy, r, startAngle, sweepAngle float32, p Paint) {
	segs := 64
	pts := make([]Point, segs+1)
	for i := 0; i <= segs; i++ {
		a := startAngle + sweepAngle*float32(i)/float32(segs)
		pts[i] = Point{cx + r*float32(math.Cos(float64(a))), cy + r*float32(math.Sin(float64(a)))}
	}
	c.DrawLines(pts, p)
}

// DrawPath draws a Path using fill or stroke.
func (c *Canvas) DrawPath(path *Path, p Paint) {
	pts := path.Tessellate(48)
	if len(pts) == 0 {
		return
	}
	if p.Fill || p.StrokeWidth == 0 {
		c.renderer.DrawFilledPolygon(pts, p.Color.ToArray(), p.alpha())
	} else {
		sw := p.StrokeWidth
		if sw <= 0 {
			sw = 1
		}
		for i := 0; i+3 < len(pts); i += 2 {
			c.renderer.DrawLine(pts[i], pts[i+1], pts[i+2], pts[i+3], sw, p.Color.ToArray(), p.alpha())
		}
	}
}

// --- Text ---

// DrawText draws a string at (x, y). y is the baseline.
func (c *Canvas) DrawText(x, y float32, text string, style TextStyle) {
	atlas := loadFont(style.FontPath, style.Size)
	if atlas == nil {
		return // font not found — fail silently
	}
	atlasID := render.TextureID(atlas.AtlasTexture())
	col := style.Color.ToArray()
	penX := x
	for _, r := range text {
		g := atlas.GlyphInfo(r)
		c.renderer.DrawGlyph(atlasID, g, penX, y, col, 1)
		penX += g.Advance
	}
}

// MeasureText returns the pixel width and height of the given text string.
func (c *Canvas) MeasureText(text string, style TextStyle) Size {
	atlas := loadFont(style.FontPath, style.Size)
	if atlas == nil {
		return Size{float32(len(text)) * style.Size * 0.6, style.Size}
	}
	w, h := atlas.MeasureString(text)
	return Size{w, h}
}

// DrawTextInRect draws text inside a bounding rectangle, wrapping lines.
// DrawTextInRect draws text within bounds, breaking lines aggressively on spaces or new lines (\n).
func (c *Canvas) DrawTextInRect(rect Rect, text string, style TextStyle) {
	atlas := loadFont(style.FontPath, style.Size)
	if atlas == nil {
		return
	}
	lh := style.Size * style.LineHeight
	if lh == 0 {
		lh = style.Size * 1.2
	}

	// Pre-process explicit lines first
	lines := strings.Split(text, "\n")
	lineY := rect.Y + style.Size

	for _, hardLine := range lines {
		words := strings.Fields(hardLine)

		// If the line is empty due to repeating newlines, just advance vertically.
		if len(words) == 0 {
			lineY += lh
			if lineY > rect.Y+rect.H {
				return
			}
			continue
		}

		penX := rect.X
		for _, word := range words {
			ww, _ := atlas.MeasureString(word + " ")
			if penX+ww > rect.X+rect.W && penX > rect.X {
				penX = rect.X
				lineY += lh
			}
			if lineY > rect.Y+rect.H {
				return
			}
			c.DrawText(penX, lineY, word+" ", style)
			penX += ww
		}
		lineY += lh
		if lineY > rect.Y+rect.H {
			return
		}
	}
}

// --- Images ---

// DrawImage draws an image at (x, y) at its natural size.
func (c *Canvas) DrawImage(img *Image, x, y float32, p Paint) {
	if img == nil {
		return
	}
	tint := p.Color.ToArray()
	if tint == ([4]float32{}) {
		tint = [4]float32{1, 1, 1, 1}
	}
	c.renderer.DrawTexture(img.id, x, y, float32(img.w), float32(img.h), 0, 0, 1, 1, tint, p.alpha())
}

// DrawImageScaled draws an image stretched to (w, h) pixels.
func (c *Canvas) DrawImageScaled(img *Image, x, y, w, h float32, p Paint) {
	if img == nil {
		return
	}
	tint := p.Color.ToArray()
	if tint == ([4]float32{}) {
		tint = [4]float32{1, 1, 1, 1}
	}
	c.renderer.DrawTexture(img.id, x, y, w, h, 0, 0, 1, 1, tint, p.alpha())
}

// DrawImageRegion draws a sub-region of an image (for spritesheets).
func (c *Canvas) DrawImageRegion(img *Image, src, dst Rect, p Paint) {
	if img == nil {
		return
	}
	u0 := src.X / float32(img.w)
	v0 := src.Y / float32(img.h)
	u1 := (src.X + src.W) / float32(img.w)
	v1 := (src.Y + src.H) / float32(img.h)
	tint := [4]float32{1, 1, 1, p.alpha()}
	c.renderer.DrawTexture(img.id, dst.X, dst.Y, dst.W, dst.H, u0, v0, u1, v1, tint, 1)
}

// --- Utilities ---

// Width returns the canvas width in pixels.
func (c *Canvas) Width() float32 { return c.w }

// Height returns the canvas height in pixels.
func (c *Canvas) Height() float32 { return c.h }

// Size returns canvas dimensions as a Size.
func (c *Canvas) Size() Size { return Size{c.w, c.h} }

// Center returns the center point of the canvas.
func (c *Canvas) Center() Point { return Point{c.w / 2, c.h / 2} }

// Renderer returns the underlying Renderer (for advanced use).
func (c *Canvas) Renderer() render.Renderer { return c.renderer }
