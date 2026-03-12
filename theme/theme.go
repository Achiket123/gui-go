// Package theme provides a design-token-based theming system for goui.
//
// Features:
//   - Palette: 20+ semantic color slots (backgrounds, text, accents, states)
//   - TypeScale: Display/H1–H3/Body/Code/Hint typography roles
//   - Spacing: Scale built from base unit (XS→XXL)
//   - Radii: Border-radius tokens
//   - ElevationScale: 5 shadow levels
//   - Motion: Animation duration constants
//
// Built-in themes: Dark(), Light()
// Custom themes: Custom(fn) for one-line overrides
// Global API: Set(t), Current(), OnChange(cb)
package theme

import (
	"sync"

	"github.com/achiket123/gui-go/canvas"
)

// ─────────────────────────────────────────────────────────────────────────────
// Color palette
// ─────────────────────────────────────────────────────────────────────────────

// Palette holds all semantic color slots.
type Palette struct {
	// Backgrounds
	BgBase    canvas.Color // main window background
	BgSurface canvas.Color // card / panel background
	BgOverlay canvas.Color // modal backdrop tint

	// Borders & separators
	Border      canvas.Color
	BorderFocus canvas.Color

	// Text
	TextPrimary   canvas.Color
	TextSecondary canvas.Color
	TextDisabled  canvas.Color
	TextInverse   canvas.Color // text on accent background

	// Accent / interactive
	Accent       canvas.Color
	AccentHover  canvas.Color
	AccentActive canvas.Color

	// Semantic states
	Success canvas.Color
	Warning canvas.Color
	Error   canvas.Color
	Info    canvas.Color

	// Scrollbar
	ScrollTrack canvas.Color
	ScrollThumb canvas.Color
	ScrollHover canvas.Color
}

// ─────────────────────────────────────────────────────────────────────────────
// Typography
// ─────────────────────────────────────────────────────────────────────────────

// TypeScale maps logical roles to concrete TextStyle values.
type TypeScale struct {
	Display   canvas.TextStyle // hero headings
	H1        canvas.TextStyle
	H2        canvas.TextStyle
	H3        canvas.TextStyle
	Body      canvas.TextStyle
	BodySmall canvas.TextStyle
	Label     canvas.TextStyle // form labels, captions
	Code      canvas.TextStyle // monospace
	Hint      canvas.TextStyle // placeholder text
}

// ─────────────────────────────────────────────────────────────────────────────
// Spacing & radii
// ─────────────────────────────────────────────────────────────────────────────

// Spacing holds the spacing scale (multiples of a base unit).
type Spacing struct {
	Base float32 // base unit in px (default 4)
	XS   float32 // 4
	SM   float32 // 8
	MD   float32 // 16
	LG   float32 // 24
	XL   float32 // 32
	XXL  float32 // 48
}

func spacingFrom(base float32) Spacing {
	return Spacing{
		Base: base,
		XS:   base,
		SM:   base * 2,
		MD:   base * 4,
		LG:   base * 6,
		XL:   base * 8,
		XXL:  base * 12,
	}
}

// Radii holds border-radius tokens.
type Radii struct {
	None float32 // 0
	SM   float32 // 4
	MD   float32 // 8
	LG   float32 // 12
	XL   float32 // 16
	Full float32 // 9999 (pill)
}

// ─────────────────────────────────────────────────────────────────────────────
// Elevation / shadow
// ─────────────────────────────────────────────────────────────────────────────

// Shadow describes a single box-shadow layer.
type Shadow struct {
	OffsetX, OffsetY float32
	Blur             float32
	Color            canvas.Color
}

// ElevationScale maps 5 levels of elevation to shadow specs.
type ElevationScale [5]Shadow

// ─────────────────────────────────────────────────────────────────────────────
// Animation tokens
// ─────────────────────────────────────────────────────────────────────────────

// Motion holds duration constants (in milliseconds) for animations.
type Motion struct {
	Fast     float32 // 100 ms
	Normal   float32 // 200 ms
	Slow     float32 // 350 ms
	VerySlow float32 // 500 ms
}

// ─────────────────────────────────────────────────────────────────────────────
// Theme
// ─────────────────────────────────────────────────────────────────────────────

// Theme is the root design-token container.
type Theme struct {
	Name      string
	Colors    Palette
	Type      TypeScale
	Space     Spacing
	Radius    Radii
	Elevation ElevationScale
	Anim      Motion
}

// ─────────────────────────────────────────────────────────────────────────────
// Global registry
// ─────────────────────────────────────────────────────────────────────────────

var (
	mu       sync.RWMutex
	current  *Theme
	handlers []func(*Theme)
)

func init() { current = Dark() }

