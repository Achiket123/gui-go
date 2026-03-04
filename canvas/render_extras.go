// Package canvas — render_extras.go
//
// Advanced rendering utilities:
//   BezierPath   — fluent SVG-like path builder (MoveTo/LineTo/QuadTo/CubicTo/ArcTo)
//   DrawBoxShadow — multi-layer blur-approximated box shadow
//   DPIScaler    — device-pixel-ratio aware coordinate wrapper
//   Grayscale    — CPU-side grayscale filter on *image.RGBA
//   Tint         — per-channel multiplicative tint
//   Brightness   — brightness adjust
//   BoxBlur      — software box-blur (arbitrary radius)
//   IconAtlas    — named icon → rune mapping over any icon font
package canvas

import (
	"image"
	"image/color"
	"math"
)

// ─────────────────────────────────────────────────────────────────────────────
// BezierPath
// ─────────────────────────────────────────────────────────────────────────────

// PathVerb classifies a path command.
type PathVerb int

const (
	VerbMoveTo  PathVerb = iota
	VerbLineTo
	VerbQuadTo  // 1 control point
	VerbCubicTo // 2 control points
	VerbArcTo
	VerbClose
)

// pathCmd stores one path command (up to 3 points).
type pathCmd struct {
	verb   PathVerb
	p1, p2, p3 Point
	// ArcTo: p1=centre, p2={rx,ry}, p3={startAngle,sweep}
}

// BezierPath is a mutable sequence of path commands.
type BezierPath struct {
	cmds  []pathCmd
	start Point
	cur   Point
}

// NewBezierPath creates an empty path.
func NewBezierPath() *BezierPath { return &BezierPath{} }

// MoveTo begins a new sub-path at (x, y).
func (bp *BezierPath) MoveTo(x, y float32) *BezierPath {
	p := Point{x, y}
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbMoveTo, p1: p})
	bp.cur = p
	bp.start = p
	return bp
}

// LineTo adds a straight line to (x, y).
func (bp *BezierPath) LineTo(x, y float32) *BezierPath {
	p := Point{x, y}
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbLineTo, p1: p})
	bp.cur = p
	return bp
}

// QuadTo adds a quadratic Bézier to (x2, y2) via control point (cx, cy).
func (bp *BezierPath) QuadTo(cx, cy, x2, y2 float32) *BezierPath {
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbQuadTo, p1: Point{cx, cy}, p2: Point{x2, y2}})
	bp.cur = Point{x2, y2}
	return bp
}

// CubicTo adds a cubic Bézier to (x3, y3) via (c1x, c1y) and (c2x, c2y).
func (bp *BezierPath) CubicTo(c1x, c1y, c2x, c2y, x3, y3 float32) *BezierPath {
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbCubicTo,
		p1: Point{c1x, c1y}, p2: Point{c2x, c2y}, p3: Point{x3, y3}})
	bp.cur = Point{x3, y3}
	return bp
}

// ArcTo adds an elliptical arc.
// centre (cx, cy), radii (rx, ry), from startAngle to startAngle+sweep (radians).
func (bp *BezierPath) ArcTo(cx, cy, rx, ry, startAngle, sweep float32) *BezierPath {
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbArcTo,
		p1: Point{cx, cy}, p2: Point{rx, ry}, p3: Point{startAngle, sweep}})
	return bp
}

// Close adds a line from the current point back to the sub-path start.
func (bp *BezierPath) Close() *BezierPath {
	bp.cmds = append(bp.cmds, pathCmd{verb: VerbClose})
	return bp
}

// RoundedRect appends a complete rounded-rectangle sub-path.
func (bp *BezierPath) RoundedRect(x, y, w, h, r float32) *BezierPath {
	pi := float32(math.Pi)
	return bp.
		MoveTo(x+r, y).
		LineTo(x+w-r, y).
		ArcTo(x+w-r, y+r, r, r, -pi/2, pi/2).
		LineTo(x+w, y+h-r).
		ArcTo(x+w-r, y+h-r, r, r, 0, pi/2).
		LineTo(x+r, y+h).
		ArcTo(x+r, y+h-r, r, r, pi/2, pi/2).
		LineTo(x, y+r).
		ArcTo(x+r, y+r, r, r, pi, pi/2).
		Close()
}

