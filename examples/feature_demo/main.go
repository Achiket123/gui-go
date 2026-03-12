// example/feature_demo/main.go
// Demonstrates all new goui features:
//   - Center, AspectRatio, Flex, Stack layout
//   - DrawTextBox with wrapping, ellipsis, centering, balanced text
//   - Responsive breakpoints
//   - ProgressBar, Toggle, Card, Badge, Divider, Tooltip widgets
//   - DirtyTracker + FrameThrottle for high-performance rendering
package main

import (
	"fmt"

	goui "github.com/achiket123/gui-go"
	"github.com/achiket123/gui-go/canvas"
	"github.com/achiket123/gui-go/ui"
)

func main() {
	w := goui.NewWindow("goui — Feature Demo", 1024, 768)

	// ── Frame-rate throttle: 60 fps active, 10 fps idle ──────────────────────
	// throttle := canvas.NewFrameThrottle(60, 10, 3*time.Second)
	// dirty    := canvas.NewDirtyTracker(1024, 768)

	// ── Responsive root layout ───────────────────────────────────────────────
	responsive := ui.NewResponsiveLayout(1024, 768)
	w.OnResize(func(width, height int) {
		responsive.Resize(float32(width), float32(height))
	})

	// ── Small layout (≤ 600 px wide) ─────────────────────────────────────────
	smallCard := ui.NewCard(ui.DefaultCardStyle(), ui.NewLabel(
		"Small screen layout.\nText wraps and scales automatically.",
		labelStyle(canvas.TextAlignCenter),
	))
	responsive.OnBreakpoint(600, ui.NewCenter(ui.NewSizedBox(0, 300, smallCard)))

	// ── Default layout ────────────────────────────────────────────────────────
	responsive.Default(buildDefaultLayout())

	// Register with window and show.
	w.AddComponent(responsive)

	// Example: register a tooltip-wrapped button in the component list.
	btn := ui.NewButton("Click me", nil)
	tooltipBtn := ui.NewTooltip("Opens a dialog", btn)
	w.AddComponent(tooltipBtn)

	w.OnDrawGL(func(c *canvas.Canvas) {
		c.Clear(canvas.Hex("#11111B"))

		// ── DrawTextBox examples ─────────────────────────────────────────────

		// 1. Left-aligned, wrapping, ellipsis on overflow.
		s1 := canvas.DefaultTextBoxStyle()
		s1.Align = canvas.TextAlignLeft
		s1.Overflow = canvas.TextOverflowEllipsis
		s1.MaxLines = 3
		c.DrawTextBox(
			canvas.Rect{X: 20, Y: 20, W: 300, H: 60},
			"This is a long paragraph that will be word-wrapped and clipped with an ellipsis after three lines of text.",
			s1,
		)

		// 2. Centred, balanced line-breaking.
		s2 := canvas.DefaultTextBoxStyle()
		s2.Align = canvas.TextAlignCenter
		s2.Balanced = true
		s2.Text.Color = canvas.Hex("#CBA6F7")
		c.DrawTextBox(
			canvas.Rect{X: 340, Y: 20, W: 300, H: 80},
			"Balanced text keeps line lengths even — no orphaned words.",
			s2,
		)

		// 3. Right-aligned, justified.
		s3 := canvas.DefaultTextBoxStyle()
		s3.Align = canvas.TextAlignJustify
		s3.Text.Color = canvas.Hex("#A6E3A1")
		c.DrawTextBox(
			canvas.Rect{X: 660, Y: 20, W: 340, H: 80},
			"Justified text stretches spaces so every line fills the full width of the text box.",
			s3,
		)

		// 4. DrawCenteredText — single line, perfectly centred.
		headerStyle := canvas.TextStyle{Color: canvas.Hex("#89B4FA"), Size: 22}
		c.DrawCenteredText(canvas.Rect{X: 0, Y: 110, W: c.Width(), H: 40}, "goui Widget Gallery", headerStyle)

		// ── Widgets ──────────────────────────────────────────────────────────

		// ProgressBar
		pb := ui.NewProgressBar(0.65, ui.DefaultProgressBarStyle())
		pb.Draw(c, 20, 170, 300, 24)

		// Badge
		badge := ui.NewBadge(7, ui.DefaultBadgeStyle())
		badge.Draw(c, 340, 165, 60, 30)

		// Toggle (on)
		tog := ui.NewToggle(true, ui.DefaultToggleStyle(), func(v bool) {
			fmt.Println("Toggle:", v)
		})
		tog.Draw(c, 430, 160, 60, 40)

		// Divider
		div := ui.NewDivider(false)
		div.Draw(c, 20, 215, c.Width()-40, 1)

		// Card with wrapped text inside.
		card := ui.NewCard(ui.DefaultCardStyle(), ui.NewLabel(
			"This Card widget draws a rounded background with a border.\nThe text inside wraps automatically when the window is resized.",
			labelStyle(canvas.TextAlignLeft),
		))
		card.Draw(c, 20, 230, 460, 130)

		// AspectRatio (16:9 box)
		ar := ui.NewAspectRatio(16.0/9.0, ui.NewCard(ui.DefaultCardStyle(),
			ui.NewLabel("16:9 AspectRatio", labelStyle(canvas.TextAlignCenter)),
		))
		ar.Draw(c, 500, 230, 480, 200)

		// Flex row: three equal columns.
		col1 := ui.NewCard(ui.DefaultCardStyle(), ui.NewLabel("Column 1", labelStyle(canvas.TextAlignCenter)))
		col2 := ui.NewCard(ui.DefaultCardStyle(), ui.NewLabel("Column 2", labelStyle(canvas.TextAlignCenter)))
		col3 := ui.NewCard(ui.DefaultCardStyle(), ui.NewLabel("Column 3", labelStyle(canvas.TextAlignCenter)))
		row := ui.NewFlex(ui.FlexRow,
			ui.Flexible(1, col1),
			ui.Flexible(1, col2),
			ui.Flexible(1, col3),
		)
		row.Gap = 12
		row.Draw(c, 20, 380, c.Width()-40, 100)

		// Flex column with spacer.
		topLabel := ui.NewLabel("Top", labelStyle(canvas.TextAlignLeft))
		botLabel := ui.NewLabel("Bottom (pushed down)", labelStyle(canvas.TextAlignLeft))
		col := ui.NewFlex(ui.FlexColumn,
			ui.FixedItem(30, topLabel),
			ui.Flexible(1, ui.NewSpacer()),
			ui.FixedItem(30, botLabel),
		)
		col.Draw(c, 20, 500, 200, 200)

		// Stack: layered components.
		bg := ui.NewCard(ui.CardStyle{Background: canvas.Hex("#181825"), Radius: 8}, nil)
		fg := ui.NewLabel("Overlaid on card", labelStyle(canvas.TextAlignCenter))
		stack := ui.NewStack(bg, fg)
		stack.Draw(c, 240, 500, 200, 80)
	})

	w.Show()
}

// ── helpers ────────────────────────────────────────────────────────────────

func buildDefaultLayout() ui.Component {
	return ui.NewPadding(canvas.All(24),
		ui.NewFlex(ui.FlexColumn,
			ui.FixedItem(48, ui.NewLabel("Default (large) layout", labelStyle(canvas.TextAlignCenter))),
			ui.Flexible(1, ui.NewSpacer()),
		),
	)
}

func labelStyle(align canvas.TextAlign) ui.LabelStyle {
	s := ui.DefaultLabelStyle()
	s.TextBox.Align = align
	s.TextBox.Overflow = canvas.TextOverflowEllipsis
	return s
}
