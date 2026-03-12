// Package gl provides an OpenGL 2D renderer using GLFW.
// It implements the render.Renderer interface entirely via CGo.
//
// Three CGo files cooperate:
//   - context.go:     Context lifecycle
//   - gl2d_renderer.go: Renderer method implementations
//   - (shader/batch/texture embedded in gl2d_renderer.go for simplicity)
package gl
