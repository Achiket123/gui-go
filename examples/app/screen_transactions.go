package main

import (
	"strings"
	"time"

	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

// TransactionsScreen shows a filterable, scrollable list of all transactions.
type TransactionsScreen struct {
	ui.BaseScreen
	st     *Styles
	store  *FinanceStore
	toasts *ui.ToastManager

	// Widgets
	searchInput *ui.TextInput
	typeFilter  *ui.RadioGroup
	catDrop     *ui.Dropdown
	sortToggle  *ui.Toggle
	modal       *ui.ModalManager
	confirmDlg  *ui.Dialog

	// Scroll
	scrollY    float32
	maxScrollY float32

	// Hover row
	hoverRow int

	// Pending delete
	pendingDeleteID int
}

func NewTransactionsScreen(st *Styles, store *FinanceStore, toasts *ui.ToastManager) *TransactionsScreen {
	s := &TransactionsScreen{
		st:              st,
		store:           store,
		toasts:          toasts,
		hoverRow:        -1,
		pendingDeleteID: -1,
	}

	s.searchInput = ui.NewTextInput("Search transactions…")
	s.searchInput.Style.Background = colMantle
	s.searchInput.Style.BorderColor = colSurface1
	s.searchInput.Style.FocusBorder = colAccent
	s.searchInput.Style.TextStyle = canvas.TextStyle{Color: colText, Size: 13, FontPath: st.FontPath}
	s.searchInput.Style.HintStyle = canvas.TextStyle{Color: colOverlay0, Size: 13, FontPath: st.FontPath}

	s.typeFilter = ui.NewRadioGroup([]string{"All", "Income", "Expense"}, 0, false)
	cbStyle := ui.DefaultCheckboxStyle()
	cbStyle.CheckedBg = colAccent
	cbStyle.LabelStyle = canvas.TextStyle{Color: colSubtext0, Size: 12, FontPath: st.FontPath}
	s.typeFilter.Style = cbStyle

	cats := []string{"All", "Food & Dining", "Housing", "Transport", "Entertainment", "Health", "Shopping", "Utilities", "Salary", "Freelance", "Investment"}
	dropStyle := ui.DefaultDropdownStyle()
	dropStyle.Background = colMantle
	dropStyle.HoverBg = colSurface0
	dropStyle.Border = colSurface1
	dropStyle.TextStyle = canvas.TextStyle{Color: colSubtext0, Size: 12, FontPath: st.FontPath}
	dropStyle.ItemHeight = 32
	s.catDrop = ui.NewDropdown(cats, dropStyle)
	s.catDrop.Placeholder = "All Categories"

	s.sortToggle = ui.NewToggle(true, ui.DefaultToggleStyle(), func(v bool) {
		store.Mutate(func(st *FinanceState) { st.SortDesc = v })
	})

	s.modal = ui.NewModalManager()
	return s
}

func (s *TransactionsScreen) Tick(delta float64) {
	s.searchInput.Tick(delta)
	s.typeFilter.Tick(delta)
	s.catDrop.Tick(delta)
	s.sortToggle.Tick(delta)
	s.modal.Tick(delta)
}

func (s *TransactionsScreen) HandleEvent(e ui.Event) bool {
	if s.modal.HandleEvent(e) {
		return true
	}
	if s.catDrop.HandleEvent(e) {
		if s.catDrop.Selected >= 0 {
			cats := []string{"All", "Food & Dining", "Housing", "Transport", "Entertainment", "Health", "Shopping", "Utilities", "Salary", "Freelance", "Investment"}
			s.store.Mutate(func(st *FinanceState) {
				st.FilterCat = cats[s.catDrop.Selected]
			})
		}
		return true
	}
	if s.typeFilter.HandleEvent(e) {
		s.store.Mutate(func(st *FinanceState) {
			st.FilterType = s.typeFilter.Selected
		})
		return true
	}
	if s.searchInput.HandleEvent(e) {
		return true
	}
	if s.sortToggle.HandleEvent(e) {
		return true
	}

	// Scroll
	if e.Type == ui.EventScroll {
		b := s.Bounds()
		if e.X >= b.X && e.X <= b.X+b.W {
			s.scrollY -= e.ScrollY * 40
			if s.scrollY < 0 {
				s.scrollY = 0
			}
			if s.scrollY > s.maxScrollY {
				s.scrollY = s.maxScrollY
			}
			return true
		}
	}

	// Row hover + click
	if e.Type == ui.EventMouseMove {
		b := s.Bounds()
		if e.Y > b.Y+140 {
			rowIdx := int((e.Y - b.Y - 140 + s.scrollY) / 52)
			s.hoverRow = rowIdx
		} else {
			s.hoverRow = -1
		}
	}

	if e.Type == ui.EventMouseDown && e.Button == 1 {
		b := s.Bounds()
		if e.Y > b.Y+140 {
			rowIdx := int((e.Y - b.Y - 140 + s.scrollY) / 52)
			txs := s.filteredAndSearched()
			if rowIdx >= 0 && rowIdx < len(txs) {
				// Check if delete button was clicked (right side)
				if e.X > b.X+b.W-80 {
					s.pendingDeleteID = txs[rowIdx].ID
					s.showDeleteConfirm(txs[rowIdx].Description)
					return true
				}
				// Select row
				s.store.Mutate(func(st *FinanceState) {
					st.SelectedTx = txs[rowIdx].ID
				})
				return true
			}
		}
	}

	return false
}

func (s *TransactionsScreen) filteredAndSearched() []Transaction {
	txs := s.store.FilteredTransactions()
	query := strings.ToLower(strings.TrimSpace(s.searchInput.Text))
	if query == "" {
		return txs
	}
	var result []Transaction
	for _, tx := range txs {
		if strings.Contains(strings.ToLower(tx.Description), query) ||
			strings.Contains(strings.ToLower(tx.Category), query) ||
			strings.Contains(strings.ToLower(tx.Account), query) {
			result = append(result, tx)
		}
	}
	return result
}

func (s *TransactionsScreen) showDeleteConfirm(desc string) {
	dlg := ui.NewConfirmDialog(
		"Delete Transaction",
		"Are you sure you want to delete '"+desc+"'? This cannot be undone.",
		func() {
			s.store.DeleteTransaction(s.pendingDeleteID)
			s.toasts.Show("Transaction deleted", ui.ToastSuccess, 2*time.Second)
			s.pendingDeleteID = -1
			s.modal.Pop()
		},
		func() {
			s.pendingDeleteID = -1
			s.modal.Pop()
		},
	)
	s.modal.Push(dlg)
}

func (s *TransactionsScreen) Draw(c *canvas.Canvas, x, y, w, h float32) {
	s.SetBounds(canvas.Rect{X: x, Y: y, W: w, H: h})
	st := s.store.Get()
	currency := st.Currency

	// ── Header ────────────────────────────────────────────────────────────────
	c.DrawRect(x, y, w, 64, canvas.FillPaint(colMantle))
	c.DrawRect(x, y+63, w, 1, canvas.FillPaint(colSurface0))
	c.DrawText(x+24, y+38, "Transactions", s.st.H1)

	// Transaction count
	countStr := itoa(len(s.filteredAndSearched())) + " records"
	cSz := c.MeasureText(countStr, s.st.Caption)
	c.DrawText(x+w-cSz.W-24, y+38, countStr, s.st.Caption)

	// ── Filter toolbar ────────────────────────────────────────────────────────
	toolbarY := y + 70
	toolbarH := float32(50)
	c.DrawRect(x, toolbarY, w, toolbarH, canvas.FillPaint(colCrust))
	c.DrawRect(x, toolbarY+toolbarH-1, w, 1, canvas.FillPaint(colSurface0))

	// Search
	s.searchInput.Draw(c, x+16, toolbarY+8, 220, 34)

	// Type filter
	filterLabel := canvas.TextStyle{Color: colOverlay0, Size: 11, FontPath: s.st.FontPath}
	c.DrawText(x+252, toolbarY+21, "Type:", filterLabel)
	s.typeFilter.Draw(c, x+290, toolbarY+8, 220, 34)

	// Category dropdown
	c.DrawText(x+526, toolbarY+21, "Category:", filterLabel)
	s.catDrop.Draw(c, x+590, toolbarY+8, 180, 34)

	// Sort toggle
	sortLblStr := "Newest First"
	if !st.SortDesc {
		sortLblStr = "Oldest First"
	}
	sortLbl := canvas.TextStyle{Color: colSubtext0, Size: 11, FontPath: s.st.FontPath}
	c.DrawText(x+w-180, toolbarY+21, sortLblStr, sortLbl)
	s.sortToggle.Draw(c, x+w-46, toolbarY+13, 34, 20)

	// ── Column headers ────────────────────────────────────────────────────────
	headerY := toolbarY + toolbarH
	c.DrawRect(x, headerY, w, 30, canvas.FillPaint(colMantle))

	headers := []struct {
		label string
		xOff  float32
	}{
		{"Date", 60},
		{"Description", 140},
		{"Category", 340},
		{"Account", 510},
		{"Amount", w - 130},
		{"", w - 60},
	}

	for _, h2 := range headers {
		c.DrawText(x+h2.xOff, headerY+20, h2.label, s.st.Caption)
	}
	c.DrawRect(x, headerY+30, w, 1, canvas.FillPaint(colSurface0))

	// ── Transaction rows ──────────────────────────────────────────────────────
	const rowH = float32(52)
	listY := headerY + 31
	listH := h - (listY - y)
	txs := s.filteredAndSearched()

	totalH := float32(len(txs)) * rowH
	s.maxScrollY = totalH - listH
	if s.maxScrollY < 0 {
		s.maxScrollY = 0
	}

	c.Save()
	c.ClipRect(x, listY, w, listH)

	for i, tx := range txs {
		ry := listY + float32(i)*rowH - s.scrollY
		if ry+rowH < listY || ry > listY+listH {
			continue
		}

		// Row background
		rowBg := colBase
		if i%2 == 0 {
			rowBg = colMantle.WithAlpha(0.5)
		}
		if i == s.hoverRow {
			rowBg = colSurface0.WithAlpha(0.6)
		}
		if tx.ID == st.SelectedTx {
			rowBg = colAccent.WithAlpha(0.1)
		}
		c.DrawRect(x, ry, w, rowH, canvas.FillPaint(rowBg))

		catCol := categoryColor(tx.Category)

		// Category icon circle
		c.DrawCircle(x+30, ry+rowH/2, 14, canvas.FillPaint(catCol.WithAlpha(0.15)))
		c.DrawCircle(x+30, ry+rowH/2, 14, canvas.StrokePaint(catCol, 1))
		icStyle := canvas.TextStyle{Color: catCol, Size: 10, FontPath: s.st.FontPath}
		ic := categoryIcon(tx.Category)
		icsz := c.MeasureText(ic, icStyle)
		c.DrawText(x+30-icsz.W/2, ry+rowH/2+4, ic, icStyle)

		// Date
		dateStr := tx.Date.Format("Jan 02, '06")
		c.DrawText(x+54, ry+rowH/2+5, dateStr, s.st.Body)

		// Description
		desc := tx.Description
		if len(desc) > 22 {
			desc = desc[:22] + "…"
		}
		c.DrawText(x+140, ry+rowH/2+5, desc, s.st.BodyBold)

		// Category badge
		c.DrawRoundedRect(x+336, ry+rowH/2-10, 120, 20, 10, canvas.FillPaint(catCol.WithAlpha(0.2)))
		catLabelStyle := canvas.TextStyle{Color: catCol, Size: 10, FontPath: s.st.FontPath}
		catLabel := tx.Category
		if len(catLabel) > 13 {
			catLabel = catLabel[:13] + "…"
		}
		clsz := c.MeasureText(catLabel, catLabelStyle)
		c.DrawText(x+336+(120-clsz.W)/2, ry+rowH/2+4, catLabel, catLabelStyle)

		// Account
		accStyle := canvas.TextStyle{Color: colSubtext0, Size: 12, FontPath: s.st.FontPath}
		accStr := tx.Account
		if len(accStr) > 16 {
			accStr = accStr[:16] + "…"
		}
		c.DrawText(x+510, ry+rowH/2+5, accStr, accStyle)

		// Amount
		amtStr := fmtMoney(tx.Amount, currency)
		amtColor := colRed
		if tx.Type == TxIncome {
			amtColor = colGreen
			amtStr = "+" + amtStr
		} else if tx.Type == TxTransfer {
			amtColor = colAccent
		}
		amtStyle := canvas.TextStyle{Color: amtColor, Size: 14, FontPath: s.st.FontPath, Bold: false}
		asz := c.MeasureText(amtStr, amtStyle)
		c.DrawText(x+w-130+60-asz.W, ry+rowH/2+5, amtStr, amtStyle)

		// Delete button (shown on hover)
		if i == s.hoverRow {
			delStyle := canvas.TextStyle{Color: colRed, Size: 18, FontPath: s.st.FontPath}
			c.DrawText(x+w-56, ry+rowH/2+6, "✕", delStyle)
		}

		// Row separator
		c.DrawRect(x, ry+rowH-1, w, 1, canvas.FillPaint(colSurface0.WithAlpha(0.5)))
	}

	// Empty state
	if len(txs) == 0 {
		msg := "No transactions match your filters"
		msz := c.MeasureText(msg, s.st.H3)
		c.DrawText(x+w/2-msz.W/2, listY+listH/2, msg, s.st.H3)
	}

	c.Restore()

	// Scrollbar
	if s.maxScrollY > 0 {
		sbH := listH * listH / totalH
		sbY := listY + (s.scrollY/s.maxScrollY)*(listH-sbH)
		c.DrawRoundedRect(x+w-6, sbY, 4, sbH, 2, canvas.FillPaint(colSurface1))
	}

	// Modal
	s.modal.Draw(c, x, y, w, h)
}
