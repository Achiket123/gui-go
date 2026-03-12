// Package ui — richtext.go
//
// RichText renders a paragraph of mixed-style inline text inside a bounding
// rectangle.  Supported span types:
//
//	Plain • Bold • Italic • BoldItalic • Code (monospace + bg)
//	Underline • Strikethrough • Link (clickable, hover color)
//	Colored (custom color) • Custom (arbitrary TextStyle)
//
// Builder API
//
//	rt := ui.NewRichText().
//	    Text("Hello ").
//	    Bold("world").
//	    Text("! Visit ").
//	    Link("example.com", "https://example.com").
//	    Text(" for more.")
//
//	rt.Draw(canvas, 20, 20, 400, 300)
//
// The widget word-wraps, respects newlines, supports left/center/right
// alignment, can clip or ellipsize overflowing lines, and exposes
// MeasureHeight so parent containers can allocate space correctly.
package ui

import (
	"strings"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/theme"
)

// ─────────────────────────────────────────────────────────────────────────────
// SpanKind
// ─────────────────────────────────────────────────────────────────────────────

// SpanKind classifies one text run.
type SpanKind int

const (
	SpanText SpanKind = iota
	SpanBold
	SpanItalic
	SpanBoldItalic
	SpanCode // monospace, highlighted background chip
	SpanUnderline
	SpanStrike
	SpanLink    // clickable; URL stored in Span.Meta
	SpanColored // custom color override
	SpanCustom  // caller-supplied TextStyle
)

// ─────────────────────────────────────────────────────────────────────────────
// Span
// ─────────────────────────────────────────────────────────────────────────────

// Span is one contiguous run of text with a uniform style.
type Span struct {
	Text  string
	Kind  SpanKind
	Color canvas.Color     // non-zero overrides span default color
	Style canvas.TextStyle // used when Kind == SpanCustom
	Meta  string           // URL for SpanLink
}

// ─────────────────────────────────────────────────────────────────────────────
// RichTextStyle
// ─────────────────────────────────────────────────────────────────────────────

// RichTextStyle holds all visual configuration for a RichText widget.
type RichTextStyle struct {
	Base       canvas.TextStyle
	Bold       canvas.TextStyle
	Italic     canvas.TextStyle
	BoldItalic canvas.TextStyle
	Code       canvas.TextStyle
	Link       canvas.TextStyle
	LinkHover  canvas.TextStyle
	CodeBg     canvas.Color

	UnderlineH float32 // underline stroke thickness (default 1)
	StrikeH    float32 // strikethrough stroke thickness (default 1)
	LineHeight float32 // line-height multiplier (default 1.5)

	Align    canvas.TextAlign
	MaxLines int  // 0 = unlimited
	Ellipsis bool // truncate last visible line with "…"
}

