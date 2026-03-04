package main

import (
	"fmt"
	"time"

	"github.com/achiket/gui-go/animation"
	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/ui"
)

// BudgetScreen shows budget usage with animated progress bars.
type BudgetScreen struct {
	ui.BaseScreen
	st     *Styles
	store  *FinanceStore
	toasts *ui.ToastManager

	// Animation
	fillAnim *animation.AnimationController

	// Widgets
	modal    *ui.ModalManager
	hoverRow int

	// Summary
	totalBudget float64
	totalSpent  float64
}

func NewBudgetScreen(st *Styles, store *FinanceStore, toasts *ui.ToastManager) *BudgetScreen {
	b := &BudgetScreen{
		st:       st,
		store:    store,
		toasts:   toasts,
		hoverRow: -1,
	}
	b.modal = ui.NewModalManager()
	b.fillAnim = animation.NewController(1000 * time.Millisecond)
	b.fillAnim.Forward()
	return b
}

func (b *BudgetScreen) OnEnter(nav *ui.Navigator) {
	b.Nav = nav
	b.fillAnim = animation.NewController(900 * time.Millisecond)
	b.fillAnim.Forward()
}

func (b *BudgetScreen) Tick(delta float64) {
	b.fillAnim.Tick(delta)
	b.modal.Tick(delta)
}

func (b *BudgetScreen) HandleEvent(e ui.Event) bool {
	if b.modal.HandleEvent(e) {
		return true
	}

	if e.Type == ui.EventMouseMove {
		bds := b.Bounds()
		if e.Y > bds.Y+160 {
			b.hoverRow = int((e.Y - bds.Y - 160) / 88)
		} else {
			b.hoverRow = -1
		}
	}

	if e.Type == ui.EventMouseDown && e.Button == 1 {
		bds := b.Bounds()
		if e.Y > bds.Y+160 {
			idx := int((e.Y - bds.Y - 160) / 88)
			st := b.store.Get()
			if idx >= 0 && idx < len(st.Budgets) {
				b.openEditBudget(idx, st.Budgets[idx])
			}
		}
	}

	return false
}

func (b *BudgetScreen) openEditBudget(idx int, budget Budget) {
	limitInput := ui.NewTextInput(fmt.Sprintf("%.0f", budget.Limit))
	limitInput.Text = fmt.Sprintf("%.0f", budget.Limit)

	dlg := ui.NewDialog(ui.DialogOptions{
		Title: "Edit Budget: " + budget.Category,
		Actions: []ui.DialogAction{
			{
				Label:   "Save",
				Primary: true,
				OnClick: func() {
					// Parse new limit
					var newLimit float64
					for _, ch := range limitInput.Text {
						if ch >= '0' && ch <= '9' {
							newLimit = newLimit*10 + float64(ch-'0')
						}
					}
					if newLimit <= 0 {
						b.toasts.Show("Please enter a valid amount", ui.ToastWarning, 2*time.Second)
						return
					}
					b.store.Mutate(func(st *FinanceState) {
						st.Budgets[idx].Limit = newLimit
					})
					b.toasts.Show("Budget updated for "+budget.Category, ui.ToastSuccess, 2*time.Second)
					b.modal.Pop()
				},
			},
			{
				Label:   "Cancel",
				Primary: false,
				OnClick: func() { b.modal.Pop() },
			},
		},
	})
	_ = limitInput
	b.modal.Push(dlg)
}

