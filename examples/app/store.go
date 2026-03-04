package main

import (
	"math/rand"
	"time"
)

// ── Domain Types ──────────────────────────────────────────────────────────────

type TxType int

const (
	TxExpense TxType = iota
	TxIncome
	TxTransfer
)

type Transaction struct {
	ID          int
	Date        time.Time
	Description string
	Amount      float64
	Category    string
	Type        TxType
	Account     string
	Note        string
}

type Budget struct {
	Category string
	Limit    float64
	Spent    float64
	Color    string // hex color
}

type Account struct {
	Name    string
	Balance float64
	Type    string // checking, savings, credit
	Color   string
}

type FinanceState struct {
	Transactions []Transaction
	Budgets      []Budget
	Accounts     []Account
	SelectedTx   int // -1 = none
	FilterType   int // 0=all 1=income 2=expense
	FilterCat    string
	SortDesc     bool
	DarkMode     bool
	Currency     string
	NotifyBudget bool
	NotifyTx     bool
}

type FinanceStore struct {
	state FinanceState
	subs  []func()
}

func NewFinanceStore() *FinanceStore {
	s := &FinanceStore{}
	s.state = seedData()
	return s
}

func (s *FinanceStore) Get() FinanceState {
	return s.state
}

func (s *FinanceStore) Mutate(fn func(*FinanceState)) {
	fn(&s.state)
	for _, sub := range s.subs {
		sub()
	}
}

func (s *FinanceStore) Subscribe(fn func()) {
	s.subs = append(s.subs, fn)
}

// ── Derived helpers ───────────────────────────────────────────────────────────

func (s *FinanceStore) TotalBalance() float64 {
	total := 0.0
	for _, a := range s.state.Accounts {
		if a.Type == "credit" {
			total -= a.Balance
		} else {
			total += a.Balance
		}
	}
	return total
}

func (s *FinanceStore) MonthlyIncome() float64 {
	now := time.Now()
	total := 0.0
	for _, tx := range s.state.Transactions {
		if tx.Type == TxIncome && tx.Date.Month() == now.Month() && tx.Date.Year() == now.Year() {
			total += tx.Amount
		}
	}
	return total
}

func (s *FinanceStore) MonthlyExpenses() float64 {
	now := time.Now()
	total := 0.0
	for _, tx := range s.state.Transactions {
		if tx.Type == TxExpense && tx.Date.Month() == now.Month() && tx.Date.Year() == now.Year() {
			total += tx.Amount
		}
	}
	return total
}

func (s *FinanceStore) NetSavings() float64 {
	return s.MonthlyIncome() - s.MonthlyExpenses()
}

// SpendingByCategory returns category -> total spent this month
func (s *FinanceStore) SpendingByCategory() map[string]float64 {
	now := time.Now()
	m := make(map[string]float64)
	for _, tx := range s.state.Transactions {
		if tx.Type == TxExpense && tx.Date.Month() == now.Month() && tx.Date.Year() == now.Year() {
			m[tx.Category] += tx.Amount
		}
	}
	return m
}

// Last6MonthsExpenses returns monthly expense totals for charting
func (s *FinanceStore) Last6MonthsExpenses() []float64 {
	now := time.Now()
	months := make([]float64, 6)
	for _, tx := range s.state.Transactions {
		if tx.Type != TxExpense {
			continue
		}
		for i := 0; i < 6; i++ {
			target := now.AddDate(0, -i, 0)
			if tx.Date.Month() == target.Month() && tx.Date.Year() == target.Year() {
				months[5-i] += tx.Amount
				break
			}
		}
	}
	return months
}

// Last6MonthsIncome returns monthly income totals for charting
func (s *FinanceStore) Last6MonthsIncome() []float64 {
	now := time.Now()
	months := make([]float64, 6)
	for _, tx := range s.state.Transactions {
		if tx.Type != TxIncome {
			continue
		}
		for i := 0; i < 6; i++ {
			target := now.AddDate(0, -i, 0)
			if tx.Date.Month() == target.Month() && tx.Date.Year() == target.Year() {
				months[5-i] += tx.Amount
				break
			}
		}
	}
	return months
}

// MonthLabels returns abbreviated month names for the past 6 months
func (s *FinanceStore) MonthLabels() []string {
	now := time.Now()
	labels := make([]string, 6)
	for i := 0; i < 6; i++ {
		t := now.AddDate(0, -(5 - i), 0)
		labels[i] = t.Format("Jan")
	}
	return labels
}

func (s *FinanceStore) AddTransaction(tx Transaction) {
	s.Mutate(func(st *FinanceState) {
		tx.ID = len(st.Transactions) + 1
		st.Transactions = append([]Transaction{tx}, st.Transactions...)
		// Update budget spent
		for i, b := range st.Budgets {
			if b.Category == tx.Category && tx.Type == TxExpense {
				st.Budgets[i].Spent += tx.Amount
			}
		}
	})
}

func (s *FinanceStore) DeleteTransaction(id int) {
	s.Mutate(func(st *FinanceState) {
		for i, tx := range st.Transactions {
			if tx.ID == id {
				st.Transactions = append(st.Transactions[:i], st.Transactions[i+1:]...)
				break
			}
		}
	})
}

