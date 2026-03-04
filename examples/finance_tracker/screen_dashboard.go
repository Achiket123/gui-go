package main

import (
	"fmt"
	"math"
	"time"

	"github.com/achiket/gui-go/animation"
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

// DashboardScreen shows a high-level financial overview.
type DashboardScreen struct {
	ui.BaseScreen
	st     *Styles
	store  *FinanceStore
	toasts *ui.ToastManager

	// Widgets
	addBtn *ui.Button
	modal  *ui.ModalManager

	// Animation
	barAnim  *animation.AnimationController
	cardAnim *animation.AnimationController
	animDone bool

	// State
	barProgress float32
}

func NewDashboardScreen(st *Styles, store *FinanceStore, toasts *ui.ToastManager) *DashboardScreen {
	d := &DashboardScreen{st: st, store: store, toasts: toasts}

	btnStyle := ui.DefaultButtonStyle()
	btnStyle.Background = colAccent
	btnStyle.HoverColor = canvas.Hex("#A6C8FF")
	btnStyle.PressColor = canvas.Hex("#6A9FDB")
	btnStyle.TextStyle = canvas.TextStyle{Color: colBase, Size: 13, FontPath: st.FontPath}
	btnStyle.BorderRadius = 8

	d.addBtn = ui.NewButton("+ Add Transaction", func() {
		d.openAddModal()
	})
	d.addBtn.Style = btnStyle

	d.modal = ui.NewModalManager()

	// Bar chart animate-in
	d.barAnim = animation.NewController(1200 * time.Millisecond)
	d.barAnim.Forward()

	d.cardAnim = animation.NewController(600 * time.Millisecond)
	d.cardAnim.Forward()

	return d
}

func (d *DashboardScreen) OnEnter(nav *ui.Navigator) {
	d.Nav = nav
	// Re-trigger animations on every visit
	d.barAnim = animation.NewController(1000 * time.Millisecond)
	d.barAnim.Forward()
}

func (d *DashboardScreen) Tick(delta float64) {
	d.addBtn.Tick(delta)
	d.barAnim.Tick(delta)
	d.cardAnim.Tick(delta)
	d.modal.Tick(delta)
}

func (d *DashboardScreen) HandleEvent(e ui.Event) bool {
	if d.modal.HandleEvent(e) {
		return true
	}
	return d.addBtn.HandleEvent(e)
}

func (d *DashboardScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	d.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})
	st := d.store.Get()
	currency := st.Currency

	animT := float32(d.barAnim.Value())
	cardT := float32(d.cardAnim.Value())
	_ = cardT

	// ── Header ────────────────────────────────────────────────────────────────
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colMantle))
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colSurface0))

	c.DrawText(x+24, y+38, "Dashboard", d.st.H1)

	// Live date/time
	now := time.Now()
	timeStr := now.Format("Mon, Jan 2, 2006  15:04")
	tsz := c.MeasureText(timeStr, d.st.Caption)
	c.DrawText(x+w-tsz.W-24, y+38, timeStr, d.st.Caption)

	// Add button
	d.addBtn.Draw(c, x+w-200, y+14, 176, 36)

	// ── Summary Cards ─────────────────────────────────────────────────────────
	const cardY = float32(80)
	const cardH = float32(96)
	const padX = float32(20)
	cardW := (w - padX*5) / 4

	cards := []struct {
		title string
		value string
		sub   string
		color canvas.Color
		trend string
	}{
		{
			"Net Worth",
			fmtMoney(d.store.TotalBalance(), currency),
			"Across all accounts",
			colAccent,
			"",
		},
		{
			"Monthly Income",
			fmtMoney(d.store.MonthlyIncome(), currency),
			"This month",
			colGreen,
			"",
		},
		{
			"Monthly Expenses",
			fmtMoney(d.store.MonthlyExpenses(), currency),
			"This month",
			colRed,
			"",
		},
		{
			"Net Savings",
			fmtMoney(d.store.NetSavings(), currency),
			"Income - Expenses",
			colMauve,
			"",
		},
	}

	for i, card := range cards {
		cx := x + padX + float32(i)*(cardW+padX)
		cy := y + cardY

		// Card background
		c.DrawRoundedRect(cx, cy, cardW, cardH, 12,
			canvas.FillPaint(colMantle))
		c.DrawRoundedRect(cx, cy, cardW, cardH, 12,
			canvas.StrokePaint(colSurface0, 1))

		// Left accent bar
		c.DrawRoundedRect(cx, cy+8, 3, cardH-16, 2,
			canvas.FillPaint(card.color))

		// Title
		c.DrawText(cx+16, cy+26, card.title, d.st.Caption)

		// Value (large)
		valStyle := canvas.TextStyle{Color: card.color, Size: 20, FontPath: d.st.FontPath}
		c.DrawText(cx+16, cy+56, card.value, valStyle)

		// Sub
		c.DrawText(cx+16, cy+76, card.sub, d.st.Caption)
	}

	// ── Accounts Bar ──────────────────────────────────────────────────────────
	const accY = float32(200)
	c.DrawText(x+padX, y+accY, "Accounts", d.st.H2)

	accW := (w - padX*5) / 4
	for i, acc := range st.Accounts {
		ax := x + padX + float32(i)*(accW+padX)
		ay := y + accY + 28
		aH := float32(70)

		col := categoryColor(acc.Name)
		if acc.Color != "" {
			col = canvas.Hex(acc.Color)
		}

		c.DrawRoundedRect(ax, ay, accW, aH, 8, canvas.FillPaint(colMantle))
		c.DrawRoundedRect(ax, ay, accW, aH, 8, canvas.StrokePaint(colSurface0, 1))

		// Circle icon
		c.DrawCircle(ax+20, ay+aH/2, 12, canvas.FillPaint(col.WithAlpha(0.2)))
		c.DrawCircle(ax+20, ay+aH/2, 12, canvas.StrokePaint(col, 1.5))
		iconStyle := canvas.TextStyle{Color: col, Size: 10, FontPath: d.st.FontPath}
		icon := string([]rune(acc.Name)[0:1])
		isz := c.MeasureText(icon, iconStyle)
		c.DrawText(ax+20-isz.W/2, ay+aH/2+4, icon, iconStyle)

		// Name
		nameStyle := canvas.TextStyle{Color: colText, Size: 12, FontPath: d.st.FontPath}
		c.DrawText(ax+38, ay+22, acc.Name, nameStyle)

		// Balance
		typeLabel := acc.Type
		balColor := colGreen
		balStr := fmtMoney(acc.Balance, currency)
		if acc.Type == "credit" {
			balColor = colRed
			balStr = "-" + fmtMoney(acc.Balance, currency)
		}
		_ = typeLabel
		balStyle := canvas.TextStyle{Color: balColor, Size: 13, FontPath: d.st.FontPath}
		c.DrawText(ax+38, ay+48, balStr, balStyle)
	}

	// ── Spending Bar Chart ────────────────────────────────────────────────────
	const chartY = float32(320)
	chartH := h - chartY - 20
	chartW := w*0.55 - padX*2

	c.DrawText(x+padX, y+chartY, "6-Month Overview", d.st.H2)

	// Chart area
	chartX := x + padX
	chartBodyY := y + chartY + 30
	chartBodyH := chartH - 60

	expenses := d.store.Last6MonthsExpenses()
	incomes := d.store.Last6MonthsIncome()
	labels := d.store.MonthLabels()

	maxVal := 1.0
	for i := range expenses {
		if expenses[i] > maxVal {
			maxVal = expenses[i]
		}
		if incomes[i] > maxVal {
			maxVal = incomes[i]
		}
	}

	barGroupW := (chartW - 40) / 6
	barW2 := barGroupW * 0.35
	barSpacing := barGroupW * 0.08

	for i := 0; i < 6; i++ {
		gx := chartX + 30 + float32(i)*barGroupW

		// Expense bar
		expH := float32(expenses[i]/maxVal) * chartBodyH * animT
		expY := chartBodyY + chartBodyH - expH
		c.DrawRoundedRect(gx, expY, barW2, expH, 3,
			canvas.FillPaint(colRed.WithAlpha(0.8)))

		// Income bar
		incH := float32(incomes[i]/maxVal) * chartBodyH * animT
		incY := chartBodyY + chartBodyH - incH
		c.DrawRoundedRect(gx+barW2+barSpacing, incY, barW2, incH, 3,
			canvas.FillPaint(colGreen.WithAlpha(0.8)))

		// Month label
		lsz := c.MeasureText(labels[i], d.st.Caption)
		c.DrawText(gx+barGroupW/2-lsz.W/2, chartBodyY+chartBodyH+16, labels[i], d.st.Caption)
	}

	// Y-axis line
	c.DrawLine(chartX+28, chartBodyY, chartX+28, chartBodyY+chartBodyH,
		canvas.StrokePaint(colSurface1, 1))
	c.DrawLine(chartX+28, chartBodyY+chartBodyH, chartX+chartW, chartBodyY+chartBodyH,
		canvas.StrokePaint(colSurface1, 1))

	// Legend
	legX := chartX + chartW - 160
	legY := chartBodyY - 4
	c.DrawRoundedRect(legX-2, legY-6, 10, 10, 2, canvas.FillPaint(colRed.WithAlpha(0.8)))
	c.DrawText(legX+12, legY+5, "Expenses", d.st.Caption)
	c.DrawRoundedRect(legX+80, legY-6, 10, 10, 2, canvas.FillPaint(colGreen.WithAlpha(0.8)))
	c.DrawText(legX+94, legY+5, "Income", d.st.Caption)

	// ── Budget Donut (right panel) ────────────────────────────────────────────
	rightX := x + chartW + padX*3
	rightW := w - chartW - padX*4 - 20

	c.DrawText(rightX, y+chartY, "Budget Status", d.st.H2)

	spending := d.store.SpendingByCategory()
	budgets := st.Budgets

	bListY := y + chartY + 34
	bRowH := float32(36)

	for i, b := range budgets {
		if float32(i)*bRowH > chartH-80 {
			break
		}
		by := bListY + float32(i)*bRowH
		barMaxW := rightW - 10

		pct := float32(b.Spent / b.Limit)
		if pct > 1 {
			pct = 1
		}
		_ = spending

		col := canvas.Hex(b.Color)
		barFillW := barMaxW * 0.6 * pct * animT

		// Category name
		catStyle := canvas.TextStyle{Color: colSubtext0, Size: 11, FontPath: d.st.FontPath}
		c.DrawText(rightX, by+12, b.Category, catStyle)

		// Amount text
		amtStr := fmt.Sprintf("%s / %s",
			fmtMoney(b.Spent, currency),
			fmtMoney(b.Limit, currency))
		asz := c.MeasureText(amtStr, d.st.Caption)
		amtCol := colSubtext0
		if pct > 0.9 {
			amtCol = colRed
		} else if pct > 0.7 {
			amtCol = colYellow
		}
		amtStyle := canvas.TextStyle{Color: amtCol, Size: 10, FontPath: d.st.FontPath}
		c.DrawText(rightX+barMaxW-asz.W, by+12, amtStr, amtStyle)

		// Track
		c.DrawRoundedRect(rightX, by+18, barMaxW*0.6, 6, 3,
			canvas.FillPaint(colSurface0))
		// Fill
		if barFillW > 0 {
			fillCol := col
			if pct > 0.9 {
				fillCol = colRed
			}
			c.DrawRoundedRect(rightX, by+18, barFillW, 6, 3,
				canvas.FillPaint(fillCol))
		}

		// Pct
		pctStr := fmt.Sprintf("%d%%", int(pct*100))
		pctStyle := canvas.TextStyle{Color: amtCol, Size: 10, FontPath: d.st.FontPath}
		c.DrawText(rightX+barMaxW*0.6+6, by+25, pctStr, pctStyle)
	}

	// ── Recent Transactions ───────────────────────────────────────────────────
	const recentX = float32(20)
	recentY := y + chartY + chartH - 20
	_ = recentX

	c.DrawText(x+padX, recentY-16, "Recent Transactions", d.st.H2)

	txs := d.store.FilteredTransactions()
	maxShow := 5
	if len(txs) < maxShow {
		maxShow = len(txs)
	}

	for i := 0; i < maxShow; i++ {
		tx := txs[i]
		txY := recentY + float32(i)*28
		if txY+28 > y+h-10 {
			break
		}

		catCol := categoryColor(tx.Category)

		// Icon circle
		c.DrawCircle(x+padX+10, txY+12, 9, canvas.FillPaint(catCol.WithAlpha(0.2)))
		icStyle := canvas.TextStyle{Color: catCol, Size: 9, FontPath: d.st.FontPath}
		ic := categoryIcon(tx.Category)
		icsz := c.MeasureText(ic, icStyle)
		c.DrawText(x+padX+10-icsz.W/2, txY+16, ic, icStyle)

		// Description
		c.DrawText(x+padX+26, txY+16, tx.Description, d.st.BodyBold)

		// Date
		dateStr := tx.Date.Format("Jan 2")
		c.DrawText(x+padX+180, txY+16, dateStr, d.st.Caption)

		// Amount
		amtStr := fmtMoney(tx.Amount, currency)
		amtColor := colRed
		if tx.Type == TxIncome {
			amtColor = colGreen
			amtStr = "+" + amtStr
		}
		amtStyle := canvas.TextStyle{Color: amtColor, Size: 13, FontPath: d.st.FontPath}
		asz := c.MeasureText(amtStr, amtStyle)
		c.DrawText(x+chartW-asz.W-20, txY+16, amtStr, amtStyle)
	}

	// ── Modal overlay ─────────────────────────────────────────────────────────
	d.modal.Draw(c, x, y, w, h)
}

