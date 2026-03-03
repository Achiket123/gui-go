package canvas

import "math"

// Cmd is a path command type.
type cmdType int

const (
	cmdMove cmdType = iota
	cmdLine
	cmdQuad
	cmdCubic
	cmdArc
	cmdClose
)

type cmd struct {
	t      cmdType
	x, y   float32 // primary point
	ax, ay float32 // control point 1 / arc params
	bx, by float32 // control point 2 / arc params
}

// Path is a sequence of drawing commands that describe a shape.
// Build it with moveTo/lineTo etc., then pass to Canvas.DrawPath.
type Path struct {
	cmds   []cmd
	cur    Point
	start  Point
	cached []float32
	dirty  bool
}

// NewPath creates an empty Path.
func NewPath() *Path { return &Path{} }

// MoveTo moves the current point without drawing.
func (p *Path) MoveTo(x, y float32) {
	p.cmds = append(p.cmds, cmd{t: cmdMove, x: x, y: y})
	p.cur = Point{x, y}
	p.start = p.cur
	p.dirty = true
}

// LineTo adds a straight line to (x, y).
func (p *Path) LineTo(x, y float32) {
	if len(p.cmds) == 0 {
		p.MoveTo(x, y)
		return
	}
	p.cmds = append(p.cmds, cmd{t: cmdLine, x: x, y: y})
	p.cur = Point{x, y}
	p.dirty = true
}

// QuadTo adds a quadratic Bézier curve.
func (p *Path) QuadTo(cpx, cpy, x, y float32) {
	p.cmds = append(p.cmds, cmd{t: cmdQuad, ax: cpx, ay: cpy, x: x, y: y})
	p.cur = Point{x, y}
	p.dirty = true
}

// CubicTo adds a cubic Bézier curve.
func (p *Path) CubicTo(cp1x, cp1y, cp2x, cp2y, x, y float32) {
	p.cmds = append(p.cmds, cmd{t: cmdCubic, ax: cp1x, ay: cp1y, bx: cp2x, by: cp2y, x: x, y: y})
	p.cur = Point{x, y}
	p.dirty = true
}

// ArcTo adds an elliptical arc.
// angle is the start angle in radians; sweep is the arc sweep in radians.
func (p *Path) ArcTo(cx, cy, rx, ry, angle, sweep float32) {
	p.cmds = append(p.cmds, cmd{t: cmdArc, x: cx, y: cy, ax: rx, ay: ry, bx: angle, by: sweep})
	p.dirty = true
}

// Close closes the current contour with a line back to the last MoveTo point.
func (p *Path) Close() {
	p.cmds = append(p.cmds, cmd{t: cmdClose, x: p.start.X, y: p.start.Y})
	p.cur = p.start
	p.dirty = true
}

// --- Convenience shape adders ---

