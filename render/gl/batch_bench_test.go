package gl

import (
	"testing"
)

func BenchmarkBatchPushQuad(b *testing.B) {
	// Manually create a Batch without calling NewBatch to avoid GL context requirement
	batch := &Batch{
		vertices: make([]float32, 0, maxVertices*floatsPerVertex),
		indices:  make([]uint32, 0, maxIndices),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if batch.NeedsFlush() {
			batch.Reset()
		}
		// Typical call as used in GL2DRenderer.DrawFilledRect
		batch.PushQuad(
			0, 0, 100, 100,
			0, 0, 1, 1,
			1, 1, 1, 1,
			ModeColor, 0, 0, 0,
		)
	}
	// Sink — forces compiler to keep the work
	_ = batch.count
}

func BenchmarkBatchPushTriangle(b *testing.B) {
	batch := &Batch{
		vertices: make([]float32, 0, maxVertices*floatsPerVertex),
		indices:  make([]uint32, 0, maxIndices),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if batch.NeedsFlush() {
			batch.Reset()
		}
		batch.PushTriangle(
			0, 0, 100, 0, 50, 100,
			1, 0, 0, 1, ModeColor,
		)
	}
	// Sink
	_ = batch.count
}

func BenchmarkBatchReset(b *testing.B) {
	batch := &Batch{
		vertices: make([]float32, 0, maxVertices*floatsPerVertex),
		indices:  make([]uint32, 0, maxIndices),
	}
	// Fill it up a bit
	for i := 0; i < 1000; i++ {
		batch.PushQuad(0, 0, 10, 10, 0, 0, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batch.Reset()
		// Re-fill a bit to make it realistic
		batch.count = 1000
	}
}
