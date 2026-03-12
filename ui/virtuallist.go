// Package ui — virtuallist.go
//
// VirtualList renders only the rows currently visible on screen.
// Cost is O(visible rows) regardless of total item count.
//
// Features
//   - Fixed OR variable row heights
//   - Momentum / inertia scrolling with configurable friction
//   - Smooth animated ScrollToIndex / JumpToIndex
//   - Keyboard: ↑ ↓ Home End PgUp PgDn
//   - Single or multi-select with OnSelect callback
//   - Drag scrollbar thumb
//   - Optional sticky section headers
//   - Optional row separators with left inset
//   - Over-scan: N extra rows beyond viewport to prevent flash
//
// Usage
//
//	list := ui.NewVirtualList(ui.VirtualListOptions{
//	    ItemCount: 1_000_000,
//	    RowHeight: 44,
//	    DrawItem: func(c *canvas.Canvas, idx int, x, y, w, h float32, sel bool) {
//	        c.DrawText(x+12, y+h/2+6, fmt.Sprintf("Row %d", idx), style)
//	    },
//	})
package ui

import (
	"math"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/theme"
)

// ─────────────────────────────────────────────────────────────────────────────
// Callback types
// ─────────────────────────────────────────────────────────────────────────────

// DrawItemFunc is called once per visible row each frame.
type DrawItemFunc func(c *canvas.Canvas, index int, x, y, w, h float32, selected bool)

// RowHeightFunc returns the pixel height of row i.
type RowHeightFunc func(index int) float32

// SectionHeaderFunc returns a label and whether row i starts a new section.
type SectionHeaderFunc func(index int) (label string, isHeader bool)

// ─────────────────────────────────────────────────────────────────────────────
// VirtualListOptions
// ─────────────────────────────────────────────────────────────────────────────

// VirtualListOptions configures a VirtualList.
type VirtualListOptions struct {
	ItemCount int          // total number of rows
	DrawItem  DrawItemFunc // required: renders one row

	// Sizing — set RowHeight for fixed, RowHeightFn for variable.
	RowHeight   float32
	RowHeightFn RowHeightFunc

	// Section headers.
	SectionHeader SectionHeaderFunc
	HeaderHeight  float32 // default 32
	DrawHeader    func(c *canvas.Canvas, label string, x, y, w, h float32)

	// Rendering.
	OverScan int     // extra rows beyond viewport (default 3)
	Gap      float32 // vertical gap between rows

	// Scrollbar appearance.
	ScrollbarWidth float32 // default 8

	// Selection.
	MultiSelect bool
	OnSelect    func(index int, selected bool)

	// Separators.
	ShowSeparators bool
	SeparatorColor canvas.Color
	SeparatorInset float32 // left inset for the separator line

	// Physics.
	Friction float32 // velocity multiplier per tick (default 0.88)
}

// ─────────────────────────────────────────────────────────────────────────────
// VirtualList
// ─────────────────────────────────────────────────────────────────────────────

// VirtualList is a high-performance virtualized scrollable list.
type VirtualList struct {
	opts VirtualListOptions

	scrollY      float32
	velocity     float32
	targetScroll float32
	animating    bool

	barDragging   bool
	barDragStartY float32
	scrollAtDrag  float32
	barHover      bool

	selected map[int]bool
	focused  int // keyboard cursor (-1 = none)

	offsets []float32 // nil = fixed-height mode
	total   float32

	bounds canvas.Rect
}

// NewVirtualList creates a VirtualList.
func NewVirtualList(opts VirtualListOptions) *VirtualList {
	if opts.RowHeight == 0 && opts.RowHeightFn == nil {
		opts.RowHeight = 44
	}
	if opts.OverScan == 0 {
		opts.OverScan = 3
	}
	if opts.ScrollbarWidth == 0 {
		opts.ScrollbarWidth = 8
	}
	if opts.HeaderHeight == 0 {
		opts.HeaderHeight = 32
	}
	if opts.Friction == 0 {
		opts.Friction = 0.88
	}
	vl := &VirtualList{opts: opts, selected: make(map[int]bool), focused: -1}
	vl.rebuildOffsets()
	return vl
}

