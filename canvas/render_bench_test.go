package canvas

import (
	"testing"

	"github.com/achiket123/gui-go/render"
)

// MockRenderer implements render.Renderer but does nothing.
// Used for benchmarking the canvas layer overhead independently of the GPU/CPU backend.
type MockRenderer struct {
	w, h int
}

func (m *MockRenderer) Init(_, _ interface{}, w, h int) error                             { m.w, m.h = w, h; return nil }
func (m *MockRenderer) Resize(w, h int)                                                   { m.w, m.h = w, h }
func (m *MockRenderer) BeginFrame(_ [4]float32)                                           {}
func (m *MockRenderer) EndFrame()                                                         {}
func (m *MockRenderer) DrawFilledRect(_, _, _, _, _ float32, _ [4]float32, _ float32)     {}
func (m *MockRenderer) DrawStrokedRect(_, _, _, _, _, _ float32, _ [4]float32, _ float32) {}
func (m *MockRenderer) DrawFilledCircle(_, _, _ float32, _ [4]float32, _ float32)         {}
func (m *MockRenderer) DrawFilledEllipse(_, _, _, _ float32, _ [4]float32, _ float32)     {}
func (m *MockRenderer) DrawLine(_, _, _, _, _ float32, _ [4]float32, _ float32)           {}
func (m *MockRenderer) DrawFilledPolygon(_ []float32, _ [4]float32, _ float32)            {}
func (m *MockRenderer) DrawGradientRect(_, _, _, _ float32, _, _ [4]float32, _, _ [2]float32, _ float32) {
}
func (m *MockRenderer) DrawTexture(_ render.TextureID, _, _, _, _, _, _, _, _ float32, _ [4]float32, _ float32) {
}
func (m *MockRenderer) DrawGlyph(_ render.TextureID, _ render.GlyphMetrics, _, _ float32, _ [4]float32, _ float32) {
}
func (m *MockRenderer) SetClipRect(_, _, _, _ float32)                       {}
func (m *MockRenderer) ClearClip()                                           {}
func (m *MockRenderer) PushTransform(_ [9]float32)                           {}
func (m *MockRenderer) PopTransform()                                        {}
func (m *MockRenderer) SetGlobalOpacity(_ float32)                           {}
func (m *MockRenderer) UploadTexture(_, _ int, _ []byte) render.TextureID    { return 0 }
func (m *MockRenderer) UpdateTexture(_ render.TextureID, _, _ int, _ []byte) {}
func (m *MockRenderer) DeleteTexture(_ render.TextureID)                     {}
func (m *MockRenderer) BeginScene3D()                                        {}
func (m *MockRenderer) EndScene3D()                                          {}
func (m *MockRenderer) Width() int                                           { return m.w }
func (m *MockRenderer) Height() int                                          { return m.h }

func BenchmarkDrawRect(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	paint := FillPaint(Blue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawRect(10, 10, 100, 100, paint)
	}
}

func BenchmarkDrawRoundedRect(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	paint := FillPaint(Blue)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawRoundedRect(10, 10, 100, 100, 10, paint)
	}
}

func BenchmarkDrawCircle(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	paint := FillPaint(Red)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawCircle(400, 300, 50, paint)
	}
}

func BenchmarkDrawText(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	style := DefaultTextStyle()

	// Pre-warm font cache so first iteration isn't an outlier
	c.DrawText(0, 0, "warmup", style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawText(50, 50, "Benchmark text line", style)
	}
}

func BenchmarkMeasureText(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	style := DefaultTextStyle()

	// Pre-warm font cache
	c.MeasureText("warmup", style)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.MeasureText("Measure this string", style)
	}
}

func BenchmarkDrawPath(b *testing.B) {
	r := &MockRenderer{800, 600}
	c := NewCanvas(r, 800, 600)
	p := NewPath()
	p.MoveTo(0, 0)
	p.LineTo(100, 100)
	p.QuadTo(200, 0, 300, 100)
	p.Close()
	paint := FillPaint(Green)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawPath(p, paint)
	}
}
