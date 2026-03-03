package gl

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/achiket/gui-go/render"
)

// Shape vertex shader — transforms pixel-space coords to NDC using a projection matrix.
const shapeVert = `
#version 120
attribute vec2 a_pos;
attribute vec2 a_uv;
attribute vec4 a_color;
attribute vec4 a_extra;  // x=mode, y=radius, z=sizeX, w=sizeY

uniform mat4 u_proj;
uniform mat3 u_model;

varying vec2  v_uv;
varying vec4  v_color;
varying vec4  v_extra;
varying vec2  v_pos;

void main() {
    v_uv    = a_uv;
    v_color = a_color;
    v_extra = a_extra;
    v_pos   = a_pos;
    vec3 pos = u_model * vec3(a_pos, 1.0);
    gl_Position = u_proj * vec4(pos.xy, 0.0, 1.0);
}
`

// Shape fragment shader — handles color, texture, glyph, and SDF rounded-rect.
const shapeFrag = `
#version 120
uniform sampler2D u_tex;
uniform float     u_opacity;

varying vec2 v_uv;
varying vec4 v_color;
varying vec4 v_extra; // x=mode, y=radius, z=sizeX, w=sizeY
varying vec2 v_pos;

float roundedBoxSDF(vec2 center, vec2 halfSize, float radius) {
    vec2 q = abs(center) - halfSize + radius;
    return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - radius;
}

float ellipseSDF(vec2 p, vec2 ab) {
    // Gradient distance approximation for ellipse SDF
    // p is relative to center. ab is half-widths.
    float f = length(p / ab) - 1.0;
    vec2 g = p / (ab * ab);
    return f / length(g);
}

void main() {
    float mode   = v_extra.x;
    float radius = v_extra.y;
    float sx     = v_extra.z;
    float sy     = v_extra.w;

    vec4 col;
    if (mode < 0.5) {
        // Solid color
        col = v_color;
        
        // Mode 0: We determine whether it's an ellipse or rounded rect based on radius value.
        // For ellipses, we pass radius = -1.0 as a signal.
        if (radius < -0.5 && sx > 0.0 && sy > 0.0) {
            vec2 center = v_uv - 0.5; // uv is 0..1, remap to -0.5..0.5
            // Multiply by size to operate in pixel space
            float d = ellipseSDF(center * vec2(sx, sy), vec2(sx*0.5, sy*0.5));
            float alpha = 1.0 - smoothstep(-0.5, 0.5, d);
            col.a *= alpha;
        } else if (radius > 0.5 && sx > 0.0 && sy > 0.0) {
            vec2 center = v_uv - 0.5;
            float d = roundedBoxSDF(center * vec2(sx, sy), vec2(sx*0.5, sy*0.5), radius);
            float alpha = 1.0 - smoothstep(-0.5, 0.5, d);
            col.a *= alpha;
        }
    } else if (mode < 1.5) {
        // Texture
        col = texture2D(u_tex, v_uv) * v_color;
    } else {
        // Glyph (alpha-only font atlas)
        float a = texture2D(u_tex, v_uv).a;
        col = vec4(v_color.rgb, v_color.a * a);
    }
    gl_FragColor = vec4(col.rgb, col.a * u_opacity);
}
`

// GL2DRenderer implements render.Renderer using OpenGL 2.1 + GLSL 1.20.
// It deliberately targets OpenGL 2.1 for maximum hardware compatibility.
type GL2DRenderer struct {
	ctx   *GLContext
	prog  uint32
	batch *Batch

	// Uniform locations
	uProj    int32
	uModel   int32
	uTex     int32
	uOpacity int32

	// State
	w, h           int
	transformStack [][16]float32
	currentTex     uint32
	fontAtlas      *FontAtlas
	opacity        float32
	globalOpacity  float32
}

// NewGL2DRenderer creates a renderer. Call Init() after.
func NewGL2DRenderer() *GL2DRenderer {
	return &GL2DRenderer{opacity: 1.0}
}