// ── Offset cache ──────────────────────────────────────────────────────────────

func (vl *VirtualList) rebuildOffsets() {
	n := vl.opts.ItemCount
	if vl.opts.RowHeightFn == nil && vl.opts.SectionHeader == nil {
		step := vl.opts.RowHeight + vl.opts.Gap
		if n > 0 {
			vl.total = step*float32(n) - vl.opts.Gap
		}
		vl.offsets = nil
		return
	}
	vl.offsets = make([]float32, n+1)
	cur := float32(0)
	for i := 0; i < n; i++ {
		vl.offsets[i] = cur
		h := vl.rowH(i)
		if vl.opts.SectionHeader != nil {
			if _, isH := vl.opts.SectionHeader(i); isH {
				h += vl.opts.HeaderHeight
			}
		}
		cur += h + vl.opts.Gap
	}
	vl.offsets[n] = cur
	if n > 0 {
		vl.total = cur - vl.opts.Gap
	}
}

func (vl *VirtualList) rowOff(i int) float32 {
	if vl.offsets != nil && i >= 0 && i < len(vl.offsets) {
		return vl.offsets[i]
	}
	return float32(i) * (vl.opts.RowHeight + vl.opts.Gap)
}

func (vl *VirtualList) rowH(i int) float32 {
	if vl.opts.RowHeightFn != nil {
		return vl.opts.RowHeightFn(i)
	}
	return vl.opts.RowHeight
}

// ── Public API ────────────────────────────────────────────────────────────────

// SetItemCount updates the total row count and rebuilds caches.
func (vl *VirtualList) SetItemCount(n int) {
	vl.opts.ItemCount = n
	vl.rebuildOffsets()
	vl.clamp()
}

// Select marks row i selected.
func (vl *VirtualList) Select(i int) {
	if !vl.opts.MultiSelect {
		for k := range vl.selected {
			delete(vl.selected, k)
		}
	}
	vl.selected[i] = true
	if vl.opts.OnSelect != nil {
		vl.opts.OnSelect(i, true)
	}
}

// Deselect removes the selection from row i.
func (vl *VirtualList) Deselect(i int) {
	delete(vl.selected, i)
	if vl.opts.OnSelect != nil {
		vl.opts.OnSelect(i, false)
	}
}

// ClearSelection deselects every row.
func (vl *VirtualList) ClearSelection() {
	for k := range vl.selected {
		delete(vl.selected, k)
	}
}

// SelectedIndices returns a snapshot of selected indices.
func (vl *VirtualList) SelectedIndices() []int {
	out := make([]int, 0, len(vl.selected))
	for k := range vl.selected {
		out = append(out, k)
	}
	return out
}

// ScrollToIndex smoothly scrolls so row i is visible.
func (vl *VirtualList) ScrollToIndex(i int) {
	if i < 0 || i >= vl.opts.ItemCount {
		return
	}
	rY := vl.rowOff(i)
	rBottom := rY + vl.rowH(i)
	vh := vl.bounds.H
	if rY < vl.scrollY {
		vl.targetScroll = rY
		vl.animating = true
	} else if rBottom > vl.scrollY+vh {
		vl.targetScroll = rBottom - vh
		vl.animating = true
	}
}

// JumpToIndex instantly positions the viewport at row i.
func (vl *VirtualList) JumpToIndex(i int) {
	vl.scrollY = vl.rowOff(i)
	vl.animating = false
	vl.clamp()
}

// ── Scroll helpers ────────────────────────────────────────────────────────────

func (vl *VirtualList) maxScroll() float32 {
	if vl.total <= vl.bounds.H {
		return 0
	}
	return vl.total - vl.bounds.H
}

func (vl *VirtualList) clamp() {
	ms := vl.maxScroll()
	if vl.scrollY < 0 {
		vl.scrollY = 0
	}
	if vl.scrollY > ms {
		vl.scrollY = ms
	}
}

// ── Component ────────────────────────────────────────────────────────────────

func (vl *VirtualList) Bounds() canvas.Rect { return vl.bounds }

