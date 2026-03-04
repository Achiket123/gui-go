// Package styles — design tokens and shared drawing helpers for TaskFlow UI.
package styles

import (
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
	"github.com/achiket/gui-go/ui"
)

// ─── Palette ──────────────────────────────────────────────────────────────────

var (
	// Primary.
	Indigo    = canvas.Hex("#6366F1")
	IndigoHov = canvas.Hex("#818CF8")
	IndigoPrs = canvas.Hex("#4F46E5")

	// Surfaces (dark theme base).
	BgBase    = canvas.Hex("#0F0F14")
	BgSurface = canvas.Hex("#17171E")
	BgCard    = canvas.Hex("#1E1E28")
	BgInput   = canvas.Hex("#252530")

	// Text.
	TextPrimary   = canvas.Hex("#E2E8F0")
	TextSecondary = canvas.Hex("#94A3B8")
	TextMuted     = canvas.Hex("#64748B")

	// Borders.
	BorderSubtle = canvas.Hex("#252535")
	BorderNormal = canvas.Hex("#2E2E3E")

	// Status colours.
	ColBacklog    = canvas.Hex("#64748B")
	ColTodo       = canvas.Hex("#94A3B8")
	ColInProgress = canvas.Hex("#6366F1")
	ColInReview   = canvas.Hex("#F59E0B")
	ColDone       = canvas.Hex("#22C55E")
	ColCancelled  = canvas.Hex("#EF4444")

	// Priority colours.
	ColUrgent = canvas.Hex("#EF4444")
	ColHigh   = canvas.Hex("#F97316")
	ColMedium = canvas.Hex("#F59E0B")
	ColLow    = canvas.Hex("#22C55E")
	ColNone   = canvas.Hex("#64748B")

	White = canvas.Hex("#FFFFFF")
	Black = canvas.Hex("#000000")
)

// ─── Status colour map ────────────────────────────────────────────────────────

func StatusColor(s string) canvas.Color {
	m := map[string]canvas.Color{
		"backlog":     ColBacklog,
		"todo":        ColTodo,
		"in_progress": ColInProgress,
		"in_review":   ColInReview,
		"done":        ColDone,
		"cancelled":   ColCancelled,
	}
	if c, ok := m[s]; ok {
		return c
	}
	return ColBacklog
}

func PriorityColor(p string) canvas.Color {
	m := map[string]canvas.Color{
		"urgent": ColUrgent,
		"high":   ColHigh,
		"medium": ColMedium,
		"low":    ColLow,
		"none":   ColNone,
	}
	if c, ok := m[p]; ok {
		return c
	}
	return ColNone
}

// ─── Text styles ──────────────────────────────────────────────────────────────

var (
	H1    = canvas.TextStyle{Color: TextPrimary, Size: 24}
	H2    = canvas.TextStyle{Color: TextPrimary, Size: 18}
	H3    = canvas.TextStyle{Color: TextPrimary, Size: 15}
	Body  = canvas.TextStyle{Color: TextPrimary, Size: 13}
	Small = canvas.TextStyle{Color: TextSecondary, Size: 11}
	Tiny  = canvas.TextStyle{Color: TextMuted, Size: 10}
	Label = canvas.TextStyle{Color: TextSecondary, Size: 12}
	Badge = canvas.TextStyle{Color: White, Size: 10}
)

// ─── Radius ───────────────────────────────────────────────────────────────────

const (
	RadiusSM = float32(4)
	RadiusMD = float32(8)
	RadiusLG = float32(12)
	RadiusXL = float32(16)
)

// ─── TaskFlow theme ───────────────────────────────────────────────────────────

// ApplyTheme installs the TaskFlow dark theme globally.
func ApplyTheme() {
	th := theme.Dark()
	th.Colors.BgBase = BgBase
	th.Colors.BgSurface = BgSurface
	th.Colors.TextPrimary = TextPrimary
	th.Colors.TextSecondary = TextSecondary
	th.Colors.Accent = Indigo
	th.Colors.Border = BorderNormal

	th.Radius.SM = RadiusSM
	th.Radius.MD = RadiusMD
	th.Radius.LG = RadiusLG

	th.Type.Body = Body

	theme.Set(th)
}

