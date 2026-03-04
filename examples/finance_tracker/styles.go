package main

import "github.com/achiket/gui-go/canvas"

// ── Palette (Catppuccin Mocha) ─────────────────────────────────────────────────
var (
	colBase     = canvas.Hex("#1E1E2E")
	colMantle   = canvas.Hex("#181825")
	colCrust    = canvas.Hex("#11111B")
	colSurface0 = canvas.Hex("#313244")
	colSurface1 = canvas.Hex("#45475A")
	colSurface2 = canvas.Hex("#585B70")
	colOverlay0 = canvas.Hex("#6C7086")
	colText     = canvas.Hex("#CDD6F4")
	colSubtext0 = canvas.Hex("#A6ADC8")
	colSubtext1 = canvas.Hex("#BAC2DE")
	colAccent   = canvas.Hex("#89B4FA") // blue
	colGreen    = canvas.Hex("#A6E3A1")
	colRed      = canvas.Hex("#F38BA8")
	colYellow   = canvas.Hex("#F9E2AF")
	colPeach    = canvas.Hex("#FAB387")
	colMauve    = canvas.Hex("#CBA6F7")
	colTeal     = canvas.Hex("#94E2D5")
	colPink     = canvas.Hex("#F5C2E7")
	colSky      = canvas.Hex("#89DCEB")
)

// ── Styles struct ─────────────────────────────────────────────────────────────

type Styles struct {
	FontPath string
	H1       canvas.TextStyle
	H2       canvas.TextStyle
	H3       canvas.TextStyle
	Body     canvas.TextStyle
	BodyBold canvas.TextStyle
	Caption  canvas.TextStyle
	Mono     canvas.TextStyle
	// Accented variants
	AccentBody canvas.TextStyle
	GreenBody  canvas.TextStyle
	RedBody    canvas.TextStyle
}

func NewStyles(fontPath string) *Styles {
	return &Styles{
		FontPath: fontPath,
		H1:       canvas.TextStyle{Color: colText, Size: 24, FontPath: fontPath},
		H2:       canvas.TextStyle{Color: colText, Size: 18, FontPath: fontPath},
		H3:       canvas.TextStyle{Color: colSubtext1, Size: 14, FontPath: fontPath},
		Body:     canvas.TextStyle{Color: colSubtext0, Size: 13, FontPath: fontPath},
		BodyBold: canvas.TextStyle{Color: colText, Size: 13, FontPath: fontPath},
		Caption:  canvas.TextStyle{Color: colOverlay0, Size: 11, FontPath: fontPath},
		Mono:     canvas.TextStyle{Color: colText, Size: 13, FontPath: fontPath},
		AccentBody: canvas.TextStyle{Color: colAccent, Size: 13, FontPath: fontPath},
		GreenBody:  canvas.TextStyle{Color: colGreen, Size: 13, FontPath: fontPath},
		RedBody:    canvas.TextStyle{Color: colRed, Size: 13, FontPath: fontPath},
	}
}

// ── Helper: format currency ────────────────────────────────────────────────────

func fmtMoney(amount float64, currency string) string {
	sym := "$"
	switch currency {
	case "EUR":
		sym = "€"
	case "GBP":
		sym = "£"
	case "JPY":
		sym = "¥"
	}
	if amount < 0 {
		return "-" + sym + fmtFloat(-amount)
	}
	return sym + fmtFloat(amount)
}

func fmtFloat(v float64) string {
	// Format with 2 decimal places and thousands separator
	cents := int(v*100 + 0.5)
	dollars := cents / 100
	c := cents % 100

	// Thousands
	s := ""
	if dollars == 0 {
		s = "0"
	} else {
		for dollars > 0 {
			group := dollars % 1000
			dollars /= 1000
			if dollars > 0 {
				s = "," + pad3(group) + s
			} else {
				s = itoa(group) + s
			}
		}
	}
	return s + "." + pad2(c)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func pad3(n int) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// ── Category icon / color lookup ──────────────────────────────────────────────

func categoryColor(cat string) canvas.Color {
	switch cat {
	case "Food & Dining":
		return colPeach
	case "Housing":
		return colAccent
	case "Transport":
		return colYellow
	case "Entertainment":
		return colMauve
	case "Health":
		return colGreen
	case "Shopping":
		return colRed
	case "Utilities":
		return colSky
	case "Salary":
		return colGreen
	case "Freelance":
		return colMauve
	case "Investment":
		return colPeach
	default:
		return colSubtext0
	}
}

func categoryIcon(cat string) string {
	switch cat {
	case "Food & Dining":
		return "F"
	case "Housing":
		return "H"
	case "Transport":
		return "T"
	case "Entertainment":
		return "E"
	case "Health":
		return "+"
	case "Shopping":
		return "S"
	case "Utilities":
		return "U"
	case "Salary":
		return "$"
	case "Freelance":
		return "L"
	case "Investment":
		return "I"
	default:
		return "?"
	}
}
