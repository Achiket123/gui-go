// Package ui — grid.go
//
// GridLayout implements a CSS-grid-style two-dimensional layout.
//
// Track sizing modes (mirrors CSS grid)
//
//	Fixed(px)   — exact pixel width/height
//	Fr(n)       — fractional share of remaining space after fixed tracks
//	Auto()      — sized to the minimum declared for that track (or 0)
//
// Usage
//
//	grid := ui.NewGridLayout(
//	    []ui.GridTrack{ui.Fixed(200), ui.Fr(1), ui.Fr(2)}, // 3 columns
//	    []ui.GridTrack{ui.Fixed(60),  ui.Auto(), ui.Fr(1)}, // 3 rows
//	    8, // gap (applies to both axes)
//	)
//	grid.Place(logo,    0, 0, 1, 1) // col 0, row 0, 1×1
//	grid.Place(content, 1, 0, 2, 2) // col 1, row 0, spans 2 cols and 2 rows
//	grid.Place(footer,  0, 2, 3, 1) // col 0, row 2, full-width
package ui

import "github.com/achiket/gui-go/canvas"

// ─────────────────────────────────────────────────────────────────────────────
// Track sizing
// ─────────────────────────────────────────────────────────────────────────────

// GridTrackKind classifies a column or row track.
type GridTrackKind int

const (
	TrackFixed GridTrackKind = iota // exact pixels
	TrackFr                         // fractional remainder
	TrackAuto                       // shrinks to MinSize
)

// GridTrack describes one column or row.
type GridTrack struct {
	Kind    GridTrackKind
	Pixels  float32 // TrackFixed: exact size
	Frac    float32 // TrackFr: fractional weight
	MinSize float32 // TrackAuto: minimum size (default 0)
}

// Fixed returns a fixed-size track.
func Fixed(px float32) GridTrack { return GridTrack{Kind: TrackFixed, Pixels: px} }

// Fr returns a track that takes a fractional share of remaining space.
func Fr(n float32) GridTrack { return GridTrack{Kind: TrackFr, Frac: n} }

// Auto returns a track sized to its MinSize (useful with later dynamic sizing).
func Auto() GridTrack { return GridTrack{Kind: TrackAuto} }

// ─────────────────────────────────────────────────────────────────────────────
// GridItem
// ─────────────────────────────────────────────────────────────────────────────

// GridItem places a component in the grid.
type GridItem struct {
	Child   Component
	Col     int // zero-based start column
	Row     int // zero-based start row
	ColSpan int // columns to span (default 1)
	RowSpan int // rows to span (default 1)
	Align   Alignment
}

// ─────────────────────────────────────────────────────────────────────────────
// GridLayout
// ─────────────────────────────────────────────────────────────────────────────

// GridLayout arranges children on a two-dimensional track grid.
type GridLayout struct {
	Cols   []GridTrack
	Rows   []GridTrack
	Gap    float32 // uniform gap; use ColGap/RowGap to override per-axis
	ColGap float32
	RowGap float32

	Items []GridItem

	// Resolved track sizes are computed fresh every Draw call.
	bounds canvas.Rect
}

// NewGridLayout creates a GridLayout with the given tracks and gap.
func NewGridLayout(cols, rows []GridTrack, gap float32) *GridLayout {
	return &GridLayout{Cols: cols, Rows: rows, Gap: gap}
}

// Place adds a child at (col, row) with the given span.
func (g *GridLayout) Place(child Component, col, row, colSpan, rowSpan int) {
	if colSpan <= 0 {
		colSpan = 1
	}
	if rowSpan <= 0 {
		rowSpan = 1
	}
	g.Items = append(g.Items, GridItem{
		Child: child, Col: col, Row: row,
		ColSpan: colSpan, RowSpan: rowSpan, Align: AlignmentTopLeft,
	})
}

// PlaceAligned is like Place but lets you specify cell alignment.
func (g *GridLayout) PlaceAligned(child Component, col, row, colSpan, rowSpan int, align Alignment) {
	g.Place(child, col, row, colSpan, rowSpan)
	g.Items[len(g.Items)-1].Align = align
}

func (g *GridLayout) Bounds() canvas.Rect { return g.bounds }

func (g *GridLayout) Tick(delta float64) {
	for _, item := range g.Items {
		if item.Child != nil {
			item.Child.Tick(delta)
		}
	}
}

func (g *GridLayout) HandleEvent(e Event) bool {
	// Iterate reverse so top-most items get events first.
	for i := len(g.Items) - 1; i >= 0; i-- {
		if g.Items[i].Child != nil && g.Items[i].Child.HandleEvent(e) {
			return true
		}
	}
	return false
}

// ── Track resolution ──────────────────────────────────────────────────────────

// resolveTracks computes pixel sizes for tracks given the available space
// (already minus all gaps).
func resolveTracks(tracks []GridTrack, available float32) []float32 {
	sizes := make([]float32, len(tracks))
	frSum := float32(0)
	used := float32(0)

	for i, t := range tracks {
		switch t.Kind {
		case TrackFixed:
			sizes[i] = t.Pixels
			used += t.Pixels
		case TrackAuto:
			sizes[i] = t.MinSize
			used += t.MinSize
		case TrackFr:
			frSum += t.Frac
		}
	}

	frSpace := available - used
	if frSpace < 0 {
		frSpace = 0
	}
	if frSum > 0 {
		for i, t := range tracks {
			if t.Kind == TrackFr {
				sizes[i] = frSpace * (t.Frac / frSum)
			}
		}
	}
	return sizes
}

// ── Draw ──────────────────────────────────────────────────────────────────────

func (g *GridLayout) Draw(c *canvas.Canvas, x, y, w, h float32) {
	g.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}

	nCols := len(g.Cols)
	nRows := len(g.Rows)
	if nCols == 0 || nRows == 0 {
		return
	}

	colGap := g.ColGap
	if colGap == 0 {
		colGap = g.Gap
	}
	rowGap := g.RowGap
	if rowGap == 0 {
		rowGap = g.Gap
	}

	// Available space after subtracting all gaps.
	colAvail := w - colGap*float32(nCols-1)
	rowAvail := h - rowGap*float32(nRows-1)

	colSizes := resolveTracks(g.Cols, colAvail)
	rowSizes := resolveTracks(g.Rows, rowAvail)

	// Compute column x-offsets.
	colX := make([]float32, nCols)
	cx := x
	for i, sz := range colSizes {
		colX[i] = cx
		cx += sz + colGap
	}

	// Compute row y-offsets.
	rowY := make([]float32, nRows)
	ry := y
	for i, sz := range rowSizes {
		rowY[i] = ry
		ry += sz + rowGap
	}

	// Draw each item.
	for _, item := range g.Items {
		if item.Child == nil {
			continue
		}
		col := clampI(item.Col, 0, nCols-1)
		row := clampI(item.Row, 0, nRows-1)
		endCol := clampI(item.Col+item.ColSpan, 1, nCols)
		endRow := clampI(item.Row+item.RowSpan, 1, nRows)

		// Cell bounding box (multi-track span).
		cellX := colX[col]
		cellY := rowY[row]

		cellW := float32(0)
		for i := col; i < endCol; i++ {
			cellW += colSizes[i]
			if i < endCol-1 {
				cellW += colGap
			}
		}
		cellH := float32(0)
		for i := row; i < endRow; i++ {
			cellH += rowSizes[i]
			if i < endRow-1 {
				cellH += rowGap
			}
		}

		item.Child.Draw(c, cellX, cellY, cellW, cellH)
	}
}
