// Package goui is a lightweight, zero-dependency Go GUI library for Linux, Windows, and macOS.
//
// It uses CGo to communicate with the OS via GLFW — only one system dependency required.
// All complexity is hidden behind a clean, idiomatic Go API.
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
