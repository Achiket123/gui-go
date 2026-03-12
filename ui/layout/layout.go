package layout

import "github.com/achiket123/gui-go/canvas"

// Align specifies child alignment within a layout row/column.
type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
	AlignSpaceBetween
	AlignSpaceAround
	AlignSpaceEvenly
)

// Row distributes children horizontally within (x, y, w, h).
// widths: pass positive values for fixed widths, -1 for flex (fills remaining space).
// Returns a Rect for each child in order.
func Row(x, y, w, h float32, widths []float32, spacing float32, align Align) []canvas.Rect {
	totalFixed := float32(0)
	flexCount := 0
	n := len(widths)
	for _, fw := range widths {
		if fw < 0 {
			flexCount++
		} else {
			totalFixed += fw
		}
	}
	if n > 1 {
		totalFixed += spacing * float32(n-1)
	}
	flexW := float32(0)
	if flexCount > 0 {
		flexW = (w - totalFixed) / float32(flexCount)
	}

	rects := make([]canvas.Rect, n)
	cx := x
	for i, fw := range widths {
		cw := fw
		if fw < 0 {
			cw = flexW
		}
		var cy, ch float32
		switch align {
		case AlignCenter:
			ch = h * 0.6
			cy = y + (h-ch)/2
		case AlignStretch:
			cy = y
			ch = h
		default:
			cy = y
			ch = h
		}
		rects[i] = canvas.Rect{X: cx, Y: cy, W: cw, H: ch}
		cx += cw + spacing
	}
	return rects
}

// Column distributes children vertically within (x, y, w, h).
// heights: pass positive values for fixed heights, -1 for flex.
func Column(x, y, w, h float32, heights []float32, spacing float32, align Align) []canvas.Rect {
	totalFixed := float32(0)
	flexCount := 0
	n := len(heights)
	for _, fh := range heights {
		if fh < 0 {
			flexCount++
		} else {
			totalFixed += fh
		}
	}
	if n > 1 {
		totalFixed += spacing * float32(n-1)
	}
	flexH := float32(0)
	if flexCount > 0 {
		flexH = (h - totalFixed) / float32(flexCount)
	}

	rects := make([]canvas.Rect, n)
	cy := y
	for i, fh := range heights {
		ch := fh
		if fh < 0 {
			ch = flexH
		}
		rects[i] = canvas.Rect{X: x, Y: cy, W: w, H: ch}
		cy += ch + spacing
	}
	return rects
}

// Grid divides (x, y, w, h) into a uniform cols×rows grid.
func Grid(x, y, w, h float32, cols, rows int, spacing float32) []canvas.Rect {
	cw := (w - spacing*float32(cols-1)) / float32(cols)
	ch := (h - spacing*float32(rows-1)) / float32(rows)
	rects := make([]canvas.Rect, cols*rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			rects[row*cols+col] = canvas.Rect{
				X: x + float32(col)*(cw+spacing),
				Y: y + float32(row)*(ch+spacing),
				W: cw,
				H: ch,
			}
		}
	}
	return rects
}
