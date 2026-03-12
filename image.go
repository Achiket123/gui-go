package goui

import (
	"image"
	"image/color"
	_ "image/png" // register PNG decoder
	"os"
	"unsafe"

	"github.com/achiket123/gui-go/platform"
)

// Image is a wrapper around a pixel buffer that can be drawn to a Canvas.
// Internally it keeps a 32-bit BGRX byte slice that matches what X11 expects.
type Image struct {
	width  int
	height int
	// pixels stores raw pixel data in BGRX byte order (4 bytes per pixel).
	pixels []byte

	// ximg is the lazy-created XImage handle; created on first draw call.
	ximg *platform.XImageHandle
}

// NewImage creates a blank (transparent/black) image of the given dimensions.
func NewImage(width, height int) *Image {
	return &Image{
		width:  width,
		height: height,
		pixels: make([]byte, width*height*4),
	}
}

// LoadPNG loads a PNG file from disk and converts it to an Image.
func LoadPNG(path string) (*Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	img := NewImage(w, h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBAModel.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.RGBA)
			img.SetPixelRaw(x, y, c.R, c.G, c.B)
		}
	}
	return img, nil
}

// Width returns the image width in pixels.
func (img *Image) Width() int { return img.width }

// Height returns the image height in pixels.
func (img *Image) Height() int { return img.height }

// offset returns the byte offset for pixel (x, y).
func (img *Image) offset(x, y int) int {
	return (y*img.width + x) * 4
}

// SetPixelRaw sets a pixel directly with RGB components (BGRX layout).
func (img *Image) SetPixelRaw(x, y int, r, g, b uint8) {
	if x < 0 || y < 0 || x >= img.width || y >= img.height {
		return
	}
	off := img.offset(x, y)
	img.pixels[off+0] = b // B
	img.pixels[off+1] = g // G
	img.pixels[off+2] = r // R
	img.pixels[off+3] = 0 // X (unused / padding)
	img.ximg = nil        // Invalidate cached XImage
}

// SetPixel sets a pixel using a goui Color.
func (img *Image) SetPixel(x, y int, c Color) {
	img.SetPixelRaw(x, y, c.R, c.G, c.B)
}

// GetPixel reads the color of a pixel.
func (img *Image) GetPixel(x, y int) Color {
	if x < 0 || y < 0 || x >= img.width || y >= img.height {
		return Black
	}
	off := img.offset(x, y)
	return Color{
		R: img.pixels[off+2],
		G: img.pixels[off+1],
		B: img.pixels[off+0],
	}
}

// ensureXImage creates or recreates the XImage handle if needed.
func (img *Image) ensureXImage(display unsafe.Pointer, screen int) {
	if img.ximg != nil {
		return
	}
	img.ximg = platform.CreateXImage(display, screen, img.width, img.height, img.pixels)
}

// xImageHandle returns the platform XImageHandle, creating it if necessary.
// Used by Canvas.DrawImage.
func (img *Image) xImageHandle(display unsafe.Pointer, screen int) *platform.XImageHandle {
	img.ensureXImage(display, screen)
	return img.ximg
}

// Destroy frees the underlying XImage. Call when you no longer need this image.
func (img *Image) Destroy() {
	platform.DestroyXImage(img.ximg)
	img.ximg = nil
}
