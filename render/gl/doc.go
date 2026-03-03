// Package gl provides an OpenGL 2D renderer using GLX (OpenGL Extension to X11).
// It implements the render.Renderer interface entirely via CGo.
//
// Three CGo files cooperate:
//   - context.go:     GLX context lifecycle
//   - gl2d_renderer.go: Renderer method implementations
//   - (shader/batch/texture embedded in gl2d_renderer.go for simplicity)
package gl