// ─── Shared button styles ─────────────────────────────────────────────────────

func PrimaryButtonStyle() ui.ButtonStyle {
	s := ui.DefaultButtonStyle()
	s.Background = Indigo
	s.HoverColor = IndigoHov
	s.PressColor = IndigoPrs
	s.TextStyle = canvas.TextStyle{Color: White, Size: 13}
	s.BorderRadius = RadiusMD
	return s
}

func SecondaryButtonStyle() ui.ButtonStyle {
	s := ui.DefaultButtonStyle()
	s.Background = canvas.Hex("#00000000") // Transparent
	s.HoverColor = canvas.Hex("#2A2A38")
	s.PressColor = BgInput
	s.TextStyle = Body
	s.BorderColor = BorderNormal
	s.BorderWidth = 1.0
	s.BorderRadius = RadiusMD
	return s
}

func DangerButtonStyle() ui.ButtonStyle {
	s := ui.DefaultButtonStyle()
	s.Background = ColCancelled
	s.HoverColor = canvas.Hex("#F87171")
	s.PressColor = canvas.Hex("#DC2626")
	s.TextStyle = canvas.TextStyle{Color: White, Size: 13}
	s.BorderRadius = RadiusMD
	return s
}

// ─── Input style ──────────────────────────────────────────────────────────────

func InputStyle() ui.TextInputStyle {
	s := ui.DefaultTextInputStyle()
	s.Background = BgInput
	s.BorderColor = BorderNormal
	s.BorderWidth = 1.0
	s.FocusBorder = Indigo
	s.TextStyle = Body
	s.HintStyle = canvas.TextStyle{Color: TextMuted, Size: Body.Size}
	s.BorderRadius = RadiusMD
	return s
}

// ─── Card drawing helper ──────────────────────────────────────────────────────

// DrawCard draws a rounded card background with an optional left accent stripe.
func DrawCard(c *canvas.Canvas, x, y, w, h float32, accentColor *canvas.Color) {
	c.DrawRoundedRect(x, y, w, h, RadiusMD, canvas.FillPaint(BgCard))
	c.DrawRoundedRect(x, y, w, h, RadiusMD, canvas.StrokePaint(BorderSubtle, 1))
	if accentColor != nil {
		c.DrawRoundedRect(x, y, 3, h, RadiusSM, canvas.FillPaint(*accentColor))
	}
}

// DrawBadge draws a small colored pill badge.
func DrawBadge(c *canvas.Canvas, label string, bg canvas.Color, x, y float32) float32 {
	tw := c.MeasureText(label, Badge).W
	bw := tw + 12
	bh := float32(18)
	c.DrawRoundedRect(x, y, bw, bh, RadiusSM, canvas.FillPaint(bg))
	c.DrawText(x+6, y+13, label, Badge)
	return bw
}

// DrawAvatar draws a circular avatar with initials.
func DrawAvatar(c *canvas.Canvas, name string, color canvas.Color, x, y, size float32) {
	c.DrawCircle(x+size/2, y+size/2, size/2, canvas.FillPaint(color))
	initial := "?"
	if len(name) > 0 {
		initial = string(name[0])
	}
	ts := canvas.TextStyle{Color: White, Size: size * 0.38}
	tw := c.MeasureText(initial, ts).W
	c.DrawText(x+size/2-tw/2, y+size/2+ts.Size*0.35, initial, ts)
}

// AvatarColor returns a deterministic color from a name.
func AvatarColor(name string) canvas.Color {
	colors := []canvas.Color{
		canvas.Hex("#6366F1"), canvas.Hex("#8B5CF6"), canvas.Hex("#EC4899"),
		canvas.Hex("#F59E0B"), canvas.Hex("#10B981"), canvas.Hex("#3B82F6"),
		canvas.Hex("#F97316"), canvas.Hex("#14B8A6"),
	}
	idx := 0
	for _, r := range name {
		idx += int(r)
	}
	return colors[idx%len(colors)]
}