// Init creates the GLX context and compiles shaders.
func (r *GL2DRenderer) Init(display, xwin interface{}, w, h int) error {
	dpy := display.(unsafe.Pointer)
	win := xwin.(uintptr)
	screenNum := 0 // default screen

	var err error
	r.ctx, err = CreateContext(dpy, win, screenNum)
	if err != nil {
		return fmt.Errorf("GL2DRenderer: %w", err)
	}

	InitExtensions()
	EnableBlend()

	r.prog, err = CompileProgram(shapeVert, shapeFrag)
	if err != nil {
		return fmt.Errorf("GL2DRenderer: shaders: %w", err)
	}

	r.uProj = UniformLoc(r.prog, "u_proj")
	r.uModel = UniformLoc(r.prog, "u_model")
	r.uTex = UniformLoc(r.prog, "u_tex")
	r.uOpacity = UniformLoc(r.prog, "u_opacity")

	r.batch = NewBatch()
	r.w, r.h = w, h

	SetViewport(w, h)
	r.updateProjection()

	return nil
}

func (r *GL2DRenderer) updateProjection() {
	// Orthographic projection: pixel coords → NDC
	// Maps (0,0)→top-left, (w,h)→bottom-right
	l, rr, t, b := float32(0), float32(r.w), float32(0), float32(r.h)
	proj := ortho(l, rr, b, t, -1, 1)
	UseProgram(r.prog)
	UniformMat4(r.uProj, &proj)
}

// ortho builds a column-major OpenGL orthographic projection matrix.
func ortho(l, r, b, t, n, f float32) [16]float32 {
	return [16]float32{
		2 / (r - l), 0, 0, 0,
		0, 2 / (t - b), 0, 0,
		0, 0, -2 / (f - n), 0,
		-(r + l) / (r - l), -(t + b) / (t - b), -(f + n) / (f - n), 1,
	}
}

func (r *GL2DRenderer) Resize(w, h int) {
	r.w, r.h = w, h
	SetViewport(w, h)
	r.updateProjection()
}

func (r *GL2DRenderer) BeginFrame(clearColor [4]float32) {
	UseProgram(r.prog)
	Uniform1i(r.uTex, 0)
	Uniform1f(r.uOpacity, 1.0)

	ident := [9]float32{1, 0, 0, 0, 1, 0, 0, 0, 1}
	UniformMat3(r.uModel, &ident)

	ClearColor(clearColor[0], clearColor[1], clearColor[2], clearColor[3])
	r.batch.Reset()
	r.opacity = 1.0
	r.globalOpacity = 1.0
}

func (r *GL2DRenderer) EndFrame() {
	r.flushBatch()
	r.ctx.SwapBuffers()
}

func (r *GL2DRenderer) flushBatch() {
	if r.batch.count == 0 {
		return
	}
	ActiveTexture0()
	if r.currentTex != 0 {
		BindTexture2D(r.currentTex)
	}
	Uniform1f(r.uOpacity, r.opacity)
	r.batch.Flush()
}

// --- 2D Primitives ---

func (r *GL2DRenderer) DrawFilledRect(x, y, w, h, cornerRadius float32, c [4]float32, opacity float32) {
	r.batch.PushQuad(
		x, y, x+w, y+h,
		0, 0, 1, 1,
		c[0], c[1], c[2], c[3]*r.effectiveAlpha(opacity),
		ModeColor, cornerRadius, w, h,
	)
}

func (r *GL2DRenderer) DrawStrokedRect(x, y, w, h, cornerRadius, strokeWidth float32, c [4]float32, opacity float32) {
	sw := strokeWidth
	// Draw 4 edge quads (top, bottom, left, right) — simple approach.
	col := [4]float32{c[0], c[1], c[2], c[3] * r.effectiveAlpha(opacity)} // #nosec G602 -- c is always a 4-element array
	// Top
	r.batch.PushQuad(x, y, x+w, y+sw, 0, 0, 1, 1, col[0], col[1], col[2], col[3], ModeColor, 0, w, sw)
	// Bottom
	r.batch.PushQuad(x, y+h-sw, x+w, y+h, 0, 0, 1, 1, col[0], col[1], col[2], col[3], ModeColor, 0, w, sw)
	// Left
	r.batch.PushQuad(x, y+sw, x+sw, y+h-sw, 0, 0, 1, 1, col[0], col[1], col[2], col[3], ModeColor, 0, sw, h)
	// Right
	r.batch.PushQuad(x+w-sw, y+sw, x+w, y+h-sw, 0, 0, 1, 1, col[0], col[1], col[2], col[3], ModeColor, 0, sw, h)
}

