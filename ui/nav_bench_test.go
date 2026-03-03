package ui

import (
	"testing"

	"github.com/achiket/gui-go/canvas"
	"github.com/achiket/gui-go/render"
)

// DummyScreen implements the Screen interface for benchmarking.
type DummyScreen struct {
	BaseScreen
}

func (s *DummyScreen) Bounds() canvas.Rect                       { return canvas.Rect{X: 0, Y: 0, W: 800, H: 600} }
func (s *DummyScreen) Tick(_ float64)                            {}
func (s *DummyScreen) Draw(_ *canvas.Canvas, _, _, _, _ float32) {}
func (s *DummyScreen) HandleEvent(_ Event) bool                  { return false }
func (s *DummyScreen) OnEnter(_ *Navigator)                      {}
func (s *DummyScreen) OnLeave()                                  {}

func BenchmarkNavigatorHandleEvent(b *testing.B) {
	nav := NewNavigator(&DummyScreen{})
	// Push a few screens to make the stack depth more realistic
	nav.Push(&DummyScreen{})
	nav.Push(&DummyScreen{})

	ev := Event{Type: EventMouseMove, X: 100, Y: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = nav.HandleEvent(ev)
	}
}

func BenchmarkNavigatorTick(b *testing.B) {
	nav := NewNavigator(&DummyScreen{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nav.Tick(0.016)
	}
}

func BenchmarkButtonHandleEvent(b *testing.B) {
	// Need a mock canvas to avoid panic in Draw
	mockRen := &MockRenderer{800, 600}
	c := canvas.NewCanvas(mockRen, 800, 600)

	btn := NewButton("Test", func() {})
	btn.Draw(c, 0, 0, 100, 40) // set bounds

	ev := Event{Type: EventMouseMove, X: 50, Y: 20}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = btn.HandleEvent(ev)
	}
}

// MockRenderer for the test
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