// Tessellate converts the path to a flat []float32 slice of (x, y) pairs.
// segs controls the number of line segments per curve (default 32).
func (bp *BezierPath) Tessellate(segs int) []float32 {
	if segs <= 0 {
		segs = 32
	}
	var pts []float32
	cur := Point{}
	start := Point{}

	for _, cmd := range bp.cmds {
		switch cmd.verb {
		case VerbMoveTo:
			cur = cmd.p1
			start = cur
			pts = append(pts, cur.X, cur.Y)

		case VerbLineTo:
			cur = cmd.p1
			pts = append(pts, cur.X, cur.Y)

		case VerbQuadTo:
			cp, end := cmd.p1, cmd.p2
			for i := 1; i <= segs; i++ {
				t := float32(i) / float32(segs)
				x, y := quadBez(cur.X, cur.Y, cp.X, cp.Y, end.X, end.Y, t)
				pts = append(pts, x, y)
			}
			cur = end

		case VerbCubicTo:
			c1, c2, end := cmd.p1, cmd.p2, cmd.p3
			for i := 1; i <= segs; i++ {
				t := float32(i) / float32(segs)
				x, y := cubicBez(cur.X, cur.Y, c1.X, c1.Y, c2.X, c2.Y, end.X, end.Y, t)
				pts = append(pts, x, y)
			}
			cur = end

		case VerbArcTo:
			cx, cy := cmd.p1.X, cmd.p1.Y
			rx, ry := cmd.p2.X, cmd.p2.Y
			a0, sweep := cmd.p3.X, cmd.p3.Y
			for i := 1; i <= segs; i++ {
				a := a0 + sweep*float32(i)/float32(segs)
				px := cx + rx*float32(math.Cos(float64(a)))
				py := cy + ry*float32(math.Sin(float64(a)))
				pts = append(pts, px, py)
				cur = Point{px, py}
			}

		case VerbClose:
			pts = append(pts, start.X, start.Y)
			cur = start
		}
	}
	return pts
}

// DrawBezierPath renders the path with the given paint.
// For filled shapes use FillPaint; for outlines use StrokePaint.
func (c *Canvas) DrawBezierPath(bp *BezierPath, p Paint, segs int) {
	pts := bp.Tessellate(segs)
	if len(pts) < 4 {
		return
	}
	sw := p.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	for i := 0; i+3 < len(pts); i += 2 {
		c.renderer.DrawLine(pts[i], pts[i+1], pts[i+2], pts[i+3], sw,
			p.Color.ToArray(), p.alpha())
	}
}

func quadBez(x0, y0, cx, cy, x1, y1, t float32) (float32, float32) {
	mt := 1 - t
	return mt*mt*x0 + 2*mt*t*cx + t*t*x1,
		mt*mt*y0 + 2*mt*t*cy + t*t*y1
}

func cubicBez(x0, y0, c1x, c1y, c2x, c2y, x1, y1, t float32) (float32, float32) {
	mt := 1 - t
	return mt*mt*mt*x0 + 3*mt*mt*t*c1x + 3*mt*t*t*c2x + t*t*t*x1,
		mt*mt*mt*y0 + 3*mt*mt*t*c1y + 3*mt*t*t*c2y + t*t*t*y1
}

// ─────────────────────────────────────────────────────────────────────────────
// Box shadow
// ─────────────────────────────────────────────────────────────────────────────

// ShadowOptions configures a box shadow.
type ShadowOptions struct {
	OffsetX, OffsetY float32
	Blur             float32 // approximated via N expanding layers
	Spread           float32 // expands the shadow rect outward
	Color            Color
	Layers           int // quality — default 8
}

