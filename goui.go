// Package goui is a lightweight, zero-dependency Go GUI library for Linux (X11).
//
// It uses CGo to call Xlib directly — no external Go packages required.
// All X11 complexity is hidden behind a clean, idiomatic Go API.
//
// # Quick start
//
//	w := goui.NewWindow("Hello", 800, 600)
//	w.OnDraw(func(c *goui.Canvas) {
//	    c.Clear()
//	    c.SetColor(goui.Blue)
//	    c.FillCircle(400, 300, 50)
//	})
//	w.Show() // blocks until window closes
package goui

import "github.com/achiket123/gui-go/animation"

// Animatable is an alias for the animation package's Animatable interface.
// The render loop calls Tick and removes finished non-looping animations.
type Animatable = animation.Animatable
