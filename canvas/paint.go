package canvas

import "math"

// StrokeCap defines how line endpoints are drawn.
type StrokeCap int

const (
	CapButt   StrokeCap = iota // Flat cap at endpoint
	CapRound                   // Round cap
	CapSquare                  // Square cap extending past endpoint
)

// StrokeJoin defines how connected line segments join.
type StrokeJoin int

const (
	JoinMiter StrokeJoin = iota
	JoinRound
	JoinBevel
)

// BlendMode controls how new pixels combine with the canvas.
type BlendMode int

const (
	BlendNormal BlendMode = iota
	BlendAdd
	BlendMultiply
	BlendScreen
)

// Color is an RGBA color with float32 components in [0, 1].
type Color struct {
	R, G, B, A float32
}

// ToArray returns a [4]float32 for the renderer.
func (c Color) ToArray() [4]float32 { return [4]float32{c.R, c.G, c.B, c.A} }

// WithAlpha returns a copy of the color with the given alpha.
func (c Color) WithAlpha(a float32) Color { return Color{c.R, c.G, c.B, a} }

// RGBA8 creates a Color from 0–255 byte components.
func RGBA8(r, g, b, a uint8) Color {
	return Color{float32(r) / 255, float32(g) / 255, float32(b) / 255, float32(a) / 255}
}

// RGB8 creates an opaque color from 0–255 bytes.
func RGB8(r, g, b uint8) Color { return RGBA8(r, g, b, 255) }

// Hex parses "#RRGGBB" or "#RGB".
func Hex(h string) Color {
	if len(h) > 0 && h[0] == '#' {
		h = h[1:]
	}
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	var v uint64
	for _, c := range h {
		v <<= 4
		if c >= '0' && c <= '9' {
			v |= uint64(c - '0')
		} else if c >= 'a' && c <= 'f' {
			v |= uint64(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			v |= uint64(c-'A') + 10
		}
	}
	return RGBA8(uint8(v>>16), uint8(v>>8), uint8(v), 255)
}

// Lerp linearly interpolates between two colors.
func Lerp(c0, c1 Color, t float32) Color {
	return Color{
		c0.R*(1-t) + c1.R*t,
		c0.G*(1-t) + c1.G*t,
		c0.B*(1-t) + c1.B*t,
		c0.A*(1-t) + c1.A*t,
	}
}

// --- Predefined colors ---

var (
	Black       = RGB8(0, 0, 0)
	White       = RGB8(255, 255, 255)
	Red         = RGB8(255, 0, 0)
	Green       = RGB8(0, 200, 0)
	Blue        = RGB8(0, 0, 255)
	Yellow      = RGB8(255, 255, 0)
	Cyan        = RGB8(0, 255, 255)
	Magenta     = RGB8(255, 0, 255)
	Gray        = RGB8(128, 128, 128)
	LightGray   = RGB8(192, 192, 192)
	DarkGray    = RGB8(64, 64, 64)
	Orange      = RGB8(255, 165, 0)
	Pink        = RGB8(255, 105, 180)
	Purple      = RGB8(128, 0, 128)
	Transparent = Color{}
)

// GradientStop is a color at a position along a gradient (0.0–1.0).
type GradientStop struct {
	Color    Color
	Position float32
}

// LinearGradient describes a linear gradient between two points.
type LinearGradient struct {
	From, To Point
	Stops    []GradientStop
}

// RadialGradient describes a radial gradient from a center point.
type RadialGradient struct {
	Center Point
	Radius float32
	Stops  []GradientStop
}

// Paint describes HOW something is drawn.
type Paint struct {
	Color       Color
	LinearGrad  *LinearGradient
	RadialGrad  *RadialGradient
	Opacity     float32 // 0=transparent, 1=opaque (default)
	StrokeWidth float32 // for outline drawing
	StrokeCap   StrokeCap
	StrokeJoin  StrokeJoin
	AntiAlias   bool
	BlendMode   BlendMode
	Fill        bool // if true, fill shape; if false, stroke it
}

// FillPaint returns a solid-color fill paint.
func FillPaint(c Color) Paint {
	return Paint{Color: c, Opacity: 1, AntiAlias: true, Fill: true}
}

// StrokePaint returns a solid-color stroke paint.
func StrokePaint(c Color, width float32) Paint {
	return Paint{Color: c, Opacity: 1, StrokeWidth: width, AntiAlias: true, StrokeCap: CapRound, StrokeJoin: JoinRound}
}

// GradientPaint returns a horizontal linear-gradient fill.
func GradientPaint(from, to Color) Paint {
	return Paint{
		Opacity: 1, AntiAlias: true, Fill: true,
		LinearGrad: &LinearGradient{
			Stops: []GradientStop{{from, 0}, {to, 1}},
		},
	}
}

// alpha returns effective opacity * paint.Color.A.
func (p Paint) alpha() float32 {
	op := p.Opacity
	if op == 0 {
		op = 1
	}
	return op
}

// Point is a 2D float coordinate.
type Point struct{ X, Y float32 }

// Dist returns the distance between two points.
func (p Point) Dist(q Point) float32 {
	dx, dy := p.X-q.X, p.Y-q.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// Rect is an axis-aligned rectangle.
type Rect struct{ X, Y, W, H float32 }

// Size is a 2D dimension.
type Size struct{ W, H float32 }

// EdgeInsets holds per-side padding values.
type EdgeInsets struct{ Top, Right, Bottom, Left float32 }

// All returns EdgeInsets with all sides set to v.
func All(v float32) EdgeInsets { return EdgeInsets{v, v, v, v} }

// Symmetric returns EdgeInsets with horizontal and vertical symmetry.
func Symmetric(h, v float32) EdgeInsets { return EdgeInsets{v, h, v, h} }

// TextStyle describes how text should be rendered.
type TextStyle struct {
	Color      Color
	Size       float32 // point size
	FontPath   string  // TTF path; empty = default font
	Bold       bool
	Italic     bool
	LineHeight float32 // multiplier, default 1.2
}

// DefaultTextStyle returns a sane default text style.
func DefaultTextStyle() TextStyle {
	return TextStyle{Color: White, Size: 14, LineHeight: 1.2}
}
