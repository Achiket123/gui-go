// Package ui — widgets_extra.go
//
// Additional widgets:
//
//	TabView    — tabbed content panels with animated indicator
//	Dropdown   — collapsed selector with popup list
//	Slider     — draggable range input with step snapping
//	Checkbox   — boolean tick-box with optional label
//	RadioGroup — mutually exclusive options (horizontal or vertical)
//	ToastManager — timed floating notifications (info/success/warning/error)
//	Navbar     — horizontal top navigation bar
//	Sidebar    — vertical navigation panel with active highlight
//	Splitter   — drag-resizable split pane
package ui

import (
	"time"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/theme"
)

// ═══════════════════════════════════════════════════════════════════════════════
// TabView
// ═══════════════════════════════════════════════════════════════════════════════

// Tab is one entry in a TabView.
type Tab struct {
	Label   string
	Content Component
}

// TabViewStyle configures a TabView.
type TabViewStyle struct {
	TabHeight      float32
	IndicatorH     float32
	ActiveText     canvas.Color
	InactiveText   canvas.Color
	IndicatorColor canvas.Color
	Background     canvas.Color
}

// DefaultTabViewStyle returns a theme-aware TabViewStyle.
func DefaultTabViewStyle() TabViewStyle {
	th := theme.Current()
	return TabViewStyle{
		TabHeight:      40,
		IndicatorH:     2,
		ActiveText:     th.Colors.TextPrimary,
		InactiveText:   th.Colors.TextSecondary,
		IndicatorColor: th.Colors.Accent,
		Background:     th.Colors.BgSurface,
	}
}

// TabView renders a tab strip and swaps between content components.
type TabView struct {
	Tabs     []Tab
	Active   int
	Style    TabViewStyle
	OnChange func(int)
	bounds   canvas.Rect
}

func NewTabView(tabs []Tab, style TabViewStyle) *TabView {
	return &TabView{Tabs: tabs, Style: style}
}

func (tv *TabView) Bounds() canvas.Rect { return tv.bounds }
func (tv *TabView) Tick(delta float64) {
	for _, t := range tv.Tabs {
		if t.Content != nil {
			t.Content.Tick(delta)
		}
	}
}

func (tv *TabView) HandleEvent(e Event) bool {
	b := tv.bounds
	n := len(tv.Tabs)
	if n == 0 {
		return false
	}
	tw := b.W / float32(n)
	if e.Type == EventMouseDown && e.Button == 1 &&
		e.Y >= b.Y && e.Y <= b.Y+tv.Style.TabHeight {
		idx := int((e.X - b.X) / tw)
		if idx >= 0 && idx < n && idx != tv.Active {
			tv.Active = idx
			if tv.OnChange != nil {
				tv.OnChange(idx)
			}
			return true
		}
	}
	if tv.Active >= 0 && tv.Active < n && tv.Tabs[tv.Active].Content != nil {
		return tv.Tabs[tv.Active].Content.HandleEvent(e)
	}
	return false
}

