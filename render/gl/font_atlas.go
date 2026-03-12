package gl

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/achiket123/gui-go/render"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	atlasSize    = 1024 // atlas texture is 1024×1024
	defaultDPI   = 96.0
	glyphPadding = 2
)

// FontAtlas holds a GPU texture atlas containing pre-rasterized glyphs.
type FontAtlas struct {
	face    font.Face
	texture uint32
	pixels  []byte // RGBA, atlasSize×atlasSize
	glyphs  map[rune]render.GlyphMetrics
	penX    int
	penY    int
	rowH    int
	size    float32
}

// LoadFont loads a TTF/OTF file and builds a FontAtlas for the given size.
func LoadFont(path string, size float32) (*FontAtlas, error) {
	data, err := os.ReadFile(path) // #nosec G304 — intentional file load
	if err != nil {
		return nil, fmt.Errorf("font_atlas: %w", err)
	}
	return newAtlasFromBytes(data, size)
}

// LoadSystemFont searches common system font directories for a font by name recursively.
func LoadSystemFont(name string, size float32) (*FontAtlas, error) {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		os.Getenv("HOME") + "/.fonts",
		os.Getenv("HOME") + "/.local/share/fonts",
	}
	var foundPath string
	lowerName := strings.ToLower(name)
	for _, dir := range dirs {
		if walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".ttf") {
				if strings.Contains(strings.ToLower(d.Name()), lowerName) {
					foundPath = path
					return fmt.Errorf("found") // break early
				}
			}
			return nil
		}); walkErr != nil && walkErr.Error() != "found" {
			// real walk error — log and continue to next directory
			_ = walkErr
		}
		if foundPath != "" {
			return LoadFont(foundPath, size)
		}
	}
	return nil, fmt.Errorf("font_atlas: system font %q not found", name)
}

func newAtlasFromBytes(data []byte, size float32) (*FontAtlas, error) {
	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("font_atlas: parse: %w", err)
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     defaultDPI,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("font_atlas: face: %w", err)
	}

	a := &FontAtlas{
		face:   face,
		pixels: make([]byte, atlasSize*atlasSize*4),
		glyphs: make(map[rune]render.GlyphMetrics),
		size:   size,
	}

	// Pre-rasterize ASCII printable characters.
	for r := rune(32); r < 128; r++ {
		a.rasterize(r)
	}

	// Upload to GPU.
	a.texture = GenTexture()
	BindTexture2D(a.texture)
	UploadTextureRGBA(atlasSize, atlasSize, a.pixels)

	return a, nil
}

// rasterize adds a single rune to the atlas, expanding if needed.
func (a *FontAtlas) rasterize(r rune) render.GlyphMetrics {
	if m, ok := a.glyphs[r]; ok {
		return m
	}

	bounds, advance, ok := a.face.GlyphBounds(r)
	if !ok {
		return render.GlyphMetrics{}
	}

	bw := (bounds.Max.X - bounds.Min.X).Ceil() + glyphPadding*2
	bh := (bounds.Max.Y - bounds.Min.Y).Ceil() + glyphPadding*2
	if bw <= 0 || bh <= 0 {
		// Space or zero-size glyph — store advance only.
		m := render.GlyphMetrics{
			Advance: float32(advance.Round()),
		}
		a.glyphs[r] = m
		return m
	}

	// Wrap to next row if needed.
	if a.penX+bw > atlasSize {
		a.penX = 0
		a.penY += a.rowH + glyphPadding
		a.rowH = 0
	}
	if a.penY+bh > atlasSize {
		// Atlas full — return empty (fail gracefully)
		return render.GlyphMetrics{}
	}
	if bh > a.rowH {
		a.rowH = bh
	}

	// Rasterize into a temporary image.
	dst := image.NewRGBA(image.Rect(0, 0, bw, bh))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)

	dot := fixed.Point26_6{
		X: fixed.I(glyphPadding) - bounds.Min.X,
		Y: fixed.I(glyphPadding) - bounds.Min.Y,
	}
	d := &font.Drawer{Dst: dst, Src: image.White, Face: a.face, Dot: dot}
	d.DrawString(string(r))

	// Copy into atlas pixel buffer.
	ox, oy := a.penX, a.penY
	for y := 0; y < bh; y++ {
		for x := 0; x < bw; x++ {
			src := dst.RGBAAt(x, y)
			off := ((oy+y)*atlasSize + (ox + x)) * 4
			a.pixels[off+0] = 255
			a.pixels[off+1] = 255
			a.pixels[off+2] = 255
			a.pixels[off+3] = src.A
		}
	}

	m := render.GlyphMetrics{
		U0:       float32(ox) / atlasSize,
		V0:       float32(oy) / atlasSize,
		U1:       float32(ox+bw) / atlasSize,
		V1:       float32(oy+bh) / atlasSize,
		BitmapW:  float32(bw),
		BitmapH:  float32(bh),
		BearingX: float32(bounds.Min.X.Round()),
		BearingY: float32(bounds.Min.Y.Round()),
		Advance:  float32(advance.Round()),
	}
	a.glyphs[r] = m
	a.penX += bw + glyphPadding
	return m
}

// GlyphInfo returns metrics for a rune, rasterizing on demand.
func (a *FontAtlas) GlyphInfo(r rune) render.GlyphMetrics {
	if m, ok := a.glyphs[r]; ok {
		return m
	}
	m := a.rasterize(r)
	// Re-upload the atlas texture after adding new glyphs.
	BindTexture2D(a.texture)
	UploadTextureRGBA(atlasSize, atlasSize, a.pixels)
	return m
}

// AtlasTexture returns the GPU texture ID.
func (a *FontAtlas) AtlasTexture() uint32 { return a.texture }

// MeasureString returns the pixel width and height for a string.
func (a *FontAtlas) MeasureString(text string) (w, h float32) {
	metrics := a.face.Metrics()
	h = float32(metrics.Height.Round())
	for _, r := range text {
		m := a.GlyphInfo(r)
		w += m.Advance
	}
	return
}

// Size returns the point size the atlas was built for.
func (a *FontAtlas) Size() float32 { return a.size }

// Destroy frees the GPU texture.
func (a *FontAtlas) Destroy() {
	DeleteTexture(a.texture)
	if err := a.face.Close(); err != nil {
		_ = err // resource teardown — nothing to do
	}
}

// circleVertices returns a fan of triangle vertices for a filled circle.
// Returns flat x,y pairs in pixel space.
func circleVertices(cx, cy, r float32, segments int) []float32 {
	pts := make([]float32, 0, (segments+1)*2)
	for i := 0; i <= segments; i++ {
		angle := 2 * math.Pi * float64(i) / float64(segments)
		pts = append(pts, cx+r*float32(math.Cos(angle)), cy+r*float32(math.Sin(angle)))
	}
	return pts
}
