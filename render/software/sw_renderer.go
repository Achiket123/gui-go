// Package software provides a CPU-side fallback renderer that uses
// Go's standard image package and the existing X11 Pixmap blitting path.
// It implements render.Renderer without requiring OpenGL.
package software

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
	"unsafe"

	"github.com/achiket123/gui-go/platform"
	"github.com/achiket123/gui-go/render"
)

// SWRenderer draws into a Go image.RGBA then blits the pixels to the X11 window
// via the existing platform Pixmap + XImage path each frame.
type SWRenderer struct {
	display unsafe.Pointer
	xwin    uintptr
	screen  int
	depth   int
	gc      unsafe.Pointer

	buf      *image.RGBA
	w, h     int
	ximg     *platform.XImageHandle
	mu       sync.Mutex
	textures map[render.TextureID]*image.RGBA
	nextTex  render.TextureID
}

func NewSWRenderer() *SWRenderer {
	return &SWRenderer{textures: make(map[render.TextureID]*image.RGBA)}
}

func (r *SWRenderer) Init(displayI, xwinI interface{}, w, h int) error {
	r.display = displayI.(unsafe.Pointer)
	r.xwin = xwinI.(uintptr)
	r.screen = platform.DefaultScreen(r.display)
	r.depth = platform.DefaultDepth(r.display, r.screen)
	r.gc = platform.CreateGC(r.display, r.xwin)
	r.resize(w, h)
	return nil
}

func (r *SWRenderer) resize(w, h int) {
	r.w, r.h = w, h
	r.buf = image.NewRGBA(image.Rect(0, 0, w, h))
	if r.ximg != nil {
		platform.DestroyXImage(r.ximg)
	}
	// Create a fresh XImage backed by r.buf.Pix
	r.ximg = platform.CreateXImage(r.display, r.screen, w, h, toBGRX(r.buf))
}

func (r *SWRenderer) Resize(w, h int) { r.resize(w, h) }

// toBGRX converts RGBA pixels to BGRX for X11.
func toBGRX(img *image.RGBA) []byte {
	src := img.Pix
	out := make([]byte, len(src))
	for i := 0; i < len(src); i += 4 {
		out[i+0] = src[i+2] // B
		out[i+1] = src[i+1] // G
		out[i+2] = src[i+0] // R
		out[i+3] = 0
	}
	return out
}

func (r *SWRenderer) BeginFrame(c [4]float32) {
	col := color.RGBA{
		R: uint8(c[0] * 255),
		G: uint8(c[1] * 255),
		B: uint8(c[2] * 255),
		A: 255,
	}
	draw.Draw(r.buf, r.buf.Bounds(), &image.Uniform{col}, image.Point{}, draw.Src)
}

func (r *SWRenderer) EndFrame() {
	// Convert to BGRX and blit via XImage.
	bgrx := toBGRX(r.buf)
	if r.ximg != nil {
		platform.DestroyXImage(r.ximg)
	}
	r.ximg = platform.CreateXImage(r.display, r.screen, r.w, r.h, bgrx)
	platform.PutXImage(r.display, r.xwin, r.gc, r.ximg, 0, 0, r.w, r.h)
	platform.Flush(r.display)
}

func (r *SWRenderer) setPixel(x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= r.w || y >= r.h {
		return
	}
	// Alpha blend over existing pixel.
	i := r.buf.PixOffset(x, y)
	bg := r.buf.Pix[i : i+4]
	fa := float64(c.A) / 255
	fb := 1 - fa
	bg[0] = uint8(float64(c.R)*fa + float64(bg[0])*fb)
	bg[1] = uint8(float64(c.G)*fa + float64(bg[1])*fb)
	bg[2] = uint8(float64(c.B)*fa + float64(bg[2])*fb)
	bg[3] = 255
}

func toRGBA(c [4]float32, opacity float32) color.RGBA {
	return color.RGBA{
		R: uint8(c[0] * 255),
		G: uint8(c[1] * 255),
		B: uint8(c[2] * 255),
		A: uint8(c[3] * opacity * 255),
	}
}