func (r *GL2DRenderer) DrawFilledCircle(cx, cy, radius float32, c [4]float32, opacity float32) {
	// Approximated with a quad + SDF in the shader.
	r.batch.PushQuad(
		cx-radius, cy-radius, cx+radius, cy+radius,
		0, 0, 1, 1,
		c[0], c[1], c[2], c[3]*r.effectiveAlpha(opacity),
		ModeColor, radius, radius*2, radius*2,
	)
}

func (r *GL2DRenderer) DrawFilledEllipse(cx, cy, rx, ry float32, c [4]float32, opacity float32) {
	// Use circle SDF but scale to ellipse via extra params.
	r.batch.PushQuad(
		cx-rx, cy-ry, cx+rx, cy+ry,
		0, 0, 1, 1,
		c[0], c[1], c[2], c[3]*r.effectiveAlpha(opacity),
		ModeColor, 0 /* no radius uniform for ellipse—handled in shader via sizeX!=sizeY */, rx*2, ry*2,
	)
}

func (r *GL2DRenderer) DrawLine(x1, y1, x2, y2, thickness float32, c [4]float32, opacity float32) {
	// Calculate a rotated quad along the line direction.
	dx := x2 - x1
	dy := y2 - y1
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if length < 0.001 {
		return
	}
	nx := -dy / length * thickness * 0.5
	ny := dx / length * thickness * 0.5
	col := [4]float32{c[0], c[1], c[2], c[3] * r.effectiveAlpha(opacity)} // #nosec G602 -- c is always a 4-element array
	// 4 corners of the thick line quad.
	ax, ay := x1+nx, y1+ny
	bx, by := x1-nx, y1-ny
	cx2, cy2 := x2+nx, y2+ny
	dx2, dy2 := x2-nx, y2-ny
	_ = cx2
	_ = cy2
	base := uint32(r.batch.count) // #nosec G115 -- bounded by batch cap
	r.batch.vertices = append(r.batch.vertices,
		ax, ay, 0, 0, col[0], col[1], col[2], col[3], ModeColor, 0, 0, 0,
		bx, by, 0, 0, col[0], col[1], col[2], col[3], ModeColor, 0, 0, 0,
		dx2, dy2, 0, 0, col[0], col[1], col[2], col[3], ModeColor, 0, 0, 0,
		cx2, cy2, 0, 0, col[0], col[1], col[2], col[3], ModeColor, 0, 0, 0,
	)
	r.batch.indices = append(r.batch.indices, base, base+1, base+2, base, base+2, base+3)
	r.batch.count += 4
}

func (r *GL2DRenderer) DrawFilledPolygon(pts []float32, c [4]float32, opacity float32) {
	if len(pts) < 6 {
		return
	}
	col := [4]float32{c[0], c[1], c[2], c[3] * r.effectiveAlpha(opacity)} // #nosec G602 -- c is always a 4-element array
	// Fan from first vertex.
	for i := 2; i < len(pts)-2; i += 2 {
		r.batch.PushTriangle(
			pts[0], pts[1],
			pts[i], pts[i+1],
			pts[i+2], pts[i+3],
			col[0], col[1], col[2], col[3], ModeColor,
		)
	}
}

func (r *GL2DRenderer) SetGlobalOpacity(alpha float32) {
	r.globalOpacity = alpha
}

func (r *GL2DRenderer) effectiveAlpha(a float32) float32 {
	op := r.globalOpacity
	if op == 0 {
		op = 1
	}
	return a * op
}

func (r *GL2DRenderer) DrawGradientRect(x, y, w, h float32, c0, c1 [4]float32, p1, p2 [2]float32, opacity float32) {
	dx, dy := p2[0]-p1[0], p2[1]-p1[1]
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		r.DrawFilledRect(x, y, w, h, 0, c0, r.effectiveAlpha(opacity))
		return
	}

	// Calculate projection value [0,1] of a point (vx, vy) along the p1->p2 gradient vector
	proj := func(vx, vy float32) float32 {
		dot := (vx-p1[0])*dx + (vy-p1[1])*dy
		t := dot / lengthSq
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		return t
	}

	a0 := c0[3] * opacity
	a1 := c1[3] * opacity

	tTL := proj(x, y)
	tTR := proj(x+w, y)
	tBR := proj(x+w, y+h)
	tBL := proj(x, y+h)

	col := func(t float32) (float32, float32, float32, float32) {
		return c0[0] + (c1[0]-c0[0])*t,
			c0[1] + (c1[1]-c0[1])*t,
			c0[2] + (c1[2]-c0[2])*t,
			a0 + (a1-a0)*t
	}

	tlR, tlG, tlB, tlA := col(tTL)
	trR, trG, trB, trA := col(tTR)
	brR, brG, brB, brA := col(tBR)
	blR, blG, blB, blA := col(tBL)

	base := uint32(r.batch.count) // #nosec G115 -- bounded by batch cap
	r.batch.vertices = append(r.batch.vertices,
		x, y, 0, 0, tlR, tlG, tlB, tlA, ModeColor, 0, 0, 0,
		x+w, y, 1, 0, trR, trG, trB, trA, ModeColor, 0, 0, 0,
		x+w, y+h, 1, 1, brR, brG, brB, brA, ModeColor, 0, 0, 0,
		x, y+h, 0, 1, blR, blG, blB, blA, ModeColor, 0, 0, 0,
	)
	r.batch.indices = append(r.batch.indices, base, base+1, base+2, base, base+2, base+3)
	r.batch.count += 4
}

