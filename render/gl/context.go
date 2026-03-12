package gl

// ContextProvider represents a platform-specific OpenGL window context.
type ContextProvider interface {
	MakeContextCurrent()
	SwapBuffers()
	Destroy()
}