func (r *SWRenderer) DrawFilledRect(x, y, w, h, _ float32, c [4]float32, opacity float32) {
	col := toRGBA(c, opacity)
	rect := image.Rect(int(x), int(y), int(x+w), int(y+h))
	draw.Draw(r.buf, rect, &image.Uniform{col}, image.Point{}, draw.Over)
}

func (r *SWRenderer) DrawStrokedRect(x, y, w, h, _, sw float32, c [4]float32, opacity float32) {
	col := toRGBA(c, opacity)
	isw := int(sw)
	r.DrawFilledRect(x, y, w, sw, 0, c, opacity)
	r.DrawFilledRect(x, y+h-sw, w, sw, 0, c, opacity)
	r.DrawFilledRect(x, y+sw, sw, h-2*sw, 0, c, opacity)
	r.DrawFilledRect(x+w-sw, y+sw, sw, h-2*sw, 0, c, opacity)
	_ = col
	_ = isw
}

func (r *SWRenderer) DrawFilledCircle(cx, cy, radius float32, c [4]float32, opacity float32) {
	col := toRGBA(c, opacity)
	ri := int(radius)
	cxi, cyi := int(cx), int(cy)
	for dy := -ri; dy <= ri; dy++ {
		for dx := -ri; dx <= ri; dx++ {
			if dx*dx+dy*dy <= ri*ri {
				r.setPixel(cxi+dx, cyi+dy, col)
			}
		}
	}
}

func (r *SWRenderer) DrawFilledEllipse(cx, cy, rx, ry float32, c [4]float32, opacity float32) {
	col := toRGBA(c, opacity)
	rxi, ryi := int(rx), int(ry)
	cxi, cyi := int(cx), int(cy)
	for dy := -ryi; dy <= ryi; dy++ {
		for dx := -rxi; dx <= rxi; dx++ {
			fx := float64(dx) / float64(rxi)
			fy := float64(dy) / float64(ryi)
			if fx*fx+fy*fy <= 1 {
				r.setPixel(cxi+dx, cyi+dy, col)
			}
		}
	}
}

