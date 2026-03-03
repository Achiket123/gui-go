package gl

import "unsafe"

// Vertex layout (in floats):
//   pos:  x, y           (2)
//   uv:   u, v           (2)
//   color: r, g, b, a   (4)
//   extra: mode, radius, size_x, size_y  (4)  — shape SDF params
// Total: 12 floats = 48 bytes per vertex.

const (
	floatsPerVertex = 12
	bytesPerVertex  = floatsPerVertex * 4
	maxVertices     = 65536
	maxIndices      = maxVertices / 4 * 6 // 6 indices per quad
)

// DrawMode tells the fragment shader how to interpret a vertex.
const (
	ModeColor    = float32(0) // solid color fill
	ModeTexture  = float32(1) // texture sampling
	ModeGlyph    = float32(2) // alpha-only texture (font glyph)
	ModeGradient = float32(3) // gradient (color = lerp between extra.extra_a and extra.extra_b)
)

// Batch accumulates vertex + index data for a frame.
// Flush() uploads to the GPU and issues a draw call.
type Batch struct {
	vertices []float32
	indices  []uint32
	count    int // current vertex count

	vao uint32
	vbo uint32
	ebo uint32
}

// NewBatch creates a Batch and allocates GPU buffers.
func NewBatch() *Batch {
	b := &Batch{
		vertices: make([]float32, 0, maxVertices*floatsPerVertex),
		indices:  make([]uint32, 0, maxIndices),
	}

	b.vao = GenVAO()
	BindVAO(b.vao)

	b.vbo = GenBuffer()
	BindArrayBuffer(b.vbo)

	b.ebo = GenBuffer()
	BindElementBuffer(b.ebo)

	// Bind attribute layout — must match floatsPerVertex layout above:
	// attr 0: pos  (2 floats, offset 0)
	// attr 1: uv   (2 floats, offset 8)
	// attr 2: col  (4 floats, offset 16)
	// attr 3: extra(4 floats, offset 32)
	AttribPointerF(0, 2, bytesPerVertex, 0)
	AttribPointerF(1, 2, bytesPerVertex, 8)
	AttribPointerF(2, 4, bytesPerVertex, 16)
	AttribPointerF(3, 4, bytesPerVertex, 32)

	BindVAO(0)
	return b
}

// Reset clears the batch for a new frame.
func (b *Batch) Reset() {
	b.vertices = b.vertices[:0]
	b.indices = b.indices[:0]
	b.count = 0
}

// NeedsFlush returns true when the batch is near capacity.
func (b *Batch) NeedsFlush() bool {
	return b.count+4 > maxVertices
}

// PushQuad adds a textured/colored quad (4 vertices, 2 triangles).
// Positions are in pixel coordinates; the renderer shader converts using the projection matrix.
func (b *Batch) PushQuad(
	x0, y0, x1, y1 float32, // top-left, bottom-right in pixels
	u0, v0, u1, v1 float32, // UV coords
	r, g, ba, a float32, // vertex color
	mode, radius, sizeX, sizeY float32, // SDF/shader extra params
) {
	base := uint32(b.count)

	// TL
	b.vertices = append(b.vertices, x0, y0, u0, v0, r, g, ba, a, mode, radius, sizeX, sizeY)
	// TR
	b.vertices = append(b.vertices, x1, y0, u1, v0, r, g, ba, a, mode, radius, sizeX, sizeY)
	// BR
	b.vertices = append(b.vertices, x1, y1, u1, v1, r, g, ba, a, mode, radius, sizeX, sizeY)
	// BL
	b.vertices = append(b.vertices, x0, y1, u0, v1, r, g, ba, a, mode, radius, sizeX, sizeY)

	b.indices = append(b.indices,
		base, base+1, base+2,
		base, base+2, base+3,
	)
	b.count += 4
}

// PushTriangle adds a single triangle (for polygon fans, etc.).
func (b *Batch) PushTriangle(
	x0, y0, x1, y1, x2, y2 float32,
	r, g, ba, a, mode float32,
) {
	base := uint32(b.count)
	dummy := float32(0)
	b.vertices = append(b.vertices,
		x0, y0, dummy, dummy, r, g, ba, a, mode, dummy, dummy, dummy,
		x1, y1, dummy, dummy, r, g, ba, a, mode, dummy, dummy, dummy,
		x2, y2, dummy, dummy, r, g, ba, a, mode, dummy, dummy, dummy,
	)
	b.indices = append(b.indices, base, base+1, base+2)
	b.count += 3
}

// Flush uploads the current batch to the GPU and draws it.
func (b *Batch) Flush() {
	if len(b.indices) == 0 {
		return
	}
	BindVAO(b.vao)
	BindArrayBuffer(b.vbo)
	BufferArraysDynamic(b.vertices)
	BindElementBuffer(b.ebo)
	BufferElementsDynamic(b.indices)

	DrawTriangleElements(len(b.indices))

	// Reset but keep allocated capacity
	b.vertices = b.vertices[:0]
	b.indices = b.indices[:0]
	b.count = 0

	BindVAO(0)

	_ = unsafe.Sizeof(b) // keep import
}

// Destroy frees GPU resources.
func (b *Batch) Destroy() {
	DeleteBuffer(b.vbo)
	DeleteBuffer(b.ebo)
	DeleteVAO(b.vao)
}