// Current returns the active theme. Safe to call from any goroutine.
func Current() *Theme {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Set activates a new theme and notifies all subscribers.
func Set(t *Theme) {
	mu.Lock()
	current = t
	cbs := make([]func(*Theme), len(handlers))
	copy(cbs, handlers)
	mu.Unlock()
	for _, cb := range cbs {
		cb(t)
	}
}

// OnChange registers a callback invoked whenever the theme changes.
// Returns an unsubscribe function.
func OnChange(fn func(*Theme)) func() {
	mu.Lock()
	handlers = append(handlers, fn)
	idx := len(handlers) - 1
	mu.Unlock()
	return func() {
		mu.Lock()
		handlers = append(handlers[:idx], handlers[idx+1:]...)
		mu.Unlock()
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Built-in themes
// ─────────────────────────────────────────────────────────────────────────────

// Dark returns the default dark (Catppuccin Mocha-inspired) theme.
func Dark() *Theme {
	sp := spacingFrom(4)
	return &Theme{
		Name: "dark",
		Colors: Palette{
			BgBase:        canvas.Hex("#11111B"),
			BgSurface:     canvas.Hex("#1E1E2E"),
			BgOverlay:     canvas.Color{R: 0, G: 0, B: 0, A: 0.6},
			Border:        canvas.Hex("#313244"),
			BorderFocus:   canvas.Hex("#89B4FA"),
			TextPrimary:   canvas.Hex("#CDD6F4"),
			TextSecondary: canvas.Hex("#BAC2DE"),
			TextDisabled:  canvas.Hex("#6C7086"),
			TextInverse:   canvas.Hex("#11111B"),
			Accent:        canvas.Hex("#89B4FA"),
			AccentHover:   canvas.Hex("#B4D0FF"),
			AccentActive:  canvas.Hex("#6699EE"),
			Success:       canvas.Hex("#A6E3A1"),
			Warning:       canvas.Hex("#F9E2AF"),
			Error:         canvas.Hex("#F38BA8"),
			Info:          canvas.Hex("#89DCEB"),
			ScrollTrack:   canvas.Hex("#181825"),
			ScrollThumb:   canvas.Hex("#45475A"),
			ScrollHover:   canvas.Hex("#585B70"),
		},
		Type:   buildTypeScale(canvas.Hex("#CDD6F4"), canvas.Hex("#6C7086")),
		Space:  sp,
		Radius: Radii{None: 0, SM: 4, MD: 8, LG: 12, XL: 16, Full: 9999},
		Elevation: ElevationScale{
			{0, 1, 2, canvas.Color{A: 0.10}},
			{0, 2, 6, canvas.Color{A: 0.15}},
			{0, 4, 12, canvas.Color{A: 0.20}},
			{0, 8, 24, canvas.Color{A: 0.25}},
			{0, 16, 40, canvas.Color{A: 0.30}},
		},
		Anim: Motion{Fast: 100, Normal: 200, Slow: 350, VerySlow: 500},
	}
}

// Light returns a light theme variant.
func Light() *Theme {
	t := Dark()
	t.Name = "light"
	t.Colors.BgBase = canvas.Hex("#EFF1F5")
	t.Colors.BgSurface = canvas.Hex("#FFFFFF")
	t.Colors.Border = canvas.Hex("#CCD0DA")
	t.Colors.BorderFocus = canvas.Hex("#1E66F5")
	t.Colors.TextPrimary = canvas.Hex("#4C4F69")
	t.Colors.TextSecondary = canvas.Hex("#6C6F85")
	t.Colors.TextDisabled = canvas.Hex("#ACB0BE")
	t.Colors.TextInverse = canvas.Hex("#FFFFFF")
	t.Colors.Accent = canvas.Hex("#1E66F5")
	t.Colors.AccentHover = canvas.Hex("#3D7EFF")
	t.Colors.AccentActive = canvas.Hex("#1553CC")
	t.Colors.ScrollTrack = canvas.Hex("#E6E9EF")
	t.Colors.ScrollThumb = canvas.Hex("#ACB0BE")
	t.Colors.ScrollHover = canvas.Hex("#8C8FA1")
	t.Type = buildTypeScale(canvas.Hex("#4C4F69"), canvas.Hex("#ACB0BE"))
	return t
}

// Custom creates a theme by mutating Dark() via a configure callback.
//
//	t := theme.Custom(func(t *theme.Theme) {
//	    t.Colors.Accent = canvas.Hex("#FF79C6")
//	    t.Radius.MD = 16
//	})
func Custom(configure func(*Theme)) *Theme {
	t := Dark()
	configure(t)
	return t
}

func buildTypeScale(primary, hint canvas.Color) TypeScale {
	mono := "monospace"
	return TypeScale{
		Display:   canvas.TextStyle{Color: primary, Size: 36, LineHeight: 1.1},
		H1:        canvas.TextStyle{Color: primary, Size: 28, LineHeight: 1.2},
		H2:        canvas.TextStyle{Color: primary, Size: 22, LineHeight: 1.25},
		H3:        canvas.TextStyle{Color: primary, Size: 18, LineHeight: 1.3},
		Body:      canvas.TextStyle{Color: primary, Size: 14, LineHeight: 1.5},
		BodySmall: canvas.TextStyle{Color: primary, Size: 12, LineHeight: 1.5},
		Label:     canvas.TextStyle{Color: primary, Size: 11, LineHeight: 1.4},
		Code:      canvas.TextStyle{Color: primary, Size: 13, LineHeight: 1.6, FontPath: mono},
		Hint:      canvas.TextStyle{Color: hint, Size: 14, LineHeight: 1.5},
	}
}
