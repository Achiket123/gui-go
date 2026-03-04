# 💰 Personal Finance Tracker

A fully-fledged desktop personal finance application built with `github.com/achiket/gui-go`.

> ~2,600 lines of Go across 8 files. Runs natively on Linux (X11/OpenGL).

---

## Features

### Screens (tabbed navigation)
| Screen | What it does |
|---|---|
| **Dashboard** | Balance cards, account overview, animated 6-month bar chart, budget progress, recent transactions |
| **Transactions** | Full searchable list, type/category filtering, newest-first toggle, scrollable rows, delete with confirm dialog |
| **Budget** | Per-category spending limits, animated fill bars, over-budget warnings, click to edit limits |
| **Analytics** | Animated donut chart, income vs expenses line chart, savings bar chart, category ranking, savings rate insight |
| **Settings** | Currency switcher, notification toggles, dark-mode toggle, data export, full reset with confirm |

### GUI library features used
- `ui.Navigator` + `ui.TabBar` — multi-screen routing
- `ui.Button` — custom styled (primary, danger, ghost)
- `ui.TextInput` — animated focus border, cursor blink
- `ui.Dropdown` — category/currency selectors
- `ui.Toggle` — notifications, sort order, dark mode
- `ui.Checkbox` / `ui.RadioGroup` — type filters
- `ui.ModalManager` + `ui.Dialog` — add/edit/confirm dialogs
- `ui.ToastManager` — success/warning/info notifications
- `ui.Splitter` — (ready for detail pane expansion)
- `canvas` drawing — rect, rounded rect, circle, line, polygon, gradient, text, clip
- `animation.AnimationController` — bar chart fill-in, pulse
- `animation.Timeline` — intro slide-in, donut sweep
- `state`-style store — reactive mutations propagate to all screens

### Data
- Seeded with **~100 realistic transactions** across 6 months
- 4 accounts (checking, savings, investment, credit)
- 7 budget categories with real monthly spending
- All figures recalculated live from transaction data

---

## Project structure

```
finance-tracker/
├── main.go                  # Window, navigator, animation controllers
├── store.go                 # Data model, seed data, computed helpers
├── styles.go                # Color palette, text styles, formatters
├── screen_dashboard.go      # Dashboard with charts + quick-add modal
├── screen_transactions.go   # Full filterable transaction list
├── screen_budget.go         # Budget progress bars + edit dialog
├── screen_analytics.go      # Donut, line, bar charts + insights
└── screen_settings.go       # All preferences + data management
```

---

## Running

```bash
# Place inside your gui-go workspace or adjust the replace directive in go.mod
cd finance-tracker
go run .
```

**Dependencies:** Only `github.com/achiket/gui-go` (pure Go + CGo, X11/OpenGL).

---

## Keyboard shortcuts

| Key | Action |
|---|---|
| `Escape` | Quit |
| `F5` | Show refresh toast |
| `Tab / click` | Navigate between screens |
| Scroll wheel | Scroll transaction list |

---

## Architecture notes

- **Store pattern** — `FinanceStore` holds all state with `Mutate()` + `Subscribe()`,
  similar to Pinia/Zustand. Screens read via `store.Get()` and mutate inside callbacks.
- **Screen interface** — each screen embeds `ui.BaseScreen` and implements
  `Draw`, `HandleEvent`, `Tick`, `OnEnter` — the library's component protocol.
- **Animations** — `animation.AnimationController` drives `Value()` 0→1 for
  bar fills, donut sweeps and intro effects; `PingPong()` drives the pulse.
- **No goroutines needed** — all UI logic is single-threaded on the render loop tick.