func (d *DashboardScreen) openAddModal() {
	// Build a quick-add transaction form as a dialog
	descInput := ui.NewTextInput("Description (e.g. Coffee)")
	amtInput := ui.NewTextInput("Amount (e.g. 12.50)")
	catDrop := ui.NewDropdown(
		[]string{"Food & Dining", "Housing", "Transport", "Entertainment", "Health", "Shopping", "Utilities", "Salary", "Freelance", "Investment"},
		ui.DefaultDropdownStyle(),
	)
	catDrop.Placeholder = "Category"
	typeDrop := ui.NewDropdown(
		[]string{"Expense", "Income", "Transfer"},
		ui.DefaultDropdownStyle(),
	)
	typeDrop.Placeholder = "Type"

	dlg := ui.NewDialog(ui.DialogOptions{
		Title:   "Add Transaction",
		Content: ui.NewSimpleLabel("Enter the details for your new transaction.", d.st.Body),
		Actions: []ui.DialogAction{
			{
				Label:   "Add",
				Primary: true,
				OnClick: func() {
					if descInput.Text == "" || amtInput.Text == "" {
						d.toasts.Show("Please fill in all fields", ui.ToastWarning, 2*time.Second)
						return
					}
					// Parse amount
					var amt float64
					var cents int
					numStr := amtInput.Text
					dotIdx := -1
					for i, ch := range numStr {
						if ch == '.' {
							dotIdx = i
						}
					}
					if dotIdx < 0 {
						for _, ch := range numStr {
							if ch >= '0' && ch <= '9' {
								cents = cents*10 + int(ch-'0')
							}
						}
						cents *= 100
					} else {
						wholePart := numStr[:dotIdx]
						fracPart := numStr[dotIdx+1:]
						for _, ch := range wholePart {
							if ch >= '0' && ch <= '9' {
								cents = cents*10 + int(ch-'0')
							}
						}
						cents *= 100
						mul := 10
						for _, ch := range fracPart {
							if ch >= '0' && ch <= '9' {
								cents += int(ch-'0') * (100 / mul)
								mul *= 10
							}
						}
					}
					amt = float64(cents) / 100.0

					txType := TxExpense
					if typeDrop.Selected == 1 {
						txType = TxIncome
					} else if typeDrop.Selected == 2 {
						txType = TxTransfer
					}

					cat := "Shopping"
					if catDrop.Selected >= 0 {
						cats := []string{"Food & Dining", "Housing", "Transport", "Entertainment", "Health", "Shopping", "Utilities", "Salary", "Freelance", "Investment"}
						cat = cats[catDrop.Selected]
					}

					d.store.AddTransaction(Transaction{
						Date:        time.Now(),
						Description: descInput.Text,
						Amount:      amt,
						Category:    cat,
						Type:        txType,
						Account:     "Main Checking",
					})
					d.toasts.Show("Transaction added!", ui.ToastSuccess, 2*time.Second)
					d.modal.Pop()
				},
			},
			{
				Label:   "Cancel",
				Primary: false,
				OnClick: func() { d.modal.Pop() },
			},
		},
	})
	_ = descInput
	_ = amtInput
	_ = catDrop
	_ = typeDrop

	d.modal.Push(dlg)
}

// ── Mini donut helper ─────────────────────────────────────────────────────────

func drawDonutSegment(c *canvas.Canvas, cx, cy, r, thickness, startAngle, sweepAngle float32, col canvas.Color) {
	steps := int(sweepAngle * 20)
	if steps < 2 {
		steps = 2
	}
	innerR := r - thickness
	for i := 0; i < steps; i++ {
		a0 := startAngle + float32(i)/float32(steps)*sweepAngle
		a1 := startAngle + float32(i+1)/float32(steps)*sweepAngle
		pts := []canvas.Point{
			{X: cx + r*float32(math.Cos(float64(a0))), Y: cy + r*float32(math.Sin(float64(a0)))},
			{X: cx + r*float32(math.Cos(float64(a1))), Y: cy + r*float32(math.Sin(float64(a1)))},
			{X: cx + innerR*float32(math.Cos(float64(a1))), Y: cy + innerR*float32(math.Sin(float64(a1)))},
			{X: cx + innerR*float32(math.Cos(float64(a0))), Y: cy + innerR*float32(math.Sin(float64(a0)))},
		}
		c.DrawPolygon(pts, canvas.FillPaint(col))
	}
}