// DrawBoxShadow draws an approximated multi-layer blur shadow behind a rect.
// Call this BEFORE drawing the widget background so the shadow appears behind it.
func (c *Canvas) DrawBoxShadow(x, y, w, h, radius float32, opts ShadowOptions) {
	layers := opts.Layers
	if layers <= 0 {
		layers = 8
	}
	blur := opts.Blur
	if blur <= 0 {
		blur = 8
	}
	spread := opts.Spread
	baseAlpha := opts.Color.A
	if baseAlpha == 0 {
		baseAlpha = 0.30
	}

	for i := 0; i < layers; i++ {
		t := float32(i) / float32(layers)
		exp := spread + blur*(1-t)
		alpha := baseAlpha * (1 - t) / float32(layers) * 3
		col := opts.Color
		col.A = alpha
		c.DrawRoundedRect(
			x+opts.OffsetX-exp,
			y+opts.OffsetY-exp,
			w+exp*2, h+exp*2,
			radius+exp,
			FillPaint(col),
		)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DPIScaler
// ─────────────────────────────────────────────────────────────────────────────

// DPIScaler wraps a Canvas and multiplies all coordinates/sizes by the
// device pixel ratio (DPR).  Typical values: 1.0 (standard), 2.0 (HiDPI).
//
//	dpi := canvas.NewDPIScaler(c, 2.0)
//	dpi.DrawRect(10, 10, 100, 40, paint) // maps to pixels 20, 20, 200, 80
type DPIScaler struct {
	C   *Canvas
	DPR float32
}

// NewDPIScaler creates a DPIScaler with the given device pixel ratio.
func NewDPIScaler(c *Canvas, dpr float32) *DPIScaler {
	if dpr <= 0 {
		dpr = 1
	}
	return &DPIScaler{C: c, DPR: dpr}
}

func (d *DPIScaler) s(v float32) float32  { return v * d.DPR }

// Scale converts a logical pixel value to physical pixels.
func (d *DPIScaler) Scale(v float32) float32 { return d.s(v) }

// ScaleStyle returns a TextStyle with the font size scaled by DPR.
func (d *DPIScaler) ScaleStyle(st TextStyle) TextStyle {
	st.Size = d.s(st.Size)
	return st
}

func (d *DPIScaler) DrawRect(x, y, w, h float32, p Paint) {
	d.C.DrawRect(d.s(x), d.s(y), d.s(w), d.s(h), p)
}
func (d *DPIScaler) DrawRoundedRect(x, y, w, h, r float32, p Paint) {
	d.C.DrawRoundedRect(d.s(x), d.s(y), d.s(w), d.s(h), d.s(r), p)
}
func (d *DPIScaler) DrawCircle(cx, cy, r float32, p Paint) {
	d.C.DrawCircle(d.s(cx), d.s(cy), d.s(r), p)
}
func (d *DPIScaler) DrawLine(x1, y1, x2, y2 float32, p Paint) {
	d.C.DrawLine(d.s(x1), d.s(y1), d.s(x2), d.s(y2), p)
}
func (d *DPIScaler) DrawText(x, y float32, text string, st TextStyle) {
	d.C.DrawText(d.s(x), d.s(y), text, d.ScaleStyle(st))
}

// Canvas returns the underlying *Canvas.
func (d *DPIScaler) Canvas() *Canvas { return d.C }

// ─────────────────────────────────────────────────────────────────────────────
// Image filters
// ─────────────────────────────────────────────────────────────────────────────

// Grayscale converts img to grayscale in-place using the luminance formula.
func Grayscale(img *image.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			lum := uint8(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B))
			img.SetRGBA(x, y, color.RGBA{lum, lum, lum, c.A})
		}
	}
}

// Tint multiplies every pixel channel by the corresponding tint value (0–255).
func Tint(img *image.RGBA, r, g, b, a uint8) {
	rf, gf, bf, af := float64(r)/255, float64(g)/255, float64(b)/255, float64(a)/255
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R) * rf),
				G: uint8(float64(c.G) * gf),
				B: uint8(float64(c.B) * bf),
				A: uint8(float64(c.A) * af),
			})
		}
	}
}