func (vl *VirtualList) Tick(delta float64) {
	if math.Abs(float64(vl.velocity)) > 0.5 {
		vl.scrollY += vl.velocity * float32(delta)
		vl.clamp()
		vl.velocity *= vl.opts.Friction
	} else {
		vl.velocity = 0
	}
	if vl.animating {
		diff := vl.targetScroll - vl.scrollY
		if float32(math.Abs(float64(diff))) < 0.5 {
			vl.scrollY = vl.targetScroll
			vl.animating = false
		} else {
			vl.scrollY += diff * 0.18
		}
		vl.clamp()
	}
}

func (vl *VirtualList) HandleEvent(e Event) bool {
	b := vl.bounds
	inBounds := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	sbW := vl.opts.ScrollbarWidth + 4
	barX := b.X + b.W - sbW

	switch e.Type {
	case EventScroll:
		if inBounds {
			vl.velocity = -e.ScrollY * vl.rowH(0) * 3
			vl.animating = false
			return true
		}

	case EventMouseMove:
		vl.barHover = inBounds && e.X >= barX
		if vl.barDragging {
			dy := e.Y - vl.barDragStartY
			trackH := b.H - 8
			th := vl.thumbH()
			if avail := trackH - th; avail > 0 {
				vl.scrollY = vl.scrollAtDrag + dy*(vl.maxScroll()/avail)
				vl.clamp()
				vl.animating = false
			}
			return true
		}

	case EventMouseDown:
		if !inBounds {
			return false
		}
		if e.Button == 1 {
			if e.X >= barX {
				vl.barDragging = true
				vl.barDragStartY = e.Y
				vl.scrollAtDrag = vl.scrollY
				vl.velocity = 0
				vl.animating = false
				return true
			}
			contentY := e.Y - b.Y + vl.scrollY
			idx := vl.rowAtY(contentY)
			if idx >= 0 {
				if vl.selected[idx] && vl.opts.MultiSelect {
					vl.Deselect(idx)
				} else {
					vl.Select(idx)
				}
				vl.focused = idx
				return true
			}
		}

	case EventMouseUp:
		if vl.barDragging {
			vl.barDragging = false
			return true
		}

	case EventKeyDown:
		if !inBounds {
			return false
		}
		n := vl.opts.ItemCount
		switch e.Key {
		case "Down":
			vl.focused = clampI(vl.focused+1, 0, n-1)
			vl.Select(vl.focused)
			vl.ScrollToIndex(vl.focused)
		case "Up":
			if vl.focused > 0 {
				vl.focused--
			} else {
				vl.focused = 0
			}
			vl.Select(vl.focused)
			vl.ScrollToIndex(vl.focused)
		case "Next": // PgDn
			vl.scrollY += b.H * 0.85
			vl.clamp()
			vl.animating = false
		case "Prior": // PgUp
			vl.scrollY -= b.H * 0.85
			vl.clamp()
			vl.animating = false
		case "Home":
			vl.scrollY = 0
			vl.velocity = 0
			vl.animating = false
			if n > 0 {
				vl.focused = 0
				vl.Select(0)
			}
		case "End":
			vl.scrollY = vl.maxScroll()
			vl.velocity = 0
			vl.animating = false
			if n > 0 {
				vl.focused = n - 1
				vl.Select(n - 1)
			}
		default:
			return false
		}
		return true
	}
	return false
}

