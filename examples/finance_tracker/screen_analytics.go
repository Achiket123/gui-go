package main

import (
	"fmt"
	"math"
	"time"

	"github.com/achiket123/gui-go/animation"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

// AnalyticsScreen shows charts, trends, and spending breakdowns.
type AnalyticsScreen struct {
	ui.BaseScreen
	st    *Styles
	store *FinanceStore

	// Animation
	donutAnim *animation.AnimationController
	lineAnim  *animation.AnimationController
	barAnim   *animation.AnimationController
}

func NewAnalyticsScreen(st *Styles, store *FinanceStore) *AnalyticsScreen {
	a := &AnalyticsScreen{st: st, store: store}
	a.donutAnim = animation.NewController(1200 * time.Millisecond)
	a.donutAnim.Forward()
	a.lineAnim = animation.NewController(1500 * time.Millisecond)
	a.lineAnim.Forward()
	a.barAnim = animation.NewController(800 * time.Millisecond)
	a.barAnim.Forward()
	return a
}

func (a *AnalyticsScreen) OnEnter(nav *ui.Navigator) {
	a.Nav = nav
	a.donutAnim = animation.NewController(1000 * time.Millisecond)
	a.donutAnim.Forward()
	a.lineAnim = animation.NewController(1200 * time.Millisecond)
	a.lineAnim.Forward()
	a.barAnim = animation.NewController(800 * time.Millisecond)
	a.barAnim.Forward()
}

func (a *AnalyticsScreen) Tick(delta float64) {
	a.donutAnim.Tick(delta)
	a.lineAnim.Tick(delta)
	a.barAnim.Tick(delta)
}

func (a *AnalyticsScreen) HandleEvent(e ui.Event) bool {
	return false
}

func (a *AnalyticsScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	a.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})
	st := a.store.Get()
	currency := st.Currency

	donutT := float32(a.donutAnim.Value())
	lineT := float32(a.lineAnim.Value())
	barT := float32(a.barAnim.Value())

	// ── Header ────────────────────────────────────────────────────────────────
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colMantle))
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colSurface0))
	c.DrawText(x+24, y+38, "Analytics", a.st.H1)

	// Month label
	now := time.Now()
	monthStr := now.Format("January 2006")
	msz := c.MeasureText(monthStr, a.st.Caption)
	c.DrawText(x+w-msz.W-24, y+38, monthStr, a.st.Caption)

	const padX = float32(20)

	// ── LEFT: Spending Donut Chart ────────────────────────────────────────────
	leftW := w * 0.38
	donutCX := x + leftW/2
	donutCY := y + 80 + 130
	donutR := float32(90)
	donutThick := float32(30)

	c.DrawText(x+padX, y+76, "Spending by Category", a.st.H2)

	spending := a.store.SpendingByCategory()
	type catSlice struct {
		name  string
		value float64
		color canvas.Color
	}

	order := []string{"Food & Dining", "Housing", "Transport", "Entertainment", "Health", "Shopping", "Utilities"}
	var slices []catSlice
	total := 0.0
	for _, name := range order {
		v := spending[name]
		if v > 0 {
			slices = append(slices, catSlice{name, v, categoryColor(name)})
			total += v
		}
	}

	if total > 0 {
		angle := float32(-math.Pi / 2)
		for _, sl := range slices {
			sweep := float32(sl.value/total) * float32(math.Pi*2) * donutT
			drawDonutSegment(c, donutCX, donutCY, donutR, donutThick, angle, sweep, sl.color)
			angle += sweep
		}
	}

	// Donut center text
	totalStr := fmtMoney(total, currency)
	tStyle := canvas.TextStyle{Color: colText, Size: 14, FontPath: a.st.FontPath}
	tsz := c.MeasureText(totalStr, tStyle)
	c.DrawText(donutCX-tsz.W/2, donutCY+6, totalStr, tStyle)
	lbl := canvas.TextStyle{Color: colOverlay0, Size: 10, FontPath: a.st.FontPath}
	lblStr := "Total Spent"
	lsz := c.MeasureText(lblStr, lbl)
	c.DrawText(donutCX-lsz.W/2, donutCY+20, lblStr, lbl)

	// Legend
	legY := donutCY + donutR + 20
	for i, sl := range slices {
		lx := x + padX + float32(i%2)*leftW*0.5
		ly := legY + float32(i/2)*20
		c.DrawRoundedRect(lx, ly+1, 10, 10, 2, canvas.FillPaint(sl.color))
		catStr := sl.name
		if len(catStr) > 14 {
			catStr = catStr[:14] + "…"
		}
		pctStr := fmt.Sprintf("%s (%.0f%%)", catStr, sl.value/total*100)
		c.DrawText(lx+14, ly+11, pctStr, a.st.Caption)
	}

	// ── MIDDLE: Line chart (Net savings trend) ────────────────────────────────
	midX := x + leftW + padX
	midW := w * 0.36
	chartTop := y + 80
	chartH := (h - 80) / 2

	c.DrawText(midX, chartTop+6, "Income vs Expenses", a.st.H2)

	expenses6 := a.store.Last6MonthsExpenses()
	incomes6 := a.store.Last6MonthsIncome()
	labels := a.store.MonthLabels()

	maxVal := 1.0
	for i := 0; i < 6; i++ {
		if expenses6[i] > maxVal {
			maxVal = expenses6[i]
		}
		if incomes6[i] > maxVal {
			maxVal = incomes6[i]
		}
	}

	chartBodyX := midX + 10
	chartBodyY := chartTop + 32
	chartBodyW := midW - 20
	chartBodyH := chartH - 50

	// Grid lines
	for gi := 0; gi <= 4; gi++ {
		gy := chartBodyY + chartBodyH - float32(gi)/4*chartBodyH
		c.DrawLine(chartBodyX, gy, chartBodyX+chartBodyW, gy,
			canvas.StrokePaint(colSurface0.WithAlpha(0.6), 1))
		val := maxVal * float64(gi) / 4
		gridLbl := fmtMoney(val, currency)
		if len(gridLbl) > 8 {
			gridLbl = gridLbl[:8]
		}
		c.DrawText(chartBodyX-50, gy+4, gridLbl, a.st.Caption)
	}

	// Income line (animated draw)
	linePoints := func(data []float64, maxV float64) []canvas.Point {
		pts := make([]canvas.Point, 6)
		for i := 0; i < 6; i++ {
			px := chartBodyX + float32(i)/5*chartBodyW
			py := chartBodyY + chartBodyH - float32(data[i]/maxV)*chartBodyH
			pts[i] = canvas.Point{X: px, Y: py}
		}
		return pts
	}

	incPts := linePoints(incomes6, maxVal)
	expPts := linePoints(expenses6, maxVal)

	// Draw income line (segmented by animation progress)
	animPts := int(lineT * 5)
	if animPts > 5 {
		animPts = 5
	}
	for i := 0; i < animPts; i++ {
		c.DrawLine(incPts[i].X, incPts[i].Y, incPts[i+1].X, incPts[i+1].Y,
			canvas.StrokePaint(colGreen, 2))
		c.DrawCircle(incPts[i].X, incPts[i].Y, 4, canvas.FillPaint(colGreen))
	}
	if animPts >= 5 {
		c.DrawCircle(incPts[5].X, incPts[5].Y, 4, canvas.FillPaint(colGreen))
	}

	for i := 0; i < animPts; i++ {
		c.DrawLine(expPts[i].X, expPts[i].Y, expPts[i+1].X, expPts[i+1].Y,
			canvas.StrokePaint(colRed, 2))
		c.DrawCircle(expPts[i].X, expPts[i].Y, 4, canvas.FillPaint(colRed))
	}
	if animPts >= 5 {
		c.DrawCircle(expPts[5].X, expPts[5].Y, 4, canvas.FillPaint(colRed))
	}

	// Month labels
	for i := 0; i < 6; i++ {
		px := chartBodyX + float32(i)/5*chartBodyW
		lsz2 := c.MeasureText(labels[i], a.st.Caption)
		c.DrawText(px-lsz2.W/2, chartBodyY+chartBodyH+16, labels[i], a.st.Caption)
	}

	// Legend
	c.DrawLine(midX, chartTop+chartH-8, midX+20, chartTop+chartH-8,
		canvas.StrokePaint(colGreen, 2))
	c.DrawText(midX+24, chartTop+chartH-3, "Income", a.st.Caption)
	c.DrawLine(midX+80, chartTop+chartH-8, midX+100, chartTop+chartH-8,
		canvas.StrokePaint(colRed, 2))
	c.DrawText(midX+104, chartTop+chartH-3, "Expenses", a.st.Caption)

	// ── BOTTOM MIDDLE: Savings trend ─────────────────────────────────────────
	savingsY := chartTop + chartH + 10
	c.DrawText(midX, savingsY+6, "Monthly Savings", a.st.H2)

	savings := make([]float64, 6)
	for i := range savings {
		savings[i] = incomes6[i] - expenses6[i]
	}
	maxSav := 1.0
	for _, v := range savings {
		if math.Abs(v) > maxSav {
			maxSav = math.Abs(v)
		}
	}

	sbY := savingsY + 32
	sbH := h - (savingsY + 32) - 30
	midLineY := sbY + sbH/2

	c.DrawLine(chartBodyX, midLineY, chartBodyX+chartBodyW, midLineY,
		canvas.StrokePaint(colSurface1, 1))

	barSavW := chartBodyW / 7
	for i, sv := range savings {
		bx := chartBodyX + float32(i)*(barSavW+2)
		barPct := float32(math.Abs(sv)/maxSav) * (sbH/2 - 4) * barT
		col := colGreen
		if sv < 0 {
			col = colRed
			c.DrawRoundedRect(bx, midLineY, barSavW, barPct, 3, canvas.FillPaint(col))
		} else {
			c.DrawRoundedRect(bx, midLineY-barPct, barSavW, barPct, 3, canvas.FillPaint(col))
		}

		// Value label
		valStr := fmtMoney(math.Abs(sv), currency)
		if len(valStr) > 8 {
			valStr = valStr[:8]
		}
		valStyle := canvas.TextStyle{Color: col, Size: 9, FontPath: a.st.FontPath}
		c.DrawText(bx, midLineY-barPct-2, valStr, valStyle)

		lsz2 := c.MeasureText(labels[i], a.st.Caption)
		c.DrawText(bx+barSavW/2-lsz2.W/2, sbY+sbH, labels[i], a.st.Caption)
	}

	// ── RIGHT: Top spending categories bar ────────────────────────────────────
	rightX := x + leftW + midW + padX*3
	rightW := w - leftW - midW - padX*4 - 10

	c.DrawText(rightX, y+76, "Top Categories", a.st.H2)

	type catRank struct {
		name  string
		value float64
	}
	var catRanks []catRank
	for name, val := range spending {
		catRanks = append(catRanks, catRank{name, val})
	}
	// Sort descending
	for i := 0; i < len(catRanks); i++ {
		for j := i + 1; j < len(catRanks); j++ {
			if catRanks[j].value > catRanks[i].value {
				catRanks[i], catRanks[j] = catRanks[j], catRanks[i]
			}
		}
	}

	maxCatVal := 1.0
	if len(catRanks) > 0 {
		maxCatVal = catRanks[0].value
	}

	const catRowH = float32(46)
	for i, cr := range catRanks {
		if i >= 8 {
			break
		}
		ry := y + 100 + float32(i)*catRowH
		catCol := categoryColor(cr.name)

		c.DrawText(rightX, ry+14, cr.name, a.st.BodyBold)

		barMaxW := rightW - 10
		fillPct := float32(cr.value/maxCatVal) * barT

		c.DrawRoundedRect(rightX, ry+20, barMaxW, 10, 5, canvas.FillPaint(colSurface0))
		if fillPct > 0 {
			c.DrawRoundedRect(rightX, ry+20, barMaxW*fillPct, 10, 5, canvas.FillPaint(catCol))
		}

		amtStr := fmtMoney(cr.value, currency)
		amtStyle := canvas.TextStyle{Color: catCol, Size: 11, FontPath: a.st.FontPath}
		asz := c.MeasureText(amtStr, amtStyle)
		c.DrawText(rightX+barMaxW-asz.W, ry+14, amtStr, amtStyle)
	}

	// ── Insight cards ──────────────────────────────────────────────────────────
	insightY := y + 100 + 8*catRowH + 10
	if insightY < y+h-80 {
		income := a.store.MonthlyIncome()
		expenses := a.store.MonthlyExpenses()
		savingsRate := 0.0
		if income > 0 {
			savingsRate = (income - expenses) / income * 100
		}

		insightStr := fmt.Sprintf("Savings rate: %.1f%%", savingsRate)
		insightCol := colGreen
		if savingsRate < 0 {
			insightCol = colRed
			insightStr = "You're overspending this month!"
		} else if savingsRate < 10 {
			insightCol = colYellow
			insightStr = fmt.Sprintf("Low savings rate: %.1f%% — try to cut expenses", savingsRate)
		}

		c.DrawRoundedRect(rightX-4, insightY, rightW+8, 44, 8, canvas.FillPaint(insightCol.WithAlpha(0.12)))
		c.DrawRoundedRect(rightX-4, insightY, rightW+8, 44, 8, canvas.StrokePaint(insightCol.WithAlpha(0.4), 1))
		c.DrawText(rightX+4, insightY+14, "💡 Insight", canvas.TextStyle{Color: insightCol, Size: 11, FontPath: a.st.FontPath})
		iStyle := canvas.TextStyle{Color: insightCol, Size: 12, FontPath: a.st.FontPath}
		c.DrawText(rightX+4, insightY+32, insightStr, iStyle)
	}
	_ = barT
}