func (tv *TabView) Draw(c *canvas.Canvas, x, y, w, h float32) {
	tv.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := tv.Style.TabHeight
	n := len(tv.Tabs)
	if n == 0 {
		return
	}
	c.DrawRect(x, y, w, th, canvas.FillPaint(tv.Style.Background))
	tw := w / float32(n)

	for i, tab := range tv.Tabs {
		tx := x + float32(i)*tw
		active := i == tv.Active
		col := tv.Style.InactiveText
		if active {
			col = tv.Style.ActiveText
		}
		c.DrawCenteredText(canvas.Rect{X: tx, Y: y, W: tw, H: th}, tab.Label,
			canvas.TextStyle{Color: col, Size: 13})
		if active {
			c.DrawRect(tx+4, y+th-tv.Style.IndicatorH, tw-8, tv.Style.IndicatorH,
				canvas.FillPaint(tv.Style.IndicatorColor))
		}
	}
	c.DrawRect(x, y+th, w, 1, canvas.FillPaint(theme.Current().Colors.Border))

	if tv.Active >= 0 && tv.Active < n && tv.Tabs[tv.Active].Content != nil {
		tv.Tabs[tv.Active].Content.Draw(c, x, y+th+1, w, h-th-1)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Dropdown
// ═══════════════════════════════════════════════════════════════════════════════

// DropdownStyle configures a Dropdown.
type DropdownStyle struct {
	Background canvas.Color
	HoverBg    canvas.Color
	Border     canvas.Color
	TextStyle  canvas.TextStyle
	ItemHeight float32
	Radius     float32
	MaxVisible int
	Chevron    canvas.Color
}

// DefaultDropdownStyle returns a theme-aware DropdownStyle.
func DefaultDropdownStyle() DropdownStyle {
	th := theme.Current()
	return DropdownStyle{
		Background: th.Colors.BgSurface,
		HoverBg:    th.Colors.Border,
		Border:     th.Colors.Border,
		TextStyle:  th.Type.Body,
		ItemHeight: 36,
		Radius:     th.Radius.MD,
		MaxVisible: 6,
		Chevron:    th.Colors.TextSecondary,
	}
}

// Dropdown is a collapsed selector that expands a list of options.
type Dropdown struct {
	Options     []string
	Selected    int // -1 = none
	Placeholder string
	Style       DropdownStyle
	OnChange    func(index int, value string)

	open        bool
	hoverItem   int
	popupBounds canvas.Rect
	bounds      canvas.Rect
}

func NewDropdown(options []string, style DropdownStyle) *Dropdown {
	return &Dropdown{Options: options, Style: style, Selected: -1, hoverItem: -1}
}

func (d *Dropdown) Bounds() canvas.Rect { return d.bounds }
func (d *Dropdown) Tick(_ float64)      {}

func (d *Dropdown) HandleEvent(e Event) bool {
	b := d.bounds
	inTrigger := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H

	if e.Type == EventMouseDown && e.Button == 1 {
		if inTrigger {
			d.open = !d.open
			return true
		}
		if d.open {
			pb := d.popupBounds
			inPopup := e.X >= pb.X && e.X <= pb.X+pb.W && e.Y >= pb.Y && e.Y <= pb.Y+pb.H
			if inPopup {
				idx := int((e.Y - pb.Y) / d.Style.ItemHeight)
				if idx >= 0 && idx < len(d.Options) {
					d.Selected = idx
					d.open = false
					if d.OnChange != nil {
						d.OnChange(idx, d.Options[idx])
					}
				}
			} else {
				d.open = false
			}
			return true
		}
	}
	if e.Type == EventMouseMove && d.open {
		pb := d.popupBounds
		if e.X >= pb.X && e.X <= pb.X+pb.W && e.Y >= pb.Y && e.Y <= pb.Y+pb.H {
			d.hoverItem = int((e.Y - pb.Y) / d.Style.ItemHeight)
		} else {
			d.hoverItem = -1
		}
	}
	return d.open
}

func (d *Dropdown) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	s := d.Style

	// Trigger box.
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.FillPaint(s.Background))
	c.DrawRoundedRect(x, y, w, h, s.Radius, canvas.StrokePaint(s.Border, 1))

	label := d.Placeholder
	if label == "" {
		label = "Select…"
	}
	if d.Selected >= 0 && d.Selected < len(d.Options) {
		label = d.Options[d.Selected]
	}
	c.DrawText(x+12, y+(h+s.TextStyle.Size)/2, label, s.TextStyle)

	// Chevron.
	chX := x + w - 22
	chY := y + h/2
	p := canvas.StrokePaint(s.Chevron, 1.5)
	if d.open {
		c.DrawLine(chX, chY+3, chX+5, chY-3, p)
		c.DrawLine(chX+5, chY-3, chX+10, chY+3, p)
	} else {
		c.DrawLine(chX, chY-3, chX+5, chY+3, p)
		c.DrawLine(chX+5, chY+3, chX+10, chY-3, p)
	}

	if !d.open {
		return
	}

	// Popup.
	vis := s.MaxVisible
	if vis <= 0 || vis > len(d.Options) {
		vis = len(d.Options)
	}
	popH := float32(vis) * s.ItemHeight
	d.popupBounds = canvas.Rect{X: x, Y: y + h + 2, W: w, H: popH}
	pb := d.popupBounds

	// Shadow.
	for i := float32(1); i <= 4; i++ {
		c.DrawRoundedRect(pb.X+i, pb.Y+i, pb.W, pb.H, s.Radius, canvas.FillPaint(canvas.Color{A: 0.04}))
	}
	c.DrawRoundedRect(pb.X, pb.Y, pb.W, pb.H, s.Radius, canvas.FillPaint(s.Background))
	c.DrawRoundedRect(pb.X, pb.Y, pb.W, pb.H, s.Radius, canvas.StrokePaint(s.Border, 1))

	for i := 0; i < vis; i++ {
		iy := pb.Y + float32(i)*s.ItemHeight
		if i == d.hoverItem {
			c.DrawRect(pb.X, iy, pb.W, s.ItemHeight, canvas.FillPaint(s.HoverBg))
		}
		if i == d.Selected {
			c.DrawRect(pb.X+6, iy+(s.ItemHeight-2)/2, 4, 2,
				canvas.FillPaint(theme.Current().Colors.Accent))
		}
		c.DrawText(pb.X+18, iy+(s.ItemHeight+s.TextStyle.Size)/2, d.Options[i], s.TextStyle)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Slider
// ═══════════════════════════════════════════════════════════════════════════════

// RangeSliderStyle configures a RangeSlider.
type RangeSliderStyle struct {
	TrackColor  canvas.Color
	FillColor   canvas.Color
	ThumbColor  canvas.Color
	ThumbRadius float32
	TrackH      float32
}

// DefaultRangeSliderStyle returns a theme-aware RangeSliderStyle.
func DefaultRangeSliderStyle() RangeSliderStyle {
	th := theme.Current()
	return RangeSliderStyle{
		TrackColor:  th.Colors.Border,
		FillColor:   th.Colors.Accent,
		ThumbColor:  th.Colors.Accent,
		ThumbRadius: 8,
		TrackH:      4,
	}
}

// RangeSlider is a draggable range input. Value ∈ [Min, Max].
type RangeSlider struct {
	Min, Max float32
	Value    float32
	Step     float32 // 0 = continuous
	Style    RangeSliderStyle
	OnChange func(float32)

	dragging bool
	hovered  bool
	bounds   canvas.Rect
}

func NewRangeSlider(min, max, value float32, style RangeSliderStyle) *RangeSlider {
	return &RangeSlider{Min: min, Max: max, Value: value, Style: style}
}

func (s *RangeSlider) Bounds() canvas.Rect { return s.bounds }
func (s *RangeSlider) Tick(_ float64)      {}

func (s *RangeSlider) frac() float32 {
	if s.Max == s.Min {
		return 0
	}
	return (s.Value - s.Min) / (s.Max - s.Min)
}

func (s *RangeSlider) setFromX(px float32) {
	tr := s.Style.ThumbRadius
	b := s.bounds
	avail := b.W - tr*2
	if avail <= 0 {
		return
	}
	f := (px - b.X - tr) / avail
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	v := s.Min + f*(s.Max-s.Min)
	if s.Step > 0 {
		v = float32(int(v/s.Step+0.5)) * s.Step
	}
	if v < s.Min {
		v = s.Min
	}
	if v > s.Max {
		v = s.Max
	}
	if v != s.Value {
		s.Value = v
		if s.OnChange != nil {
			s.OnChange(v)
		}
	}
}

func (s *RangeSlider) HandleEvent(e Event) bool {
	b := s.bounds
	in := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	switch e.Type {
	case EventMouseMove:
		s.hovered = in
		if s.dragging {
			s.setFromX(e.X)
			return true
		}
	case EventMouseDown:
		if in && e.Button == 1 {
			s.dragging = true
			s.setFromX(e.X)
			return true
		}
	case EventMouseUp:
		if s.dragging {
			s.dragging = false
			return true
		}
	}
	return false
}

func (s *RangeSlider) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	tr := s.Style.ThumbRadius
	trackY := y + (h-s.Style.TrackH)/2

	c.DrawRoundedRect(x+tr, trackY, w-tr*2, s.Style.TrackH, s.Style.TrackH/2,
		canvas.FillPaint(s.Style.TrackColor))
	fillW := (w - tr*2) * s.frac()
	c.DrawRoundedRect(x+tr, trackY, fillW, s.Style.TrackH, s.Style.TrackH/2,
		canvas.FillPaint(s.Style.FillColor))

	thumbX := x + tr + fillW
	r := tr
	if s.dragging || s.hovered {
		r = tr * 1.2
	}
	c.DrawCircle(thumbX, y+h/2, r, canvas.FillPaint(s.Style.ThumbColor))
}

// ═══════════════════════════════════════════════════════════════════════════════
// Checkbox
// ═══════════════════════════════════════════════════════════════════════════════

// CheckboxStyle configures a Checkbox.
type CheckboxStyle struct {
	Size        float32
	Radius      float32
	Border      canvas.Color
	CheckedBg   canvas.Color
	UncheckedBg canvas.Color
	CheckColor  canvas.Color
	LabelStyle  canvas.TextStyle
	Gap         float32
}

// DefaultCheckboxStyle returns a theme-aware CheckboxStyle.
func DefaultCheckboxStyle() CheckboxStyle {
	th := theme.Current()
	return CheckboxStyle{
		Size:        18,
		Radius:      4,
		Border:      th.Colors.Border,
		CheckedBg:   th.Colors.Accent,
		UncheckedBg: th.Colors.BgSurface,
		CheckColor:  canvas.White,
		LabelStyle:  th.Type.Body,
		Gap:         8,
	}
}

// Checkbox is a boolean tick-box with an optional label.
type Checkbox struct {
	Checked  bool
	Label    string
	Style    CheckboxStyle
	OnChange func(bool)
	hovered  bool
	bounds   canvas.Rect
}

func NewCheckbox(label string, checked bool, style CheckboxStyle) *Checkbox {
	return &Checkbox{Label: label, Checked: checked, Style: style}
}

func (cb *Checkbox) Bounds() canvas.Rect { return cb.bounds }
func (cb *Checkbox) Tick(_ float64)      {}

func (cb *Checkbox) HandleEvent(e Event) bool {
	b := cb.bounds
	in := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	if e.Type == EventMouseMove {
		cb.hovered = in
	}
	if e.Type == EventMouseDown && e.Button == 1 && in {
		cb.Checked = !cb.Checked
		if cb.OnChange != nil {
			cb.OnChange(cb.Checked)
		}
		return true
	}
	return false
}

func (cb *Checkbox) Draw(c *canvas.Canvas, x, y, w, h float32) {
	cb.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	s := cb.Style
	boxY := y + (h-s.Size)/2
	bg := s.UncheckedBg
	if cb.Checked {
		bg = s.CheckedBg
	}
	c.DrawRoundedRect(x, boxY, s.Size, s.Size, s.Radius, canvas.FillPaint(bg))
	c.DrawRoundedRect(x, boxY, s.Size, s.Size, s.Radius, canvas.StrokePaint(s.Border, 1))
	if cb.Checked {
		p := canvas.StrokePaint(s.CheckColor, 2)
		mx, my := x+s.Size*0.2, boxY+s.Size*0.52
		c.DrawLine(mx, my, mx+s.Size*0.25, my+s.Size*0.25, p)
		c.DrawLine(mx+s.Size*0.25, my+s.Size*0.25, mx+s.Size*0.65, my-s.Size*0.28, p)
	}
	if cb.Label != "" {
		c.DrawText(x+s.Size+s.Gap, y+(h+s.LabelStyle.Size)/2, cb.Label, s.LabelStyle)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// RadioGroup
// ═══════════════════════════════════════════════════════════════════════════════

// RadioGroup renders mutually exclusive options.
type RadioGroup struct {
	Options  []string
	Selected int
	Vertical bool // true = stacked, false = side-by-side
	Style    CheckboxStyle
	OnChange func(index int, value string)
	bounds   canvas.Rect
}

func NewRadioGroup(options []string, selected int, vertical bool) *RadioGroup {
	return &RadioGroup{Options: options, Selected: selected, Vertical: vertical,
		Style: DefaultCheckboxStyle()}
}

func (rg *RadioGroup) Bounds() canvas.Rect { return rg.bounds }
func (rg *RadioGroup) Tick(_ float64)      {}

func (rg *RadioGroup) HandleEvent(e Event) bool {
	if e.Type != EventMouseDown || e.Button != 1 {
		return false
	}
	b := rg.bounds
	n := len(rg.Options)
	if rg.Vertical {
		ih := b.H / float32(n)
		idx := int((e.Y - b.Y) / ih)
		if idx >= 0 && idx < n && idx != rg.Selected {
			rg.Selected = idx
			if rg.OnChange != nil {
				rg.OnChange(idx, rg.Options[idx])
			}
			return true
		}
	} else {
		iw := b.W / float32(n)
		idx := int((e.X - b.X) / iw)
		if idx >= 0 && idx < n && idx != rg.Selected {
			rg.Selected = idx
			if rg.OnChange != nil {
				rg.OnChange(idx, rg.Options[idx])
			}
			return true
		}
	}
	return false
}

func (rg *RadioGroup) Draw(c *canvas.Canvas, x, y, w, h float32) {
	rg.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	n := len(rg.Options)
	s := rg.Style
	for i, opt := range rg.Options {
		var ix, iy, _, ih float32
		if rg.Vertical {
			ih = h / float32(n)
			ix, iy = x, y+float32(i)*ih
		} else {
			iw := w / float32(n)
			ix, iy = x+float32(i)*iw, y
			ih = h
		}
		r := s.Size / 2
		cx2, cy2 := ix+r, iy+ih/2
		bg := s.UncheckedBg
		if i == rg.Selected {
			bg = s.CheckedBg
		}
		c.DrawCircle(cx2, cy2, r, canvas.FillPaint(bg))
		c.DrawCircle(cx2, cy2, r, canvas.StrokePaint(s.Border, 1))
		if i == rg.Selected {
			c.DrawCircle(cx2, cy2, r*0.45, canvas.FillPaint(canvas.White))
		}
		if opt != "" {
			c.DrawText(ix+s.Size+s.Gap, iy+(ih+s.LabelStyle.Size)/2, opt, s.LabelStyle)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Splitter
// ═══════════════════════════════════════════════════════════════════════════════

// Splitter divides the layout into two panes separated by a draggable divider.
type Splitter struct {
	First, Second Component
	Vertical      bool    // true = left|right, false = top|bottom
	Split         float32 // 0–1, position of divider (default 0.5)
	DividerSize   float32
	DividerColor  canvas.Color

	dragging bool
	hovered  bool
	bounds   canvas.Rect
}

func NewSplitter(first, second Component, vertical bool) *Splitter {
	th := theme.Current()
	return &Splitter{
		First: first, Second: second, Vertical: vertical,
		Split: 0.5, DividerSize: 4, DividerColor: th.Colors.Border,
	}
}

func (sp *Splitter) Bounds() canvas.Rect { return sp.bounds }
func (sp *Splitter) Tick(delta float64) {
	if sp.First != nil {
		sp.First.Tick(delta)
	}
	if sp.Second != nil {
		sp.Second.Tick(delta)
	}
}

func (sp *Splitter) divRect() canvas.Rect {
	b := sp.bounds
	ds := sp.DividerSize
	if sp.Vertical {
		dx := b.X + b.W*sp.Split - ds/2
		return canvas.Rect{X: dx, Y: b.Y, W: ds, H: b.H}
	}
	dy := b.Y + b.H*sp.Split - ds/2
	return canvas.Rect{X: b.X, Y: dy, W: b.W, H: ds}
}

func (sp *Splitter) HandleEvent(e Event) bool {
	dr := sp.divRect()
	hit := e.X >= dr.X-4 && e.X <= dr.X+dr.W+4 && e.Y >= dr.Y-4 && e.Y <= dr.Y+dr.H+4
	b := sp.bounds

	switch e.Type {
	case EventMouseMove:
		sp.hovered = hit
		if sp.dragging {
			if sp.Vertical {
				sp.Split = (e.X - b.X) / b.W
			} else {
				sp.Split = (e.Y - b.Y) / b.H
			}
			if sp.Split < 0.05 {
				sp.Split = 0.05
			}
			if sp.Split > 0.95 {
				sp.Split = 0.95
			}
			return true
		}
	case EventMouseDown:
		if hit && e.Button == 1 {
			sp.dragging = true
			return true
		}
	case EventMouseUp:
		if sp.dragging {
			sp.dragging = false
			return true
		}
	}
	if sp.First != nil && sp.First.HandleEvent(e) {
		return true
	}
	if sp.Second != nil && sp.Second.HandleEvent(e) {
		return true
	}
	return false
}

func (sp *Splitter) Draw(c *canvas.Canvas, x, y, w, h float32) {
	sp.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	ds := sp.DividerSize
	dc := sp.DividerColor
	if sp.hovered || sp.dragging {
		dc = theme.Current().Colors.Accent
	}
	if sp.Vertical {
		fw := w*sp.Split - ds/2
		sx := x + fw + ds
		sw := w - fw - ds
		if sp.First != nil {
			sp.First.Draw(c, x, y, fw, h)
		}
		c.DrawRect(x+fw, y, ds, h, canvas.FillPaint(dc))
		if sp.Second != nil {
			sp.Second.Draw(c, sx, y, sw, h)
		}
	} else {
		fh := h*sp.Split - ds/2
		sy := y + fh + ds
		sh := h - fh - ds
		if sp.First != nil {
			sp.First.Draw(c, x, y, w, fh)
		}
		c.DrawRect(x, y+fh, w, ds, canvas.FillPaint(dc))
		if sp.Second != nil {
			sp.Second.Draw(c, x, sy, w, sh)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// ToastManager
// ═══════════════════════════════════════════════════════════════════════════════

// ToastLevel classifies a notification.
type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

type toastEntry struct {
	msg   string
	level ToastLevel
	born  time.Time
	ttl   time.Duration
	alpha float32
}

// ToastManager draws timed floating notifications in a screen corner.
type ToastManager struct {
	items      []*toastEntry
	maxVisible int
	bounds     canvas.Rect
}

func NewToastManager() *ToastManager { return &ToastManager{maxVisible: 4} }

// Show queues a new toast notification.
func (tm *ToastManager) Show(msg string, level ToastLevel, ttl time.Duration) {
	if ttl == 0 {
		ttl = 3 * time.Second
	}
	tm.items = append(tm.items, &toastEntry{msg: msg, level: level, born: time.Now(), ttl: ttl})
	if len(tm.items) > tm.maxVisible {
		tm.items = tm.items[len(tm.items)-tm.maxVisible:]
	}
}

func (tm *ToastManager) Bounds() canvas.Rect      { return tm.bounds }
func (tm *ToastManager) HandleEvent(_ Event) bool { return false }

func (tm *ToastManager) Tick(_ float64) {
	now := time.Now()
	alive := tm.items[:0]
	for _, t := range tm.items {
		age := now.Sub(t.born)
		fade := 300 * time.Millisecond
		if age < fade {
			t.alpha = float32(age) / float32(fade)
		} else if age < t.ttl-fade {
			t.alpha = 1
		} else {
			rem := t.ttl - age
			if rem < 0 {
				continue
			}
			t.alpha = float32(rem) / float32(fade)
		}
		alive = append(alive, t)
	}
	tm.items = alive
}

func (tm *ToastManager) Draw(c *canvas.Canvas, x, y, w, h float32) {
	tm.bounds = canvas.Rect{X: x, Y: y, W: w, H: h}
	th := theme.Current()
	tW := float32(320)
	tH := float32(52)
	gap := float32(8)
	mr, mb := float32(16), float32(16)

	for i, t := range tm.items {
		if t.alpha <= 0 {
			continue
		}
		tx := x + w - tW - mr
		ty := y + h - mb - float32(i+1)*(tH+gap)

		bg := th.Colors.BgSurface
		bg.A = t.alpha
		c.DrawRoundedRect(tx, ty, tW, tH, th.Radius.MD, canvas.FillPaint(bg))

		stripe := toastStripeColor(t.level, th)
		stripe.A = t.alpha
		c.DrawRoundedRect(tx, ty, 4, tH, 2, canvas.FillPaint(stripe))

		ts := th.Type.Body
		ts.Color.A = t.alpha
		c.DrawText(tx+16, ty+(tH+ts.Size)/2, t.msg, ts)
	}
}

func toastStripeColor(l ToastLevel, th *theme.Theme) canvas.Color {
	switch l {
	case ToastSuccess:
		return th.Colors.Success
	case ToastWarning:
		return th.Colors.Warning
	case ToastError:
		return th.Colors.Error
	default:
		return th.Colors.Info
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Navbar
// ═══════════════════════════════════════════════════════════════════════════════

// NavbarStyle configures a Navbar.
type NavbarStyle struct {
	Height     float32
	Background canvas.Color
	TitleStyle canvas.TextStyle
	Border     canvas.Color
}

// DefaultNavbarStyle returns a theme-aware NavbarStyle.
func DefaultNavbarStyle() NavbarStyle {
	th := theme.Current()
	return NavbarStyle{
		Height:     56,
		Background: th.Colors.BgSurface,
		TitleStyle: th.Type.H3,
		Border:     th.Colors.Border,
	}
}

// Navbar renders a horizontal top bar with a title, optional leading, and trailing slots.
type Navbar struct {
	Title    string
	Style    NavbarStyle
	Leading  Component
	Trailing []Component
	bounds   canvas.Rect
}

func NewNavbar(title string, style NavbarStyle) *Navbar {
	return &Navbar{Title: title, Style: style}
}

func (n *Navbar) Bounds() canvas.Rect { return n.bounds }

func (n *Navbar) Tick(delta float64) {
	if n.Leading != nil {
		n.Leading.Tick(delta)
	}
	for _, t := range n.Trailing {
		t.Tick(delta)
	}
}

func (n *Navbar) HandleEvent(e Event) bool {
	if n.Leading != nil && n.Leading.HandleEvent(e) {
		return true
	}
	for _, t := range n.Trailing {
		if t.HandleEvent(e) {
			return true
		}
	}
	return false
}

func (n *Navbar) Draw(c *canvas.Canvas, x, y, w, _ float32) {
	nh := n.Style.Height
	n.bounds = canvas.Rect{X: x, Y: y, W: w, H: nh}
	c.DrawRect(x, y, w, nh, canvas.FillPaint(n.Style.Background))
	c.DrawRect(x, y+nh, w, 1, canvas.FillPaint(n.Style.Border))

	if n.Leading != nil {
		n.Leading.Draw(c, x+8, y, nh, nh)
	}
	c.DrawCenteredText(canvas.Rect{X: x, Y: y, W: w, H: nh}, n.Title, n.Style.TitleStyle)

	trailX := x + w - 8
	for i := len(n.Trailing) - 1; i >= 0; i-- {
		trailX -= nh
		n.Trailing[i].Draw(c, trailX, y, nh, nh)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Sidebar
// ═══════════════════════════════════════════════════════════════════════════════

// SidebarItem is one entry in a Sidebar.
type SidebarItem struct {
	Label   string
	Icon    string // Unicode glyph / emoji
	OnClick func()
}

// SidebarStyle configures a Sidebar.
type SidebarStyle struct {
	Width      float32
	Background canvas.Color
	ItemHeight float32
	TextStyle  canvas.TextStyle
	ActiveBg   canvas.Color
	HoverBg    canvas.Color
	Border     canvas.Color
	AccentBar  float32
}

// DefaultSidebarStyle returns a theme-aware SidebarStyle.
func DefaultSidebarStyle() SidebarStyle {
	th := theme.Current()
	return SidebarStyle{
		Width:      220,
		Background: th.Colors.BgSurface,
		ItemHeight: 40,
		TextStyle:  th.Type.Body,
		ActiveBg:   canvas.Color{R: th.Colors.Accent.R, G: th.Colors.Accent.G, B: th.Colors.Accent.B, A: 0.15},
		HoverBg:    th.Colors.BgBase,
		Border:     th.Colors.Border,
		AccentBar:  3,
	}
}

// Sidebar is a vertical navigation panel.
type Sidebar struct {
	Items   []SidebarItem
	Active  int
	Style   SidebarStyle
	hovered int
	bounds  canvas.Rect
}

func NewSidebar(items []SidebarItem, style SidebarStyle) *Sidebar {
	return &Sidebar{Items: items, Style: style, Active: -1, hovered: -1}
}

func (s *Sidebar) Bounds() canvas.Rect { return s.bounds }
func (s *Sidebar) Tick(_ float64)      {}

func (s *Sidebar) HandleEvent(e Event) bool {
	b := s.bounds
	in := e.X >= b.X && e.X <= b.X+b.W && e.Y >= b.Y && e.Y <= b.Y+b.H
	if !in {
		s.hovered = -1
		return false
	}
	idx := int((e.Y - b.Y) / s.Style.ItemHeight)
	if e.Type == EventMouseMove {
		s.hovered = idx
	}
	if e.Type == EventMouseDown && e.Button == 1 && idx >= 0 && idx < len(s.Items) {
		s.Active = idx
		if s.Items[idx].OnClick != nil {
			s.Items[idx].OnClick()
		}
		return true
	}
	return false
}

func (s *Sidebar) Draw(c *canvas.Canvas, x, y, w, h float32) {
	sw := s.Style.Width
	s.bounds = canvas.Rect{X: x, Y: y, W: sw, H: h}
	c.DrawRect(x, y, sw, h, canvas.FillPaint(s.Style.Background))
	c.DrawRect(x+sw, y, 1, h, canvas.FillPaint(s.Style.Border))
	th := theme.Current()

	for i, item := range s.Items {
		iy := y + float32(i)*s.Style.ItemHeight
		if i == s.Active {
			c.DrawRect(x, iy, sw, s.Style.ItemHeight, canvas.FillPaint(s.Style.ActiveBg))
			c.DrawRect(x, iy, s.Style.AccentBar, s.Style.ItemHeight, canvas.FillPaint(th.Colors.Accent))
		} else if i == s.hovered {
			c.DrawRect(x, iy, sw, s.Style.ItemHeight, canvas.FillPaint(s.Style.HoverBg))
		}
		lx := x + 16
		if item.Icon != "" {
			c.DrawText(lx, iy+(s.Style.ItemHeight+s.Style.TextStyle.Size)/2, item.Icon, s.Style.TextStyle)
			lx += 28
		}
		c.DrawText(lx, iy+(s.Style.ItemHeight+s.Style.TextStyle.Size)/2, item.Label, s.Style.TextStyle)
	}
}