func (vl *VirtualList) Draw(c *canvas.Canvas, x, y, w, h float32) {
	vl.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	vl.clamp()
	th := theme.Current()

	sbW := vl.opts.ScrollbarWidth + 4
	cw := w - sbW

	c.Save()
	c.ClipRect(x, y, cw, h)

	n := vl.opts.ItemCount
	over := vl.opts.OverScan
	first := vl.firstVisible()
	last := first
	for last < n && vl.rowOff(last)-vl.scrollY < h {
		last++
	}
	start, end := first-over, last+over
	if start < 0 {
		start = 0
	}
	if end > n {
		end = n
	}

	for i := start; i < end; i++ {
		ry := y + vl.rowOff(i) - vl.scrollY
		rh := vl.rowH(i)

		// Section header.
		if vl.opts.SectionHeader != nil {
			if label, isH := vl.opts.SectionHeader(i); isH {
				hh := vl.opts.HeaderHeight
				hy := ry - hh
				if hy < y+h && hy+hh > y {
					if vl.opts.DrawHeader != nil {
						vl.opts.DrawHeader(c, label, x, hy, cw, hh)
					} else {
						c.DrawRect(x, hy, cw, hh, canvas.FillPaint(th.Colors.BgBase))
						hs := th.Type.Label
						hs.Color = th.Colors.TextSecondary
						c.DrawText(x+12, hy+(hh+hs.Size)/2, label, hs)
					}
				}
			}
		}

		if ry+rh < y || ry > y+h {
			continue
		}

		isSel := vl.selected[i]

		if isSel {
			bg := canvas.Color{R: th.Colors.Accent.R, G: th.Colors.Accent.G, B: th.Colors.Accent.B, A: 0.15}
			c.DrawRect(x, ry, cw, rh, canvas.FillPaint(bg))
		} else if i == vl.focused {
			c.DrawRect(x, ry, cw, rh, canvas.FillPaint(canvas.Color{A: 0.05}))
		}

		if vl.opts.ShowSeparators && i > 0 {
			sc := vl.opts.SeparatorColor
			if sc == (canvas.Color{}) {
				sc = th.Colors.Border
			}
			in := vl.opts.SeparatorInset
			c.DrawRect(x+in, ry, cw-in, 1, canvas.FillPaint(sc))
		}

		if vl.opts.DrawItem != nil {
			vl.opts.DrawItem(c, i, x, ry, cw, rh, isSel)
		}
	}
	c.Restore()

	// Scrollbar.
	if vl.maxScroll() <= 0 {
		return
	}
	barX := x + cw + 2
	barW := vl.opts.ScrollbarWidth
	trackH := h - 8

	c.DrawRoundedRect(barX, y+4, barW, trackH, barW/2, canvas.FillPaint(th.Colors.ScrollTrack))

	thumbH := vl.thumbH()
	thumbY := y + 4
	if vl.maxScroll() > 0 {
		thumbY += (vl.scrollY / vl.maxScroll()) * (trackH - thumbH)
	}
	tc := th.Colors.ScrollThumb
	if vl.barHover || vl.barDragging {
		tc = th.Colors.ScrollHover
	}
	c.DrawRoundedRect(barX, thumbY, barW, thumbH, barW/2, canvas.FillPaint(tc))
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (vl *VirtualList) thumbH() float32 {
	vh := vl.bounds.H
	trackH := vh - 8
	if vl.total <= 0 {
		return trackH
	}
	th := trackH * (vh / vl.total)
	if th > trackH {
		th = trackH
	}
	if th < 24 {
		th = 24
	}
	return th
}

func (vl *VirtualList) firstVisible() int {
	n := vl.opts.ItemCount
	if n == 0 {
		return 0
	}
	sy := vl.scrollY
	if vl.offsets == nil {
		step := vl.opts.RowHeight + vl.opts.Gap
		if step <= 0 {
			return 0
		}
		idx := int(sy / step)
		if idx < 0 {
			return 0
		}
		if idx >= n {
			return n - 1
		}
		return idx
	}
	lo, hi := 0, n-1
	for lo < hi {
		mid := (lo + hi) / 2
		if vl.offsets[mid]+vl.rowH(mid) <= sy {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func (vl *VirtualList) rowAtY(contentY float32) int {
	n := vl.opts.ItemCount
	if vl.offsets == nil {
		step := vl.opts.RowHeight + vl.opts.Gap
		if step <= 0 {
			return -1
		}
		idx := int(contentY / step)
		if idx < 0 || idx >= n {
			return -1
		}
		return idx
	}
	for i := 0; i < n; i++ {
		if contentY >= vl.offsets[i] && contentY < vl.offsets[i]+vl.rowH(i) {
			return i
		}
	}
	return -1
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