func (s *FinanceStore) FilteredTransactions() []Transaction {
	st := s.state
	var result []Transaction
	for _, tx := range st.Transactions {
		if st.FilterType == 1 && tx.Type != TxIncome {
			continue
		}
		if st.FilterType == 2 && tx.Type != TxExpense {
			continue
		}
		if st.FilterCat != "" && st.FilterCat != "All" && tx.Category != st.FilterCat {
			continue
		}
		result = append(result, tx)
	}
	return result
}

// ── Seed data ─────────────────────────────────────────────────────────────────

func seedData() FinanceState {
	rng := rand.New(rand.NewSource(42))
	now := time.Now()

	accounts := []Account{
		{Name: "Main Checking", Balance: 4823.50, Type: "checking", Color: "#89B4FA"},
		{Name: "Emergency Fund", Balance: 12500.00, Type: "savings", Color: "#A6E3A1"},
		{Name: "Investment", Balance: 28340.75, Type: "savings", Color: "#CBA6F7"},
		{Name: "Credit Card", Balance: 1247.32, Type: "credit", Color: "#F38BA8"},
	}

	categories := []struct {
		name  string
		color string
		txType TxType
	}{
		{"Food & Dining", "#FAB387", TxExpense},
		{"Housing", "#89B4FA", TxExpense},
		{"Transport", "#F9E2AF", TxExpense},
		{"Entertainment", "#CBA6F7", TxExpense},
		{"Health", "#A6E3A1", TxExpense},
		{"Shopping", "#F38BA8", TxExpense},
		{"Utilities", "#89DCEB", TxExpense},
		{"Salary", "#A6E3A1", TxIncome},
		{"Freelance", "#CBA6F7", TxIncome},
		{"Investment", "#FAB387", TxIncome},
	}

	descriptions := map[string][]string{
		"Food & Dining":  {"Whole Foods", "Chipotle", "Coffee Shop", "Pizza Palace", "Sushi Spot"},
		"Housing":        {"Rent", "Insurance", "HOA Fee"},
		"Transport":      {"Gas Station", "Uber", "Metro Pass", "Parking"},
		"Entertainment":  {"Netflix", "Spotify", "Cinema", "Concert"},
		"Health":         {"Gym Membership", "Pharmacy", "Doctor Visit"},
		"Shopping":       {"Amazon", "Target", "Clothing Store"},
		"Utilities":      {"Electric Bill", "Internet", "Water"},
		"Salary":         {"Monthly Salary", "Paycheck"},
		"Freelance":      {"Client Project", "Consulting Fee"},
		"Investment":     {"Dividend", "Capital Gains"},
	}

	amounts := map[string][2]float64{
		"Food & Dining":  {8, 85},
		"Housing":        {800, 2000},
		"Transport":      {15, 120},
		"Entertainment":  {10, 60},
		"Health":         {25, 200},
		"Shopping":       {20, 300},
		"Utilities":      {50, 200},
		"Salary":         {3500, 5500},
		"Freelance":      {200, 1500},
		"Investment":     {50, 500},
	}

	var txs []Transaction
	id := 1

	for monthOffset := 0; monthOffset < 6; monthOffset++ {
		base := now.AddDate(0, -monthOffset, 0)

		// Generate 15-25 transactions per month
		count := 15 + rng.Intn(10)
		for j := 0; j < count; j++ {
			catIdx := rng.Intn(len(categories))
			cat := categories[catIdx]
			descs := descriptions[cat.name]
			desc := descs[rng.Intn(len(descs))]
			amtRange := amounts[cat.name]
			amt := amtRange[0] + rng.Float64()*(amtRange[1]-amtRange[0])
			amt = float64(int(amt*100)) / 100

			day := 1 + rng.Intn(28)
			date := time.Date(base.Year(), base.Month(), day, 10+rng.Intn(12), rng.Intn(60), 0, 0, time.Local)

			txs = append(txs, Transaction{
				ID:          id,
				Date:        date,
				Description: desc,
				Amount:      amt,
				Category:    cat.name,
				Type:        cat.txType,
				Account:     accounts[rng.Intn(len(accounts))].Name,
			})
			id++
		}
	}

	// Sort newest first
	for i := 0; i < len(txs); i++ {
		for j := i + 1; j < len(txs); j++ {
			if txs[j].Date.After(txs[i].Date) {
				txs[i], txs[j] = txs[j], txs[i]
			}
		}
	}

	budgets := []Budget{
		{Category: "Food & Dining", Limit: 400, Color: "#FAB387"},
		{Category: "Housing", Limit: 1800, Color: "#89B4FA"},
		{Category: "Transport", Limit: 200, Color: "#F9E2AF"},
		{Category: "Entertainment", Limit: 150, Color: "#CBA6F7"},
		{Category: "Health", Limit: 300, Color: "#A6E3A1"},
		{Category: "Shopping", Limit: 250, Color: "#F38BA8"},
		{Category: "Utilities", Limit: 250, Color: "#89DCEB"},
	}

	// Calculate current month spending for budgets
	for i, b := range budgets {
		for _, tx := range txs {
			if tx.Type == TxExpense && tx.Category == b.Category &&
				tx.Date.Month() == now.Month() && tx.Date.Year() == now.Year() {
				budgets[i].Spent += tx.Amount
			}
		}
	}

	return FinanceState{
		Transactions: txs,
		Budgets:      budgets,
		Accounts:     accounts,
		SelectedTx:   -1,
		FilterType:   0,
		FilterCat:    "All",
		SortDesc:     true,
		DarkMode:     true,
		Currency:     "USD",
		NotifyBudget: true,
		NotifyTx:     true,
	}
}