// DefaultRichTextStyle returns a theme-aware RichTextStyle.
func DefaultRichTextStyle() RichTextStyle {
	th := theme.Current()
	base := th.Type.Body
	code := th.Type.Code
	return RichTextStyle{
		Base:       base,
		Bold:       canvas.TextStyle{Color: base.Color, Size: base.Size, Bold: true},
		Italic:     canvas.TextStyle{Color: base.Color, Size: base.Size, Italic: true},
		BoldItalic: canvas.TextStyle{Color: base.Color, Size: base.Size, Bold: true, Italic: true},
		Code:       code,
		Link:       canvas.TextStyle{Color: th.Colors.Accent, Size: base.Size},
		LinkHover:  canvas.TextStyle{Color: th.Colors.AccentHover, Size: base.Size},
		CodeBg:     th.Colors.BgBase,
		UnderlineH: 1,
		StrikeH:    1,
		LineHeight: 1.5,
		Align:      canvas.TextAlignLeft,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// RichText widget
// ─────────────────────────────────────────────────────────────────────────────

// RichText is a multi-span, word-wrapped text component.
type RichText struct {
	Spans []Span
	Style RichTextStyle

	// OnLinkClick is called when the user clicks a SpanLink span.
	OnLinkClick func(url string)

	// Internal layout cache — invalidated whenever Spans changes.
	lines    []rtLine
	laidOutW float32 // width used for the last layout pass

	hoverURL string
	bounds   canvas.Rect
}

// NewRichText creates an empty RichText with the default theme style.
func NewRichText() *RichText {
	return &RichText{Style: DefaultRichTextStyle()}
}

// ── Builder helpers ───────────────────────────────────────────────────────────

func (rt *RichText) add(s Span) *RichText {
	rt.Spans = append(rt.Spans, s)
	rt.lines = nil // invalidate cache
	return rt
}

func (rt *RichText) Text(s string) *RichText   { return rt.add(Span{Text: s, Kind: SpanText}) }
func (rt *RichText) Bold(s string) *RichText   { return rt.add(Span{Text: s, Kind: SpanBold}) }
func (rt *RichText) Italic(s string) *RichText { return rt.add(Span{Text: s, Kind: SpanItalic}) }
func (rt *RichText) BoldItalic(s string) *RichText {
	return rt.add(Span{Text: s, Kind: SpanBoldItalic})
}
func (rt *RichText) Code(s string) *RichText      { return rt.add(Span{Text: s, Kind: SpanCode}) }
func (rt *RichText) Underline(s string) *RichText { return rt.add(Span{Text: s, Kind: SpanUnderline}) }
func (rt *RichText) Strike(s string) *RichText    { return rt.add(Span{Text: s, Kind: SpanStrike}) }
func (rt *RichText) Link(label, url string) *RichText {
	return rt.add(Span{Text: label, Kind: SpanLink, Meta: url})
}
func (rt *RichText) Colored(s string, col canvas.Color) *RichText {
	return rt.add(Span{Text: s, Kind: SpanColored, Color: col})
}
func (rt *RichText) Custom(s string, style canvas.TextStyle) *RichText {
	return rt.add(Span{Text: s, Kind: SpanCustom, Style: style})
}
func (rt *RichText) Newline() *RichText { return rt.add(Span{Text: "\n", Kind: SpanText}) }

// ── Component interface ───────────────────────────────────────────────────────

func (rt *RichText) Bounds() canvas.Rect { return rt.bounds }
func (rt *RichText) Tick(_ float64)      {}

func (rt *RichText) HandleEvent(e Event) bool {
	if e.Type == EventMouseMove {
		rt.hoverURL = rt.urlAt(e.X, e.Y)
		return rt.hoverURL != ""
	}
	if e.Type == EventMouseDown && e.Button == 1 {
		if url := rt.urlAt(e.X, e.Y); url != "" && rt.OnLinkClick != nil {
			rt.OnLinkClick(url)
			return true
		}
	}
	return false
}

// ── Layout ────────────────────────────────────────────────────────────────────

// rtGlyph is one positioned word token after layout.
type rtGlyph struct {
	text   string
	style  canvas.TextStyle
	kind   SpanKind
	meta   string  // URL for links
	x, y   float32 // baseline position relative to content origin
	w, h   float32
	codeBg canvas.Color
}

// rtLine is a slice of glyphs sharing the same baseline.
type rtLine struct {
	glyphs []rtGlyph
	y      float32
	height float32
	width  float32
}

// layout computes rt.lines for boxW / boxH.
func (rt *RichText) layout(c *canvas.Canvas, boxW, boxH float32) {
	if rt.lines != nil && rt.laidOutW == boxW {
		return // cache valid
	}
	rt.lines = nil
	rt.laidOutW = boxW
	if boxW <= 0 {
		return
	}

	lhMul := rt.Style.LineHeight
	if lhMul == 0 {
		lhMul = 1.5
	}

	type token struct {
		text   string
		style  canvas.TextStyle
		kind   SpanKind
		meta   string
		codeBg canvas.Color
		// If hardBreak == true this token triggers a line break.
		hardBreak bool
	}

	var tokens []token

	for _, span := range rt.Spans {
		st := rt.styleFor(span)
		// Handle embedded newlines.
		parts := strings.Split(span.Text, "\n")
		for pi, part := range parts {
			if pi > 0 {
				tokens = append(tokens, token{hardBreak: true})
			}
			words := strings.Fields(part)
			for wi, w := range words {
				if wi < len(words)-1 {
					w = w + " "
				}
				tokens = append(tokens, token{
					text:   w,
					style:  st,
					kind:   span.Kind,
					meta:   span.Meta,
					codeBg: rt.Style.CodeBg,
				})
			}
		}
	}

	maxLines := rt.Style.MaxLines
	lineH := rt.Style.Base.Size * lhMul
	penY := rt.Style.Base.Size // baseline of first line
	penX := float32(0)

	var curLine rtLine

	flushLine := func() bool {
		if maxLines > 0 && len(rt.lines) >= maxLines {
			return false // reached limit
		}
		curLine.y = penY
		curLine.height = lineH
		rt.lines = append(rt.lines, curLine)
		curLine = rtLine{}
		return true
	}

	for _, tok := range tokens {
		if tok.hardBreak {
			flushLine()
			penY += lineH
			penX = 0
			lineH = rt.Style.Base.Size * lhMul
			if penY > boxH {
				break
			}
			continue
		}

		w := c.MeasureText(tok.text, tok.style).W
		h := tok.style.Size * lhMul

		// Word-wrap.
		if penX+w > boxW && penX > 0 {
			if !flushLine() {
				break
			}
			penY += lineH
			penX = 0
			lineH = rt.Style.Base.Size * lhMul
			if penY > boxH {
				break
			}
		}

		g := rtGlyph{
			text:   tok.text,
			style:  tok.style,
			kind:   tok.kind,
			meta:   tok.meta,
			x:      penX,
			y:      penY,
			w:      w,
			h:      h,
			codeBg: tok.codeBg,
		}
		curLine.glyphs = append(curLine.glyphs, g)
		curLine.width += w
		penX += w
		if h > lineH {
			lineH = h
		}
	}

	// Flush remaining.
	if len(curLine.glyphs) > 0 && (maxLines == 0 || len(rt.lines) < maxLines) {
		curLine.y = penY
		curLine.height = lineH
		rt.lines = append(rt.lines, curLine)
	}
}

// ── Draw ──────────────────────────────────────────────────────────────────────

func (rt *RichText) Draw(c *canvas.Canvas, x, y, w, h float32) {
	rt.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	rt.layout(c, w, h)

	c.Save()
	c.ClipRect(x, y, w, h)

	nLines := len(rt.lines)
	for li, line := range rt.lines {
		xOff := float32(0)
		switch rt.Style.Align {
		case canvas.TextAlignCenter:
			xOff = (w - line.width) / 2
		case canvas.TextAlignRight:
			xOff = w - line.width
		}

		isLastLine := li == nLines-1

		for gi, g := range line.glyphs {
			gx := x + g.x + xOff
			gy := y + g.y

			// Ellipsis on last visible line.
			if isLastLine && rt.Style.Ellipsis && gi == len(line.glyphs)-1 {
				ellipsis := "…"
				eW := c.MeasureText(ellipsis, g.style).W
				if gx+g.w+eW > x+w {
					c.DrawText(gx+g.w-eW, gy, ellipsis, g.style)
					break
				}
			}

			// Code background chip.
			if g.kind == SpanCode {
				chipH := g.style.Size * 1.4
				c.DrawRoundedRect(gx-2, gy-g.style.Size, g.w+4, chipH, 3,
					canvas.FillPaint(g.codeBg))
			}

			// Resolve style (link hover).
			st := g.style
			if g.kind == SpanLink && g.meta != "" && g.meta == rt.hoverURL {
				st = rt.Style.LinkHover
			}

			c.DrawText(gx, gy, g.text, st)

			// Underline (links always underlined).
			if g.kind == SpanUnderline || g.kind == SpanLink {
				uh := rt.Style.UnderlineH
				if uh <= 0 {
					uh = 1
				}
				c.DrawRect(gx, gy+2, g.w, uh, canvas.FillPaint(st.Color))
			}

			// Strikethrough.
			if g.kind == SpanStrike {
				sh := rt.Style.StrikeH
				if sh <= 0 {
					sh = 1
				}
				c.DrawRect(gx, gy-g.style.Size*0.35, g.w, sh, canvas.FillPaint(st.Color))
			}
		}
	}
	c.Restore()
}

// MeasureHeight returns the pixel height needed to render all spans at boxW.
// Useful for dynamic layout (e.g. card height that fits its content).
func (rt *RichText) MeasureHeight(c *canvas.Canvas, boxW float32) float32 {
	rt.lines = nil // force full re-layout at new width
	rt.layout(c, boxW, 1e9)
	if len(rt.lines) == 0 {
		return 0
	}
	last := rt.lines[len(rt.lines)-1]
	return last.y + last.height
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (rt *RichText) urlAt(px, py float32) string {
	bx, by := rt.bounds.X, rt.bounds.Y
	for _, line := range rt.lines {
		for _, g := range line.glyphs {
			if g.kind != SpanLink {
				continue
			}
			gx := bx + g.x
			gy := by + g.y - g.style.Size
			if px >= gx && px <= gx+g.w && py >= gy && py <= gy+g.h {
				return g.meta
			}
		}
	}
	return ""
}

func (rt *RichText) styleFor(s Span) canvas.TextStyle {
	switch s.Kind {
	case SpanBold:
		return rt.Style.Bold
	case SpanItalic:
		return rt.Style.Italic
	case SpanBoldItalic:
		return rt.Style.BoldItalic
	case SpanCode:
		return rt.Style.Code
	case SpanLink:
		return rt.Style.Link
	case SpanCustom:
		return s.Style
	default: // SpanText, SpanUnderline, SpanStrike, SpanColored
		st := rt.Style.Base
		if s.Color != (canvas.Color{}) {
			st.Color = s.Color
		}
		return st
	}
}
