// Package render defines the Renderer interface — the abstraction between
// the Canvas drawing API and the underlying GPU (OpenGL) or CPU backend.
//
// Today: GL2DRenderer (OpenGL via GLX).
// Future: GL3DRenderer, VulkanRenderer — same Canvas API, different backend.
package render

// TextureID is an opaque handle to a GPU texture.
type TextureID uint32

// GlyphMetrics describes a single glyph's position in a font atlas texture.
type GlyphMetrics struct {
	// UVRect is the normalized rect in the atlas texture [0,1].
	U0, V0, U1, V1 float32
	// BitmapSize is the pixel size of the glyph.
	BitmapW, BitmapH float32
	// Bearing is the offset from the baseline to the top-left of the glyph.
	BearingX, BearingY float32
	// Advance is the horizontal advance to the next glyph.
	Advance float32
}

// Renderer is the abstraction layer between the Canvas API and the GPU/CPU backend.
// All draw methods are called between BeginFrame and EndFrame.
//
// Thread safety: all calls must be made from the render goroutine (LockOSThread).
type Renderer interface {
	// Init creates the rendering context on the native window.
	// Init creates the rendering context on the native window.
	Init(ctx interface{}, loadProc interface{}, w, h int) error

	// Resize updates the viewport when the window is resized.
	Resize(w, h int)

	// BeginFrame clears the framebuffer and resets the draw batch.
	BeginFrame(clearColor [4]float32)

	// EndFrame flushes the batch and presents the frame (SwapBuffers or XFlush).
	EndFrame()

	// --- 2D Primitives ---

	DrawFilledRect(x, y, w, h, cornerRadius float32, c [4]float32, opacity float32)
	DrawStrokedRect(x, y, w, h, cornerRadius, strokeWidth float32, c [4]float32, opacity float32)
	DrawFilledCircle(cx, cy, r float32, c [4]float32, opacity float32)
	DrawFilledEllipse(cx, cy, rx, ry float32, c [4]float32, opacity float32)
	DrawLine(x1, y1, x2, y2, thickness float32, c [4]float32, opacity float32)
	DrawFilledPolygon(pts []float32, c [4]float32, opacity float32) // flat x,y pairs
	DrawGradientRect(x, y, w, h float32, c0, c1 [4]float32, p1, p2 [2]float32, opacity float32)

	// DrawTexture draws a GPU texture at (x,y,w,h).
	// srcU0,V0,U1,V1 is the UV sub-rect (0–1). tint modulates color.
	DrawTexture(id TextureID, x, y, w, h, srcU0, srcV0, srcU1, srcV1 float32, tint [4]float32, opacity float32)

	// DrawGlyph draws a single glyph from a font atlas texture.
	DrawGlyph(atlasID TextureID, g GlyphMetrics, dstX, dstY float32, c [4]float32, opacity float32)

	// --- Clipping ---

	SetClipRect(x, y, w, h float32)
	ClearClip()

	// --- Transforms ---

	// PushTransform pushes a 3x3 affine matrix (column-major, 9 floats).
	PushTransform(mat [9]float32)
	PopTransform()
	SetGlobalOpacity(opacity float32)

	// --- Texture resource management ---

	UploadTexture(w, h int, pixels []byte) TextureID     // RGBA8
	UpdateTexture(id TextureID, w, h int, pixels []byte) // Update sub-region
	DeleteTexture(id TextureID)

	// --- Future 3D hooks (no-op in GL2DRenderer) ---

	BeginScene3D()
	EndScene3D()

	// --- State ---

	Width() int
	Height() int
}
