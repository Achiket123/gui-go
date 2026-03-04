// Package canvas — text_layout.go
// High-performance, responsive text rendering utilities.
//
// Features:
//   - Word-wrap respecting available bounds (no overflow)
//   - Text overflow modes: clip, ellipsis ("..."), fade
//   - Horizontal alignment: Left, Center, Right, Justify
//   - Balanced line breaking (equal line lengths, like CSS `text-wrap: balance`)
//   - Minimum font-size scaling so text always fits inside its box
//   - DrawTextBox: the single entry-point that handles all of the above
package canvas

import (
	"strings"
	"unicode/utf8"
)

// ────────────────────────────────────────────────────────────────────────────
// Text alignment & overflow enums
// ────────────────────────────────────────────────────────────────────────────

// TextAlign controls horizontal alignment of each line.
type TextAlign int

const (
	TextAlignLeft    TextAlign = iota
	TextAlignCenter            // centre each line within the box
	TextAlignRight             // right-justify each line
	TextAlignJustify           // stretch words to fill each line (last line: left)
)

// TextOverflow defines what happens when text doesn't fit vertically.
type TextOverflow int

const (
	TextOverflowClip    TextOverflow = iota // hard-clip at the box bottom
	TextOverflowEllipsis                    // append "…" to the last visible line
	TextOverflowFade                        // (handled by caller; clip is used internally)
)

// ────────────────────────────────────────────────────────────────────────────
// TextBoxStyle — full style descriptor for DrawTextBox
// ────────────────────────────────────────────────────────────────────────────

// TextBoxStyle bundles all options for DrawTextBox.
type TextBoxStyle struct {
	Text      TextStyle
	Align     TextAlign
	Overflow  TextOverflow
	Balanced  bool    // attempt balanced line-breaking
	MinScale  float32 // minimum font-size multiplier before overflow kicks in (0 = 1.0)
	MaxLines  int     // 0 = unlimited
	Padding   EdgeInsets
}

// DefaultTextBoxStyle returns a sensible default.
func DefaultTextBoxStyle() TextBoxStyle {
	return TextBoxStyle{
		Text:     DefaultTextStyle(),
		Align:    TextAlignLeft,
		Overflow: TextOverflowEllipsis,
		MinScale: 1.0,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// textMeasurer — thin wrapper so we can swap in test stubs
// ────────────────────────────────────────────────────────────────────────────

type textMeasurer interface {
	measureWord(word string, style TextStyle) float32
	lineHeight(style TextStyle) float32
}

type canvasMeasurer struct{ c *Canvas }

func (m canvasMeasurer) measureWord(word string, style TextStyle) float32 {
	return m.c.MeasureText(word, style).W
}
func (m canvasMeasurer) lineHeight(style TextStyle) float32 {
	lh := style.LineHeight
	if lh == 0 {
		lh = 1.2
	}
	return style.Size * lh
}

// ────────────────────────────────────────────────────────────────────────────
// Word-wrap engine
// ────────────────────────────────────────────────────────────────────────────

// wrapLine breaks a single paragraph into a slice of lines that each fit within
// maxWidth. spaceW is the width of a single space character.
func wrapLine(words []string, maxWidth float32, style TextStyle, m textMeasurer) []string {
	if len(words) == 0 {
		return nil
	}
	spaceW := m.measureWord(" ", style)
	var lines []string
	var currentWords []string
	currentW := float32(0)

	for _, word := range words {
		ww := m.measureWord(word, style)
		// A single word wider than maxWidth: it has to go on its own line.
		if ww > maxWidth {
			if len(currentWords) > 0 {
				lines = append(lines, strings.Join(currentWords, " "))
				currentWords = currentWords[:0]
				currentW = 0
			}
			lines = append(lines, word)
			continue
		}
		extra := float32(0)
		if len(currentWords) > 0 {
			extra = spaceW
		}
		if currentW+extra+ww > maxWidth && len(currentWords) > 0 {
			lines = append(lines, strings.Join(currentWords, " "))
			currentWords = []string{word}
			currentW = ww
		} else {
			currentWords = append(currentWords, word)
			currentW += extra + ww
		}
	}
	if len(currentWords) > 0 {
		lines = append(lines, strings.Join(currentWords, " "))
	}
	return lines
}

// WrapText breaks text into lines that fit within maxWidth.
// It respects explicit \n characters.
func WrapText(text string, maxWidth float32, style TextStyle, m textMeasurer) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		out = append(out, wrapLine(words, maxWidth, style, m)...)
	}
	return out
}