func (r *GL2DRenderer) DrawTexture(id render.TextureID, x, y, w, h, u0, v0, u1, v1 float32, tint [4]float32, opacity float32) {
	if uint32(id) != r.currentTex {
		r.flushBatch()
		r.currentTex = uint32(id)
	}
	r.batch.PushQuad(x, y, x+w, y+h, u0, v0, u1, v1, tint[0], tint[1], tint[2], tint[3]*r.effectiveAlpha(opacity), ModeTexture, 0, w, h)
}

func (r *GL2DRenderer) DrawGlyph(atlasID render.TextureID, g render.GlyphMetrics, dstX, dstY float32, c [4]float32, opacity float32) {
	if uint32(atlasID) != r.currentTex {
		r.flushBatch()
		r.currentTex = uint32(atlasID)
	}
	x := dstX + g.BearingX
	y := dstY + g.BearingY
	r.batch.PushQuad(x, y, x+g.BitmapW, y+g.BitmapH, g.U0, g.V0, g.U1, g.V1, c[0], c[1], c[2], c[3]*r.effectiveAlpha(opacity), ModeGlyph, 0, g.BitmapW, g.BitmapH)
}

func (r *GL2DRenderer) SetClipRect(x, y, w, h float32) {
	r.flushBatch()
	// OpenGL scissor expects y from bottom.
	ScissorOn(int(x), r.h-int(y+h), int(w), int(h))
}

func (r *GL2DRenderer) ClearClip() {
	r.flushBatch()
	ScissorOff()
}

func (r *GL2DRenderer) PushTransform(mat [9]float32) {
	r.flushBatch()
	// Convert row-major `mat` to column-major for OpenGL
	colMajor := [9]float32{
		mat[0], mat[3], mat[6],
		mat[1], mat[4], mat[7],
		mat[2], mat[5], mat[8],
	}
	UniformMat3(r.uModel, &colMajor)
}
func (r *GL2DRenderer) PopTransform() {}

func (r *GL2DRenderer) UploadTexture(w, h int, pixels []byte) render.TextureID {
	t := GenTexture()
	BindTexture2D(t)
	UploadTextureRGBA(w, h, pixels)
	return render.TextureID(t)
}

func (r *GL2DRenderer) UpdateTexture(id render.TextureID, w, h int, pixels []byte) {
	BindTexture2D(uint32(id))
	UploadTextureRGBA(w, h, pixels)
}

func (r *GL2DRenderer) DeleteTexture(id render.TextureID) {
	DeleteTexture(uint32(id))
}

func (r *GL2DRenderer) BeginScene3D() {}
func (r *GL2DRenderer) EndScene3D()   {}

func (r *GL2DRenderer) Width() int  { return r.w }
func (r *GL2DRenderer) Height() int { return r.h }

// SetDefaultFont sets the font atlas used for text rendering.
func (r *GL2DRenderer) SetDefaultFont(a *FontAtlas) { r.fontAtlas = a }

// DefaultFont returns the current default font atlas.
func (r *GL2DRenderer) DefaultFont() *FontAtlas { return r.fontAtlas }

// Destroy frees all GPU resources.
func (r *GL2DRenderer) Destroy() {
	r.batch.Destroy()
	DeleteProgram(r.prog)
	if r.fontAtlas != nil {
		r.fontAtlas.Destroy()
	}
	r.ctx.Destroy()
}