// Brightness multiplies each RGB channel by factor (>1 = brighter, <1 = darker).
func Brightness(img *image.RGBA, factor float64) {
	clamp := func(v float64) uint8 {
		if v > 255 {
			return 255
		}
		if v < 0 {
			return 0
		}
		return uint8(v)
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: clamp(float64(c.R) * factor),
				G: clamp(float64(c.G) * factor),
				B: clamp(float64(c.B) * factor),
				A: c.A,
			})
		}
	}
}

// BoxBlur applies an in-place box blur with kernel radius r (r=1 → 3×3).
func BoxBlur(img *image.RGBA, r int) {
	if r <= 0 {
		return
	}
	b := img.Bounds()
	tmp := image.NewRGBA(b)
	size := float64((2*r + 1) * (2*r + 1))

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var rr, gg, bb, aa float64
			for ky := -r; ky <= r; ky++ {
				for kx := -r; kx <= r; kx++ {
					sx := clampCoord(x+kx, b.Min.X, b.Max.X-1)
					sy := clampCoord(y+ky, b.Min.Y, b.Max.Y-1)
					c := img.RGBAAt(sx, sy)
					rr += float64(c.R)
					gg += float64(c.G)
					bb += float64(c.B)
					aa += float64(c.A)
				}
			}
			tmp.SetRGBA(x, y, color.RGBA{
				R: uint8(rr / size),
				G: uint8(gg / size),
				B: uint8(bb / size),
				A: uint8(aa / size),
			})
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, tmp.RGBAAt(x, y))
		}
	}
}

func clampCoord(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ─────────────────────────────────────────────────────────────────────────────
// IconAtlas
// ─────────────────────────────────────────────────────────────────────────────

// IconEntry maps a name to a Unicode rune in the icon font.
type IconEntry struct {
	Name string
	Rune rune
}

// IconAtlas pre-maps named icons to Unicode runes in an icon font.
// It delegates actual rendering to the canvas text engine, so no extra
// texture management is needed.
//
//	atlas := canvas.NewIconAtlas("/usr/share/fonts/MaterialSymbols.ttf", 24)
//	atlas.Register("check",    '\ue876')
//	atlas.Register("close",    '\ue5cd')
//	atlas.Register("settings", '\ue8b8')
//
//	// In your Draw function:
//	atlas.Draw(c, "check", 100, 100, canvas.White)
type IconAtlas struct {
	fontPath string
	size     float32
	icons    map[string]rune
	style    TextStyle
}

// NewIconAtlas creates an IconAtlas using the given TTF path and pixel size.
func NewIconAtlas(fontPath string, size float32) *IconAtlas {
	return &IconAtlas{
		fontPath: fontPath,
		size:     size,
		icons:    make(map[string]rune),
		style:    TextStyle{Size: size, FontPath: fontPath},
	}
}

// Register maps a name to a rune.
func (ia *IconAtlas) Register(name string, r rune) { ia.icons[name] = r }

// RegisterMany registers multiple icons at once.
func (ia *IconAtlas) RegisterMany(entries []IconEntry) {
	for _, e := range entries {
		ia.icons[e.Name] = e.Rune
	}
}

// Draw renders the named icon centred at (cx, cy) with the given colour.
func (ia *IconAtlas) Draw(c *Canvas, name string, cx, cy float32, col Color) {
	r, ok := ia.icons[name]
	if !ok {
		return
	}
	st := ia.style
	st.Color = col
	sz := c.MeasureText(string(r), st)
	c.DrawText(cx-sz.W/2, cy+sz.H/2, string(r), st)
}

// DrawAt renders the named icon with its top-left at (x, y).
func (ia *IconAtlas) DrawAt(c *Canvas, name string, x, y float32, col Color) {
	r, ok := ia.icons[name]
	if !ok {
		return
	}
	st := ia.style
	st.Color = col
	c.DrawText(x, y+ia.size, string(r), st)
}

// Has returns true if name is registered.
func (ia *IconAtlas) Has(name string) bool {
	_, ok := ia.icons[name]
	return ok
}

// Size returns the pixel size the atlas was created at.
func (ia *IconAtlas) Size() float32 { return ia.size }