// ────────────────────────────────────────────────────────────────────────────
// Balanced line breaking
// ────────────────────────────────────────────────────────────────────────────

// balanceLines attempts to re-wrap lines so the longest line is as short as
// possible (binary search on target width).
func balanceLines(text string, maxWidth float32, style TextStyle, m textMeasurer) []string {
	// Get a lower bound: average width / lines count.
	baseline := WrapText(text, maxWidth, style, m)
	if len(baseline) <= 1 {
		return baseline
	}

	lo := maxWidth * 0.3
	hi := maxWidth
	best := baseline

	// ~10 iterations is enough to converge within <1 px.
	for i := 0; i < 10; i++ {
		mid := (lo + hi) / 2
		candidate := WrapText(text, mid, style, m)
		if len(candidate) <= len(baseline) {
			best = candidate
			hi = mid
		} else {
			lo = mid
		}
	}
	return best
}

// ────────────────────────────────────────────────────────────────────────────
// Ellipsis helper
// ────────────────────────────────────────────────────────────────────────────

// truncateWithEllipsis shortens text until it (+ "…") fits within maxWidth.
func truncateWithEllipsis(text string, maxWidth float32, style TextStyle, m textMeasurer) string {
	ellipsis := "…"
	ellipsisW := m.measureWord(ellipsis, style)
	if ellipsisW > maxWidth {
		return ""
	}
	// Work backwards by rune.
	runes := []rune(text)
	for len(runes) > 0 {
		candidate := string(runes) + ellipsis
		if m.measureWord(candidate, style) <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ellipsis
}

// ────────────────────────────────────────────────────────────────────────────
// Line x-offset for alignment
// ────────────────────────────────────────────────────────────────────────────

func lineXOffset(lineW, boxW float32, align TextAlign) float32 {
	switch align {
	case TextAlignCenter:
		return (boxW - lineW) / 2
	case TextAlignRight:
		return boxW - lineW
	default:
		return 0
	}
}

// ────────────────────────────────────────────────────────────────────────────
// DrawTextBox — the main entry point
// ────────────────────────────────────────────────────────────────────────────

// DrawTextBox draws text inside rect using the given TextBoxStyle.
// It handles:
//   - Padding
//   - Word-wrapping (no overflow)
//   - Balanced line breaks
//   - Left / Center / Right / Justify alignment
//   - Ellipsis or clip when text overflows vertically
//   - MaxLines enforcement
//   - Automatic font-scale reduction (MinScale) to fit more text
func (c *Canvas) DrawTextBox(rect Rect, text string, s TextBoxStyle) {
	// Apply padding.
	inner := Rect{
		X: rect.X + s.Padding.Left,
		Y: rect.Y + s.Padding.Top,
		W: rect.W - s.Padding.Left - s.Padding.Right,
		H: rect.H - s.Padding.Top - s.Padding.Bottom,
	}
	if inner.W <= 0 || inner.H <= 0 {
		return
	}

	m := canvasMeasurer{c}
	style := s.Text

	// Auto-scale down the font if MinScale < 1 and text doesn't fit at full size.
	minScale := s.MinScale
	if minScale <= 0 {
		minScale = 1.0
	}
	if minScale < 1.0 {
		style = c.autoScaleStyle(text, inner, style, minScale, s.MaxLines, m)
	}

	lh := m.lineHeight(style)

	// Break into lines.
	var lines []string
	if s.Balanced {
		lines = balanceLines(text, inner.W, style, m)
	} else {
		lines = WrapText(text, inner.W, style, m)
	}

	// Enforce MaxLines.
	maxLines := s.MaxLines
	if maxLines <= 0 {
		// Compute how many lines physically fit.
		maxLines = int(inner.H / lh)
		if maxLines <= 0 {
			maxLines = 1
		}
	}

	truncated := len(lines) > maxLines
	if truncated {
		lines = lines[:maxLines]
	}

	// If ellipsis mode and we clipped, shorten the last line.
	if s.Overflow == TextOverflowEllipsis && truncated && len(lines) > 0 {
		last := lines[len(lines)-1]
		lines[len(lines)-1] = truncateWithEllipsis(last, inner.W, style, m)
	}

	// Draw each line.
	penY := inner.Y + style.Size // baseline of first line
	for i, line := range lines {
		if penY > inner.Y+inner.H {
			break
		}
		if line == "" {
			penY += lh
			continue
		}

		var drawLine string
		switch s.Align {
		case TextAlignJustify:
			// Last line of a paragraph (or last overall) is left-aligned.
			isLastLine := i == len(lines)-1
			if isLastLine {
				drawLine = line
				c.DrawText(inner.X, penY, drawLine, style)
			} else {
				c.drawJustifiedLine(line, inner.X, penY, inner.W, style, m)
			}
			penY += lh
			continue
		default:
			drawLine = line
		}

		lw := m.measureWord(drawLine, style)
		ox := lineXOffset(lw, inner.W, s.Align)

		// Clip to inner box before drawing (guard against any edge cases).
		c.Save()
		c.ClipRect(inner.X, inner.Y, inner.W, inner.H)
		c.DrawText(inner.X+ox, penY, drawLine, style)
		c.Restore()

		penY += lh
	}
}

// drawJustifiedLine stretches spaces in a line so it fills boxW exactly.
func (c *Canvas) drawJustifiedLine(line string, x, y, boxW float32, style TextStyle, m textMeasurer) {
	words := strings.Fields(line)
	if len(words) <= 1 {
		c.DrawText(x, y, line, style)
		return
	}
	// Total word width.
	totalWordW := float32(0)
	for _, w := range words {
		totalWordW += m.measureWord(w, style)
	}
	spaces := float32(len(words) - 1)
	extraPerSpace := (boxW - totalWordW) / spaces

	penX := x
	for i, w := range words {
		c.DrawText(penX, y, w, style)
		penX += m.measureWord(w, style)
		if i < len(words)-1 {
			penX += m.measureWord(" ", style) + extraPerSpace
		}
	}
}

// autoScaleStyle reduces style.Size proportionally until the wrapped text fits,
// but never below style.Size * minScale.
func (c *Canvas) autoScaleStyle(text string, inner Rect, style TextStyle, minScale float32, maxLines int, m textMeasurer) TextStyle {
	scaled := style
	minSize := style.Size * minScale
	for scaled.Size > minSize {
		lh := m.lineHeight(scaled)
		lines := WrapText(text, inner.W, scaled, m)
		limit := maxLines
		if limit <= 0 {
			limit = int(inner.H / lh)
		}
		if len(lines) <= limit && float32(len(lines))*lh <= inner.H {
			return scaled
		}
		scaled.Size -= 1
		if scaled.Size < minSize {
			scaled.Size = minSize
			break
		}
	}
	return scaled
}

// ────────────────────────────────────────────────────────────────────────────
// MeasureTextBox — compute height needed for text in a given width
// ────────────────────────────────────────────────────────────────────────────

// MeasureTextBox returns the pixel height needed to render text in a box of
// the given width using the provided style (useful for dynamic scroll heights).
func (c *Canvas) MeasureTextBox(text string, width float32, s TextBoxStyle) float32 {
	inner := width - s.Padding.Left - s.Padding.Right
	if inner <= 0 {
		return 0
	}
	m := canvasMeasurer{c}
	style := s.Text
	lh := m.lineHeight(style)
	var lines []string
	if s.Balanced {
		lines = balanceLines(text, inner, style, m)
	} else {
		lines = WrapText(text, inner, style, m)
	}
	return float32(len(lines))*lh + s.Padding.Top + s.Padding.Bottom
}

// ────────────────────────────────────────────────────────────────────────────
// Convenience: DrawCenteredText
// ────────────────────────────────────────────────────────────────────────────

// DrawCenteredText draws a single line of text perfectly centred (both axes)
// within rect. It does not wrap; long strings are clipped.
func (c *Canvas) DrawCenteredText(rect Rect, text string, style TextStyle) {
	sz := c.MeasureText(text, style)
	x := rect.X + (rect.W-sz.W)/2
	y := rect.Y + (rect.H+style.Size)/2 // approximate vertical centre via baseline
	c.Save()
	c.ClipRect(rect.X, rect.Y, rect.W, rect.H)
	c.DrawText(x, y, text, style)
	c.Restore()
}

// ────────────────────────────────────────────────────────────────────────────
// UTF-8 safe helpers
// ────────────────────────────────────────────────────────────────────────────

// RuneCount returns the number of Unicode code-points in s.
// Prefer this over len(s) when working with multi-byte text.
func RuneCount(s string) int { return utf8.RuneCountInString(s) }