func (b *BudgetScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	b.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})
	st := b.store.Get()
	currency := st.Currency
	animT := float32(b.fillAnim.Value())

	// ── Header ────────────────────────────────────────────────────────────────
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colMantle))
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colSurface0))
	c.DrawText(x+24, y+38, "Budget Planner", b.st.H1)

	subTitle := "Click any budget to edit"
	subSz := c.MeasureText(subTitle, b.st.Caption)
	c.DrawText(x+w-subSz.W-24, y+38, subTitle, b.st.Caption)

	// ── Summary Row ───────────────────────────────────────────────────────────
	totalBudget := 0.0
	totalSpent := 0.0
	overBudget := 0
	for _, bgt := range st.Budgets {
		totalBudget += bgt.Limit
		totalSpent += bgt.Spent
		if bgt.Spent > bgt.Limit {
			overBudget++
		}
	}
	remaining := totalBudget - totalSpent

	summaries := []struct {
		label string
		value string
		color canvas.Color
	}{
		{"Total Budget", fmtMoney(totalBudget, currency), colAccent},
		{"Total Spent", fmtMoney(totalSpent, currency), colRed},
		{"Remaining", fmtMoney(remaining, currency), colGreen},
		{"Over Budget", fmt.Sprintf("%d categories", overBudget), colYellow},
	}

	sumCardW := (w - 100) / 4
	for i, sum := range summaries {
		sx := x + 20 + float32(i)*(sumCardW+20)
		sy := y + 74
		c.DrawRoundedRect(sx, sy, sumCardW, 60, 8, canvas.FillPaint(colMantle))
		c.DrawRoundedRect(sx, sy, sumCardW, 60, 8, canvas.StrokePaint(colSurface0, 1))
		c.DrawRoundedRect(sx, sy+8, 3, 44, 2, canvas.FillPaint(sum.color))
		c.DrawText(sx+14, sy+22, sum.label, b.st.Caption)
		valStyle := canvas.TextStyle{Color: sum.color, Size: 16, FontPath: b.st.FontPath}
		c.DrawText(sx+14, sy+46, sum.value, valStyle)
	}

	// ── Budget List ───────────────────────────────────────────────────────────
	const rowH = float32(88)
	const padX = float32(20)
	listY := y + 156

	for i, bgt := range st.Budgets {
		rowY := listY + float32(i)*rowH
		if rowY+rowH > y+h-10 {
			break
		}

		isHovered := i == b.hoverRow

		bgColor := colMantle
		if isHovered {
			bgColor = colSurface0.WithAlpha(0.5)
		}

		c.DrawRoundedRect(x+padX, rowY+4, w-padX*2, rowH-8, 12, canvas.FillPaint(bgColor))
		c.DrawRoundedRect(x+padX, rowY+4, w-padX*2, rowH-8, 12, canvas.StrokePaint(colSurface0, 1))

		catColor := canvas.Hex(bgt.Color)

		// Category icon
		c.DrawCircle(x+padX+30, rowY+rowH/2, 16, canvas.FillPaint(catColor.WithAlpha(0.2)))
		c.DrawCircle(x+padX+30, rowY+rowH/2, 16, canvas.StrokePaint(catColor, 1.5))
		icStyle := canvas.TextStyle{Color: catColor, Size: 11, FontPath: b.st.FontPath}
		ic := categoryIcon(bgt.Category)
		icsz := c.MeasureText(ic, icStyle)
		c.DrawText(x+padX+30-icsz.W/2, rowY+rowH/2+4, ic, icStyle)

		// Category name
		catNameStyle := canvas.TextStyle{Color: colText, Size: 15, FontPath: b.st.FontPath}
		c.DrawText(x+padX+56, rowY+rowH/2-8, bgt.Category, catNameStyle)

		pct := bgt.Spent / bgt.Limit
		pctColor := catColor
		if pct > 0.9 {
			pctColor = colRed
		} else if pct > 0.7 {
			pctColor = colYellow
		}

		pctStr := fmt.Sprintf("%.0f%%", pct*100)
		pctStyle := canvas.TextStyle{Color: pctColor, Size: 13, FontPath: b.st.FontPath}
		c.DrawText(x+padX+56, rowY+rowH/2+14, pctStr+" used", pctStyle)

		// ── Progress bar ──────────────────────────────────────────────────────
		barX := x + padX + 200
		barY := rowY + rowH/2 - 8
		barW := w - padX*2 - 200 - 180
		barH := float32(12)

		c.DrawRoundedRect(barX, barY, barW, barH, barH/2, canvas.FillPaint(colSurface0))

		fillPct := float32(pct)
		if fillPct > 1 {
			fillPct = 1
		}
		fillW := barW * fillPct * animT

		if fillW > 0 {
			fillColor := catColor
			if pct > 0.9 {
				fillColor = colRed
			}
			c.DrawRoundedRect(barX, barY, fillW, barH, barH/2, canvas.FillPaint(fillColor))

			// Shine
			if fillW > 8 {
				c.DrawRoundedRect(barX+2, barY+2, fillW-4, barH/2-2, barH/4,
					canvas.FillPaint(canvas.White.WithAlpha(0.12)))
			}
		}

		// Over budget marker
		if pct > 1 {
			overStyle := canvas.TextStyle{Color: colRed, Size: 10, FontPath: b.st.FontPath}
			c.DrawText(barX+barW+8, barY+10, "OVER", overStyle)
		}

		// Warning if close
		if pct > 0.85 && pct <= 1 {
			warnStyle := canvas.TextStyle{Color: colYellow, Size: 10, FontPath: b.st.FontPath}
			c.DrawText(barX+barW+8, barY+10, "WARN", warnStyle)
		}

		// Amount labels (right side)
		labelX := x + w - padX - 160
		spentLabel := fmt.Sprintf("Spent: %s", fmtMoney(bgt.Spent, currency))
		limitLabel := fmt.Sprintf("Limit: %s", fmtMoney(bgt.Limit, currency))
		remLabel := fmt.Sprintf("Left:  %s", fmtMoney(bgt.Limit-bgt.Spent, currency))

		spentColor := colSubtext0
		if pct > 0.9 {
			spentColor = colRed
		}
		c.DrawText(labelX, rowY+rowH/2-12, spentLabel, canvas.TextStyle{Color: spentColor, Size: 12, FontPath: b.st.FontPath})
		c.DrawText(labelX, rowY+rowH/2+6, limitLabel, canvas.TextStyle{Color: colSubtext0, Size: 12, FontPath: b.st.FontPath})
		remColor := colGreen
		if bgt.Spent > bgt.Limit {
			remColor = colRed
		}
		c.DrawText(labelX, rowY+rowH/2+22, remLabel, canvas.TextStyle{Color: remColor, Size: 12, FontPath: b.st.FontPath})

		// Edit hint on hover
		if isHovered {
			editStyle := canvas.TextStyle{Color: colAccent, Size: 11, FontPath: b.st.FontPath}
			esz := c.MeasureText("✎ Edit", editStyle)
			c.DrawText(x+w-padX-esz.W, rowY+rowH/2+4, "✎ Edit", editStyle)
		}
	}

	// ── Divider ───────────────────────────────────────────────────────────────
	totalRowY := listY + float32(len(st.Budgets))*rowH + 8
	if totalRowY < y+h-50 {
		c.DrawRect(x+padX, totalRowY, w-padX*2, 1, canvas.FillPaint(colSurface1))
		totalRowY += 12

		// Total row
		c.DrawText(x+padX+56, totalRowY+16, "Monthly Total", b.st.H3)

		// Overall progress
		totalPct := float32(totalSpent / totalBudget)
		if totalPct > 1 {
			totalPct = 1
		}
		barX := x + padX + 200
		barW := w - padX*2 - 200 - 180
		c.DrawRoundedRect(barX, totalRowY+6, barW, 14, 7, canvas.FillPaint(colSurface0))
		fillW := barW * totalPct * animT
		if fillW > 0 {
			fillCol := colAccent
			if totalPct > 0.9 {
				fillCol = colRed
			} else if totalPct > 0.7 {
				fillCol = colYellow
			}
			gradPaint := canvas.GradientPaint(fillCol, canvas.Lerp(fillCol, canvas.White, 0.2))
			gradPaint.LinearGrad.From = canvas.Point{X: barX, Y: 0}
			gradPaint.LinearGrad.To = canvas.Point{X: barX + fillW, Y: 0}
			c.DrawRoundedRect(barX, totalRowY+6, fillW, 14, 7, gradPaint)
		}

		totalAmtStr := fmt.Sprintf("%s / %s  (%.0f%%)",
			fmtMoney(totalSpent, currency),
			fmtMoney(totalBudget, currency),
			totalPct*100)
		c.DrawText(x+padX+200+barW+12, totalRowY+18, totalAmtStr, b.st.Body)
	}

	// Modal
	b.modal.Draw(c, x, y, w, h)
}