// AddRect adds an axis-aligned rectangle subpath.
func (p *Path) AddRect(x, y, w, h float32) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// AddRoundedRect adds a rounded rectangle subpath.
func (p *Path) AddRoundedRect(x, y, w, h, r float32) {
	if r <= 0 {
		p.AddRect(x, y, w, h)
		return
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.ArcTo(x+w-r, y+r, r, r, -math.Pi/2, math.Pi/2)
	p.LineTo(x+w, y+h-r)
	p.ArcTo(x+w-r, y+h-r, r, r, 0, math.Pi/2)
	p.LineTo(x+r, y+h)
	p.ArcTo(x+r, y+h-r, r, r, math.Pi/2, math.Pi/2)
	p.LineTo(x, y+r)
	p.ArcTo(x+r, y+r, r, r, math.Pi, math.Pi/2)
	p.Close()
}

// AddCircle adds a full circle subpath.
func (p *Path) AddCircle(cx, cy, r float32) {
	p.MoveTo(cx+r, cy)
	p.ArcTo(cx, cy, r, r, 0, 2*math.Pi)
	p.Close()
}

// AddPolygon adds a closed polygon subpath.
func (p *Path) AddPolygon(pts []Point) {
	if len(pts) == 0 {
		return
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, pt := range pts[1:] {
		p.LineTo(pt.X, pt.Y)
	}
	p.Close()
}

// Tessellate converts path commands to a flat list of triangle vertices.
// Returns flat x,y pairs suitable for DrawFilledPolygon.
// Arc segments are approximated with multiple line segments.
func (p *Path) Tessellate(segments int) []float32 {
	if !p.dirty && p.cached != nil {
		return p.cached
	}
	if segments <= 0 {
		segments = 32
	}
	var pts []float32
	var contour []float32
	flush := func() {
		if len(contour) > 0 {
			pts = append(pts, contour...)
			contour = nil
		}
	}
	curX, curY := float32(0), float32(0)
	for _, c := range p.cmds {
		switch c.t {
		case cmdMove:
			flush()
			curX, curY = c.x, c.y
			contour = append(contour, curX, curY)
		case cmdLine:
			curX, curY = c.x, c.y
			contour = append(contour, curX, curY)
		case cmdClose:
			if len(contour) >= 4 {
				contour = append(contour, contour[0], contour[1])
			}
			flush()
		case cmdArc:
			cx, cy, rx, ry := c.x, c.y, c.ax, c.ay
			startAngle, sweep := float64(c.bx), float64(c.by)
			for i := 0; i <= segments; i++ {
				angle := startAngle + sweep*float64(i)/float64(segments)
				x := cx + rx*float32(math.Cos(angle))
				y := cy + ry*float32(math.Sin(angle))
				contour = append(contour, x, y)
			}
			curX = cx + rx*float32(math.Cos(startAngle+sweep))
			curY = cy + ry*float32(math.Sin(startAngle+sweep))
		case cmdQuad:
			// De Casteljau for quadratic
			x0, y0 := curX, curY
			cpx, cpy := c.ax, c.ay
			x1, y1 := c.x, c.y
			for i := 1; i <= segments; i++ {
				t := float32(i) / float32(segments)
				mt := 1 - t
				x := mt*mt*x0 + 2*mt*t*cpx + t*t*x1
				y := mt*mt*y0 + 2*mt*t*cpy + t*t*y1
				contour = append(contour, x, y)
			}
			curX, curY = x1, y1
		case cmdCubic:
			x0, y0 := curX, curY
			cp1x, cp1y := c.ax, c.ay
			cp2x, cp2y := c.bx, c.by
			x1, y1 := c.x, c.y
			for i := 1; i <= segments; i++ {
				t := float32(i) / float32(segments)
				mt := 1 - t
				x := mt*mt*mt*x0 + 3*mt*mt*t*cp1x + 3*mt*t*t*cp2x + t*t*t*x1
				y := mt*mt*mt*y0 + 3*mt*mt*t*cp1y + 3*mt*t*t*cp2y + t*t*t*y1
				contour = append(contour, x, y)
			}
			curX, curY = x1, y1
		}
	}
	flush()
	p.cached = pts
	p.dirty = false
	return pts
}

// Bounds returns the bounding box of the path.
func (p *Path) Bounds() Rect {
	pts := p.Tessellate(16)
	if len(pts) == 0 {
		return Rect{}
	}
	minX, minY, maxX, maxY := pts[0], pts[1], pts[0], pts[1]
	for i := 2; i < len(pts); i += 2 {
		if pts[i] < minX {
			minX = pts[i]
		}
		if pts[i] > maxX {
			maxX = pts[i]
		}
		if pts[i+1] < minY {
			minY = pts[i+1]
		}
		if pts[i+1] > maxY {
			maxY = pts[i+1]
		}
	}
	return Rect{minX, minY, maxX - minX, maxY - minY}
}

// Contains tests whether a point is inside the path (even-odd rule).
func (p *Path) Contains(x, y float32) bool {
	pts := p.Tessellate(16)
	n := len(pts) / 2
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := pts[i*2], pts[i*2+1]
		xj, yj := pts[j*2], pts[j*2+1]
		if ((yi > y) != (yj > y)) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
			inside = !inside
		}
		j = i
	}
	return inside
}