func (r *SWRenderer) DrawLine(x1, y1, x2, y2, thickness float32, c [4]float32, opacity float32) {
	col := toRGBA(c, opacity)
	// Bresenham-like integer line.
	dx := int(math.Abs(float64(x2 - x1)))
	dy := int(math.Abs(float64(y2 - y1)))
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	x, y := int(x1), int(y1)
	err := dx - dy
	t := int(thickness/2) + 1
	for {
		for ox := -t; ox <= t; ox++ {
			for oy := -t; oy <= t; oy++ {
				r.setPixel(x+ox, y+oy, col)
			}
		}
		if x == int(x2) && y == int(y2) {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func (r *SWRenderer) DrawFilledPolygon(pts []float32, c [4]float32, opacity float32) {
	// Simple scanline fill — adequate for software fallback.
	if len(pts) < 6 {
		return
	}
	col := toRGBA(c, opacity)
	n := len(pts) / 2
	minY, maxY := int(pts[1]), int(pts[1])
	for i := 1; i < n; i++ {
		if y := int(pts[i*2+1]); y < minY {
			minY = y
		} else if y > maxY {
			maxY = y
		}
	}
	for y := minY; y <= maxY; y++ {
		var xs []int
		j := n - 1
		for i := 0; i < n; i++ {
			yi := int(pts[i*2+1])
			yj := int(pts[j*2+1])
			if (yi < y && yj >= y) || (yj < y && yi >= y) {
				xi := pts[i*2]
				xj := pts[j*2]
				xc := int(xi + (float32(y)-float32(yi))/(float32(yj)-float32(yi))*(xj-xi))
				xs = append(xs, xc)
			}
			j = i
		}
		for i := 0; i+1 < len(xs); i += 2 {
			a, b := xs[i], xs[i+1]
			if a > b {
				a, b = b, a
			}
			for x := a; x <= b; x++ {
				r.setPixel(x, y, col)
			}
		}
	}
}

func (r *SWRenderer) DrawGradientRect(x, y, w, h float32, c0, c1 [4]float32, p1, p2 [2]float32, opacity float32) {
	dx, dy := p2[0]-p1[0], p2[1]-p1[1]
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		r.DrawFilledRect(x, y, w, h, 0, c0, opacity)
		return
	}

	for row := 0; row < int(h); row++ {
		for col := 0; col < int(w); col++ {
			vx, vy := x+float32(col), y+float32(row)
			dot := (vx-p1[0])*dx + (vy-p1[1])*dy
			t := dot / lengthSq
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}

			rc := [4]float32{
				c0[0]*(1-t) + c1[0]*t,
				c0[1]*(1-t) + c1[1]*t,
				c0[2]*(1-t) + c1[2]*t,
				c0[3]*(1-t) + c1[3]*t,
			}
			r.setPixel(int(x)+col, int(y)+row, toRGBA(rc, opacity))
		}
	}
}

func (r *SWRenderer) DrawTexture(id render.TextureID, x, y, w, h, u0, v0, u1, v1 float32, tint [4]float32, opacity float32) {
	tex, ok := r.textures[id]
	if !ok {
		return
	}
	sw := float32(tex.Bounds().Dx())
	sh := float32(tex.Bounds().Dy())
	for dy := 0; dy < int(h); dy++ {
		for dx := 0; dx < int(w); dx++ {
			tu := u0 + (u1-u0)*float32(dx)/w
			tv := v0 + (v1-v0)*float32(dy)/h
			sx := int(tu * sw)
			sy := int(tv * sh)
			if sx < 0 {
				sx = 0
			}
			if sy < 0 {
				sy = 0
			}
			if sx >= int(sw) {
				sx = int(sw) - 1
			}
			if sy >= int(sh) {
				sy = int(sh) - 1
			}
			src := tex.RGBAAt(sx, sy)
			r.setPixel(int(x)+dx, int(y)+dy, color.RGBA{
				R: uint8(float64(src.R) * float64(tint[0])),
				G: uint8(float64(src.G) * float64(tint[1])),
				B: uint8(float64(src.B) * float64(tint[2])),
				A: uint8(float64(src.A) * float64(tint[3]) * float64(opacity)),
			})
		}
	}
}

func (r *SWRenderer) DrawGlyph(_ render.TextureID, _ render.GlyphMetrics, _, _ float32, _ [4]float32, _ float32) {
	// Software glyph rendering is not implemented — use GL renderer for text.
}

func (r *SWRenderer) SetClipRect(x, y, w, h float32) {
	// Approximation: use Go's image Sub-image bounds for clipping.
	// Full stencil clipping is not implemented in the SW renderer.
}
func (r *SWRenderer) ClearClip()                 {}
func (r *SWRenderer) PushTransform(_ [9]float32) {}
func (r *SWRenderer) PopTransform()              {}

func (r *SWRenderer) UploadTexture(w, h int, pixels []byte) render.TextureID {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, pixels)
	r.mu.Lock()
	id := r.nextTex
	r.nextTex++
	r.textures[id] = img
	r.mu.Unlock()
	return id
}

func (r *SWRenderer) UpdateTexture(id render.TextureID, w, h int, pixels []byte) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	copy(img.Pix, pixels)
	r.mu.Lock()
	r.textures[id] = img
	r.mu.Unlock()
}

func (r *SWRenderer) DeleteTexture(id render.TextureID) {
	r.mu.Lock()
	delete(r.textures, id)
	r.mu.Unlock()
}

func (r *SWRenderer) BeginScene3D() {}
func (r *SWRenderer) EndScene3D()   {}

func (r *SWRenderer) SetGlobalOpacity(_ float32) {}

func (r *SWRenderer) Width() int  { return r.w }
func (r *SWRenderer) Height() int { return r.h }
